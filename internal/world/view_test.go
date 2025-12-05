package world

import "testing"

// TestNewWorldView validates snapshot contents for self, others, locations, and recent events.
func TestNewWorldView(t *testing.T) {
	t.Run("includes self and other agents", func(t *testing.T) {
		w := NewWorld()
		w.Timestep = 3
		addAgent(t, w, "agent-1", "loc_default")
		addAgent(t, w, "agent-2", "loc_default")
		addEvent(t, w, Event{ID: "evt-1", Tick: 1, ActorID: "agent-1"})
		addEvent(t, w, Event{ID: "evt-2", Tick: 2, ActorID: "agent-2"})

		view := NewWorldView(w, "agent-1", 10)

		if view.WorldID != w.ID {
			t.Fatalf("expected WorldID %s, got %s", w.ID, view.WorldID)
		}
		if view.Tick != w.Timestep {
			t.Fatalf("expected Tick %d, got %d", w.Timestep, view.Tick)
		}
		if view.Self.ID != "agent-1" {
			t.Fatalf("expected self ID agent-1, got %s", view.Self.ID)
		}
		if len(view.OtherAgents) != 1 || view.OtherAgents[0].ID != "agent-2" {
			t.Fatalf("expected one other agent agent-2, got %+v", view.OtherAgents)
		}
		if !hasLocation(view.Locations, "loc_default") {
			t.Fatalf("expected locations to include loc_default, got %+v", view.Locations)
		}
		if len(view.RecentEvents) != 2 || view.RecentEvents[1].ID != "evt-2" {
			t.Fatalf("expected two events ending with evt-2, got %+v", view.RecentEvents)
		}
	})

	t.Run("truncates recent events when limit is set", func(t *testing.T) {
		w := NewWorld()
		addEvent(t, w, Event{ID: "evt-1", Tick: 1})
		addEvent(t, w, Event{ID: "evt-2", Tick: 2})
		addEvent(t, w, Event{ID: "evt-3", Tick: 3})

		view := NewWorldView(w, "agent-missing", 2)

		if len(view.RecentEvents) != 2 {
			t.Fatalf("expected 2 events, got %d", len(view.RecentEvents))
		}
		if view.RecentEvents[0].ID != "evt-2" || view.RecentEvents[1].ID != "evt-3" {
			t.Fatalf("expected last two events [evt-2 evt-3], got [%s %s]", view.RecentEvents[0].ID, view.RecentEvents[1].ID)
		}
	})

	t.Run("handles missing agent gracefully", func(t *testing.T) {
		w := NewWorld()
		addAgent(t, w, "agent-present", "loc_default")

		view := NewWorldView(w, "agent-missing", 5)

		if view.Self.ID != "" {
			t.Fatalf("expected zero-value Self when agent missing, got %s", view.Self.ID)
		}
		if len(view.OtherAgents) != 1 || view.OtherAgents[0].ID != "agent-present" {
			t.Fatalf("expected other agents to include agent-present, got %+v", view.OtherAgents)
		}
	})
}
