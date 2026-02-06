package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/domain"
)

type Service struct {
	repo          BookingRepo
	bus           EventBus
	domainService *domain.BookingDomainService
	uow           UnitOfWork
	logger        *slog.Logger
}

func NewService(
	Repo BookingRepo,
	EventBus EventBus,
	AvailabilityChecker AvailabilityChecker,
	PriceCalculator PriceCalculator,
	UnitOfWork UnitOfWork,
	Logger *slog.Logger,
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
		logger:        Logger,
	}
}

func (s *Service) CreateBooking(ctx context.Context, input CreateBookingInput) (uuid.UUID, error) {
	if err := input.Validate(); err != nil {
		return uuid.Nil, fmt.Errorf("invalid input: %w", err)
	}

	existing, err := s.repo.FindByIdempotencyKey(ctx, input.IdempotencyKey)
	if err == nil && existing != nil {
		s.logger.Info("idempotent booking request, returning existing",
			"booking_id", existing.ID(),
			"idempotency_key", input.IdempotencyKey,
		)
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
		s.logger.Error("failed to create booking",
			"room_id", input.RoomID,
			"user_id", input.UserID,
			"error", err,
		)
		return uuid.Nil, err
	}

	s.logger.Info("booking created",
		"booking_id", bookingID,
		"room_id", input.RoomID,
		"user_id", input.UserID,
	)
	return bookingID, nil
}

func (s *Service) GetBooking(ctx context.Context, id uuid.UUID) (*BookingResponse, error) {
	booking, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find booking: %w", err)
	}

	statusName := "pending"
	switch booking.Status() {
	case domain.Paid:
		statusName = "paid"
	case domain.Cancelled:
		statusName = "cancelled"
	}

	return &BookingResponse{
		ID:            booking.ID(),
		RoomID:        booking.RoomID(),
		UserID:        booking.UserID(),
		From:          booking.Slot().From,
		To:            booking.Slot().To,
		PriceAmount:   booking.Price().Amount,
		PriceCurrency: booking.Price().Currency,
		Status:        statusName,
	}, nil
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

	if err != nil {
		s.logger.Error("failed to confirm payment",
			"booking_id", input.BookingID,
			"error", err,
		)
		return err
	}

	s.logger.Info("payment confirmed",
		"booking_id", input.BookingID,
		"transaction_id", input.TransactionID,
	)
	return nil
}
