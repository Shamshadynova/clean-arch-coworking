package application

import (
	"context"
	"fmt"
	"log/slog"
	"errors"

	"github.com/google/uuid"

	
	"github.com/example/coworking/internal/booking/domain"
)

type Service struct {
	repo          BookingRepo //интерфейс interfaces.go сохраняет бронирование, ищет бронирование
	bus           EventBus ///интерфейс interfaces.go - публикация события 
	domainService *domain.BookingDomainService //проверка доступности, расчет цены
	uow           UnitOfWork //транзакция 
	logger        *slog.Logger
}
 //создаем конструктор, который будет собирать приложение-сервис бронирования 
func NewService( //объекты, котторые нужны для работы 
	Repo BookingRepo, //отв.за работу с хранилищем бронирования 
	EventBus EventBus, //публикация событий системы
	AvailabilityChecker AvailabilityChecker, //проверка доступности комнаты 
	PriceCalculator PriceCalculator,//расчет стоимости бронирования 
	UnitOfWork UnitOfWork, //управление транзакциями
	Logger *slog.Logger, //запись логов
) *Service {
	domainService := domain.NewBookingDomainService( //создаем доменный сервис бронирования 
		AvailabilityChecker, //проверка доступности комнаты 
		PriceCalculator, //стоимость бронирования 
	)

	return &Service{ //объект Service сохраняем все зависимости, которые передали 
		repo:          Repo,
		bus:           EventBus,
		domainService: domainService,
		uow:           UnitOfWork,
		logger:        Logger,
	}
}
//вызываем метод у сервиса s.CreateBooking, который принимает контекст r.Context() из хттп запроса и структуру с данными бронирования из dto
//возвращает id бронирования или ошибку
func (s *Service) CreateBooking(ctx context.Context, input CreateBookingInput) (uuid.UUID, error) {
	//проверка валидности вх.данных 
	//input вызывает метод Validate у CreateBookingInput
	//если ошибка не равна nil, значит данные не корректны 
	if err := input.Validate(); err != nil {
		return uuid.Nil, fmt.Errorf("invalid input: %w", err)
	}
//создаем переменную bookingID, для временного хранения данных 
//из транзакции 
	var bookingID uuid.UUID
//запускаем транзакцию
	err := s.uow.Execute(ctx, func(repo BookingRepo, eventStore EventStore) error {
 //проверяем IdempotencyKey уникальность и  ищем бронирование по ключу 
		existing, err := repo.FindByIdempotencyKey(ctx, input.IdempotencyKey)
	
		if err != nil && !errors.Is(err, domain.ErrBookingNotFound) { //если запись в БД не найдена 
			return fmt.Errorf("find by idempotency key: %w", err)
		}
		if existing != nil {
			s.logger.Info("idempotent booking request, returning existing",
				"booking_id", existing.ID(),
				"idempotency_key", input.IdempotencyKey,
			)
			
			bookingID = existing.ID() //сохраняем найденный айди 
			return nil //выходим из транзакции 
		}

		//создаем диапозон дат slot, передаем вх данные 
		slot, err := domain.NewDateRange(input.From, input.To)
		if err != nil { //если даты некорретны, то ошибка 
			return fmt.Errorf("invalid booking period: %w", err)
		}

		//создаем бронирование через доменный сервис 
		//передаем данные 
		booking, err := s.domainService.CreateValidatedBooking(
			ctx,
			input.RoomID,
			input.UserID,
			slot,
		)
		if err != nil {
			return fmt.Errorf("booking validation failed: %w", err)
		}
		//устанавливаем ключ 
		booking.SetIdempotencyKey(input.IdempotencyKey)

		//сохраняем бронь в базу 
		if err := repo.Save(ctx, booking); err != nil {
			return fmt.Errorf("failed to save booking: %w", err)
		}
//сохраняем айди созданного бронирования 
		bookingID = booking.ID()
		return nil
	})
//если внутри транзакции ошибка 
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
	)

	return bookingID, nil
}

func (s *Service) CancelBooking(ctx context.Context, id uuid.UUID) error {
    
    err := s.uow.Execute(ctx, func(repo BookingRepo, eventStore EventStore) error {

        booking, err := repo.FindByIDForUpdate(ctx, id) //ищем внутри транзакции бронь по id
        if err != nil { 
			//если ошибка, транзакция прерывается 
            return fmt.Errorf("find booking: %w", err)
        }

		//если бронь пустая (сущ ли такая бронь)
        if booking == nil {
			//если из памяти или бд ничего не вернул, значит ошибка 
            return domain.ErrBookingNotFound
        }

		//вызываем метод Cancel() у самого объекта booking
        if err := booking.Cancel(); err != nil {
			//если вернул ошибку, то транзакция прерывается 
            return err
        }

        //если отмена разрешена используем метод Save у репозитория repo (память или бд) 
        if err := repo.Save(ctx, booking); err != nil {
			//если запись не удалась, возвращаем ошибку 
            return fmt.Errorf("save booking: %w", err)
        }
		//если все прошло успешно 
        return nil
    }) 

	if err != nil {
		s.logger.Error("failed to cancel booking", 
		"booking_id", id, 
		"error", err, 
		)
		return err
	}

	s.logger.Info("booking cancelled successfully",
        "booking_id", id,)

    return nil // Возвращаем результат транзакции наружу, если внутри Execute ошибка
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
