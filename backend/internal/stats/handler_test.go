package stats_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/stats"
)

// ─── mock service ────────────────────────────────────────────────────────────

type mockStatsService struct {
	getOverview    func(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.StatsOverview, error)
	getMemberStats func(ctx context.Context, teamID, userID uuid.UUID, from, to *openapi_types.Date) (*gen.MemberAttendanceStats, error)
}

func (m *mockStatsService) GetOverview(ctx context.Context, teamID uuid.UUID, from, to *openapi_types.Date) (*gen.StatsOverview, error) {
	return m.getOverview(ctx, teamID, from, to)
}

func (m *mockStatsService) GetMemberStats(ctx context.Context, teamID, userID uuid.UUID, from, to *openapi_types.Date) (*gen.MemberAttendanceStats, error) {
	return m.getMemberStats(ctx, teamID, userID, from, to)
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
