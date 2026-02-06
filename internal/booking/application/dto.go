package application

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type CreateBookingInput struct {
	RoomID         uuid.UUID
	UserID         uuid.UUID
	From           time.Time
	To             time.Time
	IdempotencyKey string
}

func (i CreateBookingInput) Validate() error {
	if i.RoomID == uuid.Nil {
		return errors.New("room ID is required")
	}
	if i.UserID == uuid.Nil {
		return errors.New("user ID is required")
	}
	if i.From.IsZero() || i.To.IsZero() {
		return errors.New("booking dates are required")
	}
	if i.From.After(i.To) {
		return errors.New("invalid date range: from date must be before to date")
	}
	if i.From.Before(time.Now().Add(-time.Hour)) {
		return errors.New("cannot book in the past")
	}
	if i.IdempotencyKey == "" {
		return errors.New("idempotency key is required")
	}
	return nil
}

type ConfirmPaymentInput struct {
	BookingID      uuid.UUID
	TransactionID  string
	IdempotencyKey string
}

func (i ConfirmPaymentInput) Validate() error {
	if i.BookingID == uuid.Nil {
		return errors.New("booking ID is required")
	}
	if i.TransactionID == "" {
		return errors.New("transaction ID is required")
	}
	if i.IdempotencyKey == "" {
		return errors.New("idempotency key is required")
	}
	return nil
}