package stats_test

import (
	"context"
	"encoding/json"
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
	withReadTxFn        func(ctx context.Context, fn func(stats.OverviewReader) error) error
	attendanceMatrixFn  func(ctx context.Context, teamID uuid.UUID, from, to string) ([]stats.MatrixColumnRow, []stats.MatrixCellRow, error)
	eventGroupStatsFn   func(ctx context.Context, teamID uuid.UUID, from, to string) ([]stats.EventGroupStatRow, error)
	listThresholdsFn    func(ctx context.Context, teamID uuid.UUID) ([]stats.GroupThresholdRow, error)
	setThresholdsFn     func(ctx context.Context, teamID uuid.UUID, items []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error)
}

// EventGroupStats defaults to "no groups" when a test doesn't set
// eventGroupStatsFn, so the pre-existing GetOverview tests (which predate
// the per-group breakdown) need no change.
func (m *mockRepo) EventGroupStats(ctx context.Context, teamID uuid.UUID, from, to string) ([]stats.EventGroupStatRow, error) {
	if m.eventGroupStatsFn == nil {
		return nil, nil
	}
	return m.eventGroupStatsFn(ctx, teamID, from, to)
}

func (m *mockRepo) ListGroupThresholds(ctx context.Context, teamID uuid.UUID) ([]stats.GroupThresholdRow, error) {
	return m.listThresholdsFn(ctx, teamID)
}

func (m *mockRepo) SetGroupThresholds(ctx context.Context, teamID uuid.UUID, items []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error) {
	return m.setThresholdsFn(ctx, teamID, items)
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

// WithReadTx runs fn directly against the mock itself (which already
// implements stats.OverviewReader), since unit tests have no live
// transaction to hand out.
func (m *mockRepo) WithReadTx(ctx context.Context, fn func(stats.OverviewReader) error) error {
	if m.withReadTxFn != nil {
		return m.withReadTxFn(ctx, fn)
	}
	return fn(m)
}

func (m *mockRepo) AttendanceMatrix(ctx context.Context, teamID uuid.UUID, from, to string) ([]stats.MatrixColumnRow, []stats.MatrixCellRow, error) {
	return m.attendanceMatrixFn(ctx, teamID, from, to)
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

	// Carol has no counted events (no data yet), so she must be excluded
	// from the average entirely rather than averaged in as a 0% -- the
	// average is the mean of only Alice's and Bob's quotes.
	assert.InDelta(t, 0.5, overview.Avg, 0.001, "average should exclude members with zero counted events (no data), not score them as 0%")

	require.Len(t, overview.Events, 2)
	assert.True(t, overview.Events[0].Enough, "0.6 attendance ratio should meet the 0.5 threshold")
	assert.False(t, overview.Events[1].Enough, "0.3 attendance ratio should not meet the 0.5 threshold")
	// Regression: Type used to be dropped entirely between EventStatRow and
	// gen.EventStat, always defaulting the client to the generic "event" icon.
	assert.Equal(t, gen.EventType("auftritt"), overview.Events[0].Type)
	assert.Equal(t, gen.EventType("training"), overview.Events[1].Type)
	assert.Equal(t, 2, overview.PastCount)
}

// TestService_GetOverview_AverageExcludesMembersWithNoCountedEvents pins the
// no-data-vs-0% distinction for the team-wide average in isolation: members
// with Counted == 0 must be dropped from both the numerator and denominator,
// and the all-zero case must yield avg == 0 without panicking (no
// division by zero).
func TestService_GetOverview_AverageExcludesMembersWithNoCountedEvents(t *testing.T) {
	t.Parallel()

	t.Run("mix of members with and without counted events", func(t *testing.T) {
		t.Parallel()

		repo := &mockRepo{
			memberStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.MemberStatRow, error) {
				return []stats.MemberStatRow{
					{UserID: uuid.New(), Name: "Alice", Yes: 10, Counted: 10}, // 1.0
					{UserID: uuid.New(), Name: "Bob", Yes: 0, Counted: 0},     // no data -- excluded
					{UserID: uuid.New(), Name: "Carol", Yes: 0, Counted: 0},   // no data -- excluded
				}, nil
			},
			eventStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventStatRow, error) {
				return nil, nil
			},
		}

		svc := stats.NewService(repo)
		overview, err := svc.GetOverview(context.Background(), uuid.New(), nil, nil)
		require.NoError(t, err)

		require.Len(t, overview.Members, 3)
		// Per-member quotes are unaffected: the two no-data members still
		// report 0 individually (the frontend maps this to "-", not this
		// backend field).
		assert.InDelta(t, 1.0, overview.Members[0].Quote, 0.001)
		assert.InDelta(t, 0, overview.Members[1].Quote, 0.001)
		assert.InDelta(t, 0, overview.Members[2].Quote, 0.001)

		// The average must be Alice's 1.0 alone -- Bob and Carol contribute
		// neither to the sum nor the count.
		assert.InDelta(t, 1.0, overview.Avg, 0.001, "average must only be computed over members with at least one counted event")
	})

	t.Run("no member has any counted events", func(t *testing.T) {
		t.Parallel()

		repo := &mockRepo{
			memberStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.MemberStatRow, error) {
				return []stats.MemberStatRow{
					{UserID: uuid.New(), Name: "Alice", Yes: 0, Counted: 0},
					{UserID: uuid.New(), Name: "Bob", Yes: 0, Counted: 0},
				}, nil
			},
			eventStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventStatRow, error) {
				return nil, nil
			},
		}

		svc := stats.NewService(repo)
		overview, err := svc.GetOverview(context.Background(), uuid.New(), nil, nil)
		require.NoError(t, err)

		require.Len(t, overview.Members, 2)
		assert.InDelta(t, 0, overview.Avg, 0.001, "with nothing to average, avg must be 0 rather than NaN or a panic")
	})
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

// Pins the exact default window width so a future change to defaultDateRange
// (or to how GetMemberStats/GetOverview call it) can't silently drift from
// "3 months" without a test failing -- this is the path every stats endpoint
// takes when the caller omits from/to.
func TestService_GetOverview_DefaultRangeIsExactlyThreeMonths(t *testing.T) {
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

	gotFrom, err := time.Parse("2006-01-02", capturedFrom)
	require.NoError(t, err)
	gotTo, err := time.Parse("2006-01-02", capturedTo)
	require.NoError(t, err)

	wantFrom := gotTo.AddDate(0, -3, 0)
	assert.Equal(t, wantFrom.Format("2006-01-02"), gotFrom.Format("2006-01-02"))
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

// Pins that GetMemberStats' from/to (now also accepted by the handler, via
// GetMemberStatsParams) are used verbatim rather than falling back to the
// default (defaultDateRange only falls back when nil), so a future change
// can't silently start ignoring an explicitly-passed range.
func TestService_GetMemberStats_UsesExplicitRangeWhenGiven(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	var capturedFrom, capturedTo string
	repo := &mockRepo{
		singleMemberStatsFn: func(_ context.Context, _, _ uuid.UUID, from, to string) (*stats.MemberStatRow, error) {
			capturedFrom, capturedTo = from, to
			return &stats.MemberStatRow{UserID: userID, Yes: 1, Counted: 1}, nil
		},
	}

	from := openapi_types.Date{Time: mustParseDate(t, "2026-01-01")}
	to := openapi_types.Date{Time: mustParseDate(t, "2026-01-31")}

	svc := stats.NewService(repo)
	_, err := svc.GetMemberStats(context.Background(), teamID, userID, &from, &to)
	require.NoError(t, err)
	assert.Equal(t, "2026-01-01", capturedFrom)
	assert.Equal(t, "2026-01-31", capturedTo)
}

func TestService_GetAttendanceMatrix_AssemblesSortsAndReconciles(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	e1, e2 := uuid.New(), uuid.New()
	alice, bob := uuid.New(), uuid.New()

	repo := &mockRepo{
		attendanceMatrixFn: func(context.Context, uuid.UUID, string, string) ([]stats.MatrixColumnRow, []stats.MatrixCellRow, error) {
			cols := []stats.MatrixColumnRow{
				{EventID: e1, Title: "Training A", Type: "training", Date: "2026-01-01"},
				{EventID: e2, Title: "Match B", Type: "auftritt", Date: "2026-01-06"},
			}
			// Cells arrive name-ordered (Alice before Bob), but Bob attends more,
			// so Bob must sort first. Alice: yes + pending. Bob: yes + yes.
			cells := []stats.MatrixCellRow{
				{UserID: alice, Name: "Alice", AvatarColor: "#a", EventID: &e1, Eff: "yes"},
				{UserID: alice, Name: "Alice", AvatarColor: "#a", EventID: &e2, Eff: "pending"},
				{UserID: bob, Name: "Bob", AvatarColor: "#b", EventID: &e1, Eff: "yes"},
				{UserID: bob, Name: "Bob", AvatarColor: "#b", EventID: &e2, Eff: "yes"},
			}
			return cols, cells, nil
		},
	}

	svc := stats.NewService(repo)
	m, err := svc.GetAttendanceMatrix(context.Background(), teamID, nil, nil)
	require.NoError(t, err)

	require.Len(t, m.Events, 2)
	assert.Equal(t, e1, m.Events[0].Id, "columns keep the query order (date asc)")
	assert.Equal(t, e2, m.Events[1].Id)

	require.Len(t, m.Members, 2)
	assert.Equal(t, "Bob", m.Members[0].Name, "most-attending member sorts first")
	assert.Equal(t, "Alice", m.Members[1].Name)

	// Row aggregate reconciles with the cells: Bob 2 yes, Alice 1 yes.
	assert.Equal(t, 2, m.Members[0].Yes)
	assert.Equal(t, 2, m.Members[0].Counted)
	assert.Equal(t, 1, m.Members[1].Yes)
	assert.Equal(t, 1, m.Members[1].Counted, "pending is not counted, only yes/no/maybe")

	// Cells keyed by event id.
	assert.Equal(t, gen.AttendanceStatus("yes"), m.Members[1].Cells[e1.String()])
	assert.Equal(t, gen.AttendanceStatus("pending"), m.Members[1].Cells[e2.String()])
}

func TestService_GetAttendanceMatrix_MemberWithNoEvents(t *testing.T) {
	t.Parallel()

	solo := uuid.New()
	repo := &mockRepo{
		attendanceMatrixFn: func(context.Context, uuid.UUID, string, string) ([]stats.MatrixColumnRow, []stats.MatrixCellRow, error) {
			// LEFT JOIN placeholder: a member with no events in range yields one
			// row with a nil EventID. The member must still appear, cells empty.
			cells := []stats.MatrixCellRow{
				{UserID: solo, Name: "Solo", AvatarColor: "#s", EventID: nil, Eff: "pending"},
			}
			return nil, cells, nil
		},
	}

	svc := stats.NewService(repo)
	m, err := svc.GetAttendanceMatrix(context.Background(), uuid.New(), nil, nil)
	require.NoError(t, err)
	require.Len(t, m.Members, 1)
	assert.Equal(t, "Solo", m.Members[0].Name)
	assert.Equal(t, 0, m.Members[0].Yes)
	assert.Empty(t, m.Members[0].Cells, "a placeholder nil-event row must not create a cell")
}

func TestService_GetAttendanceMatrix_DefaultsDateRangeWhenUnset(t *testing.T) {
	t.Parallel()

	var capturedFrom, capturedTo string
	repo := &mockRepo{
		attendanceMatrixFn: func(_ context.Context, _ uuid.UUID, from, to string) ([]stats.MatrixColumnRow, []stats.MatrixCellRow, error) {
			capturedFrom, capturedTo = from, to
			return nil, nil, nil
		},
	}

	svc := stats.NewService(repo)
	_, err := svc.GetAttendanceMatrix(context.Background(), uuid.New(), nil, nil)
	require.NoError(t, err)

	gotFrom, err := time.Parse("2006-01-02", capturedFrom)
	require.NoError(t, err)
	gotTo, err := time.Parse("2006-01-02", capturedTo)
	require.NoError(t, err)
	assert.Equal(t, gotTo.AddDate(0, -3, 0).Format("2006-01-02"), gotFrom.Format("2006-01-02"),
		"matrix must reuse the overview's 3-month default window")
}

// TestService_GetOverview_AttachesGroupsToTheirEvent pins that each event's
// Groups holds exactly the group rows of that event, in the repository's
// order, with the threshold carried through -- and that an event with no
// group rows serializes as an empty array, never null (Groups is a required
// array in the schema).
func TestService_GetOverview_AttachesGroupsToTheirEvent(t *testing.T) {
	t.Parallel()

	ev1, ev2, ev3 := uuid.New(), uuid.New(), uuid.New()
	trainer, member := uuid.New(), uuid.New()
	blue := "#1565C0"
	repo := &mockRepo{
		memberStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.MemberStatRow, error) {
			return nil, nil
		},
		eventStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventStatRow, error) {
			return []stats.EventStatRow{
				{EventID: ev1, Title: "Match", Type: "auftritt", Date: "2026-01-15", Yes: 5, Counted: 6},
				{EventID: ev2, Title: "Training", Type: "training", Date: "2026-01-20", Yes: 2, Counted: 6},
				{EventID: ev3, Title: "Fest", Type: "event", Date: "2026-01-25", Yes: 0, Counted: 0},
			}, nil
		},
		eventGroupStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventGroupStatRow, error) {
			return []stats.EventGroupStatRow{
				{EventID: ev1, RoleID: trainer, Name: "Trainer", Color: &blue, Yes: 2, Counted: 2, ThresholdMode: "count", ThresholdValue: 1},
				{EventID: ev1, RoleID: member, Name: "Mitglied", Yes: 5, Counted: 6, ThresholdMode: "percent", ThresholdValue: 50},
				{EventID: ev2, RoleID: trainer, Name: "Trainer", Color: &blue, Yes: 0, Counted: 2, ThresholdMode: "count", ThresholdValue: 1},
				{EventID: ev2, RoleID: member, Name: "Mitglied", Yes: 2, Counted: 6, ThresholdMode: "percent", ThresholdValue: 50},
			}, nil
		},
	}

	overview, err := stats.NewService(repo).GetOverview(context.Background(), uuid.New(), nil, nil)
	require.NoError(t, err)
	require.Len(t, overview.Events, 3)

	g1 := overview.Events[0].Groups
	require.Len(t, g1, 2)
	assert.Equal(t, trainer, g1[0].RoleId)
	assert.Equal(t, "Trainer", g1[0].Name)
	require.NotNil(t, g1[0].Color)
	assert.Equal(t, blue, *g1[0].Color)
	assert.Equal(t, 2, g1[0].Yes)
	assert.Equal(t, 2, g1[0].Counted)
	assert.Equal(t, gen.StatsThresholdMode("count"), g1[0].ThresholdMode)
	assert.Equal(t, 1, g1[0].ThresholdValue)
	assert.Equal(t, member, g1[1].RoleId)
	assert.Nil(t, g1[1].Color)
	assert.Equal(t, gen.StatsThresholdMode("percent"), g1[1].ThresholdMode)
	assert.Equal(t, 50, g1[1].ThresholdValue)

	g2 := overview.Events[1].Groups
	require.Len(t, g2, 2)
	assert.Equal(t, 0, g2[0].Yes)
	assert.Equal(t, 2, g2[1].Yes)

	require.NotNil(t, overview.Events[2].Groups, "an event without group rows must serialize as [], not null")
	assert.Empty(t, overview.Events[2].Groups)
	raw, err := json.Marshal(overview.Events[2])
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"groups":[]`)
}

func TestService_GetOverview_PropagatesGroupStatsError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("group query failed")
	repo := &mockRepo{
		memberStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.MemberStatRow, error) {
			return nil, nil
		},
		eventStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventStatRow, error) {
			return nil, nil
		},
		eventGroupStatsFn: func(context.Context, uuid.UUID, string, string) ([]stats.EventGroupStatRow, error) {
			return nil, wantErr
		},
	}
	_, err := stats.NewService(repo).GetOverview(context.Background(), uuid.New(), nil, nil)
	require.ErrorIs(t, err, wantErr)
}

func TestService_ListGroupThresholds(t *testing.T) {
	t.Parallel()

	teamID, roleID := uuid.New(), uuid.New()
	repo := &mockRepo{
		listThresholdsFn: func(_ context.Context, gotTeam uuid.UUID) ([]stats.GroupThresholdRow, error) {
			assert.Equal(t, teamID, gotTeam)
			return []stats.GroupThresholdRow{{RoleID: roleID, Mode: "percent", Value: 50}}, nil
		},
	}
	rows, err := stats.NewService(repo).ListGroupThresholds(context.Background(), teamID)
	require.NoError(t, err)
	assert.Equal(t, []stats.GroupThresholdRow{{RoleID: roleID, Mode: "percent", Value: 50}}, rows)

	wantErr := errors.New("db unavailable")
	repo.listThresholdsFn = func(context.Context, uuid.UUID) ([]stats.GroupThresholdRow, error) { return nil, wantErr }
	_, err = stats.NewService(repo).ListGroupThresholds(context.Background(), teamID)
	require.ErrorIs(t, err, wantErr)
}

func TestService_SetGroupThresholds_PassesValidItemsThrough(t *testing.T) {
	t.Parallel()

	teamID, r1, r2 := uuid.New(), uuid.New(), uuid.New()
	// Boundary values of both modes are all valid.
	items := []stats.GroupThresholdRow{
		{RoleID: r1, Mode: "percent", Value: 0},
		{RoleID: r1, Mode: "percent", Value: 100},
		{RoleID: r2, Mode: "count", Value: 0},
		{RoleID: r2, Mode: "count", Value: 10000},
	}
	var got []stats.GroupThresholdRow
	repo := &mockRepo{
		setThresholdsFn: func(_ context.Context, gotTeam uuid.UUID, in []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error) {
			assert.Equal(t, teamID, gotTeam)
			got = in
			return in, nil
		},
	}
	rows, err := stats.NewService(repo).SetGroupThresholds(context.Background(), teamID, items)
	require.NoError(t, err)
	assert.Equal(t, items, got)
	assert.Equal(t, items, rows)
}

func TestService_SetGroupThresholds_RejectsInvalidItemsWithoutWriting(t *testing.T) {
	t.Parallel()

	roleID := uuid.New()
	valid := stats.GroupThresholdRow{RoleID: roleID, Mode: "percent", Value: 50}
	cases := map[string]stats.GroupThresholdRow{
		"unknown mode":      {RoleID: roleID, Mode: "ratio", Value: 10},
		"empty mode":        {RoleID: roleID, Mode: "", Value: 10},
		"negative percent":  {RoleID: roleID, Mode: "percent", Value: -1},
		"percent above 100": {RoleID: roleID, Mode: "percent", Value: 101},
		"negative count":    {RoleID: roleID, Mode: "count", Value: -1},
		"count above cap":   {RoleID: roleID, Mode: "count", Value: 10001},
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo := &mockRepo{
				setThresholdsFn: func(context.Context, uuid.UUID, []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error) {
					t.Error("repository must not be called when any item is invalid")
					return nil, nil
				},
			}
			// A valid item ahead of the bad one must not be written either.
			_, err := stats.NewService(repo).SetGroupThresholds(context.Background(), uuid.New(), []stats.GroupThresholdRow{valid, bad})
			require.ErrorIs(t, err, stats.ErrInvalidThreshold)
		})
	}
}

func TestService_SetGroupThresholds_PropagatesRoleNotInTeam(t *testing.T) {
	t.Parallel()

	repo := &mockRepo{
		setThresholdsFn: func(context.Context, uuid.UUID, []stats.GroupThresholdRow) ([]stats.GroupThresholdRow, error) {
			return nil, stats.ErrRoleNotInTeam
		},
	}
	_, err := stats.NewService(repo).SetGroupThresholds(context.Background(), uuid.New(),
		[]stats.GroupThresholdRow{{RoleID: uuid.New(), Mode: "count", Value: 3}})
	require.ErrorIs(t, err, stats.ErrRoleNotInTeam)
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return parsed
}
