package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/application"
	"github.com/example/coworking/internal/booking/domain"
	"github.com/example/coworking/internal/booking/infrastructure/postgres"
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
	db postgres.DB //для сохранения событий в базу данных 
	bus    application.EventBus //для публикации событий в систему 
}

//создаем конструктор NewEventStore новое хранилище событий 
//принимает интерфейс шину событий из пакета interfaces и возвращаем интерфейс хранилище событий 
func NewEventStore(db postgres.DB, bus application.EventBus) application.EventStore {
	return &EventStore{ //создаем новый EventStore
		db: db, //сохраняем базу данных
		bus:    bus, //сохраняем EventBus
	}
}
//создаем метод сохранения события  SaveEvents для структуры EventStore
//принимает контекст для отмены операции и слайс доменных событий (интерфейс Event из domain), возвращаем ошибку
func (s *EventStore) SaveEvents(ctx context.Context, events []domain.Event) error {
	
	for _, event := range events { //перебираем все доменные события 
		data, err := json.Marshal(event) //преобразуем событие в json
		if err != nil { //если сериализация не удалась, то возвращаем ошибку
			return err
		}
		_, err = s.db.ExecContext(ctx, 
			`
			INSERT INTO outbox_events (
				id, event_type, event_data, published, created_at
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
			uuid.New(), //генерируем новый id для события
			getEventType(event), //определяем тип события
			data, //данные события в формате json
			false, //событие еще не опубликовано
			time.Now(), //время создания события
		)

		if err != nil { //если сохранение в базу данных не удалось, то возвращаем ошибку
			return err
		}
	}
	go s.publishPendingEvents(ctx)
	return nil
}

//метод publishPendingEvents для отправки события (воркер)
func (s *EventStore) publishPendingEvents(ctx context.Context) {
	//делаем запрос к бд для получаения событий, которые еще не отправили published = false
	rows, err := s.db.QueryContext(ctx,
		`
			SELECT id, event_type, event_data
			FROM outbox_events
			WHERE published = false
		`,
	)
	if err != nil { //если запрос к базе данных не удался, то завершаем работу
		return
	}

	defer rows.Close()

	for rows.Next() { //делаем перебор по каждому событию 
    	var event OutboxEvent //для хранения события из бд 
 //сканируем данные  из бд в структуру 
    if err := rows.Scan(&event.ID, &event.EventType, &event.EventData); err != nil {
        continue //если ошибка, то пропускаем, так как нам нужно отправить как можно больше событий 
    }

    var domainEvent domain.Event //для хранения корректного события 

    switch event.EventType { //если событие 
    case "RoomBooked": //типа RoomBooked
        var e domain.RoomBooked //то нужно распарсить в структуру го
        if err := json.Unmarshal(event.EventData, &e); err == nil {
            domainEvent = e
        }

    case "BookingConfirmed":
        var e domain.BookingConfirmed
        if err := json.Unmarshal(event.EventData, &e); err == nil {
            domainEvent = e
        }
    }

    if domainEvent == nil {
        continue
    }

    // отправляем одно событие
    if err := s.bus.Publish(ctx, []domain.Event{domainEvent}); err != nil {
        continue
    }

    // помечаем true
    _, err := s.db.ExecContext(ctx, `
        UPDATE outbox_events
        SET published = true
        WHERE id = $1
    `, event.ID)

    if err != nil {
        continue
    }
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