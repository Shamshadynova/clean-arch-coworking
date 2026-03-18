package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/example/coworking/internal/booking/domain"
)

type BookingService interface {
	CreateBooking(ctx context.Context, input CreateBookingInput) (uuid.UUID, error)
	GetBooking(ctx context.Context, id uuid.UUID) (*BookingResponse, error)
	ConfirmPayment(ctx context.Context, input ConfirmPaymentInput) error
	CancelBooking(ctx context.Context, id uuid.UUID) error
}

type BookingRepo interface {
	Save(ctx context.Context, b *domain.Booking) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
	FindByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error)
	FindAllByRoomID(ctx context.Context, roomID uuid.UUID) ([]*domain.Booking, error)
}

type EventBus interface {
	Publish(ctx context.Context, events []domain.Event) error
}

type PaymentGateway interface {
	Charge(ctx context.Context, bookingID string, amount int64, currency string) (string, error)
}

type AvailabilityChecker interface {
	CheckAvailability(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) error
}

type PriceCalculator interface {
	CalculatePrice(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) (domain.Money, error)
}

type UnitOfWork interface {
	Execute(ctx context.Context, fn func(BookingRepo, EventStore) error) error
}

type EventStore interface {
	SaveEvents(ctx context.Context, events []domain.Event) error
}
