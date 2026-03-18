package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/domain"
)

// TODO: replace in-memory store with PostgreSQL implementation.
//структура  BookingRepository имеет поля 
type BookingRepository struct {
	mu              sync.RWMutex //mu для защиты от двойной записи 
	store           map[uuid.UUID]*domain.Booking //store мапа с ключом типа uuid.UUID и значением указатель на структуру Booking из файла domain
	idempotencyKeys map[string]*domain.Booking //идемпотентный ключ защита от двойного бронирования мапа с ключом типа стринг и значением указатель на структуру Booking из файла domain
}

//создаем конструктор NewBookingRepository
//создаем новый объект структуры BookingRepository. Готовим место где хранить данные  
func NewBookingRepository() *BookingRepository { 
	return &BookingRepository{//возвращаем указатель на новую структуру 
		store:           make(map[uuid.UUID]*domain.Booking), //новая мапа для хранения бронирования
		idempotencyKeys: make(map[string]*domain.Booking), //новая мапа для уникальных ключей 
	}
}

//метод Save для структуры BookingRepository
//принимает контекст (для отмены операции) и переменную типа с данными о бронировании, возвращаем ошибку 
func (r *BookingRepository) Save(_ context.Context, booking *domain.Booking) error {
	r.mu.Lock() //блокируем запись. Только одна горутина может менять мапу 
	defer r.mu.Unlock() //снимаем блокировку, когда запись сделана 
	r.store[booking.ID()] = booking //добавляем новое бронирование в таблицу. Ключ id бронирования
	if booking.IdempotencyKey() != "" { //если идемпотетный ключ не пустой
		r.idempotencyKeys[booking.IdempotencyKey()] = booking //то сохраняем его в табл
	}
	//возвращаем нил, если все ок  
	return nil
}

//создаем метод FindByID поиска по id 
//принимает контекст (для отмены операции) и id типа uuid.UUID
//возвращаем найденное бронирование или ошибку
func (r *BookingRepository) FindByID(_ context.Context, id uuid.UUID) (*domain.Booking, error) {
	r.mu.RLock() //читать может много горутин, но на запись только одна горутина 
	defer r.mu.RUnlock()//снимаем блокировку, после того как функция отработает 
	booking, ok := r.store[id] //ищем бронирование в мапе store
	if !ok { //если бронирование по id не найдено 
		//то возвращем ошибку
		return nil, domain.ErrBookingNotFound
	} //если все ок, возвращаем бронирование 
	return booking, nil
}

//метод FindByIdempotencyKey поиска по ключу 
//не был ли уже выполнен такой же POST запрос 
func (r *BookingRepository) FindByIdempotencyKey(_ context.Context, key string) (*domain.Booking, error) {
	r.mu.RLock() //блокируем чтение 
	defer r.mu.RUnlock() //снимаем блокировку после выполнения функции 
	booking, ok := r.idempotencyKeys[key] //ищем бронирование в мапе idempotencyKeys
	if !ok { //если не найдено, выводим ошибку 
		return nil, domain.ErrBookingNotFound
	}
	//если все ок, возвращаем найденное бронирование 
	return booking, nil
}

func (r *BookingRepository) FindAllByRoomID(ctx context.Context, roomID uuid.UUID) ([]*domain.Booking, error) {
	var result []*domain.Booking //сощдаем переменную для нового результата

	for _, b := range r.store { //перебираем все бронирования в памяти
		if b.RoomID() == roomID { //фильтруем нужную комнаты 
			result = append(result, b) //добавляем в результат (переменную)
		}
	}

	return result, nil //возвращаем список 
}

