package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"

	"booking-system/pkg/apperror"
)

type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	SaveRefreshToken(ctx context.Context, id, userID, token string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, token string) (userID, role string, err error)
	RevokeRefreshToken(ctx context.Context, token string) error
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, email, password, name, role)
		VALUES (:id, :email, :password, :name, :role)
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	return err
}

func (r *mysqlRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	query := `
		SELECT id, email, password, name, role, created_at, updated_at
		FROM   users
		WHERE  email = ?
	`
	err := r.db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {

		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	query := `
		SELECT id, email, password, name, role, created_at, updated_at
		FROM   users
		WHERE  id = ?
	`
	err := r.db.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) SaveRefreshToken(
	ctx context.Context, id, userID, token string, expiresAt time.Time,
) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, id, userID, token, expiresAt)
	return err
}

func (r *mysqlRepository) GetRefreshToken(ctx context.Context, token string) (string, string, error) {
	var result struct {
		UserID string `db:"user_id"`
		Role   string `db:"role"`
	}

	query := `
		SELECT u.id AS user_id, u.role
		FROM   refresh_tokens rt
		JOIN   users u ON u.id = rt.user_id
		WHERE  rt.token      = ?
		  AND  rt.revoked    = 0
		  AND  rt.expires_at > NOW()
	`
	err := r.db.GetContext(ctx, &result, query, token)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", apperror.ErrInvalidToken
	}
	if err != nil {
		return "", "", err
	}
	return result.UserID, result.Role, nil
}

func (r *mysqlRepository) RevokeRefreshToken(ctx context.Context, token string) error {

	query := `UPDATE refresh_tokens SET revoked = 1 WHERE token = ?`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}
