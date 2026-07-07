GOPATH          := $(shell go env GOPATH)
OAPI_CODEGEN    := $(GOPATH)/bin/oapi-codegen
GOOSE           := $(GOPATH)/bin/goose
GOLANGCI_LINT   := $(GOPATH)/bin/golangci-lint
GOVULNCHECK     := $(GOPATH)/bin/govulncheck
DATABASE_URL    ?= postgres://teammanager:teammanager@localhost:5432/teammanager?sslmode=disable

.PHONY: all build generate migrate migrate-down migrate-status test test-unit test-integration test-coverage lint vuln run clean tools

all: generate build

## Install code-generation and quality tools (run once after cloning)
# All tools are pinned (not @latest) so a compromised or yanked upstream
# release can't execute unpinned on a dev machine or in CI; bump deliberately.
# oapi-codegen and goose are pinned to match the versions in go.mod so the
# CLI used to (re)generate code / run migrations agrees with the library
# version the rest of the build depends on. golangci-lint is pinned to match
# .github/workflows/ci.yml's version so local `make lint` and CI agree on
# findings — bump both together deliberately.
tools:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.1
	go install github.com/pressly/goose/v3/cmd/goose@v3.24.2
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.12.2
	go install golang.org/x/vuln/cmd/govulncheck@v1.5.0

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

## Run all tests (requires DATABASE_URL pointing to a test DB)
test:
	go test ./... -race -count=1

## Run only unit/short tests (no external dependencies required)
test-unit:
	go test ./... -race -count=1 -short

## Run integration tests (requires Docker for testcontainers). The old
## "-run Integration" filter matched zero test names (no Test* function in
## this codebase contains that substring) and silently reported "ok" having
## run nothing; Docker-backed tests are actually gated by
## testutil.RequireDocker checking testing.Short(), not by test naming, so
## the correct way to select them is to run everything WITHOUT -short.
test-integration:
	go test ./... -race -count=1

## Run tests with coverage report
test-coverage:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

## Lint with golangci-lint
lint:
	$(GOLANGCI_LINT) run ./...

## Check for known vulnerabilities
vuln:
	$(GOVULNCHECK) ./...

## Start the server locally
run: build
	./bin/server

## Clean generated and built files
clean:
	rm -rf bin/ internal/gen/ coverage.out coverage.html
