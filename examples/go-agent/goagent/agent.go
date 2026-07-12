// Package goagent is a minimal, deterministic A2A agent for agentic-sim: no
// LLM, no SDK, just a rule ("head to market, then trade with whoever's
// there"). It exists to exercise the real A2A wire protocol in tests and as
// a zero-dependency local dev target, alongside the LLM-backed examples in
// examples/adk-agent and examples/langgraph-agent.
package goagent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/a2aproject/a2a-go/a2asrv/eventqueue"
	"github.com/jdebrux/agentic-sim/internal/world"
)

const marketLocation = "loc_market"

// Decide implements the agent's entire strategy: get to the market, then
// trade with the first other agent found there, resting if alone.
func Decide(view world.WorldView) world.AgentAction {
	if view.Self.Location != marketLocation {
		return world.AgentAction{
			Type:     world.ActionMove,
			Location: marketLocation,
			Reason:   "heading to market",
		}
	}

	for _, other := range view.OtherAgents {
		if other.Location == marketLocation {
			return world.AgentAction{
				Type:     world.ActionTrade,
				TargetID: other.ID,
				Reason:   fmt.Sprintf("trading with %s at market", other.ID),
			}
		}
	}

	return world.AgentAction{
		Type:   world.ActionRest,
		Reason: "waiting at market",
	}
}

// executor bridges A2A requests into Decide.
type executor struct{}

var _ a2asrv.AgentExecutor = (*executor)(nil)

func (*executor) Execute(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	action, err := decideFromMessage(reqCtx.Message)
	if err != nil {
		action = world.AgentAction{Type: world.ActionIdle, Reason: err.Error()}
	}

	actionJSON, err := json.Marshal(action)
	if err != nil {
		return fmt.Errorf("marshal action: %w", err)
	}

	return q.Write(ctx, a2a.NewMessage(a2a.MessageRoleAgent, a2a.TextPart{Text: string(actionJSON)}))
}

func (*executor) Cancel(ctx context.Context, reqCtx *a2asrv.RequestContext, q eventqueue.Queue) error {
	return nil
}

func decideFromMessage(msg *a2a.Message) (world.AgentAction, error) {
	if msg == nil {
		return world.AgentAction{}, fmt.Errorf("no message in request")
	}

	var text string
	for _, part := range msg.Parts {
		if tp, ok := part.(a2a.TextPart); ok {
			text = tp.Text
			break
		}
	}
	if text == "" {
		return world.AgentAction{}, fmt.Errorf("no text part in request message")
	}

	var view world.WorldView
	if err := json.Unmarshal([]byte(text), &view); err != nil {
		return world.AgentAction{}, fmt.Errorf("unmarshal world view: %w", err)
	}

	return Decide(view), nil
}

// NewAgentCard builds the A2A Agent Card for this agent, advertised at
// baseURL (the JSON-RPC endpoint is baseURL + "/invoke").
func NewAgentCard(baseURL string) *a2a.AgentCard {
	return &a2a.AgentCard{
		Name:               "go-agent",
		Description:        "Deterministic rule-based agent: heads to the market and trades with whoever it finds there.",
		URL:                baseURL + "/invoke",
		PreferredTransport: a2a.TransportProtocolJSONRPC,
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
		Capabilities:       a2a.AgentCapabilities{Streaming: false},
		Skills: []a2a.AgentSkill{
			{
				ID:          "trade_at_market",
				Name:        "Trade at market",
				Description: "Moves to the market location and trades with any other agent found there",
				Tags:        []string{"simulation", "rule-based"},
			},
		},
	}
}

// Register mounts this agent's A2A endpoints (JSON-RPC handler and the
// well-known agent card) onto mux. Registering into a caller-supplied mux
// (rather than returning a new one) lets callers reserve a listen address
// before the card's URL is known — see examples/go-agent/main.go and the
// e2e test in internal/api/e2e_test.go.
func Register(mux *http.ServeMux, card *a2a.AgentCard) {
	handler := a2asrv.NewHandler(&executor{})
	mux.Handle("/invoke", a2asrv.NewJSONRPCHandler(handler))
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(card))
}
