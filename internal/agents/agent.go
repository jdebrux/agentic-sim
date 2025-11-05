package agents

import "github.com/jdebrux/agentic-sim/internal/world"

// Agent defines the behavior of an autonomous entity in the world.
type Agent interface {
	GetName() string
	Act(w *world.World) string
}
