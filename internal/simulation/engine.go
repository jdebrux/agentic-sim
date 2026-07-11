package simulation

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jdebrux/agentic-sim/internal/world"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// AgentClient defines how the engine communicates with an external agent.
// Phase 1 uses a stub implementation; Phase 3 wires it to A2A.
type AgentClient interface {
	GetID() string
	GetName() string
	Decide(ctx context.Context, view world.WorldView) (world.AgentAction, error)
}

// AgentRegistration is how agents enter the sim.
type AgentRegistration struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Location     string       `json:"location"`
	Traits       world.Traits `json:"traits"`
	Goals        []string     `json:"goals"`
	Energy       int          `json:"energy"`
	Credits      int          `json:"credits"`
	AgentCardURL string       `json:"agent_card_url,omitempty"`
}

// Engine manages the simulation lifecycle.
type Engine struct {
	World   *world.World
	Clients []AgentClient
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
	Tick      time.Duration
	Agents    []AgentRegistration
	Locations []world.Location
}

// NewEngine initializes a new simulation engine with defaults.
func NewEngine() *Engine {
	return NewEngineWithConfig(EngineConfig{})
}

// NewEngineWithConfig creates an engine from config.
func NewEngineWithConfig(cfg EngineConfig) *Engine {
	w := world.NewWorld()

	locations := cfg.Locations
	if len(locations) == 0 {
		locations = defaultLocations()
	}
	w.Locations = make(map[string]world.Location, len(locations))
	for _, loc := range locations {
		w.Locations[loc.ID] = loc
	}

	agents := cfg.Agents
	if len(agents) == 0 {
		agents = defaultAgents()
	}
	for _, a := range agents {
		w.Agents[a.ID] = &world.AgentState{
			ID:       a.ID,
			Name:     a.Name,
			Location: a.Location,
			Traits:   a.Traits,
			Goals:    a.Goals,
			Energy:   a.Energy,
			Credits:  a.Credits,
		}
	}

	return &Engine{
		World: w,
		Tick:  tickOrDefault(cfg.Tick, 1*time.Second),
	}
}

func defaultLocations() []world.Location {
	return []world.Location{
		{ID: "loc_default", Name: "Central Plaza", Description: "A neutral starting point for all agents."},
		{ID: "loc_market", Name: "Marketplace", Description: "Bustling area for trading and chatting."},
		{ID: "loc_park", Name: "Park", Description: "A quiet green space for strolling."},
	}
}

func defaultAgents() []AgentRegistration {
	return []AgentRegistration{
		{
			ID:       "agent-alice",
			Name:     "Alice",
			Location: "loc_default",
			Traits:   world.Traits{Friendliness: 5, Curiosity: 5},
			Goals:    []string{"explore", "socialize"},
			Energy:   100,
			Credits:  10,
		},
		{
			ID:       "agent-bob",
			Name:     "Bob",
			Location: "loc_default",
			Traits:   world.Traits{Friendliness: 5, Curiosity: 5},
			Goals:    []string{"explore", "socialize"},
			Energy:   100,
			Credits:  10,
		},
	}
}

func tickOrDefault(t time.Duration, def time.Duration) time.Duration {
	if t > 0 {
		return t
	}
	return def
}

// Run starts the simulation loop for a given duration.
func (e *Engine) Run(ctx context.Context, duration time.Duration) {
	tracer := otel.Tracer("simulation.engine")

	ticker := time.NewTicker(e.Tick)
	defer ticker.Stop()

	end := time.After(duration)

	for step := 0; ; step++ {
		select {
		case <-ctx.Done():
			slog.Info("simulation stopped by context")
			return
		case <-end:
			slog.Info("simulation duration complete")
			return
		case <-ticker.C:
			step++
			slog.Info("simulation tick", "step", step, "timestep", e.World.Timestep)

			ctx, span := tracer.Start(ctx, "simulation.tick")
			span.SetAttributes(
				attribute.Int("tick.step", step),
				attribute.Int64("world.timestep", e.World.Timestep),
				attribute.Int("agent.count", len(e.Clients)),
			)

			actions := make([]world.AgentAction, 0, len(e.Clients))
			for _, client := range e.Clients {
				view := world.NewWorldView(e.World, client.GetID(), 10)
				action, err := client.Decide(ctx, view)
				if err != nil {
					slog.Warn("agent decision failed",
						"agent", client.GetID(),
						"error", err,
					)
					action = world.AgentAction{
						ActorID: client.GetID(),
						Type:    world.ActionIdle,
						Reason:  "decision error",
					}
				}
				actions = append(actions, action)
			}

			for _, action := range actions {
				e.handleAction(ctx, action)
			}
			e.World.Advance(ctx)
			e.Metrics.Ticks++

			span.End()
		}
	}
}

// handleAction applies an agent action to the world and records an event.
func (e *Engine) handleAction(ctx context.Context, action world.AgentAction) {
	tracer := otel.Tracer("simulation.engine")
	ctx, span := tracer.Start(ctx, "world.apply_action")
	defer span.End()

	span.SetAttributes(
		attribute.String("action.type", string(action.Type)),
		attribute.String("actor.id", action.ActorID),
		attribute.String("target.id", action.TargetID),
		attribute.String("action.location", action.Location),
	)

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
	case world.ActionMove:
		err = e.World.MoveAgent(action.ActorID, action.Location)
		e.applyEnergyDelta(&event, action.ActorID, -5)
	case world.ActionInteract, world.ActionGreet:
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
	case world.ActionTrade:
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
	case world.ActionSpeak, world.ActionIdle:
		e.applyEnergyDelta(&event, action.ActorID, -1)
	case world.ActionRest:
		e.applyEnergyDelta(&event, action.ActorID, +10)
	default:
		err = fmt.Errorf("unsupported action type: %s", action.Type)
	}

	if err != nil {
		slog.Warn("action failed", "actor", action.ActorID, "error", err)
		event.Payload["error"] = err.Error()
		span.RecordError(err)
		span.SetStatus(codes.Error, "action failed")
	} else {
		slog.Info("action applied",
			"tick", e.World.Timestep,
			"actor", action.ActorID,
			"type", action.Type,
			"target", action.TargetID,
			"location", action.Location,
			"reason", action.Reason,
		)
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
