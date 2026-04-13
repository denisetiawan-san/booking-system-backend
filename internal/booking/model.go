package booking

import "time"

type Booking struct {
	ID         string    `db:"id"              json:"id"`
	UserID     string    `db:"user_id"         json:"user_id"`
	RoomID     string    `db:"room_id"         json:"room_id"`
	CheckIn    time.Time `db:"check_in"        json:"check_in"`
	CheckOut   time.Time `db:"check_out"       json:"check_out"`
	Status     string    `db:"status"          json:"status"`
	TotalPrice float64   `db:"total_price"     json:"total_price"`
	Notes      string    `db:"notes"           json:"notes"`

	IdempotencyKey string    `db:"idempotency_key" json:"idempotency_key"`
	CreatedAt      time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"      json:"updated_at"`
}

type CreateBookingRequest struct {
	RoomID   string `json:"room_id"  validate:"required"`
	CheckIn  string `json:"check_in"  validate:"required"`
	CheckOut string `json:"check_out" validate:"required"`
	Notes    string `json:"notes"`
}

type BookingResponse struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"room_id"`
	RoomName   string    `json:"room_name"`
	CheckIn    time.Time `json:"check_in"`
	CheckOut   time.Time `json:"check_out"`
	Status     string    `json:"status"`
	TotalPrice float64   `json:"total_price"`
	TotalNight int       `json:"total_night"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

type BookingWithRoom struct {
	Booking
	RoomName string `db:"room_name"`
}
