# agentic-sim

[![CI](https://github.com/jdebrux/agentic-sim/actions/workflows/ci.yml/badge.svg)](https://github.com/jdebrux/agentic-sim/actions/workflows/ci.yml)
[![Go Reference](https://img.shields.io/badge/go-1.25-blue)](go.mod)

A Go framework for simulating autonomous AI agents interacting in a shared world. The Go server hosts world state, enforces physics, and runs the clock; agent *reasoning* happens entirely in separate processes reachable over the [A2A protocol](https://github.com/a2aproject/a2a-go).

```
POST /simulate  →  world ticks  →  each agent gets a JSON world-view  →  each agent replies with a JSON action  →  simserver validates & applies it  →  repeat
```

## Why

Most agent-simulation frameworks bundle the world engine and the agent's reasoning into one process. In this project we split them deliberately:

- **The simulator never calls an LLM.** It has no concept of prompts, models, or providers — it just serialises world state to JSON and applies whatever action comes back.
- **Agents are opaque, external, and swappable.** A Python ADK agent backed by a Gemini model, a LangGraph agent backed by a Claude model, and a hand-rolled Go agent with no LLM at all can all join the same simulation, side by side, because they all speak the same A2A + JSON-action contract.
- **The world is authoritative.** Agents propose actions; the server validates co-location, energy costs, and location rules, and is the only thing that mutates state. A malformed or slow agent degrades to `idle` so it never crashes the tick loop.


## Quickstart

### Option A — full stack with reference agents (Docker Compose)

```bash
# API_KEY/MODEL are generic LiteLLM passthroughs — any provider works here
# with no code changes; see "Building your own agent" below for OpenRouter
# and self-hosted examples. This example assumes OpenAI.
export API_KEY=sk-...          # your provider's API key
export ADK_MODEL=gpt-4o        # model string for the ADK agent (Alice)
export LANGGRAPH_MODEL=gpt-4o  # model string for the LangGraph agent (Bob)

docker compose up -d --build
```

This starts `simserver` (`:8080`) plus two LLM-backed reference agents, `adk-agent` (`:9001`, Google ADK) and `langgraph-agent` (`:9002`, LangGraph). Then kick off a run:

```bash
curl -X POST localhost:8080/simulate -d '{
  "duration_ms": 30000, "tick_ms": 1000,
  "agents": [
    {"id": "alice", "name": "Alice", "location": "loc_default",
     "traits": {"friendliness": 7, "curiosity": 3},
     "goals": ["explore", "trade"], "energy": 100, "credits": 10,
     "agent_card_url": "http://adk-agent:9001"},
    {"id": "bob", "name": "Bob", "location": "loc_default",
     "traits": {"friendliness": 4, "curiosity": 8},
     "goals": ["gather_info"], "energy": 100, "credits": 5,
     "agent_card_url": "http://langgraph-agent:9002"}
  ]
}'
```

Watch it happen live:

```bash
curl -N localhost:8080/simulate/<id>/stream
```

### Option B — simserver only (no agents)

```bash
go build -o ./bin/simserver ./cmd/simserver
PORT=8080 TICK_MS=1000 ./bin/simserver
```

Useful for exercising the API/engine on its own, but a run with no reachable agents just ticks forward producing no actions — see [Configuring a run](#configuring-a-run) for what an agent needs to actually participate.

## HTTP API

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | `{"status", "uptime_seconds", "active_run_count"}` |
| `/.well-known/agent-card.json` | GET | simserver's own A2A Agent Card |
| `/simulate` | POST | Start a run; returns `{"id", "status"}` immediately (the run continues after the response) |
| `/simulate/{id}` | GET | Run status: `{"id", "status", "ticks", "events", "error"}` |
| `/simulate/{id}` | DELETE | Stop (if active) and forget a run — `204`/`404` |
| `/simulate/{id}/events` | GET | Full recorded event history for the run |
| `/simulate/{id}/stream` | GET | Server-Sent Events: replays history, then streams new events live until the run ends |
| `/metrics` | GET | Aggregate counters across all runs (total/running/completed, ticks, events) |

## Configuring a run

`POST /simulate` takes agents and locations inline — nothing is hardcoded beyond a fallback default (two placeholder agents, three placeholder locations) used only when a field is omitted entirely.

```json
{
  "duration_ms": 30000,
  "tick_ms": 1000,
  "agents": [
    {
      "id": "alice",
      "name": "Alice",
      "location": "loc_default",
      "traits": {"friendliness": 7, "curiosity": 3},
      "goals": ["explore", "trade"],
      "energy": 100,
      "credits": 10,
      "agent_card_url": "http://localhost:9001"
    }
  ],
  "locations": [
    {"id": "loc_default", "name": "Central Plaza"},
    {"id": "loc_market", "name": "Marketplace"},
    {"id": "loc_park", "name": "Park"}
  ]
}
```

An agent only actually participates (gets asked to decide each tick) if `agent_card_url` points at a reachable A2A server. Agents without one are seeded into the world but never act.

Each tick, every agent's `Decide` call runs concurrently and is bounded by `decision_timeout_ms` (default: the tick duration itself, so no agent can outlast its own tick). An agent that errors or blows past its timeout degrades to `idle` for that tick — it never stalls the others or the run as a whole.

### World rules

| Action | Effect | Energy | Constraints |
|---|---|---|---|
| `move` | Relocate to `location` | −5 | Destination must exist |
| `speak` | Broadcast `message` | −1 | — |
| `greet` | Message a specific `target_id` | −1 | Must be co-located with target |
| `interact` | Generic interaction with `target_id` | −1 | Must be co-located with target |
| `trade` | Transfer 1 credit to `target_id` | −3 | Must be co-located with target, and both in `loc_market` |
| `rest` | Recover energy | +10 | — |
| `idle` | No-op | −1 | Also the fallback for errors/timeouts/malformed replies |

Energy is clamped to `[0, 100]`. Every action — including failed ones — produces an `Event`; failures carry an `error` field in `payload` rather than aborting the tick.

## Building your own agent

Any process that can serve an A2A Agent Card and respond to `SendMessage` qualifies — no SDK required. Each tick, your agent receives a `WorldView` as a JSON text message:

```json
{
  "world_id": "world-1",
  "tick": 5,
  "self": {"id": "alice", "name": "Alice", "location": "loc_default", "traits": {"friendliness": 7, "curiosity": 3}, "goals": ["explore", "trade"], "energy": 85, "credits": 9},
  "other_agents": [{"id": "bob", "name": "Bob", "location": "loc_market", "energy": 92, "credits": 5}],
  "locations": [{"id": "loc_default", "name": "Central Plaza"}, {"id": "loc_market", "name": "Marketplace"}],
  "recent_events": [{"type": "move", "actor_id": "bob", "tick": 4}],
  "at_market": false
}
```

...and must reply with exactly one JSON object describing an action:

```json
{"action": "move", "location": "loc_market", "reason": "Heading to market to trade"}
```

Valid `action` values: `move`, `speak`, `greet`, `interact`, `trade`, `rest`, `idle` (see the field reference in `internal/world/action.go`). Malformed, non-JSON, or slow replies are treated as `idle` with a `reason` explaining why — they never fail the run.

Three reference implementations exist in [`examples/`](examples/):

| Agent | Framework | Notes |
|---|---|---|
| [`examples/adk-agent`](examples/adk-agent) | Google ADK | Uses ADK's `to_a2a()` helper — minimal boilerplate |
| [`examples/langgraph-agent`](examples/langgraph-agent) | LangGraph | Hand-wired A2A server around a one-node graph |
| [`examples/go-agent`](examples/go-agent) | none (Go, no LLM) | Deterministic rule-based agent proving "no SDK required"; also backs the A2A end-to-end test in `internal/api/e2e_test.go` |

Both route through [LiteLLM](https://docs.litellm.ai/docs/providers) via three generic env vars, so the same agent works against OpenAI, OpenRouter, or a self-hosted OpenAI-compatible endpoint (Ollama, vLLM, ...) with no code changes:

```bash
# OpenAI directly
export API_KEY=sk-...

# OpenRouter
export MODEL=openrouter/anthropic/claude-3.5-sonnet
export API_KEY=sk-or-...

# Self-hosted OpenAI-compatible server
export MODEL=openai/llama3
export API_BASE=http://localhost:11434/v1
export API_KEY=unused
```

See each agent's `README.md` for the exact configuration reference.

## Observability

Every tick, action, and A2A round-trip is an OpenTelemetry span; ticks, actions, decision latency/errors, and A2A round-trip latency are recorded as metrics. Traces print to stdout by default, or export via OTLP/gRPC if `OTEL_EXPORTER_OTLP_ENDPOINT` is set. For a live view of a specific run without a tracing backend, `/simulate/{id}/stream` is usually the faster path.

## Development

```bash
go build -o ./bin/simserver ./cmd/simserver   # build
go test ./... -v -cover                        # test everything (what CI runs)
go test ./internal/simulation/... -run TestEngineDefaultAgents -v   # single test
go vet ./...                                   # must pass (CI enforces)
golangci-lint run ./...                        # advisory only (CI tolerates failures)
docker build -t simserver .                    # multi-stage, distroless nonroot image
```

Branch names pushed to the remote (other than `main`) must match `^(feature|fix|chore|docs|refactor|test|perf)/[a-z0-9._-]+$`.

## Project layout

```
cmd/simserver/       entrypoint: telemetry → config → manager → HTTP server
internal/config/     env-var config (PORT, TICK_MS only)
internal/api/        HTTP handlers, request/response types, A2A agent-card serving
internal/simulation/ Engine (tick loop, rules), Manager (run lifecycle), A2A client
internal/world/      pure domain types: World, AgentState, Location, Event, WorldView, AgentAction
internal/telemetry/  OpenTelemetry setup and metric definitions
examples/            reference A2A agent implementations
```

## Environment variables

| Variable | Default | Applies to |
|---|---|---|
| `PORT` | `8080` | `simserver` |
| `TICK_MS` | `1000` | `simserver` — default tick interval |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | *(stdout)* | `simserver` — OpenTelemetry collector |
| `MODEL` | `gpt-4o` | example agents — any [LiteLLM](https://docs.litellm.ai/docs/providers) model string |
| `API_KEY` | *(unset)* | example agents — falls back to LiteLLM's provider auto-detection if unset |
| `API_BASE` | *(unset)* | example agents — custom endpoint for self-hosted models |
