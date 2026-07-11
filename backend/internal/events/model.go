package events

import (
	"time"

	"github.com/google/uuid"
)

// EventRow mirrors the events DB table.
type EventRow struct {
	Id                uuid.UUID
	TeamId            uuid.UUID
	SeriesId          *uuid.UUID
	Type              string
	Title             string
	Date              time.Time
	Location          *string
	Note              *string
	Result            *string
	MeetTime          *string
	StartTime         *string
	EndTime           *string
	MeetTimeMandatory *bool
	ResponseMode      *string
	NominatedRoleIds  []uuid.UUID
	Status            string
	CreatedAt         time.Time
}

// EventSeriesRow mirrors the event_series DB table.
type EventSeriesRow struct {
	Id                uuid.UUID
	TeamId            uuid.UUID
	Type              string
	Title             string
	Location          *string
	Note              *string
	MeetTime          *string
	StartTime         *string
	EndTime           *string
	MeetTimeMandatory *bool
	ResponseMode      *string
	NominatedRoleIds  []uuid.UUID
	RepeatWeeks       int
	CreatedAt         time.Time
}

// AttendanceDBRow is the DB representation of the attendance table.
type AttendanceDBRow struct {
	Id               uuid.UUID
	EventId          uuid.UUID
	UserId           uuid.UUID
	Status           string
	Reason           *string
	ReasonId         *string
	ReasonVisibility *string
	At               *time.Time
}

// CommentRow matches event_comments enriched with user fields.
type CommentRow struct {
	Id            uuid.UUID
	EventId       uuid.UUID
	UserId        uuid.UUID
	Text          string
	CreatedAt     time.Time
	ActorName     *string
	ActorColor    *string
	HasActorPhoto *bool
}

// AttendanceEnriched is the result of a JOIN between attendance and users tables.
type AttendanceEnriched struct {
	UserId           uuid.UUID
	Status           string
	Reason           *string
	ReasonId         *string
	ReasonVisibility *string
	At               *time.Time
	Name             string
	AvatarColor      string
	HasPhoto         bool
}

// EventSummaryData holds aggregated attendance counts for an event.
// EventSummaryData holds aggregated attendance counts for an event.
//
// Known, accepted limitation: these counts reflect each attendance row's
// stored status at the time it was recorded, not a live re-evaluation
// against the event's current NominatedRoleIds. UpdateEvent validates a new
// nominated-role set but never reconciles existing attendance rows against
// it, so if an organizer changes which roles are nominated after members
// have already responded, Nominated/Yes/No/Maybe keep counting those
// now-irrelevant responses, and newly-eligible members aren't reflected
// until they individually respond. Reconciling on every nomination change
// would mean either bulk-flipping other members' already-recorded answers
// (silently destroying real user input) or a larger read-path change to
// compute eligibility live rather than trusting the stored status -- judged
// disproportionate to how rarely nominated roles change after responses
// have started coming in.
type EventSummaryData struct {
	Yes          int
	No           int
	Maybe        int
	Pending      int
	NotNominated int
	Nominated    int
	Total        int
}

// CreateEventParams holds the fields used to create a new event or series.
type CreateEventParams struct {
	Type              string
	Title             string
	Date              time.Time
	Location          *string
	Note              *string
	MeetTime          *string
	StartTime         *string
	EndTime           *string
	MeetTimeMandatory *bool
	ResponseMode      *string
	NominatedRoleIds  []uuid.UUID
	Recurring         bool
	RepeatWeeks       int
}

// UpdateEventParams holds the fields used to update an event.
type UpdateEventParams struct {
	Type              *string
	Title             *string
	Date              *time.Time
	Location          *string
	Note              *string
	MeetTime          *string
	StartTime         *string
	EndTime           *string
	MeetTimeMandatory *bool
	ResponseMode      *string
	NominatedRoleIds  []uuid.UUID
}
