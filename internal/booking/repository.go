package booking

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"booking-system/pkg/apperror"
)

type Repository interface {
	GetByIdempotencyKey(ctx context.Context, key string) (*Booking, error)

	CheckOverlap(ctx context.Context, tx *sqlx.Tx, roomID string, checkIn, checkOut time.Time) (bool, error)

	LockRoom(ctx context.Context, tx *sqlx.Tx, roomID string) error

	Create(ctx context.Context, tx *sqlx.Tx, booking *Booking) error

	GetByID(ctx context.Context, id string) (*BookingWithRoom, error)

	GetByUserID(ctx context.Context, userID string) ([]BookingWithRoom, error)

	Cancel(ctx context.Context, id string) error

	BeginTx(ctx context.Context) (*sqlx.Tx, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *mysqlRepository) GetByIdempotencyKey(ctx context.Context, key string) (*Booking, error) {
	var b Booking
	query := `
		SELECT id, user_id, room_id, check_in, check_out, status,
		       total_price, notes, idempotency_key, created_at, updated_at
		FROM   bookings
		WHERE  idempotency_key = ?
	`
	err := r.db.GetContext(ctx, &b, query, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *mysqlRepository) LockRoom(ctx context.Context, tx *sqlx.Tx, roomID string) error {

	var id string
	query := `SELECT id FROM rooms WHERE id = ? AND is_active = 1 FOR UPDATE`
	err := tx.GetContext(ctx, &id, query, roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return apperror.ErrNotFound
	}
	return err
}

func (r *mysqlRepository) CheckOverlap(ctx context.Context, tx *sqlx.Tx, roomID string, checkIn, checkOut time.Time) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM   bookings
		WHERE  room_id  = ?
		  AND  status  != 'cancelled'
		  AND  check_in  < ?
		  AND  check_out > ?
	`
	err := tx.GetContext(ctx, &count, query, roomID, checkOut, checkIn)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *mysqlRepository) Create(ctx context.Context, tx *sqlx.Tx, b *Booking) error {
	query := `
		INSERT INTO bookings
		    (id, user_id, room_id, check_in, check_out, status, total_price, notes, idempotency_key)
		VALUES
		    (:id, :user_id, :room_id, :check_in, :check_out, :status, :total_price, :notes, :idempotency_key)
	`
	_, err := tx.NamedExecContext(ctx, query, b)
	return err
}

func (r *mysqlRepository) GetByID(ctx context.Context, id string) (*BookingWithRoom, error) {
	var b BookingWithRoom
	query := `
		SELECT b.id, b.user_id, b.room_id, b.check_in, b.check_out,
		       b.status, b.total_price, b.notes, b.idempotency_key,
		       b.created_at, b.updated_at,
		       r.name AS room_name
		FROM   bookings b
		JOIN   rooms r ON r.id = b.room_id
		WHERE  b.id = ?
	`
	err := r.db.GetContext(ctx, &b, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *mysqlRepository) GetByUserID(ctx context.Context, userID string) ([]BookingWithRoom, error) {
	var bookings []BookingWithRoom
	query := `
		SELECT b.id, b.user_id, b.room_id, b.check_in, b.check_out,
		       b.status, b.total_price, b.notes, b.idempotency_key,
		       b.created_at, b.updated_at,
		       r.name AS room_name
		FROM   bookings b
		JOIN   rooms r ON r.id = b.room_id
		WHERE  b.user_id = ?
		ORDER  BY b.created_at DESC
	`
	if err := r.db.SelectContext(ctx, &bookings, query, userID); err != nil {
		return nil, err
	}
	if bookings == nil {
		bookings = []BookingWithRoom{}
	}
	return bookings, nil
}

func (r *mysqlRepository) Cancel(ctx context.Context, id string) error {
	query := `UPDATE bookings SET status = 'cancelled' WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
