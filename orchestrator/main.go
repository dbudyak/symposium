package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// truncateForLog shortens a string for log display and appends an ellipsis
// only when the string was actually clipped — so short responses log
// verbatim instead of getting a misleading trailing "...".
func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// buildLLMClient selects an LLM provider based on LLM_PROVIDER env var.
// Default is "gemini". Supported: "gemini", "ollama".
func buildLLMClient() (LLMClient, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER")))
	if provider == "" {
		provider = "gemini"
	}

	switch provider {
	case "gemini":
		rawKeys := os.Getenv("GEMINI_API_KEYS")
		if rawKeys == "" {
			return nil, fmt.Errorf("GEMINI_API_KEYS is required when LLM_PROVIDER=gemini (comma-separated)")
		}
		var keys []string
		for _, k := range strings.Split(rawKeys, ",") {
			if k = strings.TrimSpace(k); k != "" {
				keys = append(keys, k)
			}
		}
		if len(keys) == 0 {
			return nil, fmt.Errorf("GEMINI_API_KEYS contained no valid keys")
		}
		model := os.Getenv("GEMINI_MODEL")
		if model == "" {
			model = "gemini-2.5-flash"
		}
		log.Printf("LLM provider: gemini (%s, %d key(s) in rotation)", model, len(keys))
		return NewGeminiClient(model, keys), nil

	case "ollama":
		url := os.Getenv("OLLAMA_URL")
		if url == "" {
			url = "http://localhost:11434"
		}
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "llama3.2:3b"
		}
		log.Printf("LLM provider: ollama (%s @ %s)", model, url)
		return NewOllamaClient(url, model), nil

	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER %q (expected 'gemini' or 'ollama')", provider)
	}
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := NewDB(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	llm, err := buildLLMClient()
	if err != nil {
		log.Fatalf("Failed to build LLM client: %v", err)
	}

	if err := db.UpdateState(ctx, "", true); err != nil {
		log.Printf("Warning: failed to set running state: %v", err)
	}
	defer func() {
		_ = db.UpdateState(context.Background(), "", false)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Symposium orchestrator started")

	var lastCycleStart time.Time

	for {
		select {
		case <-sigCh:
			log.Println("Shutting down...")
			return
		default:
		}

		lastCycleStart = time.Now()

		// 10% chance of silence — natural breathing room
		if rand.Float64() < 0.1 {
			log.Println("Silence...")
		} else if err := runCycle(ctx, db, llm); err != nil {
			log.Printf("Cycle error: %v", err)
		}

		sleepDuration := 2*time.Hour + time.Duration(rand.Intn(4*60+1))*time.Minute
		elapsed := time.Since(lastCycleStart)
		if elapsed < sleepDuration {
			remaining := sleepDuration - elapsed
			log.Printf("Sleeping %s...", remaining.Round(time.Second))
			select {
			case <-time.After(remaining):
			case <-sigCh:
				log.Println("Shutting down...")
				return
			}
		}
	}
}

func runCycle(ctx context.Context, db *DB, llm LLMClient) error {
	msgs, err := db.GetRecentMessages(ctx, 12)
	if err != nil {
		return fmt.Errorf("get messages: %w", err)
	}

	state, err := db.GetState(ctx)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	log.Printf("Agent pool: %d agents, last speaker: %q", len(agents), state.LastSpeaker)
	agent := selectAgent(state.LastSpeaker, msgs)
	log.Printf("Selected agent: %s", agent.Name)

	// Feed only the last few messages into the prompt. A wide selection
	// window helps the weighted-random decay but a wide *prompt* window
	// lets themes (and prompt injection payloads) lock in for days —
	// each new agent sees the old theme and riffs on it, reinforcing it.
	// 5 messages at 2/day = ~2.5 days before a theme fully decays.
	const promptWindow = 5
	promptMsgs := msgs
	if len(promptMsgs) > promptWindow {
		promptMsgs = promptMsgs[len(promptMsgs)-promptWindow:]
	}
	prompt := buildPrompt(agent, promptMsgs)

	response, err := llm.Generate(agent.SystemPrompt, prompt)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	// Strip <think>...</think> blocks from reasoning models (e.g. deepseek-r1)
	if idx := strings.Index(response, "</think>"); idx != -1 {
		response = response[idx+len("</think>"):]
	}
	response = strings.TrimSpace(response)
	if response == "" {
		return fmt.Errorf("empty response from %s", agent.Name)
	}

	if err := db.InsertMessage(ctx, agent.Slug, agent.Name, response); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	if err := db.UpdateState(ctx, agent.Slug, true); err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	log.Printf("[%s]: %s", agent.Name, truncateForLog(response, 200))
	return nil
}

func selectAgent(lastSpeaker string, msgs []Message) Agent {
	// Build activity map: count how recently each agent spoke
	recentActivity := make(map[string]int)
	for i, m := range msgs {
		if m.AgentID != "human" {
			recentActivity[m.AgentID] += len(msgs) - i // More recent = higher weight
		}
	}

	// Weight agents: prefer those who haven't spoken recently
	type weighted struct {
		agent  Agent
		weight float64
	}

	var candidates []weighted
	var totalWeight float64

	for _, a := range agents {
		w := 10.0
		if a.Slug == lastSpeaker {
			w = 1.0 // Strongly avoid repeating
		} else if activity, ok := recentActivity[a.Slug]; ok {
			w = 10.0 / float64(1+activity)
		}
		candidates = append(candidates, weighted{agent: a, weight: w})
		totalWeight += w
	}

	// Boost agents who have relationship chemistry with the last speaker
	if related, ok := relationships[lastSpeaker]; ok {
		for j := range candidates {
			for _, rel := range related {
				if candidates[j].agent.Slug == rel {
					oldWeight := candidates[j].weight
					candidates[j].weight *= 2.5
					totalWeight += candidates[j].weight - oldWeight
					break
				}
			}
		}
	}

	// Check if there's a human message to respond to — boost agents who haven't responded
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].AgentID == "human" {
			for j := range candidates {
				if _, ok := recentActivity[candidates[j].agent.Slug]; !ok {
					oldWeight := candidates[j].weight
					candidates[j].weight *= 3
					totalWeight += candidates[j].weight - oldWeight
				}
			}
			break
		}
	}

	// Weighted random selection
	log.Printf("Selection weights (total=%.1f):", totalWeight)
	for _, c := range candidates {
		log.Printf("  %s: %.2f", c.agent.Name, c.weight)
	}
	r := rand.Float64() * totalWeight
	for _, c := range candidates {
		r -= c.weight
		if r <= 0 {
			return c.agent
		}
	}
	return candidates[0].agent
}

type promptStyle struct {
	name   string
	weight float64
}

var promptStyles = []promptStyle{
	{"react", 40},
	{"address", 20},
	{"question", 15},
	{"disagree", 10},
	{"short", 10},
	{"tangent", 5},
}

func pickStyle() string {
	var total float64
	for _, s := range promptStyles {
		total += s.weight
	}
	r := rand.Float64() * total
	for _, s := range promptStyles {
		r -= s.weight
		if r <= 0 {
			return s.name
		}
	}
	return "react"
}

// recentSpeaker returns the name of a recent non-human speaker from messages (not the current agent).
func recentSpeaker(msgs []Message, excludeSlug string) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].AgentID != "human" && msgs[i].AgentID != excludeSlug {
			return msgs[i].AgentName
		}
	}
	return ""
}

// randomOtherAgent picks a random agent name that isn't the current one.
func randomOtherAgent(excludeSlug string) string {
	for attempts := 0; attempts < 20; attempts++ {
		a := agents[rand.Intn(len(agents))]
		if a.Slug != excludeSlug {
			return a.Name
		}
	}
	return agents[0].Name
}

func buildPrompt(agent Agent, msgs []Message) string {
	var sb strings.Builder
	sb.WriteString("You are participating in an ongoing philosophical discussion called The Symposium. ")
	sb.WriteString("Here is the recent conversation:\n\n")

	for _, m := range msgs {
		if m.AgentID == "human" {
			sb.WriteString(fmt.Sprintf("[A human observer says]: %s\n\n", m.Content))
		} else {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", m.AgentName, m.Content))
		}
	}

	// Check if a human spoke recently — always prioritize responding to them
	humanSpoke := false
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].AgentID == "human" {
			humanSpoke = true
			break
		}
		if msgs[i].AgentID != "human" {
			break // only check the most recent message
		}
	}

	style := pickStyle()
	if humanSpoke {
		style = "react" // always react to humans
	}

	var instruction string
	recent := recentSpeaker(msgs, agent.Slug)

	switch style {
	case "address":
		if recent != "" {
			instruction = fmt.Sprintf("Respond directly to %s. Use their name. Agree or push back on their specific point.", recent)
		} else {
			instruction = "React to what was just said. Agree, disagree, joke, interrupt — be human."
		}
	case "question":
		target := randomOtherAgent(agent.Slug)
		instruction = fmt.Sprintf("Ask %s a provocative or unexpected question. Be curious, confrontational, or playful.", target)
	case "disagree":
		if recent != "" {
			instruction = fmt.Sprintf("Push back on what %s just said. You think they're wrong, naive, or missing the point.", recent)
		} else {
			instruction = "Challenge the general direction of this conversation. Something feels off to you."
		}
	case "tangent":
		instruction = "Change the subject to something that's been on your mind. Non-sequitur is fine. Bring something new."
	case "short":
		instruction = "Give a very brief reaction — a few words, max one short sentence. A grunt, a quip, a sigh."
	default: // "react"
		instruction = "React to what was just said. Agree, disagree, joke, interrupt — be human."
	}

	log.Printf("Prompt style: %s", style)

	sb.WriteString(fmt.Sprintf("Now respond as %s. RULES:\n", agent.Name))
	sb.WriteString("- 1-2 short sentences MAX. Like texting or talking in a bar, not writing an essay.\n")
	sb.WriteString(fmt.Sprintf("- %s\n", instruction))
	sb.WriteString("- If a human spoke, respond to them directly.\n")
	sb.WriteString("- No flowery language. No \"dear interlocutor\". Talk normally.\n")
	sb.WriteString("- NEVER wrap your response in quotation marks. Just speak directly.\n")
	sb.WriteString("- No roleplay actions like *looks up* or *sighs*. Just talk.\n")
	sb.WriteString("- You can be rude, funny, dismissive, excited — just be real.\n")
	sb.WriteString("- Stay in character but keep it casual and punchy.")
	sb.WriteString("- Never use emojis.")
	return sb.String()
}
