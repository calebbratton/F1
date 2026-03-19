.PHONY: run build seed migrate test lint tidy docker-up docker-down

## Start the API (reads .env)
run:
	go run ./cmd/api

## Build both binaries
build:
	go build -o bin/f1-api  ./cmd/api
	go build -o bin/f1-seed ./cmd/seed

## Seed the DB from Jolpica API (default: 2020–now)
## Use FROM=1950 to import all history
seed:
	go run ./cmd/seed -from $(or $(FROM),2020)

## Apply all migrations manually (alternative to docker-compose auto-migration)
## Requires: DSN="postgres://user:pass@host/db?sslmode=disable"
migrate:
	psql "$(DSN)" -f migrations/001_init.sql
	psql "$(DSN)" -f migrations/002_live_state.sql

## Run all tests
test:
	go test ./...

## Run go vet
lint:
	go vet ./...

## Tidy dependencies
tidy:
	go mod tidy

## Start Postgres + API via Docker Compose
docker-up:
	docker compose up --build -d
	@echo "API: http://localhost:8080"

## Stop Docker Compose services
docker-down:
	docker compose down
