//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}
	if direction != "up" && direction != "down" {
		log.Fatal("argument harus 'up' atau 'down'")
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("gagal buka koneksi:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("gagal ping database:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (version)
		)
	`)
	if err != nil {
		log.Fatal("gagal buat tabel schema_migrations:", err)
	}

	files, err := filepath.Glob(fmt.Sprintf("migrations/*.%s.sql", direction))
	if err != nil || len(files) == 0 {
		log.Fatalf("tidak ada file migration ditemukan untuk direction '%s'", direction)
	}
	sort.Strings(files)

	if direction == "down" {
		for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
			files[i], files[j] = files[j], files[i]
		}
	}

	for _, file := range files {
		version := strings.TrimSuffix(filepath.Base(file), "."+direction+".sql")

		if direction == "up" {
			var count int
			db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
			if count > 0 {
				log.Printf("[skip]    %s\n", version)
				continue
			}
		}

		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("gagal baca file %s: %v", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			log.Fatalf("gagal jalankan migration %s: %v", file, err)
		}

		if direction == "up" {
			db.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version)
			log.Printf("[applied] %s\n", version)
		} else {
			db.Exec("DELETE FROM schema_migrations WHERE version = ?", version)
			log.Printf("[rolled back] %s\n", version)
		}
	}

	log.Printf("migration '%s' selesai\n", direction)
}
