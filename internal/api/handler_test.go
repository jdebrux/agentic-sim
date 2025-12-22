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
	_ = cfg
	_ = duration
	return s.startID, s.startErr
}

func (s *stubManager) Status(id string) (simulation.RunStatus, bool) {
	rs, ok := s.status[id]
	return rs, ok
}

func (s *stubManager) Metrics() simulation.ManagerMetrics {
	return s.metrics
}

func TestHealth(t *testing.T) {
	m := &stubManager{}
	h := NewHandler(m)
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
	h := NewHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	body := []byte(`{"duration_ms":10,"use_simple_runner":true}`)
	req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"id":"run-1"`)) {
		t.Fatalf("expected run id in response, got %s", rec.Body.String())
	}
}

func TestSimulateStatus(t *testing.T) {
	m := &stubManager{
		status: map[string]simulation.RunStatus{
			"run-1": {ID: "run-1", State: "completed", Ticks: 3, Events: 5},
		},
	}
	h := NewHandler(m)
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
	h := NewHandler(m)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/simulate/missing", nil)
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
	h := NewHandler(m)
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
