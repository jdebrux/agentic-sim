package simulation

import (
	"testing"

	"github.com/jdebrux/agentic-sim/internal/adk"
	"github.com/jdebrux/agentic-sim/internal/agents"
)

func TestEngineRunnerModeSelection(t *testing.T) {
	cases := []struct {
		name   string
		mode   string
		expect any
	}{
		{"scripted", "scripted", nil},
		{"simple", "simple", (*adk.SimpleRunner)(nil)},
		{"rule", "rule", (*adk.RuleRunner)(nil)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngineWithConfig(EngineConfig{RunnerMode: tc.mode})
			for _, a := range engine.Agents {
				ba, ok := a.(*agents.BasicAgent)
				if !ok {
					t.Fatalf("expected BasicAgent, got %T", a)
				}

				switch tc.expect.(type) {
				case nil:
					if ba.Runner != nil {
						t.Fatalf("expected nil runner, got %T", ba.Runner)
					}
				case *adk.SimpleRunner:
					if _, ok := ba.Runner.(*adk.SimpleRunner); !ok {
						t.Fatalf("expected SimpleRunner, got %T", ba.Runner)
					}
				case *adk.RuleRunner:
					if _, ok := ba.Runner.(*adk.RuleRunner); !ok {
						t.Fatalf("expected RuleRunner, got %T", ba.Runner)
					}
				default:
					t.Fatalf("unknown expected type %T", tc.expect)
				}
			}
		})
	}
}

func TestEngineReasonerSelection(t *testing.T) {
	engine := NewEngineWithConfig(EngineConfig{ReasonerProvider: "mock"})
	for _, a := range engine.Agents {
		ba, ok := a.(*agents.BasicAgent)
		if !ok {
			t.Fatalf("expected BasicAgent, got %T", a)
		}
		if ba.Runner != nil {
			t.Fatalf("expected no runner when reasoner configured, got %T", ba.Runner)
		}
		if ba.Reasoner == nil {
			t.Fatalf("expected reasoner to be set")
		}
	}
}
