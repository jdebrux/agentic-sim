package api

import (
	"context"
	"net/http"
	"time"

	"github.com/jdebrux/agentic-sim/internal/simulation"
)

// DefaultSimulationService runs simulations using the provided engine factory.
type DefaultSimulationService struct {
	NewEngine func(cfg simulation.EngineConfig) *simulation.Engine
}

func NewDefaultSimulationService(factory func(cfg simulation.EngineConfig) *simulation.Engine) *DefaultSimulationService {
	return &DefaultSimulationService{NewEngine: factory}
}

func (s *DefaultSimulationService) RunSimulation(_ *http.Request, cfg simulation.EngineConfig, duration time.Duration) (int64, int, error) {
	engine := s.NewEngine(cfg)
	engine.Run(context.Background(), duration)
	return engine.World.Timestep, len(engine.World.Events), nil
}
