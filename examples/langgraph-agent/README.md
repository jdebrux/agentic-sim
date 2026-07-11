# LangGraph Agent

An A2A agent for [agentic-sim](../../README.md) built with [LangGraph](https://langchain-ai.github.io/langgraph/).
Unlike the ADK example, LangGraph has no built-in A2A export helper, so this
example wires the A2A server (`AgentExecutor` → `DefaultRequestHandler` →
`A2AStarletteApplication`) by hand around a one-node graph.

## Setup

```bash
uv sync
export OPENAI_API_KEY=sk-...   # or ANTHROPIC_API_KEY / GEMINI_API_KEY, matched to MODEL below
uv run main.py
```

The agent listens on `:9002` and serves its Agent Card at
`http://localhost:9002/.well-known/agent-card.json`.

## Configuration

| Env var      | Default            | Purpose                                                    |
|--------------|---------------------|-------------------------------------------------------------|
| `HOST`       | `localhost`         | Hostname advertised in the Agent Card                        |
| `PORT`       | `9002`              | Port to bind and advertise                                   |
| `MODEL`      | `gpt-4o`            | Any [LiteLLM](https://docs.litellm.ai/docs/providers) model string (via `langchain-litellm`) |
| `AGENT_NAME` | `langgraph-agent`   | Name shown in the Agent Card                                  |

## Joining a simulation

```bash
curl -X POST localhost:8080/simulate -d '{
  "duration_ms": 10000, "tick_ms": 1000,
  "agents": [{"id": "bob", "name": "Bob", "location": "loc_default",
              "traits": {"friendliness": 4, "curiosity": 8},
              "goals": ["gather_info"], "energy": 100, "credits": 5,
              "agent_card_url": "http://localhost:9002"}]
}'
```

## How it works

`main.py` builds a single-node `StateGraph` (`decide`) that sends the
incoming world-view JSON to the chat model alongside a system prompt
documenting the world-view/action contract (matching
`internal/world/view.go` / `internal/world/action.go`). The model's raw text
response is expected to be the action JSON with nothing else around it;
`LangGraphAgentExecutor.execute` runs the graph once per A2A request and
enqueues the result as the agent's reply.

## Compatibility note

This example pins `a2a-sdk>=0.3.4,<0.4`, matching A2A protocol `0.3.0` — the
version simserver's `a2a-go` client (non-`/v2` module) also speaks. Newer
`a2a-sdk` releases (1.x) implement a different, not-yet-widely-adopted
protocol revision and will not interoperate.
