package simulation

import (
	"testing"

	"github.com/jdebrux/agentic-sim/internal/world"
)

func TestEngineDefaultAgents(t *testing.T) {
	engine := NewEngineWithConfig(EngineConfig{})

	if len(engine.World.Agents) != 2 {
		t.Fatalf("expected 2 default agents, got %d", len(engine.World.Agents))
	}
	if _, ok := engine.World.Agents["agent-alice"]; !ok {
		t.Fatalf("expected agent-alice in world")
	}
	if _, ok := engine.World.Agents["agent-bob"]; !ok {
		t.Fatalf("expected agent-bob in world")
	}
}

func TestEngineDefaultLocations(t *testing.T) {
	engine := NewEngineWithConfig(EngineConfig{})

	want := []string{"loc_default", "loc_market", "loc_park"}
	for _, id := range want {
		if _, ok := engine.World.Locations[id]; !ok {
			t.Fatalf("expected location %s", id)
		}
	}
}

func TestEngineCustomAgents(t *testing.T) {
	engine := NewEngineWithConfig(EngineConfig{
		Agents: []AgentRegistration{
			{
				ID:       "agent-carol",
				Name:     "Carol",
				Location: "loc_market",
				Traits:   world.Traits{Friendliness: 7, Curiosity: 3},
				Goals:    []string{"trade"},
				Energy:   80,
				Credits:  20,
			},
		},
	})

	if len(engine.World.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(engine.World.Agents))
	}
	carol, ok := engine.World.Agents["agent-carol"]
	if !ok {
		t.Fatalf("expected agent-carol")
	}
	if carol.Energy != 80 || carol.Credits != 20 {
		t.Fatalf("expected energy=80 credits=20, got energy=%d credits=%d", carol.Energy, carol.Credits)
	}
}

func TestEngineCustomLocations(t *testing.T) {
	engine := NewEngineWithConfig(EngineConfig{
		Locations: []world.Location{
			{ID: "loc_arena", Name: "Arena"},
		},
	})

	if len(engine.World.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(engine.World.Locations))
	}
	if _, ok := engine.World.Locations["loc_arena"]; !ok {
		t.Fatalf("expected loc_arena")
	}
}
