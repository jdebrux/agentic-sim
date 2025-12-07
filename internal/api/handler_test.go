package api

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jdebrux/agentic-sim/internal/simulation"
)

type stubSimService struct {
	lastCfg      simulation.EngineConfig
	lastDuration time.Duration
	ticks        int64
	events       int
	err          error
}

func (s *stubSimService) RunSimulation(r *http.Request, cfg simulation.EngineConfig, duration time.Duration) (int64, int, error) {
	_ = r
	s.lastCfg = cfg
	s.lastDuration = duration
	return s.ticks, s.events, s.err
}

func TestHealth(t *testing.T) {
	svc := &stubSimService{}
	h := NewHandler(svc)
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

func TestSimulate_Success(t *testing.T) {
	svc := &stubSimService{ticks: 3, events: 5}
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.Register(mux)

	body := []byte(`{"duration_ms":10,"tick_ms":2,"use_simple_runner":true}`)
	req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if svc.lastDuration != 10*time.Millisecond {
		t.Fatalf("expected duration 10ms, got %v", svc.lastDuration)
	}
	if svc.lastCfg.Tick != 2*time.Millisecond {
		t.Fatalf("expected tick 2ms, got %v", svc.lastCfg.Tick)
	}
	if !svc.lastCfg.UseSimpleRunner {
		t.Fatalf("expected UseSimpleRunner to be true")
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"ticks":3`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"events":5`)) {
		t.Fatalf("expected ticks/events in response, got %s", rec.Body.String())
	}
}

func TestSimulate_InvalidRequest(t *testing.T) {
	svc := &stubSimService{}
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.Register(mux)

	body := []byte(`{"duration_ms":0}`)
	req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSimulate_ServiceError(t *testing.T) {
	svc := &stubSimService{err: errors.New("boom")}
	h := NewHandler(svc)
	mux := http.NewServeMux()
	h.Register(mux)

	body := []byte(`{"duration_ms":5}`)
	req := httptest.NewRequest(http.MethodPost, "/simulate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
