package booking

import "time"

// Booking merepresentasikan satu pemesanan kamar.
// Setiap baris adalah kontrak: user X memesan room Y dari tanggal A sampai B.
type Booking struct {
	ID             string     `db:"id"              json:"id"`
	UserID         string     `db:"user_id"         json:"user_id"`
	RoomID         string     `db:"room_id"         json:"room_id"`
	CheckIn        time.Time  `db:"check_in"        json:"check_in"`
	CheckOut       time.Time  `db:"check_out"       json:"check_out"`
	Status         string     `db:"status"          json:"status"`
	TotalPrice     float64    `db:"total_price"     json:"total_price"`
	Notes          string     `db:"notes"           json:"notes"`
	// IdempotencyKey mencegah double booking akibat double click atau retry request.
	// UNIQUE constraint di database memastikan satu key hanya menghasilkan satu booking.
	IdempotencyKey string     `db:"idempotency_key" json:"idempotency_key"`
	CreatedAt      time.Time  `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"      json:"updated_at"`
}

// ── Request structs ───────────────────────────────────────────────────────────

// CreateBookingRequest adalah body JSON yang dikirim client saat memesan kamar.
// Idempotency-Key dikirim lewat header, bukan body — ini adalah konvensi REST.
type CreateBookingRequest struct {
	RoomID   string `json:"room_id"  validate:"required"`
	CheckIn  string `json:"check_in"  validate:"required"` // format: 2006-01-02
	CheckOut string `json:"check_out" validate:"required"` // format: 2006-01-02
	Notes    string `json:"notes"`
}

// ── Response structs ──────────────────────────────────────────────────────────

// BookingResponse adalah detail booking yang dikirim ke client.
// Menyertakan data room untuk mengurangi kebutuhan request tambahan dari client.
type BookingResponse struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"room_id"`
	RoomName   string    `json:"room_name"`   // diambil dari JOIN dengan tabel rooms
	CheckIn    time.Time `json:"check_in"`
	CheckOut   time.Time `json:"check_out"`
	Status     string    `json:"status"`
	TotalPrice float64   `json:"total_price"`
	TotalNight int       `json:"total_night"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

// BookingWithRoom adalah struct untuk scan hasil query JOIN untuk scan hasil query JOIN.
// Tidak dikirim langsung ke client — dikonversi ke BookingResponse dulu.
type BookingWithRoom struct {
	Booking
	RoomName string `db:"room_name"` // alias dari JOIN dengan tabel rooms
}
