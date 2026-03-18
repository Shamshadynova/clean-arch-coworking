package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/application"
	"github.com/example/coworking/internal/booking/domain"
)

// BookingHandler contains HTTP handlers for the booking resource.
type BookingHandler struct {
	svc    application.BookingService //объект, который содержит бизнес-логику из папки application в файле interfaces 
	logger *slog.Logger //для записи логов, без копирования 
}

// NewRouter builds the HTTP handler tree with routes and middleware.
//принимает объект бизнес-логики и логгер для записи логов. Вовращает обработчик http(интерфейс)
func NewRouter(svc application.BookingService, logger *slog.Logger) http.Handler {
	//создаем структуру BookingHandler и туда передаем зависимости
	h := &BookingHandler{svc: svc, logger: logger}

	//создание роутера (маршрута) 
	//стандартный роутер. Его задача получить запрос, посмотреть URL, вызвать нужную функцию
	//если зарег.POST /bookings и приходит запрос POST /bookings, то ServeMux вызывает CreateBooking() из application.BookingService
	mux := http.NewServeMux()
	//регистрация endpoints
	//HandleFunc — это метод структуры ServeMux
	mux.HandleFunc("POST /bookings", h.CreateBooking)
	mux.HandleFunc("POST /bookings/{id}/cancel", h.CancelBooking) 
	mux.HandleFunc("GET /bookings/{id}", h.GetBooking)//маршрут GET /bookings/{id} и метод структуры BookingHandler

	// Apply middleware chain: Recovery -> Logger -> RequestID -> mux
	//создается перемнная  handler типа http.Handler 
	var handler http.Handler = mux
	//каждый middleware это функция 
	handler = RequestID(handler) //добавляет уникальный запрос 
	handler = Logger(handler, logger)//логирует HTTP запрос
	handler = Recovery(handler, logger) //ловит панику

	return handler //возвращает готовый HTTP обработчик
}

// CreateBooking handles POST /bookings.
//вызывается метод CreateBooking для структуры BookingHandler принимает 
//Стандартные параметры для HTTP
//w http.ResponseWriter объект для отправки ответа, r *http.Request объект HTTP запроса
func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	//создаем струкутру запроса
	//она описывает, какой JSON мы ожидаем от клиента
	var req struct {
		RoomID         string    `json:"room_id"` //поле room_id записать в RoomID 
		UserID         string    `json:"user_id"`
		From           time.Time `json:"from"`
		To             time.Time `json:"to"`
		IdempotencyKey string    `json:"idempotency_key"`
	}
	//берем тело запроса r.Body (то есть объект HTTP запроса)
	//создаем json.NewDecoder и декодируем в структуру .Decode(&req)
	//передаем &req, что бы Decode записал значения в структуру 
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		//обработка ошибок. Если JSON неправильный
		//возвращаем ответ от сервера 400 Bad Request
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//проверяем если req.RoomID и req.UserID пустые
	if req.RoomID == "" || req.UserID == "" {
		//возвращаем ошибку текст ошибки и ошибку от сервера 400 Bad Request
		http.Error(w, "room_id and user_id are required", http.StatusBadRequest)
		return
	}
	//парсинг UUID (уникальный идетификатор) комнаты 
	//берем строку req.RoomID "550e8400-e29b-41d4-a716-446655440000"
	//и превращаем ее в тип uuid.UUID из библиотеки "github.com/google/uuid"
	roomID, err := uuid.Parse(req.RoomID)
	//если номер не соотв формату, то возращаем ошибку 
	if err != nil {
		http.Error(w, "invalid room_id", http.StatusBadRequest)
		return
	}

	//парсинг UUID (уникальный идетификатор) пользователя 
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	//если клиент не передал IdempotencyKey (пустая строка) 
	if req.IdempotencyKey == "" {
		//сервер создает новый 
		req.IdempotencyKey = uuid.New().String()
	}

	//создаем input для сервиса
	//добавляем данные в application DTO
	input := application.CreateBookingInput{
		RoomID:         roomID,
		UserID:         userID,
		From:           req.From,
		To:             req.To,
		IdempotencyKey: req.IdempotencyKey,
	}

	//h это структура BookingHandler
	//svc application.BookingService
	//handler говорит сервису создать бронирование (используй контекст при запросе, данные в application DTO)
	id, err := h.svc.CreateBooking(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}

	//w - http.ResponseWriter объект через который сервер отправляет ответ клиенту
	//устанавливаем .Set("Content-Type". Это говорит клиенту, что ответ будет JSON
	w.Header().Set("Content-Type", "application/json")
	//статус ответа HTTP/1.1 201 Created
	w.WriteHeader(http.StatusCreated)
	//json.NewEncoder(w) пишет в w, .Encode конвертирует го-объект в JSON
	//ошибку игнорируем, поэтому _
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()}) //кодируем мапу в JSON. id имеет тип: uuid.UUID, поэтому id.String() превращает UUID в строку, так как  JSON ожидает строку
}


func (h *BookingHandler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")//получаесм id из url 
	bookingID, err := uuid.Parse(rawID)//парсим в uuid.UUID
	if err != nil {
		http.Error(w, "invalid booking id", http.StatusBadRequest)
		return
	}
//вызываем метод сервиса и записываем результат (ошибку) в переменнную 
	err = h.svc.CancelBooking(r.Context(), bookingID)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrBookingAlreadyPaid): //бронь уже оплачена 
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, domain.ErrAlreadyCancelled):  //бронирование уже отменено
			http.Error(w, err.Error(), http.StatusConflict )
		case errors.Is(err, domain.ErrBookingNotFound): //бронирование не найдено 
			http.Error(w, err.Error(), http.StatusNotFound)
		default: //системные ошибки 
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetBooking handles GET /bookings/{id}.
func (h *BookingHandler) GetBooking(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	bookingID, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "invalid booking id", http.StatusBadRequest)
		return
	}

	resp, err := h.svc.GetBooking(r.Context(), bookingID)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidRange), errors.Is(err, domain.ErrInvalidTransaction):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrWrongState):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrBookingNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
