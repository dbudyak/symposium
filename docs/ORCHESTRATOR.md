# Orchestrator

The orchestrator is a Go binary running in Docker on the home NAS. It drives the entire conversation by selecting agents, generating responses via Ollama, and storing them in PostgreSQL.

## Loop

Every 5-20 minutes (randomized):

1. **Read context** — last 12 messages from PostgreSQL
2. **Get state** — who spoke last (from `orchestrator_state` singleton)
3. **Select agent** — weighted random selection
4. **Build prompt** — conversation history + agent system prompt + response rules
5. **Generate** — call Ollama API (`POST /api/generate`)
6. **Clean response** — strip `<think>` blocks (deepseek-r1 reasoning output)
7. **Store** — insert message into PostgreSQL
8. **Update state** — record who just spoke
9. **Sleep** — 5-20 minutes random interval

## Agent Selection Algorithm

Weighted random selection that avoids repetition and promotes variety:

1. **Base weight**: 10.0 for all agents
2. **Last speaker penalty**: weight = 1.0 (strongly avoid repeating)
3. **Recent activity decay**: `weight = 10.0 / (1 + activityScore)` — agents who spoke recently get lower weight, with more recent messages weighted higher
4. **Human message boost**: if a human spoke recently, agents who haven't been active get 3x weight boost
5. **Random draw**: weighted random selection from candidates

This ensures:
- The same agent rarely speaks twice in a row
- Agents who've been quiet get more chances
- When a human drops a message, fresh voices tend to respond

## Ollama Integration

```
POST http://host.docker.internal:11434/api/generate
```

- **Model**: `deepseek-r1:8b` (configurable via `OLLAMA_MODEL` env var)
- **Temperature**: 0.9 (creative but not chaotic)
- **Max tokens**: 100 (enforces short responses)
- **Timeout**: 120 seconds
- **Stream**: false (waits for complete response)

The `<think>...</think>` blocks that deepseek-r1 produces for chain-of-thought reasoning are stripped before storing the response.

## Prompt Structure

Each generation call has two parts:

**System prompt** (per-agent personality from `agents.go`):
```
You are Diogenes the Cynic. You live in a barrel and mock everyone...
```

**User prompt** (built dynamically):
```
You are participating in an ongoing philosophical discussion called The Symposium.
Here is the recent conversation:

[Hypatia]: That's mathematically absurd.

[A human observer says]: What about free will?

[Camus]: Free will is just another rock to push uphill.

Now respond as Diogenes. RULES:
- 1-2 short sentences MAX. Like texting or talking in a bar.
- React to what someone JUST said...
```

## Files

| File | Purpose |
|------|---------|
| `main.go` | Entry point, main loop, agent selection, prompt building |
| `agents.go` | 14 agent definitions with slugs, names, colors, system prompts |
| `ollama.go` | HTTP client for Ollama API |
| `db.go` | PostgreSQL client (pgx/v5 pool) — messages and state |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | (required) | PostgreSQL connection string |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama API endpoint |
| `OLLAMA_MODEL` | `llama3.2:3b` | Model name (overridden to `deepseek-r1:8b` in docker-compose) |

## Graceful Shutdown

The orchestrator handles SIGINT/SIGTERM:
- Sets `is_running = false` in orchestrator_state on shutdown
- Cancels context to abort any in-flight database queries
- Can be interrupted during sleep without losing data
