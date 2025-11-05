package simulation

import (
	"context"
	"log"
	"time"

	"github.com/jdebrux/agentic-sim/internal/agents"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// Engine manages the simulation lifecycle.
type Engine struct {
	World  *world.World
	Agents []agents.Agent
	Tick   time.Duration
}

// NewEngine initializes a new simulation engine.
func NewEngine() *Engine {
	return &Engine{
		World:  world.NewWorld(),
		Agents: []agents.Agent{
			agents.NewBasicAgent("Alice"),
			agents.NewBasicAgent("Bob"),
		},
		Tick: 1 * time.Second,
	}
}

// Run starts the simulation loop for a given duration.
func (e *Engine) Run(ctx context.Context, duration time.Duration) {
	ticker := time.NewTicker(e.Tick)
	defer ticker.Stop()

	end := time.After(duration)
	step := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("Simulation stopped by context.")
			return
		case <-end:
			log.Println("Simulation duration complete.")
			return
		case <-ticker.C:
			step++
			log.Printf("=== Simulation Step %d ===", step)
			for _, a := range e.Agents {
				action := a.Act(e.World)
				log.Printf("%s: %s", a.GetName(), action)
			}
			e.World.Advance()
		}
	}
}
