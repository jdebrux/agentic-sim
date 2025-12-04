package simulation

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jdebrux/agentic-sim/internal/agents"
	"github.com/jdebrux/agentic-sim/internal/model"
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
	w := world.NewWorld()

	agentList := []agents.Agent{
		agents.NewBasicAgent("agent-alice", "Alice"),
		agents.NewBasicAgent("agent-bob", "Bob"),
	}

	for _, a := range agentList {
		w.Agents[a.GetID()] = &world.AgentState{
			ID:       a.GetID(),
			Name:     a.GetName(),
			Location: "loc_default",
		}
	}

	return &Engine{
		World:  w,
		Agents: agentList,
		Tick:   1 * time.Second,
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
			actions := make([]model.AgentAction, 0, len(e.Agents))
			for _, a := range e.Agents {
				view := world.NewWorldView(e.World, a.GetID(), 10)
				action := a.Tick(ctx, view)
				actions = append(actions, action)
			}

			for _, action := range actions {
				e.handleAction(ctx, action)
			}
			e.World.Advance()
		}
	}
}

// handleAction applies an agent action to the world and records an event.
func (e *Engine) handleAction(ctx context.Context, action model.AgentAction) {
	_ = ctx

	event := world.Event{
		ID:        fmt.Sprintf("evt-%d-%d", e.World.Timestep, time.Now().UnixNano()),
		WorldID:   e.World.ID,
		Tick:      e.World.Timestep,
		Timestamp: time.Now(),
		Type:      string(action.Type),
		ActorID:   action.ActorID,
		TargetID:  action.TargetID,
		Payload: map[string]any{
			"tool":      action.ToolName,
			"tool_args": action.ToolArgs,
			"message":   action.Message,
			"location":  action.Location,
		},
	}

	var err error

	switch action.Type {
	case model.ActionMove:
		err = e.World.MoveAgent(action.ActorID, action.Location)
	case model.ActionInteract:
		// For now, just validate both actors exist; interactions can be expanded later.
		if _, ok := e.World.GetAgent(action.ActorID); !ok {
			err = fmt.Errorf("actor %s not found for interaction", action.ActorID)
			break
		}
		if action.TargetID == "" {
			err = fmt.Errorf("interaction requires target")
			break
		}
		if _, ok := e.World.GetAgent(action.TargetID); !ok {
			err = fmt.Errorf("target %s not found for interaction", action.TargetID)
		}
	case model.ActionSpeak, model.ActionIdle:
		// No world mutation required.
	default:
		err = fmt.Errorf("unsupported action type: %s", action.Type)
	}

	if err != nil {
		log.Printf("action failed for %s: %v", action.ActorID, err)
		event.Payload["error"] = err.Error()
	} else {
		log.Printf("action applied for %s: %s", action.ActorID, action.Type)
	}

	e.World.AddEvent(event)
}
