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

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

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

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.WriteError(w, apperror.ErrUnauthorized)
				return
			}

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

			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			ctx = context.WithValue(ctx, ContextKeyRole, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

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

func GetUserID(r *http.Request) string {
	userID, _ := r.Context().Value(ContextKeyUserID).(string)
	return userID
}

func GetRole(r *http.Request) string {
	role, _ := r.Context().Value(ContextKeyRole).(string)
	return role
}
