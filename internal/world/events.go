package world

import "time"

// Event represents a structured occurrence within the world.
type Event struct {
	ID        string         `json:"id"`
	WorldID   string         `json:"world_id"`
	Tick      int64          `json:"tick"`
	Timestamp time.Time      `json:"timestamp"`
	Type      string         `json:"type"`
	ActorID   string         `json:"actor_id"`
	TargetID  string         `json:"target_id,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
}
