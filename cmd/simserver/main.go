package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jdebrux/agentic-sim/internal/api"
	"github.com/jdebrux/agentic-sim/internal/simulation"
	"github.com/jdebrux/agentic-sim/pkg/utils"
)

func main() {
	utils.InitLogger()
	log.Println("Starting Agentic Simulation Environment (HTTP server)...")

	mux := http.NewServeMux()
	manager := simulation.NewInMemoryManager(func(cfg simulation.EngineConfig) *simulation.Engine {
		if cfg.RunnerMode == "" {
			cfg.RunnerMode = defaultRunnerMode()
		}
		// Backward compatibility: honor UseSimpleRunner if set and mode not set.
		if cfg.UseSimpleRunner && cfg.RunnerMode == "" {
			cfg.RunnerMode = "simple"
		}
		return simulation.NewEngineWithConfig(cfg)
	})
	handler := api.NewHandler(manager)
	handler.Register(mux)

	addr := getAddr()
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func defaultRunnerMode() string {
	mode := strings.ToLower(os.Getenv("RUNNER_MODE"))
	switch mode {
	case "scripted", "simple", "rule":
		return mode
	default:
		return "simple"
	}
}

func getAddr() string {
	if v := os.Getenv("PORT"); v != "" {
		return ":" + v
	}
	return ":8080"
}
