package simulation

import (
	"context"
	"testing"
	"time"

	"github.com/jdebrux/agentic-sim/internal/agents"
	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// mockAgent returns a predetermined action for testing the engine loop.
type mockAgent struct {
	id     string
	name   string
	action model.AgentAction
}

func (m mockAgent) GetID() string   { return m.id }
func (m mockAgent) GetName() string { return m.name }
func (m mockAgent) Tick(ctx context.Context, view world.WorldView) model.AgentAction {
	_ = ctx
	_ = view
	return m.action
}

// TestEngineHandleAction covers action application and event recording.
func TestEngineHandleAction(t *testing.T) {
	t.Run("applies move action", func(t *testing.T) {
		w := world.NewWorld()
		w.Locations["loc_target"] = world.Location{ID: "loc_target"}
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_default"}

		engine := &Engine{World: w}
		action := model.AgentAction{
			ActorID:  "agent-1",
			Type:     model.ActionMove,
			Location: "loc_target",
		}

		engine.handleAction(context.Background(), action)

		if got := w.Agents["agent-1"].Location; got != "loc_target" {
			t.Fatalf("expected agent location loc_target, got %s", got)
		}
		if len(w.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(w.Events))
		}
		event := w.Events[0]
		if event.Type != string(action.Type) {
			t.Fatalf("expected event type %s, got %s", action.Type, event.Type)
		}
		if errVal, ok := event.Payload["error"]; ok && errVal != nil {
			t.Fatalf("expected no error in event payload, got %v", errVal)
		}
	})

	t.Run("records events even on failure", func(t *testing.T) {
		w := world.NewWorld()
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_default"}
		engine := &Engine{World: w}
		action := model.AgentAction{
			ActorID:  "agent-1",
			Type:     model.ActionMove,
			Location: "loc_missing",
		}

		engine.handleAction(context.Background(), action)

		if got := w.Agents["agent-1"].Location; got != "loc_default" {
			t.Fatalf("expected agent location loc_default, got %s", got)
		}
		if len(w.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(w.Events))
		}
		event := w.Events[0]
		if event.Type != string(action.Type) {
			t.Fatalf("expected event type %s, got %s", action.Type, event.Type)
		}
		errVal, ok := event.Payload["error"]
		if !ok || errVal == nil || errVal == "" {
			t.Fatalf("expected error in event payload, got %v", errVal)
		}
	})
}

// TestEngineRunCoversLoop verifies ticks advance and actions are processed.
func TestEngineRunCoversLoop(t *testing.T) {
	w := world.NewWorld()

	agentA := mockAgent{
		id:   "agent-1",
		name: "A",
		action: model.AgentAction{
			ActorID: "agent-1",
			Type:    model.ActionIdle,
		},
	}
	agentB := mockAgent{
		id:   "agent-2",
		name: "B",
		action: model.AgentAction{
			ActorID: "agent-2",
			Type:    model.ActionSpeak,
			Message: "hello",
		},
	}

	w.Agents[agentA.id] = &world.AgentState{ID: agentA.id, Name: agentA.name, Location: "loc_default"}
	w.Agents[agentB.id] = &world.AgentState{ID: agentB.id, Name: agentB.name, Location: "loc_default"}

	engine := &Engine{
		World:  w,
		Agents: []agents.Agent{agentA, agentB},
		Tick:   5 * time.Millisecond,
	}

	engine.Run(context.Background(), 25*time.Millisecond)

	if w.Timestep == 0 {
		t.Fatalf("expected timestep to advance, got %d", w.Timestep)
	}
	if len(w.Events) == 0 {
		t.Fatalf("expected at least one event to be recorded")
	}
}