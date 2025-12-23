package simulation

import (
	"context"
	"sync"
	"time"

	"github.com/jdebrux/agentic-sim/internal/storage"
)

// Manager tracks simulation runs by ID.
type Manager interface {
	Start(ctx context.Context, cfg EngineConfig, duration time.Duration) (string, error)
	Status(id string) (RunStatus, bool)
	Metrics() ManagerMetrics
}

// RunStatus describes the state of a simulation run.
type RunStatus struct {
	ID     string
	State  string // running|completed|error
	Ticks  int64
	Events int64
	Error  string
	Mode   string
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
	metrics   ManagerMetrics
	store     storage.Store
}

// ManagerOption configures the manager.
type ManagerOption func(m *InMemoryManager)

// WithStore sets a persistence store to save runs.
func WithStore(store storage.Store) ManagerOption {
	return func(m *InMemoryManager) {
		m.store = store
	}
}

func NewInMemoryManager(factory func(cfg EngineConfig) *Engine, opts ...ManagerOption) *InMemoryManager {
	m := &InMemoryManager{
		newEngine: factory,
		runs:      make(map[string]*runRecord),
		store:     nil,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *InMemoryManager) Start(ctx context.Context, cfg EngineConfig, duration time.Duration) (string, error) {
	id := generateRunID()
	rec := &runRecord{
		status: RunStatus{
			ID:    id,
			State: "running",
			Mode:  cfg.RunnerMode,
		},
	}

	m.mu.Lock()
	m.runs[id] = rec
	m.metrics.TotalRuns++
	m.metrics.Running++
	m.metrics.LastMode = cfg.RunnerMode
	m.mu.Unlock()

	go func() {
		engine := m.newEngine(cfg)
		engine.Run(context.Background(), duration)

		m.mu.Lock()
		m.metrics.Running--
		m.metrics.Completed++
		m.metrics.TotalTicks += engine.Metrics.Ticks
		m.metrics.TotalEvents += engine.Metrics.Events
		m.mu.Unlock()

		rec.mu.Lock()
		rec.status.Ticks = engine.World.Timestep
		rec.status.Events = m.safeEventsCount(engine)
		rec.status.State = "completed"
		rec.mu.Unlock()

		if m.store != nil {
			_ = m.store.SaveRun(context.Background(), id, engine.World, engine.World.Events)
		}
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

// ManagerMetrics is a snapshot of aggregate run metrics.
type ManagerMetrics struct {
	TotalRuns   int64  `json:"total_runs"`
	Running     int64  `json:"running_runs"`
	Completed   int64  `json:"completed_runs"`
	Errored     int64  `json:"errored_runs"`
	TotalTicks  int64  `json:"ticks_total"`
	TotalEvents int64  `json:"events_total"`
	LastMode    string `json:"runner_mode_last,omitempty"`
}

func (m *InMemoryManager) Metrics() ManagerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
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
