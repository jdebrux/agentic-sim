package agents

import (
	"context"
	"fmt"

	"github.com/jdebrux/agentic-sim/internal/adk"
	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

type BasicAgent struct {
	ID     string
	Name   string
	Runner adk.Runner
}

func NewBasicAgent(id, name string) *BasicAgent {
	return &BasicAgent{
		ID:   id,
		Name: name,
	}
}

func NewBasicAgentWithRunner(id, name string, runner adk.Runner) *BasicAgent {
	return &BasicAgent{
		ID:     id,
		Name:   name,
		Runner: runner,
	}
}

func (a *BasicAgent) GetID() string {
	return a.ID
}

func (a *BasicAgent) GetName() string {
	return a.Name
}

func (a *BasicAgent) Tick(ctx context.Context, view world.WorldView) model.AgentAction {
	if a.Runner != nil {
		resp, err := a.Runner.DecideAction(ctx, adk.ReasoningRequest{
			AgentID:   a.ID,
			AgentName: a.Name,
			View:      view,
		})
		if err == nil {
			return resp.Action
		}
	}

	// Placeholder behavior: announce current location and tick.
	msg := fmt.Sprintf("is at %s on tick %d", view.Self.Location, view.Tick)

	return model.AgentAction{
		ActorID:  a.ID,
		Type:     model.ActionSpeak,
		Message:  msg,
		ToolName: "speak",
		ToolArgs: map[string]string{
			"message": msg,
		},
		Metadata: map[string]string{
			"agent_name": a.Name,
		},
	}
}
