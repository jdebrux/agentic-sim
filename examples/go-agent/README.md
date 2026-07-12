# Go Agent

A minimal, deterministic A2A agent for [agentic-sim](../../README.md), written
in Go with no SDK and no LLM. It exists to prove the "any process that can
serve an A2A Agent Card qualifies" claim in the main README, to give
contributors a zero-dependency local dev target, and to back the end-to-end
test in `internal/api/e2e_test.go`.

Its entire strategy: if not at the market, move there; once at the market,
trade with the first other agent found there, or rest if alone. See
`goagent/agent.go`'s `Decide` function.

## Running it

```bash
go run ./examples/go-agent
```

The agent listens on `:9003` and serves its Agent Card at
`http://localhost:9003/.well-known/agent-card.json`.

## Configuration

| Env var | Default | Purpose                        |
|---------|---------|---------------------------------|
| `PORT`  | `9003`  | Port to bind and advertise      |

## Joining a simulation

```bash
curl -X POST localhost:8080/simulate -d '{
  "duration_ms": 10000, "tick_ms": 1000,
  "agents": [{"id": "carol", "name": "Carol", "location": "loc_default",
              "traits": {"friendliness": 5, "curiosity": 5},
              "goals": ["trade"], "energy": 100, "credits": 5,
              "agent_card_url": "http://localhost:9003"}]
}'
```

## How it works

`goagent/agent.go` implements `a2asrv.AgentExecutor` directly against
`github.com/a2aproject/a2a-go/a2asrv` — the Go-native server counterpart to
the client already used in `internal/simulation/a2a_client.go`. `Execute`
extracts the incoming world-view JSON from the request message's text part,
unmarshals it into `world.WorldView`, calls the pure `Decide` function, and
writes the resulting `world.AgentAction` back as a text-part reply — the same
contract documented in `internal/world/view.go` / `internal/world/action.go`
and used by the LLM-backed examples.

`main.go` is a thin CLI wrapper around `goagent.NewAgentCard` and
`goagent.Register`; both are exported so tests (and other Go code in this
module) can mount the same agent onto an `httptest.Server` without spawning a
subprocess.
