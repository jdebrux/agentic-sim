package adk

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jdebrux/agentic-sim/internal/model"
	"github.com/jdebrux/agentic-sim/internal/world"
)

// TestMockRunner ensures the mock increments calls and returns configured responses.
func TestMockRunner(t *testing.T) {
	t.Run("returns configured response", func(t *testing.T) {
		expected := ReasoningResponse{
			Action: model.AgentAction{
				ActorID:  "agent-1",
				Type:     model.ActionSpeak,
				Message:  "hello",
				ToolName: "speak",
			},
			Notes: map[string]string{"note": "test"},
		}
		mock := &MockRunner{Response: expected}

		resp, err := mock.DecideAction(context.Background(), ReasoningRequest{
			AgentID:   "agent-1",
			AgentName: "Alice",
			View:      world.WorldView{},
			Goal:      "greet",
		})

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !reflect.DeepEqual(resp, expected) {
			t.Fatalf("expected response %+v, got %+v", expected, resp)
		}
		if mock.Calls != 1 {
			t.Fatalf("expected Calls to be 1, got %d", mock.Calls)
		}
	})

	t.Run("propagates configured error", func(t *testing.T) {
		expectedErr := errors.New("runner error")
		expectedResp := ReasoningResponse{
			Action: model.AgentAction{ActorID: "agent-1", Type: model.ActionIdle},
		}
		mock := &MockRunner{Response: expectedResp, Err: expectedErr}

		resp, err := mock.DecideAction(context.Background(), ReasoningRequest{})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error %v, got %v", expectedErr, err)
		}
		if !reflect.DeepEqual(resp, expectedResp) {
			t.Fatalf("expected response %+v, got %+v", expectedResp, resp)
		}
		if mock.Calls != 1 {
			t.Fatalf("expected Calls to be 1, got %d", mock.Calls)
		}
	})
}
