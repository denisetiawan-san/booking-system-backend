
FROM golang:1.25-alpine AS builder

WORKDIR /app


RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd/api/main.go


FROM debian:stable-slim

WORKDIR /app


RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./main"]
