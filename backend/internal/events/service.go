package events

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	ErrCreateEventNilBody           = errors.New("events.Service.CreateEvent: nil body")
	ErrCreateEventNoRow             = errors.New("events.Service.CreateEvent: no row returned")
	ErrUpdateEventNilBody           = errors.New("events.Service.UpdateEvent: nil body")
	ErrInvalidNominatedRoleIDs      = errors.New("nominated_role_ids contain roles not belonging to this team")
	ErrSetAttendanceForbidden       = errors.New("events.Service.SetAttendance: caller may not set another member's attendance")
	ErrSetNominationForbidden       = errors.New("events.Service.SetNomination: caller lacks events:write")
	ErrAttendanceStatusNotNominated = errors.New("events.Service.SetAttendance: status 'not_nominated' may only be set via SetNomination")
	ErrRepeatWeeksTooLarge          = fmt.Errorf("repeat_weeks must be between 1 and %d", maxRepeatWeeks)
	ErrTooManyComments              = fmt.Errorf("event has reached the maximum of %d comments", maxCommentsPerEvent)
)

// maxRepeatWeeks caps how many events a single recurring series may create.
// The OpenAPI spec only declares a minimum, and no request-schema validator
// is wired into the router, so this is the only enforcement point; without
// it, CreateSeries would loop an attacker-controlled number of times inside
// one DB transaction.
const maxRepeatWeeks = 104

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
	GetMyAttendance(ctx context.Context, eventID, userID, teamID string) (*AttendanceDBRow, error)
	GetAttendanceSummaries(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID]EventSummaryData, error)
	GetMyAttendances(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]AttendanceDBRow, error)
	ListAttendance(ctx context.Context, eventID, teamID string) ([]AttendanceEnriched, error)
	GetReasonVisibilityContext(ctx context.Context, teamID, viewerID string) (teamRoleIDs, viewerRoleIDs []string, err error)
	SetAttendance(ctx context.Context, eventID, callerID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*AttendanceDBRow, error)
	SetNomination(ctx context.Context, eventID, callerID, userID, teamID string, nominated bool) error
	ListComments(ctx context.Context, eventID, teamID string, limit, offset int) ([]CommentRow, error)
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
	summaries, err := s.repo.GetAttendanceSummaries(ctx, eventIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("events.Service.ListEvents: %w", err)
	}
	var myAttendances map[uuid.UUID]AttendanceDBRow
	if userID != "" {
		myAttendances, err = s.repo.GetMyAttendances(ctx, eventIDs, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("events.Service.ListEvents: %w", err)
		}
	}

	out := make([]gen.TeamEvent, 0, len(rows))
	for i := range rows {
		ev := toGenEvent(&rows[i], summaries[rows[i].Id])
		if myAtt, ok := myAttendances[rows[i].Id]; ok {
			st := gen.AttendanceStatus(myAtt.Status)
			ev.MyStatus = &st
			ev.MyReason = myAtt.Reason
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
	repeatWeeks := 1
	if body.RepeatWeeks != nil {
		repeatWeeks = *body.RepeatWeeks
	}
	if repeatWeeks < 1 || repeatWeeks > maxRepeatWeeks {
		return nil, ErrRepeatWeeksTooLarge
	}

	params := CreateEventParams{
		Type:              string(body.Type),
		Title:             body.Title,
		Date:              body.Date.Time,
		Location:          body.Location,
		Note:              body.Note,
		MeetTime:          body.MeetTime,
		StartTime:         body.StartTime,
		EndTime:           body.EndTime,
		MeetTimeMandatory: body.MeetTimeMandatory,
		Recurring:         recurring,
		RepeatWeeks:       repeatWeeks,
	}
	if body.ResponseMode != nil {
		rm := string(*body.ResponseMode)
		params.ResponseMode = &rm
	}
	if body.NominatedRoleIds != nil {
		params.NominatedRoleIds = *body.NominatedRoleIds
	}

	if err := s.validateNominatedRoles(ctx, teamID, params.NominatedRoleIds); err != nil {
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
	}
	if body.Type != nil {
		t := string(*body.Type)
		params.Type = &t
	}
	if body.Date != nil {
		d := body.Date.Time
		params.Date = &d
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

	if err := s.validateNominatedRoles(ctx, teamID, params.NominatedRoleIds); err != nil {
		return nil, err
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

// ListComments returns paginated comments for an event scoped to teamID.
func (s *Service) ListComments(ctx context.Context, eventID, teamID string, limit, offset int) ([]gen.EventComment, error) {
	rows, err := s.repo.ListComments(ctx, eventID, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("events.Service.ListComments: %w", err)
	}

	out := make([]gen.EventComment, 0, len(rows))
	for _, c := range rows {
		out = append(out, toGenComment(&c))
	}
	return out, nil
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

	needsRedactionCheck := false
	for _, a := range attendanceRows {
		if a.UserId.String() != viewerID && (a.Reason != nil || a.ReasonId != nil) && !reasonSharedWithTeam(a.ReasonVisibility) {
			needsRedactionCheck = true
			break
		}
	}

	var canSeeReasons bool
	if needsRedactionCheck {
		teamRoleIDs, viewerRoleIDs, err := s.repo.GetReasonVisibilityContext(ctx, teamID, viewerID)
		if err != nil {
			return nil, fmt.Errorf("events.Service.ListAttendance: %w", err)
		}
		canSeeReasons = roleSetsIntersect(teamRoleIDs, viewerRoleIDs)
	}

	out := make([]gen.AttendanceRow, 0, len(attendanceRows))
	for _, a := range attendanceRows {
		if a.UserId.String() != viewerID && !canSeeReasons && !reasonSharedWithTeam(a.ReasonVisibility) {
			a.Reason = nil
			a.ReasonId = nil
		}
		out = append(out, toGenAttendanceRow(&a))
	}
	return out, nil
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
		if s.permChecker == nil {
			return nil, ErrSetAttendanceForbidden
		}
		teamUUID, err := uuid.Parse(teamID)
		if err != nil {
			return nil, fmt.Errorf("events.Service.SetAttendance: parse teamID: %w", err)
		}
		callerUUID, err := uuid.Parse(callerID)
		if err != nil {
			return nil, fmt.Errorf("events.Service.SetAttendance: parse callerID: %w", err)
		}
		perms, err := s.permChecker.GetPermissions(ctx, teamUUID, callerUUID)
		if err != nil {
			return nil, fmt.Errorf("events.Service.SetAttendance: check permissions: %w", err)
		}
		if perms.Events != "write" {
			return nil, ErrSetAttendanceForbidden
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
	rec := toGenAttendanceRecord(a)
	return &rec, nil
}

// SetNomination sets or removes a user's nomination on an event scoped to teamID.
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

	ev := toGenEvent(row, summary)

	if userID != "" {
		myAtt, err := s.repo.GetMyAttendance(ctx, row.Id.String(), userID, teamID)
		if err != nil {
			return gen.TeamEvent{}, fmt.Errorf("enrichEvent.GetMyAttendance: %w", err)
		}
		if myAtt != nil {
			st := gen.AttendanceStatus(myAtt.Status)
			ev.MyStatus = &st
			ev.MyReason = myAtt.Reason
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
	}

	if row.SeriesId != nil {
		sid := *row.SeriesId
		ev.SeriesId = &sid
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
		Id:             c.Id,
		EventId:        c.EventId,
		UserId:         c.UserId,
		Text:           c.Text,
		CreatedAt:      c.CreatedAt,
		AuthorName:     c.ActorName,
		AuthorColor:    c.ActorColor,
		HasAuthorPhoto: c.HasActorPhoto,
	}
}

// toGenAttendanceRow maps an AttendanceEnriched to gen.AttendanceRow.
func toGenAttendanceRow(a *AttendanceEnriched) gen.AttendanceRow {
	row := gen.AttendanceRow{
		UserId:      a.UserId,
		Status:      gen.AttendanceStatus(a.Status),
		Name:        a.Name,
		AvatarColor: a.AvatarColor,
		HasPhoto:    &a.HasPhoto,
		Reason:      a.Reason,
		ReasonId:    a.ReasonId,
	}
	if a.ReasonVisibility != nil {
		rv := gen.AttendanceRowReasonVisibility(*a.ReasonVisibility)
		row.ReasonVisibility = &rv
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
var _ = time.Time{}
