include .env

AIR := $(HOME)/go/bin/air

run:
	go run main.go

dev:
	$(AIR)

migrate-up:
	goose postgres "$(DATABASE_URL)" up -dir ./migrations

migrate-down:
	goose postgres "$(DATABASE_URL)" down -dir ./migrations

connect:
	