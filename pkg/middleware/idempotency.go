package middleware

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"booking-system/pkg/response"
)

func Idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")

		if key == "" {
			log.Warn().
				Str("path", r.URL.Path).
				Msg("request ditolak: header Idempotency-Key tidak ada")

			response.WriteValidationError(w,
				"header Idempotency-Key wajib diisi. "+
					"Generate UUID unik di client sebelum setiap request booking.")
			return
		}

		if len(key) > 64 {
			response.WriteValidationError(w, "Idempotency-Key terlalu panjang (maksimal 64 karakter)")
			return
		}

		next.ServeHTTP(w, r)
	})
}
