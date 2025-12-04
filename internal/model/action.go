package model

// ActionType enumerates the supported agent actions.
type ActionType string

const (
	ActionIdle     ActionType = "idle"
	ActionMove     ActionType = "move"
	ActionSpeak    ActionType = "speak"
	ActionInteract ActionType = "interact"
)

// AgentAction captures an agent's intended action for a tick.
// ToolName/ToolArgs allow mapping to ADK tool calls.
type AgentAction struct {
	ActorID  string
	Type     ActionType
	TargetID string            // optional, used for interactions
	Location string            // optional, used for move targets
	Message  string            // optional, used for speak
	ToolName string            // optional, ADK tool identifier
	ToolArgs map[string]string // optional, ADK tool arguments
	Metadata map[string]string // optional, free-form annotations
}
