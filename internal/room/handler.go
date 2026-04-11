package room

import (
	"encoding/json"
	"net/http"
	"strconv"

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

// RegisterRoutes mendaftarkan semua route room.
//
// Route design:
//   GET  /rooms                   → list semua kamar aktif (PUBLIC)
//   GET  /rooms/available         → cari kamar tersedia untuk tanggal tertentu (PUBLIC)
//   GET  /rooms/{id}              → detail satu kamar (PUBLIC)
//   POST /rooms                   → buat kamar baru (ADMIN ONLY)
//   PATCH /rooms/{id}             → update kamar (ADMIN ONLY)
func (h *Handler) RegisterRoutes(r chi.Router, jwtSecret string) {
	r.Route("/rooms", func(r chi.Router) {
		// Public routes — siapa saja bisa lihat kamar
		r.Get("/", h.List)
		r.Get("/available", h.SearchAvailable)
		r.Get("/{id}", h.GetByID)

		// Admin only routes — hanya admin yang bisa kelola kamar
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtSecret))
			r.Use(middleware.RequireRole("admin"))
			r.Post("/", h.Create)
			r.Patch("/{id}", h.Update)
		})
	})
}

// List godoc
// GET /api/v1/rooms?page=1&limit=10
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	result, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "ok", result)
}

// SearchAvailable godoc
// GET /api/v1/rooms/available?check_in=2026-04-10&check_out=2026-04-12&capacity=2
//
// Endpoint ini menjawab pertanyaan: "Kamar mana yang bisa saya pesan untuk tanggal ini?"
// Response menyertakan kalkulasi total harga untuk rentang tanggal yang diminta.
func (h *Handler) SearchAvailable(w http.ResponseWriter, r *http.Request) {
	req := SearchAvailabilityRequest{
		CheckIn:  r.URL.Query().Get("check_in"),
		CheckOut: r.URL.Query().Get("check_out"),
	}
	if cap := r.URL.Query().Get("capacity"); cap != "" {
		req.Capacity, _ = strconv.Atoi(cap)
	}

	if err := h.validate.Struct(req); err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}

	results, err := h.service.SearchAvailable(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "ok", results)
}

// GetByID godoc
// GET /api/v1/rooms/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	room, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteSuccess(w, http.StatusOK, "ok", room)
}

// Create godoc
// POST /api/v1/rooms
// Header: Authorization: Bearer <admin_token>
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "format JSON tidak valid")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}

	room, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusCreated, "kamar berhasil dibuat", room)
}

// Update godoc
// PATCH /api/v1/rooms/{id}
// Header: Authorization: Bearer <admin_token>
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req UpdateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "format JSON tidak valid")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}

	room, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "kamar berhasil diupdate", room)
}
