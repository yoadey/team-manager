GOPATH       := $(shell go env GOPATH)
OAPI_CODEGEN := $(GOPATH)/bin/oapi-codegen
GOOSE        := $(GOPATH)/bin/goose
DATABASE_URL ?= postgres://teammanager:teammanager@localhost:5432/teammanager?sslmode=disable

.PHONY: all build generate migrate migrate-down test lint clean tools

all: generate build

## Install code-generation tools (run once after cloning)
tools:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest

## Generate Go server code from OpenAPI spec
generate:
	mkdir -p internal/gen
	$(OAPI_CODEGEN) -config oapi-codegen.yaml openapi/openapi.yaml

## Run database migrations (up)
migrate:
	$(GOOSE) -dir internal/db/migrations postgres "$(DATABASE_URL)" up

## Roll back last migration
migrate-down:
	$(GOOSE) -dir internal/db/migrations postgres "$(DATABASE_URL)" down

## Migration status
migrate-status:
	$(GOOSE) -dir internal/db/migrations postgres "$(DATABASE_URL)" status

## Build server binary
build:
	go build -o bin/server ./cmd/server

## Run tests (requires DATABASE_URL pointing to a test DB)
test:
	go test ./... -race -count=1

## Run tests with coverage
test-coverage:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

## Lint with golangci-lint
lint:
	golangci-lint run ./...

## Start the server locally
run: build
	./bin/server

## Clean generated and built files
clean:
	rm -rf bin/ internal/gen/ coverage.out coverage.html
