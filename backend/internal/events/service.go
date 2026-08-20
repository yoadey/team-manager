package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// Sentinel errors for the events package.
var (
	ErrCreateEventNilBody      = errors.New("events.Service.CreateEvent: nil body")
	ErrCreateEventNoRow        = errors.New("events.Service.CreateEvent: no row returned")
	ErrUpdateEventNilBody      = errors.New("events.Service.UpdateEvent: nil body")
	ErrInvalidNominatedRoleIDs = errors.New("nominated_role_ids contain roles not belonging to this team")
	// ErrInvalidCrossTeamIds is returned when crossTeamIds contains an id
	// that isn't a real team, mirroring ErrInvalidNominatedRoleIDs'
	// identical existence-check pattern.
	ErrInvalidCrossTeamIds = errors.New("crossTeamIds must refer to teams that exist")
	// ErrCrossTeamWriteForbidden is returned when the caller lacks
	// events:write in one or more of the teams a create/update would target
	// (see design.md's "Create restricted to write-in-all-targets").
	ErrCrossTeamWriteForbidden      = errors.New("events.Service: caller lacks events:write in one or more targeted teams")
	ErrSetAttendanceForbidden       = errors.New("events.Service.SetAttendance: caller may not set another member's attendance")
	ErrSetNominationForbidden       = errors.New("events.Service.SetNomination: caller lacks events:write")
	ErrAttendanceStatusNotNominated = errors.New("events.Service.SetAttendance: status 'not_nominated' may only be set via SetNomination")
	ErrEventCancelled               = errors.New("events.Service.SetAttendance: cannot change attendance on a cancelled event")
	ErrRepeatWeeksTooLarge          = fmt.Errorf("repeat_weeks must be between 1 and %d", maxRepeatWeeks)
	ErrTooManyComments              = fmt.Errorf("event has reached the maximum of %d comments", maxCommentsPerEvent)
	// ErrRecurrenceEndDateBeforeDate is returned when a recurring series'
	// endDate is set but falls before the series' own start date -- the
	// end-date alternative to repeatWeeks (see CreateEvent) has no
	// occurrences to generate in that case.
	ErrRecurrenceEndDateBeforeDate = errors.New("endDate must be on or after date")
	// ErrCancelLeadTimePassed is returned when a member without a role
	// permitting late responses (events:write) attempts to change their
	// attendance after the event's cancelLeadMinutes-derived cutoff
	// (EventStartInstant - cancelLeadMinutes) has passed.
	ErrCancelLeadTimePassed = errors.New("events.Service.SetAttendance: cancellation lead time has passed")
	// ErrMultiDayEndDateOnRecurringEvent is returned when a create request
	// sets both recurring: true and multiDayEndDate -- see design.md's
	// "Mutually exclusive with recurring" decision.
	ErrMultiDayEndDateOnRecurringEvent = errors.New("multiDayEndDate: cannot be set on a recurring event")
	// ErrMultiDaySpanTooLong is returned when multiDayEndDate is set but the
	// resulting span exceeds maxMultiDaySpanDays.
	ErrMultiDaySpanTooLong = fmt.Errorf("multiDayEndDate: span must not exceed %d days", maxMultiDaySpanDays)
)

// maxRepeatWeeks caps how many events a single recurring series may create.
// The OpenAPI spec only declares a minimum, and no request-schema validator
// is wired into the router, so this is the only enforcement point; without
// it, CreateSeries would loop an attacker-controlled number of times inside
// one DB transaction.
const maxRepeatWeeks = 104

// maxMultiDaySpanDays caps how far apart date/multiDayEndDate may be,
// mirroring absences' identical maxAbsenceSpanDays cap
// (internal/absences/handler.go) and its DB-level backstop
// (events_multiday_span_within_limit, migration 00025): without a bound, an
// arbitrarily large span would make every calendar render -- which expands
// the event across every day it covers (frontend's groupEventsByDate) --
// do unbounded work for a single event.
const maxMultiDaySpanDays = 1095 // ~3 years

// eventRepo is the interface the Service relies on.
type eventRepo interface {
	ListEvents(ctx context.Context, teamID string, scope gen.ListEventsParamsScope, limit int, cur *ListCursor) ([]EventRow, error)
	GetEvent(ctx context.Context, eventID, teamID string) (*EventRow, error)
	CreateEvent(ctx context.Context, teamID string, params *CreateEventParams) (*EventRow, error)
	CreateSeries(ctx context.Context, teamID string, params *CreateEventParams) ([]EventRow, error)
	UpdateEvent(ctx context.Context, eventID, teamID string, params *UpdateEventParams, scope string) (*EventRow, error)
	SetStatus(ctx context.Context, eventID, teamID, status, scope string) (*EventRow, error)
	DeleteEvent(ctx context.Context, eventID, teamID string, scope string) error
	GetAttendanceSummary(ctx context.Context, eventID, teamID string) (EventSummaryData, error)
	GetMyEffectiveAttendance(ctx context.Context, eventID, userID, teamID string) (*EffectiveAttendance, error)
	GetAttendanceSummaries(ctx context.Context, eventIDs []uuid.UUID, teamID string) (map[uuid.UUID]EventSummaryData, error)
	// TeamsExist reports whether every id in teamIDs is a real team --
	// used to validate crossTeamIds on create/update before the more
	// expensive per-team permission check (requireEventsWriteInTeams).
	TeamsExist(ctx context.Context, teamIDs []uuid.UUID) (bool, error)
	// GetEventTeams returns every team an event targets (owning team plus
	// crossTeamIds), each with its name.
	GetEventTeams(ctx context.Context, eventID string) ([]EventTeamRow, error)
	// ListEventTeamsBatch is GetEventTeams' batched counterpart, keyed by
	// event ID -- used by ListEvents to avoid one query per event.
	ListEventTeamsBatch(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID][]EventTeamRow, error)
	// ListEventMemberTeams returns, per user, the subset of an event's
	// targeted teams they belong to -- the raw material ListAttendance uses
	// to compute each attendee's team badge.
	ListEventMemberTeams(ctx context.Context, eventID string) (map[uuid.UUID][]EventTeamRow, error)
	GetMyEffectiveAttendances(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]EffectiveAttendance, error)
	ListAttendance(ctx context.Context, eventID, teamID string) ([]AttendanceEnriched, error)
	GetReasonVisibilityContext(ctx context.Context, teamID, viewerID string) (teamRoleIDs, viewerRoleIDs []string, err error)
	SetAttendance(ctx context.Context, eventID, callerID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*AttendanceDBRow, error)
	SetNomination(ctx context.Context, eventID, callerID, userID, teamID string, nominated bool) error
	ListComments(ctx context.Context, eventID, teamID string, limit int, cur *CommentCursor) ([]CommentRow, error)
	CountComments(ctx context.Context, eventID, teamID string) (int, error)
	AddComment(ctx context.Context, eventID, userID, teamID, text string) (*CommentRow, error)
	DeleteComment(ctx context.Context, commentID, userID, teamID string) error
}

// jobEnqueuer is satisfied by *jobs.Client.
type jobEnqueuer interface {
	EnqueueNotification(ctx context.Context, args jobs.NotificationArgs) error
}

// teamRoleChecker verifies that a set of role IDs all belong to a given team.
// Implemented by *roles.Repository.
type teamRoleChecker interface {
	RolesExistForTeam(ctx context.Context, teamID string, roleIDs []uuid.UUID) (bool, error)
}

// permissionChecker returns a user's effective RBAC permissions for a team.
// Implemented by *members.Repository.
type permissionChecker interface {
	GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error)
}

// Service implements event business logic.
type Service struct {
	repo        eventRepo
	jobs        jobEnqueuer
	pager       *pagination.Paginator
	roleChecker teamRoleChecker
	permChecker permissionChecker
	logger      *slog.Logger
}

// NewService creates a new Service. pager may be nil (uses default Paginator).
// roleChecker may be nil; when set, nominated_role_ids are validated to belong
// to the event's team before any create or update is persisted. permChecker
// may be nil in tests that don't exercise SetAttendance; production callers
// must supply it so that setting another member's attendance requires
// events:write (see SetAttendance).
func NewService(repo eventRepo, enq jobEnqueuer, pager *pagination.Paginator, roleChecker teamRoleChecker, permChecker permissionChecker, logger *slog.Logger) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, jobs: enq, pager: pager, roleChecker: roleChecker, permChecker: permChecker, logger: logger}
}

// validateNominatedRoles checks that all provided role IDs belong to teamID.
func (s *Service) validateNominatedRoles(ctx context.Context, teamID string, roleIDs []uuid.UUID) error {
	if s.roleChecker == nil || len(roleIDs) == 0 {
		return nil
	}
	ok, err := s.roleChecker.RolesExistForTeam(ctx, teamID, roleIDs)
	if err != nil {
		return fmt.Errorf("events: validate nominated roles: %w", err)
	}
	if !ok {
		return ErrInvalidNominatedRoleIDs
	}
	return nil
}

// ─── ListEvents ─────────────────────────────────────────────────────────────

// ListEvents returns a keyset page of events (enriched with attendance summary
// and the user's status) plus the cursor for the next page (nil on the last
// page). cursor is the opaque token from a prior page ("" = first page).
func (s *Service) ListEvents(ctx context.Context, teamID, userID string, scope gen.ListEventsParamsScope, cursor string, limit int) ([]gen.TeamEvent, *string, error) {
	var cur *ListCursor
	var decoded ListCursor
	if ok, err := s.pager.Decode(cursor, &decoded); err != nil {
		return nil, nil, fmt.Errorf("events.Service.ListEvents: %w", err)
	} else if ok {
		cur = &decoded
	}

	rows, err := s.repo.ListEvents(ctx, teamID, scope, limit+1, cur)
	if err != nil {
		return nil, nil, fmt.Errorf("events.Service.ListEvents: %w", err)
	}

	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		token, err := s.pager.Encode(ListCursor{Date: last.Date, ID: last.Id})
		if err != nil {
			return nil, nil, fmt.Errorf("events.Service.ListEvents: %w", err)
		}
		next = &token
	}

	eventIDs := make([]uuid.UUID, len(rows))
	for i := range rows {
		eventIDs[i] = rows[i].Id
	}
	summaries, err := s.repo.GetAttendanceSummaries(ctx, eventIDs, teamID)
	if err != nil {
		return nil, nil, fmt.Errorf("events.Service.ListEvents: %w", err)
	}
	eventTeams, err := s.repo.ListEventTeamsBatch(ctx, eventIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("events.Service.ListEvents: %w", err)
	}
	var myAttendances map[uuid.UUID]EffectiveAttendance
	if userID != "" {
		myAttendances, err = s.repo.GetMyEffectiveAttendances(ctx, eventIDs, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("events.Service.ListEvents: %w", err)
		}
	}

	out := make([]gen.TeamEvent, 0, len(rows))
	for i := range rows {
		ev := toGenEvent(&rows[i], summaries[rows[i].Id])
		ev.CrossTeamIds = crossTeamIDsFrom(rows[i].TeamId, eventTeams[rows[i].Id])
		if myAtt, ok := myAttendances[rows[i].Id]; ok {
			st := gen.AttendanceStatus(myAtt.Status)
			ev.MyStatus = &st
			ev.MyReason = myAtt.Reason
			ev.MyAuto = &myAtt.Auto
		}
		out = append(out, ev)
	}
	return out, next, nil
}

// ─── GetEvent ───────────────────────────────────────────────────────────────

// GetEvent retrieves a single event by ID enriched with summary and user status.
func (s *Service) GetEvent(ctx context.Context, teamID, userID, eventID string) (*gen.TeamEvent, error) {
	row, err := s.repo.GetEvent(ctx, eventID, teamID)
	if err != nil {
		return nil, fmt.Errorf("events.Service.GetEvent: %w", err)
	}

	ev, err := s.enrichEvent(ctx, row, userID, teamID)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

// ─── CreateEvent ────────────────────────────────────────────────────────────

// CreateEvent creates a single event or a recurring series.
// For recurring events, it returns the first event in the series.
func (s *Service) CreateEvent(ctx context.Context, teamID, userID string, body *gen.CreateEventJSONRequestBody) (*gen.TeamEvent, error) { //nolint:gocognit,cyclop // complexity inherent in event creation business logic
	if body == nil {
		return nil, ErrCreateEventNilBody
	}

	recurring := body.Recurring != nil && *body.Recurring
	if body.MultiDayEndDate != nil {
		if recurring {
			return nil, ErrMultiDayEndDateOnRecurringEvent
		}
		if body.MultiDayEndDate.Before(body.Date.Time) {
			return nil, ErrMultiDayEndDateBeforeDate
		}
		if body.MultiDayEndDate.Sub(body.Date.Time) > maxMultiDaySpanDays*24*time.Hour {
			return nil, ErrMultiDaySpanTooLong
		}
	}
	repeatWeeks := 1
	if body.RepeatWeeks != nil {
		repeatWeeks = *body.RepeatWeeks
	}
	// endDate is the alternative to repeatWeeks: when both a recurring
	// series and an endDate are given, occurrences are derived weekly from
	// date up to and including endDate instead of using repeatWeeks as a
	// fixed count -- see design.md's "Deferred item scoping" for the
	// recurrence-end-date entry. Only meaningful for a recurring series;
	// endDate on a non-recurring create is silently ignored, matching how
	// repeatWeeks is already ignored for a non-recurring create.
	var repeatEndDate *time.Time
	if recurring && body.EndDate != nil {
		end := body.EndDate.Time
		if end.Before(body.Date.Time) {
			return nil, ErrRecurrenceEndDateBeforeDate
		}
		weeks := int(end.Sub(body.Date.Time).Hours()/(24*7)) + 1
		if weeks > maxRepeatWeeks {
			return nil, ErrRepeatWeeksTooLarge
		}
		repeatEndDate = &end
	} else if repeatWeeks < 1 || repeatWeeks > maxRepeatWeeks {
		return nil, ErrRepeatWeeksTooLarge
	}

	var multiDayEndDate *time.Time
	if body.MultiDayEndDate != nil {
		d := body.MultiDayEndDate.Time
		multiDayEndDate = &d
	}
	params := CreateEventParams{
		Type:              string(body.Type),
		Title:             body.Title,
		Date:              body.Date.Time,
		EndDate:           multiDayEndDate,
		Location:          body.Location,
		Note:              body.Note,
		MeetTime:          body.MeetTime,
		StartTime:         body.StartTime,
		EndTime:           body.EndTime,
		MeetTimeMandatory: body.MeetTimeMandatory,
		Recurring:         recurring,
		RepeatWeeks:       repeatWeeks,
		RepeatEndDate:     repeatEndDate,
		CancelLeadMinutes: body.CancelLeadMinutes,
	}
	if body.ResponseMode != nil {
		rm := string(*body.ResponseMode)
		params.ResponseMode = &rm
	}
	if body.NominatedRoleIds != nil {
		params.NominatedRoleIds = *body.NominatedRoleIds
	}
	if body.ExcludeFromStats != nil {
		params.ExcludeFromStats = *body.ExcludeFromStats
	}
	if body.CrossTeamIds != nil {
		params.CrossTeamIds = *body.CrossTeamIds
	}

	if err := s.validateNominatedRoles(ctx, teamID, params.NominatedRoleIds); err != nil {
		return nil, err
	}
	// The URL's teamId (the owning team) already had events:write validated
	// by RequirePermission before this handler ran; crossTeamIds names
	// *additional* teams, so every one of them needs the same check here.
	if err := s.validateCrossTeamIds(ctx, userID, params.CrossTeamIds); err != nil {
		return nil, err
	}

	var row *EventRow
	if recurring {
		rows, err := s.repo.CreateSeries(ctx, teamID, &params)
		if err != nil {
			return nil, fmt.Errorf("events.Service.CreateEvent(series): %w", err)
		}
		if len(rows) > 0 {
			row = &rows[0]
		}
	} else {
		var err error
		row, err = s.repo.CreateEvent(ctx, teamID, &params)
		if err != nil {
			return nil, fmt.Errorf("events.Service.CreateEvent: %w", err)
		}
	}

	if row == nil {
		return nil, ErrCreateEventNoRow
	}

	// Enqueue notification (best-effort; ignore error so it doesn't fail the request).
	if s.jobs != nil {
		if teamUUID, err2 := uuid.Parse(teamID); err2 == nil {
			if actorUUID, err2 := uuid.Parse(userID); err2 == nil {
				evID := row.Id
				evTitle := row.Title
				evDate := row.Date
				if err := s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
					TeamID:     teamUUID,
					Type:       "event_created",
					ActorID:    actorUUID,
					EventID:    &evID,
					EventTitle: &evTitle,
					EventDate:  &evDate,
				}); err != nil {
					s.logger.Warn("events: failed to enqueue notification", slog.String("eventId", evID.String()), slog.String("type", "event_created"), slog.String("error", err.Error()))
				}
			}
		}
	}

	return s.enrichEventOrFallback(ctx, row, userID, teamID), nil
}

// ─── UpdateEvent ────────────────────────────────────────────────────────────

// UpdateEvent updates an event (or series) and returns the updated event.
func (s *Service) UpdateEvent(ctx context.Context, teamID, userID, eventID, scope string, body *gen.UpdateEventJSONRequestBody) (*gen.TeamEvent, error) {
	if body == nil {
		return nil, ErrUpdateEventNilBody
	}

	params := UpdateEventParams{
		Title:             body.Title,
		Location:          body.Location,
		Note:              body.Note,
		MeetTime:          body.MeetTime,
		StartTime:         body.StartTime,
		EndTime:           body.EndTime,
		MeetTimeMandatory: body.MeetTimeMandatory,
		CancelLeadMinutes: body.CancelLeadMinutes,
		ExcludeFromStats:  body.ExcludeFromStats,
	}
	if body.Type != nil {
		t := string(*body.Type)
		params.Type = &t
	}
	if body.Date != nil {
		d := body.Date.Time
		params.Date = &d
	}
	if body.MultiDayEndDate != nil {
		d := body.MultiDayEndDate.Time
		params.EndDate = &d
	}
	if body.ClearMultiDayEndDate != nil && *body.ClearMultiDayEndDate {
		params.ClearEndDate = true
	}
	if body.ResponseMode != nil {
		rm := string(*body.ResponseMode)
		params.ResponseMode = &rm
	}
	if body.NominatedRoleIds != nil {
		// Direct assignment (not append onto the nil zero value) so an
		// explicit empty array ("clear all nominations") stays a non-nil
		// empty slice -- append(nil, emptySlice...) returns nil per Go's
		// append semantics, which buildUpdateSets' `!= nil` check would then
		// read as "field not provided," silently no-op'ing the clear.
		params.NominatedRoleIds = *body.NominatedRoleIds
	}
	if body.CrossTeamIds != nil {
		// Same nil-vs-non-nil-empty convention as NominatedRoleIds above:
		// direct assignment so an explicit [] (un-share back to single-team)
		// stays a non-nil empty slice, distinguishable from "not provided".
		params.CrossTeamIds = *body.CrossTeamIds
	}

	if err := s.validateNominatedRoles(ctx, teamID, params.NominatedRoleIds); err != nil {
		return nil, err
	}
	// crossTeamIds present in the patch replaces the full additional-target
	// set; the URL's teamId already has events:write validated by
	// RequirePermission, so only the (possibly new) crossTeamIds need
	// checking here -- see design.md's "same all-targets-write check across
	// the FULL resulting target set".
	if body.CrossTeamIds != nil {
		if err := s.validateCrossTeamIds(ctx, userID, params.CrossTeamIds); err != nil {
			return nil, err
		}
	}

	row, err := s.repo.UpdateEvent(ctx, eventID, teamID, &params, scope)
	if err != nil {
		return nil, fmt.Errorf("events.Service.UpdateEvent: %w", err)
	}

	return s.enrichEventOrFallback(ctx, row, userID, teamID), nil
}

// ─── DeleteEvent ────────────────────────────────────────────────────────────

// DeleteEvent deletes an event or series scoped to the given teamID.
func (s *Service) DeleteEvent(ctx context.Context, eventID, teamID, scope string) error {
	if err := s.repo.DeleteEvent(ctx, eventID, teamID, scope); err != nil {
		return fmt.Errorf("events.Service.DeleteEvent: %w", err)
	}
	return nil
}

// ─── SetStatus ──────────────────────────────────────────────────────────────

// SetStatus updates event status and returns the updated event.
func (s *Service) SetStatus(ctx context.Context, userID, eventID, teamID, status, scope string) (*gen.TeamEvent, error) {
	row, err := s.repo.SetStatus(ctx, eventID, teamID, status, scope)
	if err != nil {
		return nil, fmt.Errorf("events.Service.SetStatus: %w", err)
	}

	// Enqueue cancellation notification (best-effort).
	if s.jobs != nil && status == "cancelled" {
		if actorUUID, err2 := uuid.Parse(userID); err2 == nil {
			evID := row.Id
			evTitle := row.Title
			evDate := row.Date
			if err := s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
				TeamID:     row.TeamId,
				Type:       "event_cancelled",
				ActorID:    actorUUID,
				EventID:    &evID,
				EventTitle: &evTitle,
				EventDate:  &evDate,
			}); err != nil {
				s.logger.Warn("events: failed to enqueue notification", slog.String("eventId", evID.String()), slog.String("type", "event_cancelled"), slog.String("error", err.Error()))
			}
		}
	}

	return s.enrichEventOrFallback(ctx, row, userID, teamID), nil
}

// ─── Comments ───────────────────────────────────────────────────────────────

// ListComments returns a keyset page of an event's comments (oldest-first) plus
// the cursor for the next page (nil on the last page). cursor is the opaque
// token from a prior page ("" = first page).
func (s *Service) ListComments(ctx context.Context, eventID, teamID string, limit int, cursor string) ([]gen.EventComment, *string, error) {
	var cur *CommentCursor
	var decoded CommentCursor
	if ok, err := s.pager.Decode(cursor, &decoded); err != nil {
		return nil, nil, fmt.Errorf("events.Service.ListComments: %w", err)
	} else if ok {
		cur = &decoded
	}

	// Fetch one extra row to detect whether a further page exists.
	rows, err := s.repo.ListComments(ctx, eventID, teamID, limit+1, cur)
	if err != nil {
		return nil, nil, fmt.Errorf("events.Service.ListComments: %w", err)
	}

	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		token, err := s.pager.Encode(CommentCursor{CreatedAt: last.CreatedAt, ID: last.Id})
		if err != nil {
			return nil, nil, fmt.Errorf("events.Service.ListComments: %w", err)
		}
		next = &token
	}

	out := make([]gen.EventComment, 0, len(rows))
	for _, c := range rows {
		out = append(out, toGenComment(&c))
	}
	return out, next, nil
}

// AddComment adds a comment to an event scoped to teamID. Returns
// ErrTooManyComments once the event has reached maxCommentsPerEvent --
// events/comments is a self-service write reachable by any team member with
// no RBAC gate and no other natural bound, unlike finances' write paths
// which already enforce an equivalent per-team cap (maxTransactionsPerTeam).
func (s *Service) AddComment(ctx context.Context, eventID, userID, teamID, text string) (*gen.EventComment, error) {
	count, err := s.repo.CountComments(ctx, eventID, teamID)
	if err != nil {
		return nil, fmt.Errorf("events.Service.AddComment: %w", err)
	}
	if count >= maxCommentsPerEvent {
		return nil, ErrTooManyComments
	}
	c, err := s.repo.AddComment(ctx, eventID, userID, teamID, text)
	if err != nil {
		return nil, fmt.Errorf("events.Service.AddComment: %w", err)
	}
	gc := toGenComment(c)
	return &gc, nil
}

// DeleteComment deletes a comment if the user owns it and it belongs to teamID.
func (s *Service) DeleteComment(ctx context.Context, commentID, userID, teamID string) error {
	if err := s.repo.DeleteComment(ctx, commentID, userID, teamID); err != nil {
		return fmt.Errorf("events.Service.DeleteComment: %w", err)
	}
	return nil
}

// ─── Attendance ─────────────────────────────────────────────────────────────

// ListAttendance returns all attendance rows for an event scoped to teamID.
// An attendance reason is only included for the viewer's own row, for rows
// the member explicitly marked reasonVisibility="team", or for viewers
// holding one of the team's reason-visibility roles — mirroring the
// frontend's canSeeReason gate, but enforced here so a member can't read a
// teammate's private reason by calling the API directly. Matches the
// RequirePermission middleware, which treats events/attendance as
// self-service (any member may read it), so this redaction is the only
// enforcement point for reason confidentiality. A nil/unset ReasonVisibility
// (e.g. rows written before the field existed) is treated the same as
// "trainers" -- the more restrictive default -- not as an implicit "team".
//
// This applies regardless of attendance status: SetAttendance places no
// restriction on which status a reason/reasonId/reasonVisibility may
// accompany (a "yes, but running late" reason is a legitimate use case), so
// gating redaction on status=="no" would let a private reason attached to
// any other status leak to every team member unredacted -- reason
// confidentiality has to be a property of the reason itself, not of the
// status it happens to be attached to.
func (s *Service) ListAttendance(ctx context.Context, eventID, teamID, viewerID string) ([]gen.AttendanceRow, error) {
	attendanceRows, err := s.repo.ListAttendance(ctx, eventID, teamID)
	if err != nil {
		return nil, fmt.Errorf("events.Service.ListAttendance: %w", err)
	}

	canSeeReasons, err := s.canViewerSeeAttendanceReasons(ctx, teamID, viewerID, attendanceRows)
	if err != nil {
		return nil, err
	}
	badges, err := s.resolveCrossTeamBadgeContext(ctx, eventID, teamID)
	if err != nil {
		return nil, err
	}

	out := make([]gen.AttendanceRow, 0, len(attendanceRows))
	for _, a := range attendanceRows {
		if a.UserId.String() != viewerID && !canSeeReasons && !reasonSharedWithTeam(a.ReasonVisibility) {
			a.Reason = nil
			a.ReasonId = nil
		}
		row := toGenAttendanceRow(&a)
		badges.applyBadge(&row, a.UserId)
		out = append(out, row)
	}
	return out, nil
}

// canViewerSeeAttendanceReasons decides whether viewerID may see other
// members' declined-attendance reasons on this roster (see ListAttendance's
// doc comment on the redaction rule it implements) -- split out purely to
// keep ListAttendance's cognitive complexity under the repo's
// golangci-lint threshold. The reason-visibility-role lookup is skipped
// entirely (canSeeReasons stays false) when no row on the roster actually
// has a reason to protect.
func (s *Service) canViewerSeeAttendanceReasons(ctx context.Context, teamID, viewerID string, attendanceRows []AttendanceEnriched) (bool, error) {
	needsRedactionCheck := false
	for _, a := range attendanceRows {
		if a.UserId.String() != viewerID && (a.Reason != nil || a.ReasonId != nil) && !reasonSharedWithTeam(a.ReasonVisibility) {
			needsRedactionCheck = true
			break
		}
	}
	if !needsRedactionCheck {
		return false, nil
	}
	teamRoleIDs, viewerRoleIDs, err := s.repo.GetReasonVisibilityContext(ctx, teamID, viewerID)
	if err != nil {
		return false, fmt.Errorf("events.Service.ListAttendance: %w", err)
	}
	return roleSetsIntersect(teamRoleIDs, viewerRoleIDs), nil
}

// crossTeamBadgeContext holds everything ListAttendance needs to compute
// (and apply) each roster row's team badge -- see computeTeamBadge for the
// display rule itself.
type crossTeamBadgeContext struct {
	// active is false for a single-team event: applyBadge is then a no-op,
	// keeping a single-team roster byte-for-byte on today's behavior.
	active       bool
	viewerTeamID uuid.UUID
	memberTeams  map[uuid.UUID][]EventTeamRow
}

// applyBadge sets row.TeamName and strips every profile-identifying or
// free-text field (per spec.md's "Merged attendance without profile
// access") when userID is a foreign attendee (not in the viewer's own
// team). A no-op for a single-team event, or when userID shares the
// viewer's own team.
func (c crossTeamBadgeContext) applyBadge(row *gen.AttendanceRow, userID uuid.UUID) {
	if !c.active {
		return
	}
	badge := computeTeamBadge(c.viewerTeamID, c.memberTeams[userID])
	if badge == nil {
		return
	}
	row.TeamName = badge
	row.MembershipId = nil
	row.Group = nil
	row.Title = nil
	row.PrimaryRole = nil
	row.Reason = nil
	row.ReasonId = nil
	row.ReasonVisibility = nil
}

// resolveCrossTeamBadgeContext only looks up per-user cross-team membership
// data (ListEventMemberTeams) when the event actually targets more than one
// team -- skipping that extra query entirely for a single-team event, so
// its roster stays on the exact same query shape/cost as before this
// feature. Split out of ListAttendance purely to keep that function's
// cognitive complexity under the repo's golangci-lint threshold.
func (s *Service) resolveCrossTeamBadgeContext(ctx context.Context, eventID, teamID string) (crossTeamBadgeContext, error) {
	eventTeams, err := s.repo.GetEventTeams(ctx, eventID)
	if err != nil {
		return crossTeamBadgeContext{}, fmt.Errorf("events.Service.ListAttendance: %w", err)
	}
	if len(eventTeams) <= 1 {
		return crossTeamBadgeContext{}, nil
	}
	memberTeams, err := s.repo.ListEventMemberTeams(ctx, eventID)
	if err != nil {
		return crossTeamBadgeContext{}, fmt.Errorf("events.Service.ListAttendance: %w", err)
	}
	viewerTeamID, err := uuid.Parse(teamID)
	if err != nil {
		return crossTeamBadgeContext{}, fmt.Errorf("events.Service.ListAttendance: parse teamID: %w", err)
	}
	return crossTeamBadgeContext{active: true, viewerTeamID: viewerTeamID, memberTeams: memberTeams}, nil
}

// reasonSharedWithTeam reports whether the declining member explicitly opted
// their decline reason into team-wide visibility, bypassing the
// reason-visibility-role check entirely for that row.
func reasonSharedWithTeam(reasonVisibility *string) bool {
	return reasonVisibility != nil && *reasonVisibility == "team"
}

// roleSetsIntersect reports whether any ID in a is also present in b.
func roleSetsIntersect(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[id]; ok {
			return true
		}
	}
	return false
}

// SetAttendance upserts an attendance record scoped to teamID. callerID is the
// authenticated user making the request; userID is the member the attendance
// row is being set for (may differ from callerID). Setting another member's
// attendance requires events:write — self-service callers may only set their
// own. Returns ErrSetAttendanceForbidden if the caller lacks that permission.
// Once the event's cancelLeadMinutes-derived cutoff has passed, a response
// is also rejected (ErrCancelLeadTimePassed) unless the caller holds
// events:write -- the same permission that lets an organizer set attendance
// for another member also lets them (or anyone else holding it) respond, or
// adjust a response, past the cutoff; there is no separate "late response"
// permission.
func (s *Service) SetAttendance(ctx context.Context, eventID, callerID, userID, teamID string, req gen.SetAttendanceRequest) (*gen.AttendanceRecord, error) {
	// status="not_nominated" is exclusively SetNomination's domain (an
	// events:write-gated organizer action, never self-service). Without this,
	// a member with only events:read could PUT their own attendance with
	// status="not_nominated" via this self-service endpoint, achieving the
	// same DB state SetNomination's permission gate exists to control --
	// AttendanceStatus's OpenAPI enum has no separate "settable by clients"
	// subset, so the handler-level Valid() check alone doesn't catch this.
	if req.Status == gen.NotNominated {
		return nil, ErrAttendanceStatusNotNominated
	}

	if callerID != userID {
		if err := s.requireCallerEventsWrite(ctx, callerID, teamID, ErrSetAttendanceForbidden); err != nil {
			return nil, err
		}
	}

	// Reject attendance changes on a cancelled event: a cancelled event isn't
	// happening, so recording (or rewriting) attendance for it is meaningless
	// and lets anyone alter the record after the fact. This guard is placed
	// after the permission check so an unauthorized caller still gets 403, not
	// a leak of the event's status. A GetEvent not-found error (wrapped here)
	// still satisfies the handler's pgx.ErrNoRows → 404 mapping.
	ev, err := s.repo.GetEvent(ctx, eventID, teamID)
	if err != nil {
		return nil, fmt.Errorf("events.Service.SetAttendance: load event: %w", err)
	}
	if ev.Status == "cancelled" {
		return nil, ErrEventCancelled
	}

	// Reject a response once the event's cancelLeadMinutes-derived cutoff
	// has passed (EventStartInstant - cancelLeadMinutes), unless the caller
	// holds events:write -- see the repository's identical, race-closing
	// re-check inside the write itself.
	if ev.CancelLeadMinutes != nil {
		start := EventStartInstant(ev.Date, ev.StartTime, ev.MeetTime)
		cutoff := start.Add(-time.Duration(*ev.CancelLeadMinutes) * time.Minute)
		if time.Now().After(cutoff) {
			if err := s.requireCallerEventsWrite(ctx, callerID, teamID, ErrCancelLeadTimePassed); err != nil {
				return nil, err
			}
		}
	}

	statusStr := string(req.Status)
	var reasonVisStr *string
	if req.ReasonVisibility != nil {
		rv := string(*req.ReasonVisibility)
		reasonVisStr = &rv
	}

	a, err := s.repo.SetAttendance(ctx, eventID, callerID, userID, teamID, &statusStr, req.Reason, req.ReasonId, reasonVisStr)
	if err != nil {
		return nil, fmt.Errorf("events.Service.SetAttendance: %w", err)
	}

	s.enqueueAttendanceNotification(ctx, ev, teamID, userID, req.Status, statusStr)

	rec := toGenAttendanceRecord(a)
	return &rec, nil
}

// enqueueAttendanceNotification best-effort enqueues an "attendance"
// notification for an actual response (yes/no/maybe) -- not for a reset to
// "pending", which the frontend has no distinct rendering for and isn't
// "feedback" to announce. The actor is the member the attendance belongs to
// (userID), not the caller -- an organizer setting another member's
// attendance shouldn't be attributed as the responder.
func (s *Service) enqueueAttendanceNotification(ctx context.Context, ev *EventRow, teamID, userID string, status gen.AttendanceStatus, statusStr string) {
	if s.jobs == nil || (status != gen.Yes && status != gen.No && status != gen.Maybe) {
		return
	}
	teamUUID, err := uuid.Parse(teamID)
	if err != nil {
		return
	}
	actorUUID, err := uuid.Parse(userID)
	if err != nil {
		return
	}
	evID := ev.Id
	evTitle := ev.Title
	evDate := ev.Date
	if err := s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
		TeamID:     teamUUID,
		Type:       "attendance",
		ActorID:    actorUUID,
		EventID:    &evID,
		EventTitle: &evTitle,
		EventDate:  &evDate,
		Status:     &statusStr,
	}); err != nil {
		s.logger.Warn("events: failed to enqueue notification", slog.String("eventId", evID.String()), slog.String("type", "attendance"), slog.String("error", err.Error()))
	}
}

// requireCallerEventsWrite checks whether callerID currently holds
// events:write for teamID, returning onDenied (the caller-facing sentinel
// appropriate to whichever gate is calling this -- ErrSetAttendanceForbidden
// for "acting on another member", ErrCancelLeadTimePassed for "responding
// after the deadline") when it doesn't, nil when it does. Shared by
// SetAttendance's events:write gates, which otherwise duplicate the same
// permChecker plumbing.
func (s *Service) requireCallerEventsWrite(ctx context.Context, callerID, teamID string, onDenied error) error {
	if s.permChecker == nil {
		return onDenied
	}
	teamUUID, err := uuid.Parse(teamID)
	if err != nil {
		return fmt.Errorf("events.Service.SetAttendance: parse teamID: %w", err)
	}
	callerUUID, err := uuid.Parse(callerID)
	if err != nil {
		return fmt.Errorf("events.Service.SetAttendance: parse callerID: %w", err)
	}
	perms, err := s.permChecker.GetPermissions(ctx, teamUUID, callerUUID)
	if err != nil {
		return fmt.Errorf("events.Service.SetAttendance: check permissions: %w", err)
	}
	if perms.Events != "write" {
		return onDenied
	}
	return nil
}

// SetNomination sets or clears a member's nomination for an event. Unlike
// SetAttendance, this is never self-service — nominating (even oneself) is
// an organizer action gated on events:write, matching the frontend's
// canEdit-only nominate/denominate controls. callerID is the authenticated
// user making the request. Returns ErrSetNominationForbidden if the caller
// lacks events:write.
func (s *Service) SetNomination(ctx context.Context, eventID, callerID, teamID string, req gen.SetNominationRequest) error {
	if s.permChecker == nil {
		return ErrSetNominationForbidden
	}
	teamUUID, err := uuid.Parse(teamID)
	if err != nil {
		return fmt.Errorf("events.Service.SetNomination: parse teamID: %w", err)
	}
	callerUUID, err := uuid.Parse(callerID)
	if err != nil {
		return fmt.Errorf("events.Service.SetNomination: parse callerID: %w", err)
	}
	perms, err := s.permChecker.GetPermissions(ctx, teamUUID, callerUUID)
	if err != nil {
		return fmt.Errorf("events.Service.SetNomination: check permissions: %w", err)
	}
	if perms.Events != "write" {
		return ErrSetNominationForbidden
	}

	if err := s.repo.SetNomination(ctx, eventID, callerID, req.UserId.String(), teamID, req.Nominated); err != nil {
		return fmt.Errorf("events.Service.SetNomination: %w", err)
	}
	return nil
}

// ─── internal helpers ────────────────────────────────────────────────────────

// enrichEvent converts an EventRow to a gen.TeamEvent, fetching summary and user attendance.
func (s *Service) enrichEvent(ctx context.Context, row *EventRow, userID, teamID string) (gen.TeamEvent, error) {
	summary, err := s.repo.GetAttendanceSummary(ctx, row.Id.String(), teamID)
	if err != nil {
		return gen.TeamEvent{}, fmt.Errorf("enrichEvent.GetAttendanceSummary: %w", err)
	}
	eventTeams, err := s.repo.GetEventTeams(ctx, row.Id.String())
	if err != nil {
		return gen.TeamEvent{}, fmt.Errorf("enrichEvent.GetEventTeams: %w", err)
	}

	ev := toGenEvent(row, summary)
	ev.CrossTeamIds = crossTeamIDsFrom(row.TeamId, eventTeams)

	if userID != "" {
		myAtt, err := s.repo.GetMyEffectiveAttendance(ctx, row.Id.String(), userID, teamID)
		if err != nil {
			return gen.TeamEvent{}, fmt.Errorf("enrichEvent.GetMyEffectiveAttendance: %w", err)
		}
		if myAtt != nil {
			st := gen.AttendanceStatus(myAtt.Status)
			ev.MyStatus = &st
			ev.MyReason = myAtt.Reason
			ev.MyAuto = &myAtt.Auto
		}
	}
	return ev, nil
}

// enrichEventOrFallback wraps enrichEvent for write-path callers whose
// underlying mutation has already committed: an enrichment failure (e.g. a
// transient timeout on the read-only summary/attendance queries) must not be
// reported as a request failure, since the caller would see a false error
// for an already-successful write and could retry it -- for CreateEvent that
// means minting a duplicate event/series. Falls back to the row's own data
// with a zero-value summary and no MyStatus; the next list/detail fetch
// picks up the real numbers. GetEvent (a plain read, no prior write) calls
// enrichEvent directly instead, since there a genuine failure should be
// reported as one.
func (s *Service) enrichEventOrFallback(ctx context.Context, row *EventRow, userID, teamID string) *gen.TeamEvent {
	ev, err := s.enrichEvent(ctx, row, userID, teamID)
	if err != nil {
		s.logger.Warn("events: failed to enrich event after write, returning partial result",
			slog.String("eventId", row.Id.String()), slog.String("error", err.Error()))
		fallback := toGenEvent(row, EventSummaryData{})
		return &fallback
	}
	return &ev
}

// toGenEvent maps an EventRow + summary to gen.TeamEvent.
// crossTeamIDsFrom returns every id in allTargets other than owning, or nil
// when there are none (a single-team event) -- mirrors toGenEvent's
// NominatedRoleIds nil-vs-populated convention, so a single-team event's
// JSON omits crossTeamIds entirely rather than encoding an empty array.
func crossTeamIDsFrom(owning uuid.UUID, allTargets []EventTeamRow) *[]openapi_types.UUID {
	var ids []openapi_types.UUID
	for _, t := range allTargets {
		if t.TeamID == owning {
			continue
		}
		ids = append(ids, t.TeamID)
	}
	if len(ids) == 0 {
		return nil
	}
	return &ids
}

// validateCrossTeamIds is the shared CreateEvent/UpdateEvent pre-check for a
// non-nil crossTeamIds: reject unknown/nonexistent team ids
// (ErrInvalidCrossTeamIds, surfaced as 400) before checking events:write in
// each of them (ErrCrossTeamWriteForbidden, 403) -- so a bad id doesn't get
// misreported as a permission problem. A nil or empty crossTeamIds is a
// no-op (nothing to validate).
func (s *Service) validateCrossTeamIds(ctx context.Context, callerID string, crossTeamIds []uuid.UUID) error {
	if len(crossTeamIds) == 0 {
		return nil
	}
	exists, err := s.repo.TeamsExist(ctx, crossTeamIds)
	if err != nil {
		return fmt.Errorf("events.Service: check crossTeamIds exist: %w", err)
	}
	if !exists {
		return ErrInvalidCrossTeamIds
	}
	return s.requireEventsWriteInTeams(ctx, callerID, crossTeamIds)
}

// requireEventsWriteInTeams checks that callerID holds events:write in
// every team in teamIDs -- used by CreateEvent/UpdateEvent to validate
// crossTeamIds on top of the URL's own {teamId}, which RequirePermission
// middleware already validated. Returns ErrCrossTeamWriteForbidden for the
// first team found missing it (or if no permChecker is configured, matching
// requireCallerEventsWrite's identical fail-closed default).
func (s *Service) requireEventsWriteInTeams(ctx context.Context, callerID string, teamIDs []uuid.UUID) error {
	if len(teamIDs) == 0 {
		return nil
	}
	if s.permChecker == nil {
		return ErrCrossTeamWriteForbidden
	}
	callerUUID, err := uuid.Parse(callerID)
	if err != nil {
		return fmt.Errorf("events.Service: parse callerID: %w", err)
	}
	for _, teamID := range teamIDs {
		perms, err := s.permChecker.GetPermissions(ctx, teamID, callerUUID)
		if err != nil {
			return fmt.Errorf("events.Service: check cross-team permissions: %w", err)
		}
		if perms.Events != "write" {
			return ErrCrossTeamWriteForbidden
		}
	}
	return nil
}

// computeTeamBadge implements the display rule (spec.md's "Team badge
// follows the viewer's own team, then alphabetical order"): nil if
// viewerTeamID is among memberships (the attendee shares the viewer's own
// team -- no badge), otherwise the alphabetically-first (case-insensitive)
// team name in memberships. nil (no badge at all) when memberships is
// empty -- defensive; every roster row comes from a targeted team's
// membership list, so this shouldn't happen in practice.
func computeTeamBadge(viewerTeamID uuid.UUID, memberships []EventTeamRow) *string {
	if len(memberships) == 0 {
		return nil
	}
	best := ""
	for _, t := range memberships {
		if t.TeamID == viewerTeamID {
			return nil
		}
		if best == "" || strings.ToLower(t.TeamName) < strings.ToLower(best) {
			best = t.TeamName
		}
	}
	return &best
}

func toGenEvent(row *EventRow, summary EventSummaryData) gen.TeamEvent {
	ev := gen.TeamEvent{
		Id:        row.Id,
		TeamId:    row.TeamId,
		Type:      gen.EventType(row.Type),
		Title:     row.Title,
		Date:      openapi_types.Date{Time: row.Date},
		Status:    gen.EventStatus(row.Status),
		Recurring: row.SeriesId != nil,
		Summary: gen.EventSummary{
			Yes:          summary.Yes,
			No:           summary.No,
			Maybe:        summary.Maybe,
			Pending:      summary.Pending,
			NotNominated: summary.NotNominated,
			Nominated:    summary.Nominated,
			Total:        summary.Total,
		},
		Location:          row.Location,
		Note:              row.Note,
		Result:            row.Result,
		MeetTime:          row.MeetTime,
		StartTime:         row.StartTime,
		EndTime:           row.EndTime,
		MeetTimeMandatory: row.MeetTimeMandatory,
		CancelLeadMinutes: row.CancelLeadMinutes,
		ExcludeFromStats:  row.ExcludeFromStats,
	}

	if row.SeriesId != nil {
		sid := *row.SeriesId
		ev.SeriesId = &sid
	}

	if row.EndDate != nil {
		ev.MultiDayEndDate = &openapi_types.Date{Time: *row.EndDate}
	}

	if row.ResponseMode != nil {
		rm := gen.ResponseMode(*row.ResponseMode)
		ev.ResponseMode = &rm
	}

	if len(row.NominatedRoleIds) > 0 {
		ids := make([]openapi_types.UUID, len(row.NominatedRoleIds))
		copy(ids, row.NominatedRoleIds)
		ev.NominatedRoleIds = &ids
	}

	return ev
}

// toGenComment maps a CommentRow to gen.EventComment.
func toGenComment(c *CommentRow) gen.EventComment {
	return gen.EventComment{
		Id:                 c.Id,
		EventId:            c.EventId,
		UserId:             c.UserId,
		Text:               c.Text,
		CreatedAt:          c.CreatedAt,
		AuthorName:         c.ActorName,
		AuthorColor:        c.ActorColor,
		HasAuthorPhoto:     c.HasActorPhoto,
		AuthorMembershipId: c.AuthorMembershipId,
	}
}

// toGenAttendanceRow maps an AttendanceEnriched to gen.AttendanceRow.
func toGenAttendanceRow(a *AttendanceEnriched) gen.AttendanceRow {
	row := gen.AttendanceRow{
		UserId:       a.UserId,
		MembershipId: &a.MembershipId,
		Status:       gen.AttendanceStatus(a.Status),
		Name:         a.Name,
		AvatarColor:  a.AvatarColor,
		HasPhoto:     &a.HasPhoto,
		Reason:       a.Reason,
		ReasonId:     a.ReasonId,
		Group:        a.Group,
		Title:        a.Title,
		Auto:         &a.Auto,
		Absent:       &a.Absent,
	}
	if a.ReasonVisibility != nil {
		rv := gen.AttendanceRowReasonVisibility(*a.ReasonVisibility)
		row.ReasonVisibility = &rv
	}
	if a.PrimaryRole != nil {
		role := teams.ToGenRole(*a.PrimaryRole)
		row.PrimaryRole = &role
	}
	return row
}

// toGenAttendanceRecord maps an AttendanceDBRow to gen.AttendanceRecord.
func toGenAttendanceRecord(a *AttendanceDBRow) gen.AttendanceRecord {
	rec := gen.AttendanceRecord{
		Id:      a.Id,
		EventId: a.EventId,
		UserId:  a.UserId,
		Status:  gen.AttendanceStatus(a.Status),
		Reason:  a.Reason,
		At:      a.At,
	}
	if a.ReasonVisibility != nil {
		rv := gen.AttendanceRecordReasonVisibility(*a.ReasonVisibility)
		rec.ReasonVisibility = &rv
	}
	if a.ReasonId != nil {
		rec.ReasonId = a.ReasonId
	}
	return rec
}

// ensure time is used (time.Time in toGenAttendanceRecord).
