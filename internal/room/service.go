package room

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"booking-system/pkg/apperror"
)

const dateLayout = "2006-01-02"

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
		IsActive:      true,
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
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

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

func (s *service) SearchAvailable(ctx context.Context, req SearchAvailabilityRequest) ([]RoomAvailabilityResponse, error) {
	checkIn, err := time.Parse(dateLayout, req.CheckIn)
	if err != nil {
		return nil, apperror.New(400, fmt.Sprintf("format check_in tidak valid, gunakan format: %s (contoh: 2026-04-10)", dateLayout))
	}
	checkOut, err := time.Parse(dateLayout, req.CheckOut)
	if err != nil {
		return nil, apperror.New(400, fmt.Sprintf("format check_out tidak valid, gunakan format: %s (contoh: 2026-04-10)", dateLayout))
	}

	if !checkOut.After(checkIn) {
		return nil, apperror.ErrInvalidDateRange
	}

	today := time.Now().Truncate(24 * time.Hour)
	if checkIn.Before(today) {
		return nil, apperror.ErrCheckInPast
	}

	minCapacity := req.Capacity
	if minCapacity < 1 {
		minCapacity = 1
	}

	rooms, err := s.repo.FindAvailable(ctx, checkIn, checkOut, minCapacity)
	if err != nil {
		return nil, err
	}

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
