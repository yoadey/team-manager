package stats_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/stats"
)

// ─── mock service ────────────────────────────────────────────────────────────

type mockStatsService struct {
	getOverview      func(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.StatsOverview, error)
	getMemberStats   func(ctx context.Context, teamID, userID uuid.UUID, from, to *openapi_types.Date) (*gen.MemberAttendanceStats, error)
	getAttendanceMat func(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.AttendanceMatrix, error)
	listThresholds   func(ctx context.Context, teamID uuid.UUID) ([]stats.GroupThresholdRow, error)
	setThresholds    func(ctx context.Context, teamID uuid.UUID, items []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error)
}

func (m *mockStatsService) ListGroupThresholds(ctx context.Context, teamID uuid.UUID) ([]stats.GroupThresholdRow, error) {
	return m.listThresholds(ctx, teamID)
}

func (m *mockStatsService) SetGroupThresholds(ctx context.Context, teamID uuid.UUID, items []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error) {
	return m.setThresholds(ctx, teamID, items)
}

func (m *mockStatsService) GetOverview(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.StatsOverview, error) {
	return m.getOverview(ctx, teamID, from, to)
}

func (m *mockStatsService) GetMemberStats(ctx context.Context, teamID, userID uuid.UUID, from, to *openapi_types.Date) (*gen.MemberAttendanceStats, error) {
	return m.getMemberStats(ctx, teamID, userID, from, to)
}

func (m *mockStatsService) GetAttendanceMatrix(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.AttendanceMatrix, error) {
	return m.getAttendanceMat(ctx, teamID, from, to)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

var (
	statsTeamID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	statsUserID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
)

func statsAuthedCtx() context.Context {
	user := &auth.UserRow{
		Id:          statsTeamID,
		Name:        "Test User",
		Email:       "test@example.com",
		AvatarColor: "#6366f1",
		CreatedAt:   time.Now(),
	}
	return auth.ContextWithUser(context.Background(), user)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestHandler_GetStatsOverview_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := stats.NewHandler(&mockStatsService{}, slog.Default())
	_, err := h.GetStatsOverview(context.Background(), gen.GetStatsOverviewRequestObject{TeamId: statsTeamID})
	require.Error(t, err)
}

func TestHandler_GetAttendanceMatrix_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := stats.NewHandler(&mockStatsService{}, slog.Default())
	_, err := h.GetAttendanceMatrix(context.Background(), gen.GetAttendanceMatrixRequestObject{TeamId: statsTeamID})
	require.Error(t, err)
}

func TestHandler_GetAttendanceMatrix_Success(t *testing.T) {
	t.Parallel()

	matrix := &gen.AttendanceMatrix{
		From:    openapi_types.Date{Time: time.Now().AddDate(0, -3, 0)},
		To:      openapi_types.Date{Time: time.Now()},
		Events:  []gen.AttendanceMatrixColumn{},
		Members: []gen.AttendanceMatrixRow{},
	}
	svc := &mockStatsService{
		getAttendanceMat: func(_ context.Context, _ uuid.UUID, _, _ *openapi_types.Date) (*gen.AttendanceMatrix, error) {
			return matrix, nil
		},
	}
	h := stats.NewHandler(svc, slog.Default())

	resp, err := h.GetAttendanceMatrix(statsAuthedCtx(), gen.GetAttendanceMatrixRequestObject{TeamId: statsTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetAttendanceMatrixResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetStatsOverview_Success(t *testing.T) {
	t.Parallel()

	overview := &gen.StatsOverview{
		From:      openapi_types.Date{Time: time.Now().AddDate(0, -3, 0)},
		To:        openapi_types.Date{Time: time.Now()},
		Members:   []gen.MemberStat{},
		Events:    []gen.EventStat{},
		Avg:       0.75,
		PastCount: 10,
	}
	svc := &mockStatsService{
		getOverview: func(_ context.Context, _ uuid.UUID, _, _ *openapi_types.Date) (*gen.StatsOverview, error) {
			return overview, nil
		},
	}
	h := stats.NewHandler(svc, slog.Default())

	resp, err := h.GetStatsOverview(statsAuthedCtx(), gen.GetStatsOverviewRequestObject{TeamId: statsTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetStatsOverviewResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)

	var result gen.StatsOverview
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, 10, result.PastCount)
}

func TestHandler_GetStatsOverview_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockStatsService{
		getOverview: func(_ context.Context, _ uuid.UUID, _, _ *openapi_types.Date) (*gen.StatsOverview, error) {
			return nil, errors.New("db error")
		},
	}
	h := stats.NewHandler(svc, slog.Default())
	_, err := h.GetStatsOverview(statsAuthedCtx(), gen.GetStatsOverviewRequestObject{TeamId: statsTeamID})
	require.Error(t, err)
}

func TestHandler_GetMemberStats_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := stats.NewHandler(&mockStatsService{}, slog.Default())
	_, err := h.GetMemberStats(context.Background(), gen.GetMemberStatsRequestObject{
		TeamId: statsTeamID,
		UserId: statsUserID,
	})
	require.Error(t, err)
}

func TestHandler_GetMemberStats_Success(t *testing.T) {
	t.Parallel()
	memberStats := &gen.MemberAttendanceStats{
		Yes:     8,
		Counted: 10,
		Quote:   0.8,
	}
	svc := &mockStatsService{
		getMemberStats: func(_ context.Context, _, _ uuid.UUID, _, _ *openapi_types.Date) (*gen.MemberAttendanceStats, error) {
			return memberStats, nil
		},
	}
	h := stats.NewHandler(svc, slog.Default())

	resp, err := h.GetMemberStats(statsAuthedCtx(), gen.GetMemberStatsRequestObject{
		TeamId: statsTeamID,
		UserId: statsUserID,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetMemberStatsResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)

	var result gen.MemberAttendanceStats
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Equal(t, 8, result.Yes)
	assert.InEpsilon(t, float32(0.8), result.Quote, 0.001)
}

func TestHandler_GetMemberStats_ForwardsFromTo(t *testing.T) {
	t.Parallel()
	from := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	to := openapi_types.Date{Time: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)}

	var gotFrom, gotTo *openapi_types.Date
	svc := &mockStatsService{
		getMemberStats: func(_ context.Context, _, _ uuid.UUID, from, to *openapi_types.Date) (*gen.MemberAttendanceStats, error) {
			gotFrom, gotTo = from, to
			return &gen.MemberAttendanceStats{}, nil
		},
	}
	h := stats.NewHandler(svc, slog.Default())

	_, err := h.GetMemberStats(statsAuthedCtx(), gen.GetMemberStatsRequestObject{
		TeamId: statsTeamID,
		UserId: statsUserID,
		Params: gen.GetMemberStatsParams{From: &from, To: &to},
	})
	require.NoError(t, err)
	require.NotNil(t, gotFrom)
	require.NotNil(t, gotTo)
	assert.True(t, from.Equal(gotFrom.Time))
	assert.True(t, to.Equal(gotTo.Time))
}

func TestHandler_GetMemberStats_NonMemberReturnsNotFound(t *testing.T) {
	t.Parallel()
	svc := &mockStatsService{
		getMemberStats: func(_ context.Context, _, _ uuid.UUID, _, _ *openapi_types.Date) (*gen.MemberAttendanceStats, error) {
			return nil, pgx.ErrNoRows
		},
	}
	h := stats.NewHandler(svc, slog.Default())

	_, err := h.GetMemberStats(statsAuthedCtx(), gen.GetMemberStatsRequestObject{
		TeamId: statsTeamID,
		UserId: statsUserID,
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Status)
}

// ─── group thresholds ────────────────────────────────────────────────────────

func TestHandler_ListStatsGroupThresholds_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := stats.NewHandler(&mockStatsService{}, slog.Default())
	_, err := h.ListStatsGroupThresholds(context.Background(), gen.ListStatsGroupThresholdsRequestObject{TeamId: statsTeamID})
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Status)
}

func TestHandler_ListStatsGroupThresholds_Success(t *testing.T) {
	t.Parallel()

	r1, r2 := uuid.New(), uuid.New()
	svc := &mockStatsService{
		listThresholds: func(_ context.Context, teamID uuid.UUID) ([]stats.GroupThresholdRow, error) {
			assert.Equal(t, statsTeamID, teamID)
			return []stats.GroupThresholdRow{
				{RoleID: r1, Mode: "percent", Value: 50},
				{RoleID: r2, Mode: "count", Value: 8},
			}, nil
		},
	}
	h := stats.NewHandler(svc, slog.Default())
	resp, err := h.ListStatsGroupThresholds(statsAuthedCtx(), gen.ListStatsGroupThresholdsRequestObject{TeamId: statsTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListStatsGroupThresholdsResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
	var body gen.StatsGroupThresholdList
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Items, 2)
	assert.Equal(t, gen.StatsGroupThreshold{RoleId: r1, Mode: "percent", Value: 50}, body.Items[0])
	assert.Equal(t, gen.StatsGroupThreshold{RoleId: r2, Mode: "count", Value: 8}, body.Items[1])
}

// An empty team role list must serialize as {"items":[]}, not null.
func TestHandler_ListStatsGroupThresholds_EmptyIsArray(t *testing.T) {
	t.Parallel()
	svc := &mockStatsService{
		listThresholds: func(context.Context, uuid.UUID) ([]stats.GroupThresholdRow, error) { return nil, nil },
	}
	resp, err := stats.NewHandler(svc, slog.Default()).ListStatsGroupThresholds(statsAuthedCtx(),
		gen.ListStatsGroupThresholdsRequestObject{TeamId: statsTeamID})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListStatsGroupThresholdsResponse(w))
	assert.JSONEq(t, `{"items":[]}`, w.Body.String())
}

func TestHandler_ListStatsGroupThresholds_ServiceError(t *testing.T) {
	t.Parallel()
	svc := &mockStatsService{
		listThresholds: func(context.Context, uuid.UUID) ([]stats.GroupThresholdRow, error) {
			return nil, errors.New("db unavailable")
		},
	}
	_, err := stats.NewHandler(svc, slog.Default()).ListStatsGroupThresholds(statsAuthedCtx(),
		gen.ListStatsGroupThresholdsRequestObject{TeamId: statsTeamID})
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusInternalServerError, apiErr.Status)
}

func TestHandler_SetStatsGroupThresholds_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := stats.NewHandler(&mockStatsService{}, slog.Default())
	_, err := h.SetStatsGroupThresholds(context.Background(), gen.SetStatsGroupThresholdsRequestObject{
		TeamId: statsTeamID,
		Body:   &gen.SetStatsGroupThresholdsJSONRequestBody{},
	})
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Status)
}

func TestHandler_SetStatsGroupThresholds_MissingBody(t *testing.T) {
	t.Parallel()
	h := stats.NewHandler(&mockStatsService{}, slog.Default())
	_, err := h.SetStatsGroupThresholds(statsAuthedCtx(), gen.SetStatsGroupThresholdsRequestObject{TeamId: statsTeamID})
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
}

func TestHandler_SetStatsGroupThresholds_Success(t *testing.T) {
	t.Parallel()

	roleID := uuid.New()
	var got []stats.GroupThresholdRow
	svc := &mockStatsService{
		setThresholds: func(_ context.Context, teamID uuid.UUID, items []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error) {
			assert.Equal(t, statsTeamID, teamID)
			got = items
			return items, nil
		},
	}
	h := stats.NewHandler(svc, slog.Default())
	resp, err := h.SetStatsGroupThresholds(statsAuthedCtx(), gen.SetStatsGroupThresholdsRequestObject{
		TeamId: statsTeamID,
		Body: &gen.SetStatsGroupThresholdsJSONRequestBody{
			Items: []gen.StatsGroupThreshold{{RoleId: roleID, Mode: gen.StatsThresholdMode("count"), Value: 8}},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []stats.GroupThresholdRow{{RoleID: roleID, Mode: "count", Value: 8}}, got)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitSetStatsGroupThresholdsResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
	var body gen.StatsGroupThresholdList
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	assert.Equal(t, gen.StatsGroupThreshold{RoleId: roleID, Mode: "count", Value: 8}, body.Items[0])
}

func TestHandler_SetStatsGroupThresholds_TooManyItems(t *testing.T) {
	t.Parallel()
	svc := &mockStatsService{
		setThresholds: func(context.Context, uuid.UUID, []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error) {
			t.Error("service must not be called for an oversized request")
			return nil, nil
		},
	}
	items := make([]gen.StatsGroupThreshold, 201)
	for i := range items {
		items[i] = gen.StatsGroupThreshold{RoleId: uuid.New(), Mode: "percent", Value: 50}
	}
	_, err := stats.NewHandler(svc, slog.Default()).SetStatsGroupThresholds(statsAuthedCtx(), gen.SetStatsGroupThresholdsRequestObject{
		TeamId: statsTeamID,
		Body:   &gen.SetStatsGroupThresholdsJSONRequestBody{Items: items},
	})
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
}

func TestHandler_SetStatsGroupThresholds_ErrorMapping(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		svcErr error
		status int
	}{
		"invalid threshold": {fmt.Errorf("wrapped: %w", stats.ErrInvalidThreshold), http.StatusBadRequest},
		"role not in team":  {fmt.Errorf("wrapped: %w", stats.ErrRoleNotInTeam), http.StatusUnprocessableEntity},
		"unexpected":        {errors.New("db unavailable"), http.StatusInternalServerError},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			svc := &mockStatsService{
				setThresholds: func(context.Context, uuid.UUID, []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error) {
					return nil, tc.svcErr
				},
			}
			_, err := stats.NewHandler(svc, slog.Default()).SetStatsGroupThresholds(statsAuthedCtx(), gen.SetStatsGroupThresholdsRequestObject{
				TeamId: statsTeamID,
				Body: &gen.SetStatsGroupThresholdsJSONRequestBody{
					Items: []gen.StatsGroupThreshold{{RoleId: uuid.New(), Mode: "percent", Value: 50}},
				},
			})
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tc.status, apiErr.Status)
		})
	}
}
