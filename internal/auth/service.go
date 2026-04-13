package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"booking-system/pkg/apperror"
	"booking-system/pkg/middleware"
)

type Config struct {
	JWTSecret          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (*TokenResponse, error)
	Login(ctx context.Context, req LoginRequest) (*TokenResponse, error)
	RefreshToken(ctx context.Context, req RefreshRequest) (*TokenResponse, error)
	GetProfile(ctx context.Context, userID string) (*UserResponse, error)
}

type service struct {
	repo   Repository
	config Config
}

func NewService(repo Repository, config Config) Service {
	return &service{repo: repo, config: config}
}

func (s *service) Register(ctx context.Context, req RegisterRequest) (*TokenResponse, error) {

	existing, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil && err != apperror.ErrNotFound {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("gagal hash password")
		return nil, err
	}

	user := &User{
		ID:       uuid.NewString(),
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
		Role:     "customer",
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("gagal membuat user baru")
		return nil, err
	}

	log.Info().
		Str("user_id", user.ID).
		Str("email", user.Email).
		Msg("user baru berhasil terdaftar")

	return s.generateTokenPair(ctx, user.ID, user.Role)
}

func (s *service) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == apperror.ErrNotFound {

			return nil, apperror.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		log.Warn().Str("email", req.Email).Msg("percobaan login dengan password salah")
		return nil, apperror.ErrInvalidCredentials
	}

	log.Info().Str("user_id", user.ID).Msg("user berhasil login")
	return s.generateTokenPair(ctx, user.ID, user.Role)
}

func (s *service) RefreshToken(ctx context.Context, req RefreshRequest) (*TokenResponse, error) {
	userID, role, err := s.repo.GetRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	if err := s.repo.RevokeRefreshToken(ctx, req.RefreshToken); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gagal revoke refresh token lama")
		return nil, err
	}

	log.Info().Str("user_id", userID).Msg("refresh token digunakan, token pair baru dibuat")
	return s.generateTokenPair(ctx, userID, role)
}

func (s *service) GetProfile(ctx context.Context, userID string) (*UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}, nil
}

func (s *service) generateTokenPair(ctx context.Context, userID, role string) (*TokenResponse, error) {
	accessToken, err := middleware.GenerateAccessToken(userID, role, s.config.JWTSecret, s.config.AccessTokenExpiry)
	if err != nil {
		return nil, err
	}

	refreshToken, err := middleware.GenerateRefreshToken(userID, role, s.config.JWTSecret, s.config.RefreshTokenExpiry)
	if err != nil {
		return nil, err
	}

	tokenID := uuid.NewString()
	expiresAt := time.Now().Add(s.config.RefreshTokenExpiry)
	if err := s.repo.SaveRefreshToken(ctx, tokenID, userID, refreshToken, expiresAt); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gagal menyimpan refresh token")
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.config.AccessTokenExpiry.Seconds()),
	}, nil
}
