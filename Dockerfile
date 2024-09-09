FROM golang:1.23-alpine3.20 AS builder

COPY go.mod go.sum ./
RUN go mod download

COPY ./ /welnex-api
WORKDIR /welnex-api
RUN go build -o ./bin/api ./cmd/api

# Lightweight docker container with binary only
FROM alpine:latest

WORKDIR /app

COPY --from=builder /welnex-api/bin ./bin
COPY --from=builder /welnex-api/config ./config

CMD ["./bin/api"]
