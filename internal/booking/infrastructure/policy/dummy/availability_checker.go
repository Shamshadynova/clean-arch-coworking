package dummy

import (
	"context"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/application"
	"github.com/example/coworking/internal/booking/domain"
)

// TODO: replace with real availability check against booking repository.
// Current implementation always returns nil, allowing double-bookings.
type AvailabilityChecker struct{
	repo application.BookingRepo //получаем бронирование из памяти 
}
//создаем конструктор 
//принимем репозиторий бронирования, возвращаем указатель на структуру 
func NewAvailabilityChecker(repo application.BookingRepo) *AvailabilityChecker {
	return &AvailabilityChecker{ //и сохраняем репо внутрь структуры
		repo: repo,
	}
}
//создаем метод для структуры 
//проверяем свободна ли комната в указанный период 
func (a *AvailabilityChecker) CheckAvailability(ctx context.Context, roomID uuid.UUID, slot domain.DateRange) error {
	//вызываем метод репозитория FindAllByRoomID
	//передаем контекст и айди комнаты 
	//получаем bookings список всех бронирований по этой комнате 
	bookings, err := a.repo.FindAllByRoomID(ctx, roomID)
	if err != nil {
		return  err
	}
	//перебираем бронирования 
	//b текущее бронирование 
	//если находим статус Cancelled
	for _, b := range bookings {
		if b.Status() == domain.Cancelled {
			continue //то пропускам его 
		}
	//проверка пересечения дат 
	//Slot() диапозон сущ бронирования 
	//slot новый запрашиваемый диапозон 
	//IsOverlapping проверяет пересекаются ли даты 
		if b.Slot().IsOverlapping(slot) {
			return domain.ErrRoomNotAvailable //если пересекаются, возвращаем ошибку
		}
	}
	return nil

}