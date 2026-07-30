package calendarfeed

import (
	"time"

	"github.com/google/uuid"
)

// TokenRow mirrors the calendar_feed_tokens DB table.
type TokenRow struct {
	Id               uuid.UUID
	UserId           uuid.UUID
	TeamId           uuid.UUID
	Token            string
	CreatedAt        time.Time
	RevokedAt        *time.Time
	Types            []string
	IncludeBirthdays bool
}

// defaultFeedTypes is the content selection a newly-issued token starts
// with -- every event type, matching this package's pre-selection behavior
// before content became configurable.
var defaultFeedTypes = []string{"training", "auftritt", "event"}
