package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/jdebrux/agentic-sim/internal/api"
	"github.com/jdebrux/agentic-sim/internal/config"
	"github.com/jdebrux/agentic-sim/internal/simulation"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("starting agentic simulation environment")

	appCfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	manager := simulation.NewInMemoryManager(func(cfg simulation.EngineConfig) *simulation.Engine {
		if cfg.Tick == 0 {
			cfg.Tick = appCfg.DefaultTick
		}
		return simulation.NewEngineWithConfig(cfg)
	})
	handler := api.NewHandler(manager, appCfg.DefaultTick)
	handler.Register(mux)

	addr := getAddr(appCfg.HTTPPort)
	slog.Info("listening", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
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
