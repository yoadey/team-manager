FROM golang:1.26@sha256:079e59808d2d252516e27e3f3a9c003740dee7f75e55aa71528766d52bcfc16a AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

FROM gcr.io/distroless/static-debian12@sha256:9c346e4be81b5ca7ff31a0d89eaeade58b0f95cfd3baed1f36083ddb47ca3160
COPY --from=builder /server /server
COPY --from=builder /app/internal/db/migrations /internal/db/migrations
ENV MIGRATIONS_DIR=/internal/db/migrations
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
