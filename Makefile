all: coverage swag push

up: swag docker-build
	docker compose up

push: docker-build
	docker push jus1d/dreik-api:latest

build:
	go build -o bin/api cmd/api/main.go

docker-build:
	docker build -t jus1d/dreik-api:latest .

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

swag:
	swag init -g ./cmd/api/main.go

test:
	go test ./...

clean:
	rm coverage.out
	rm coverage.html
