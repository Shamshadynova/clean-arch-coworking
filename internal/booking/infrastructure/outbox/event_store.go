package outbox

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/application"
	"github.com/example/coworking/internal/booking/domain"
)

type OutboxEvent struct {
	ID        uuid.UUID
	EventType string
	EventData json.RawMessage
	Published bool
	CreatedAt time.Time
}

type EventStore struct {
	mu     sync.RWMutex
	events []OutboxEvent
	bus    application.EventBus
}

func NewEventStore(bus application.EventBus) application.EventStore {
	return &EventStore{
		events: make([]OutboxEvent, 0),
		bus:    bus,
	}
}

func (s *EventStore) SaveEvents(ctx context.Context, events []domain.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		
		outboxEvent := OutboxEvent{
			ID:        uuid.New(),
			EventType: getEventType(event),
			EventData: data,
			Published: false,
			CreatedAt: time.Now(),
		}
		
		s.events = append(s.events, outboxEvent)
	}
	
	go s.publishPendingEvents(ctx)
	
	return nil
}

func (s *EventStore) publishPendingEvents(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	var domainEvents []domain.Event
	for i, event := range s.events {
		if !event.Published {
			var domainEvent domain.Event
			switch event.EventType {
			case "RoomBooked":
				var e domain.RoomBooked
				if err := json.Unmarshal(event.EventData, &e); err == nil {
					domainEvent = e
				}
			case "BookingConfirmed":
				var e domain.BookingConfirmed
				if err := json.Unmarshal(event.EventData, &e); err == nil {
					domainEvent = e
				}
			}
			
			if domainEvent != nil {
				domainEvents = append(domainEvents, domainEvent)
				s.events[i].Published = true
			}
		}
	}
	
	if len(domainEvents) > 0 {
		_ = s.bus.Publish(ctx, domainEvents)
	}
}

func getEventType(event domain.Event) string {
	switch event.(type) {
	case domain.RoomBooked:
		return "RoomBooked"
	case domain.BookingConfirmed:
		return "BookingConfirmed"
	default:
		return "Unknown"
	}
}