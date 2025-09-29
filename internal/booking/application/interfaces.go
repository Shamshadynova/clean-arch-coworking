package application

import (
	"context"
	"time"

	"github.com/google/uuid"
		"github.com/example/coworking/internal/booking/domain"

)

type BookingService interface {
	CreateBooking(ctx context.Context, roomID, userID uuid.UUID, from, to time.Time) (uuid.UUID, error)
	ConfirmPayment(ctx context.Context, bookingID uuid.UUID, txID string) error
}

type BookingRepo interface {
	Save(ctx context.Context, b *domain.Booking) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
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
