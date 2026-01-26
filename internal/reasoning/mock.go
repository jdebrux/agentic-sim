package reasoning

import (
	"context"

	"github.com/jdebrux/agentic-sim/internal/model"
)

// MockReasoner returns a configured response and tracks calls; useful for tests.
type MockReasoner struct {
	Response Response
	Err      error
	Calls    int
}

func (m *MockReasoner) DecideAction(ctx context.Context, req Request) (Response, error) {
	_ = ctx
	_ = req
	m.Calls++
	resp := m.Response
	if resp.Action.ActorID == "" {
		resp.Action.ActorID = req.AgentID
	}
	return resp, m.Err
}

// NoopReasoner returns a simple speak action noting that no provider was selected.
type NoopReasoner struct{}

func (n *NoopReasoner) DecideAction(ctx context.Context, req Request) (Response, error) {
	_ = ctx
	msg := "noop reasoner for " + req.AgentID
	return Response{
		Action: model.AgentAction{
			ActorID:  req.AgentID,
			Type:     model.ActionSpeak,
			Message:  msg,
			Reason:   "noop_reasoner",
			ToolName: "speak",
		},
		Notes: map[string]string{"provider": "noop"},
	}, nil
}
