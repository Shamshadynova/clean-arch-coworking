package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/example/coworking/internal/booking/domain"
	"github.com/google/uuid"
)


type BookingRepository struct {
	db *sql.DB
}

// нужна только для чтения из базы
type bookingRow struct {
	id             uuid.UUID
	roomID         uuid.UUID
	userID         uuid.UUID
	fromDate       time.Time
	toDate         time.Time
	status         int
	idempotencyKey string
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// Save сохраняет бронирование
// принимает контекст (для отмены операции) и переменную типа с данными о бронировании, возвращаем ошибку
func (r *BookingRepository) Save(ctx context.Context, booking *domain.Booking) error {
	// вызывается метод ExecContext у r.db, указываем, что нам нужна только ошибка
	// ExecContext выполнит sql запрос без возврата строк
	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO bookings (
			id, room_id, user_id, from_date, to_date, status, idempotency_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE
		-- если запись есть, то обновляем. если нет, то добавляем
		SET status = EXCLUDED.status,
			idempotency_key = EXCLUDED.idempotency_key
		`,
		// передаем значения в параметры ($1, $2, $3, $4, $5, $6, $7)
		booking.ID(),
		booking.RoomID(),
		booking.UserID(),
		booking.Slot().From,
		booking.Slot().To,
		booking.Status(),
		booking.IdempotencyKey(),
	)

	if err != nil {
		return fmt.Errorf("repository save booking error: %w", err)
	}

	return nil
}

// FindByID поиск по id
// принимает контекст (для отмены операции и управления запросом) и id типа uuid.UUID
// возвращаем найденное бронирование или ошибку
func (r *BookingRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	// выполняем sql запрос и ожидаем одну строку, так как QueryRowContext
	// запишем в row результат выполнения запроса
	row := r.db.QueryRowContext(
		ctx,
		`
		SELECT id, room_id, user_id, from_date, to_date, status, idempotency_key
		FROM bookings
		WHERE id = $1
		`,
		id, // пишем айди в $1
	)

	// создаем экземпляр структуры bookingRow для хранения данных из БД
	var bi bookingRow

	// считываем данные из БД в структуру, либо ошибка
	err := row.Scan(
		&bi.id,
		&bi.roomID,
		&bi.userID,
		&bi.fromDate,
		&bi.toDate,
		&bi.status,
		&bi.idempotencyKey,
	)

	if err == sql.ErrNoRows {
		// если запись не найдена
		return nil, domain.ErrBookingNotFound
	}

	if err != nil {
		// если произошла другая ошибка
		return nil, fmt.Errorf("scan booking error: %w", err)
	}

	// проверяем даты из БД. Проверяем валидацию slot
	slot, err := domain.NewDateRange(bi.fromDate, bi.toDate)
	if err != nil {
		// если ошибка
		return nil, domain.ErrInvalidRange
	}

	// восстанавливаем сущность Booking из данных
	booking := domain.RestoreBooking(
		bi.id,
		bi.roomID,
		bi.userID,
		slot,
		bi.status,
		bi.idempotencyKey,
	)

	// возвращаем результат запроса
	return booking, nil
}

func (r *BookingRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	row := r.db.QueryRowContext(
		ctx,
		// в row (переменная типа *sql.Row) пишем результат
		`
		SELECT id, room_id, user_id, from_date, to_date, status, idempotency_key
		FROM bookings
		WHERE id = $1
		FOR UPDATE
		-- блокировка строки, пока не закончится транзакция
		`,
		id, // передаем в $1
	)

	var bi bookingRow

	err := row.Scan(
		&bi.id,
		&bi.roomID,
		&bi.userID,
		&bi.fromDate,
		&bi.toDate,
		&bi.status,
		&bi.idempotencyKey,
	)

	if err == sql.ErrNoRows {
		// если запись не найдена
		return nil, domain.ErrBookingNotFound
	}

	if err != nil {
		// если произошла другая ошибка
		return nil, fmt.Errorf("scan booking error: %w", err)
	}

	// проверяем даты из БД. Проверяем валидацию slot
	slot, err := domain.NewDateRange(bi.fromDate, bi.toDate)
	if err != nil {
		// если ошибка
		return nil, domain.ErrInvalidRange
	}

	// восстанавливаем сущность Booking из данных
	booking := domain.RestoreBooking(
		bi.id,
		bi.roomID,
		bi.userID,
		slot,
		bi.status,
		bi.idempotencyKey,
	)

	// возвращаем результат запроса
	return booking, nil
}

// FindByIdempotencyKey поиск по ключу
// не был ли уже выполнен такой же POST запрос
func (r *BookingRepository) FindByIdempotencyKey(ctx context.Context, key string) (*domain.Booking, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
		SELECT id, room_id, user_id, from_date, to_date, status, idempotency_key
		FROM bookings
		WHERE idempotency_key = $1
		`,
		key, // передаем в $1
	)

	var bi bookingRow

	err := row.Scan(
		&bi.id,
		&bi.roomID,
		&bi.userID,
		&bi.fromDate,
		&bi.toDate,
		&bi.status,
		&bi.idempotencyKey,
	)

	if err == sql.ErrNoRows {
		// если запись не найдена
		return nil, domain.ErrBookingNotFound
	}

	if err != nil {
		// если произошла другая ошибка
		return nil, fmt.Errorf("scan booking error: %w", err)
	}

	// проверяем даты из БД. Проверяем валидацию slot
	slot, err := domain.NewDateRange(bi.fromDate, bi.toDate)
	if err != nil {
		// если ошибка
		return nil, domain.ErrInvalidRange
	}

	// восстанавливаем сущность Booking из данных
	booking := domain.RestoreBooking(
		bi.id,
		bi.roomID,
		bi.userID,
		slot,
		bi.status,
		bi.idempotencyKey,
	)

	// возвращаем результат запроса
	return booking, nil
}

// FindAllByRoomID поиск всех бронирований по roomID
func (r *BookingRepository) FindAllByRoomID(ctx context.Context, roomID uuid.UUID) ([]*domain.Booking, error) {
	rows, err := r.db.QueryContext(
		ctx,
		// возвращаем несколько строк или ошибку
		`
		SELECT id, room_id, user_id, from_date, to_date, status, idempotency_key
		FROM bookings
		WHERE room_id = $1
		`,
		roomID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close() // освобождаем ресурс. Нужно, когда используем много строк

	var result []*domain.Booking

	for rows.Next() {
		// переход к след. строке результата
		var bi bookingRow

		err := rows.Scan(
			&bi.id,
			&bi.roomID,
			&bi.userID,
			&bi.fromDate,
			&bi.toDate,
			&bi.status,
			&bi.idempotencyKey,
		)

		if err != nil {
			return nil, err
		}

		slot, err := domain.NewDateRange(bi.fromDate, bi.toDate)
		if err != nil {
			return nil, err
		}

		booking := domain.RestoreBooking(
			bi.id,
			bi.roomID,
			bi.userID,
			slot,
			bi.status,
			bi.idempotencyKey,
		)

		// добавляем бронирование в результат
		result = append(result, booking)
	}

	// проверяем ошибки после цикла
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}















