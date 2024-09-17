all: coverage swag build push

build:
	go build -o bin/api cmd/api/main.go

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

swag:
	swag init -g ./cmd/api/main.go

push:
	docker build -t jus1d/dreik-api:latest .
	docker push jus1d/dreik-api:latest

clean:
	rm coverage.out
	rm coverage.html
