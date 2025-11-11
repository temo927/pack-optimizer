# Explicitly set shell to bash for cross-platform compatibility (macOS & Linux)
SHELL := /bin/bash

.PHONY: dev up down test itest test-docker itest-docker api-compile help

help:
	@echo "Available targets:"
	@echo "  make up          - Start all services with Docker Compose"
	@echo "  make down        - Stop services and remove volumes"
	@echo "  make test        - Run all unit tests"
	@echo "  make itest       - Run integration tests"
	@echo "  make api-compile - Compile the Go API binary"

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
