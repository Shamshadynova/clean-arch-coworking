package domain

import "errors"

var (
	ErrInvalidRange       = errors.New("invalid date range")
	ErrRoomNotAvailable   = errors.New("room is unavailable")
	ErrWrongState         = errors.New("booking in wrong state")
	ErrBookingNotFound    = errors.New("booking not found")
	ErrInvalidTransaction = errors.New("invalid payment transaction")
	ErrBookingAlreadyPaid = errors.New("cannot cancel a booking that is already paid")
	ErrAlreadyCancelled   = errors.New("booking is already cancelled")
)
