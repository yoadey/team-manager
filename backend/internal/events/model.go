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
	// ExcludeFromStats removes this event from every attendance-statistics
	// computation while leaving it otherwise fully functional (RSVP,
	// comments, notifications, cancellation are all unaffected).
	ExcludeFromStats bool
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
	// ExcludeFromStats is the template value each generated occurrence's own
	// EventRow.ExcludeFromStats is seeded from (see CancelLeadMinutes).
	ExcludeFromStats bool
	CreatedAt        time.Time
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

// AttendanceEnriched is a roster row (one per current team member, or --
// for a cross-team event -- per distinct user across every targeted team's
// membership list) enriched with that member's effective attendance for one
// event -- an explicit SetAttendance/SetNomination record if one exists,
// otherwise the result of applying opt_out/absence-based defaulting (see
// computeEffectiveAttendance). MembershipId/Group/Title/PrimaryRole describe
// whichever single membership was picked to represent this user (the
// viewer's own team's membership when the user has one, else an arbitrary
// targeted team's) -- see Repository.ListAttendance. The display-rule team
// badge is computed separately by Service.ListAttendance/
// resolveCrossTeamBadgeContext from Repository.ListEventMemberTeams (the
// full per-user membership-across-targets picture), not from anything on
// this struct.
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
	Title            *string
	// Auto is true when Status was derived from opt_out/absence-based
	// defaulting rather than an explicit attendance record.
	Auto bool
	// Absent is true when the member has a planned absence covering the
	// event's date -- set regardless of Auto, since a member can explicitly
	// respond and still have a later-logged overlapping absence.
	Absent      bool
	PrimaryRole *teams.RoleRow
}

// EventTeamRow is one team an event targets (id + name), or -- when
// returned from Repository.ListEventMemberTeams -- one of the targeted
// teams a specific attendee belongs to. The shape is identical for both
// uses.
type EventTeamRow struct {
	TeamID   uuid.UUID
	TeamName string
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
	// CrossTeamIds are additional teams (besides the owning team the event
	// is created under) this event targets. Every generated event/series
	// occurrence gets one event_teams row per team in {owning} ∪
	// CrossTeamIds (see Repository.CreateEvent/CreateSeries). Nil/empty
	// creates a normal single-team event.
	CrossTeamIds []uuid.UUID
	Recurring    bool
	RepeatWeeks  int
	// RepeatEndDate, when set, takes precedence over RepeatWeeks: the
	// repository generates weekly occurrences from Date up to and
	// including RepeatEndDate instead of a fixed count.
	RepeatEndDate *time.Time
	// CancelLeadMinutes is the optional lead-time-based response cutoff,
	// set on the created event (or seeded onto every occurrence of a
	// created series).
	CancelLeadMinutes *int
	// ExcludeFromStats is set on the created event (or the created series'
	// template, seeded onto every generated occurrence).
	ExcludeFromStats bool
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
	// CrossTeamIds, when non-nil, replaces the full set of additional
	// target teams (besides the owning team). An explicit empty (non-nil)
	// slice un-shares the event back to single-team; nil means "not
	// provided in this patch" -- leave the current target set unchanged.
	// Mirrors NominatedRoleIds' identical nil-vs-empty convention.
	CrossTeamIds      []uuid.UUID
	CancelLeadMinutes *int
	ExcludeFromStats  *bool
}
