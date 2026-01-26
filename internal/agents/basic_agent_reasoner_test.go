package agents

import (
	"context"
	"testing"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/reasoning"
	"github.com/jdebrux/agentic-sim/internal/world"
)

func TestBasicAgentTick_UsesReasonerWhenNoRunner(t *testing.T) {
	expected := model.AgentAction{ActorID: "agent-1", Type: model.ActionSpeak, Message: "from reasoner"}
	mock := &reasoning.MockReasoner{
		Response: reasoning.Response{Action: expected},
	}
	agent := NewBasicAgentWithReasoner("agent-1", "Alice", mock)
	view := world.WorldView{Self: world.AgentState{ID: "agent-1"}}

	action := agent.Tick(context.Background(), view)

	if action.ActorID != expected.ActorID || action.Type != expected.Type || action.Message != expected.Message {
		t.Fatalf("expected %#v, got %#v", expected, action)
	}
	if mock.Calls != 1 {
		t.Fatalf("expected reasoner to be called once, got %d", mock.Calls)
	}
}
