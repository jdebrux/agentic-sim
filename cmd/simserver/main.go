package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jdebrux/agentic-sim/internal/api"
	"github.com/jdebrux/agentic-sim/internal/simulation"
)

func main() {
	log.Println("Starting Agentic Simulation Environment (HTTP server)...")

	mux := http.NewServeMux()
	simService := api.NewDefaultSimulationService(simulation.NewEngineWithConfig)
	handler := api.NewHandler(simService)
	handler.Register(mux)

	addr := getAddr()
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getAddr() string {
	if v := os.Getenv("PORT"); v != "" {
		return ":" + v
	}
	return ":8080"
}
