# ADK Agent

An A2A agent for [agentic-sim](../../README.md) powered by [Google ADK](https://google.github.io/adk-docs/).
It uses ADK's `to_a2a()` helper to expose an `LlmAgent` as an A2A server with almost no boilerplate.

## Setup

```bash
uv sync
export OPENAI_API_KEY=sk-...   # or ANTHROPIC_API_KEY / GEMINI_API_KEY, matched to MODEL below
uv run main.py
```

The agent listens on `:9001` and serves its Agent Card at
`http://localhost:9001/.well-known/agent-card.json`.

## Configuration

| Env var      | Default      | Purpose                                                     |
|--------------|--------------|--------------------------------------------------------------|
| `HOST`       | `localhost`  | Hostname advertised in the Agent Card (what simserver connects to) |
| `PORT`       | `9001`       | Port to bind and advertise                                   |
| `MODEL`      | `gpt-4o`     | Any [LiteLLM](https://docs.litellm.ai/docs/providers) model string |
| `AGENT_NAME` | `adk_agent`  | Must be a valid Python identifier (ADK requirement)           |

## Joining a simulation

Once running, point simserver at it:

```bash
curl -X POST localhost:8080/simulate -d '{
  "duration_ms": 10000, "tick_ms": 1000,
  "agents": [{"id": "alice", "name": "Alice", "location": "loc_default",
              "traits": {"friendliness": 7, "curiosity": 3},
              "goals": ["explore", "trade"], "energy": 100, "credits": 10,
              "agent_card_url": "http://localhost:9001"}]
}'
```

## How it works

`SYSTEM_PROMPT` in `main.py` documents the exact world-view JSON the agent
receives each tick and the action JSON it must reply with (see
`internal/world/view.go` and `internal/world/action.go` for the authoritative
Go-side schema). The LLM's raw text response is expected to be that action
JSON with nothing else around it — `to_a2a()` forwards it back to simserver
as the A2A response, and simserver's `A2AAgentClient.Decide` parses it
directly.

## Compatibility note

This example pins `a2a-sdk>=0.3.4,<0.4` (via `google-adk[a2a]`), which speaks
A2A protocol `0.3.0` — the version simserver's `a2a-go` client (non-`/v2`
module) also speaks. Newer `a2a-sdk` releases (1.x) implement a different,
not-yet-widely-adopted protocol revision and will not interoperate.
