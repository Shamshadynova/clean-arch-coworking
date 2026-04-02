package domain

import (
	"strings"

	"github.com/google/uuid"
)
//для статуса бронирования 
type BookingStatus int //BookingStatus может быть только числовое значение 

const ( //константы статусов бронирования 
	Pending BookingStatus = iota //ожидание оплаты = автоматичсеки присваивает числа 
	Paid //оплаячено 
	Cancelled //отменено 
)
 //создаем структуру Booking 
type Booking struct {
	//с полями 
	id              uuid.UUID //тип уникальный идетификатор 
	roomID          uuid.UUID//тип уникальный идетификатор 
	userID          uuid.UUID//тип уникальный идетификатор 
	slot            DateRange //структура из date_range
	price           Money //структура из money
	status          BookingStatus //наш статус, который мы создали
	events          []Event //слайс событий, интерфейс из events
	idempotencyKey  string //строка
	transactionID   string //строка 
}
//создаем функцию на новое бронирование
//функция принимает id комнаты и пользователя тип уникальный идетификатор 
//диапозон дат тип структура из date_range
//стоимость тип структура из money
//возвращает указатель на структуру (запишутся новые данные) или ошибку 
func NewBooking(roomID, userID uuid.UUID, slot DateRange, price Money) (*Booking, error) {
	//если диапозон с методом IsZero из date_range
	//не выбран (пустой, 0) вызвращаем nil и ошибку из errors
	if slot.IsZero() {
		return nil, ErrInvalidRange
	}
	//если все успешно, создаем объект бронирования 
	b := &Booking{
		id:     uuid.New(), //генерация нового уникального идентификатора 
		roomID: roomID, //id комнаты 
		userID: userID, //id пользователя 
		slot:   slot, //диапозон 
		price:  price,//стоимость 
		status: Pending, //статус бронирования 
	}
	//создание события 
	//алиас (b) структуры Booking вызывает метод raise
	//в который передаем структуру RoomBooked из events
	b.raise(RoomBooked{ //передаем данные о бронировании 
		BookingID: b.id.String(), //id созданного бронирования 
		RoomID: roomID.String(), //id комнаты 
		UserID: userID.String()}) //id пользователя 
		//еслии все успешно возвращаем данные о бронировании и nil если ошибки нет 
	return b, nil
}

// конструктор для восстановления из БД 
func MarshalBooking(
	id uuid.UUID,
	roomID uuid.UUID,
	userID uuid.UUID,
	slot DateRange,
	status int,
	idempotencyKey string,
) *Booking {
	return &Booking{
		id: id,
		roomID: roomID,
		userID: userID,
		slot: slot,
		status: BookingStatus(status),
		idempotencyKey: idempotencyKey,
	}
}

//создаются методы для структуры Booking 
func (b *Booking) ID() uuid.UUID         { return b.id }
func (b *Booking) RoomID() uuid.UUID      { return b.roomID }
func (b *Booking) UserID() uuid.UUID      { return b.userID }
func (b *Booking) Slot() DateRange        { return b.slot }
func (b *Booking) Price() Money           { return b.price }
func (b *Booking) Status() BookingStatus  { return b.status }
func (b *Booking) TransactionID() string  { return b.transactionID }
//метод ConfirmPayment для структуры Booking
//принимает id платежной транзакции и возвращает ошибку, если что-то не так
func (b *Booking) ConfirmPayment(txID string) error {
	//если id пустой или содержит пробелы 
	if strings.TrimSpace(txID) == "" { //удаляет пробелы в начале и в конце строки 
		//то возвращаем ошибку 
		return ErrInvalidTransaction
	}
	//если бронирование не в режиме ожидания оплаты 
	if b.status != Pending {
		//возвращаем ошибку 
		return ErrWrongState
	}
	//бронирование оплачено
	b.status = Paid
	//сохраняем id транзакции 
	b.transactionID = txID
	//бронирование вызывает метод raise для отправки события 
	//создаем событие BookingConfirmed - бронирование успешно оплачено 
	b.raise(BookingConfirmed{BookingID: b.id.String(), TxID: txID})
	//возвращаем nil если все прошло успешно 
	return nil
}

func (b *Booking) Cancel() error {
	if b.status == Paid { //если бронь уже оплачена
		return ErrBookingAlreadyPaid //вовзращаем ошибку 
	}
	if b.status == Cancelled { //если бронь уже отменена 
		return ErrAlreadyCancelled //возвращаем ошибку 
	}
	
	b.status = Cancelled //в других случаях отменяем бронирование 

	//событие отмены, для отправки уведомления пользователю 
	b.raise(BookingCancelled{BookingID: b.id.String()})
	return nil
}


//метод SetIdempotencyKey для структуры Booking 
func (b *Booking) SetIdempotencyKey(key string) {
	//добавляет ключ и сохраняет в поле idempotencyKey
	//защита от повторных POST ЗАПРОСОВ 
	b.idempotencyKey = key
}

func (b *Booking) IdempotencyKey() string {
	//возвращает добавляенный ключ 
	return b.idempotencyKey
}

func (b *Booking) IsPaymentConfirmed(txID string) bool {
	//возвращаем true если оба условия выпаолнены 
	return b.status == Paid && b.transactionID == txID
}

func (b *Booking) PullEvents() []Event {
	//сохраняем все события из структуры 
	ev := b.events
	//очищаем список событий 
	b.events = nil
	return ev
}

func (b *Booking) raise(e Event) {
	//добавляем событие в список 
	b.events = append(b.events, e)
}
