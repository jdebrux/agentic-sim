package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jdebrux/agentic-sim/internal/api"
	"github.com/jdebrux/agentic-sim/internal/config"
	"github.com/jdebrux/agentic-sim/internal/simulation"
	"github.com/jdebrux/agentic-sim/internal/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// shutdownTimeout bounds how long a SIGINT/SIGTERM waits for in-flight HTTP
// requests and active simulation runs to finish before the process exits.
const shutdownTimeout = 30 * time.Second

func main() {
	// ctx is cancelled on SIGINT/SIGTERM and is the parent for every HTTP
	// request's context (via Server.BaseContext below), so the whole app
	// shares one root context whose cancellation is driven by the OS signal.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	slog.Info("starting agentic simulation environment")

	shutdownTelemetry, err := telemetry.Setup(ctx, "agentic-sim")
	if err != nil {
		slog.Error("failed to setup telemetry", "error", err)
		os.Exit(1)
	}
	defer func() {
		// Use a fresh context: ctx is already cancelled by the time this
		// runs, and an already-done context would make span exporters bail
		// out without flushing.
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(flushCtx); err != nil {
			slog.Error("telemetry shutdown error", "error", err)
		}
	}()

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
	srv := &http.Server{
		Addr:    addr,
		Handler: otelhttp.NewHandler(mux, "agentic-sim"),
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		if err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining active runs", "timeout", shutdownTimeout)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("http server shutdown error", "error", err)
		}

		if err := manager.Shutdown(shutdownCtx); err != nil {
			slog.Warn("active simulation runs did not finish before shutdown timeout", "error", err)
		} else {
			slog.Info("all simulation runs drained")
		}
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
