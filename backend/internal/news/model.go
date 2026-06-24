package news

import (
	"time"

	"github.com/google/uuid"
)

// NewsRow mirrors the DB news table.
type NewsRow struct {
	Id        uuid.UUID
	TeamId    uuid.UUID
	AuthorId  uuid.UUID
	Title     string
	Body      string
	Pinned    bool
	CreatedAt time.Time
	// Joined from users
	AuthorName  *string
	AuthorColor *string
	PhotoData   []byte
}
