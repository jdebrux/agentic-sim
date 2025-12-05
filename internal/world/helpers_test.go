package world

import "testing"

func addAgent(t *testing.T, w *World, id, loc string) {
	t.Helper()
	w.Agents[id] = &AgentState{ID: id, Name: id, Location: loc}
}

func addLocation(t *testing.T, w *World, id string) {
	t.Helper()
	w.Locations[id] = Location{ID: id}
}

func addEvent(t *testing.T, w *World, evt Event) {
	t.Helper()
	w.AddEvent(evt)
}

func defaultLocationID(w *World) string {
	if loc, ok := w.Locations["loc_default"]; ok {
		return loc.ID
	}
	return ""
}

func hasLocation(locs []Location, id string) bool {
	for _, loc := range locs {
		if loc.ID == id {
			return true
		}
	}
	return false
}
