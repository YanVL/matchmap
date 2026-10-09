.PHONY: dev down migrate-up migrate-force

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

migrate-up:
	docker compose -f docker-compose.yml run --rm migrate up

migrate-force:
	docker compose -f docker-compose.yml run --rm migrate force $(VERSION)

seed:
	docker compose exec db psql -U matchmap -d matchmap -f /seeds/dev.sql

test:
	go test ./internal/match -v

test-race:
	go test ./internal/match -race -v

format:
	go fmt ./...