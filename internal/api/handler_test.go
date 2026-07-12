package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jdebrux/agentic-sim/internal/simulation"
	"github.com/jdebrux/agentic-sim/internal/world"
)

type stubManager struct {
	startID  string
	startErr error
	lastCfg  simulation.EngineConfig

	status  map[string]simulation.RunStatus
	metrics simulation.ManagerMetrics

	events map[string][]world.Event
	subs   map[string]chan world.Event

	deletable   map[string]bool
	deletedIDs  []string
	shutdownErr error
}

func (s *stubManager) Start(ctx context.Context, cfg simulation.EngineConfig, duration time.Duration) (string, error) {
	_ = ctx
	_ = duration
	s.lastCfg = cfg
	return s.startID, s.startErr
}

func (s *stubManager) Status(id string) (simulation.RunStatus, bool) {
	rs, ok := s.status[id]
	return rs, ok
}

func (s *stubManager) Metrics() simulation.ManagerMetrics {
	return s.metrics
}

func (s *stubManager) Events(id string) ([]world.Event, bool) {
	events, ok := s.events[id]
	return events, ok
}

func (s *stubManager) Subscribe(id string) ([]world.Event, <-chan world.Event, func(), bool) {
	ch, ok := s.subs[id]
	if !ok {
		return nil, nil, nil, false
	}
	return s.events[id], ch, func() {}, true
}

func (s *stubManager) Delete(id string) bool {
	ok := s.deletable[id]
	if ok {
		s.deletedIDs = append(s.deletedIDs, id)
	}
	return ok
}

func (s *stubManager) Shutdown(ctx context.Context) error {
	_ = ctx
	return s.shutdownErr
}

func newTestHandler(m simulation.Manager) *Handler {
	return NewHandler(m, time.Second)
}

func TestHealth(t *testing.T) {
	m := &stubManager{metrics: simulation.ManagerMetrics{Running: 3}}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Fatalf("expected status ok in body, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"active_run_count":3`)) {
		t.Fatalf("expected active_run_count in body, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"uptime_seconds"`)) {
		t.Fatalf("expected uptime_seconds in body, got %s", rec.Body.String())
	}
}

func TestSimulate_StartsRun(t *testing.T) {
	m := &stubManager{startID: "run-1"}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	body := []byte(`{"duration_ms":10,"tick_ms":5,"decision_timeout_ms":3}`)
	req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"id":"run-1"`)) {
		t.Fatalf("expected run id in response, got %s", rec.Body.String())
	}
	if m.lastCfg.Tick != 5*time.Millisecond {
		t.Fatalf("expected tick 5ms, got %v", m.lastCfg.Tick)
	}
	if m.lastCfg.DecisionTimeout != 3*time.Millisecond {
		t.Fatalf("expected decision timeout 3ms, got %v", m.lastCfg.DecisionTimeout)
	}
}

func TestSimulate_CustomAgentsAndLocations(t *testing.T) {
	m := &stubManager{startID: "run-1"}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	body := []byte(`{
		"duration_ms": 100,
		"tick_ms": 10,
		"agents": [
			{"id": "alice", "name": "Alice", "location": "plaza", "traits": {"friendliness": 7, "curiosity": 3}, "goals": ["trade"], "energy": 80, "credits": 20}
		],
		"locations": [
			{"id": "plaza", "name": "Central Plaza"},
			{"id": "market", "name": "Marketplace"}
		]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(m.lastCfg.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(m.lastCfg.Agents))
	}
	alice := m.lastCfg.Agents[0]
	if alice.ID != "alice" || alice.Name != "Alice" || alice.Location != "plaza" {
		t.Fatalf("expected alice at plaza, got %+v", alice)
	}
	if len(m.lastCfg.Locations) != 2 {
		t.Fatalf("expected 2 locations, got %d", len(m.lastCfg.Locations))
	}
	if m.lastCfg.Locations[0].ID != "plaza" || m.lastCfg.Locations[1].ID != "market" {
		t.Fatalf("expected plaza and market, got %+v", m.lastCfg.Locations)
	}
}

func TestSimulate_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"missing duration", `{"duration_ms":0}`, http.StatusBadRequest},
		{"negative tick", `{"duration_ms":10,"tick_ms":-1}`, http.StatusBadRequest},
		{"tick too large", `{"duration_ms":10,"tick_ms":10}`, http.StatusBadRequest},
		{"negative decision timeout", `{"duration_ms":10,"decision_timeout_ms":-1}`, http.StatusBadRequest},
		{"malformed json", `{"duration_ms":10,`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &stubManager{}
			h := newTestHandler(m)
			mux := http.NewServeMux()
			h.Register(mux)

			req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestSimulateStatus(t *testing.T) {
	m := &stubManager{
		status: map[string]simulation.RunStatus{
			"run-1": {ID: "run-1", State: "completed", Ticks: 3, Events: 5},
		},
	}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/simulate/run-1", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"ticks":3`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"events":5`)) {
		t.Fatalf("expected ticks/events in status response, got %s", rec.Body.String())
	}
}

func TestSimulateStatus_NotFound(t *testing.T) {
	m := &stubManager{status: map[string]simulation.RunStatus{}}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/simulate/missing", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSimulateStatus_MethodNotAllowed(t *testing.T) {
	m := &stubManager{status: map[string]simulation.RunStatus{}}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/simulate/run-1", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestSimulateDelete(t *testing.T) {
	m := &stubManager{deletable: map[string]bool{"run-1": true}}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodDelete, "/simulate/run-1", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if len(m.deletedIDs) != 1 || m.deletedIDs[0] != "run-1" {
		t.Fatalf("expected run-1 to be deleted, got %v", m.deletedIDs)
	}
}

func TestSimulateDelete_NotFound(t *testing.T) {
	m := &stubManager{deletable: map[string]bool{}}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodDelete, "/simulate/missing", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestMetrics(t *testing.T) {
	m := &stubManager{
		metrics: simulation.ManagerMetrics{
			TotalRuns:   2,
			Running:     1,
			Completed:   1,
			Errored:     0,
			TotalTicks:  10,
			TotalEvents: 20,
		},
	}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"total_runs":2`)) {
		t.Fatalf("expected metrics in response, got %s", rec.Body.String())
	}
}

func TestMetrics_MethodNotAllowed(t *testing.T) {
	m := &stubManager{}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestSimulateEvents(t *testing.T) {
	m := &stubManager{
		events: map[string][]world.Event{
			"run-1": {{ID: "evt-1", Type: "speak", ActorID: "alice"}},
		},
	}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/simulate/run-1/events", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"id":"evt-1"`)) {
		t.Fatalf("expected event in body, got %s", rec.Body.String())
	}
}

func TestSimulateEvents_NotFound(t *testing.T) {
	m := &stubManager{events: map[string][]world.Event{}}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/simulate/missing/events", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSimulateStream(t *testing.T) {
	ch := make(chan world.Event, 4)
	m := &stubManager{
		events: map[string][]world.Event{
			"run-1": {{ID: "evt-0", Type: "idle", ActorID: "alice"}},
		},
		subs: map[string]chan world.Event{"run-1": ch},
	}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	ch <- world.Event{ID: "evt-1", Type: "speak", ActorID: "alice"}
	close(ch)

	req := httptest.NewRequest(http.MethodGet, "/simulate/run-1/stream", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream content type, got %s", ct)
	}
	body := rec.Body.String()
	if !bytes.Contains([]byte(body), []byte(`"id":"evt-0"`)) {
		t.Fatalf("expected backlog event in stream, got %s", body)
	}
	if !bytes.Contains([]byte(body), []byte(`"id":"evt-1"`)) {
		t.Fatalf("expected live event in stream, got %s", body)
	}
}

func TestSimulateStream_NotFound(t *testing.T) {
	m := &stubManager{subs: map[string]chan world.Event{}}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/simulate/missing/stream", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAgentCard(t *testing.T) {
	m := &stubManager{}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"name":"agentic-sim"`)) {
		t.Fatalf("expected agentic-sim name in body, got %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"preferredTransport":"JSONRPC"`)) {
		t.Fatalf("expected preferredTransport in body, got %s", rec.Body.String())
	}
}

func TestAgentCard_MethodNotAllowed(t *testing.T) {
	m := &stubManager{}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/.well-known/agent-card.json", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
