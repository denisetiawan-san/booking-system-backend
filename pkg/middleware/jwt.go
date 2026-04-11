package middleware

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims adalah payload yang disimpan di dalam JWT token.
// UserID dan Role digunakan oleh middleware Auth untuk identifikasi user.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken membuat JWT access token berumur pendek (default 15 menit).
// Token ini dikirim di setiap request protected endpoint:
//   Authorization: Bearer <token>
func GenerateAccessToken(userID, role, secret string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken membuat refresh token berumur panjang (default 7 hari).
// Token ini disimpan di database dan dipakai untuk mendapat access token baru.
// Setelah digunakan, token lama di-revoke (rotation strategy).
func GenerateRefreshToken(userID, role, secret string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// validateToken memverifikasi JWT token string.
// Dipanggil oleh middleware Auth untuk setiap request protected.
// Mengembalikan userID dan role kalau token valid.
func validateToken(tokenString, secret string) (userID, role string, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// Pastikan signing method adalah HMAC (HS256)
		// Ini mencegah "algorithm confusion attack" di mana attacker
		// mengubah header algorithm ke "none" atau RSA
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("signing method tidak valid")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", "", err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", "", errors.New("token tidak valid")
	}

	return claims.UserID, claims.Role, nil
}
