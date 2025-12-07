package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jdebrux/agentic-sim/internal/simulation"
)

// SimulationService abstracts running a simulation for a given duration.
type SimulationService interface {
	RunSimulation(r *http.Request, cfg simulation.EngineConfig, duration time.Duration) (ticks int64, events int, err error)
}

// Handler wires HTTP endpoints to services.
type Handler struct {
	Sim SimulationService
}

func NewHandler(sim SimulationService) *Handler {
	return &Handler{Sim: sim}
}

// Register attaches endpoints to a mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/simulate", h.simulate)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type simulateRequest struct {
	DurationMs      int64 `json:"duration_ms"`
	TickMs          int64 `json:"tick_ms"`
	UseSimpleRunner bool  `json:"use_simple_runner"`
}

type simulateResponse struct {
	Status string `json:"status"`
	Ticks  int64  `json:"ticks"`
	Events int    `json:"events"`
}

func (h *Handler) simulate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req simulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.DurationMs <= 0 {
		http.Error(w, "duration_ms must be > 0", http.StatusBadRequest)
		return
	}

	cfg := simulation.EngineConfig{
		UseSimpleRunner: req.UseSimpleRunner,
	}
	if req.TickMs > 0 {
		cfg.Tick = time.Duration(req.TickMs) * time.Millisecond
	}

	ticks, events, err := h.Sim.RunSimulation(r, cfg, time.Duration(req.DurationMs)*time.Millisecond)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, simulateResponse{
		Status: "completed",
		Ticks:  ticks,
		Events: events,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
