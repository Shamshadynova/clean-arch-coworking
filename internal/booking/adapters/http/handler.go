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
	svc    application.BookingService
	logger *slog.Logger
}

// NewRouter builds the HTTP handler tree with routes and middleware.
func NewRouter(svc application.BookingService, logger *slog.Logger) http.Handler {
	h := &BookingHandler{svc: svc, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /bookings", h.CreateBooking)
	mux.HandleFunc("GET /bookings/{id}", h.GetBooking)

	// Apply middleware chain: Recovery -> Logger -> RequestID -> mux
	var handler http.Handler = mux
	handler = RequestID(handler)
	handler = Logger(handler, logger)
	handler = Recovery(handler, logger)

	return handler
}

// CreateBooking handles POST /bookings.
func (h *BookingHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoomID         string    `json:"room_id"`
		UserID         string    `json:"user_id"`
		From           time.Time `json:"from"`
		To             time.Time `json:"to"`
		IdempotencyKey string    `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.RoomID == "" || req.UserID == "" {
		http.Error(w, "room_id and user_id are required", http.StatusBadRequest)
		return
	}
	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		http.Error(w, "invalid room_id", http.StatusBadRequest)
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = uuid.New().String()
	}

	input := application.CreateBookingInput{
		RoomID:         roomID,
		UserID:         userID,
		From:           req.From,
		To:             req.To,
		IdempotencyKey: req.IdempotencyKey,
	}

	id, err := h.svc.CreateBooking(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
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
