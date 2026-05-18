.PHONY: run build test migrate-up migrate-down

run:
	go run cmd/bot/main.go

build:
	go build -o bin/bot cmd/bot/main.go

test:
	go test ./...

migrate-up:
	# Add your migration tool command here, e.g., golang-migrate
	@echo "Running migrations up..."

migrate-down:
	@echo "Running migrations down..."
