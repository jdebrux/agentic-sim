package adk

import (
	"context"
	"strings"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// RuleRunner implements a simple rule-based runner for deterministic decisions.
type RuleRunner struct {
	RestThreshold int
}

func NewRuleRunner() *RuleRunner {
	return &RuleRunner{
		RestThreshold: 20,
	}
}

func (r *RuleRunner) DecideAction(ctx context.Context, req ReasoningRequest) (ReasoningResponse, error) {
	_ = ctx

	self := req.View.Self
	goal := strings.TrimSpace(req.Goal)

	// 1) Rest if energy is low.
	if self.Energy < r.RestThreshold {
		return resp(model.AgentAction{
			ActorID:  req.AgentID,
			Type:     model.ActionRest,
			Reason:   "low energy",
			ToolName: "rest",
		}, "rest_low_energy"), nil
	}

	// 2) Move to goal location if provided and known.
	if goalLoc := r.matchLocation(goal, req.View.Locations); goalLoc != "" && self.Location != goalLoc {
		return resp(model.AgentAction{
			ActorID:  req.AgentID,
			Type:     model.ActionMove,
			Location: goalLoc,
			Reason:   "goal location",
			ToolName: "move",
			ToolArgs: map[string]string{"destination": goalLoc},
		}, "move_to_goal"), nil
	}

	// 3) Trade in market if co-located with another agent and have credits.
	if req.View.AtMarket && self.Credits > 0 {
		for _, other := range req.View.OtherAgents {
			if other.Location == self.Location {
				return resp(model.AgentAction{
					ActorID:  req.AgentID,
					TargetID: other.ID,
					Type:     model.ActionTrade,
					Reason:   "co-located at market",
					ToolName: "trade",
					ToolArgs: map[string]string{"target": other.ID},
				}, "trade_at_market"), nil
			}
		}
	}

	// 4) Greet co-located agents.
	for _, other := range req.View.OtherAgents {
		if other.Location == self.Location {
			msg := "Hello " + other.Name + "!"
			return resp(model.AgentAction{
				ActorID:  req.AgentID,
				TargetID: other.ID,
				Type:     model.ActionGreet,
				Message:  msg,
				Reason:   "co-located agent",
				ToolName: "greet",
			}, "greet_co_located"), nil
		}
	}

	// 5) Move to market if it exists and we're not there.
	if self.Location != "loc_market" {
		for _, loc := range req.View.Locations {
			if loc.ID == "loc_market" {
				return resp(model.AgentAction{
					ActorID:  req.AgentID,
					Type:     model.ActionMove,
					Location: "loc_market",
					Reason:   "explore market",
					ToolName: "move",
					ToolArgs: map[string]string{"destination": "loc_market"},
				}, "move_to_market"), nil
			}
		}
	}

	// 6) Fallback: speak status/goal echo.
	msg := "Staying at " + self.Location
	if goal != "" {
		msg = "Goal: " + goal + " | " + msg
	}
	return resp(model.AgentAction{
		ActorID:  req.AgentID,
		Type:     model.ActionSpeak,
		Message:  msg,
		Reason:   "fallback status",
		ToolName: "speak",
		ToolArgs: map[string]string{"message": msg},
	}, "speak_status"), nil
}

func (r *RuleRunner) matchLocation(goal string, locs []world.Location) string {
	if goal == "" {
		return ""
	}
	lowerGoal := strings.ToLower(goal)
	for _, loc := range locs {
		if strings.ToLower(loc.ID) == lowerGoal || strings.ToLower(loc.Name) == lowerGoal {
			return loc.ID
		}
	}
	return ""
}

func resp(action model.AgentAction, policy string) ReasoningResponse {
	return ReasoningResponse{
		Action: action,
		Notes:  map[string]string{"policy": policy},
	}
}
