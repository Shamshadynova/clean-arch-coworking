package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/domain"
)

type Service struct {
	repo          BookingRepo
	bus           EventBus
	domainService *domain.BookingDomainService
	uow           UnitOfWork
}

func NewService(
	Repo                BookingRepo,
	EventBus            EventBus,
	AvailabilityChecker AvailabilityChecker,
	PriceCalculator     PriceCalculator,
	UnitOfWork          UnitOfWork,
) *Service {
	domainService := domain.NewBookingDomainService(
		AvailabilityChecker,
		PriceCalculator,
	)
	
	return &Service{
		repo:          Repo,
		bus:           EventBus,
		domainService: domainService,
		uow:           UnitOfWork,
	}
}

func (s *Service) CreateBooking(ctx context.Context, input CreateBookingInput) (uuid.UUID, error) {
	if err := input.Validate(); err != nil {
		return uuid.Nil, fmt.Errorf("invalid input: %w", err)
	}
	
	existing, err := s.repo.FindByIdempotencyKey(ctx, input.IdempotencyKey)
	if err == nil && existing != nil {
		return existing.ID(), nil
	}
	
	slot, err := domain.NewDateRange(input.From, input.To)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid booking period: %w", err)
	}
	
	var bookingID uuid.UUID
	err = s.uow.Execute(ctx, func(repo BookingRepo, eventStore EventStore) error {
		booking, err := s.domainService.CreateValidatedBooking(
			ctx,
			input.RoomID,
			input.UserID,
			slot,
		)
		if err != nil {
			return fmt.Errorf("booking validation failed: %w", err)
		}
		
		booking.SetIdempotencyKey(input.IdempotencyKey)
		
		if err := repo.Save(ctx, booking); err != nil {
			return fmt.Errorf("failed to save booking: %w", err)
		}
		
		bookingID = booking.ID()
		return nil
	})
	
	if err != nil {
		return uuid.Nil, err
	}
	
	return bookingID, nil
}

func (s *Service) ConfirmPayment(ctx context.Context, input ConfirmPaymentInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}
	
	err := s.uow.Execute(ctx, func(repo BookingRepo, eventStore EventStore) error {
		booking, err := repo.FindByID(ctx, input.BookingID)
		if err != nil {
			return fmt.Errorf("find booking: %w", err)
		}
		
		if booking.IsPaymentConfirmed(input.TransactionID) {
			return nil
		}
		
		if err := booking.ConfirmPayment(input.TransactionID); err != nil {
			return fmt.Errorf("confirm payment: %w", err)
		}
		
		if err := repo.Save(ctx, booking); err != nil {
			return fmt.Errorf("save booking: %w", err)
		}
		
		return nil
	})
	
	return err
}