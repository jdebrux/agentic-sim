package agents

import "github.com/jdebrux/agentic-sim/internal/world"

type BasicAgent struct {
	Name string
}

func NewBasicAgent(name string) *BasicAgent {
	return &BasicAgent{Name: name}
}

func (a *BasicAgent) GetName() string {
	return a.Name
}

func (a *BasicAgent) Act(w *world.World) string {
	// This is placeholder logic for now
	return "looks around curiously."
}