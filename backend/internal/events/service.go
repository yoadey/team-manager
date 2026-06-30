package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/pagination"
)

// Sentinel errors for the events package.
var (
	ErrCreateEventNilBody     = errors.New("events.Service.CreateEvent: nil body")
	ErrCreateEventNoRow       = errors.New("events.Service.CreateEvent: no row returned")
	ErrUpdateEventNilBody     = errors.New("events.Service.UpdateEvent: nil body")
	ErrInvalidNominatedRoleIDs = errors.New("nominated_role_ids contain roles not belonging to this team")
)

// eventRepo is the interface the Service relies on.
type eventRepo interface {
	ListEvents(ctx context.Context, teamID string, scope string, limit int, cur *ListCursor) ([]EventRow, error)
	GetEvent(ctx context.Context, eventID string) (*EventRow, error)
	CreateEvent(ctx context.Context, teamID string, params *CreateEventParams) (*EventRow, error)
	CreateSeries(ctx context.Context, teamID string, params *CreateEventParams) ([]EventRow, error)
	UpdateEvent(ctx context.Context, eventID string, params *UpdateEventParams, scope string) (*EventRow, error)
	SetStatus(ctx context.Context, eventID string, status string, scope string) (*EventRow, error)
	DeleteEvent(ctx context.Context, eventID string, scope string) error
	GetAttendanceSummary(ctx context.Context, eventID string) (EventSummaryData, error)
	GetMyAttendance(ctx context.Context, eventID, userID string) (*AttendanceDBRow, error)
	ListAttendance(ctx context.Context, eventID string) ([]AttendanceEnriched, error)
	SetAttendance(ctx context.Context, eventID, userID string, status, reason, reasonID, reasonVisibility *string) (*AttendanceDBRow, error)
	SetNomination(ctx context.Context, eventID, userID string, nominated bool) error
	ListComments(ctx context.Context, eventID string, limit, offset int) ([]CommentRow, error)
	AddComment(ctx context.Context, eventID, userID, text string) (*CommentRow, error)
	DeleteComment(ctx context.Context, commentID, userID string) error
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

// Service implements event business logic.
type Service struct {
	repo        eventRepo
	jobs        jobEnqueuer
	pager       *pagination.Paginator
	roleChecker teamRoleChecker
}

// NewService creates a new Service. pager may be nil (uses default Paginator).
// roleChecker may be nil; when set, nominated_role_ids are validated to belong
// to the event's team before any create or update is persisted.
func NewService(repo eventRepo, enq jobEnqueuer, pager *pagination.Paginator, roleChecker teamRoleChecker) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, jobs: enq, pager: pager, roleChecker: roleChecker}
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
func (s *Service) ListEvents(ctx context.Context, teamID, userID, scope, cursor string, limit int) ([]gen.TeamEvent, *string, error) {
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

	out := make([]gen.TeamEvent, 0, len(rows))
	for i := range rows {
		ev, err := s.enrichEvent(ctx, &rows[i], userID)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, ev)
	}
	return out, next, nil
}

// ─── GetEvent ───────────────────────────────────────────────────────────────

// GetEvent retrieves a single event by ID enriched with summary and user status.
func (s *Service) GetEvent(ctx context.Context, teamID, userID, eventID string) (*gen.TeamEvent, error) {
	row, err := s.repo.GetEvent(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("events.Service.GetEvent: %w", err)
	}

	ev, err := s.enrichEvent(ctx, row, userID)
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
	if body.RepeatWeeks != nil && *body.RepeatWeeks > 0 {
		repeatWeeks = *body.RepeatWeeks
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
		params.NominatedRoleIds = append(params.NominatedRoleIds, *body.NominatedRoleIds...)
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
				_ = s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
					TeamID:     teamUUID,
					Type:       "event_created",
					ActorID:    actorUUID,
					EventID:    &evID,
					EventTitle: &evTitle,
					EventDate:  &evDate,
				})
			}
		}
	}

	ev, err := s.enrichEvent(ctx, row, userID)
	if err != nil {
		return nil, err
	}
	return &ev, nil
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
		params.NominatedRoleIds = append(params.NominatedRoleIds, *body.NominatedRoleIds...)
	}

	if err := s.validateNominatedRoles(ctx, teamID, params.NominatedRoleIds); err != nil {
		return nil, err
	}

	row, err := s.repo.UpdateEvent(ctx, eventID, &params, scope)
	if err != nil {
		return nil, fmt.Errorf("events.Service.UpdateEvent: %w", err)
	}

	ev, err := s.enrichEvent(ctx, row, userID)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

// ─── DeleteEvent ────────────────────────────────────────────────────────────

// DeleteEvent deletes an event or series.
func (s *Service) DeleteEvent(ctx context.Context, eventID, scope string) error {
	if err := s.repo.DeleteEvent(ctx, eventID, scope); err != nil {
		return fmt.Errorf("events.Service.DeleteEvent: %w", err)
	}
	return nil
}

// ─── SetStatus ──────────────────────────────────────────────────────────────

// SetStatus updates event status and returns the updated event.
func (s *Service) SetStatus(ctx context.Context, userID, eventID, status, scope string) (*gen.TeamEvent, error) {
	row, err := s.repo.SetStatus(ctx, eventID, status, scope)
	if err != nil {
		return nil, fmt.Errorf("events.Service.SetStatus: %w", err)
	}

	// Enqueue cancellation notification (best-effort).
	if s.jobs != nil && status == "cancelled" {
		if actorUUID, err2 := uuid.Parse(userID); err2 == nil {
			evID := row.Id
			evTitle := row.Title
			evDate := row.Date
			_ = s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
				TeamID:     row.TeamId,
				Type:       "event_cancelled",
				ActorID:    actorUUID,
				EventID:    &evID,
				EventTitle: &evTitle,
				EventDate:  &evDate,
			})
		}
	}

	ev, err := s.enrichEvent(ctx, row, userID)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

// ─── Comments ───────────────────────────────────────────────────────────────

// ListComments returns paginated comments for an event.
func (s *Service) ListComments(ctx context.Context, eventID string, limit, offset int) ([]gen.EventComment, error) {
	rows, err := s.repo.ListComments(ctx, eventID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("events.Service.ListComments: %w", err)
	}

	out := make([]gen.EventComment, 0, len(rows))
	for _, c := range rows {
		out = append(out, toGenComment(&c))
	}
	return out, nil
}

// AddComment adds a comment to an event.
func (s *Service) AddComment(ctx context.Context, eventID, userID, text string) (*gen.EventComment, error) {
	c, err := s.repo.AddComment(ctx, eventID, userID, text)
	if err != nil {
		return nil, fmt.Errorf("events.Service.AddComment: %w", err)
	}
	gc := toGenComment(c)
	return &gc, nil
}

// DeleteComment deletes a comment if the user owns it.
func (s *Service) DeleteComment(ctx context.Context, commentID, userID string) error {
	if err := s.repo.DeleteComment(ctx, commentID, userID); err != nil {
		return fmt.Errorf("events.Service.DeleteComment: %w", err)
	}
	return nil
}

// ─── Attendance ─────────────────────────────────────────────────────────────

// ListAttendance returns all attendance rows for an event.
func (s *Service) ListAttendance(ctx context.Context, eventID string) ([]gen.AttendanceRow, error) {
	attendanceRows, err := s.repo.ListAttendance(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("events.Service.ListAttendance: %w", err)
	}

	out := make([]gen.AttendanceRow, 0, len(attendanceRows))
	for _, a := range attendanceRows {
		out = append(out, toGenAttendanceRow(&a))
	}
	return out, nil
}

// SetAttendance upserts an attendance record.
func (s *Service) SetAttendance(ctx context.Context, eventID, userID string, req gen.SetAttendanceRequest) (*gen.AttendanceRecord, error) {
	statusStr := string(req.Status)
	var reasonVisStr *string
	if req.ReasonVisibility != nil {
		rv := string(*req.ReasonVisibility)
		reasonVisStr = &rv
	}

	a, err := s.repo.SetAttendance(ctx, eventID, userID, &statusStr, req.Reason, req.ReasonId, reasonVisStr)
	if err != nil {
		return nil, fmt.Errorf("events.Service.SetAttendance: %w", err)
	}
	rec := toGenAttendanceRecord(a)
	return &rec, nil
}

// SetNomination sets or removes a user's nomination on an event.
func (s *Service) SetNomination(ctx context.Context, eventID, userID string, req gen.SetNominationRequest) error {
	if err := s.repo.SetNomination(ctx, eventID, req.UserId.String(), req.Nominated); err != nil {
		return fmt.Errorf("events.Service.SetNomination: %w", err)
	}
	return nil
}

// ─── internal helpers ────────────────────────────────────────────────────────

// enrichEvent converts an EventRow to a gen.TeamEvent, fetching summary and user attendance.
func (s *Service) enrichEvent(ctx context.Context, row *EventRow, userID string) (gen.TeamEvent, error) {
	summary, err := s.repo.GetAttendanceSummary(ctx, row.Id.String())
	if err != nil {
		return gen.TeamEvent{}, fmt.Errorf("enrichEvent.GetAttendanceSummary: %w", err)
	}

	ev := toGenEvent(row, summary)

	if userID != "" {
		myAtt, err := s.repo.GetMyAttendance(ctx, row.Id.String(), userID)
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
