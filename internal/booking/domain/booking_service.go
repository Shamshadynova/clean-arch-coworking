package domain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type BookingDomainService struct {
	availabilityChecker AvailabilityChecker //проверка, свободна ли комната
	priceCalculator     PriceCalculator //расчет стоимости 
}

type AvailabilityChecker interface { 
	//метод проверяет свободна ли комната в заданный период 
	//принимает контекст, id комнаты и диапозон 
	//возвращает ошибку, если комната недоступна 
	CheckAvailability(ctx context.Context, roomID uuid.UUID, slot DateRange) error
}

type PriceCalculator interface {
	//метод производит расчет стоимости 
	//принимает контекст, id комнаты и диапозон 
	//возвращает стоимость или ошибку 
	CalculatePrice(ctx context.Context, roomID uuid.UUID, slot DateRange) (Money, error)
}

//создаем контурктор 
//функция принимает зависимости AvailabilityChecker (свободна ли комната)
//и стоимость PriceCalculator комнаты 
func NewBookingDomainService(
	availabilityChecker AvailabilityChecker,
	priceCalculator PriceCalculator,
) *BookingDomainService { //возвращает указатель на структуру и все зависимости сохраняем в структуру 
	return &BookingDomainService{
		availabilityChecker: availabilityChecker,
		priceCalculator:     priceCalculator,
	}
}

//создаем метод  CreateValidatedBooking для *BookingDomainService
func (s *BookingDomainService) CreateValidatedBooking(
	ctx context.Context, //принимает контекст 
	roomID, userID uuid.UUID, //id комнаты и пользователя типа uuid.UUID
	slot DateRange, //диапозон дат 
) (*Booking, error) { //возвращает указатель на структуру Booking из файла booking.go или ошибку 
	//проверка доступности комнаты 
	 //алиас s *BookingDomainService вызывает availabilityChecker с методом CheckAvailability
	 //передаем контекст, id комнаты и диапозон 
	if err := s.availabilityChecker.CheckAvailability(ctx, roomID, slot); err != nil {
		//если не nil, то ошибка room not available
		return nil, fmt.Errorf("room not available: %w", err)
	}
	
	//расчет стоимости
	//алиас s *BookingDomainService вызывает priceCalculator с методом CalculatePrice
	////передаем контекст, id комнаты и диапозон 
	price, err := s.priceCalculator.CalculatePrice(ctx, roomID, slot)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate price: %w", err)
	}
	
	//подтверждение бронирования или ошибка 
	//вызываем функцию NewBooking из файла booking.go 
	//передаем id комнаты и пользователя, диапозон и цену 
	booking, err := NewBooking(roomID, userID, slot, price)
	if err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}
	// если все шаги прошли успешно
	// возвращаем созданное бронирование
	return booking, nil
}