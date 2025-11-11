SHELL := /usr/bin/bash

.PHONY: dev up down test build api-compile migrate

up:
\tdocker compose up --build

down:
 \tdocker compose down -v

test:
 \tcd backend && go test ./...

itest:
 \tcd backend && go test -tags=integration ./...

api-compile:
 \tcd backend && go build ./cmd/api


