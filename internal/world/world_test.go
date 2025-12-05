package world

import (
	"testing"
)

// TestMoveAgent covers success and failure cases for moving agents.
func TestMoveAgent(t *testing.T) {
	t.Run("moves agent to valid location", func(t *testing.T) {
		w := NewWorld()
		addLocation(t, w, "loc_target")
		addAgent(t, w, "agent_1", defaultLocationID(w))

		err := w.MoveAgent("agent_1", "loc_target")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if w.Agents["agent_1"].Location != "loc_target" {
			t.Fatalf("expected agent location to be 'loc_target', got %v", w.Agents["agent_1"].Location)
		}
	})

	t.Run("fails when agent missing", func(t *testing.T) {
		w := NewWorld()
		addLocation(t, w, "loc_target")

		err := w.MoveAgent("agent_missing", "loc_target")
		if err == nil {
			t.Fatal("expected error for missing agent, got nil")
		}
	})

	t.Run("fails when location missing", func(t *testing.T) {
		w := NewWorld()
		addAgent(t, w, "agent_1", defaultLocationID(w))

		initialLocation := w.Agents["agent_1"].Location

		err := w.MoveAgent("agent_1", "loc_target")
		if err == nil {
			t.Fatal("expected error for missing location, got nil")
		}
		if w.Agents["agent_1"].Location != initialLocation {
			t.Fatalf("expected agent location to remain %v on failure, got %v", initialLocation, w.Agents["agent_1"].Location)
		}
	})
}

// TestAddEvent ensures events are appended with correct ordering and capacity.
func TestAddEvent(t *testing.T) {
	w := NewWorld()

	evt1 := Event{ID: "evt1", Tick: 1, ActorID: "a1"}
	evt2 := Event{ID: "evt2", Tick: 2, ActorID: "a2"}

	w.AddEvent(evt1)
	w.AddEvent(evt2)

	if got, want := len(w.Events), 2; got != want {
		t.Fatalf("expected %d events, got %d", want, got)
	}
	if w.Events[0].ID != evt1.ID || w.Events[0].Tick != evt1.Tick || w.Events[0].ActorID != evt1.ActorID {
		t.Fatalf("first event mismatch: %+v", w.Events[0])
	}
	if w.Events[1].ID != evt2.ID || w.Events[1].Tick != evt2.Tick || w.Events[1].ActorID != evt2.ActorID {
		t.Fatalf("second event mismatch: %+v", w.Events[1])
	}
}
