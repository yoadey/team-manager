// Package calendarshare implements read-only, redacted cross-team calendar
// visibility: a team ("owner") can grant another team ("viewer") a
// schedule-only view of its events -- time, location, title, and type,
// never attendance, participants, comments, or notes. See design.md under
// openspec/changes/shared-team-calendar for the full design.
package calendarshare

import (
	"time"

	"github.com/google/uuid"
)

// ShareRow is one calendar_shares grant, enriched with the counterpart
// team's display name for the API response. Which team TeamId/TeamName
// refer to depends on the listing direction: Repository.ListGrantedByOwner
// returns the viewer team per row; Repository.ListGrantedToViewer returns
// the owner team per row.
type ShareRow struct {
	TeamId    uuid.UUID
	TeamName  string
	CreatedAt time.Time
}

// RedactedEventRow is the scheduling-only projection of an event exposed
// through a calendar share -- selected by its own dedicated query
// (Repository.ListRedactedEvents) rather than derived from the full event
// serializer, so a new sensitive event field can't leak into a shared
// calendar just by existing on the events table.
type RedactedEventRow struct {
	Id        uuid.UUID
	Type      string
	Title     string
	Date      time.Time
	StartTime *string
	EndTime   *string
	Location  *string
}
