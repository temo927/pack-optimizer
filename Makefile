.PHONY: dev up down test itest test-docker itest-docker api-compile

up:
	docker compose up --build

down:
	docker compose down -v

test:
	cd backend && go test ./...

itest:
	cd backend && go test -tags=integration ./...

test-docker:
	docker compose exec api go test ./...

itest-docker:
	docker compose exec api go test -tags=integration ./internal/integration/...

api-compile:
	cd backend && go build ./cmd/api
