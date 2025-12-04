package adk

import (
	"context"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// Runner defines the ADK client's decision surface.
type Runner interface {
	DecideAction(ctx context.Context, req ReasoningRequest) (ReasoningResponse, error)
}

// ReasoningRequest captures the context passed to the ADK runner.
type ReasoningRequest struct {
	AgentID   string
	AgentName string
	View      world.WorldView
	Goal      string
}

// ReasoningResponse wraps the decided action.
type ReasoningResponse struct {
	Action model.AgentAction
	Notes  map[string]string
}

// MockRunner returns a fixed response for tests.
type MockRunner struct {
	Response ReasoningResponse
	Err      error
	Calls    int
}

func (m *MockRunner) DecideAction(ctx context.Context, req ReasoningRequest) (ReasoningResponse, error) {
	_ = ctx
	_ = req
	m.Calls++
	return m.Response, m.Err
}
