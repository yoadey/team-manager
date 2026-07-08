package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserRow mirrors the DB users table.
type UserRow struct {
	Id           uuid.UUID
	Name         string
	Email        string
	Phone        *string
	AvatarColor  string
	HasPhoto     bool
	Birthday     *time.Time
	Address      *string
	PasswordHash string
	CreatedAt    time.Time
}

// SessionRow mirrors the DB sessions table.
type SessionRow struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	TokenHash string
	Provider  string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Claims are the JWT claims used by this service (RS256).
type Claims struct {
	jwt.RegisteredClaims
	UserId string `json:"uid"`
}

// contextKey is an unexported type for context keys in this package.
type contextKey string

// userContextKey is the key under which *UserRow is stored in request context.
const userContextKey contextKey = "auth_user"
