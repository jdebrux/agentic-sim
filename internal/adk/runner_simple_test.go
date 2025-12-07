package adk

import (
	"context"
	"testing"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

func TestSimpleRunnerDecideAction_GreetWhenCoLocated(t *testing.T) {
	runner := &SimpleRunner{}
	view := world.WorldView{
		Self: world.AgentState{ID: "a1", Name: "A", Location: "loc_default"},
		OtherAgents: []world.AgentState{
			{ID: "a2", Name: "B", Location: "loc_default"},
		},
		Locations: []world.Location{{ID: "loc_default"}},
	}
	resp, err := runner.DecideAction(context.Background(), ReasoningRequest{
		AgentID:   "a1",
		AgentName: "A",
		View:      view,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionGreet || resp.Action.TargetID != "a2" {
		t.Fatalf("expected greet action toward a2, got %+v", resp.Action)
	}
}

func TestSimpleRunnerDecideAction_MoveToMarket(t *testing.T) {
	runner := &SimpleRunner{}
	view := world.WorldView{
		Self:        world.AgentState{ID: "a1", Name: "A", Location: "loc_default"},
		OtherAgents: []world.AgentState{},
		Locations: []world.Location{
			{ID: "loc_default"},
			{ID: "loc_market"},
		},
	}
	resp, err := runner.DecideAction(context.Background(), ReasoningRequest{
		AgentID:   "a1",
		AgentName: "A",
		View:      view,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionMove || resp.Action.Location != "loc_market" {
		t.Fatalf("expected move to market, got %+v", resp.Action)
	}
}

func TestSimpleRunnerDecideAction_SpeakFallback(t *testing.T) {
	runner := &SimpleRunner{}
	view := world.WorldView{
		Self:        world.AgentState{ID: "a1", Name: "A", Location: "loc_default"},
		OtherAgents: []world.AgentState{},
		Locations:   []world.Location{{ID: "loc_default"}},
	}
	resp, err := runner.DecideAction(context.Background(), ReasoningRequest{
		AgentID:   "a1",
		AgentName: "A",
		View:      view,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionSpeak || resp.Action.Message == "" {
		t.Fatalf("expected speak fallback, got %+v", resp.Action)
	}
}
