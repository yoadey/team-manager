.PHONY: dev build generate migrate test lint install tools

## Install all tools (run once after cloning)
install:
	npm install
	npm --prefix frontend ci
	cd backend && make tools

## Start full stack via Docker Compose
dev:
	docker compose up --build

## Generate code from OpenAPI spec
generate:
	cd backend && make generate

## Run DB migrations
migrate:
	cd backend && make migrate

## Run all tests
test:
	npm --prefix frontend run test
	cd backend && make test

## Lint everything
lint:
	npm --prefix frontend run lint
	cd backend && make lint

## Build backend binary
build:
	cd backend && make build
