.PHONY: run build migrate seed lint tidy

## Run the API server
run:
	go run ./cmd/api

## Build the binary
build:
	go build -o bin/f1-api ./cmd/api

## Apply database migrations (requires psql and DB_* env vars or .env)
migrate:
	psql "$(DSN)" -f migrations/001_init.sql

## Load example seed data
seed:
	psql "$(DSN)" -f scripts/seed_example.sql

## Tidy modules
tidy:
	go mod tidy

## Run go vet + staticcheck
lint:
	go vet ./...
