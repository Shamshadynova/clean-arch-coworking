package dummy

import (
	"context"

	"github.com/example/coworking/internal/booking/domain"
)


type EventBus struct{}

func (EventBus) Publish(ctx context.Context, events []domain.Event) error {
	return nil
}

func NewEventBus() EventBus {
	return EventBus{}
}
