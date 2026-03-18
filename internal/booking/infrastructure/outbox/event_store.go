package outbox

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/application"
	"github.com/example/coworking/internal/booking/domain"
)
//структура исходящих событий 
//для хранения доменных событий перед отправкой 
type OutboxEvent struct {
	ID        uuid.UUID //id события 
	EventType string //тип события 
	EventData json.RawMessage //данные события в формате json
	Published bool //флаг опубликовано ли событие 
	CreatedAt time.Time //время создния события 
}
//хранилище событий 
type EventStore struct {
	mu     sync.RWMutex //для защиты от паралельного доступа 
	events []OutboxEvent //список событий, которые нужно отправить 
	bus    application.EventBus //для публикации событий в систему 
}

//создаем конструктор NewEventStore новое хранилище событий 
//принимает интерфейс шину событий из пакета interfaces и возвращаем интерфейс хранилище событий 
func NewEventStore(bus application.EventBus) application.EventStore {
	return &EventStore{ //создаем новый EventStore
		events: make([]OutboxEvent, 0), //создаем пустой список событий
		bus:    bus, //сохраняем EventBus
	}
}
//создаем метод сохранения события  SaveEvents для структуры EventStore
//принимает контекст для отмены операции и слайс доменных событий (интерфейс Event из domain), возвращаем ошибку
func (s *EventStore) SaveEvents(ctx context.Context, events []domain.Event) error {
	s.mu.Lock() //блокируем запись
	defer s.mu.Unlock()//снимаем блокировку после того как функция выполнилась 
	
	for _, event := range events { //перебираем все доменные события 
		data, err := json.Marshal(event) //преобразуем событие в json
		if err != nil { //если сериализация не удалась, то возвращаем ошибку
			return err
		}
		//создаем структуру события 
		outboxEvent := OutboxEvent{
			ID:        uuid.New(), 
			EventType: getEventType(event),
			EventData: data,
			Published: false,
			CreatedAt: time.Now(),
		}
		//добавляем событие в очередь. Кладем в список 
		s.events = append(s.events, outboxEvent)
	}
	
	// TODO: replace goroutine-based publish with a reliable polling publisher.
	// Current approach may lose events if the process crashes before publishing.
	go s.publishPendingEvents(ctx) //горутина, которая отправит событие 
	
	return nil
}

//метод publishPendingEvents для отправки события 
func (s *EventStore) publishPendingEvents(ctx context.Context) {
	s.mu.Lock()//блокируем запись. ТОлько одна горутина может писать 
	defer s.mu.Unlock()//снимаем блокировку после выполнения функции 
	//создаем переменную куда положим слайс доменных событий (очередь)
	//для отправки через EventBus 
	var domainEvents []domain.Event
	for i, event := range s.events { //перебираем сохраненные события из OutboxEvent
		if !event.Published { //если событие не опубликовано 
			var domainEvent domain.Event //создаем переменную для доменного события 
			switch event.EventType { //определяем тип события 
			case "RoomBooked": //если событие RoomBooked
				var e domain.RoomBooked //создаем переменную структуры RoomBooked
				//парсим json в структуру 
				if err := json.Unmarshal(event.EventData, &e); err == nil {
					domainEvent = e //записываем событие в domainEvent
				}
			case "BookingConfirmed": //для типа события BookingConfirmed
				var e domain.BookingConfirmed
				if err := json.Unmarshal(event.EventData, &e); err == nil {
					domainEvent = e
				}
			}
			//если произошла ошибка 
			if domainEvent != nil {
				//добавляем событие в список domainEvents
				domainEvents = append(domainEvents, domainEvent)
				//отмечаем событие как опубликованное 
				//чтобы не отправлять событие повторно
				s.events[i].Published = true
			}
		}
	}
	//проверяем,е есть ли события для отправки в другие сервисы 
	if len(domainEvents) > 0 {
		//отправляем собыие в EventBus игнорируя ошибку 
		_ = s.bus.Publish(ctx, domainEvents)
	}
}

func getEventType(event domain.Event) string {
	switch event.(type) {
	case domain.RoomBooked:
		return "RoomBooked"
	case domain.BookingConfirmed:
		return "BookingConfirmed"
	default:
		return "Unknown"
	}
}