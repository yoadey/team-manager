package news_test

import (
	"context"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/news"
)

// ─── mock repository ────────────────────────────────────────────────────────

type mockRepo struct {
	listByTeam          func(ctx context.Context, teamID uuid.UUID) ([]*news.NewsRow, error)
	create              func(ctx context.Context, teamID, authorID uuid.UUID, title, body string, pinned bool) (*news.NewsRow, error)
	update              func(ctx context.Context, id uuid.UUID, title, body *string, pinned *bool) (*news.NewsRow, error)
	delete              func(ctx context.Context, id uuid.UUID) error
	insertNotification  func(ctx context.Context, teamID, actorID uuid.UUID, title string) error
}

func (m *mockRepo) ListByTeam(ctx context.Context, teamID uuid.UUID) ([]*news.NewsRow, error) {
	return m.listByTeam(ctx, teamID)
}
func (m *mockRepo) Create(ctx context.Context, teamID, authorID uuid.UUID, title, body string, pinned bool) (*news.NewsRow, error) {
	return m.create(ctx, teamID, authorID, title, body, pinned)
}
func (m *mockRepo) Update(ctx context.Context, id uuid.UUID, title, body *string, pinned *bool) (*news.NewsRow, error) {
	return m.update(ctx, id, title, body, pinned)
}
func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.delete(ctx, id)
}
func (m *mockRepo) InsertNotification(ctx context.Context, teamID, actorID uuid.UUID, title string) error {
	if m.insertNotification != nil {
		return m.insertNotification(ctx, teamID, actorID, title)
	}
	return nil
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
		listByTeam: func(_ context.Context, tid uuid.UUID) ([]*news.NewsRow, error) {
			assert.Equal(t, teamID, tid)
			return []*news.NewsRow{row}, nil
		},
	}

	svc := news.NewService(repo)
	result, err := svc.ListByTeam(context.Background(), teamID)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, openapi_types.UUID(row.Id), result[0].Id)
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

	svc := news.NewService(repo)
	body := &gen.CreateNewsRequest{
		Title: "Team Update",
		Body:  "We won the match!",
	}
	result, err := svc.Create(context.Background(), teamID, authorID, body)

	require.NoError(t, err)
	assert.Equal(t, openapi_types.UUID(row.Id), result.Id)
	assert.Equal(t, row.Title, result.Title)
}

func TestService_Update(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	row := makeNewsRow()
	newTitle := "Updated Title"

	repo := &mockRepo{
		update: func(_ context.Context, nid uuid.UUID, title, body *string, pinned *bool) (*news.NewsRow, error) {
			assert.Equal(t, id, nid)
			assert.Equal(t, "Updated Title", *title)
			return row, nil
		},
	}

	svc := news.NewService(repo)
	result, err := svc.Update(context.Background(), id, &gen.UpdateNewsRequest{Title: &newTitle})

	require.NoError(t, err)
	assert.Equal(t, openapi_types.UUID(row.Id), result.Id)
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	called := false

	repo := &mockRepo{
		delete: func(_ context.Context, nid uuid.UUID) error {
			assert.Equal(t, id, nid)
			called = true
			return nil
		},
	}

	svc := news.NewService(repo)
	err := svc.Delete(context.Background(), id)

	require.NoError(t, err)
	assert.True(t, called)
}
