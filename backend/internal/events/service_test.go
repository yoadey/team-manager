package events_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock repository ────────────────────────────────────────────────────────

// mockSvcRepo satisfies the unexported eventRepo interface via structural typing.
type mockSvcRepo struct {
	listEventsFn                func(ctx context.Context, teamID string, scope gen.ListEventsParamsScope, limit int, cur *events.ListCursor) ([]events.EventRow, error)
	getEventFn                  func(ctx context.Context, eventID, teamID string) (*events.EventRow, error)
	createEventFn               func(ctx context.Context, teamID string, params *events.CreateEventParams) (*events.EventRow, error)
	createSeriesFn              func(ctx context.Context, teamID string, params *events.CreateEventParams) ([]events.EventRow, error)
	updateEventFn               func(ctx context.Context, eventID, teamID string, params *events.UpdateEventParams, scope string) (*events.EventRow, error)
	setStatusFn                 func(ctx context.Context, eventID, teamID, status, scope string) (*events.EventRow, error)
	deleteEventFn               func(ctx context.Context, eventID, teamID, scope string) error
	getAttendanceSummaryFn      func(ctx context.Context, eventID, teamID string) (events.EventSummaryData, error)
	getMyAttendanceFn           func(ctx context.Context, eventID, userID, teamID string) (*events.AttendanceDBRow, error)
	getAttendanceSummariesFn    func(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID]events.EventSummaryData, error)
	getMyAttendancesFn          func(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]events.AttendanceDBRow, error)
	getMyEffectiveAttendanceFn  func(ctx context.Context, eventID, userID, teamID string) (*events.EffectiveAttendance, error)
	getMyEffectiveAttendancesFn func(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]events.EffectiveAttendance, error)
	listAttendanceFn            func(ctx context.Context, eventID, teamID string) ([]events.AttendanceEnriched, error)
	getReasonVisibilityCtxFn    func(ctx context.Context, teamID, viewerID string) ([]string, []string, error)
	setAttendanceFn             func(ctx context.Context, eventID, callerID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*events.AttendanceDBRow, error)
	setNominationFn             func(ctx context.Context, eventID, callerID, userID, teamID string, nominated bool) error
	listCommentsFn              func(ctx context.Context, eventID, teamID string, limit, offset int) ([]events.CommentRow, error)
	countCommentsFn             func(ctx context.Context, eventID, teamID string) (int, error)
	addCommentFn                func(ctx context.Context, eventID, userID, teamID, text string) (*events.CommentRow, error)
	deleteCommentFn             func(ctx context.Context, commentID, userID, teamID string) error
}

func (m *mockSvcRepo) ListEvents(ctx context.Context, teamID string, scope gen.ListEventsParamsScope, limit int, cur *events.ListCursor) ([]events.EventRow, error) {
	return m.listEventsFn(ctx, teamID, scope, limit, cur)
}

func (m *mockSvcRepo) GetEvent(ctx context.Context, eventID, teamID string) (*events.EventRow, error) {
	return m.getEventFn(ctx, eventID, teamID)
}

func (m *mockSvcRepo) CreateEvent(ctx context.Context, teamID string, params *events.CreateEventParams) (*events.EventRow, error) {
	return m.createEventFn(ctx, teamID, params)
}

func (m *mockSvcRepo) CreateSeries(ctx context.Context, teamID string, params *events.CreateEventParams) ([]events.EventRow, error) {
	return m.createSeriesFn(ctx, teamID, params)
}

func (m *mockSvcRepo) UpdateEvent(ctx context.Context, eventID, teamID string, params *events.UpdateEventParams, scope string) (*events.EventRow, error) {
	return m.updateEventFn(ctx, eventID, teamID, params, scope)
}

func (m *mockSvcRepo) SetStatus(ctx context.Context, eventID, teamID, status, scope string) (*events.EventRow, error) {
	return m.setStatusFn(ctx, eventID, teamID, status, scope)
}

func (m *mockSvcRepo) DeleteEvent(ctx context.Context, eventID, teamID, scope string) error {
	return m.deleteEventFn(ctx, eventID, teamID, scope)
}

func (m *mockSvcRepo) GetAttendanceSummary(ctx context.Context, eventID, teamID string) (events.EventSummaryData, error) {
	return m.getAttendanceSummaryFn(ctx, eventID, teamID)
}

func (m *mockSvcRepo) GetMyAttendance(ctx context.Context, eventID, userID, teamID string) (*events.AttendanceDBRow, error) {
	return m.getMyAttendanceFn(ctx, eventID, userID, teamID)
}

func (m *mockSvcRepo) GetAttendanceSummaries(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID]events.EventSummaryData, error) {
	if m.getAttendanceSummariesFn != nil {
		return m.getAttendanceSummariesFn(ctx, eventIDs)
	}
	return map[uuid.UUID]events.EventSummaryData{}, nil
}

func (m *mockSvcRepo) GetMyAttendances(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]events.AttendanceDBRow, error) {
	if m.getMyAttendancesFn != nil {
		return m.getMyAttendancesFn(ctx, eventIDs, userID)
	}
	return map[uuid.UUID]events.AttendanceDBRow{}, nil
}

func (m *mockSvcRepo) GetMyEffectiveAttendance(ctx context.Context, eventID, userID, teamID string) (*events.EffectiveAttendance, error) {
	if m.getMyEffectiveAttendanceFn != nil {
		return m.getMyEffectiveAttendanceFn(ctx, eventID, userID, teamID)
	}
	return &events.EffectiveAttendance{Status: "pending"}, nil
}

func (m *mockSvcRepo) GetMyEffectiveAttendances(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]events.EffectiveAttendance, error) {
	if m.getMyEffectiveAttendancesFn != nil {
		return m.getMyEffectiveAttendancesFn(ctx, eventIDs, userID)
	}
	return map[uuid.UUID]events.EffectiveAttendance{}, nil
}

func (m *mockSvcRepo) ListAttendance(ctx context.Context, eventID, teamID string) ([]events.AttendanceEnriched, error) {
	return m.listAttendanceFn(ctx, eventID, teamID)
}

func (m *mockSvcRepo) GetReasonVisibilityContext(ctx context.Context, teamID, viewerID string) (teamRoleIDs, viewerRoleIDs []string, err error) {
	if m.getReasonVisibilityCtxFn != nil {
		return m.getReasonVisibilityCtxFn(ctx, teamID, viewerID)
	}
	return nil, nil, nil
}

func (m *mockSvcRepo) SetAttendance(ctx context.Context, eventID, callerID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*events.AttendanceDBRow, error) {
	return m.setAttendanceFn(ctx, eventID, callerID, userID, teamID, status, reason, reasonID, reasonVisibility)
}

func (m *mockSvcRepo) SetNomination(ctx context.Context, eventID, callerID, userID, teamID string, nominated bool) error {
	return m.setNominationFn(ctx, eventID, callerID, userID, teamID, nominated)
}

func (m *mockSvcRepo) ListComments(ctx context.Context, eventID, teamID string, limit, offset int) ([]events.CommentRow, error) {
	return m.listCommentsFn(ctx, eventID, teamID, limit, offset)
}

func (m *mockSvcRepo) CountComments(ctx context.Context, eventID, teamID string) (int, error) {
	if m.countCommentsFn != nil {
		return m.countCommentsFn(ctx, eventID, teamID)
	}
	return 0, nil
}

func (m *mockSvcRepo) AddComment(ctx context.Context, eventID, userID, teamID, text string) (*events.CommentRow, error) {
	return m.addCommentFn(ctx, eventID, userID, teamID, text)
}

func (m *mockSvcRepo) DeleteComment(ctx context.Context, commentID, userID, teamID string) error {
	return m.deleteCommentFn(ctx, commentID, userID, teamID)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func svcMakeEventRow(title string) events.EventRow {
	return events.EventRow{
		Id:     uuid.New(),
		TeamId: uuid.MustParse(testTeamID),
		Type:   "training",
		Title:  title,
		Date:   time.Now().UTC(),
		Status: "active",
	}
}

func zeroSummaryFn(_ context.Context, _, _ string) (events.EventSummaryData, error) {
	return events.EventSummaryData{}, nil
}

func nilMyAttendanceFn(_ context.Context, _, _, _ string) (*events.AttendanceDBRow, error) {
	return nil, nil
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestEventService_ListEvents_Upcoming(t *testing.T) {
	t.Parallel()

	rows := []events.EventRow{
		svcMakeEventRow("Event 1"),
		svcMakeEventRow("Event 2"),
		svcMakeEventRow("Event 3"),
	}

	var capturedScope gen.ListEventsParamsScope
	repo := &mockSvcRepo{
		listEventsFn: func(_ context.Context, _ string, scope gen.ListEventsParamsScope, _ int, _ *events.ListCursor) ([]events.EventRow, error) {
			capturedScope = scope
			return rows, nil
		},
		getAttendanceSummaryFn: zeroSummaryFn,
		getMyAttendanceFn:      nilMyAttendanceFn,
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	result, next, err := svc.ListEvents(context.Background(), testTeamID, testUserID, gen.Upcoming, "", 50)
	require.NoError(t, err)
	assert.Nil(t, next)
	assert.Len(t, result, 3)
	assert.Equal(t, gen.Upcoming, capturedScope, "scope should be passed through to repository")
}

// Regression test: enrichEvent/ListEvents used to read GetMyAttendance(s),
// which stays nil for a member who never explicitly responded -- MyAuto was
// never populated at all, and MyStatus silently stayed nil instead of
// reflecting opt_out/absence-based defaulting. This verifies the service
// layer actually copies the repository's resolved Auto/Status/Reason
// through to gen.TeamEvent for both the single-event and list paths.
func TestEventService_GetEvent_And_ListEvents_PopulateMyAutoFromEffectiveAttendance(t *testing.T) {
	t.Parallel()

	row := svcMakeEventRow("Opt-out Training")

	repo := &mockSvcRepo{
		getEventFn: func(_ context.Context, _, _ string) (*events.EventRow, error) {
			return &row, nil
		},
		getAttendanceSummaryFn: zeroSummaryFn,
		getMyEffectiveAttendanceFn: func(_ context.Context, _, _, _ string) (*events.EffectiveAttendance, error) {
			return &events.EffectiveAttendance{Status: "yes", Auto: true}, nil
		},
		listEventsFn: func(_ context.Context, _ string, _ gen.ListEventsParamsScope, _ int, _ *events.ListCursor) ([]events.EventRow, error) {
			return []events.EventRow{row}, nil
		},
		getMyEffectiveAttendancesFn: func(_ context.Context, eventIDs []uuid.UUID, _ string) (map[uuid.UUID]events.EffectiveAttendance, error) {
			return map[uuid.UUID]events.EffectiveAttendance{
				row.Id: {Status: "yes", Auto: true},
			}, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())

	single, err := svc.GetEvent(context.Background(), testTeamID, testUserID, row.Id.String())
	require.NoError(t, err)
	require.NotNil(t, single.MyStatus)
	assert.Equal(t, gen.Yes, *single.MyStatus)
	require.NotNil(t, single.MyAuto)
	assert.True(t, *single.MyAuto)

	list, _, err := svc.ListEvents(context.Background(), testTeamID, testUserID, gen.Upcoming, "", 50)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NotNil(t, list[0].MyStatus)
	assert.Equal(t, gen.Yes, *list[0].MyStatus)
	require.NotNil(t, list[0].MyAuto)
	assert.True(t, *list[0].MyAuto)
}

func TestEventService_CreateEvent_Recurring(t *testing.T) {
	t.Parallel()

	sid := uuid.New()
	eventRows := []events.EventRow{
		svcMakeEventRow("Weekly Training"),
		svcMakeEventRow("Weekly Training"),
		svcMakeEventRow("Weekly Training"),
	}
	for i := range eventRows {
		eventRows[i].SeriesId = &sid
	}

	seriesCalled := false
	createCalled := false

	repo := &mockSvcRepo{
		createSeriesFn: func(_ context.Context, teamID string, params *events.CreateEventParams) ([]events.EventRow, error) {
			seriesCalled = true
			assert.True(t, params.Recurring)
			assert.Equal(t, 3, params.RepeatWeeks)
			return eventRows, nil
		},
		createEventFn: func(_ context.Context, teamID string, params *events.CreateEventParams) (*events.EventRow, error) {
			createCalled = true
			return &eventRows[0], nil
		},
		getAttendanceSummaryFn: zeroSummaryFn,
		getMyAttendanceFn:      nilMyAttendanceFn,
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	repeatWeeks := 3
	recurring := true
	body := &gen.CreateEventRequest{
		Type:        gen.Training,
		Title:       "Weekly Training",
		Date:        openapi_types.Date{Time: time.Now().UTC()},
		Recurring:   &recurring,
		RepeatWeeks: &repeatWeeks,
	}

	result, err := svc.CreateEvent(context.Background(), testTeamID, testUserID, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, seriesCalled, "CreateSeries should be called for recurring events")
	assert.False(t, createCalled, "CreateEvent should NOT be called for recurring events")
	assert.True(t, result.Recurring, "resulting event should have Recurring=true")
	assert.NotNil(t, result.SeriesId)
}

// Regression test: CreateEvent/CreateSeries commits the write before
// enrichEvent's read-only summary/attendance queries run. A transient
// failure in those reads (e.g. a deadline hit right after the write) used to
// propagate as an error from CreateEvent itself, reporting an already
// -successful (and, for a recurring series, already fully committed) write
// as a failure -- inviting a client retry that would mint a duplicate
// series. It must instead fall back to the row's own data with a zero-value
// summary rather than fail the request.
func TestEventService_CreateEvent_EnrichmentFailureDoesNotFailAlreadyCommittedWrite(t *testing.T) {
	t.Parallel()

	row := svcMakeEventRow("Weekly Training")
	repo := &mockSvcRepo{
		createEventFn: func(_ context.Context, _ string, _ *events.CreateEventParams) (*events.EventRow, error) {
			return &row, nil
		},
		getAttendanceSummaryFn: func(_ context.Context, _, _ string) (events.EventSummaryData, error) {
			return events.EventSummaryData{}, errors.New("transient deadline exceeded")
		},
		getMyAttendanceFn: nilMyAttendanceFn,
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	body := &gen.CreateEventRequest{
		Type:  gen.Training,
		Title: "Weekly Training",
		Date:  openapi_types.Date{Time: time.Now().UTC()},
	}
	result, err := svc.CreateEvent(context.Background(), testTeamID, testUserID, body)
	require.NoError(t, err, "an enrichment failure after a committed write must not fail the request")
	require.NotNil(t, result)
	assert.Equal(t, row.Id, result.Id)
	assert.Equal(t, 0, result.Summary.Total, "falls back to a zero-value summary rather than propagating the enrichment error")
}

// Regression test: UpdateEventJSONRequestBody.NominatedRoleIds is a
// *[]uuid.UUID, distinguishing "omitted" (nil) from "explicit empty array"
// (non-nil, len 0) -- a client clearing all nominated roles sends the
// latter. The service used to build params.NominatedRoleIds via
// append(params.NominatedRoleIds, *body.NominatedRoleIds...), and Go's
// append(nil, ...zero elements) returns nil, so the field silently stayed
// nil and buildUpdateSets' `!= nil` check treated the clear request as "not
// provided," never actually clearing the column.
func TestEventService_UpdateEvent_ClearsNominatedRoleIdsWithExplicitEmptyArray(t *testing.T) {
	t.Parallel()

	var capturedRoleIDs []uuid.UUID
	capturedWasNil := true
	repo := &mockSvcRepo{
		updateEventFn: func(_ context.Context, _, _ string, params *events.UpdateEventParams, _ string) (*events.EventRow, error) {
			capturedRoleIDs = params.NominatedRoleIds
			capturedWasNil = params.NominatedRoleIds == nil
			row := svcMakeEventRow("Training")
			return &row, nil
		},
		getAttendanceSummaryFn: zeroSummaryFn,
		getMyAttendanceFn:      nilMyAttendanceFn,
	}
	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())

	emptyRoleIDs := []uuid.UUID{}
	body := &gen.UpdateEventJSONRequestBody{NominatedRoleIds: &emptyRoleIDs}
	_, err := svc.UpdateEvent(context.Background(), testTeamID, testUserID, uuid.New().String(), "single", body)
	require.NoError(t, err)

	assert.False(t, capturedWasNil, "NominatedRoleIds must stay non-nil so buildUpdateSets actually clears the column")
	assert.Empty(t, capturedRoleIDs)
}

func TestEventService_CreateEvent_Recurring_RejectsExcessiveRepeatWeeks(t *testing.T) {
	t.Parallel()

	repo := &mockSvcRepo{
		createSeriesFn: func(_ context.Context, _ string, _ *events.CreateEventParams) ([]events.EventRow, error) {
			t.Fatal("CreateSeries must not be called when repeatWeeks exceeds the cap")
			return nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	repeatWeeks := 100000
	recurring := true
	body := &gen.CreateEventRequest{
		Type:        gen.Training,
		Title:       "Runaway Series",
		Date:        openapi_types.Date{Time: time.Now().UTC()},
		Recurring:   &recurring,
		RepeatWeeks: &repeatWeeks,
	}

	_, err := svc.CreateEvent(context.Background(), testTeamID, testUserID, body)
	require.ErrorIs(t, err, events.ErrRepeatWeeksTooLarge)
}

// Regression test: a non-positive repeatWeeks used to be silently coerced to
// 1 instead of rejected, even though the OpenAPI spec declares minimum: 1 --
// a client sending 0 or a negative value got a single non-recurring event
// created with no error, instead of the 400 the spec promises.
func TestEventService_CreateEvent_Recurring_RejectsNonPositiveRepeatWeeks(t *testing.T) {
	t.Parallel()

	repo := &mockSvcRepo{
		createSeriesFn: func(_ context.Context, _ string, _ *events.CreateEventParams) ([]events.EventRow, error) {
			t.Fatal("CreateSeries must not be called when repeatWeeks is non-positive")
			return nil, nil
		},
		createEventFn: func(_ context.Context, _ string, _ *events.CreateEventParams) (*events.EventRow, error) {
			t.Fatal("CreateEvent must not be called when repeatWeeks is non-positive")
			return nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	for _, invalid := range []int{0, -1} {
		repeatWeeks := invalid
		recurring := true
		body := &gen.CreateEventRequest{
			Type:        gen.Training,
			Title:       "Invalid Repeat",
			Date:        openapi_types.Date{Time: time.Now().UTC()},
			Recurring:   &recurring,
			RepeatWeeks: &repeatWeeks,
		}
		_, err := svc.CreateEvent(context.Background(), testTeamID, testUserID, body)
		require.ErrorIs(t, err, events.ErrRepeatWeeksTooLarge, "repeatWeeks=%d must be rejected", invalid)
	}
}

func TestEventService_SetAttendance(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	capturedStatus := ""
	rec := &events.AttendanceDBRow{
		Id:      uuid.New(),
		EventId: eventID,
		UserId:  userID,
		Status:  "yes",
	}

	repo := &mockSvcRepo{
		setAttendanceFn: func(_ context.Context, evID, _, uID, tID string, status, _, _, _ *string) (*events.AttendanceDBRow, error) {
			assert.Equal(t, eventID.String(), evID)
			assert.Equal(t, userID.String(), uID)
			assert.Equal(t, teamID.String(), tID)
			if status != nil {
				capturedStatus = *status
			}
			return rec, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	req := gen.SetAttendanceRequest{
		UserId: userID,
		Status: gen.Yes,
	}

	result, err := svc.SetAttendance(context.Background(), eventID.String(), userID.String(), userID.String(), teamID.String(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "yes", capturedStatus)
	assert.Equal(t, gen.Yes, result.Status)
}

// TestEventService_SetAttendance_RejectsNotNominatedStatus regression-tests a
// gap where status="not_nominated" -- exclusively SetNomination's domain, an
// events:write-gated organizer action -- could be set via the self-service
// SetAttendance endpoint with no permission check at all when callerID ==
// userID, since AttendanceStatus's OpenAPI enum has no separate "settable by
// clients" subset and the handler's Valid() check accepts any enum member.
// A member with only events:read could otherwise unilaterally achieve the
// same DB state SetNomination's events:write gate exists to control.
func TestEventService_SetAttendance_RejectsNotNominatedStatus(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	userID := uuid.New()
	teamID := uuid.New()

	repo := &mockSvcRepo{
		setAttendanceFn: func(_ context.Context, _, _, _, _ string, _, _, _, _ *string) (*events.AttendanceDBRow, error) {
			t.Fatal("repository must not be called for status=not_nominated")
			return nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	req := gen.SetAttendanceRequest{UserId: userID, Status: gen.NotNominated}

	_, err := svc.SetAttendance(context.Background(), eventID.String(), userID.String(), userID.String(), teamID.String(), req)
	require.ErrorIs(t, err, events.ErrAttendanceStatusNotNominated)
}

// mockPermChecker satisfies the unexported permissionChecker interface via
// structural typing.
type mockPermChecker struct {
	perms teams.PermissionsJSON
	err   error
}

func (m *mockPermChecker) GetPermissions(_ context.Context, _, _ uuid.UUID) (teams.PermissionsJSON, error) {
	return m.perms, m.err
}

func TestEventService_SetAttendance_ForOtherMember_RequiresEventsWrite(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	callerID := uuid.New()
	targetUserID := uuid.New()
	teamID := uuid.New()

	repo := &mockSvcRepo{
		setAttendanceFn: func(_ context.Context, _, _, _, _ string, _, _, _, _ *string) (*events.AttendanceDBRow, error) {
			t.Fatal("repository must not be called when caller lacks events:write")
			return nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, &mockPermChecker{perms: teams.PermissionsJSON{Events: "read"}}, slog.Default())
	req := gen.SetAttendanceRequest{UserId: targetUserID, Status: gen.Yes}

	_, err := svc.SetAttendance(context.Background(), eventID.String(), callerID.String(), targetUserID.String(), teamID.String(), req)
	require.ErrorIs(t, err, events.ErrSetAttendanceForbidden)
}

func TestEventService_SetAttendance_ForOtherMember_AllowedWithEventsWrite(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	callerID := uuid.New()
	targetUserID := uuid.New()
	teamID := uuid.New()

	rec := &events.AttendanceDBRow{Id: uuid.New(), EventId: eventID, UserId: targetUserID, Status: "yes"}
	repo := &mockSvcRepo{
		setAttendanceFn: func(_ context.Context, _, _, uID, _ string, _, _, _, _ *string) (*events.AttendanceDBRow, error) {
			assert.Equal(t, targetUserID.String(), uID)
			return rec, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, &mockPermChecker{perms: teams.PermissionsJSON{Events: "write"}}, slog.Default())
	req := gen.SetAttendanceRequest{UserId: targetUserID, Status: gen.Yes}

	result, err := svc.SetAttendance(context.Background(), eventID.String(), callerID.String(), targetUserID.String(), teamID.String(), req)
	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestEventService_SetNomination_RequiresEventsWrite guards against a
// regression where a middleware path-parsing bug made this endpoint
// self-service (any member, regardless of permissions) instead of requiring
// events:write — nominating another member is an organizer-only action,
// never self-service, even for nominating oneself. This test exercises the
// service-layer check directly so the endpoint stays safe even if the
// middleware's route classification regresses again.
func TestEventService_SetNomination_RequiresEventsWrite(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	callerID := uuid.New()
	targetUserID := uuid.New()
	teamID := uuid.New()

	repo := &mockSvcRepo{
		setNominationFn: func(context.Context, string, string, string, string, bool) error {
			t.Fatal("repository must not be called when caller lacks events:write")
			return nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, &mockPermChecker{perms: teams.PermissionsJSON{Events: "read"}}, slog.Default())
	req := gen.SetNominationRequest{UserId: targetUserID, Nominated: true}

	err := svc.SetNomination(context.Background(), eventID.String(), callerID.String(), teamID.String(), req)
	require.ErrorIs(t, err, events.ErrSetNominationForbidden)
}

func TestEventService_SetNomination_NilPermChecker_Forbidden(t *testing.T) {
	t.Parallel()

	repo := &mockSvcRepo{
		setNominationFn: func(context.Context, string, string, string, string, bool) error {
			t.Fatal("repository must not be called when there is no permission checker")
			return nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	req := gen.SetNominationRequest{UserId: uuid.New(), Nominated: true}

	err := svc.SetNomination(context.Background(), uuid.New().String(), uuid.New().String(), uuid.New().String(), req)
	require.ErrorIs(t, err, events.ErrSetNominationForbidden)
}

func TestEventService_SetNomination_AllowedWithEventsWrite(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	callerID := uuid.New()
	targetUserID := uuid.New()
	teamID := uuid.New()

	called := false
	repo := &mockSvcRepo{
		setNominationFn: func(_ context.Context, evID, cID, uID, tID string, nominated bool) error {
			called = true
			assert.Equal(t, eventID.String(), evID)
			assert.Equal(t, callerID.String(), cID)
			assert.Equal(t, targetUserID.String(), uID)
			assert.Equal(t, teamID.String(), tID)
			assert.True(t, nominated)
			return nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, &mockPermChecker{perms: teams.PermissionsJSON{Events: "write"}}, slog.Default())
	req := gen.SetNominationRequest{UserId: targetUserID, Nominated: true}

	err := svc.SetNomination(context.Background(), eventID.String(), callerID.String(), teamID.String(), req)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestEventService_SetStatus_PassesTeamIDThrough(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	otherTeamID := uuid.New()
	row := svcMakeEventRow("Cancel Me")
	row.Id = eventID

	var capturedTeamID string
	repo := &mockSvcRepo{
		setStatusFn: func(_ context.Context, evID, tID, status, scope string) (*events.EventRow, error) {
			assert.Equal(t, eventID.String(), evID)
			capturedTeamID = tID
			assert.Equal(t, "cancelled", status)
			assert.Equal(t, "single", scope)
			return &row, nil
		},
		getAttendanceSummaryFn: zeroSummaryFn,
		getMyAttendanceFn:      nilMyAttendanceFn,
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	_, err := svc.SetStatus(context.Background(), testUserID, eventID.String(), otherTeamID.String(), "cancelled", "single")
	require.NoError(t, err)
	assert.Equal(t, otherTeamID.String(), capturedTeamID, "teamID must be threaded through to the repository so cross-team status changes are rejected at the DB layer")
}

func TestEventService_ListAttendance_RedactsDeclineReasonWithoutRole(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	otherUserID := uuid.New()
	reason := "private medical reason"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: otherUserID, Status: "no", Reason: &reason, Name: "Other"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			// Team requires role "trainer-role"; viewer has no roles at all.
			return []string{"trainer-role"}, nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Nil(t, rows[0].Reason, "a viewer without a reason-visibility role must not see another member's decline reason")
}

func TestEventService_ListAttendance_ShowsOwnDeclineReason(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	reason := "my own reason"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: viewerID, Status: "no", Reason: &reason, Name: "Self"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			t.Fatal("reason-visibility context must not be fetched for the viewer's own row")
			return nil, nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].Reason)
	assert.Equal(t, reason, *rows[0].Reason)
}

func TestEventService_ListAttendance_ShowsDeclineReasonWithMatchingRole(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	otherUserID := uuid.New()
	reason := "private medical reason"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: otherUserID, Status: "no", Reason: &reason, Name: "Other"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			return []string{"trainer-role"}, []string{"trainer-role"}, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].Reason)
	assert.Equal(t, reason, *rows[0].Reason)
}

// Regression test: a declining member choosing reasonVisibility="team" was
// silently ignored -- the redaction logic never read the per-row field at
// all, so their reason was redacted for any viewer outside the reason-
// visibility roles exactly as if they'd chosen "trainers", making the
// documented "team" option a no-op.
func TestEventService_ListAttendance_ShowsDeclineReasonWithTeamVisibility(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	otherUserID := uuid.New()
	reason := "shared with everyone"
	visibility := "team"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: otherUserID, Status: "no", Reason: &reason, ReasonVisibility: &visibility, Name: "Other"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			t.Fatal("reason-visibility role context must not be fetched for a row explicitly shared with the team")
			return nil, nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].Reason, "reasonVisibility=team must be visible to any teammate regardless of reason-visibility roles")
	assert.Equal(t, reason, *rows[0].Reason)
}

// A nil/unset ReasonVisibility (e.g. rows predating the field) must keep the
// more restrictive "trainers"-equivalent behavior, not be treated as an
// implicit "team".
func TestEventService_ListAttendance_NilReasonVisibility_StillRedacted(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	otherUserID := uuid.New()
	reason := "private medical reason"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: otherUserID, Status: "no", Reason: &reason, ReasonVisibility: nil, Name: "Other"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			return []string{"trainer-role"}, nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Nil(t, rows[0].Reason)
}

// Regression test: SetAttendance places no restriction on which status a
// reason may accompany (e.g. "yes, but running late"), but redaction used to
// only fire for status=="no" -- a private reason attached to "yes" or
// "maybe" leaked to every team member unredacted, regardless of
// reasonVisibility or the caller's reason-visibility role.
func TestEventService_ListAttendance_RedactsReasonOnNonDeclineStatus(t *testing.T) {
	t.Parallel()

	eventID := uuid.New()
	teamID := uuid.New()
	viewerID := uuid.New()
	otherUserID := uuid.New()
	reason := "running late, medical appointment"

	repo := &mockSvcRepo{
		listAttendanceFn: func(_ context.Context, _, _ string) ([]events.AttendanceEnriched, error) {
			return []events.AttendanceEnriched{
				{UserId: otherUserID, Status: "yes", Reason: &reason, Name: "Other"},
			}, nil
		},
		getReasonVisibilityCtxFn: func(_ context.Context, _, _ string) ([]string, []string, error) {
			return []string{"trainer-role"}, nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	rows, err := svc.ListAttendance(context.Background(), eventID.String(), teamID.String(), viewerID.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Nil(t, rows[0].Reason, "a reason attached to a non-'no' status must still be redacted from a viewer without a matching role")
}

// Regression test: AddComment used to have no per-event cap at all --
// events/comments is a self-service write reachable by any team member with
// no RBAC gate and no other natural bound, unlike finances' equivalent
// write paths (maxTransactionsPerTeam). Once an event reaches
// maxCommentsPerEvent, AddComment must reject the write instead of calling
// through to the repository.
func TestEventService_AddComment_TooManyComments_Blocked(t *testing.T) {
	t.Parallel()

	repo := &mockSvcRepo{
		countCommentsFn: func(context.Context, string, string) (int, error) {
			return 2000, nil
		},
		addCommentFn: func(context.Context, string, string, string, string) (*events.CommentRow, error) {
			t.Fatal("repository AddComment must not be called once the cap is reached")
			return nil, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	_, err := svc.AddComment(context.Background(), uuid.New().String(), uuid.New().String(), uuid.New().String(), "one too many")
	require.ErrorIs(t, err, events.ErrTooManyComments)
}

func TestEventService_AddComment_BelowCap_Allowed(t *testing.T) {
	t.Parallel()

	called := false
	repo := &mockSvcRepo{
		countCommentsFn: func(context.Context, string, string) (int, error) {
			return 1999, nil
		},
		addCommentFn: func(context.Context, string, string, string, string) (*events.CommentRow, error) {
			called = true
			return &events.CommentRow{Id: uuid.New(), Text: "hi", CreatedAt: time.Now()}, nil
		},
	}

	svc := events.NewService(repo, nil, nil, nil, nil, slog.Default())
	_, err := svc.AddComment(context.Background(), uuid.New().String(), uuid.New().String(), uuid.New().String(), "hi")
	require.NoError(t, err)
	assert.True(t, called, "repository AddComment must be called when below the cap")
}
