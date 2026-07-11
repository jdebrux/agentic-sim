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

// TestManagerDeleteCancelsRunningEngine verifies Delete both forgets the run
// immediately and cancels its context, so the engine's goroutine stops well
// before its full requested duration would otherwise elapse.
func TestManagerDeleteCancelsRunningEngine(t *testing.T) {
	factory := func(cfg EngineConfig) *Engine {
		return &Engine{World: world.NewWorld(), Tick: cfg.Tick}
	}
	m := NewInMemoryManager(factory)

	id, err := m.Start(context.Background(), EngineConfig{Tick: 2 * time.Millisecond}, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error starting run: %v", err)
	}
	if _, ok := m.Status(id); !ok {
		t.Fatalf("expected run %s to be tracked immediately after Start", id)
	}

	if ok := m.Delete(id); !ok {
		t.Fatalf("expected delete to find run %s", id)
	}
	if _, ok := m.Status(id); ok {
		t.Fatalf("expected run %s to be forgotten after delete", id)
	}

	// The run's 5s duration would make Shutdown block for that long if
	// cancellation didn't take effect; a short timeout here proves it did.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := m.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected deleted run's engine to stop promptly, got: %v", err)
	}
}

// TestManagerDeleteUnknownRun verifies Delete reports false for an id that
// was never started.
func TestManagerDeleteUnknownRun(t *testing.T) {
	m := NewInMemoryManager(func(cfg EngineConfig) *Engine { return &Engine{World: world.NewWorld()} })

	if ok := m.Delete("missing"); ok {
		t.Fatalf("expected delete to report not found")
	}
}

// TestManagerShutdownDrainsCompletedRuns verifies Shutdown returns nil once a
// naturally-completing run finishes, without needing Delete or a timeout.
func TestManagerShutdownDrainsCompletedRuns(t *testing.T) {
	factory := func(cfg EngineConfig) *Engine {
		return &Engine{World: world.NewWorld(), Tick: cfg.Tick}
	}
	m := NewInMemoryManager(factory)

	if _, err := m.Start(context.Background(), EngineConfig{Tick: 2 * time.Millisecond}, 10*time.Millisecond); err != nil {
		t.Fatalf("unexpected error starting run: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected short-lived run to drain naturally, got: %v", err)
	}
}

// TestManagerShutdownTimesOutOnLongRun verifies Shutdown reports the
// context's error if a run outlives the caller's patience.
func TestManagerShutdownTimesOutOnLongRun(t *testing.T) {
	factory := func(cfg EngineConfig) *Engine {
		return &Engine{World: world.NewWorld(), Tick: cfg.Tick}
	}
	m := NewInMemoryManager(factory)

	if _, err := m.Start(context.Background(), EngineConfig{Tick: 2 * time.Millisecond}, 5*time.Second); err != nil {
		t.Fatalf("unexpected error starting run: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := m.Shutdown(shutdownCtx); err == nil {
		t.Fatalf("expected Shutdown to time out waiting for a 5s run")
	}
}
