package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/jdebrux/agentic-sim/internal/api"
	"github.com/jdebrux/agentic-sim/internal/config"
	"github.com/jdebrux/agentic-sim/internal/simulation"
	"github.com/jdebrux/agentic-sim/pkg/utils"
)

func main() {
	utils.InitLogger()
	log.Println("Starting Agentic Simulation Environment (HTTP server)...")

	appCfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	mux := http.NewServeMux()
	manager := simulation.NewInMemoryManager(func(cfg simulation.EngineConfig) *simulation.Engine {
		if cfg.RunnerMode == "" {
			cfg.RunnerMode = appCfg.DefaultRunnerMode
		}
		// Backward compatibility: honor UseSimpleRunner if set and mode not set.
		if cfg.UseSimpleRunner && cfg.RunnerMode == "" {
			cfg.RunnerMode = "simple"
		}
		if cfg.Tick == 0 {
			cfg.Tick = appCfg.DefaultTick
		}
		return simulation.NewEngineWithConfig(cfg)
	})
	handler := api.NewHandler(manager, appCfg.DefaultTick, appCfg.DefaultRunnerMode)
	handler.Register(mux)

	addr := getAddr(appCfg.HTTPPort)
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getAddr(port string) string {
	if port == "" {
		return ":8080"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}
