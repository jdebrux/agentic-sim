package world

import (
	"fmt"
	"log/slog"
)

// AgentState holds in-memory state for an agent in the world.
type AgentState struct {
	ID       string
	Name     string
	Location string
	Traits   Traits
	Goals    []string
	Mood     string
	Energy   int
	Credits  int
}

// Traits captures simple personality traits for agents.
type Traits struct {
	Friendliness int
	Curiosity    int
}

// Location represents a place within the world.
type Location struct {
	ID          string
	Name        string
	Description string
}

// World represents the environment in which agents exist.
type World struct {
	ID        string
	Timestep  int64
	Agents    map[string]*AgentState
	Locations map[string]Location
	Events    []Event
}

func NewWorld() *World {
	// Seed with a few default locations to keep the initial world valid.
	defaultLocations := []Location{
		{ID: "loc_default", Name: "Central Plaza", Description: "A neutral starting point for all agents."},
		{ID: "loc_market", Name: "Marketplace", Description: "Bustling area for trading and chatting."},
		{ID: "loc_park", Name: "Park", Description: "A quiet green space for strolling."},
	}

	locs := make(map[string]Location, len(defaultLocations))
	for _, loc := range defaultLocations {
		locs[loc.ID] = loc
	}

	return &World{
		ID:        "world-1",
		Timestep:  0,
		Agents:    make(map[string]*AgentState),
		Locations: locs,
		Events:    []Event{},
	}
}

func (w *World) Advance() {
	w.Timestep++
	slog.Info("world advanced", "timestep", w.Timestep)
}

// AddEvent appends a structured event to the world's history.
func (w *World) AddEvent(evt Event) {
	w.Events = append(w.Events, evt)
}

// GetAgent returns the agent state if present.
func (w *World) GetAgent(agentID string) (*AgentState, bool) {
	agent, ok := w.Agents[agentID]
	return agent, ok
}

// MoveAgent updates an agent's location if both agent and destination exist.
func (w *World) MoveAgent(agentID, locationID string) error {
	agent, ok := w.Agents[agentID]
	if !ok {
		return fmt.Errorf("agent %s not found", agentID)
	}

	if _, ok := w.Locations[locationID]; !ok {
		return fmt.Errorf("location %s not found", locationID)
	}

	agent.Location = locationID
	return nil
}
