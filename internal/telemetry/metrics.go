package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter            = otel.Meter("agentic-sim")
	ticksTotal       = must(meter.Int64Counter("simulation.ticks.total"))
	actionsTotal     = must(meter.Int64Counter("simulation.actions.total"))
	decisionLatency  = must(meter.Float64Histogram("agent.decision.latency"))
	decisionErrors   = must(meter.Int64Counter("agent.decision.errors"))
	roundtripLatency = must(meter.Float64Histogram("a2a.roundtrip.latency"))
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// withRunID appends a run_id attribute to attrs if ctx carries one (see
// WithRunID), so metrics can be filtered/grouped per simulation run.
func withRunID(ctx context.Context, attrs ...attribute.KeyValue) []attribute.KeyValue {
	if id := RunIDFromContext(ctx); id != "" {
		attrs = append(attrs, attribute.String("run_id", id))
	}
	return attrs
}

func RecordTick(ctx context.Context) {
	ticksTotal.Add(ctx, 1, metric.WithAttributes(withRunID(ctx)...))
}

func RecordAction(ctx context.Context, actionType, actorID string) {
	actionsTotal.Add(ctx, 1,
		metric.WithAttributes(withRunID(ctx,
			attribute.String("action_type", actionType),
			attribute.String("actor_id", actorID),
		)...),
	)
}

func RecordDecisionLatency(ctx context.Context, agentID string, duration time.Duration) {
	decisionLatency.Record(ctx, duration.Seconds(),
		metric.WithAttributes(withRunID(ctx,
			attribute.String("agent_id", agentID),
		)...),
	)
}

func RecordDecisionError(ctx context.Context, agentID, errorType string) {
	decisionErrors.Add(ctx, 1,
		metric.WithAttributes(withRunID(ctx,
			attribute.String("agent_id", agentID),
			attribute.String("error_type", errorType),
		)...),
	)
}

func RecordRoundtripLatency(ctx context.Context, agentID string, duration time.Duration) {
	roundtripLatency.Record(ctx, duration.Seconds(),
		metric.WithAttributes(withRunID(ctx,
			attribute.String("agent_id", agentID),
		)...),
	)
}
