package auth

import "time"

// User merepresentasikan satu baris di tabel users.
// Tag db: dipakai sqlx untuk mapping kolom → field struct secara otomatis.
// Tag json: dipakai encoding/json untuk serialisasi ke/dari JSON.
type User struct {
	ID        string    `db:"id"         json:"id"`
	Email     string    `db:"email"      json:"email"`
	Password  string    `db:"password"   json:"-"` // json:"-" = tidak pernah dikirim ke client
	Name      string    `db:"name"       json:"name"`
	Role      string    `db:"role"       json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// ── Request structs ───────────────────────────────────────────────────────────
// Struct ini hanya menerima JSON dari client.
// Dipisah dari User agar field seperti Role tidak bisa diisi langsung oleh client.

type RegisterRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name"     validate:"required,min=2"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ── Response structs ──────────────────────────────────────────────────────────

// TokenResponse dikirim ke client setelah register, login, atau refresh berhasil.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // dalam detik
}

// UserResponse adalah data user yang aman dikirim ke client (tanpa password).
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}
