package world

// AgentCard describes an agent's capabilities, following the A2A spec.
// It is used both by the simserver to describe itself and by external agents.
type AgentCard struct {
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	URL                string            `json:"url"`
	Provider           *AgentProvider    `json:"provider,omitempty"`
	Version            string            `json:"version"`
	Capabilities       AgentCapabilities `json:"capabilities"`
	Skills             []AgentSkill      `json:"skills"`
	DefaultInputModes  []string          `json:"defaultInputModes"`
	DefaultOutputModes []string          `json:"defaultOutputModes"`
}

// AgentProvider identifies the organization providing the agent.
type AgentProvider struct {
	Organization string `json:"organization"`
	URL          string `json:"url,omitempty"`
}

// AgentCapabilities declares what an agent can do.
type AgentCapabilities struct {
	Streaming              bool `json:"streaming,omitempty"`
	PushNotifications      bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
}

// AgentSkill describes a skill exposed by an agent.
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Examples    []string `json:"examples,omitempty"`
	InputModes  []string `json:"inputModes,omitempty"`
	OutputModes []string `json:"outputModes,omitempty"`
}

// Task represents an A2A task (the unit of work between agents).
type Task struct {
	ID        string     `json:"id"`
	SessionID string     `json:"sessionId,omitempty"`
	Status    TaskStatus `json:"status"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
	History   []Message  `json:"history,omitempty"`
}

// TaskStatus captures the current state of a task.
type TaskStatus struct {
	State     TaskState `json:"state"`
	Message   *Message  `json:"message,omitempty"`
	Timestamp string    `json:"timestamp,omitempty"`
}

// TaskState is the lifecycle state of an A2A task.
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateCanceled      TaskState = "canceled"
	TaskStateFailed        TaskState = "failed"
	TaskStateRejected      TaskState = "rejected"
)

// Message is a single turn in an A2A conversation.
type Message struct {
	Role     MessageRole    `json:"role"`
	Parts    []Part         `json:"parts"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// MessageRole identifies who sent a message.
type MessageRole string

const (
	MessageRoleUser  MessageRole = "user"
	MessageRoleAgent MessageRole = "agent"
)

// Part is a piece of content within a message or artifact.
type Part struct {
	Type string `json:"type"` // "text", "file", "data"
	Text string `json:"text,omitempty"`
}

// Artifact is a deliverable produced during a task.
type Artifact struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parts       []Part         `json:"parts"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}
