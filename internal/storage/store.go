package storage

import (
	"context"

	"github.com/jdebrux/agentic-sim/internal/world"
)

// Store defines persistence for simulation runs.
type Store interface {
	SaveRun(ctx context.Context, runID string, w *world.World, events []world.Event) error
}
