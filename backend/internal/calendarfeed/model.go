package calendarfeed

import (
	"time"

	"github.com/google/uuid"
)

// TokenRow mirrors the calendar_feed_tokens DB table.
type TokenRow struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	TeamId    uuid.UUID
	Token     string
	CreatedAt time.Time
	RevokedAt *time.Time
}
