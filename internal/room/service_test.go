package room_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"booking-system/internal/room"
	"booking-system/pkg/apperror"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, r *room.Room) error {
	return m.Called(ctx, r).Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*room.Room, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*room.Room), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, limit, offset int) ([]room.Room, int, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]room.Room), args.Int(1), args.Error(2)
}

func (m *MockRepository) Update(ctx context.Context, r *room.Room) error {
	return m.Called(ctx, r).Error(0)
}

func (m *MockRepository) FindAvailable(ctx context.Context, checkIn, checkOut time.Time, minCap int) ([]room.Room, error) {
	args := m.Called(ctx, checkIn, checkOut, minCap)
	return args.Get(0).([]room.Room), args.Error(1)
}

func TestCreate_Success(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*room.Room")).Return(nil)

	result, err := svc.Create(context.Background(), room.CreateRoomRequest{
		Name:          "Kamar Deluxe",
		Capacity:      2,
		PricePerNight: 500000,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "Kamar Deluxe", result.Name)
	assert.True(t, result.IsActive)
}

func TestList_NormalizesDefaultPagination(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	repo.On("List", mock.Anything, 10, 0).Return([]room.Room{}, 0, nil)

	result, err := svc.List(context.Background(), 0, 0)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
	repo.AssertExpectations(t)
}

func TestList_CapsLimitAt100(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	repo.On("List", mock.Anything, 100, 0).Return([]room.Room{}, 0, nil)

	result, err := svc.List(context.Background(), 1, 500)

	assert.NoError(t, err)
	assert.Equal(t, 100, result.Limit)
}

func TestUpdate_PartialUpdate(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	existingRoom := &room.Room{
		ID:            "room-1",
		Name:          "Kamar Lama",
		PricePerNight: 300000,
		Capacity:      2,
		IsActive:      true,
	}

	newPrice := 450000.0
	req := room.UpdateRoomRequest{
		PricePerNight: &newPrice,
	}

	repo.On("GetByID", mock.Anything, "room-1").Return(existingRoom, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*room.Room")).Return(nil)

	result, err := svc.Update(context.Background(), "room-1", req)

	assert.NoError(t, err)
	assert.Equal(t, "Kamar Lama", result.Name)
	assert.Equal(t, 450000.0, result.PricePerNight)
	assert.Equal(t, 2, result.Capacity)
}

func TestUpdate_RoomNotFound(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	repo.On("GetByID", mock.Anything, "tidak-ada").Return(nil, apperror.ErrNotFound)

	newPrice := 100000.0
	_, err := svc.Update(context.Background(), "tidak-ada", room.UpdateRoomRequest{
		PricePerNight: &newPrice,
	})

	assert.ErrorIs(t, err, apperror.ErrNotFound)
	repo.AssertNotCalled(t, "Update")
}

func TestSearchAvailable_InvalidDateFormat(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	_, err := svc.SearchAvailable(context.Background(), room.SearchAvailabilityRequest{
		CheckIn:  "10-04-2099",
		CheckOut: "2099-04-15",
	})

	assert.Error(t, err)
}

func TestSearchAvailable_CheckOutBeforeCheckIn(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	_, err := svc.SearchAvailable(context.Background(), room.SearchAvailabilityRequest{
		CheckIn:  "2099-06-15",
		CheckOut: "2099-06-10",
	})

	assert.ErrorIs(t, err, apperror.ErrInvalidDateRange)
}

func TestSearchAvailable_CheckInInPast(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	_, err := svc.SearchAvailable(context.Background(), room.SearchAvailabilityRequest{
		CheckIn:  "2020-01-01",
		CheckOut: "2020-01-05",
	})

	assert.ErrorIs(t, err, apperror.ErrCheckInPast)
}

func TestSearchAvailable_CalculatesCorrectTotalPrice(t *testing.T) {
	repo := new(MockRepository)
	svc := room.NewService(repo)

	mockRooms := []room.Room{
		{ID: "room-1", Name: "Kamar A", PricePerNight: 500000, Capacity: 2, IsActive: true},
	}

	repo.On("FindAvailable", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockRooms, nil)

	results, err := svc.SearchAvailable(context.Background(), room.SearchAvailabilityRequest{
		CheckIn:  "2099-06-10",
		CheckOut: "2099-06-13",
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, 3, results[0].TotalNight)
	assert.Equal(t, 1500000.0, results[0].TotalPrice)
}
