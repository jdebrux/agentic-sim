// Command go-agent runs the deterministic rule-based A2A agent standalone,
// so it can be pointed at a running simserver without any LLM or API key.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jdebrux/agentic-sim/examples/go-agent/goagent"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9003"
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%s", port)
	card := goagent.NewAgentCard(baseURL)

	mux := http.NewServeMux()
	goagent.Register(mux, card)

	addr := ":" + port
	log.Printf("go-agent listening on %s, card at %s/.well-known/agent-card.json", addr, baseURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("go-agent server error: %v", err)
	}
}
