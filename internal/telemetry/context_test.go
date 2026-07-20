package telemetry

import (
	"context"
	"testing"
)

func TestRunIDContextRoundTrip(t *testing.T) {
	ctx := WithRunID(context.Background(), "run-123")
	if got := RunIDFromContext(ctx); got != "run-123" {
		t.Fatalf("expected run-123, got %q", got)
	}
}

func TestRunIDContextDefaultsEmpty(t *testing.T) {
	if got := RunIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty run id for a context without one, got %q", got)
	}
}
