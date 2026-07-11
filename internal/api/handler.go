package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jdebrux/agentic-sim/internal/simulation"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// Handler wires HTTP endpoints to services.
type Handler struct {
	Manager     simulation.Manager
	DefaultTick time.Duration
}

func NewHandler(m simulation.Manager, defaultTick time.Duration) *Handler {
	return &Handler{
		Manager:     m,
		DefaultTick: defaultTick,
	}
}

// Register attaches endpoints to a mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/.well-known/agent-card", h.agentCard)
	mux.HandleFunc("/simulate", h.simulate)
	mux.HandleFunc("/simulate/", h.simulateStatus)
	mux.HandleFunc("/metrics", h.metrics)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

type agentDefinition struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Location     string           `json:"location"`
	Traits       world.Traits     `json:"traits"`
	Goals        []string         `json:"goals"`
	Energy       int              `json:"energy"`
	Credits      int              `json:"credits"`
	AgentCardURL string           `json:"agent_card_url,omitempty"`
	AgentCard    *world.AgentCard `json:"agent_card,omitempty"`
}

type locationDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type simulateRequest struct {
	DurationMs int64                `json:"duration_ms"`
	TickMs     int64                `json:"tick_ms"`
	Agents     []agentDefinition    `json:"agents,omitempty"`
	Locations  []locationDefinition `json:"locations,omitempty"`
}

type simulateResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
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
	if req.TickMs < 0 {
		http.Error(w, "tick_ms must be >= 0", http.StatusBadRequest)
		return
	}
	if req.TickMs > 0 && req.TickMs >= req.DurationMs {
		http.Error(w, "tick_ms must be less than duration_ms", http.StatusBadRequest)
		return
	}

	cfg := simulation.EngineConfig{
		Tick: h.DefaultTick,
	}
	if req.TickMs > 0 {
		cfg.Tick = time.Duration(req.TickMs) * time.Millisecond
	}
	if len(req.Agents) > 0 {
		cfg.Agents = make([]simulation.AgentRegistration, len(req.Agents))
		for i, a := range req.Agents {
			cfg.Agents[i] = simulation.AgentRegistration{
				ID:           a.ID,
				Name:         a.Name,
				Location:     a.Location,
				Traits:       a.Traits,
				Goals:        a.Goals,
				Energy:       a.Energy,
				Credits:      a.Credits,
				AgentCardURL: a.AgentCardURL,
			}
		}
	}
	if len(req.Locations) > 0 {
		cfg.Locations = make([]world.Location, len(req.Locations))
		for i, l := range req.Locations {
			cfg.Locations[i] = world.Location{
				ID:          l.ID,
				Name:        l.Name,
				Description: l.Description,
			}
		}
	}

	runID, err := h.Manager.Start(r.Context(), cfg, time.Duration(req.DurationMs)*time.Millisecond)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, simulateResponse{
		ID:     runID,
		Status: "running",
	})
}

type simulateStatusResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Ticks  int64  `json:"ticks"`
	Events int64  `json:"events"`
	Error  string `json:"error,omitempty"`
}

func (h *Handler) simulateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Path[len("/simulate/"):]
	if id == "" {
		http.Error(w, "missing run id", http.StatusBadRequest)
		return
	}

	status, ok := h.Manager.Status(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	resp := simulateStatusResponse{
		ID:     status.ID,
		Status: status.State,
		Ticks:  status.Ticks,
		Events: status.Events,
	}
	if status.Error != "" {
		resp.Error = status.Error
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	m := h.Manager.Metrics()
	writeJSON(w, http.StatusOK, m)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
