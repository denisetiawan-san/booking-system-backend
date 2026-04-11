# ── Stage 1: Builder ─────────────────────────────────────────────────────────
# Gunakan Go image untuk compile. Image ini besar tapi hanya dipakai saat build.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git untuk go modules yang pakai VCS
RUN apk add --no-cache git

# Copy go.mod dan go.sum dulu (terpisah dari source code).
# Docker cache layer: kalau go.mod tidak berubah, layer download tidak diulang.
COPY go.mod go.sum ./
RUN go mod download

# Copy semua source code
COPY . .

# Build binary. Flag:
# CGO_ENABLED=0 → binary tidak bergantung pada C library (bisa jalan di Alpine/Debian slim)
# GOOS=linux    → target OS Linux (untuk container)
# GOARCH=amd64  → target arsitektur x86-64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd/api/main.go

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
# Image runtime yang kecil — tidak perlu Go compiler, hanya binary yang sudah di-compile.
FROM debian:stable-slim

WORKDIR /app

# Install certificate (untuk HTTPS request) dan timezone data
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

# Copy hanya binary dan file yang dibutuhkan saat runtime
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./main"]
