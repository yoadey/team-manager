.PHONY: dev build generate generate-ts migrate test lint install tools

## Install all tools (run once after cloning)
install:
	npm install
	npm --prefix frontend ci
	cd backend && make tools

## Start full stack via Docker Compose
dev:
	docker compose up --build

## Generate Go server code from OpenAPI spec
generate:
	cd backend && make generate

## Generate TypeScript client types from OpenAPI spec
generate-ts:
	npm --prefix frontend exec -- openapi-typescript backend/openapi/openapi.yaml -o frontend/src/api/types.gen.ts
	npm --prefix frontend exec -- openapi-zod-client backend/openapi/openapi.yaml -o frontend/src/api/zod.gen.ts --export-schemas

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
