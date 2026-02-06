.PHONY: run test lint migrate db-up db-down

APP_PORT ?= 8080

## run: start the booking service
run:
	go run ./cmd/booking-service

## test: run all unit tests
test:
	go test -v -race -count=1 ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## migrate: run database migrations (placeholder)
migrate:
	@echo "TODO: implement migrations (e.g. golang-migrate)"

## db-up: start PostgreSQL via docker-compose
db-up:
	docker compose up -d postgres

## db-down: stop PostgreSQL
db-down:
	docker compose down
