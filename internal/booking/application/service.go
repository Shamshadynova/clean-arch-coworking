package application

import (
	"context"
	"fmt"
	"log/slog"

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
		//тогда возвращаем пустой id и ошибку "invalid input
		return uuid.Nil, fmt.Errorf("invalid input: %w", err)
	}

	//проверка idempotency
	//сервис (s) обращается к repo и вызывает метод FindByIdempotencyKey у интерфейса BookingRepo
	//принимает контекст и уникальный ключ запроса, который приходит от клиента
	existing, err := s.repo.FindByIdempotencyKey(ctx, input.IdempotencyKey)
	//если ошибка равна nil(все норм ошибки нет) или бронирование не равно nil (бронирование найдено)
	//значит клиент с ключом уже существует 
	if err == nil && existing != nil {
		//сервис вызывает логгерс методом инфо из стандартной библиотеки и записывает информацию 
		s.logger.Info("idempotent booking request, returning existing", //получен повторный запрос бронирования, возвращаем сущ. бронирование
			"booking_id", existing.ID(), //добавляем поля id бронирования 
			"idempotency_key", input.IdempotencyKey, //ключ идемпотентности, по которому нашли бронь
		)
		return existing.ID(), nil //вовращаем id сущ. бронирования 
	}

	//диапозон дат бронирования 
	//возвращаем диапозон или ошибку := domain вызывает функцию NewDateRange(из domain/date_range)
	//передаем input.From (начало) и input.To (конец) бронирования 
	slot, err := domain.NewDateRange(input.From, input.To)
	//если функция вернула ошибку 
	if err != nil {
		//то возвращаем пустой id uuid.Nil и ошибку invalid booking period
		return uuid.Nil, fmt.Errorf("invalid booking period: %w", err)
	}

	//создается переменная типа uuid.UUID
	var bookingID uuid.UUID
	//запуск транзакции 
	//сервис (s) вызывает интерфейс UnitOfWork с методом Execute и пекредает функцию 
	//которая принимает repo и eventStore, возвращает ошибку 
	err = s.uow.Execute(ctx, func(repo BookingRepo, eventStore EventStore) error {
		//создаем бронирование. Выхываем доменный сервис с методом CreateValidatedBooking
		booking, err := s.domainService.CreateValidatedBooking(
			//передаем параметры
			ctx, //контекст запроса 
			input.RoomID, //id комнаты, которую пользователь хочет забронировать
			input.UserID, //id пользователя, который делает бронирование
			slot, //диапозон бронирования 
		)
		//если произошла ошибка, возвращаем текст booking validation failed
		if err != nil {
			return fmt.Errorf("booking validation failed: %w", err)
		}

		//успешное бронирование вызывает метод SetIdempotencyKey для структуры  Booking struct
		//и предаем вх.данные input.Уникальный ключ запроса
		booking.SetIdempotencyKey(input.IdempotencyKey)

		//сохраняем бронирование в базе данных через repo.Save(передаем контекст и бронирование)
		if err := repo.Save(ctx, booking); err != nil {
			//если ошибка, то выводим текст failed to save booking и откатываем транзакцию
			return fmt.Errorf("failed to save booking: %w", err)
		}

		//добавляем в раннее созданную переменную новый id бронирования 
		bookingID = booking.ID()
		
		//ошибок нет, транзакция зафиксировалась 
		return nil
	})

	//если ошибка внутри транзакции, то пишем в лог 
	if err != nil {
		//вызываем сервис.логгер.тип Error 
		s.logger.Error("failed to create booking",
			"room_id", input.RoomID,
			"user_id", input.UserID,
			"error", err,
		)
		//возвращаем пустой идентификатор и ошибку 
		return uuid.Nil, err
	}

	//сервпис пишет в лог. info если бронирование успешно создано 
	s.logger.Info("booking created",
		"booking_id", bookingID,
		"room_id", input.RoomID,
		"user_id", input.UserID,
	)
	//возвращаем успешное бронирование 
	return bookingID, nil
}

func (s *Service) CancelBooking(ctx context.Context, id uuid.UUID) error {
    
    err := s.uow.Execute(ctx, func(repo BookingRepo, eventStore EventStore) error {

        booking, err := repo.FindByID(ctx, id) //ищем внутри транзакции бронь по id
        if err != nil { 
			//если ошибка, транзакция прерывается 
            return fmt.Errorf("find booking: %w", err)
        }

		//если бронь пустая (сущ ли такая бронь)
        if booking == nil {
			//если из памяти или бд ничего не вернул, значит ошибка 
            return fmt.Errorf("booking not found")
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
