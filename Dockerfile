FROM golang:1.23-alpine3.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ./bin/api ./cmd/api

# Lightweight docker container with binary only
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin ./bin
COPY --from=builder /app/templates/* ./templates/

CMD ["./bin/api"]
