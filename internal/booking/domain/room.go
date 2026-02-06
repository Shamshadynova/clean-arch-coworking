package domain

import "github.com/google/uuid"

// Room represents a bookable space in the coworking area.
// TODO: integrate with external room-catalog service for real data.
type Room struct {
	id         uuid.UUID
	name       string
	capacity   int
	hourlyRate Money
}

// NewRoom creates a new Room instance.
func NewRoom(id uuid.UUID, name string, capacity int, hourlyRate Money) *Room {
	return &Room{
		id:         id,
		name:       name,
		capacity:   capacity,
		hourlyRate: hourlyRate,
	}
}

func (r *Room) ID() uuid.UUID      { return r.id }
func (r *Room) Name() string       { return r.name }
func (r *Room) Capacity() int      { return r.capacity }
func (r *Room) HourlyRate() Money  { return r.hourlyRate }
