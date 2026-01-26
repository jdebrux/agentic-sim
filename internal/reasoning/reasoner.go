package reasoning

import (
	"context"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// Reasoner abstracts a provider-specific decision maker (LLM or otherwise).
type Reasoner interface {
	DecideAction(ctx context.Context, req Request) (Response, error)
}

// Request captures the context sent to a reasoner.
type Request struct {
	AgentID   string
	AgentName string
	View      world.WorldView
	Goal      string
}

// Response wraps the decided action with optional notes.
type Response struct {
	Action model.AgentAction
	Notes  map[string]string
}
