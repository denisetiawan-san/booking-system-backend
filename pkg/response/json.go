package response

import (
	"encoding/json"
	"net/http"

	"booking-system/pkg/apperror"
)

// JSONResponse adalah format standar SEMUA response dari API ini.
// Konsistensi format memudahkan frontend/client memproses response.
//
// Sukses: { "success": true, "message": "...", "data": {...} }
// Error:  { "success": false, "error": "..." }
type JSONResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// WriteSuccess menulis response sukses ke ResponseWriter.
// statusCode: HTTP status (200, 201, dll)
// message: pesan deskriptif untuk client
// data: payload utama (bisa struct, slice, map, atau nil)
func WriteSuccess(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	write(w, statusCode, JSONResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// WriteError menulis response error ke ResponseWriter.
// Secara otomatis menentukan status code dari tipe error:
// - *AppError → pakai status code yang sudah terdefinisi
// - error lain → 500 Internal Server Error (tidak expose detail ke client)
func WriteError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*apperror.AppError); ok {
		write(w, appErr.Code, JSONResponse{
			Success: false,
			Error:   appErr.Message,
		})
		return
	}
	// Unexpected error (bug, DB down, dll) — jangan expose detail ke client
	write(w, http.StatusInternalServerError, JSONResponse{
		Success: false,
		Error:   "terjadi kesalahan pada server, silakan coba lagi",
	})
}

// WriteValidationError menulis error validasi (400) ke ResponseWriter.
func WriteValidationError(w http.ResponseWriter, message string) {
	write(w, http.StatusBadRequest, JSONResponse{
		Success: false,
		Error:   message,
	})
}

// write adalah helper internal yang menangani penulisan JSON ke ResponseWriter.
func write(w http.ResponseWriter, statusCode int, body JSONResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}
