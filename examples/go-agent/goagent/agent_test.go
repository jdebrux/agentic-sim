package goagent

import (
	"testing"

	"github.com/jdebrux/agentic-sim/internal/world"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name       string
		view       world.WorldView
		wantType   world.ActionType
		wantTarget string
	}{
		{
			name: "not at market moves to market",
			view: world.WorldView{
				Self: world.AgentState{ID: "a", Location: "loc_default"},
			},
			wantType: world.ActionMove,
		},
		{
			name: "at market alone rests",
			view: world.WorldView{
				Self: world.AgentState{ID: "a", Location: "loc_market"},
			},
			wantType: world.ActionRest,
		},
		{
			name: "at market with another agent trades",
			view: world.WorldView{
				Self: world.AgentState{ID: "a", Location: "loc_market"},
				OtherAgents: []world.AgentState{
					{ID: "b", Location: "loc_park"},
					{ID: "c", Location: "loc_market"},
				},
			},
			wantType:   world.ActionTrade,
			wantTarget: "c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := Decide(tt.view)
			if action.Type != tt.wantType {
				t.Fatalf("expected action %s, got %s", tt.wantType, action.Type)
			}
			if action.Type == world.ActionMove && action.Location != marketLocation {
				t.Fatalf("expected move to %s, got %s", marketLocation, action.Location)
			}
			if tt.wantTarget != "" && action.TargetID != tt.wantTarget {
				t.Fatalf("expected target %s, got %s", tt.wantTarget, action.TargetID)
			}
		})
	}
}
