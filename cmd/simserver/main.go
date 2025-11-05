package main

import (
	"context"
	"log"
	"time"

	"github.com/jdebrux/agentic-sim/internal/simulation"
)

func main() {
	log.Println("Starting Agentic Simulation Environment...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := simulation.NewEngine()
	engine.Run(ctx, 5*time.Second) // run for 5 seconds as a test

	log.Println("Simulation completed.")
}
