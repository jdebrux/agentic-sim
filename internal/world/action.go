package world

// ActionType enumerates the supported agent actions.
type ActionType string

const (
	ActionIdle     ActionType = "idle"
	ActionMove     ActionType = "move"
	ActionSpeak    ActionType = "speak"
	ActionInteract ActionType = "interact"
	ActionGreet    ActionType = "greet"
	ActionTrade    ActionType = "trade"
	ActionRest     ActionType = "rest"
)

// AgentAction captures an agent's intended action for a tick.
// ToolName/ToolArgs allow mapping to ADK tool calls.
type AgentAction struct {
	ActorID  string            `json:"actor_id,omitempty"`
	Type     ActionType        `json:"action"`
	TargetID string            `json:"target_id,omitempty"` // used for interactions
	Location string            `json:"location,omitempty"`  // used for move targets
	Message  string            `json:"message,omitempty"`   // used for speak
	Reason   string            `json:"reason,omitempty"`    // explains intent (e.g., move reason)
	ToolName string            `json:"tool_name,omitempty"` // ADK tool identifier
	ToolArgs map[string]string `json:"tool_args,omitempty"` // ADK tool arguments
	Metadata map[string]string `json:"metadata,omitempty"`  // free-form annotations
}
