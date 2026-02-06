package transaction

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/application"
	"github.com/example/coworking/internal/booking/domain"
)

type unitOfWork struct {
	bookingRepo application.BookingRepo
	eventStore  application.EventStore
	mu          sync.Mutex
}

func NewUnitOfWork(bookingRepo application.BookingRepo, eventStore application.EventStore) application.UnitOfWork {
	return &unitOfWork{
		bookingRepo: bookingRepo,
		eventStore:  eventStore,
	}
}

func (u *unitOfWork) Execute(ctx context.Context, fn func(application.BookingRepo, application.EventStore) error) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	
	// Create transactional wrappers
	transactionalRepo := &transactionalRepo{
		repo:   u.bookingRepo,
		events: make([]domain.Event, 0),
	}
	
	transactionalEventStore := &transactionalEventStore{
		store: u.eventStore,
		repo:  transactionalRepo,
	}
	
	// Execute business logic
	if err := fn(transactionalRepo, transactionalEventStore); err != nil {
		return err
	}
	
	// Save collected events after successful execution
	if len(transactionalRepo.events) > 0 {
		if err := u.eventStore.SaveEvents(ctx, transactionalRepo.events); err != nil {
			return err
		}
	}
	
	return nil
}

type transactionalRepo struct {
	repo   application.BookingRepo
	events []domain.Event
	mu     sync.Mutex
}

func (t *transactionalRepo) Save(ctx context.Context, b *domain.Booking) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	// Collect events before saving
	events := b.PullEvents()
	t.events = append(t.events, events...)
	
	return t.repo.Save(ctx, b)
}

func (t *transactionalRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return t.repo.FindByID(ctx, id)
}

func (t *transactionalRepo) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error) {
	return t.repo.FindByIdempotencyKey(ctx, key)
}

type transactionalEventStore struct {
	store application.EventStore
	repo  *transactionalRepo
}

func (t *transactionalEventStore) SaveEvents(ctx context.Context, events []domain.Event) error {
	t.repo.mu.Lock()
	defer t.repo.mu.Unlock()
	
	// Collect events for later processing
	t.repo.events = append(t.repo.events, events...)
	return nil
}