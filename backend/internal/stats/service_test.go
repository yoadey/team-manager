package stats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/stats"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	memberStatsFn       func(ctx context.Context, teamID uuid.UUID, from, to string) ([]stats.MemberStatRow, error)
	eventStatsFn        func(ctx context.Context, teamID uuid.UUID, from, to string) ([]stats.EventStatRow, error)
	singleMemberStatsFn func(ctx context.Context, teamID, userID uuid.UUID, from, to string) (*stats.MemberStatRow, error)
}

func (m *mockRepo) MemberStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]stats.MemberStatRow, error) {
	return m.memberStatsFn(ctx, teamID, from, to)
}

func (m *mockRepo) EventStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]stats.EventStatRow, error) {
	return m.eventStatsFn(ctx, teamID, from, to)
}

func (m *mockRepo) SingleMemberStats(ctx context.Context, teamID, userID uuid.UUID, from, to string) (*stats.MemberStatRow, error) {
	return m.singleMemberStatsFn(ctx, teamID, userID, from, to)
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestService_GetOverview_ComputesQuotesAndAverage(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	repo := &mockRepo{
		memberStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.MemberStatRow, error) {
			return []stats.MemberStatRow{
				{UserID: uuid.New(), Name: "Alice", Yes: 8, Counted: 10}, // 0.8
				{UserID: uuid.New(), Name: "Bob", Yes: 2, Counted: 10},   // 0.2
				{UserID: uuid.New(), Name: "Carol", Yes: 0, Counted: 0},  // no events counted -> 0
			}, nil
		},
		eventStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventStatRow, error) {
			return []stats.EventStatRow{
				{EventID: uuid.New(), Title: "Match", Type: "auftritt", Date: "2026-01-15", Yes: 6, Counted: 10},    // 0.6 -> Enough
				{EventID: uuid.New(), Title: "Training", Type: "training", Date: "2026-01-20", Yes: 3, Counted: 10}, // 0.3 -> not Enough
			}, nil
		},
	}

	svc := stats.NewService(repo)
	overview, err := svc.GetOverview(context.Background(), teamID, nil, nil)
	require.NoError(t, err)

	require.Len(t, overview.Members, 3)
	assert.InDelta(t, 0.8, overview.Members[0].Quote, 0.001)
	assert.InDelta(t, 0.2, overview.Members[1].Quote, 0.001)
	assert.InDelta(t, 0, overview.Members[2].Quote, 0.001, "a member with 0 counted events must have a 0 quote, not NaN or a divide-by-zero panic")

	assert.InDelta(t, 1.0/3.0, overview.Avg, 0.001, "average should be the mean of the three members' quotes above")

	require.Len(t, overview.Events, 2)
	assert.True(t, overview.Events[0].Enough, "0.6 attendance ratio should meet the 0.5 threshold")
	assert.False(t, overview.Events[1].Enough, "0.3 attendance ratio should not meet the 0.5 threshold")
	// Regression: Type used to be dropped entirely between EventStatRow and
	// gen.EventStat, always defaulting the client to the generic "event" icon.
	assert.Equal(t, gen.EventType("auftritt"), overview.Events[0].Type)
	assert.Equal(t, gen.EventType("training"), overview.Events[1].Type)
	assert.Equal(t, 2, overview.PastCount)
}

func TestService_GetOverview_DefaultsDateRangeWhenUnset(t *testing.T) {
	t.Parallel()

	var capturedFrom, capturedTo string
	repo := &mockRepo{
		memberStatsFn: func(_ context.Context, _ uuid.UUID, from, to string) ([]stats.MemberStatRow, error) {
			capturedFrom, capturedTo = from, to
			return nil, nil
		},
		eventStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventStatRow, error) { return nil, nil },
	}

	svc := stats.NewService(repo)
	_, err := svc.GetOverview(context.Background(), uuid.New(), nil, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, capturedFrom)
	assert.NotEmpty(t, capturedTo)
	assert.Less(t, capturedFrom, capturedTo, "default range should start before it ends")
}

func TestService_GetOverview_UsesExplicitDateRange(t *testing.T) {
	t.Parallel()

	var capturedFrom, capturedTo string
	repo := &mockRepo{
		memberStatsFn: func(_ context.Context, _ uuid.UUID, from, to string) ([]stats.MemberStatRow, error) {
			capturedFrom, capturedTo = from, to
			return nil, nil
		},
		eventStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventStatRow, error) { return nil, nil },
	}

	from := openapi_types.Date{Time: mustParseDate(t, "2026-02-01")}
	to := openapi_types.Date{Time: mustParseDate(t, "2026-02-28")}

	svc := stats.NewService(repo)
	_, err := svc.GetOverview(context.Background(), uuid.New(), &from, &to)
	require.NoError(t, err)
	assert.Equal(t, "2026-02-01", capturedFrom)
	assert.Equal(t, "2026-02-28", capturedTo)
}

// Regression test: from/to were previously passed straight into a Postgres
// BETWEEN clause with no bound on how far apart they could be, so a caller
// could force a full-history aggregation (e.g. from=0001-01-01) on every
// request. The effective range must now be clamped to maxStatsRangeDays.
func TestService_GetOverview_ClampsOversizedDateRange(t *testing.T) {
	t.Parallel()

	var capturedFrom, capturedTo string
	repo := &mockRepo{
		memberStatsFn: func(_ context.Context, _ uuid.UUID, from, to string) ([]stats.MemberStatRow, error) {
			capturedFrom, capturedTo = from, to
			return nil, nil
		},
		eventStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventStatRow, error) { return nil, nil },
	}

	from := openapi_types.Date{Time: mustParseDate(t, "0001-01-01")}
	to := openapi_types.Date{Time: mustParseDate(t, "2026-02-28")}

	svc := stats.NewService(repo)
	_, err := svc.GetOverview(context.Background(), uuid.New(), &from, &to)
	require.NoError(t, err)
	assert.Equal(t, "2026-02-28", capturedTo)
	gotFrom, err := time.Parse("2006-01-02", capturedFrom)
	require.NoError(t, err)
	gotTo, err := time.Parse("2006-01-02", capturedTo)
	require.NoError(t, err)
	assert.LessOrEqual(t, gotTo.Sub(gotFrom), 730*24*time.Hour)
}

func TestService_GetOverview_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("db unavailable")
	repo := &mockRepo{
		memberStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.MemberStatRow, error) {
			return nil, wantErr
		},
	}

	svc := stats.NewService(repo)
	_, err := svc.GetOverview(context.Background(), uuid.New(), nil, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, wantErr)
}

func TestService_GetMemberStats(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	repo := &mockRepo{
		singleMemberStatsFn: func(_ context.Context, gotTeamID, gotUserID uuid.UUID, _, _ string) (*stats.MemberStatRow, error) {
			assert.Equal(t, teamID, gotTeamID)
			assert.Equal(t, userID, gotUserID)
			return &stats.MemberStatRow{UserID: userID, Yes: 4, Counted: 5}, nil
		},
	}

	svc := stats.NewService(repo)
	result, err := svc.GetMemberStats(context.Background(), teamID, userID, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, result.Yes)
	assert.Equal(t, 5, result.Counted)
	assert.InDelta(t, 0.8, result.Quote, 0.001)
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return parsed
}
