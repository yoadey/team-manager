FROM golang:1.26@sha256:f96cc555eb8db430159a3aa6797cd5bae561945b7b0fe7d0e284c63a3b291609 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

FROM gcr.io/distroless/static-debian12@sha256:22fd79fd75eab2372585b44517f8a094349938919dc613aafc37e4bdc9967c82
COPY --from=builder /server /server
COPY --from=builder /app/internal/db/migrations /internal/db/migrations
ENV MIGRATIONS_DIR=/internal/db/migrations
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
