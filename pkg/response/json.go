package response

import (
	"encoding/json"
	"net/http"

	"booking-system/pkg/apperror"
)

type JSONResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func WriteSuccess(w http.ResponseWriter, statusCode int, message string, data interface{}) {
	write(w, statusCode, JSONResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func WriteError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*apperror.AppError); ok {
		write(w, appErr.Code, JSONResponse{
			Success: false,
			Error:   appErr.Message,
		})
		return
	}
	write(w, http.StatusInternalServerError, JSONResponse{
		Success: false,
		Error:   "terjadi kesalahan pada server, silakan coba lagi",
	})
}

func WriteValidationError(w http.ResponseWriter, message string) {
	write(w, http.StatusBadRequest, JSONResponse{
		Success: false,
		Error:   message,
	})
}

func write(w http.ResponseWriter, statusCode int, body JSONResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}
