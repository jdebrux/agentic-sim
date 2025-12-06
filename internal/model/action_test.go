package model

import "testing"

// TestAgentActionTypes validates the action type constants and basic struct usage.
func TestAgentActionTypes(t *testing.T) {
	t.Run("supports known action types", func(t *testing.T) {
		want := map[ActionType]string{
			ActionIdle: "idle",
			ActionMove: "move",
			ActionSpeak: "speak",
			ActionInteract: "interact",
		}

		for k, v := range want {
			if string(k) != v {
				t.Fatalf("expected %q to equal %q", k, v)
			}
		}
	})

	t.Run("no unexpected values", func(t *testing.T) {
		seen := map[ActionType]bool{}
		for _, a := range []ActionType{ActionIdle, ActionMove, ActionSpeak, ActionInteract} {
			if seen[a] {
				t.Fatalf("duplicate action: %q", a)
			}
			seen[a] = true
		}
		if len(seen) != 4 {
			t.Fatalf("expected 4 actions, got %d", len(seen))
		}
	})
}
