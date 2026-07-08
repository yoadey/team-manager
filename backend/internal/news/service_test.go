package news_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/news"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	listByTeam func(ctx context.Context, teamID uuid.UUID, limit int, cur *news.ListCursor) ([]*news.NewsRow, error)
	create     func(ctx context.Context, teamID, authorID uuid.UUID, title, body string, pinned bool) (*news.NewsRow, error)
	update     func(ctx context.Context, id, teamID uuid.UUID, title, body *string, pinned *bool) (*news.NewsRow, error)
	delete     func(ctx context.Context, id, teamID uuid.UUID) error
}

func (m *mockRepo) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *news.ListCursor) ([]*news.NewsRow, error) {
	return m.listByTeam(ctx, teamID, limit, cur)
}

func (m *mockRepo) Create(ctx context.Context, teamID, authorID uuid.UUID, title, body string, pinned bool) (*news.NewsRow, error) {
	return m.create(ctx, teamID, authorID, title, body, pinned)
}

func (m *mockRepo) Update(ctx context.Context, id, teamID uuid.UUID, title, body *string, pinned *bool) (*news.NewsRow, error) {
	return m.update(ctx, id, teamID, title, body, pinned)
}

func (m *mockRepo) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	return m.delete(ctx, id, teamID)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func makeNewsRow() *news.NewsRow {
	name := "Bob"
	color := "#10b981"
	return &news.NewsRow{
		Id:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		TeamId:      uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		AuthorId:    uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Title:       "Team Update",
		Body:        "We won the match!",
		Pinned:      false,
		CreatedAt:   time.Now(),
		AuthorName:  &name,
		AuthorColor: &color,
	}
}

// ─── tests ───────────────────────────────────────────────────────────────────

func TestService_ListByTeam(t *testing.T) {
	t.Parallel()

	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	row := makeNewsRow()

	repo := &mockRepo{
		listByTeam: func(_ context.Context, tid uuid.UUID, _ int, _ *news.ListCursor) ([]*news.NewsRow, error) {
			assert.Equal(t, teamID, tid)
			return []*news.NewsRow{row}, nil
		},
	}

	svc := news.NewService(repo, nil, nil, slog.Default())
	result, next, err := svc.ListByTeam(context.Background(), teamID, 50, "")

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Nil(t, next, "single page should have no next cursor")
	assert.Equal(t, row.Id, result[0].Id)
	assert.Equal(t, row.Title, result[0].Title)
}

func TestService_Create(t *testing.T) {
	t.Parallel()

	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	authorID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	row := makeNewsRow()

	repo := &mockRepo{
		create: func(_ context.Context, tid, aid uuid.UUID, title, body string, pinned bool) (*news.NewsRow, error) {
			assert.Equal(t, teamID, tid)
			assert.Equal(t, authorID, aid)
			assert.Equal(t, "Team Update", title)
			return row, nil
		},
	}

	svc := news.NewService(repo, nil, nil, slog.Default())
	body := &gen.CreateNewsRequest{
		Title: "Team Update",
		Body:  "We won the match!",
	}
	result, err := svc.Create(context.Background(), teamID, authorID, body)

	require.NoError(t, err)
	assert.Equal(t, row.Id, result.Id)
	assert.Equal(t, row.Title, result.Title)
}

// mockJobEnqueuer satisfies jobEnqueuer for tests exercising the
// best-effort notification path.
type mockJobEnqueuer struct {
	err error
}

func (m *mockJobEnqueuer) EnqueueNotification(context.Context, jobs.NotificationArgs) error {
	return m.err
}

// TestService_Create_NotificationEnqueueFailure_StillSucceeds regression-tests
// that a failed best-effort notification enqueue doesn't fail the request
// (the write already succeeded) -- this was already true before, but a
// failure here used to be discarded with no trace at all; the logger
// parameter added alongside this test makes it observable instead. There's
// no assertion on log output itself (would require capturing the slog
// handler), just that the documented best-effort semantics survived the change.
func TestService_Create_NotificationEnqueueFailure_StillSucceeds(t *testing.T) {
	t.Parallel()

	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	authorID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	row := makeNewsRow()

	repo := &mockRepo{
		create: func(context.Context, uuid.UUID, uuid.UUID, string, string, bool) (*news.NewsRow, error) {
			return row, nil
		},
	}

	svc := news.NewService(repo, &mockJobEnqueuer{err: errors.New("river unavailable")}, nil, slog.Default())
	body := &gen.CreateNewsRequest{Title: "Team Update", Body: "We won the match!"}
	result, err := svc.Create(context.Background(), teamID, authorID, body)

	require.NoError(t, err)
	assert.Equal(t, row.Id, result.Id)
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	row := makeNewsRow()
	newTitle := "Updated Title"

	repo := &mockRepo{
		update: func(_ context.Context, nid, tid uuid.UUID, title, body *string, pinned *bool) (*news.NewsRow, error) {
			assert.Equal(t, id, nid)
			assert.Equal(t, teamID, tid)
			assert.Equal(t, "Updated Title", *title)
			return row, nil
		},
	}

	svc := news.NewService(repo, nil, nil, slog.Default())
	result, err := svc.Update(context.Background(), id, teamID, &gen.UpdateNewsRequest{Title: &newTitle})

	require.NoError(t, err)
	assert.Equal(t, row.Id, result.Id)
}

func TestService_Update_WrongTeam_PropagatesNoRows(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	wrongTeamID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	newTitle := "Updated Title"

	repo := &mockRepo{
		update: func(_ context.Context, _, _ uuid.UUID, _, _ *string, _ *bool) (*news.NewsRow, error) {
			return nil, pgx.ErrNoRows
		},
	}

	svc := news.NewService(repo, nil, nil, slog.Default())
	_, err := svc.Update(context.Background(), id, wrongTeamID, &gen.UpdateNewsRequest{Title: &newTitle})

	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	teamID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	called := false

	repo := &mockRepo{
		delete: func(_ context.Context, nid, tid uuid.UUID) error {
			assert.Equal(t, id, nid)
			assert.Equal(t, teamID, tid)
			called = true
			return nil
		},
	}

	svc := news.NewService(repo, nil, nil, slog.Default())
	err := svc.Delete(context.Background(), id, teamID)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestService_Delete_WrongTeam_PropagatesNoRows(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	wrongTeamID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")

	repo := &mockRepo{
		delete: func(_ context.Context, _, _ uuid.UUID) error {
			return pgx.ErrNoRows
		},
	}

	svc := news.NewService(repo, nil, nil, slog.Default())
	err := svc.Delete(context.Background(), id, wrongTeamID)

	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}
