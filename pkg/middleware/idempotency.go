package middleware

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"booking-system/pkg/response"
)

// Idempotency adalah middleware yang memvalidasi keberadaan header Idempotency-Key.
//
// ═══════════════════════════════════════════════════════════════
// KENAPA BOOKING BUTUH IDEMPOTENCY?
// ═══════════════════════════════════════════════════════════════
//
// Skenario masalah:
//   User klik "Pesan Kamar" → request terkirim → koneksi timeout di tengah jalan
//   User tidak tahu apakah booking berhasil atau tidak
//   User klik "Pesan Kamar" lagi → request kedua terkirim
//   Tanpa idempotency: dua booking terbuat untuk kamar dan tanggal yang sama
//
// Dengan idempotency:
//   Request pertama  → booking dibuat, key disimpan
//   Request kedua    → key ditemukan → return booking yang sama, tidak buat baru
//
// Middleware ini hanya memvalidasi bahwa header ADA dan formatnya benar.
// Logika cek ke database (apakah key sudah pernah dipakai) ada di booking service.
func Idempotency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")

		if key == "" {
			// Header tidak ada → tolak request sebelum sampai ke handler
			log.Warn().
				Str("path", r.URL.Path).
				Msg("request ditolak: header Idempotency-Key tidak ada")

			response.WriteValidationError(w,
				"header Idempotency-Key wajib diisi. "+
					"Generate UUID unik di client sebelum setiap request booking.")
			return
		}

		// Batasi panjang key untuk mencegah abuse
		if len(key) > 64 {
			response.WriteValidationError(w, "Idempotency-Key terlalu panjang (maksimal 64 karakter)")
			return
		}

		// Key valid, lanjutkan ke handler
		next.ServeHTTP(w, r)
	})
}
