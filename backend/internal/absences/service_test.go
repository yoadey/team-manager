package absences_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/absences"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	listByTeam        func(ctx context.Context, teamID uuid.UUID, limit int, cur *absences.ListCursor) ([]*absences.AbsenceRow, error)
	listByUser        func(ctx context.Context, teamID, userID uuid.UUID, limit int, cur *absences.ListCursor) ([]*absences.AbsenceRow, error)
	create            func(ctx context.Context, teamID, userID uuid.UUID, fromDate, toDate string, reason *string) (*absences.AbsenceRow, error)
	update            func(ctx context.Context, id, teamID, userID uuid.UUID, fromDate, toDate, reason *string) (*absences.AbsenceRow, error)
	delete            func(ctx context.Context, id, teamID, userID uuid.UUID) error
	getOwner          func(ctx context.Context, id, teamID uuid.UUID) (uuid.UUID, error)
	setStatsRelevance func(ctx context.Context, id, teamID uuid.UUID, notRelevant bool, setBy uuid.UUID) (*absences.AbsenceRow, error)
}

func (m *mockRepo) GetOwner(ctx context.Context, id, teamID uuid.UUID) (uuid.UUID, error) {
	return m.getOwner(ctx, id, teamID)
}

func (m *mockRepo) SetStatsRelevance(ctx context.Context, id, teamID uuid.UUID, notRelevant bool, setBy uuid.UUID) (*absences.AbsenceRow, error) {
	return m.setStatsRelevance(ctx, id, teamID, notRelevant, setBy)
}

// ─── mock permission checker ────────────────────────────────────────────────

type mockPerms struct {
	get func(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error)
}

func (m *mockPerms) GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error) {
	return m.get(ctx, teamID, userID)
}

func (m *mockRepo) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *absences.ListCursor) ([]*absences.AbsenceRow, error) {
	return m.listByTeam(ctx, teamID, limit, cur)
}

func (m *mockRepo) ListByUser(ctx context.Context, teamID, userID uuid.UUID, limit int, cur *absences.ListCursor) ([]*absences.AbsenceRow, error) {
	return m.listByUser(ctx, teamID, userID, limit, cur)
}

func (m *mockRepo) Create(ctx context.Context, teamID, userID uuid.UUID, fromDate, toDate string, reason *string) (*absences.AbsenceRow, error) {
	return m.create(ctx, teamID, userID, fromDate, toDate, reason)
}

func (m *mockRepo) Update(ctx context.Context, id, teamID, userID uuid.UUID, fromDate, toDate, reason *string) (*absences.AbsenceRow, error) {
	return m.update(ctx, id, teamID, userID, fromDate, toDate, reason)
}

func (m *mockRepo) Delete(ctx context.Context, id, teamID, userID uuid.UUID) error {
	return m.delete(ctx, id, teamID, userID)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func makeAbsenceRow() *absences.AbsenceRow {
	name := "Alice"
	color := "#6366f1"
	return &absences.AbsenceRow{
		Id:                uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		UserId:            uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		TeamId:            uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		FromDate:          time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
		ToDate:            time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		CreatedAt:         time.Now(),
		MemberName:        &name,
		MemberAvatarColor: &color,
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestService_ListByTeam(t *testing.T) {
	t.Parallel()

	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	row := makeAbsenceRow()

	repo := &mockRepo{
		listByTeam: func(_ context.Context, tid uuid.UUID, _ int, _ *absences.ListCursor) ([]*absences.AbsenceRow, error) {
			assert.Equal(t, teamID, tid)
			return []*absences.AbsenceRow{row}, nil
		},
	}

	svc := absences.NewService(repo, nil, nil)
	result, next, err := svc.ListByTeam(context.Background(), teamID, 50, "")

	require.NoError(t, err)
	assert.Nil(t, next)
	require.Len(t, result, 1)
	assert.Equal(t, row.Id, result[0].Id)
	assert.Equal(t, *row.MemberName, *result[0].MemberName)
}

func TestService_ListByTeam_PopulatesMemberMembershipId(t *testing.T) {
	t.Parallel()

	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	membershipID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	row := makeAbsenceRow()
	row.MembershipId = &membershipID

	repo := &mockRepo{
		listByTeam: func(_ context.Context, _ uuid.UUID, _ int, _ *absences.ListCursor) ([]*absences.AbsenceRow, error) {
			return []*absences.AbsenceRow{row}, nil
		},
	}

	svc := absences.NewService(repo, nil, nil)
	result, _, err := svc.ListByTeam(context.Background(), teamID, 50, "")

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].MemberMembershipId, "Absence.MemberMembershipId must be populated so the frontend can build the member's photo URL")
	assert.Equal(t, membershipID, *result[0].MemberMembershipId)
}

func TestService_ListByUser(t *testing.T) {
	t.Parallel()

	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	row := makeAbsenceRow()

	repo := &mockRepo{
		listByUser: func(_ context.Context, tid, uid uuid.UUID, _ int, _ *absences.ListCursor) ([]*absences.AbsenceRow, error) {
			assert.Equal(t, teamID, tid)
			assert.Equal(t, userID, uid)
			return []*absences.AbsenceRow{row}, nil
		},
	}

	svc := absences.NewService(repo, nil, nil)
	result, next, err := svc.ListByUser(context.Background(), teamID, userID, 50, "")

	require.NoError(t, err)
	assert.Nil(t, next)
	require.Len(t, result, 1)
	assert.Equal(t, row.UserId, result[0].UserId)
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	row := makeAbsenceRow()

	repo := &mockRepo{
		create: func(_ context.Context, tid, uid uuid.UUID, from, to string, reason *string) (*absences.AbsenceRow, error) {
			assert.Equal(t, teamID, tid)
			assert.Equal(t, userID, uid)
			return row, nil
		},
	}

	svc := absences.NewService(repo, nil, nil)
	body := &gen.CreateAbsenceRequest{
		UserId: userID,
		From:   openapi_types.Date{Time: time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)},
		To:     openapi_types.Date{Time: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)},
	}
	result, err := svc.Create(context.Background(), teamID, body)

	require.NoError(t, err)
	assert.Equal(t, row.Id, result.Id)
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	called := false

	repo := &mockRepo{
		delete: func(_ context.Context, absID, tid, uid uuid.UUID) error {
			assert.Equal(t, id, absID)
			assert.Equal(t, teamID, tid)
			assert.Equal(t, userID, uid)
			called = true
			return nil
		},
	}

	svc := absences.NewService(repo, nil, nil)
	err := svc.Delete(context.Background(), id, teamID, userID)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestService_Delete_WrongTeam_PropagatesNoRows(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	wrongTeamID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	repo := &mockRepo{
		delete: func(_ context.Context, _, _, _ uuid.UUID) error {
			return pgx.ErrNoRows
		},
	}

	svc := absences.NewService(repo, nil, nil)
	err := svc.Delete(context.Background(), id, wrongTeamID, userID)

	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	row := makeAbsenceRow()
	reason := "holiday"

	repo := &mockRepo{
		update: func(_ context.Context, absID, tid, uid uuid.UUID, from, to *string, r *string) (*absences.AbsenceRow, error) {
			assert.Equal(t, id, absID)
			assert.Equal(t, teamID, tid)
			assert.Equal(t, userID, uid)
			assert.Equal(t, "holiday", *r)
			return row, nil
		},
	}

	svc := absences.NewService(repo, nil, nil)
	body := &gen.UpdateAbsenceRequest{Reason: &reason}
	result, err := svc.Update(context.Background(), id, teamID, userID, body)

	require.NoError(t, err)
	assert.Equal(t, row.Id, result.Id)
}

func TestService_Update_WrongTeam_PropagatesNoRows(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	wrongTeamID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	reason := "holiday"

	repo := &mockRepo{
		update: func(_ context.Context, _, _, _ uuid.UUID, _, _ *string, _ *string) (*absences.AbsenceRow, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := absences.NewService(repo, nil, nil)
	body := &gen.UpdateAbsenceRequest{Reason: &reason}
	_, err := svc.Update(context.Background(), id, wrongTeamID, userID, body)

	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestService_SetStatsRelevance_OwnerAlwaysAllowed(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	userID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	row := makeAbsenceRow()
	row.NotRelevantForStats = true

	permsCalled := false
	repo := &mockRepo{
		getOwner: func(_ context.Context, absID, tid uuid.UUID) (uuid.UUID, error) {
			assert.Equal(t, id, absID)
			assert.Equal(t, teamID, tid)
			return userID, nil // caller is the owner
		},
		setStatsRelevance: func(_ context.Context, absID, tid uuid.UUID, notRelevant bool, setBy uuid.UUID) (*absences.AbsenceRow, error) {
			assert.Equal(t, id, absID)
			assert.Equal(t, teamID, tid)
			assert.True(t, notRelevant)
			assert.Equal(t, userID, setBy)
			return row, nil
		},
	}
	perms := &mockPerms{get: func(context.Context, uuid.UUID, uuid.UUID) (teams.PermissionsJSON, error) {
		permsCalled = true
		return teams.PermissionsJSON{}, nil
	}}

	svc := absences.NewService(repo, nil, perms)
	result, err := svc.SetStatsRelevance(context.Background(), id, teamID, userID, true)

	require.NoError(t, err)
	assert.True(t, result.NotRelevantForStats)
	assert.False(t, permsCalled, "owner acting on their own absence must never trigger a permission lookup")
}

func TestService_SetStatsRelevance_NonOwnerWithoutEventsWrite_Forbidden(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	ownerID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	callerID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	repo := &mockRepo{
		getOwner: func(context.Context, uuid.UUID, uuid.UUID) (uuid.UUID, error) {
			return ownerID, nil
		},
		setStatsRelevance: func(context.Context, uuid.UUID, uuid.UUID, bool, uuid.UUID) (*absences.AbsenceRow, error) {
			t.Fatal("must not write when the caller lacks permission")
			return nil, nil
		},
	}
	perms := &mockPerms{get: func(context.Context, uuid.UUID, uuid.UUID) (teams.PermissionsJSON, error) {
		return teams.PermissionsJSON{Events: "read"}, nil
	}}

	svc := absences.NewService(repo, nil, perms)
	_, err := svc.SetStatsRelevance(context.Background(), id, teamID, callerID, true)

	require.Error(t, err)
	assert.ErrorIs(t, err, absences.ErrForbiddenStatsRelevance)
}

func TestService_SetStatsRelevance_NonOwnerWithEventsWrite_Allowed(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	ownerID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	callerID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	row := makeAbsenceRow()
	row.NotRelevantForStats = true

	repo := &mockRepo{
		getOwner: func(context.Context, uuid.UUID, uuid.UUID) (uuid.UUID, error) {
			return ownerID, nil
		},
		setStatsRelevance: func(_ context.Context, _, _ uuid.UUID, notRelevant bool, setBy uuid.UUID) (*absences.AbsenceRow, error) {
			assert.True(t, notRelevant)
			assert.Equal(t, callerID, setBy)
			return row, nil
		},
	}
	perms := &mockPerms{get: func(context.Context, uuid.UUID, uuid.UUID) (teams.PermissionsJSON, error) {
		return teams.PermissionsJSON{Events: "write"}, nil
	}}

	svc := absences.NewService(repo, nil, perms)
	result, err := svc.SetStatsRelevance(context.Background(), id, teamID, callerID, true)

	require.NoError(t, err)
	assert.True(t, result.NotRelevantForStats)
}

func TestService_SetStatsRelevance_AbsenceNotFound(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	callerID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	repo := &mockRepo{
		getOwner: func(context.Context, uuid.UUID, uuid.UUID) (uuid.UUID, error) {
			return uuid.Nil, pgx.ErrNoRows
		},
	}

	svc := absences.NewService(repo, nil, nil)
	_, err := svc.SetStatsRelevance(context.Background(), id, teamID, callerID, true)

	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}
