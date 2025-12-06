package agents

import (
	"context"
	"testing"

	"github.com/jdebrux/agentic-sim/internal/adk"
	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// TestBasicAgentTick covers runner delegation and fallback behavior.
func TestBasicAgentTick(t *testing.T) {
	t.Run("delegates to runner when present", func(t *testing.T) {
		expected := model.AgentAction{ActorID: "agent-1", Type: model.ActionSpeak, Message: "hello from runner"}
		mock := &adk.MockRunner{
			Response: adk.ReasoningResponse{Action: expected},
		}
		agent := NewBasicAgentWithRunner("agent-1", "Alice", mock)
		view := world.WorldView{Self: world.AgentState{ID: "agent-1"}}

		action := agent.Tick(context.Background(), view)

		// Compare fields individually since AgentAction contains a map and cannot be compared directly.
		if action.ActorID != expected.ActorID || action.Type != expected.Type || action.Message != expected.Message {
			t.Fatalf("expected %#v, got %#v", expected, action)
		}
		if mock.Calls != 1 {
			t.Fatalf("expected runner to be called once, got %d", mock.Calls)
		}
	})

	t.Run("falls back to placeholder when runner errors", func(t *testing.T) {
		mock := &adk.MockRunner{
			Err:      assertErrorSentinel{}, // sentinel to force error path
			Response: adk.ReasoningResponse{Action: model.AgentAction{Message: "runner message"}},
		}
		agent := NewBasicAgentWithRunner("agent-1", "Alice", mock)
		view := world.NewWorldView(&world.World{
			ID:       "world-1",
			Timestep: 2,
			Agents: map[string]*world.AgentState{
				"agent-1": {ID: "agent-1", Name: "Alice", Location: "loc_default"},
			},
			Locations: map[string]world.Location{
				"loc_default": {ID: "loc_default"},
			},
		}, "agent-1", 0)

		action := agent.Tick(context.Background(), view)

		if action.ActorID != "agent-1" {
			t.Fatalf("expected ActorID agent-1, got %s", action.ActorID)
		}
		if action.Type != model.ActionSpeak {
			t.Fatalf("expected fallback action type speak, got %s", action.Type)
		}
		if action.Message == "runner message" {
			t.Fatalf("expected fallback message, got runner message")
		}
		if mock.Calls != 1 {
			t.Fatalf("expected runner to be called once, got %d", mock.Calls)
		}
	})

	t.Run("falls back to placeholder when no runner provided", func(t *testing.T) {
		agent := NewBasicAgent("agent-1", "Alice")
		view := world.NewWorldView(&world.World{
			ID:       "world-1",
			Timestep: 5,
			Agents: map[string]*world.AgentState{
				"agent-1": {ID: "agent-1", Name: "Alice", Location: "loc_default"},
			},
			Locations: map[string]world.Location{
				"loc_default": {ID: "loc_default"},
			},
		}, "agent-1", 0)

		action := agent.Tick(context.Background(), view)

		if action.ActorID != "agent-1" {
			t.Fatalf("expected ActorID agent-1, got %s", action.ActorID)
		}
		if action.Type != model.ActionSpeak {
			t.Fatalf("expected fallback action type speak, got %s", action.Type)
		}
		if action.Message == "" {
			t.Fatalf("expected fallback message to be populated")
		}
	})
}

// helper creates a minimal world view for tests.
func testWorldView(t *testing.T) world.WorldView {
	t.Helper()

	w := world.NewWorld()
	w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_default"}
	return world.NewWorldView(w, "agent-1", 3)
}

// helper creates a mock response for runner-based tests.
func testRunnerAction() model.AgentAction {
	return model.AgentAction{
		ActorID: "agent-1",
		Type:    model.ActionSpeak,
		Message: "hello",
	}
}

// ensures helpers are used to avoid linter complaints about unused symbols when tests are skipped.
var (
	_ = testWorldView
	_ = testRunnerAction
	_ = context.Background
	_ = adk.MockRunner{}
)

// assertErrorSentinel is a lightweight error to drive runner failures in tests.
type assertErrorSentinel struct{}

func (assertErrorSentinel) Error() string { return "runner error" }
