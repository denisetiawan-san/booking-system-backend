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

const dateLayout = "2006-01-02"

type Service interface {
	Create(ctx context.Context, userID, idempotencyKey string, req CreateBookingRequest) (*BookingResponse, error)

	GetByID(ctx context.Context, id, userID string) (*BookingResponse, error)

	GetMyBookings(ctx context.Context, userID string) ([]BookingResponse, error)

	Cancel(ctx context.Context, id, userID string) error
}

type service struct {
	repo     Repository
	roomRepo room.Repository
}

func NewService(repo Repository, roomRepo room.Repository) Service {
	return &service{repo: repo, roomRepo: roomRepo}
}

func (s *service) Create(ctx context.Context, userID, idempotencyKey string, req CreateBookingRequest) (*BookingResponse, error) {

	existingBooking, err := s.repo.GetByIdempotencyKey(ctx, idempotencyKey)
	if err != nil && err != apperror.ErrNotFound {
		return nil, err
	}
	if existingBooking != nil {
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

	checkIn, err := time.Parse(dateLayout, req.CheckIn)
	if err != nil {
		return nil, apperror.New(400, fmt.Sprintf("format check_in tidak valid, gunakan format: %s (contoh: 2099-06-10)", dateLayout))
	}
	checkOut, err := time.Parse(dateLayout, req.CheckOut)
	if err != nil {
		return nil, apperror.New(400, fmt.Sprintf("format check_out tidak valid, gunakan format: %s (contoh: 2099-06-12)", dateLayout))
	}

	if !checkOut.After(checkIn) {
		return nil, apperror.ErrInvalidDateRange
	}

	today := time.Now().Truncate(24 * time.Hour)
	if checkIn.Before(today) {
		return nil, apperror.ErrCheckInPast
	}

	roomData, err := s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, err
	}

	totalNight := int(checkOut.Sub(checkIn).Hours() / 24)
	totalPrice := roomData.PricePerNight * float64(totalNight)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}

	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error().Err(rbErr).Msg("gagal rollback transaksi booking")
			}
		}
	}()

	if err = s.repo.LockRoom(ctx, tx, req.RoomID); err != nil {
		return nil, err
	}

	hasOverlap, err := s.repo.CheckOverlap(ctx, tx, req.RoomID, checkIn, checkOut)
	if err != nil {
		return nil, err
	}
	if hasOverlap {
		log.Info().
			Str("room_id", req.RoomID).
			Str("check_in", req.CheckIn).
			Str("check_out", req.CheckOut).
			Msg("kamar tidak tersedia: ada booking yang overlap")
		return nil, apperror.ErrRoomNotAvailable
	}

	newBooking := &Booking{
		ID:             uuid.NewString(),
		UserID:         userID,
		RoomID:         req.RoomID,
		CheckIn:        checkIn,
		CheckOut:       checkOut,
		Status:         "confirmed",
		TotalPrice:     totalPrice,
		Notes:          req.Notes,
		IdempotencyKey: idempotencyKey,
	}

	if err = s.repo.Create(ctx, tx, newBooking); err != nil {
		log.Error().Err(err).Str("room_id", req.RoomID).Msg("gagal membuat booking")
		return nil, err
	}

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

	created, err := s.repo.GetByID(ctx, newBooking.ID)
	if err != nil {
		return nil, err
	}
	return toBookingResponse(created), nil
}

func (s *service) GetByID(ctx context.Context, id, userID string) (*BookingResponse, error) {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if b.UserID != userID {
		return nil, apperror.ErrBookingNotOwned
	}

	return toBookingResponse(b), nil
}

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

func (s *service) Cancel(ctx context.Context, id, userID string) error {
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if b.UserID != userID {
		return apperror.ErrBookingNotOwned
	}

	if b.Status == "cancelled" {
		return apperror.ErrAlreadyCancelled
	}

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
