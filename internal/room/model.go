package room

import "time"

type Room struct {
	ID            string    `db:"id"              json:"id"`
	Name          string    `db:"name"            json:"name"`
	Description   string    `db:"description"     json:"description"`
	Capacity      int       `db:"capacity"        json:"capacity"`
	PricePerNight float64   `db:"price_per_night" json:"price_per_night"`
	IsActive      bool      `db:"is_active"       json:"is_active"`
	CreatedAt     time.Time `db:"created_at"      json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"      json:"updated_at"`
}

type CreateRoomRequest struct {
	Name          string  `json:"name"           validate:"required,min=2"`
	Description   string  `json:"description"`
	Capacity      int     `json:"capacity"       validate:"required,min=1"`
	PricePerNight float64 `json:"price_per_night" validate:"required,gt=0"`
}

type UpdateRoomRequest struct {
	Name          *string  `json:"name"           validate:"omitempty,min=2"`
	Description   *string  `json:"description"`
	Capacity      *int     `json:"capacity"       validate:"omitempty,min=1"`
	PricePerNight *float64 `json:"price_per_night" validate:"omitempty,gt=0"`
	IsActive      *bool    `json:"is_active"`
}

type SearchAvailabilityRequest struct {
	CheckIn  string `json:"check_in"  validate:"required"`
	CheckOut string `json:"check_out" validate:"required"`
	Capacity int    `json:"capacity"`
}

type RoomAvailabilityResponse struct {
	Room       Room    `json:"room"`
	TotalNight int     `json:"total_night"`
	TotalPrice float64 `json:"total_price"`
}

type ListRoomResponse struct {
	Rooms []Room `json:"rooms"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}
