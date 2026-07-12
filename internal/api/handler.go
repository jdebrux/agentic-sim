package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jdebrux/agentic-sim/internal/simulation"
	"github.com/jdebrux/agentic-sim/internal/world"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Handler wires HTTP endpoints to services.
type Handler struct {
	Manager     simulation.Manager
	DefaultTick time.Duration
	StartedAt   time.Time
}

func NewHandler(m simulation.Manager, defaultTick time.Duration) *Handler {
	return &Handler{
		Manager:     m,
		DefaultTick: defaultTick,
		StartedAt:   time.Now(),
	}
}

// Register attaches endpoints to a mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/.well-known/agent-card.json", h.agentCard)
	mux.HandleFunc("/simulate", h.simulate)
	mux.HandleFunc("/simulate/", h.simulateRoute)
	mux.HandleFunc("/metrics", h.metrics)
}

type healthResponse struct {
	Status         string  `json:"status"`
	UptimeSeconds  float64 `json:"uptime_seconds"`
	ActiveRunCount int64   `json:"active_run_count"`
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:         "ok",
		UptimeSeconds:  time.Since(h.StartedAt).Seconds(),
		ActiveRunCount: h.Manager.Metrics().Running,
	})
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
	DurationMs        int64                `json:"duration_ms"`
	TickMs            int64                `json:"tick_ms"`
	DecisionTimeoutMs int64                `json:"decision_timeout_ms,omitempty"`
	Agents            []agentDefinition    `json:"agents,omitempty"`
	Locations         []locationDefinition `json:"locations,omitempty"`
}

type simulateResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (h *Handler) simulate(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("api")
	ctx, span := tracer.Start(r.Context(), "http.simulate")
	defer span.End()
	r = r.WithContext(ctx)

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req simulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	span.SetAttributes(
		attribute.Int64("duration_ms", req.DurationMs),
		attribute.Int("agent.count", len(req.Agents)),
	)

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
	if req.DecisionTimeoutMs < 0 {
		http.Error(w, "decision_timeout_ms must be >= 0", http.StatusBadRequest)
		return
	}

	cfg := simulation.EngineConfig{
		Tick: h.DefaultTick,
	}
	if req.TickMs > 0 {
		cfg.Tick = time.Duration(req.TickMs) * time.Millisecond
	}
	if req.DecisionTimeoutMs > 0 {
		cfg.DecisionTimeout = time.Duration(req.DecisionTimeoutMs) * time.Millisecond
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "start simulation")
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

// simulateRoute dispatches everything under /simulate/{id}[/stream|/events].
func (h *Handler) simulateRoute(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/simulate/")
	if path == "" {
		http.Error(w, "missing run id", http.StatusBadRequest)
		return
	}

	id, sub, hasSub := strings.Cut(path, "/")
	switch {
	case !hasSub:
		h.simulateByID(w, r, id)
	case sub == "stream":
		h.simulateStream(w, r, id)
	case sub == "events":
		h.simulateEvents(w, r, id)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// simulateByID handles the bare /simulate/{id} route: GET for status, DELETE
// to stop and forget the run.
func (h *Handler) simulateByID(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		h.simulateStatus(w, r, id)
	case http.MethodDelete:
		h.simulateDelete(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) simulateStatus(w http.ResponseWriter, r *http.Request, id string) {
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

// simulateDelete stops (if still running) and forgets a run, freeing its
// history and subscribers.
func (h *Handler) simulateDelete(w http.ResponseWriter, r *http.Request, id string) {
	if ok := h.Manager.Delete(id); !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type simulateEventsResponse struct {
	ID     string        `json:"id"`
	Events []world.Event `json:"events"`
}

// simulateEvents returns the full event history recorded for a run so far,
// for post-hoc analysis.
func (h *Handler) simulateEvents(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	events, ok := h.Manager.Events(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, simulateEventsResponse{ID: id, Events: events})
}

// simulateStream streams a run's events as Server-Sent Events in real time,
// starting with any events already recorded before the client connected.
func (h *Handler) simulateStream(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	backlog, live, unsubscribe, ok := h.Manager.Subscribe(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	for _, evt := range backlog {
		writeSSEEvent(w, evt)
	}
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-live:
			if !ok {
				return
			}
			writeSSEEvent(w, evt)
			flusher.Flush()
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, evt world.Event) {
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, data)
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
