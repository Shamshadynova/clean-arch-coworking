package domain_test

import (
	"testing"

	"github.com/example/coworking/internal/booking/domain"
)

func TestNewMoney(t *testing.T) {
	m := domain.NewMoney(1500, "EUR")
	if m.Amount != 1500 {
		t.Errorf("expected amount 1500, got %d", m.Amount)
	}
	if m.Currency != "EUR" {
		t.Errorf("expected currency EUR, got %s", m.Currency)
	}
}

func TestMoney_Add_SameCurrency(t *testing.T) {
	a := domain.NewMoney(100, "USD")
	b := domain.NewMoney(250, "USD")

	result := a.Add(b)
	if result.Amount != 350 {
		t.Errorf("expected 350, got %d", result.Amount)
	}
	if result.Currency != "USD" {
		t.Errorf("expected USD, got %s", result.Currency)
	}
}

func TestMoney_Add_CurrencyMismatch_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on currency mismatch, got none")
		}
	}()

	a := domain.NewMoney(100, "USD")
	b := domain.NewMoney(200, "EUR")
	a.Add(b)
}
