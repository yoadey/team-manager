package events

import (
	"time"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/teams"
)

// EventRow mirrors the events DB table.
type EventRow struct {
	Id       uuid.UUID
	TeamId   uuid.UUID
	SeriesId *uuid.UUID
	Type     string
	Title    string
	Date     time.Time
	// EndDate is the optional last day of a multi-day span: when set, the
	// event occurs on every calendar day from Date through EndDate
	// inclusive. Only ever set on non-recurring events (SeriesId nil).
	EndDate           *time.Time
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
	// CancelLeadMinutes is the optional cutoff, expressed as minutes before
	// the event's start (see EventStartInstant), after which a
	// non-privileged member can no longer change their attendance response
	// for this event (see Service.SetAttendance).
	CancelLeadMinutes *int
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
	// RepeatEndDate is the alternative to RepeatWeeks: when set, the series
	// was originally defined by an end date rather than a fixed occurrence
	// count (RepeatWeeks still holds the derived occurrence count either
	// way -- see Repository.CreateSeries).
	RepeatEndDate *time.Time
	// CancelLeadMinutes is the template value each generated occurrence's
	// own EventRow.CancelLeadMinutes is seeded from.
	CancelLeadMinutes *int
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
	Id                 uuid.UUID
	EventId            uuid.UUID
	UserId             uuid.UUID
	Text               string
	CreatedAt          time.Time
	ActorName          *string
	ActorColor         *string
	HasActorPhoto      *bool
	AuthorMembershipId *uuid.UUID
}

// AttendanceEnriched is a roster row (one per current team member) enriched
// with that member's effective attendance for one event -- an explicit
// SetAttendance/SetNomination record if one exists, otherwise the result of
// applying opt_out/absence-based defaulting (see computeEffectiveAttendance).
type AttendanceEnriched struct {
	UserId           uuid.UUID
	MembershipId     uuid.UUID
	Status           string
	Reason           *string
	ReasonId         *string
	ReasonVisibility *string
	At               *time.Time
	Name             string
	AvatarColor      string
	HasPhoto         bool
	Group            *string
	// Auto is true when Status was derived from opt_out/absence-based
	// defaulting rather than an explicit attendance record.
	Auto bool
	// Absent is true when the member has a planned absence covering the
	// event's date -- set regardless of Auto, since a member can explicitly
	// respond and still have a later-logged overlapping absence.
	Absent      bool
	PrimaryRole *teams.RoleRow
}

// EffectiveAttendance is the resolved attendance state for a single
// (event, member) pair -- the same defaulting AttendanceEnriched carries,
// without the roster display fields, for the single-member "my attendance"
// read paths (enrichEvent/ListEvents's myStatus/myAuto/myReason).
type EffectiveAttendance struct {
	Status           string
	Reason           *string
	ReasonId         *string
	ReasonVisibility *string
	At               *time.Time
	Auto             bool
	Absent           bool
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
	Type  string
	Title string
	Date  time.Time
	// EndDate is the optional last day of a multi-day span (see
	// EventRow.EndDate). Only valid when Recurring is false.
	EndDate           *time.Time
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
	// RepeatEndDate, when set, takes precedence over RepeatWeeks: the
	// repository generates weekly occurrences from Date up to and
	// including RepeatEndDate instead of a fixed count.
	RepeatEndDate *time.Time
	// CancelLeadMinutes is the optional lead-time-based response cutoff,
	// set on the created event (or seeded onto every occurrence of a
	// created series).
	CancelLeadMinutes *int
}

// UpdateEventParams holds the fields used to update an event.
type UpdateEventParams struct {
	Type  *string
	Title *string
	Date  *time.Time
	// EndDate is the optional last day of a multi-day span (see
	// EventRow.EndDate). nil means "not provided in this patch," matching
	// every other optional field in this struct. Mutually exclusive with
	// ClearEndDate (validated in handler.go) -- to explicitly clear an
	// already-set EndDate back to NULL, set ClearEndDate instead, since a
	// plain *time.Time can't distinguish "not provided" from "clear".
	EndDate           *time.Time
	ClearEndDate      bool
	Location          *string
	Note              *string
	MeetTime          *string
	StartTime         *string
	EndTime           *string
	MeetTimeMandatory *bool
	ResponseMode      *string
	NominatedRoleIds  []uuid.UUID
	CancelLeadMinutes *int
}
