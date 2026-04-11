package booking

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"booking-system/pkg/apperror"
)

// Repository mendefinisikan kontrak database untuk domain booking.
type Repository interface {
	// GetByIdempotencyKey mencari booking berdasarkan idempotency key.
	// Dipanggil pertama kali saat create booking untuk cek duplikasi.
	GetByIdempotencyKey(ctx context.Context, key string) (*Booking, error)

	// CheckOverlap memeriksa apakah ada booking aktif yang overlap dengan
	// rentang tanggal yang diminta untuk room tertentu.
	// Ini adalah query kritis yang menentukan apakah kamar tersedia atau tidak.
	//
	// DIJALANKAN DI DALAM TRANSAKSI dengan SELECT ... FOR UPDATE pada room
	// agar tidak terjadi race condition (dua user booking kamar yang sama bersamaan).
	CheckOverlap(ctx context.Context, tx *sqlx.Tx, roomID string, checkIn, checkOut time.Time) (bool, error)

	// LockRoom mengunci baris room di dalam transaksi dengan SELECT ... FOR UPDATE.
	// Ini adalah langkah PERTAMA dalam proses create booking untuk mencegah race condition.
	LockRoom(ctx context.Context, tx *sqlx.Tx, roomID string) error

	// Create menyimpan booking baru ke database di dalam transaksi.
	Create(ctx context.Context, tx *sqlx.Tx, booking *Booking) error

	// GetByID mengambil satu booking berdasarkan ID, beserta data room-nya.
	GetByID(ctx context.Context, id string) (*BookingWithRoom, error)

	// GetByUserID mengambil semua booking milik satu user.
	GetByUserID(ctx context.Context, userID string) ([]BookingWithRoom, error)

	// Cancel mengubah status booking menjadi 'cancelled'.
	Cancel(ctx context.Context, id string) error

	// BeginTx memulai transaksi database baru.
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

// LockRoom mengunci baris room dengan SELECT ... FOR UPDATE.
//
// ═══════════════════════════════════════════════════════════════
// INI ADALAH KUNCI MENCEGAH RACE CONDITION DI BOOKING SYSTEM
// ═══════════════════════════════════════════════════════════════
//
// Tanpa locking, bisa terjadi skenario ini:
//   Waktu T1: User A cek kamar 101 untuk 10-15 Apr → tersedia
//   Waktu T2: User B cek kamar 101 untuk 10-15 Apr → tersedia (sebelum A commit)
//   Waktu T3: User A buat booking → berhasil
//   Waktu T4: User B buat booking → BERHASIL juga (DOUBLE BOOKING!)
//
// Dengan SELECT FOR UPDATE:
//   Waktu T1: User A lock baris room 101 → cek → tersedia → buat booking → commit → unlock
//   Waktu T2: User B coba lock room 101 → BLOCKED (menunggu A selesai)
//   Waktu T3: A commit, lock dilepas
//   Waktu T4: B berhasil lock → cek lagi → sudah ada booking! → return ErrRoomNotAvailable
//
// Kenapa lock room, bukan lock booking?
// Karena booking belum ada saat kita cek — yang ada adalah room.
// Lock room menjamin hanya satu transaksi yang bisa cek dan buat booking untuk room ini sekaligus.
func (r *mysqlRepository) LockRoom(ctx context.Context, tx *sqlx.Tx, roomID string) error {
	// SELECT ... FOR UPDATE mengunci baris room ini sampai transaksi commit/rollback.
	// Transaksi lain yang mencoba SELECT FOR UPDATE row yang sama akan MENUNGGU.
	var id string
	query := `SELECT id FROM rooms WHERE id = ? AND is_active = 1 FOR UPDATE`
	err := tx.GetContext(ctx, &id, query, roomID)
	if errors.Is(err, sql.ErrNoRows) {
		return apperror.ErrNotFound
	}
	return err
}

// CheckOverlap memeriksa apakah ada booking aktif yang tanggalnya overlap.
//
// Formula overlap: booking_existing.check_in < request.check_out
//                  AND booking_existing.check_out > request.check_in
//
// Artinya: booking yang ada MULAI sebelum kita selesai DAN SELESAI setelah kita mulai.
// Kalau keduanya benar, ada overlap.
//
// Harus dijalankan DI DALAM TRANSAKSI yang sama dengan LockRoom
// agar atomik — tidak ada booking baru yang bisa masuk di antara check dan insert.
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
	// Parameter: roomID, checkOut (end of new booking), checkIn (start of new booking)
	err := tx.GetContext(ctx, &count, query, roomID, checkOut, checkIn)
	if err != nil {
		return false, err
	}
	return count > 0, nil // true = ada overlap = kamar tidak tersedia
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
	// JOIN dengan rooms untuk mendapat nama kamar sekaligus
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
