package api

import (
	"fmt"
	"net/http"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// simAgentCard returns the A2A AgentCard for the simserver.
func simAgentCard(baseURL string) *world.AgentCard {
	return &world.AgentCard{
		Name:        "agentic-sim",
		Description: "Multi-agent simulation environment. External A2A agents can join a shared world and interact.",
		URL:         baseURL,
		Version:     "1.0.0",
		Capabilities: world.AgentCapabilities{
			Streaming: true,
		},
		Skills: []world.AgentSkill{
			{
				ID:          "simulate",
				Name:        "Run Simulation",
				Description: "Start an agentic simulation with specified agents and duration",
				InputModes:  []string{"application/json"},
				OutputModes: []string{"application/json"},
				Tags:        []string{"simulation", "multi-agent"},
			},
			{
				ID:          "join",
				Name:        "Join Simulation",
				Description: "Add an external A2A agent to a running simulation",
				InputModes:  []string{"application/json"},
				OutputModes: []string{"application/json"},
				Tags:        []string{"simulation", "join"},
			},
		},
		DefaultInputModes:  []string{"application/json"},
		DefaultOutputModes: []string{"application/json"},
	}
}

// agentCardHandler serves the A2A well-known agent card endpoint.
func (h *Handler) agentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	card := simAgentCard(fmt.Sprintf("http://%s", r.Host))

	// Convert internal world.AgentCard to a2a.AgentCard for serialization.
	a2aCard := &a2a.AgentCard{
		Name:               card.Name,
		Description:        card.Description,
		Version:            card.Version,
		DefaultInputModes:  card.DefaultInputModes,
		DefaultOutputModes: card.DefaultOutputModes,
		Capabilities: a2a.AgentCapabilities{
			Streaming: card.Capabilities.Streaming,
		},
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(card.URL, a2a.TransportProtocolJSONRPC),
		},
		Skills: make([]a2a.AgentSkill, len(card.Skills)),
	}
	if card.Provider != nil {
		a2aCard.Provider = &a2a.AgentProvider{Org: card.Provider.Organization, URL: card.Provider.URL}
	}
	for i, s := range card.Skills {
		a2aCard.Skills[i] = a2a.AgentSkill{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Tags:        s.Tags,
			Examples:    s.Examples,
			InputModes:  s.InputModes,
			OutputModes: s.OutputModes,
		}
	}

	writeJSON(w, http.StatusOK, a2aCard)
}
