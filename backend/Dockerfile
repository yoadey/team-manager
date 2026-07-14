# Pinned to match go.mod's `go` directive exactly -- CI's setup-go (every Go
# job) installs that exact version via go-version-file: backend/go.mod, so
# building the shipped image with a different toolchain would mean the
# binary CI actually tests/scans (go test -race, govulncheck, CodeQL) is not
# the one built into the container. Bump both together.
FROM golang:1.25.12@sha256:d7912cedddfa15b2900a8dfb7187df0af5ec2cb424a371139b5b352fd3e6b740 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /healthcheck ./cmd/healthcheck

FROM gcr.io/distroless/static-debian12@sha256:9c346e4be81b5ca7ff31a0d89eaeade58b0f95cfd3baed1f36083ddb47ca3160
COPY --from=builder /server /server
COPY --from=builder /healthcheck /healthcheck
COPY --from=builder /app/internal/db/migrations /internal/db/migrations
ENV MIGRATIONS_DIR=/internal/db/migrations
EXPOSE 8080
USER nonroot:nonroot
# distroless/static has no shell, so HEALTHCHECK must use exec form with a
# real binary rather than "CMD-SHELL ... || ..." — see cmd/healthcheck.
HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=5 \
    CMD ["/healthcheck"]
ENTRYPOINT ["/server"]
