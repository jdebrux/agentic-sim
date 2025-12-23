package memory

import (
	"context"
	"sync"

	"github.com/jdebrux/agentic-sim/internal/world"
)

// Store is an in-memory implementation of storage.Store for tests/local use.
type Store struct {
	mu   sync.RWMutex
	runs map[string]storageRecord
}

type storageRecord struct {
	World  *world.World
	Events []world.Event
}

func New() *Store {
	return &Store{
		runs: make(map[string]storageRecord),
	}
}

func (s *Store) SaveRun(ctx context.Context, runID string, w *world.World, events []world.Event) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[runID] = storageRecord{World: w, Events: append([]world.Event(nil), events...)}
	return nil
}

// Get returns a stored run, if present.
func (s *Store) Get(runID string) (storageRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.runs[runID]
	return rec, ok
}
