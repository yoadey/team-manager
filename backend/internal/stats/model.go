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
	Type    string
	Date    string // YYYY-MM-DD
	Yes     int
	Counted int
}

// MatrixColumnRow is one event column of the attendance matrix.
type MatrixColumnRow struct {
	EventID uuid.UUID
	Title   string
	Type    string
	Date    string // YYYY-MM-DD
}

// MatrixCellRow is one member's effective status for one event. EventID is nil
// for a member who has no events in range (a LEFT JOIN placeholder row that
// still lets the member appear as an empty matrix row).
type MatrixCellRow struct {
	UserID      uuid.UUID
	Name        string
	AvatarColor string
	HasPhoto    bool
	EventID     *uuid.UUID
	Eff         string // yes | no | maybe | pending
}

// AbsenceRow holds one member's absence (effective status "no") from one
// event, for the absence table view.
type AbsenceRow struct {
	UserID     uuid.UUID
	Name       string
	EventID    uuid.UUID
	EventTitle string
	Date       string // YYYY-MM-DD
}
