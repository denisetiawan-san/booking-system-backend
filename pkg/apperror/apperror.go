package apperror

import "net/http"

type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

var (
	ErrInvalidCredentials = New(http.StatusUnauthorized, "email atau password salah")
	ErrEmailAlreadyExists = New(http.StatusConflict, "email sudah terdaftar")
	ErrUnauthorized       = New(http.StatusUnauthorized, "unauthorized: token diperlukan")
	ErrForbidden          = New(http.StatusForbidden, "akses ditolak: role tidak mencukupi")
	ErrInvalidToken       = New(http.StatusUnauthorized, "token tidak valid atau sudah expired")
)

var (
	ErrNotFound = New(http.StatusNotFound, "data tidak ditemukan")
)

var (
	ErrRoomNotAvailable = New(http.StatusConflict,
		"kamar tidak tersedia untuk tanggal yang dipilih, sudah ada booking yang overlap")

	ErrInvalidDateRange = New(http.StatusBadRequest,
		"tanggal check-out harus setelah tanggal check-in")

	ErrCheckInPast = New(http.StatusBadRequest,
		"tanggal check-in tidak boleh di masa lalu")

	ErrCannotCancelPastBooking = New(http.StatusConflict,
		"tidak bisa membatalkan booking yang tanggal check-in sudah lewat")

	ErrBookingNotOwned = New(http.StatusForbidden,
		"kamu tidak memiliki akses ke booking ini")

	ErrAlreadyCancelled = New(http.StatusConflict,
		"booking ini sudah dibatalkan sebelumnya")

	ErrDuplicateRequest = New(http.StatusConflict,
		"request duplikat: booking dengan idempotency key ini sudah pernah dibuat")
)

var (
	ErrValidation = New(http.StatusBadRequest, "validasi request gagal")
)
