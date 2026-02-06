package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/domain"
)

type BookingRepository struct {
	mu              sync.RWMutex
	store           map[uuid.UUID]*domain.Booking
	idempotencyKeys map[string]*domain.Booking
}

func NewBookingRepository() *BookingRepository {
	return &BookingRepository{
		store:           make(map[uuid.UUID]*domain.Booking),
		idempotencyKeys: make(map[string]*domain.Booking),
	}
}

func (r *BookingRepository) Save(_ context.Context, booking *domain.Booking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[booking.ID()] = booking
	if booking.IdempotencyKey() != "" {
		r.idempotencyKeys[booking.IdempotencyKey()] = booking
	}
	return nil
}

func (r *BookingRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	booking, ok := r.store[id]
	if !ok {
		return nil, domain.ErrBookingNotFound
	}
	return booking, nil
}

func (r *BookingRepository) FindByIdempotencyKey(_ context.Context, key string) (*domain.Booking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	booking, ok := r.idempotencyKeys[key]
	if !ok {
		return nil, domain.ErrBookingNotFound
	}
	return booking, nil
}
