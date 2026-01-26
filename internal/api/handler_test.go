package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jdebrux/agentic-sim/internal/simulation"
)

type stubManager struct {
	startID  string
	startErr error

	status  map[string]simulation.RunStatus
	metrics simulation.ManagerMetrics
}

func (s *stubManager) Start(ctx context.Context, cfg simulation.EngineConfig, duration time.Duration) (string, error) {
	_ = ctx
	_ = duration
	s.metrics.LastMode = cfg.RunnerMode
	return s.startID, s.startErr
}

func (s *stubManager) Status(id string) (simulation.RunStatus, bool) {
	rs, ok := s.status[id]
	return rs, ok
}

func (s *stubManager) Metrics() simulation.ManagerMetrics {
	return s.metrics
}

func newTestHandler(m simulation.Manager) *Handler {
	return NewHandler(m, time.Second, "simple", "mock")
}

func TestHealth(t *testing.T) {
	m := &stubManager{}
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
}

func TestSimulate_StartsRun(t *testing.T) {
	m := &stubManager{startID: "run-1"}
	h := newTestHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	body := []byte(`{"duration_ms":10,"runner_mode":"rule"}`)
	req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"id":"run-1"`)) {
		t.Fatalf("expected run id in response, got %s", rec.Body.String())
	}
	if m.metrics.LastMode == "" {
		t.Fatalf("expected runner mode recorded on start")
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
		{"invalid runner mode", `{"duration_ms":10,"runner_mode":"invalid"}`, http.StatusBadRequest},
		{"invalid reasoner provider", `{"duration_ms":10,"reasoner_provider":"invalid"}`, http.StatusBadRequest},
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
			"run-1": {ID: "run-1", State: "completed", Ticks: 3, Events: 5, Mode: "rule"},
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
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"runner_mode":"rule"`)) {
		t.Fatalf("expected runner_mode in status response, got %s", rec.Body.String())
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

func TestMetrics(t *testing.T) {
	m := &stubManager{
		metrics: simulation.ManagerMetrics{
			TotalRuns:   2,
			Running:     1,
			Completed:   1,
			Errored:     0,
			TotalTicks:  10,
			TotalEvents: 20,
			LastMode:    "rule",
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
