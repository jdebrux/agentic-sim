package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2aclient"
	"github.com/a2aproject/a2a-go/v2/a2aclient/agentcard"
	"github.com/jdebrux/agentic-sim/internal/telemetry"
	"github.com/jdebrux/agentic-sim/internal/world"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// A2AAgentClient wraps an external A2A agent for the sim engine.
type A2AAgentClient struct {
	id      string
	name    string
	cardURL string
	client  *a2aclient.Client
}

// NewA2AAgentClient connects to an external agent via its Agent Card URL.
func NewA2AAgentClient(ctx context.Context, agentID, agentName, cardURL string) (*A2AAgentClient, error) {
	tracer := otel.Tracer("simulation.a2a")
	ctx, span := tracer.Start(ctx, "a2a.connect")
	defer span.End()

	span.SetAttributes(
		attribute.String("agent.id", agentID),
		attribute.String("agent.name", agentName),
		attribute.String("agent.card_url", cardURL),
	)

	card, err := agentcard.DefaultResolver.Resolve(ctx, cardURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "resolve agent card")
		return nil, fmt.Errorf("resolve agent card at %s: %w", cardURL, err)
	}

	client, err := a2aclient.NewFromCard(ctx, card)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "connect to agent")
		return nil, fmt.Errorf("connect to agent at %s: %w", cardURL, err)
	}

	slog.Info("connected to external agent",
		"agent_id", agentID,
		"agent_name", agentName,
		"card_url", cardURL,
	)

	return &A2AAgentClient{
		id:      agentID,
		name:    agentName,
		cardURL: cardURL,
		client:  client,
	}, nil
}

func (a *A2AAgentClient) GetID() string   { return a.id }
func (a *A2AAgentClient) GetName() string { return a.name }

// Decide sends the world view as an A2A message and parses the action from the response.
func (a *A2AAgentClient) Decide(ctx context.Context, view world.WorldView) (world.AgentAction, error) {
	tracer := otel.Tracer("simulation.a2a")
	ctx, span := tracer.Start(ctx, "agent.decide")
	defer span.End()

	span.SetAttributes(
		attribute.String("agent.id", a.id),
		attribute.String("agent.name", a.name),
		attribute.String("agent.card_url", a.cardURL),
		attribute.String("agent.location", view.Self.Location),
		attribute.Int64("world.tick", view.Tick),
	)

	start := time.Now()
	viewJSON, err := json.Marshal(view)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal world view")
		telemetry.RecordDecisionError(ctx, a.id, "marshal_world_view")
		return world.AgentAction{}, fmt.Errorf("marshal world view: %w", err)
	}

	msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewTextPart(string(viewJSON)))

	ctx, roundtripSpan := tracer.Start(ctx, "agent.a2a_roundtrip")
	resp, err := a.client.SendMessage(ctx, &a2a.SendMessageRequest{Message: msg})
	roundtripSpan.End()
	roundtrip := time.Since(start)

	telemetry.RecordRoundtripLatency(ctx, a.id, roundtrip)
	roundtripSpan.SetAttributes(
		attribute.Int64("a2a.roundtrip_ms", roundtrip.Milliseconds()),
		attribute.String("agent.id", a.id),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "send message")
		telemetry.RecordDecisionError(ctx, a.id, "send_message")
		return world.AgentAction{}, fmt.Errorf("send message to agent: %w", err)
	}

	action, err := a.parseResponse(resp)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parse response")
		telemetry.RecordDecisionError(ctx, a.id, "parse_response")
		return action, err
	}

	span.SetAttributes(attribute.String("action.type", string(action.Type)))
	telemetry.RecordDecisionLatency(ctx, a.id, roundtrip)
	return action, nil
}

func (a *A2AAgentClient) parseResponse(resp a2a.SendMessageResult) (world.AgentAction, error) {
	switch v := resp.(type) {
	case *a2a.Task:
		return a.parseTaskResponse(v)
	case *a2a.Message:
		return a.parseMessageResponse(v)
	default:
		return world.AgentAction{
			ActorID: a.id,
			Type:    world.ActionIdle,
			Reason:  "unsupported response type",
		}, nil
	}
}

func (a *A2AAgentClient) parseTaskResponse(task *a2a.Task) (world.AgentAction, error) {
	if task == nil || len(task.Artifacts) == 0 {
		return world.AgentAction{
			ActorID: a.id,
			Type:    world.ActionIdle,
			Reason:  "no artifacts in agent response",
		}, nil
	}

	lastArtifact := task.Artifacts[len(task.Artifacts)-1]
	if lastArtifact == nil || len(lastArtifact.Parts) == 0 {
		return world.AgentAction{
			ActorID: a.id,
			Type:    world.ActionIdle,
			Reason:  "empty artifact in agent response",
		}, nil
	}

	return a.parseArtifactPart(lastArtifact.Parts[0])
}

func (a *A2AAgentClient) parseMessageResponse(msg *a2a.Message) (world.AgentAction, error) {
	if msg == nil || len(msg.Parts) == 0 {
		return world.AgentAction{
			ActorID: a.id,
			Type:    world.ActionIdle,
			Reason:  "empty message in agent response",
		}, nil
	}
	return a.parseArtifactPart(msg.Parts[0])
}

func (a *A2AAgentClient) parseArtifactPart(part *a2a.Part) (world.AgentAction, error) {
	if part == nil {
		return world.AgentAction{
			ActorID: a.id,
			Type:    world.ActionIdle,
			Reason:  "nil part in agent response",
		}, nil
	}

	text := part.Text()
	if text == "" {
		return world.AgentAction{
			ActorID: a.id,
			Type:    world.ActionIdle,
			Reason:  "non-text or empty response from agent",
		}, nil
	}

	var action world.AgentAction
	if err := json.Unmarshal([]byte(text), &action); err != nil {
		slog.Warn("failed to parse agent action",
			"agent", a.id,
			"raw", text,
			"error", err,
		)
		return world.AgentAction{
			ActorID: a.id,
			Type:    world.ActionIdle,
			Reason:  fmt.Sprintf("parse error: %v", err),
		}, nil
	}

	action.ActorID = a.id
	return action, nil
}
