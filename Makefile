.PHONY: dev up down test itest api-compile

up:
	docker compose up --build

down:
	docker compose down -v

test:
	cd backend && go test ./...

itest:
	cd backend && go test -tags=integration ./...

api-compile:
	cd backend && go build ./cmd/api
