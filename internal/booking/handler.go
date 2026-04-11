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

// RegisterRoutes mendaftarkan semua route booking.
//
// Route design:
//   POST   /bookings           → buat booking baru (perlu auth + Idempotency-Key)
//   GET    /bookings           → list semua booking milik user ini (perlu auth)
//   GET    /bookings/{id}      → detail satu booking (perlu auth, harus punya)
//   DELETE /bookings/{id}      → cancel booking (perlu auth, harus punya)
//
// Semua booking route memerlukan autentikasi.
// Tidak ada booking yang bisa dibuat atau dilihat tanpa login.
func (h *Handler) RegisterRoutes(r chi.Router, jwtSecret string) {
	r.Route("/bookings", func(r chi.Router) {
		// Semua route booking butuh auth
		r.Use(middleware.Auth(jwtSecret))

		// POST /bookings: perlu tambahan middleware Idempotency
		// r.With() menambahkan middleware hanya untuk route ini
		r.With(middleware.Idempotency).Post("/", h.Create)

		r.Get("/", h.GetMyBookings)
		r.Get("/{id}", h.GetByID)
		r.Delete("/{id}", h.Cancel)
	})
}

// Create godoc
// POST /api/v1/bookings
// Header: Authorization: Bearer <token>
// Header: Idempotency-Key: <uuid>
// Body: CreateBookingRequest
//
// Endpoint ini membuat booking baru. Dua hal penting:
// 1. Memerlukan header Idempotency-Key untuk mencegah double booking
// 2. Proses checkout atomic dengan transaction + locking
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	// Idempotency key dari header — middleware sudah validasi keberadaannya
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

// GetMyBookings godoc
// GET /api/v1/bookings
// Header: Authorization: Bearer <token>
//
// Mengambil semua booking milik user yang sedang login.
// User hanya bisa lihat booking miliknya sendiri, tidak booking orang lain.
func (h *Handler) GetMyBookings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	bookings, err := h.service.GetMyBookings(r.Context(), userID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "ok", bookings)
}

// GetByID godoc
// GET /api/v1/bookings/{id}
// Header: Authorization: Bearer <token>
//
// Mengambil detail satu booking. User hanya bisa lihat booking miliknya.
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

// Cancel godoc
// DELETE /api/v1/bookings/{id}
// Header: Authorization: Bearer <token>
//
// Membatalkan booking. Aturan bisnis:
// - Hanya pemilik booking yang bisa cancel
// - Tidak bisa cancel booking yang sudah cancelled
// - Tidak bisa cancel booking yang check_in-nya sudah lewat
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := middleware.GetUserID(r)

	if err := h.service.Cancel(r.Context(), id, userID); err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "booking berhasil dibatalkan", nil)
}
