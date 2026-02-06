package dummy

import (
	"context"

	"github.com/example/coworking/internal/booking/domain"
)


// TODO: implement real event bus (e.g. NATS, RabbitMQ) for cross-service communication.
type EventBus struct{}

func (EventBus) Publish(ctx context.Context, events []domain.Event) error {
	return nil
}

func NewEventBus() EventBus {
	return EventBus{}
}
