package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"booking-system/pkg/apperror"
	"booking-system/pkg/response"
)

// contextKey adalah tipe khusus untuk key di context.
// Menggunakan tipe sendiri mencegah key collision dengan library lain
// yang mungkin juga menyimpan data di context dengan key string biasa.
type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role"
)

// Logger adalah middleware yang mencatat setiap HTTP request.
// Dipasang sebagai global middleware — berjalan untuk SETIAP request.
//
// Output contoh:
//   10:30:15 INF request method=POST path=/api/v1/bookings status=201 duration=45ms ip=127.0.0.1
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Bungkus ResponseWriter untuk menangkap status code
		wrapped := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapped.statusCode).
			Dur("duration", time.Since(start)).
			Str("ip", r.RemoteAddr).
			Msg("request")
	})
}

// wrappedWriter membungkus http.ResponseWriter agar bisa menangkap status code
// yang di-set oleh handler di downstream.
type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Auth adalah middleware yang memvalidasi JWT Bearer token dari header Authorization.
//
// Kalau token valid: user_id dan role disimpan ke context, request dilanjutkan.
// Kalau token tidak ada atau tidak valid: request langsung ditolak dengan 401.
//
// Cara pakai di router (via chi):
//   r.Group(func(r chi.Router) {
//       r.Use(middleware.Auth(jwtSecret))
//       r.Post("/bookings", handler.Create)
//   })
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.WriteError(w, apperror.ErrUnauthorized)
				return
			}

			// Format yang diharapkan: "Bearer <token>"
			// SplitN dengan N=2 agar token yang mengandung spasi tidak terpotong
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.WriteError(w, apperror.ErrInvalidToken)
				return
			}

			userID, role, err := validateToken(parts[1], jwtSecret)
			if err != nil {
				response.WriteError(w, apperror.ErrInvalidToken)
				return
			}

			// Simpan user info ke context — bisa diambil di handler dengan GetUserID()
			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyRole, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole adalah middleware yang memvalidasi role user.
// HARUS dipakai SETELAH middleware Auth karena butuh data dari context yang diisi Auth.
//
// Cara pakai:
//   r.Group(func(r chi.Router) {
//       r.Use(middleware.Auth(jwtSecret))
//       r.Use(middleware.RequireRole("admin"))
//       r.Post("/rooms", handler.Create)  // hanya admin
//   })
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := r.Context().Value(ContextKeyRole).(string)
			if !ok || userRole != role {
				response.WriteError(w, apperror.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID mengambil user ID dari context request.
// Hanya valid di handler yang sudah dilindungi middleware Auth.
func GetUserID(r *http.Request) string {
	userID, _ := r.Context().Value(ContextKeyUserID).(string)
	return userID
}

// GetRole mengambil role dari context request.
func GetRole(r *http.Request) string {
	role, _ := r.Context().Value(ContextKeyRole).(string)
	return role
}
