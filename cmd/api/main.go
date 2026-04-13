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

	"booking-system/internal/auth"
	"booking-system/internal/booking"
	"booking-system/internal/room"
	"booking-system/pkg/database"
	"booking-system/pkg/logger"
	"booking-system/pkg/middleware"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Warn().Msg(".env tidak ditemukan, menggunakan environment variable sistem")
	}

	logger.Init(getEnv("APP_ENV", "development"))

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

	validate := validator.New()
	jwtSecret := getEnv("JWT_SECRET", "ganti-ini-di-production")
	accessExpiry, _ := time.ParseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m"))
	refreshExpiry, _ := time.ParseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h"))

	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, auth.Config{
		JWTSecret:          jwtSecret,
		AccessTokenExpiry:  accessExpiry,
		RefreshTokenExpiry: refreshExpiry,
	})
	authHandler := auth.NewHandler(authSvc, validate)

	roomRepo := room.NewRepository(db)
	roomSvc := room.NewService(roomRepo)
	roomHandler := room.NewHandler(roomSvc, validate)

	bookingRepo := booking.NewRepository(db)
	bookingSvc := booking.NewService(bookingRepo, roomRepo)
	bookingHandler := booking.NewHandler(bookingSvc, validate)

	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key"},
	}).Handler)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"booking-system"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		authHandler.RegisterRoutes(r, jwtSecret)
		roomHandler.RegisterRoutes(r, jwtSecret)
		bookingHandler.RegisterRoutes(r, jwtSecret)
	})

	port := getEnv("APP_PORT", "8080")
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", port).Msg("booking system server berjalan")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("sinyal shutdown diterima, mematikan server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("error saat graceful shutdown")
	}

	log.Info().Msg("server berhenti dengan bersih")
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
