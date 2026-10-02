ifneq (,$(wildcard .env))
include .env
export
endif
export GOCACHE := /tmp/football-go-cache
NODE_BIN := $(HOME)/.nvm/versions/node/$(shell cat .nvmrc 2>/dev/null)/bin
export PATH := $(NODE_BIN):$(PATH)
.PHONY: db-up db-down db-reset migrate migrate-down migrate-status seed backend worker frontend dev test verify test-integration

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

db-reset:
	@test "$(CONFIRM)" = "DELETE_DATABASE" || (echo 'DESTRUCTIVE: use make db-reset CONFIRM=DELETE_DATABASE to delete all local database data'; exit 1)
	docker compose down --volumes
	$(MAKE) db-up migrate seed

migrate:
	cd backend && go run ./cmd/manage migrate-up

migrate-down:
	@test "$(CONFIRM)" = "DROP_SCHEMA" || (echo 'DESTRUCTIVE: use make migrate-down CONFIRM=DROP_SCHEMA'; exit 1)
	cd backend && go run ./cmd/manage migrate-down

migrate-status:
	cd backend && go run ./cmd/manage migrate-status

seed:
	cd backend && go run ./cmd/manage seed

backend:
	cd backend && go run ./cmd/api

worker:
	cd backend && go run ./cmd/worker $(ARGS)

frontend:
	cd frontend && npm run dev

dev: db-up migrate seed
	bash scripts/dev.sh

test:
	cd backend && go test ./...
	cd frontend && npm test

verify: test
	cd backend && go vet ./... && go build ./...
	cd frontend && npm run lint && npm run typecheck && npm run build

# Uses an isolated temporary schema, never the application tables.
test-integration:
	@cd backend && TEST_DATABASE_URL="$(DATABASE_URL)" go test -race ./...
