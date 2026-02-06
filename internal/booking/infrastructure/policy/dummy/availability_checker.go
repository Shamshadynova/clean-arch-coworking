package dummy

import (
	"context"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/domain"
)

type AvailabilityChecker struct{}

func NewAvailabilityChecker() *AvailabilityChecker {
	return &AvailabilityChecker{}
}

func (a *AvailabilityChecker) CheckAvailability(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) error {
	return nil
}