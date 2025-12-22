package adk

import (
	"context"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// Runner defines the ADK client's decision surface.
type Runner interface {
	DecideAction(ctx context.Context, req ReasoningRequest) (ReasoningResponse, error)
}

// ReasoningRequest captures the context passed to the ADK runner.
type ReasoningRequest struct {
	AgentID   string
	AgentName string
	View      world.WorldView
	Goal      string
}

// ReasoningResponse wraps the decided action.
type ReasoningResponse struct {
	Action model.AgentAction
	Notes  map[string]string
}

// MockRunner returns a fixed response for tests.
type MockRunner struct {
	Response ReasoningResponse
	Err      error
	Calls    int
}

func (m *MockRunner) DecideAction(ctx context.Context, req ReasoningRequest) (ReasoningResponse, error) {
	_ = ctx
	_ = req
	m.Calls++
	return m.Response, m.Err
}

// SimpleRunner is a synchronous, rule-based runner for local/testing use.
// It chooses a deterministic action from the provided world view without LLMs.
type SimpleRunner struct{}

func (s *SimpleRunner) DecideAction(ctx context.Context, req ReasoningRequest) (ReasoningResponse, error) {
	_ = ctx

	// Rest if energy is low.
	if req.View.Self.Energy < 20 {
		return ReasoningResponse{
			Action: model.AgentAction{
				ActorID:  req.AgentID,
				Type:     model.ActionRest,
				Reason:   "low energy",
				ToolName: "rest",
			},
			Notes: map[string]string{"policy": "rest_low_energy"},
		}, nil
	}

	// Prefer trade when co-located at market.
	if req.View.AtMarket {
		for _, other := range req.View.OtherAgents {
			if other.Location == req.View.Self.Location && req.View.Self.Credits > 0 {
				return ReasoningResponse{
					Action: model.AgentAction{
						ActorID:  req.AgentID,
						TargetID: other.ID,
						Type:     model.ActionTrade,
						Reason:   "co-located at market",
						ToolName: "trade",
						ToolArgs: map[string]string{"target": other.ID},
					},
					Notes: map[string]string{"policy": "trade_at_market"},
				}, nil
			}
		}
	}

	// Prefer greeting a co-located agent if any.
	for _, other := range req.View.OtherAgents {
		if other.Location == req.View.Self.Location {
			msg := "Hello " + other.Name + "!"
			return ReasoningResponse{
				Action: model.AgentAction{
					ActorID:  req.AgentID,
					TargetID: other.ID,
					Type:     model.ActionGreet,
					Message:  msg,
					Reason:   "co-located agent",
					ToolName: "greet",
				},
				Notes: map[string]string{"policy": "greet_co_located"},
			}, nil
		}
	}

	// Otherwise, move to market if not already there and it exists.
	if req.View.Self.Location != "loc_market" {
		for _, loc := range req.View.Locations {
			if loc.ID == "loc_market" {
				return ReasoningResponse{
					Action: model.AgentAction{
						ActorID:  req.AgentID,
						Type:     model.ActionMove,
						Location: "loc_market",
						Reason:   "explore market",
						ToolName: "move",
						ToolArgs: map[string]string{"destination": "loc_market"},
					},
					Notes: map[string]string{"policy": "move_to_market"},
				}, nil
			}
		}
	}

	// Fallback: speak current status.
	msg := "Staying put at " + req.View.Self.Location
	return ReasoningResponse{
		Action: model.AgentAction{
			ActorID:  req.AgentID,
			Type:     model.ActionSpeak,
			Message:  msg,
			Reason:   "no other actions available",
			ToolName: "speak",
			ToolArgs: map[string]string{"message": msg},
		},
		Notes: map[string]string{"policy": "speak_status"},
	}, nil
}
