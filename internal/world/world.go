package world

import "log"

// World represents the environment in which agents exist.
type World struct {
	Timestep int
	Events   []string
}

func NewWorld() *World {
	return &World{
		Timestep: 0,
		Events:   []string{},
	}
}

func (w *World) Advance() {
	w.Timestep++
	log.Printf("World advanced to timestep %d", w.Timestep)
}
