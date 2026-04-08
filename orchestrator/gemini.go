package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GeminiClient talks to Google's Generative Language API (v1beta).
//
// It holds a pool of API keys and rotates through them:
//   - On each call it starts at the current cursor.
//   - On success it advances the cursor (spreads load across restarts and calls).
//   - On a retriable failure (429, 5xx, 403, network error) it tries the next key.
//   - On a non-retriable failure (400 bad request) it returns the error immediately —
//     retrying would just fail on every key.
type GeminiClient struct {
	model  string
	keys   []string
	client *http.Client

	mu     sync.Mutex
	cursor int
}

func NewGeminiClient(model string, keys []string) *GeminiClient {
	if len(keys) == 0 {
		// Programming error — caller must validate.
		panic("gemini: at least one API key is required")
	}
	return &GeminiClient{
		model:  model,
		keys:   keys,
		client: &http.Client{Timeout: 120 * time.Second},
		// Start at a random position so orchestrator restarts don't all hammer key[0] first.
		cursor: rand.Intn(len(keys)),
	}
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type geminiRequest struct {
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  geminiGenConfig `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finishReason"`
	} `json:"candidates"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// retriableErr marks errors that should cause key rotation.
type retriableErr struct{ err error }

func (r *retriableErr) Error() string { return r.err.Error() }
func (r *retriableErr) Unwrap() error { return r.err }

func (g *GeminiClient) Generate(systemPrompt, prompt string) (string, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Role: "user", Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenConfig{
			Temperature:     0.9,
			MaxOutputTokens: 80 + rand.Intn(171),
		},
	}
	if systemPrompt != "" {
		reqBody.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal gemini request: %w", err)
	}

	g.mu.Lock()
	start := g.cursor
	g.mu.Unlock()

	var lastErr error
	for i := 0; i < len(g.keys); i++ {
		idx := (start + i) % len(g.keys)
		text, err := g.callKey(g.keys[idx], body)
		if err == nil {
			g.mu.Lock()
			g.cursor = (idx + 1) % len(g.keys)
			g.mu.Unlock()
			if i > 0 {
				log.Printf("gemini: key %d succeeded after %d failover(s)", idx, i)
			}
			return text, nil
		}
		lastErr = err

		var retriable *retriableErr
		if !errors.As(err, &retriable) {
			// Hard failure (e.g. 400 bad request) — don't waste keys.
			return "", err
		}
		log.Printf("gemini: key %d failed (retriable): %v", idx, err)
	}

	return "", fmt.Errorf("all %d gemini keys failed: %w", len(g.keys), lastErr)
}

func (g *GeminiClient) callKey(apiKey string, body []byte) (string, error) {
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		g.model, apiKey,
	)

	resp, err := g.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		// Network-level errors are retriable.
		return "", &retriableErr{err: fmt.Errorf("gemini request: %w", err)}
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("gemini status %d: %s", resp.StatusCode, truncate(string(respBytes), 300))
		// 429 (rate limit), 403 (key disabled/quota), 5xx — rotate to next key.
		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusForbidden ||
			resp.StatusCode >= 500 {
			return "", &retriableErr{err: err}
		}
		// 400 etc. — not retriable.
		return "", err
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decode gemini response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("gemini api error: %s (%s)", parsed.Error.Message, parsed.Error.Status)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: no candidates in response: %s", truncate(string(respBytes), 300))
	}

	var sb strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	return sb.String(), nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
