package simulation

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jdebrux/agentic-sim/internal/adk"
	"github.com/jdebrux/agentic-sim/internal/agents"
	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// Engine manages the simulation lifecycle.
type Engine struct {
	World   *world.World
	Agents  []agents.Agent
	Tick    time.Duration
	Metrics Metrics
}

// Metrics captures basic counters for observability.
type Metrics struct {
	Ticks  int64
	Events int64
}

// EngineConfig toggles engine behavior.
type EngineConfig struct {
	UseSimpleRunner bool
	Tick            time.Duration
}

// NewEngine initializes a new simulation engine with defaults.
func NewEngine() *Engine {
	return NewEngineWithConfig(EngineConfig{})
}

// NewEngineWithConfig allows optional toggles (e.g., simple runner).
func NewEngineWithConfig(cfg EngineConfig) *Engine {
	w := world.NewWorld()

	agentList := []agents.Agent{
		newAgentWithConfig("agent-alice", "Alice", cfg),
		newAgentWithConfig("agent-bob", "Bob", cfg),
	}

	for _, a := range agentList {
		w.Agents[a.GetID()] = &world.AgentState{
			ID:       a.GetID(),
			Name:     a.GetName(),
			Location: "loc_default",
			Traits: world.Traits{
				Friendliness: 5,
				Curiosity:    5,
			},
			Goals:   []string{"explore", "socialize"},
			Energy:  100,
			Credits: 10,
		}
	}

	return &Engine{
		World:  w,
		Agents: agentList,
		Tick:   tickOrDefault(cfg.Tick, 1*time.Second),
	}
}

func newAgentWithConfig(id, name string, cfg EngineConfig) agents.Agent {
	if cfg.UseSimpleRunner {
		return agents.NewBasicAgentWithRunner(id, name, &adk.SimpleRunner{})
	}
	return agents.NewBasicAgent(id, name)
}

func tickOrDefault(t time.Duration, def time.Duration) time.Duration {
	if t > 0 {
		return t
	}
	return def
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
				log.Printf("action.request tick=%d actor=%s type=%s target=%s location=%s reason=%s", e.World.Timestep, action.ActorID, action.Type, action.TargetID, action.Location, action.Reason)
				actions = append(actions, action)
			}

			for _, action := range actions {
				e.handleAction(ctx, action)
			}
			e.World.Advance()
			e.Metrics.Ticks++
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
			"reason":    action.Reason,
		},
	}

	var err error

	switch action.Type {
	case model.ActionMove:
		err = e.World.MoveAgent(action.ActorID, action.Location)
		e.applyEnergyDelta(&event, action.ActorID, -5)
	case model.ActionInteract, model.ActionGreet:
		actor, ok := e.World.GetAgent(action.ActorID)
		if !ok {
			err = fmt.Errorf("actor %s not found for interaction", action.ActorID)
			break
		}
		if action.TargetID == "" {
			err = fmt.Errorf("interaction requires target")
			break
		}
		target, ok := e.World.GetAgent(action.TargetID)
		if !ok {
			err = fmt.Errorf("target %s not found for interaction", action.TargetID)
			break
		}
		if actor.Location != target.Location {
			err = fmt.Errorf("interaction requires co-location at %s", actor.Location)
		}
		e.applyEnergyDelta(&event, action.ActorID, -1)
	case model.ActionTrade:
		actor, ok := e.World.GetAgent(action.ActorID)
		if !ok {
			err = fmt.Errorf("actor %s not found for trade", action.ActorID)
			break
		}
		if action.TargetID == "" {
			err = fmt.Errorf("trade requires target")
			break
		}
		target, ok := e.World.GetAgent(action.TargetID)
		if !ok {
			err = fmt.Errorf("target %s not found for trade", action.TargetID)
			break
		}
		if actor.Location != target.Location {
			err = fmt.Errorf("trade requires co-location at %s", actor.Location)
			break
		}
		if actor.Location != "loc_market" {
			err = fmt.Errorf("trade allowed only in market, current %s", actor.Location)
			break
		}
		err = e.transferCredits(actor, target, 1)
		if err == nil {
			event.Payload["credits_transferred"] = 1
			event.Payload["actor_credits"] = actor.Credits
			event.Payload["target_credits"] = target.Credits
		}
		e.applyEnergyDelta(&event, action.ActorID, -3)
	case model.ActionSpeak, model.ActionIdle:
		// No world mutation required.
		e.applyEnergyDelta(&event, action.ActorID, -1)
	case model.ActionRest:
		e.applyEnergyDelta(&event, action.ActorID, +10)
	default:
		err = fmt.Errorf("unsupported action type: %s", action.Type)
	}

	if err != nil {
		log.Printf("action failed for %s: %v", action.ActorID, err)
		event.Payload["error"] = err.Error()
	} else {
		log.Printf("action.applied tick=%d actor=%s type=%s target=%s location=%s reason=%s", e.World.Timestep, action.ActorID, action.Type, action.TargetID, action.Location, action.Reason)
	}

	e.World.AddEvent(event)
	e.Metrics.Events++
}

func (e *Engine) adjustEnergy(agentID string, delta int) {
	agent, _ := e.World.GetAgent(agentID)
	if agent == nil {
		return
	}
	agent.Energy += delta
	if agent.Energy > 100 {
		agent.Energy = 100
	}
	if agent.Energy < 0 {
		agent.Energy = 0
	}
}

func (e *Engine) applyEnergyDelta(event *world.Event, agentID string, delta int) {
	before := e.getEnergy(agentID)
	e.adjustEnergy(agentID, delta)
	after := e.getEnergy(agentID)
	event.Payload["energy_delta"] = delta
	event.Payload["energy_before"] = before
	event.Payload["energy_after"] = after
}

func (e *Engine) getEnergy(agentID string) int {
	agent, _ := e.World.GetAgent(agentID)
	if agent == nil {
		return 0
	}
	return agent.Energy
}

func (e *Engine) transferCredits(actor, target *world.AgentState, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("trade amount must be positive")
	}
	if actor.Credits < amount {
		return fmt.Errorf("insufficient credits for trade: have %d need %d", actor.Credits, amount)
	}
	actor.Credits -= amount
	target.Credits += amount
	return nil
}
