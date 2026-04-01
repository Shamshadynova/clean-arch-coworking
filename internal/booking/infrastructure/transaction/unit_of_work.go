package transaction

import (
	"context"
	"database/sql"
	"sync"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/application"
	"github.com/example/coworking/internal/booking/domain"
	"github.com/example/coworking/internal/booking/infrastructure/postgres"
	"github.com/example/coworking/internal/booking/infrastructure/outbox"
)

// TODO: replace mutex-based UoW with SQL transaction (BEGIN/COMMIT/ROLLBACK)
// when switching to PostgreSQL.

type unitOfWork struct {
	db   *sql.DB //создадим пул соединений. потокобезопасен 
	bus application.EventBus //отправка событий, не связана с бд 
}
//создаем констурктор 
//принимает два интерфейса 
//возвращает интерфейс UnitOfWork 
func NewUnitOfWork(db *sql.DB, bus application.EventBus) application.UnitOfWork {
	return &unitOfWork{ //создаем структуру 
		db: db,
		bus : bus, 
	}
}
//создаем метод основной для UnitOfWork 
//принимает контекст и функцию бизнес логики
//выполнение бизнес-операции внутри транзакции 
func (u *unitOfWork) Execute(ctx context.Context, fn func(application.BookingRepo, application.EventStore) error) error {
	tx, err := u.db.BeginTx(ctx, nil) //начало транзакции
	 if err != nil { //если не удалось начать транзакцию, то завершаем работу
		return err
	 }

	 repo := postgres.NewBookingRepository(u.db) //непонимаю, если я передаю u.db, то обхожу транзакцию? 
	 eventStore := outbox.NewEventStore(u.bus) //вне транзакции 
	
	// Create transactional wrappers
	transactionalRepo := &transactionalRepo{ //обертка для сбора событий 
		repo:   repo,
		events: make([]domain.Event, 0),
	}
	
	transactionalEventStore := &transactionalEventStore{
		store: eventStore,
		repo:  transactionalRepo,
	}
	
	// Execute business logic
	err = fn(transactionalRepo, transactionalEventStore)
	if err != nil { //если ошибка 
		tx.Rollback() //откаьываем транзакцию 
		return err
	}

	// Save collected events after successful execution
	if len(transactionalRepo.events) > 0 {
		if err := eventStore.SaveEvents(ctx, transactionalRepo.events); err != nil {
			tx.Rollback()
			return err
		}
	}
	
	return tx.Commit()
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

func (t *transactionalRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return t.repo.FindByIDForUpdate(ctx, id)
}

func (t *transactionalRepo) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error) {
	return t.repo.FindByIdempotencyKey(ctx, key)
}

func (t *transactionalRepo) FindAllByRoomID(ctx context.Context, roomID uuid.UUID) ([]*domain.Booking, error) {
	return t.repo.FindAllByRoomID(ctx, roomID)
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