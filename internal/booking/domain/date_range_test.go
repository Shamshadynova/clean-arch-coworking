package domain_test

import (
	"testing"
	"time"

	"github.com/example/coworking/internal/booking/domain"
)

func TestNewDateRange_Valid(t *testing.T) {
	from := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	dr, err := domain.NewDateRange(from, to)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if dr.From != from || dr.To != to {
		t.Error("date range fields do not match inputs")
	}
}

func TestNewDateRange_Reversed(t *testing.T) {
	from := time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	_, err := domain.NewDateRange(from, to)
	if err != domain.ErrInvalidRange {
		t.Errorf("expected ErrInvalidRange, got %v", err)
	}
}

func TestNewDateRange_ZeroFrom(t *testing.T) {
	_, err := domain.NewDateRange(time.Time{}, time.Now())
	if err != domain.ErrInvalidRange {
		t.Errorf("expected ErrInvalidRange, got %v", err)
	}
}

func TestIsOverlapping_True(t *testing.T) {
	a, _ := domain.NewDateRange(
		time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC),
	)
	b, _ := domain.NewDateRange(
		time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 1, 16, 0, 0, 0, time.UTC),
	)

	if !a.IsOverlapping(b) {
		t.Error("expected ranges to overlap")
	}
}

func TestIsOverlapping_False_Adjacent(t *testing.T) {
	a, _ := domain.NewDateRange(
		time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
	)
	b, _ := domain.NewDateRange(
		time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC),
	)

	if a.IsOverlapping(b) {
		t.Error("expected adjacent ranges not to overlap")
	}
}
