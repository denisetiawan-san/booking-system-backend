package database

import (
	"fmt"
	"time"

	// Import blank untuk registrasi driver MySQL ke database/sql
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Config menyimpan semua parameter koneksi database.
// Diisi dari environment variable di main.go.
type Config struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxOpenConns    int           // maksimal koneksi yang bisa dibuka sekaligus
	MaxIdleConns    int           // koneksi idle yang disimpan di pool
	ConnMaxLifetime time.Duration // berapa lama koneksi dipakai sebelum diganti baru
}

// NewMySQL membuka koneksi ke MySQL dan memverifikasi dengan Ping.
// Koneksi pakai connection pool — tidak buka/tutup koneksi setiap query.
//
// Parameter DSN penting:
// parseTime=true   → otomatis scan kolom TIMESTAMP/DATE ke time.Time Go
// charset=utf8mb4  → support emoji dan karakter Unicode penuh
func NewMySQL(cfg Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	// Set connection pool parameters
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Ping untuk verifikasi koneksi benar-benar berhasil
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("gagal ping database: %w", err)
	}

	return db, nil
}
