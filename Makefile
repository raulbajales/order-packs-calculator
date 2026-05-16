ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

DB_URL ?= postgres://postgres:postgres@localhost:5432/challenge?sslmode=disable

.PHONY: run build test integration-test docker-up docker-down

## run: start the development server, needs DB_URL and WEB_DIR env vars
run:
	DATABASE_URL=$(DB_URL) WEB_DIR=$(ROOT)web go -C $(ROOT) run ./cmd/web

## build: compile the binary to ./bin/web
build:
	go -C $(ROOT) build -o $(ROOT)bin/web ./cmd/web

## test: run unit tests only (no DB required)
test:
	go -C $(ROOT) test ./... -v -count=1

## integration-test: run all tests including integration tests (requires DB, run make docker-up first)
integration-test:
	DATABASE_URL=$(DB_URL) go -C $(ROOT) test ./... -v -count=1 -tags=integration

## docker-up: start postgres and the app via docker compose
docker-up:
	docker-compose -f $(ROOT)docker-compose.yml up --build -d

## docker-down: stop and remove containers
docker-down:
	docker-compose -f $(ROOT)docker-compose.yml down
