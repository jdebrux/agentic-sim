package world

import (
	"cmp"
	"slices"
)

// EventTypeTick is the Type of the per-tick world snapshot event. Unlike
// action events, tick events are never appended to World.Events (see
// Engine.OnEvent), so they never appear in a WorldView's RecentEvents.
const EventTypeTick = "tick"

// AgentSnapshot is a flat, deep-copied view of one agent for UI/observer
// consumption. It deliberately omits Goals and Traits, which aren't needed
// by observers and would otherwise require copying a slice.
type AgentSnapshot struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location string `json:"location"`
	Mood     string `json:"mood,omitempty"`
	Energy   int    `json:"energy"`
	Credits  int    `json:"credits"`
}

// WorldSnapshot is a point-in-time, deep-copied view of the whole world.
type WorldSnapshot struct {
	WorldID   string          `json:"world_id"`
	Timestep  int64           `json:"timestep"`
	Agents    []AgentSnapshot `json:"agents"`
	Locations []Location      `json:"locations"`
}

// Snapshot deep-copies the world's current state into a WorldSnapshot.
// Agents and Locations are sorted by ID for deterministic output. The
// result shares no memory with w, so it remains valid after w is further
// mutated.
func Snapshot(w *World) WorldSnapshot {
	agents := make([]AgentSnapshot, 0, len(w.Agents))
	for _, a := range w.Agents {
		agents = append(agents, AgentSnapshot{
			ID:       a.ID,
			Name:     a.Name,
			Location: a.Location,
			Mood:     a.Mood,
			Energy:   a.Energy,
			Credits:  a.Credits,
		})
	}
	slices.SortFunc(agents, func(a, b AgentSnapshot) int { return cmp.Compare(a.ID, b.ID) })

	locations := make([]Location, 0, len(w.Locations))
	for _, loc := range w.Locations {
		locations = append(locations, loc)
	}
	slices.SortFunc(locations, func(a, b Location) int { return cmp.Compare(a.ID, b.ID) })

	return WorldSnapshot{
		WorldID:   w.ID,
		Timestep:  w.Timestep,
		Agents:    agents,
		Locations: locations,
	}
}
