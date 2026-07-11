package simulation

import (
	"context"
	"testing"
	"time"

	"github.com/jdebrux/agentic-sim/internal/world"
)

// mockAgent returns a predetermined action for testing the engine loop.
type mockAgent struct {
	id     string
	name   string
	action world.AgentAction
}

func (m mockAgent) GetID() string   { return m.id }
func (m mockAgent) GetName() string { return m.name }
func (m mockAgent) Decide(ctx context.Context, view world.WorldView) (world.AgentAction, error) {
	_ = ctx
	_ = view
	return m.action, nil
}

// TestEngineHandleAction covers action application and event recording.
func TestEngineHandleAction(t *testing.T) {
	t.Run("applies move action", func(t *testing.T) {
		w := world.NewWorld()
		w.Locations["loc_target"] = world.Location{ID: "loc_target"}
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_default"}

		engine := &Engine{World: w}
		action := world.AgentAction{
			ActorID:  "agent-1",
			Type:     world.ActionMove,
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
		action := world.AgentAction{
			ActorID:  "agent-1",
			Type:     world.ActionMove,
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

	t.Run("records error for invalid interaction", func(t *testing.T) {
		w := world.NewWorld()
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_default"}
		engine := &Engine{World: w}
		action := world.AgentAction{
			ActorID:  "agent-1",
			Type:     world.ActionInteract,
			TargetID: "",
		}

		engine.handleAction(context.Background(), action)

		if len(w.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(w.Events))
		}
		event := w.Events[0]
		errVal, ok := event.Payload["error"]
		if !ok || errVal == nil || errVal == "" {
			t.Fatalf("expected error in event payload for invalid interaction, got %v", errVal)
		}
	})

	t.Run("applies greet when co-located", func(t *testing.T) {
		w := world.NewWorld()
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_default"}
		w.Agents["agent-2"] = &world.AgentState{ID: "agent-2", Name: "B", Location: "loc_default"}
		engine := &Engine{World: w}
		action := world.AgentAction{
			ActorID:  "agent-1",
			TargetID: "agent-2",
			Type:     world.ActionGreet,
			Message:  "hi",
		}

		engine.handleAction(context.Background(), action)

		if len(w.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(w.Events))
		}
		if errVal, ok := w.Events[0].Payload["error"]; ok && errVal != nil {
			t.Fatalf("expected no error for valid greet, got %v", errVal)
		}
	})

	t.Run("interaction requires co-location", func(t *testing.T) {
		w := world.NewWorld()
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_default"}
		w.Agents["agent-2"] = &world.AgentState{ID: "agent-2", Name: "B", Location: "loc_market"}
		engine := &Engine{World: w}
		action := world.AgentAction{
			ActorID:  "agent-1",
			TargetID: "agent-2",
			Type:     world.ActionInteract,
		}

		engine.handleAction(context.Background(), action)

		if len(w.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(w.Events))
		}
		errVal, ok := w.Events[0].Payload["error"]
		if !ok || errVal == nil || errVal == "" {
			t.Fatalf("expected co-location error, got %v", errVal)
		}
	})

	t.Run("trade succeeds only in market with co-located target", func(t *testing.T) {
		w := world.NewWorld()
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_market", Credits: 5}
		w.Agents["agent-2"] = &world.AgentState{ID: "agent-2", Name: "B", Location: "loc_market", Credits: 2}
		engine := &Engine{World: w}
		action := world.AgentAction{
			ActorID:  "agent-1",
			TargetID: "agent-2",
			Type:     world.ActionTrade,
		}

		engine.handleAction(context.Background(), action)

		if len(w.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(w.Events))
		}
		if errVal, ok := w.Events[0].Payload["error"]; ok && errVal != nil {
			t.Fatalf("expected no error for valid trade, got %v", errVal)
		}
		if w.Agents["agent-1"].Credits != 4 || w.Agents["agent-2"].Credits != 3 {
			t.Fatalf("expected credits transfer 1: got %d and %d", w.Agents["agent-1"].Credits, w.Agents["agent-2"].Credits)
		}
	})

	t.Run("trade fails outside market", func(t *testing.T) {
		w := world.NewWorld()
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_default", Credits: 5}
		w.Agents["agent-2"] = &world.AgentState{ID: "agent-2", Name: "B", Location: "loc_default", Credits: 2}
		engine := &Engine{World: w}
		action := world.AgentAction{
			ActorID:  "agent-1",
			TargetID: "agent-2",
			Type:     world.ActionTrade,
		}

		engine.handleAction(context.Background(), action)

		if len(w.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(w.Events))
		}
		errVal, ok := w.Events[0].Payload["error"]
		if !ok || errVal == nil || errVal == "" {
			t.Fatalf("expected trade error outside market, got %v", errVal)
		}
		if w.Agents["agent-1"].Credits != 5 || w.Agents["agent-2"].Credits != 2 {
			t.Fatalf("expected no credit change on failed trade, got %d and %d", w.Agents["agent-1"].Credits, w.Agents["agent-2"].Credits)
		}
	})

	t.Run("trade fails with insufficient credits", func(t *testing.T) {
		w := world.NewWorld()
		w.Agents["agent-1"] = &world.AgentState{ID: "agent-1", Name: "A", Location: "loc_market", Credits: 0}
		w.Agents["agent-2"] = &world.AgentState{ID: "agent-2", Name: "B", Location: "loc_market", Credits: 5}
		engine := &Engine{World: w}
		action := world.AgentAction{
			ActorID:  "agent-1",
			TargetID: "agent-2",
			Type:     world.ActionTrade,
		}

		engine.handleAction(context.Background(), action)

		if len(w.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(w.Events))
		}
		errVal, ok := w.Events[0].Payload["error"]
		if !ok || errVal == nil || errVal == "" {
			t.Fatalf("expected trade error for insufficient credits, got %v", errVal)
		}
		if w.Agents["agent-1"].Credits != 0 || w.Agents["agent-2"].Credits != 5 {
			t.Fatalf("expected no credit change on failed trade, got %d and %d", w.Agents["agent-1"].Credits, w.Agents["agent-2"].Credits)
		}
	})
}

// TestEngineRunCoversLoop verifies ticks advance and actions are processed.
func TestEngineRunCoversLoop(t *testing.T) {
	w := world.NewWorld()
	w.Locations["loc_target"] = world.Location{ID: "loc_target"}

	moveAgent := mockAgent{
		id:   "agent-1",
		name: "Mover",
		action: world.AgentAction{
			ActorID:  "agent-1",
			Type:     world.ActionMove,
			Location: "loc_target",
		},
	}
	speakAgent := mockAgent{
		id:   "agent-2",
		name: "Speaker",
		action: world.AgentAction{
			ActorID: "agent-2",
			Type:    world.ActionSpeak,
			Message: "hello",
		},
	}

	w.Agents[moveAgent.id] = &world.AgentState{ID: moveAgent.id, Name: moveAgent.name, Location: "loc_default"}
	w.Agents[speakAgent.id] = &world.AgentState{ID: speakAgent.id, Name: speakAgent.name, Location: "loc_default"}

	engine := &Engine{
		World:   w,
		Clients: []AgentClient{moveAgent, speakAgent},
		Tick:    5 * time.Millisecond,
	}

	engine.Run(context.Background(), 12*time.Millisecond)

	if w.Timestep == 0 {
		t.Fatalf("expected timestep to advance, got %d", w.Timestep)
	}
	if len(w.Events) < 2 {
		t.Fatalf("expected at least one event per agent, got %d", len(w.Events))
	}
	if got := w.Agents[moveAgent.id].Location; got != "loc_target" {
		t.Fatalf("expected mover to reach loc_target, got %s", got)
	}
}
