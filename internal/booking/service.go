package booking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"booking-system/internal/room"
	"booking-system/pkg/apperror"
)

// dateLayout adalah format tanggal standar yang dipakai di seluruh aplikasi.
const dateLayout = "2006-01-02"

// Service mendefinisikan business logic untuk domain booking.
type Service interface {
	// Create adalah operasi paling kompleks — seluruh proses booking ada di sini.
	Create(ctx context.Context, userID, idempotencyKey string, req CreateBookingRequest) (*BookingResponse, error)

	// GetByID mengambil detail satu booking.
	GetByID(ctx context.Context, id, userID string) (*BookingResponse, error)

	// GetMyBookings mengambil semua booking milik user yang login.
	GetMyBookings(ctx context.Context, userID string) ([]BookingResponse, error)

	// Cancel membatalkan booking.
	Cancel(ctx context.Context, id, userID string) error
}

type service struct {
	repo     Repository
	roomRepo room.Repository // dibutuhkan untuk ambil harga kamar saat buat booking
}

func NewService(repo Repository, roomRepo room.Repository) Service {
	return &service{repo: repo, roomRepo: roomRepo}
}

// Create memproses pembuatan booking baru.
//
// ═══════════════════════════════════════════════════════════════
// INI ADALAH INTI DARI SELURUH PROJECT — BACA DENGAN SEKSAMA
// ═══════════════════════════════════════════════════════════════
//
// FLOW LENGKAP:
//
//  1. Idempotency check  → apakah request ini sudah pernah diproses?
//  2. Validasi tanggal   → format benar? check_out > check_in? tidak di masa lalu?
//  3. Ambil data room    → room ada? hitung harga total
//  4. BEGIN TRANSACTION
//  5. Lock room          → SELECT ... FOR UPDATE (kunci baris room)
//  6. Cek overlap        → ada booking lain di tanggal yang sama?
//  7. Insert booking     → simpan booking baru
//  8. COMMIT
//
// KENAPA URUTAN INI PENTING?
//
// Step 5 (lock) HARUS dilakukan SEBELUM step 6 (cek overlap).
// Kalau tidak, bisa terjadi race condition:
//
//   Thread A: cek overlap → tidak ada → (sebelum insert)
//   Thread B: cek overlap → tidak ada → insert booking
//   Thread A: insert booking → DOUBLE BOOKING!
//
// Dengan lock:
//   Thread A: lock room → cek overlap → tidak ada → insert → commit → release lock
//   Thread B: coba lock → BLOCKED sampai A selesai
//   Thread B: lock berhasil → cek overlap → ADA (booking A) → return ErrRoomNotAvailable
//
// MENCEGAH DEADLOCK:
// Deadlock terjadi kalau dua transaksi saling menunggu lock yang dipegang satu sama lain.
// Contoh deadlock:
//   Tx A: lock room 101 → tunggu lock room 202
//   Tx B: lock room 202 → tunggu lock room 101
//   → Kedua transaksi saling tunggu selamanya = DEADLOCK
//
// Cara kita mencegahnya:
//   - Setiap transaksi hanya lock SATU room (tidak ada multi-room booking)
//   - Urutan operasi konsisten (lock → cek → insert, tidak pernah terbalik)
//   - MySQL InnoDB mendeteksi deadlock otomatis dan rollback salah satu transaksi
func (s *service) Create(ctx context.Context, userID, idempotencyKey string, req CreateBookingRequest) (*BookingResponse, error) {

	// ── STEP 1: Idempotency check ─────────────────────────────────────────────
	// Cek apakah booking dengan key ini sudah pernah dibuat.
	// Ini mencegah double booking akibat user double-click atau network retry.
	existingBooking, err := s.repo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil && err != apperror.ErrNotFound {
		return nil, err
	}
	if existingBooking != nil {
		// Sudah pernah dibuat — kembalikan booking yang sama tanpa proses ulang.
		log.Info().
			Str("idempotency_key", idempotencyKey).
			Str("existing_booking_id", existingBooking.ID).
			Msg("idempotency key sudah digunakan, mengembalikan booking yang ada")

		existing, err := s.repo.GetByID(ctx, existingBooking.ID)
		if err != nil {
			return nil, err
		}
		return toBookingResponse(existing), nil
	}

	// ── STEP 2: Validasi dan parse tanggal ────────────────────────────────────
	checkIn, err := time.Parse(dateLayout, req.CheckIn)
	if err != nil {
		return nil, apperror.New(400, fmt.Sprintf("format check_in tidak valid, gunakan format: %s (contoh: 2099-06-10)", dateLayout))
	}
	checkOut, err := time.Parse(dateLayout, req.CheckOut)
	if err != nil {
		return nil, apperror.New(400, fmt.Sprintf("format check_out tidak valid, gunakan format: %s (contoh: 2099-06-12)", dateLayout))
	}

	// check_out harus setelah check_in (minimal 1 hari)
	if !checkOut.After(checkIn) {
		return nil, apperror.ErrInvalidDateRange
	}

	// check_in tidak boleh di masa lalu
	today := time.Now().Truncate(24 * time.Hour)
	if checkIn.Before(today) {
		return nil, apperror.ErrCheckInPast
	}

	// ── STEP 3: Ambil data room dan hitung harga ──────────────────────────────
	roomData, err := s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}

	// Hitung total harga berdasarkan jumlah malam
	totalNight := int(checkOut.Sub(checkIn).Hours() / 24)
	totalPrice := roomData.PricePerNight * float64(totalNight)

	// ── STEP 4: Mulai transaksi ───────────────────────────────────────────────
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}

	// Defer rollback — kalau ada error di langkah berikutnya, semua perubahan dibatalkan.
	// Kalau commit sudah berhasil, rollback pada transaksi yang sudah commit adalah no-op.
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error().Err(rbErr).Msg("gagal rollback transaksi booking")
			}
		}
	}()

	// ── STEP 5: Lock room ─────────────────────────────────────────────────────
	// SELECT ... FOR UPDATE pada baris room ini.
	// Transaksi lain yang mencoba lock room yang sama akan MENUNGGU di sini
	// sampai transaksi ini commit atau rollback.
	// Ini adalah garis pertahanan utama melawan double booking.
	if err = s.repo.LockRoom(ctx, tx, req.RoomID); err != nil {
		return nil, err // room tidak ada atau sudah tidak aktif
	}

	// ── STEP 6: Cek overlap ───────────────────────────────────────────────────
	// Setelah mendapatkan lock, cek apakah ada booking lain yang tanggalnya overlap.
	// Pengecekan ini dilakukan SETELAH lock agar hasilnya dijamin akurat —
	// tidak ada transaksi lain yang bisa insert booking baru sambil kita mengecek.
	hasOverlap, err := s.repo.CheckOverlap(ctx, tx, req.RoomID, checkIn, checkOut)
	if err != nil {
		return nil, err
	}
	if hasOverlap {
		// Ada booking lain di tanggal yang sama → kamar tidak tersedia
		log.Info().
			Str("room_id", req.RoomID).
			Str("check_in", req.CheckIn).
			Str("check_out", req.CheckOut).
			Msg("kamar tidak tersedia: ada booking yang overlap")
		return nil, apperror.ErrRoomNotAvailable
	}

	// ── STEP 7: Buat booking ──────────────────────────────────────────────────
	newBooking := &Booking{
		ID:             uuid.NewString(),
		UserID:         userID,
		RoomID:         req.RoomID,
		CheckIn:        checkIn,
		CheckOut:       checkOut,
		Status:         "confirmed", // langsung confirmed di sistem sederhana ini
		TotalPrice:     totalPrice,
		Notes:          req.Notes,
		IdempotencyKey: idempotencyKey,
	}

	if err = s.repo.Create(ctx, tx, newBooking); err != nil {
		log.Error().Err(err).Str("room_id", req.RoomID).Msg("gagal membuat booking")
		return nil, err
	}

	// ── STEP 8: Commit ────────────────────────────────────────────────────────
	// Semua langkah berhasil → commit untuk membuat perubahan permanen.
	// Lock dilepas setelah commit — transaksi lain yang menunggu bisa jalan.
	if err = tx.Commit(); err != nil {
		log.Error().Err(err).Str("booking_id", newBooking.ID).Msg("gagal commit transaksi booking")
		return nil, err
	}

	log.Info().
		Str("booking_id", newBooking.ID).
		Str("user_id", userID).
		Str("room_id", req.RoomID).
		Str("check_in", req.CheckIn).
		Str("check_out", req.CheckOut).
		Float64("total_price", totalPrice).
		Int("total_night", totalNight).
		Msg("booking berhasil dibuat")

	// Ambil booking lengkap dengan data room untuk response
	created, err := s.repo.GetByID(ctx, newBooking.ID)
	if err != nil {
		return nil, err
	}
	return toBookingResponse(created), nil
}

// GetByID mengambil detail booking.
// Validasi: user hanya bisa lihat booking miliknya sendiri (kecuali admin).
func (s *service) GetByID(ctx context.Context, id, userID string) (*BookingResponse, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Ownership check: user biasa hanya bisa lihat booking miliknya
	if b.UserID != userID {
		return nil, apperror.ErrBookingNotOwned
	}

	return toBookingResponse(b), nil
}

// GetMyBookings mengambil semua booking milik user yang sedang login.
func (s *service) GetMyBookings(ctx context.Context, userID string) ([]BookingResponse, error) {
	bookings, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var responses []BookingResponse
	for _, b := range bookings {
		bCopy := b
		responses = append(responses, *toBookingResponse(&bCopy))
	}

	if responses == nil {
		responses = []BookingResponse{}
	}
	return responses, nil
}

// Cancel membatalkan booking.
//
// Validasi bisnis:
//   1. Booking harus milik user yang request (atau admin)
//   2. Booking tidak boleh sudah cancelled
//   3. Tanggal check_in belum lewat (tamu belum check in)
//      — Kalau sudah check in, tidak bisa cancel via sistem (harus front desk)
func (s *service) Cancel(ctx context.Context, id, userID string) error {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Cek ownership
	if b.UserID != userID {
		return apperror.ErrBookingNotOwned
	}

	// Cek sudah cancelled
	if b.Status == "cancelled" {
		return apperror.ErrAlreadyCancelled
	}

	// Cek tanggal check_in sudah lewat atau belum
	// Kalau check_in sudah lewat, tamu dianggap sudah ada di kamar
	today := time.Now().Truncate(24 * time.Hour)
	if !b.CheckIn.After(today) {
		return apperror.ErrCannotCancelPastBooking
	}

	if err := s.repo.Cancel(ctx, id); err != nil {
		return err
	}

	log.Info().
		Str("booking_id", id).
		Str("user_id", userID).
		Msg("booking berhasil dibatalkan")

	return nil
}

// toBookingResponse mengkonversi BookingWithRoom ke BookingResponse.
func toBookingResponse(b *BookingWithRoom) *BookingResponse {
	totalNight := int(b.CheckOut.Sub(b.CheckIn).Hours() / 24)
	return &BookingResponse{
		ID:         b.ID,
		RoomID:     b.RoomID,
		RoomName:   b.RoomName,
		CheckIn:    b.CheckIn,
		CheckOut:   b.CheckOut,
		Status:     b.Status,
		TotalPrice: b.TotalPrice,
		TotalNight: totalNight,
		Notes:      b.Notes,
		CreatedAt:  b.CreatedAt,
	}
}


