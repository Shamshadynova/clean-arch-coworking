package dummy

import (
	"context"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/domain"
)

// TODO: replace with real availability check against booking repository.
// Current implementation always returns nil, allowing double-bookings.
type AvailabilityChecker struct{}

func NewAvailabilityChecker() *AvailabilityChecker {
	return &AvailabilityChecker{}
}

func (a *AvailabilityChecker) CheckAvailability(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) error {
	return nil
}