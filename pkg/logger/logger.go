package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init mengkonfigurasi zerolog global satu kali saat aplikasi start.
//
// Development mode: output berwarna dan mudah dibaca manusia di terminal.
// Contoh output development:
//   10:30:15 INF booking berhasil dibuat booking_id=uuid room_id=uuid
//
// Production mode: output JSON untuk log aggregator (Datadog, Loki, Grafana).
// Contoh output production:
//   {"level":"info","time":"2026-04-08T10:30:15Z","booking_id":"uuid","message":"booking berhasil dibuat"}
func Init(env string) {
	zerolog.TimeFieldFormat = time.RFC3339

	if env != "production" {
		// ConsoleWriter membuat output mudah dibaca saat development
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		})
	} else {
		// JSON format untuk production — mudah di-parse oleh log aggregator
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}
