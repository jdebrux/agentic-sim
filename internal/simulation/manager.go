package simulation

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/jdebrux/agentic-sim/internal/world"
)

// Manager tracks simulation runs by ID.
type Manager interface {
	Start(ctx context.Context, cfg EngineConfig, duration time.Duration) (string, error)
	Status(id string) (RunStatus, bool)
	Metrics() ManagerMetrics
	// Events returns the full event history recorded for a run so far.
	Events(id string) ([]world.Event, bool)
	// Subscribe registers a live listener for a run's events, returning the
	// event history recorded so far and a channel for events from this point
	// on — atomically, so no event is missed or duplicated between the two.
	// The channel is closed once the run completes; call the returned
	// unsubscribe func to stop listening early (e.g. on client disconnect).
	Subscribe(id string) ([]world.Event, <-chan world.Event, func(), bool)
	// List returns a status summary for every known run, sorted by ID (IDs
	// are timestamps, so this is chronological).
	List() []RunStatus
	// WorldSnapshot returns the latest world snapshot recorded for a run.
	// ok is false if the run is unknown or no snapshot has been emitted yet
	// (a brief window right after Start).
	WorldSnapshot(id string) (world.WorldSnapshot, bool)
	// Delete forgets a run, freeing its history and subscribers. If the run
	// is still active, its engine is signaled to stop as soon as possible;
	// the underlying goroutine finishes asynchronously. Reports false if the
	// run id is unknown.
	Delete(id string) bool
	// Shutdown blocks until every active run has finished, or ctx is done —
	// whichever comes first. Used to drain in-flight runs during a graceful
	// shutdown without forcibly cancelling them.
	Shutdown(ctx context.Context) error
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
	id       string
	cancel   context.CancelFunc
	mu       sync.RWMutex
	status   RunStatus
	events   []world.Event
	latest   *world.WorldSnapshot
	subs     map[chan world.Event]struct{}
	finished bool
}

// addEvent records an event to the run's history and fans it out to any
// live subscribers. Slow subscribers are dropped rather than blocking the
// simulation tick loop.
func (r *runRecord) addEvent(evt world.Event) {
	r.mu.Lock()
	r.events = append(r.events, evt)
	if evt.Type == world.EventTypeTick {
		if snap, ok := evt.Payload["snapshot"].(world.WorldSnapshot); ok {
			r.latest = &snap
		}
	}
	subs := make([]chan world.Event, 0, len(r.subs))
	for ch := range r.subs {
		subs = append(subs, ch)
	}
	r.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- evt:
		default:
			slog.Warn("dropping event for slow SSE subscriber", "run", r.id)
		}
	}
}

// statusSnapshot returns the run's status, substituting live counts (history
// length, latest snapshot timestep) while the run is still active.
func (r *runRecord) statusSnapshot() RunStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	st := r.status
	if !r.finished {
		st.Events = int64(len(r.events))
		if r.latest != nil {
			st.Ticks = r.latest.Timestep
		}
	}
	return st
}

// eventsSnapshot returns a copy of the run's event history so far.
func (r *runRecord) eventsSnapshot() []world.Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]world.Event, len(r.events))
	copy(out, r.events)
	return out
}

// subscribe registers a new live listener and atomically returns the event
// history recorded up to that point, so callers can replay it before
// switching to the live channel without missing or duplicating events. If
// the run has already finished, the returned channel is immediately closed.
func (r *runRecord) subscribe() ([]world.Event, chan world.Event, func()) {
	ch := make(chan world.Event, 32)

	r.mu.Lock()
	backlog := make([]world.Event, len(r.events))
	copy(backlog, r.events)
	if r.finished {
		r.mu.Unlock()
		close(ch)
		return backlog, ch, func() {}
	}
	if r.subs == nil {
		r.subs = make(map[chan world.Event]struct{})
	}
	r.subs[ch] = struct{}{}
	r.mu.Unlock()

	unsubscribe := func() {
		r.mu.Lock()
		delete(r.subs, ch)
		r.mu.Unlock()
	}
	return backlog, ch, unsubscribe
}

// closeSubs closes every live subscriber channel and marks the run finished
// so later Subscribe calls get an already-closed channel instead of hanging.
func (r *runRecord) closeSubs() {
	r.mu.Lock()
	subs := r.subs
	r.subs = nil
	r.finished = true
	r.mu.Unlock()

	for ch := range subs {
		close(ch)
	}
}

// InMemoryManager is a simple manager that runs simulations in-process.
type InMemoryManager struct {
	newEngine func(cfg EngineConfig) *Engine
	runs      map[string]*runRecord
	mu        sync.RWMutex
	metrics   ManagerMetrics
	wg        sync.WaitGroup
}

func NewInMemoryManager(factory func(cfg EngineConfig) *Engine) *InMemoryManager {
	return &InMemoryManager{
		newEngine: factory,
		runs:      make(map[string]*runRecord),
	}
}

func (m *InMemoryManager) Start(ctx context.Context, cfg EngineConfig, duration time.Duration) (string, error) {
	id := generateRunID()

	// Detach from the request's cancellation so the run survives the HTTP
	// handler returning, while keeping trace context for span correlation.
	// A cancel func is kept so Delete can still stop the run early, and
	// Shutdown can drain it, independent of the originating request.
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	rec := &runRecord{
		id:     id,
		cancel: cancel,
		status: RunStatus{
			ID:    id,
			State: "running",
		},
	}

	m.mu.Lock()
	m.runs[id] = rec
	m.metrics.TotalRuns++
	m.metrics.Running++
	m.mu.Unlock()

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		defer cancel()

		engine := m.newEngine(cfg)
		engine.RunID = id
		engine.OnEvent = rec.addEvent
		engine.Clients = m.connectAgents(runCtx, cfg)
		engine.Run(runCtx, duration)

		m.mu.Lock()
		m.metrics.Running--
		m.metrics.Completed++
		m.metrics.TotalTicks += engine.Metrics.Ticks
		m.metrics.TotalEvents += engine.Metrics.Events
		m.mu.Unlock()

		rec.mu.Lock()
		rec.status.Ticks = engine.World.Timestep
		rec.status.Events = int64(len(rec.events))
		rec.status.State = "completed"
		rec.mu.Unlock()

		rec.closeSubs()
	}()

	return id, nil
}

// Delete removes a run's record, freeing its history and subscribers. If the
// run is still active, cancelling its context signals the engine's tick loop
// to stop at the next select; the goroutine's own cleanup still runs
// asynchronously against the now-detached record.
func (m *InMemoryManager) Delete(id string) bool {
	m.mu.Lock()
	rec, ok := m.runs[id]
	if ok {
		delete(m.runs, id)
	}
	m.mu.Unlock()
	if !ok {
		return false
	}

	rec.cancel()
	return true
}

// Shutdown blocks until every run started so far has finished, or ctx is
// done, whichever happens first.
func (m *InMemoryManager) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *InMemoryManager) connectAgents(ctx context.Context, cfg EngineConfig) []AgentClient {
	clients := make([]AgentClient, 0, len(cfg.Agents))
	for _, a := range cfg.Agents {
		if a.AgentCardURL == "" {
			slog.Warn("agent has no card URL, skipping connection", "agent", a.ID)
			continue
		}
		client, err := NewA2AAgentClient(ctx, a.ID, a.Name, a.AgentCardURL)
		if err != nil {
			slog.Error("failed to connect to external agent", "agent", a.ID, "error", err)
			continue
		}
		clients = append(clients, client)
	}
	return clients
}

func (m *InMemoryManager) Status(id string) (RunStatus, bool) {
	m.mu.RLock()
	rec, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return RunStatus{}, false
	}
	return rec.statusSnapshot(), true
}

// List returns a status summary for every known run, sorted by ID.
func (m *InMemoryManager) List() []RunStatus {
	m.mu.RLock()
	recs := make([]*runRecord, 0, len(m.runs))
	for _, rec := range m.runs {
		recs = append(recs, rec)
	}
	m.mu.RUnlock()

	out := make([]RunStatus, 0, len(recs))
	for _, rec := range recs {
		out = append(out, rec.statusSnapshot())
	}
	slices.SortFunc(out, func(a, b RunStatus) int { return cmp.Compare(a.ID, b.ID) })
	return out
}

// WorldSnapshot returns the latest world snapshot recorded for a run. ok is
// false if the run is unknown or no snapshot has been emitted yet.
func (m *InMemoryManager) WorldSnapshot(id string) (world.WorldSnapshot, bool) {
	m.mu.RLock()
	rec, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return world.WorldSnapshot{}, false
	}

	rec.mu.RLock()
	defer rec.mu.RUnlock()
	if rec.latest == nil {
		return world.WorldSnapshot{}, false
	}
	return *rec.latest, true
}

func (m *InMemoryManager) Events(id string) ([]world.Event, bool) {
	m.mu.RLock()
	rec, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return rec.eventsSnapshot(), true
}

func (m *InMemoryManager) Subscribe(id string) ([]world.Event, <-chan world.Event, func(), bool) {
	m.mu.RLock()
	rec, ok := m.runs[id]
	m.mu.RUnlock()
	if !ok {
		return nil, nil, nil, false
	}
	backlog, ch, unsubscribe := rec.subscribe()
	return backlog, ch, unsubscribe, true
}

// ManagerMetrics is a snapshot of aggregate run metrics.
type ManagerMetrics struct {
	TotalRuns   int64 `json:"total_runs"`
	Running     int64 `json:"running_runs"`
	Completed   int64 `json:"completed_runs"`
	Errored     int64 `json:"errored_runs"`
	TotalTicks  int64 `json:"ticks_total"`
	TotalEvents int64 `json:"events_total"`
}

func (m *InMemoryManager) Metrics() ManagerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

func generateRunID() string {
	return time.Now().Format("20060102T150405.000000000")
}
