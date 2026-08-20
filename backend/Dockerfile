# Pinned to match go.mod's `go` directive exactly -- CI's setup-go (every Go
# job) installs that exact version via go-version-file: backend/go.mod, so
# building the shipped image with a different toolchain would mean the
# binary CI actually tests/scans (go test -race, govulncheck, CodeQL) is not
# the one built into the container. Bump both together.
FROM golang:1.25.13@sha256:cbff9d1a9041b316010f2da6b701b6c0d597718cb90928c85eb597334a0d23d4 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /healthcheck ./cmd/healthcheck

FROM gcr.io/distroless/static-debian12@sha256:6447365a6337c3732f412d1b74357b30a633831955b2bc45552b0086be907687
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
