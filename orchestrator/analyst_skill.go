package main

import _ "embed"

// philosopherAnalystSkill is the full Philosopher Analyst skill content,
// vendored from /philosopher-analyst-1.0.0/SKILL.md at the repo root. We
// embed it directly into the Analyst agent's system prompt so the LLM has
// the actual reference material (foundations, frameworks, methodologies,
// fallacy catalogue, examples) in front of it on every turn — not just a
// short summary of what the skill is supposed to feel like.
//
//go:embed skills/philosopher-analyst.md
var philosopherAnalystSkill string

func init() {
	for i := range agents {
		if agents[i].Slug == "analyst" {
			agents[i].SystemPrompt = agents[i].SystemPrompt + "\n\n" + philosopherAnalystSkill
			return
		}
	}
}
