package simulation

import (
	"context"
	"testing"
	"time"

	"github.com/jdebrux/agentic-sim/internal/world"
)

// TestRunRecordPubSub covers the low-level event history + fan-out
// primitives the Manager uses to back the SSE stream and events endpoints.
func TestRunRecordPubSub(t *testing.T) {
	rec := &runRecord{id: "run-1"}

	backlog, live, unsubscribe := rec.subscribe()
	defer unsubscribe()
	if len(backlog) != 0 {
		t.Fatalf("expected empty backlog before any events, got %d", len(backlog))
	}

	evt := world.Event{ID: "evt-1", Type: "speak", ActorID: "agent-1"}
	rec.addEvent(evt)

	select {
	case got := <-live:
		if got.ID != evt.ID {
			t.Fatalf("expected event %s, got %s", evt.ID, got.ID)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for live event")
	}

	if history := rec.eventsSnapshot(); len(history) != 1 || history[0].ID != evt.ID {
		t.Fatalf("expected history to contain the recorded event, got %+v", history)
	}

	rec.closeSubs()
	if _, ok := <-live; ok {
		t.Fatalf("expected live channel to be closed after closeSubs")
	}

	// Subscribing after the run has finished should hand back an
	// already-closed channel rather than hanging.
	lateBacklog, lateCh, lateUnsub := rec.subscribe()
	defer lateUnsub()
	if len(lateBacklog) != 1 {
		t.Fatalf("expected late subscriber to see prior history, got %d", len(lateBacklog))
	}
	if _, ok := <-lateCh; ok {
		t.Fatalf("expected late subscriber channel to already be closed")
	}
}

// TestManagerSubscribeAndEvents verifies the Manager wires a run's events
// through to Events() and Subscribe(), and closes the live channel once the
// run completes.
func TestManagerSubscribeAndEvents(t *testing.T) {
	factory := func(cfg EngineConfig) *Engine {
		return &Engine{World: world.NewWorld(), Tick: cfg.Tick}
	}
	m := NewInMemoryManager(factory)

	id, err := m.Start(context.Background(), EngineConfig{Tick: 2 * time.Millisecond}, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error starting run: %v", err)
	}

	backlog, live, unsubscribe, ok := m.Subscribe(id)
	if !ok {
		t.Fatalf("expected subscribe to find run %s", id)
	}
	defer unsubscribe()
	if len(backlog) != 0 {
		t.Fatalf("expected empty initial backlog, got %d", len(backlog))
	}

	select {
	case _, ok := <-live:
		if ok {
			t.Fatalf("expected no events from an agent-less run")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for run to complete and close the channel")
	}

	if _, ok := m.Events(id); !ok {
		t.Fatalf("expected event history to be present for completed run %s", id)
	}
}

// TestManagerSubscribeUnknownRun verifies Subscribe/Events report ok=false
// for an id that was never started.
func TestManagerSubscribeUnknownRun(t *testing.T) {
	m := NewInMemoryManager(func(cfg EngineConfig) *Engine { return &Engine{World: world.NewWorld()} })

	if _, _, _, ok := m.Subscribe("missing"); ok {
		t.Fatalf("expected Subscribe to report not found")
	}
	if _, ok := m.Events("missing"); ok {
		t.Fatalf("expected Events to report not found")
	}
}
