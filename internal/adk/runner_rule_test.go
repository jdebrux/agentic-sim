package adk

import (
	"context"
	"testing"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

func TestRuleRunner_RestWhenLowEnergy(t *testing.T) {
	r := NewRuleRunner()
	view := world.WorldView{
		Self: world.AgentState{ID: "a1", Name: "A", Location: "loc_default", Energy: 10},
	}
	resp, err := r.DecideAction(context.Background(), ReasoningRequest{AgentID: "a1", AgentName: "A", View: view})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionRest {
		t.Fatalf("expected rest, got %+v", resp.Action)
	}
}

func TestRuleRunner_MoveToGoalLocation(t *testing.T) {
	r := NewRuleRunner()
	view := world.WorldView{
		Self:        world.AgentState{ID: "a1", Name: "A", Location: "loc_default", Energy: 100},
		Locations:   []world.Location{{ID: "loc_default"}, {ID: "loc_market"}},
		OtherAgents: []world.AgentState{},
	}
	resp, err := r.DecideAction(context.Background(), ReasoningRequest{AgentID: "a1", AgentName: "A", Goal: "loc_market", View: view})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionMove || resp.Action.Location != "loc_market" {
		t.Fatalf("expected move to goal location, got %+v", resp.Action)
	}
}

func TestRuleRunner_TradeInMarket(t *testing.T) {
	r := NewRuleRunner()
	view := world.WorldView{
		Self: world.AgentState{ID: "a1", Name: "A", Location: "loc_market", Energy: 100, Credits: 2},
		OtherAgents: []world.AgentState{
			{ID: "a2", Name: "B", Location: "loc_market"},
		},
		Locations: []world.Location{{ID: "loc_market"}},
		AtMarket:  true,
	}
	resp, err := r.DecideAction(context.Background(), ReasoningRequest{AgentID: "a1", AgentName: "A", View: view})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionTrade || resp.Action.TargetID != "a2" {
		t.Fatalf("expected trade to a2, got %+v", resp.Action)
	}
}

func TestRuleRunner_GreetWhenCoLocated(t *testing.T) {
	r := NewRuleRunner()
	view := world.WorldView{
		Self: world.AgentState{ID: "a1", Name: "A", Location: "loc_default", Energy: 100, Credits: 0},
		OtherAgents: []world.AgentState{
			{ID: "a2", Name: "B", Location: "loc_default"},
		},
		Locations: []world.Location{{ID: "loc_default"}},
	}
	resp, err := r.DecideAction(context.Background(), ReasoningRequest{AgentID: "a1", AgentName: "A", View: view})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionGreet || resp.Action.TargetID != "a2" {
		t.Fatalf("expected greet a2, got %+v", resp.Action)
	}
}

func TestRuleRunner_MoveToMarket(t *testing.T) {
	r := NewRuleRunner()
	view := world.WorldView{
		Self:        world.AgentState{ID: "a1", Name: "A", Location: "loc_default", Energy: 100},
		OtherAgents: []world.AgentState{},
		Locations: []world.Location{
			{ID: "loc_default"},
			{ID: "loc_market"},
		},
	}
	resp, err := r.DecideAction(context.Background(), ReasoningRequest{AgentID: "a1", AgentName: "A", View: view})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionMove || resp.Action.Location != "loc_market" {
		t.Fatalf("expected move to market, got %+v", resp.Action)
	}
}

func TestRuleRunner_SpeakFallback(t *testing.T) {
	r := NewRuleRunner()
	view := world.WorldView{
		Self:        world.AgentState{ID: "a1", Name: "A", Location: "loc_default", Energy: 100, Credits: 0},
		OtherAgents: []world.AgentState{},
		Locations:   []world.Location{{ID: "loc_default"}},
	}
	resp, err := r.DecideAction(context.Background(), ReasoningRequest{AgentID: "a1", AgentName: "A", Goal: "stay", View: view})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.Action.Type != model.ActionSpeak || resp.Action.Message == "" {
		t.Fatalf("expected speak fallback, got %+v", resp.Action)
	}
}
