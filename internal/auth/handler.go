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

func (h *Handler) RegisterRoutes(r chi.Router, jwtSecret string) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtSecret))
			r.Get("/profile", h.Profile)
		})
	})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteValidationError(w, "format JSON tidak valid")
		return
	}
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

func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	profile, err := h.service.GetProfile(r.Context(), userID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteSuccess(w, http.StatusOK, "ok", profile)
}
