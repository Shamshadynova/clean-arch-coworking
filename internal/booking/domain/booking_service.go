package domain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type BookingDomainService struct {
	availabilityChecker AvailabilityChecker
	priceCalculator     PriceCalculator
}

type AvailabilityChecker interface {
	CheckAvailability(ctx context.Context, roomID uuid.UUID, slot DateRange) error
}

type PriceCalculator interface {
	CalculatePrice(ctx context.Context, roomID uuid.UUID, slot DateRange) (Money, error)
}

func NewBookingDomainService(
	availabilityChecker AvailabilityChecker,
	priceCalculator PriceCalculator,
) *BookingDomainService {
	return &BookingDomainService{
		availabilityChecker: availabilityChecker,
		priceCalculator:     priceCalculator,
	}
}

func (s *BookingDomainService) CreateValidatedBooking(
	ctx context.Context,
	roomID, userID uuid.UUID,
	slot DateRange,
) (*Booking, error) {
	if err := s.availabilityChecker.CheckAvailability(ctx, roomID, slot); err != nil {
		return nil, fmt.Errorf("room not available: %w", err)
	}
	
	price, err := s.priceCalculator.CalculatePrice(ctx, roomID, slot)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate price: %w", err)
	}
	
	booking, err := NewBooking(roomID, userID, slot, price)
	if err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}
	
	return booking, nil
}