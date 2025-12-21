package simulation

import (
	"context"
	"sync"
	"time"
)

// Manager tracks simulation runs by ID.
type Manager interface {
	Start(ctx context.Context, cfg EngineConfig, duration time.Duration) (string, error)
	Status(id string) (RunStatus, bool)
}

// RunStatus describes the state of a simulation run.
type RunStatus struct {
	ID     string
	State  string // running|completed|error
	Ticks  int64
	Events int64
	Error  string
}

type runRecord struct {
	status RunStatus
	mu     sync.RWMutex
}

// InMemoryManager is a simple manager that runs simulations in-process.
type InMemoryManager struct {
	newEngine func(cfg EngineConfig) *Engine
	runs      map[string]*runRecord
	mu        sync.RWMutex
}

func NewInMemoryManager(factory func(cfg EngineConfig) *Engine) *InMemoryManager {
	return &InMemoryManager{
		newEngine: factory,
		runs:      make(map[string]*runRecord),
	}
}

func (m *InMemoryManager) Start(ctx context.Context, cfg EngineConfig, duration time.Duration) (string, error) {
	id := generateRunID()
	rec := &runRecord{
		status: RunStatus{
			ID:    id,
			State: "running",
		},
	}

	m.mu.Lock()
	m.runs[id] = rec
	m.mu.Unlock()

	go func() {
		engine := m.newEngine(cfg)
		engine.Run(context.Background(), duration)

		rec.mu.Lock()
		rec.status.Ticks = engine.World.Timestep
		rec.status.Events = m.safeEventsCount(engine)
		rec.status.State = "completed"
		rec.mu.Unlock()
	}()

	return id, nil
}

func (m *InMemoryManager) Status(id string) (RunStatus, bool) {
	m.mu.RLock()
	rec, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return RunStatus{}, false
	}
	rec.mu.RLock()
	defer rec.mu.RUnlock()
	return rec.status, true
}

func (m *InMemoryManager) safeEventsCount(e *Engine) int64 {
	if e == nil || e.World == nil {
		return 0
	}
	return int64(len(e.World.Events))
}

func generateRunID() string {
	return time.Now().Format("20060102T150405.000000000")
}
