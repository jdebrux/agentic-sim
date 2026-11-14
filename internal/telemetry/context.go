package telemetry

import "context"

type runIDKey struct{}

// WithRunID returns a context carrying the simulation run id. Every Record*
// helper in this package picks it up as a run_id metric attribute, so
// callers that don't have direct access to the engine (e.g. the A2A client)
// still get their metrics tagged correctly as long as they're passed a
// context descending from this one.
func WithRunID(ctx context.Context, runID string) context.Context {
	return context.WithValue(ctx, runIDKey{}, runID)
}

// RunIDFromContext returns the run id previously set with WithRunID, or ""
// if none was set.
func RunIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(runIDKey{}).(string)
	return id
}
