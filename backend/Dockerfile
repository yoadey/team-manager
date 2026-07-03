FROM golang:1.26@sha256:f96cc555eb8db430159a3aa6797cd5bae561945b7b0fe7d0e284c63a3b291609 AS builder
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
