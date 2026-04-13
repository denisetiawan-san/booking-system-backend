package room

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"booking-system/pkg/apperror"
)

type Repository interface {
	Create(ctx context.Context, room *Room) error
	GetByID(ctx context.Context, id string) (*Room, error)
	List(ctx context.Context, limit, offset int) ([]Room, int, error)
	Update(ctx context.Context, room *Room) error

	FindAvailable(ctx context.Context, checkIn, checkOut time.Time, minCapacity int) ([]Room, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) Create(ctx context.Context, room *Room) error {
	query := `
		INSERT INTO rooms (id, name, description, capacity, price_per_night, is_active)
		VALUES (:id, :name, :description, :capacity, :price_per_night, :is_active)
	`
	_, err := r.db.NamedExecContext(ctx, query, room)
	return err
}

func (r *mysqlRepository) GetByID(ctx context.Context, id string) (*Room, error) {
	var room Room
	query := `
		SELECT id, name, description, capacity, price_per_night, is_active, created_at, updated_at
		FROM   rooms
		WHERE  id = ?
	`
	err := r.db.GetContext(ctx, &room, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *mysqlRepository) List(ctx context.Context, limit, offset int) ([]Room, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM rooms WHERE is_active = 1`); err != nil {
		return nil, 0, err
	}

	var rooms []Room
	query := `
		SELECT id, name, description, capacity, price_per_night, is_active, created_at, updated_at
		FROM   rooms
		WHERE  is_active = 1
		ORDER  BY created_at DESC
		LIMIT  ? OFFSET ?
	`
	if err := r.db.SelectContext(ctx, &rooms, query, limit, offset); err != nil {
		return nil, 0, err
	}

	if rooms == nil {
		rooms = []Room{}
	}
	return rooms, total, nil
}

func (r *mysqlRepository) Update(ctx context.Context, room *Room) error {
	query := `
		UPDATE rooms
		SET    name           = :name,
		       description    = :description,
		       capacity       = :capacity,
		       price_per_night = :price_per_night,
		       is_active      = :is_active
		WHERE  id             = :id
	`
	_, err := r.db.NamedExecContext(ctx, query, room)
	return err
}

func (r *mysqlRepository) FindAvailable(ctx context.Context, checkIn, checkOut time.Time, minCapacity int) ([]Room, error) {
	var rooms []Room
	query := `
		SELECT r.id, r.name, r.description, r.capacity, r.price_per_night, r.is_active,
		       r.created_at, r.updated_at
		FROM   rooms r
		WHERE  r.is_active = 1
		  AND  r.capacity  >= ?
		  AND  NOT EXISTS (
		           SELECT 1
		           FROM   bookings b
		           WHERE  b.room_id = r.id
		             AND  b.status != 'cancelled'
		             -- Kondisi overlap: booking yang ada TIDAK selesai sebelum check_in kita
		             -- DAN TIDAK mulai setelah check_out kita
		             AND  b.check_in  < ?
		             AND  b.check_out > ?
		       )
		ORDER  BY r.price_per_night ASC
	`

	if err := r.db.SelectContext(ctx, &rooms, query, minCapacity, checkOut, checkIn); err != nil {
		return nil, err
	}
	if rooms == nil {
		rooms = []Room{}
	}
	return rooms, nil
}
