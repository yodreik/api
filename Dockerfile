FROM golang:1.23-alpine3.20 AS builder

COPY go.mod go.sum ./
RUN go mod download

COPY ./ /app
WORKDIR /app
RUN go build -o ./bin/api ./cmd/api

# Lightweight docker container with binary only
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin ./bin

CMD ["./bin/api"]
