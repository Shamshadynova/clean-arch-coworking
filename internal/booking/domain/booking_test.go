package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/example/coworking/internal/booking/domain"
)

func validSlot(t *testing.T) domain.DateRange {
	t.Helper()
	from := time.Now().Add(24 * time.Hour)
	to := from.Add(2 * time.Hour)
	slot, err := domain.NewDateRange(from, to)
	if err != nil {
		t.Fatalf("unexpected error creating date range: %v", err)
	}
	return slot
}

func TestNewBooking_Success(t *testing.T) {
	slot := validSlot(t)
	price := domain.NewMoney(500, "USD")

	booking, err := domain.NewBooking(uuid.New(), uuid.New(), slot, price)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.ID() == uuid.Nil {
		t.Error("expected non-nil booking ID")
	}
	if booking.Status() != domain.Pending {
		t.Errorf("expected status Pending, got %v", booking.Status())
	}

	events := booking.PullEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if _, ok := events[0].(domain.RoomBooked); !ok {
		t.Errorf("expected RoomBooked event, got %T", events[0])
	}
}

func TestNewBooking_ZeroSlot(t *testing.T) {
	price := domain.NewMoney(100, "USD")
	_, err := domain.NewBooking(uuid.New(), uuid.New(), domain.DateRange{}, price)
	if err != domain.ErrInvalidRange {
		t.Errorf("expected ErrInvalidRange, got %v", err)
	}
}

func TestConfirmPayment_Success(t *testing.T) {
	slot := validSlot(t)
	booking, _ := domain.NewBooking(uuid.New(), uuid.New(), slot, domain.NewMoney(100, "USD"))
	_ = booking.PullEvents() // clear creation events

	err := booking.ConfirmPayment("tx-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if booking.Status() != domain.Paid {
		t.Errorf("expected status Paid, got %v", booking.Status())
	}

	events := booking.PullEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if ev, ok := events[0].(domain.BookingConfirmed); !ok {
		t.Errorf("expected BookingConfirmed event, got %T", events[0])
	} else if ev.TxID != "tx-123" {
		t.Errorf("expected TxID tx-123, got %s", ev.TxID)
	}
}

func TestConfirmPayment_EmptyTxID(t *testing.T) {
	slot := validSlot(t)
	booking, _ := domain.NewBooking(uuid.New(), uuid.New(), slot, domain.NewMoney(100, "USD"))

	err := booking.ConfirmPayment("   ")
	if err != domain.ErrInvalidTransaction {
		t.Errorf("expected ErrInvalidTransaction, got %v", err)
	}
}

func TestConfirmPayment_AlreadyPaid(t *testing.T) {
	slot := validSlot(t)
	booking, _ := domain.NewBooking(uuid.New(), uuid.New(), slot, domain.NewMoney(100, "USD"))
	_ = booking.ConfirmPayment("tx-001")

	err := booking.ConfirmPayment("tx-002")
	if err != domain.ErrWrongState {
		t.Errorf("expected ErrWrongState, got %v", err)
	}
}
