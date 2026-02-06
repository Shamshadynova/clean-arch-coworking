package domain

import (
	"strings"

	"github.com/google/uuid"
)

type BookingStatus int

const (
	Pending BookingStatus = iota
	Paid
	Cancelled
)

type Booking struct {
	id              uuid.UUID
	roomID          uuid.UUID
	userID          uuid.UUID
	slot            DateRange
	price           Money
	status          BookingStatus
	events          []Event
	idempotencyKey  string
	transactionID   string
}

func NewBooking(roomID, userID uuid.UUID, slot DateRange, price Money) (*Booking, error) {
	if slot.IsZero() {
		return nil, ErrInvalidRange
	}
	b := &Booking{
		id:     uuid.New(),
		roomID: roomID,
		userID: userID,
		slot:   slot,
		price:  price,
		status: Pending,
	}
	b.raise(RoomBooked{BookingID: b.id.String(), RoomID: roomID.String(), UserID: userID.String()})
	return b, nil
}

func (b *Booking) ID() uuid.UUID         { return b.id }
func (b *Booking) RoomID() uuid.UUID      { return b.roomID }
func (b *Booking) UserID() uuid.UUID      { return b.userID }
func (b *Booking) Slot() DateRange        { return b.slot }
func (b *Booking) Price() Money           { return b.price }
func (b *Booking) Status() BookingStatus  { return b.status }
func (b *Booking) TransactionID() string  { return b.transactionID }

func (b *Booking) ConfirmPayment(txID string) error {
	if strings.TrimSpace(txID) == "" {
		return ErrInvalidTransaction
	}
	if b.status != Pending {
		return ErrWrongState
	}
	b.status = Paid
	b.transactionID = txID
	b.raise(BookingConfirmed{BookingID: b.id.String(), TxID: txID})
	return nil
}

func (b *Booking) SetIdempotencyKey(key string) {
	b.idempotencyKey = key
}

func (b *Booking) IdempotencyKey() string {
	return b.idempotencyKey
}

func (b *Booking) IsPaymentConfirmed(txID string) bool {
	return b.status == Paid && b.transactionID == txID
}

func (b *Booking) PullEvents() []Event {
	ev := b.events
	b.events = nil
	return ev
}

func (b *Booking) raise(e Event) {
	b.events = append(b.events, e)
}
