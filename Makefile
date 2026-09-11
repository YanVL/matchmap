.PHONY: dev down migrate-up migrate-force

dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

migrate-up:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml run --rm migrate \
		-path /migrations \
		-database "$(shell grep '^DATABASE_URL=' .env | cut -d '=' -f2-)" \
		up

migrate-force:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml run --rm migrate \
		-path /migrations \
		-database "$(shell grep '^DATABASE_URL=' .env | cut -d '=' -f2-)" \
		force $(VERSION)