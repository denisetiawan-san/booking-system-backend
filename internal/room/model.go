package room

import "time"

// Room merepresentasikan satu kamar/ruangan yang bisa dipesan.
type Room struct {
	ID             string    `db:"id"              json:"id"`
	Name           string    `db:"name"            json:"name"`
	Description    string    `db:"description"     json:"description"`
	Capacity       int       `db:"capacity"        json:"capacity"`
	PricePerNight  float64   `db:"price_per_night" json:"price_per_night"`
	IsActive       bool      `db:"is_active"       json:"is_active"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"      json:"updated_at"`
}

// ── Request structs ───────────────────────────────────────────────────────────

// CreateRoomRequest dipakai admin saat membuat kamar baru.
type CreateRoomRequest struct {
	Name          string  `json:"name"           validate:"required,min=2"`
	Description   string  `json:"description"`
	Capacity      int     `json:"capacity"       validate:"required,min=1"`
	PricePerNight float64 `json:"price_per_night" validate:"required,gt=0"`
}

// UpdateRoomRequest dipakai admin untuk partial update kamar.
// Semua field pointer agar bisa bedakan "tidak dikirim" vs "dikirim dengan nilai kosong/0".
type UpdateRoomRequest struct {
	Name          *string  `json:"name"           validate:"omitempty,min=2"`
	Description   *string  `json:"description"`
	Capacity      *int     `json:"capacity"       validate:"omitempty,min=1"`
	PricePerNight *float64 `json:"price_per_night" validate:"omitempty,gt=0"`
	IsActive      *bool    `json:"is_active"`
}

// SearchAvailabilityRequest dipakai user untuk mencari kamar yang tersedia.
type SearchAvailabilityRequest struct {
	CheckIn   string `json:"check_in"  validate:"required"` // format: 2006-01-02
	CheckOut  string `json:"check_out" validate:"required"` // format: 2006-01-02
	Capacity  int    `json:"capacity"`                       // filter kapasitas minimal
}

// ── Response structs ──────────────────────────────────────────────────────────

// RoomAvailabilityResponse adalah response saat pencarian kamar tersedia.
// Menyertakan kalkulasi total harga untuk rentang tanggal yang diminta.
type RoomAvailabilityResponse struct {
	Room       Room    `json:"room"`
	TotalNight int     `json:"total_night"`  // berapa malam
	TotalPrice float64 `json:"total_price"`  // total harga = harga_per_malam * jumlah_malam
}

// ListRoomResponse membungkus list room dengan info pagination.
type ListRoomResponse struct {
	Rooms []Room `json:"rooms"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
