package world

// WorldView is a read-only snapshot of the world for a specific agent.
// It is shaped to be easily serialized for ADK context building.
type WorldView struct {
	WorldID      string
	Tick         int64
	Self         AgentState
	OtherAgents  []AgentState
	Locations    []Location
	RecentEvents []Event
}

// NewWorldView builds a read-only view for the given agent.
func NewWorldView(w *World, agentID string, recentEventLimit int) WorldView {
	self := AgentState{}
	others := make([]AgentState, 0, len(w.Agents))

	for _, a := range w.Agents {
		if a.ID == agentID {
			self = *a
			continue
		}
		others = append(others, *a)
	}

	locations := make([]Location, 0, len(w.Locations))
	for _, loc := range w.Locations {
		locations = append(locations, loc)
	}

	events := w.Events
	if recentEventLimit > 0 && len(events) > recentEventLimit {
		events = events[len(events)-recentEventLimit:]
	}

	eventsCopy := make([]Event, len(events))
	copy(eventsCopy, events)

	return WorldView{
		WorldID:      w.ID,
		Tick:         w.Timestep,
		Self:         self,
		OtherAgents:  others,
		Locations:    locations,
		RecentEvents: eventsCopy,
	}
}
