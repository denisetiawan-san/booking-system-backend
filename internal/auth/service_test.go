package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"booking-system/internal/auth"
	"booking-system/pkg/apperror"
)

// MockRepository adalah implementasi palsu dari auth.Repository.
// Dipakai di test agar tidak butuh database sungguhan.
// Setiap method di-record dan bisa di-assert apakah dipanggil dengan argumen yang benar.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateUser(ctx context.Context, user *auth.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockRepository) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.User), args.Error(1)
}

func (m *MockRepository) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*auth.User), args.Error(1)
}

func (m *MockRepository) SaveRefreshToken(ctx context.Context, id, userID, token string, expiresAt time.Time) error {
	return m.Called(ctx, id, userID, token, expiresAt).Error(0)
}

func (m *MockRepository) GetRefreshToken(ctx context.Context, token string) (string, string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

// newTestService membuat service dengan konfigurasi standar untuk test
func newTestService(repo auth.Repository) auth.Service {
	return auth.NewService(repo, auth.Config{
		JWTSecret:          "test-secret-booking-system-32chars",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	})
}

// bcryptHash adalah hash yang sudah dibuat sebelumnya dari "password123"
// Kita hardcode agar test tidak perlu generate hash baru setiap kali jalan
const bcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// ── Test Register ─────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	repo := new(MockRepository)
	svc := newTestService(repo)

	// Setup expectation: email belum ada
	repo.On("GetUserByEmail", mock.Anything, "budi@test.com").
		Return(nil, apperror.ErrNotFound)
	// CreateUser akan dipanggil dengan User apapun (UUID-nya random)
	repo.On("CreateUser", mock.Anything, mock.AnythingOfType("*auth.User")).
		Return(nil)
	// SaveRefreshToken akan dipanggil
	repo.On("SaveRefreshToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	result, err := svc.Register(context.Background(), auth.RegisterRequest{
		Email: "budi@test.com", Password: "password123", Name: "Budi",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	assert.Equal(t, 900, result.ExpiresIn) // 15 menit = 900 detik
	repo.AssertExpectations(t)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	repo := new(MockRepository)
	svc := newTestService(repo)

	// Email sudah ada di database
	repo.On("GetUserByEmail", mock.Anything, "budi@test.com").
		Return(&auth.User{ID: "existing", Email: "budi@test.com"}, nil)

	_, err := svc.Register(context.Background(), auth.RegisterRequest{
		Email: "budi@test.com", Password: "password123", Name: "Budi",
	})

	assert.ErrorIs(t, err, apperror.ErrEmailAlreadyExists)
	// Pastikan CreateUser TIDAK dipanggil karena email sudah ada
	repo.AssertNotCalled(t, "CreateUser")
}

// ── Test Login ────────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	repo := new(MockRepository)
	svc := newTestService(repo)

	repo.On("GetUserByEmail", mock.Anything, "budi@test.com").
		Return(&auth.User{
			ID: "user-1", Email: "budi@test.com",
			Password: bcryptHash, Role: "customer",
		}, nil)
	repo.On("SaveRefreshToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	result, err := svc.Login(context.Background(), auth.LoginRequest{
		Email: "budi@test.com", Password: "password123",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := new(MockRepository)
	svc := newTestService(repo)

	repo.On("GetUserByEmail", mock.Anything, "budi@test.com").
		Return(&auth.User{
			ID: "user-1", Email: "budi@test.com",
			Password: bcryptHash, Role: "customer",
		}, nil)

	_, err := svc.Login(context.Background(), auth.LoginRequest{
		Email: "budi@test.com", Password: "salah",
	})

	assert.ErrorIs(t, err, apperror.ErrInvalidCredentials)
}

func TestLogin_EmailNotFound_ReturnsSameErrorAsWrongPassword(t *testing.T) {
	// Test ini memverifikasi bahwa "email tidak ada" dan "password salah"
	// memberikan response error yang SAMA (anti user enumeration attack)
	repo := new(MockRepository)
	svc := newTestService(repo)

	repo.On("GetUserByEmail", mock.Anything, "tidakada@test.com").
		Return(nil, apperror.ErrNotFound)

	_, err := svc.Login(context.Background(), auth.LoginRequest{
		Email: "tidakada@test.com", Password: "password123",
	})

	// HARUS ErrInvalidCredentials, bukan ErrNotFound
	assert.ErrorIs(t, err, apperror.ErrInvalidCredentials)
}

// ── Test Refresh Token ────────────────────────────────────────────────────────

func TestRefreshToken_Success(t *testing.T) {
	repo := new(MockRepository)
	svc := newTestService(repo)

	repo.On("GetRefreshToken", mock.Anything, "valid-token").
		Return("user-1", "customer", nil)
	repo.On("RevokeRefreshToken", mock.Anything, "valid-token").
		Return(nil)
	repo.On("SaveRefreshToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	result, err := svc.RefreshToken(context.Background(), auth.RefreshRequest{
		RefreshToken: "valid-token",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	// Token lama harus di-revoke
	repo.AssertCalled(t, "RevokeRefreshToken", mock.Anything, "valid-token")
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	repo := new(MockRepository)
	svc := newTestService(repo)

	repo.On("GetRefreshToken", mock.Anything, "invalid-token").
		Return("", "", apperror.ErrInvalidToken)

	_, err := svc.RefreshToken(context.Background(), auth.RefreshRequest{
		RefreshToken: "invalid-token",
	})

	assert.ErrorIs(t, err, apperror.ErrInvalidToken)
	// RevokeRefreshToken tidak boleh dipanggil kalau token tidak valid
	repo.AssertNotCalled(t, "RevokeRefreshToken")
}
