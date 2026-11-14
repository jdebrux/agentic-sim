package api_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jdebrux/agentic-sim/examples/go-agent/goagent"
	"github.com/jdebrux/agentic-sim/internal/api"
	"github.com/jdebrux/agentic-sim/internal/simulation"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// TestE2E_RuleAgentsMoveAndTrade drives the real production stack — Handler,
// InMemoryManager, Engine, and A2AAgentClient — over real HTTP/A2A sockets
// against two deterministic rule-based agents (no LLM, no API key). It
// exists to give the A2A wire path (card resolution -> Decide round-trip ->
// action parsing -> world mutation -> SSE fan-out) real CI coverage, which
// today only happens with a live LLM behind examples/adk-agent or
// examples/langgraph-agent.
func TestE2E_RuleAgentsMoveAndTrade(t *testing.T) {
	agentAURL := startAgentServer(t)
	agentBURL := startAgentServer(t)
	simURL := startSimserver(t)

	client := &http.Client{Timeout: 5 * time.Second}

	reqBody := simulateRequestBody{
		DurationMs: 150,
		TickMs:     20,
		Agents: []agentBody{
			{ID: "a", Name: "A", Location: "loc_default", Energy: 100, Credits: 10, AgentCardURL: agentAURL},
			{ID: "b", Name: "B", Location: "loc_default", Energy: 100, Credits: 10, AgentCardURL: agentBURL},
		},
	}
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	resp, err := client.Post(simURL+"/simulate", "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		t.Fatalf("POST /simulate: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 from /simulate, got %d: %s", resp.StatusCode, b)
	}
	var simResp simulateResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&simResp); err != nil {
		t.Fatalf("decode /simulate response: %v", err)
	}
	if simResp.ID == "" {
		t.Fatalf("expected a run id in /simulate response")
	}

	streamResp, err := client.Get(simURL + "/simulate/" + simResp.ID + "/stream")
	if err != nil {
		t.Fatalf("GET /simulate/{id}/stream: %v", err)
	}
	defer streamResp.Body.Close()

	events := readSSEEvents(t, streamResp.Body)
	if len(events) == 0 {
		t.Fatalf("expected at least one event from the stream")
	}

	movedWithoutError := map[string]bool{}
	sawSuccessfulTrade := false
	var lastSnapshotAgents []string
	tickEventCount := 0
	for _, evt := range events {
		_, hasErr := evt.Payload["error"]
		switch evt.Type {
		case "move":
			if !hasErr {
				movedWithoutError[evt.ActorID] = true
			}
		case "trade":
			if _, ok := evt.Payload["credits_transferred"]; ok && !hasErr {
				sawSuccessfulTrade = true
			}
		case world.EventTypeTick:
			tickEventCount++
			lastSnapshotAgents = snapshotAgentIDs(t, evt)
		}
	}
	if !movedWithoutError["a"] || !movedWithoutError["b"] {
		t.Fatalf("expected both agents to move to market without error, got events: %+v", events)
	}
	if !sawSuccessfulTrade {
		t.Fatalf("expected at least one successful trade event, got events: %+v", events)
	}
	if tickEventCount == 0 {
		t.Fatalf("expected at least one tick snapshot event, got events: %+v", events)
	}
	if len(lastSnapshotAgents) != 2 {
		t.Fatalf("expected the final tick snapshot to list both agents, got %v", lastSnapshotAgents)
	}

	statusResp, err := client.Get(simURL + "/simulate/" + simResp.ID)
	if err != nil {
		t.Fatalf("GET /simulate/{id}: %v", err)
	}
	defer statusResp.Body.Close()
	var status simulateStatusBody
	if err := json.NewDecoder(statusResp.Body).Decode(&status); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	if status.Status != "completed" {
		t.Fatalf("expected status completed, got %q", status.Status)
	}
	if status.Ticks <= 0 {
		t.Fatalf("expected ticks > 0, got %d", status.Ticks)
	}

	listResp, err := client.Get(simURL + "/simulate")
	if err != nil {
		t.Fatalf("GET /simulate: %v", err)
	}
	defer listResp.Body.Close()
	var list simulateListBody
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode /simulate list response: %v", err)
	}
	found := false
	for _, rs := range list.Runs {
		if rs.ID == simResp.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected GET /simulate to list run %s, got %+v", simResp.ID, list.Runs)
	}

	worldResp, err := client.Get(simURL + "/simulate/" + simResp.ID + "/world")
	if err != nil {
		t.Fatalf("GET /simulate/{id}/world: %v", err)
	}
	defer worldResp.Body.Close()
	if worldResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(worldResp.Body)
		t.Fatalf("expected 200 from /simulate/{id}/world, got %d: %s", worldResp.StatusCode, b)
	}
	var snap world.WorldSnapshot
	if err := json.NewDecoder(worldResp.Body).Decode(&snap); err != nil {
		t.Fatalf("decode world snapshot response: %v", err)
	}
	if len(snap.Agents) != 2 {
		t.Fatalf("expected 2 agents in final world snapshot, got %+v", snap.Agents)
	}
}

// snapshotAgentIDs extracts the agent IDs from a tick event's snapshot
// payload. Since the event was JSON-decoded, the payload's "snapshot" value
// is a generic map, not a typed world.WorldSnapshot — re-marshal/unmarshal
// it into the real type to inspect it.
func snapshotAgentIDs(t *testing.T, evt world.Event) []string {
	t.Helper()

	raw, ok := evt.Payload["snapshot"]
	if !ok {
		t.Fatalf("expected tick event to carry a snapshot payload, got %+v", evt.Payload)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal snapshot payload: %v", err)
	}
	var snap world.WorldSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("unmarshal snapshot payload: %v", err)
	}
	ids := make([]string, len(snap.Agents))
	for i, a := range snap.Agents {
		ids[i] = a.ID
	}
	return ids
}

// startAgentServer boots a real goagent instance on a system-chosen port and
// returns its base URL. httptest.NewUnstartedServer populates Listener (and
// therefore the port) before Start is called, which lets us build the
// agent's card — whose URL must embed that same port — before any request
// is served.
func startAgentServer(t *testing.T) string {
	t.Helper()

	mux := http.NewServeMux()
	srv := httptest.NewUnstartedServer(mux)
	agentURL := "http://" + srv.Listener.Addr().String()
	goagent.Register(mux, goagent.NewAgentCard(agentURL))
	srv.Start()
	t.Cleanup(srv.Close)

	return agentURL
}

// testDefaultTick mirrors config.Load's default tick; every request in this
// test overrides it explicitly, so its exact value doesn't matter.
const testDefaultTick = time.Second

// startSimserver boots the same production Handler+InMemoryManager+Engine
// wiring as cmd/simserver/main.go, mounted on a real httptest.Server (rather
// than httptest.NewRecorder) so the SSE stream behaves like a real streaming
// connection. Returns the server's base URL.
func startSimserver(t *testing.T) string {
	t.Helper()

	manager := simulation.NewInMemoryManager(func(cfg simulation.EngineConfig) *simulation.Engine {
		if cfg.Tick == 0 {
			cfg.Tick = testDefaultTick
		}
		return simulation.NewEngineWithConfig(cfg)
	})
	handler := api.NewHandler(manager, testDefaultTick)
	mux := http.NewServeMux()
	handler.Register(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = manager.Shutdown(ctx)
	})

	return srv.URL
}

// readSSEEvents reads an SSE body until EOF (the connection closes once the
// run completes) and decodes every "data: " line as a world.Event.
func readSSEEvents(t *testing.T, body io.Reader) []world.Event {
	t.Helper()

	var events []world.Event
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		var evt world.Event
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			t.Fatalf("unmarshal SSE event %q: %v", data, err)
		}
		events = append(events, evt)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read SSE stream: %v", err)
	}
	return events
}

type simulateRequestBody struct {
	DurationMs int64       `json:"duration_ms"`
	TickMs     int64       `json:"tick_ms"`
	Agents     []agentBody `json:"agents"`
}

type agentBody struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Location     string `json:"location"`
	Energy       int    `json:"energy"`
	Credits      int    `json:"credits"`
	AgentCardURL string `json:"agent_card_url"`
}

type simulateResponseBody struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type simulateStatusBody struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Ticks  int64  `json:"ticks"`
	Events int64  `json:"events"`
	Error  string `json:"error,omitempty"`
}

type simulateListBody struct {
	Runs []simulateStatusBody `json:"runs"`
}
