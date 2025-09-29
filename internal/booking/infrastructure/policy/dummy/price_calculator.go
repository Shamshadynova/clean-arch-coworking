package dummy

import (
	"context"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/domain"
)

type PriceCalculator struct{}

func NewPriceCalculator() *PriceCalculator {
	return &PriceCalculator{}
}

func (p *PriceCalculator) CalculatePrice(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) (domain.Money, error) {
	return domain.NewMoney(100, "USD"), nil
}