Booking System Backend

REST API backend untuk sistem pemesanan kamar hotel/ruangan dengan fitur:
JWT auth, double booking prevention dengan SELECT FOR UPDATE,
date range overlap detection, dan idempotency protection.

- Tech Stack: Go · MySQL · Docker · Nginx · JWT · zerolog

Quick Start

Prasyarat

Pastikan sudah terinstall di komputer kamu:

- [Go 1.21+](https://go.dev/dl/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (pastikan sudah berjalan)
- [Git](https://git-scm.com/)

Catatan port: Project ini menggunakan port yang berbeda dari project lain
agar bisa jalan bersamaan di satu komputer:

- Nginx: :8090 (bukan :80)
- MySQL: :3307 (bukan :3306)

1. Clone repository

git clone https://github.com/USERNAME/booking-system.git
cd booking-system

2. Buat file konfigurasi

cp .env.example .env

Buka file .env yang baru dibuat, lalu sesuaikan nilainya:

APP_ENV=development
APP_PORT=8080

DB_HOST=127.0.0.1
DB_PORT=3307 # port di host komputer kamu (bukan port internal Docker)
DB_NAME=booking_db
DB_USER=booking_user
DB_PASSWORD=secret123 # bebas diganti
DB_ROOT_PASSWORD=rootsecret123 # bebas diganti

JWT_SECRET=isi-dengan-string-acak-panjang-minimal-32-karakter
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

Penting: DB_PORT=3307 adalah port di sisi host komputer.
App container di dalam Docker akan otomatis menggunakan port 3306 (port internal MySQL).
Ini sudah di-handle di docker-compose.yml.

3. Download dependency Go

go mod tidy

4. Jalankan Docker

docker compose up -d --build

Tunggu sekitar 30–60 detik. Cek status:

docker compose ps

Semua container harus berstatus Up:

booking_app Up
booking_mysql Up (healthy)
booking_nginx Up

5. Jalankan database migration

go run ./scripts/migrate.go up

Koneksi Database (TablePlus)

| Field    | Value                        |
| -------- | ---------------------------- |
| Host     | 127.0.0.1                    |
| Port     | 3307                         |
| Database | booking_db                   |
| Username | booking_user                 |
| Password | (sesuai DB_PASSWORD di .env) |

6. Verifikasi server berjalan
   http://localhost:8090/health

Response yang diharapkan:
{ "status": "ok", "service": "booking-system" }

Cara Test API

Import file Postman collection:
docs/booking_postman_collection.json

Cara import:

1. Buka Postman
2. Klik Import → drag & drop booking_postman_collection.json
3. Buat Postman Environment dengan variable:

| Variable       | Value awal                         |
| -------------- | ---------------------------------- |
| base_url       | http://localhost:8090/api/v1       |
| admin_token    | (kosong, isi setelah login)        |
| customer_token | (kosong, isi setelah login)        |
| room_id        | (kosong, isi setelah buat room)    |
| booking_id     | (kosong, isi setelah buat booking) |

Perintah Berguna

- Lihat log real-time
  docker compose logs -f app

- Matikan semua container
  docker compose down

- Rebuild setelah perubahan kode
  docker compose down && docker compose build --no-cache && docker compose up -d

- Rollback migration
  go run ./scripts/migrate.go down

- Unit test
  go test ./... -v
