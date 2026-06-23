package stats

import "github.com/google/uuid"

// MemberStatRow holds the raw attendance aggregation for one member.
type MemberStatRow struct {
	UserID      uuid.UUID
	Name        string
	AvatarColor string
	HasPhoto    bool
	Yes         int
	Counted     int
}

// EventStatRow holds per-event attendance counts.
type EventStatRow struct {
	EventID uuid.UUID
	Title   string
	Date    string // YYYY-MM-DD
	Yes     int
	Counted int
}
