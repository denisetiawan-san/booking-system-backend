package booking_test

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"booking-system/internal/booking"
	"booking-system/internal/room"
	"booking-system/pkg/apperror"
)

// ─────────────────────────────────────────────────────────────────────────────
// Mock Repositories
// ─────────────────────────────────────────────────────────────────────────────

type MockBookingRepo struct{ mock.Mock }

func (m *MockBookingRepo) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqlx.Tx), args.Error(1)
}
func (m *MockBookingRepo) GetByIdempotencyKey(ctx context.Context, key string) (*booking.Booking, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*booking.Booking), args.Error(1)
}
func (m *MockBookingRepo) LockRoom(ctx context.Context, tx *sqlx.Tx, roomID string) error {
	return m.Called(ctx, tx, roomID).Error(0)
}
func (m *MockBookingRepo) CheckOverlap(ctx context.Context, tx *sqlx.Tx, roomID string, ci, co time.Time) (bool, error) {
	args := m.Called(ctx, tx, roomID, ci, co)
	return args.Bool(0), args.Error(1)
}
func (m *MockBookingRepo) Create(ctx context.Context, tx *sqlx.Tx, b *booking.Booking) error {
	return m.Called(ctx, tx, b).Error(0)
}
func (m *MockBookingRepo) GetByID(ctx context.Context, id string) (*booking.BookingWithRoom, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*booking.BookingWithRoom), args.Error(1)
}
func (m *MockBookingRepo) GetByUserID(ctx context.Context, userID string) ([]booking.BookingWithRoom, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]booking.BookingWithRoom), args.Error(1)
}
func (m *MockBookingRepo) Cancel(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

type MockRoomRepo struct{ mock.Mock }

func (m *MockRoomRepo) Create(ctx context.Context, r *room.Room) error {
	return m.Called(ctx, r).Error(0)
}
func (m *MockRoomRepo) GetByID(ctx context.Context, id string) (*room.Room, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*room.Room), args.Error(1)
}
func (m *MockRoomRepo) List(ctx context.Context, limit, offset int) ([]room.Room, int, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]room.Room), args.Int(1), args.Error(2)
}
func (m *MockRoomRepo) Update(ctx context.Context, r *room.Room) error {
	return m.Called(ctx, r).Error(0)
}
func (m *MockRoomRepo) FindAvailable(ctx context.Context, ci, co time.Time, minCap int) ([]room.Room, error) {
	args := m.Called(ctx, ci, co, minCap)
	return args.Get(0).([]room.Room), args.Error(1)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Idempotency
// ─────────────────────────────────────────────────────────────────────────────

// TestCreate_IdempotencyKeyAlreadyUsed memverifikasi bahwa request duplikat
// mengembalikan booking yang sudah ada tanpa membuat booking baru.
// Ini adalah proteksi utama melawan double-click dari user.
func TestCreate_IdempotencyKeyAlreadyUsed(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	roomRepo := new(MockRoomRepo)
	svc := booking.NewService(bookingRepo, roomRepo)

	existingBooking := &booking.Booking{
		ID:             "booking-existing",
		UserID:         "user-1",
		RoomID:         "room-1",
		IdempotencyKey: "key-123",
		Status:         "confirmed",
	}
	existingWithRoom := &booking.BookingWithRoom{
		Booking:  *existingBooking,
		RoomName: "Kamar Deluxe",
	}

	// Simulasi: idempotency key sudah pernah dipakai
	bookingRepo.On("GetByIdempotencyKey", mock.Anything, "key-123").
		Return(existingBooking, nil)
	bookingRepo.On("GetByID", mock.Anything, "booking-existing").
		Return(existingWithRoom, nil)

	result, err := svc.Create(context.Background(), "user-1", "key-123", booking.CreateBookingRequest{
		RoomID:   "room-1",
		CheckIn:  "2099-06-10",
		CheckOut: "2099-06-12",
	})

	assert.NoError(t, err)
	assert.Equal(t, "booking-existing", result.ID)
	// Pastikan tidak ada transaksi baru yang dimulai
	bookingRepo.AssertNotCalled(t, "BeginTx")
	bookingRepo.AssertNotCalled(t, "LockRoom")
}

// ─────────────────────────────────────────────────────────────────────────────
// Test: Cancel validations
// ─────────────────────────────────────────────────────────────────────────────

// TestCancel_NotOwner memverifikasi bahwa user tidak bisa cancel booking orang lain.
func TestCancel_NotOwner(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	roomRepo := new(MockRoomRepo)
	svc := booking.NewService(bookingRepo, roomRepo)

	futureCheckIn := time.Now().AddDate(0, 0, 10)
	futureCheckOut := futureCheckIn.AddDate(0, 0, 2)

	// Booking ini milik user-1
	bookingRepo.On("GetByID", mock.Anything, "booking-1").
		Return(&booking.BookingWithRoom{
			Booking: booking.Booking{
				ID:       "booking-1",
				UserID:   "user-1", // milik user-1
				Status:   "confirmed",
				CheckIn:  futureCheckIn,
				CheckOut: futureCheckOut,
			},
		}, nil)

	// user-2 mencoba cancel booking milik user-1
	err := svc.Cancel(context.Background(), "booking-1", "user-2")

	assert.ErrorIs(t, err, apperror.ErrBookingNotOwned)
	bookingRepo.AssertNotCalled(t, "Cancel")
}

// TestCancel_AlreadyCancelled memverifikasi bahwa booking yang sudah cancelled
// tidak bisa di-cancel lagi.
func TestCancel_AlreadyCancelled(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	roomRepo := new(MockRoomRepo)
	svc := booking.NewService(bookingRepo, roomRepo)

	futureDate := time.Now().AddDate(0, 0, 10)

	bookingRepo.On("GetByID", mock.Anything, "booking-1").
		Return(&booking.BookingWithRoom{
			Booking: booking.Booking{
				ID:       "booking-1",
				UserID:   "user-1",
				Status:   "cancelled", // sudah cancelled
				CheckIn:  futureDate,
				CheckOut: futureDate.AddDate(0, 0, 2),
			},
		}, nil)

	err := svc.Cancel(context.Background(), "booking-1", "user-1")

	assert.ErrorIs(t, err, apperror.ErrAlreadyCancelled)
}

// TestCancel_CheckInAlreadyPassed memverifikasi bahwa booking yang check_in-nya
// sudah lewat tidak bisa dibatalkan (tamu sudah check in).
func TestCancel_CheckInAlreadyPassed(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	roomRepo := new(MockRoomRepo)
	svc := booking.NewService(bookingRepo, roomRepo)

	// Check_in sudah 5 hari yang lalu
	pastCheckIn := time.Now().AddDate(0, 0, -5)
	pastCheckOut := time.Now().AddDate(0, 0, 2)

	bookingRepo.On("GetByID", mock.Anything, "booking-1").
		Return(&booking.BookingWithRoom{
			Booking: booking.Booking{
				ID:       "booking-1",
				UserID:   "user-1",
				Status:   "confirmed",
				CheckIn:  pastCheckIn,
				CheckOut: pastCheckOut,
			},
		}, nil)

	err := svc.Cancel(context.Background(), "booking-1", "user-1")

	assert.ErrorIs(t, err, apperror.ErrCannotCancelPastBooking)
}

// TestCancel_Success memverifikasi skenario happy path cancel booking.
func TestCancel_Success(t *testing.T) {
	bookingRepo := new(MockBookingRepo)
	roomRepo := new(MockRoomRepo)
	svc := booking.NewService(bookingRepo, roomRepo)

	// Check_in 10 hari ke depan — masih bisa di-cancel
	futureCheckIn := time.Now().AddDate(0, 0, 10)

	bookingRepo.On("GetByID", mock.Anything, "booking-1").
		Return(&booking.BookingWithRoom{
			Booking: booking.Booking{
				ID:       "booking-1",
				UserID:   "user-1",
				Status:   "confirmed",
				CheckIn:  futureCheckIn,
				CheckOut: futureCheckIn.AddDate(0, 0, 2),
			},
		}, nil)
	bookingRepo.On("Cancel", mock.Anything, "booking-1").Return(nil)

	err := svc.Cancel(context.Background(), "booking-1", "user-1")

	assert.NoError(t, err)
	bookingRepo.AssertCalled(t, "Cancel", mock.Anything, "booking-1")
}
