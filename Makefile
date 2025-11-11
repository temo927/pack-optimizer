.PHONY: dev up down test itest test-docker itest-docker api-compile

up:
	docker compose up --build

down:
	docker compose down -v

test:
	cd backend && go test -v ./...

itest:
	cd backend && go test -v -tags=integration ./...

api-compile:
	cd backend && go build ./cmd/api
