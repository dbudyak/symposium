package main

// LLMClient is the interface every LLM provider implements.
// Each call is a single-shot text generation: system prompt + user prompt -> response.
type LLMClient interface {
	Generate(systemPrompt, prompt string) (string, error)
}
