package room

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"booking-system/pkg/apperror"
)

// dateLayout adalah format tanggal yang dipakai di seluruh sistem.
// Format ini mengikuti standar ISO 8601 dan mudah dibaca manusia.
const dateLayout = "2006-01-02"

// Service mendefinisikan business logic untuk domain room.
type Service interface {
	Create(ctx context.Context, req CreateRoomRequest) (*Room, error)
	GetByID(ctx context.Context, id string) (*Room, error)
	List(ctx context.Context, page, limit int) (*ListRoomResponse, error)
	Update(ctx context.Context, id string, req UpdateRoomRequest) (*Room, error)
	SearchAvailable(ctx context.Context, req SearchAvailabilityRequest) ([]RoomAvailabilityResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, req CreateRoomRequest) (*Room, error) {
	room := &Room{
		ID:            uuid.NewString(),
		Name:          req.Name,
		Description:   req.Description,
		Capacity:      req.Capacity,
		PricePerNight: req.PricePerNight,
		IsActive:      true, // kamar baru langsung aktif
	}

	if err := s.repo.Create(ctx, room); err != nil {
		log.Error().Err(err).Str("name", req.Name).Msg("gagal membuat room baru")
		return nil, err
	}

	log.Info().Str("room_id", room.ID).Str("name", room.Name).Msg("room baru berhasil dibuat")
	return room, nil
}

func (s *service) GetByID(ctx context.Context, id string) (*Room, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, page, limit int) (*ListRoomResponse, error) {
	// Normalisasi pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	rooms, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	return &ListRoomResponse{Rooms: rooms, Total: total, Page: page, Limit: limit}, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRoomRequest) (*Room, error) {
	// Ambil data yang ada dulu
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Overlay hanya field yang dikirim (tidak nil)
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Capacity != nil {
		existing.Capacity = *req.Capacity
	}
	if req.PricePerNight != nil {
		existing.PricePerNight = *req.PricePerNight
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}

	log.Info().Str("room_id", id).Msg("room berhasil diupdate")
	return existing, nil
}

// SearchAvailable mencari kamar yang tersedia untuk rentang tanggal tertentu.
//
// Flow:
//   1. Parse dan validasi tanggal dari string ke time.Time
//   2. Validasi bisnis: check_in tidak di masa lalu, check_out setelah check_in
//   3. Query database untuk kamar yang tidak memiliki booking overlap
//   4. Hitung total harga untuk setiap kamar yang ditemukan
func (s *service) SearchAvailable(ctx context.Context, req SearchAvailabilityRequest) ([]RoomAvailabilityResponse, error) {
	// Parse tanggal dari string ke time.Time
	checkIn, err := time.Parse(dateLayout, req.CheckIn)
	if err != nil {
		return nil, apperror.New(400, fmt.Sprintf("format check_in tidak valid, gunakan format: %s (contoh: 2026-04-10)", dateLayout))
	}
	checkOut, err := time.Parse(dateLayout, req.CheckOut)
	if err != nil {
		return nil, apperror.New(400, fmt.Sprintf("format check_out tidak valid, gunakan format: %s (contoh: 2026-04-10)", dateLayout))
	}

	// Validasi: check_out harus setelah check_in
	if !checkOut.After(checkIn) {
		return nil, apperror.ErrInvalidDateRange
	}

	// Validasi: check_in tidak boleh di masa lalu
	// Kita bandingkan hanya tanggalnya (truncate jam) agar "hari ini" masih valid
	today := time.Now().Truncate(24 * time.Hour)
	if checkIn.Before(today) {
		return nil, apperror.ErrCheckInPast
	}

	// Normalisasi kapasitas minimal
	minCapacity := req.Capacity
	if minCapacity < 1 {
		minCapacity = 1
	}

	// Cari kamar yang tersedia
	rooms, err := s.repo.FindAvailable(ctx, checkIn, checkOut, minCapacity)
	if err != nil {
		return nil, err
	}

	// Hitung total harga untuk setiap kamar
	// Jumlah malam = selisih hari antara check_out dan check_in
	totalNight := int(checkOut.Sub(checkIn).Hours() / 24)

	var results []RoomAvailabilityResponse
	for _, r := range rooms {
		results = append(results, RoomAvailabilityResponse{
			Room:       r,
			TotalNight: totalNight,
			TotalPrice: r.PricePerNight * float64(totalNight),
		})
	}

	if results == nil {
		results = []RoomAvailabilityResponse{}
	}

	return results, nil
}


