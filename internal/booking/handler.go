package booking

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"booking-system/pkg/middleware"
	"booking-system/pkg/response"
)

type Handler struct {
	service  Service
	validate *validator.Validate
}

func NewHandler(service Service, validate *validator.Validate) *Handler {
	return &Handler{service: service, validate: validate}
}

func (h *Handler) RegisterRoutes(r chi.Router, jwtSecret string) {
	r.Route("/bookings", func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		r.With(middleware.Idempotency).Post("/", h.Create)

		r.Get("/", h.GetMyBookings)
		r.Get("/{id}", h.GetByID)
		r.Delete("/{id}", h.Cancel)
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	idempotencyKey := r.Header.Get("Idempotency-Key")

	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "format JSON tidak valid")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}

	booking, err := h.service.Create(r.Context(), userID, idempotencyKey, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusCreated, "booking berhasil dibuat", booking)
}

func (h *Handler) GetMyBookings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	bookings, err := h.service.GetMyBookings(r.Context(), userID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "ok", bookings)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r)

	booking, err := h.service.GetByID(r.Context(), id, userID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "ok", booking)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r)

	if err := h.service.Cancel(r.Context(), id, userID); err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "booking berhasil dibatalkan", nil)
}
