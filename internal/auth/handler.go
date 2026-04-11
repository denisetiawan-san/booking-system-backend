package auth

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

// RegisterRoutes mendaftarkan semua route auth ke router chi.
//
// Route design:
//   POST /auth/register  → public, tidak perlu token
//   POST /auth/login     → public, tidak perlu token
//   POST /auth/refresh   → public (token expired tapi butuh refresh)
//   GET  /auth/profile   → protected, perlu token valid
func (h *Handler) RegisterRoutes(r chi.Router, jwtSecret string) {
	r.Route("/auth", func(r chi.Router) {
		// Public routes
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)

		// Protected routes — middleware Auth dijalankan dulu
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtSecret))
			r.Get("/profile", h.Profile)
		})
	})
}

// Register godoc
// POST /api/v1/auth/register
// Body: { "email": "...", "password": "...", "name": "..." }
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	// Decode JSON body ke struct
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "format JSON tidak valid")
		return
	}
	// Validasi field berdasarkan tag validate: di struct
	if err := h.validate.Struct(req); err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}

	result, err := h.service.Register(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusCreated, "registrasi berhasil", result)
}

// Login godoc
// POST /api/v1/auth/login
// Body: { "email": "...", "password": "..." }
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "format JSON tidak valid")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}

	result, err := h.service.Login(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "login berhasil", result)
}

// Refresh godoc
// POST /api/v1/auth/refresh
// Body: { "refresh_token": "..." }
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "format JSON tidak valid")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.WriteValidationError(w, err.Error())
		return
	}

	result, err := h.service.RefreshToken(r.Context(), req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "token berhasil diperbarui", result)
}

// Profile godoc
// GET /api/v1/auth/profile
// Header: Authorization: Bearer <token>
func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	// GetUserID mengambil user ID dari context yang diisi oleh middleware Auth
	userID := middleware.GetUserID(r)

	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "ok", profile)
}
