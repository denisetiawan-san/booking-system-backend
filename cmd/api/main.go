package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"

	// Setiap domain di-import dan di-wire di sini.
	// main.go adalah satu-satunya file yang "tahu" semua domain.
	"booking-system/internal/auth"
	"booking-system/internal/booking"
	"booking-system/internal/room"
	"booking-system/pkg/database"
	"booking-system/pkg/logger"
	"booking-system/pkg/middleware"
)

func main() {
	// ── 1. Load konfigurasi ───────────────────────────────────────────────────
	// godotenv membaca file .env dan mengisi os.Getenv.
	// Di Docker, env var sudah diisi docker-compose sehingga .env tidak wajib ada.
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg(".env tidak ditemukan, menggunakan environment variable sistem")
	}

	// ── 2. Setup logger ───────────────────────────────────────────────────────
	logger.Init(getEnv("APP_ENV", "development"))

	// ── 3. Koneksi database ───────────────────────────────────────────────────
	connMaxLifetime, _ := time.ParseDuration(getEnv("DB_CONN_MAX_LIFETIME", "5m"))
	db, err := database.NewMySQL(database.Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "3306"),
		Name:            getEnv("DB_NAME", "booking_db"),
		User:            getEnv("DB_USER", "root"),
		Password:        getEnv("DB_PASSWORD", ""),
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: connMaxLifetime,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("gagal konek ke database")
	}
	defer db.Close()
	log.Info().Msg("koneksi database berhasil")

	// ── 4. Shared dependencies ────────────────────────────────────────────────
	// validate dibuat sekali dan di-share ke semua handler.
	validate := validator.New()
	jwtSecret := getEnv("JWT_SECRET", "ganti-ini-di-production")
	accessExpiry, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	refreshExpiry, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))

	// ── 5. Dependency injection ───────────────────────────────────────────────
	// Pola: NewRepository(db) → NewService(repo) → NewHandler(svc, validate)
	// Urutan ini tidak boleh dibalik.

	// Auth
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, auth.Config{
		JWTSecret:          jwtSecret,
		AccessTokenExpiry:  accessExpiry,
		RefreshTokenExpiry: refreshExpiry,
	})
	authHandler := auth.NewHandler(authSvc, validate)

	// Room
	// Room repo di-share ke booking service agar booking service bisa ambil harga kamar
	roomRepo := room.NewRepository(db)
	roomSvc := room.NewService(roomRepo)
	roomHandler := room.NewHandler(roomSvc, validate)

	// Booking — menerima roomRepo sebagai dependency karena butuh data harga kamar
	bookingRepo := booking.NewRepository(db)
	bookingSvc := booking.NewService(bookingRepo, roomRepo)
	bookingHandler := booking.NewHandler(bookingSvc, validate)

	// ── 6. Setup router ───────────────────────────────────────────────────────
	r := chi.NewRouter()

	// Global middleware — dijalankan untuk SETIAP request, dalam urutan ini:
	// 1. Recoverer: tangkap panic, jangan crash server
	// 2. RequestID: tambah X-Request-Id untuk tracing
	// 3. Logger: catat setiap request (method, path, status, durasi)
	// 4. CORS: izinkan request dari browser dan Postman
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key"},
	}).Handler)

	// Health check — endpoint sederhana untuk membuktikan server hidup
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"booking-system"}`))
	})

	// Semua API route menggunakan prefix /api/v1
	r.Route("/api/v1", func(r chi.Router) {
		authHandler.RegisterRoutes(r, jwtSecret)
		roomHandler.RegisterRoutes(r, jwtSecret)
		bookingHandler.RegisterRoutes(r, jwtSecret)
	})

	// ── 7. Start server dengan graceful shutdown ──────────────────────────────
	port := getEnv("APP_PORT", "8080")
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server berjalan di goroutine terpisah agar tidak block sinyal shutdown
	go func() {
		log.Info().Str("port", port).Msg("booking system server berjalan")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Block sampai SIGINT (Ctrl+C) atau SIGTERM (docker stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("sinyal shutdown diterima, mematikan server...")

	// Beri waktu 30 detik untuk request yang sedang berjalan selesai
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("error saat graceful shutdown")
	}

	log.Info().Msg("server berhenti dengan bersih")
}

// getEnv mengambil nilai env var, kembalikan defaultValue kalau tidak ada.
func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
