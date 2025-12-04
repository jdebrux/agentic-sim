package world

import "time"

// Event represents a structured occurrence within the world.
type Event struct {
	ID        string
	WorldID   string
	Tick      int64
	Timestamp time.Time
	Type      string
	ActorID   string
	TargetID  string
	Payload   map[string]any
}
