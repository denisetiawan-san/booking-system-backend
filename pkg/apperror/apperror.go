package apperror

import "net/http"

// AppError adalah tipe error khusus yang membawa HTTP status code-nya sendiri.
// Dengan ini, handler tidak perlu tahu "error ini → status berapa" — sudah built-in.
//
// Cara pakai di handler:
//   result, err := service.DoSomething(ctx, req)
//   if err != nil {
//       response.WriteError(w, err)  // otomatis pakai status dari AppError
//       return
//   }
type AppError struct {
	Code    int    // HTTP status code yang akan dikirim ke client
	Message string // Pesan yang akan ditampilkan ke client
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// ── Auth errors ───────────────────────────────────────────────────────────────
var (
	ErrInvalidCredentials = New(http.StatusUnauthorized, "email atau password salah")
	ErrEmailAlreadyExists = New(http.StatusConflict, "email sudah terdaftar")
	ErrUnauthorized       = New(http.StatusUnauthorized, "unauthorized: token diperlukan")
	ErrForbidden          = New(http.StatusForbidden, "akses ditolak: role tidak mencukupi")
	ErrInvalidToken       = New(http.StatusUnauthorized, "token tidak valid atau sudah expired")
)

// ── Resource errors ───────────────────────────────────────────────────────────
var (
	ErrNotFound = New(http.StatusNotFound, "data tidak ditemukan")
)

// ── Booking-specific errors ───────────────────────────────────────────────────
var (
	// ErrRoomNotAvailable adalah error paling penting di project ini.
	// Muncul ketika tanggal yang diminta overlap dengan booking yang sudah ada.
	ErrRoomNotAvailable = New(http.StatusConflict,
		"kamar tidak tersedia untuk tanggal yang dipilih, sudah ada booking yang overlap")

	// ErrInvalidDateRange muncul ketika check_out <= check_in
	ErrInvalidDateRange = New(http.StatusBadRequest,
		"tanggal check-out harus setelah tanggal check-in")

	// ErrCheckInPast muncul ketika user mencoba booking tanggal yang sudah lewat
	ErrCheckInPast = New(http.StatusBadRequest,
		"tanggal check-in tidak boleh di masa lalu")

	// ErrCannotCancelConfirmed muncul ketika user mencoba cancel booking yang
	// tanggal check-in-nya sudah lewat (tamu sudah check in)
	ErrCannotCancelPastBooking = New(http.StatusConflict,
		"tidak bisa membatalkan booking yang tanggal check-in sudah lewat")

	// ErrBookingNotOwned muncul ketika user mencoba cancel booking milik orang lain
	ErrBookingNotOwned = New(http.StatusForbidden,
		"kamu tidak memiliki akses ke booking ini")

	// ErrAlreadyCancelled muncul ketika user mencoba cancel booking yang sudah cancelled
	ErrAlreadyCancelled = New(http.StatusConflict,
		"booking ini sudah dibatalkan sebelumnya")

	// ErrDuplicateRequest muncul ketika idempotency key sudah pernah dipakai
	ErrDuplicateRequest = New(http.StatusConflict,
		"request duplikat: booking dengan idempotency key ini sudah pernah dibuat")
)

// ── Validation errors ─────────────────────────────────────────────────────────
var (
	ErrValidation = New(http.StatusBadRequest, "validasi request gagal")
)
