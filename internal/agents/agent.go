package agents

import (
	"context"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// Agent defines the behavior of an autonomous entity in the world.
type Agent interface {
	GetID() string
	GetName() string
	Tick(ctx context.Context, view world.WorldView) model.AgentAction
}
