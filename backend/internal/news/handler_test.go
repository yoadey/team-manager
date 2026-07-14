package news_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/news"
)

// ─── mock service ───────────────────────────────────────────────────────────

type mockNewsService struct {
	create func(ctx context.Context, teamID, authorID uuid.UUID, body *gen.CreateNewsRequest) (gen.NewsItem, error)
	update func(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateNewsRequest) (gen.NewsItem, error)
}

func (m *mockNewsService) ListByTeam(_ context.Context, _ uuid.UUID, _ int, _ string) ([]gen.NewsItem, *string, error) {
	return nil, nil, nil
}

func (m *mockNewsService) Create(ctx context.Context, teamID, authorID uuid.UUID, body *gen.CreateNewsRequest) (gen.NewsItem, error) {
	return m.create(ctx, teamID, authorID, body)
}

func (m *mockNewsService) Update(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateNewsRequest) (gen.NewsItem, error) {
	return m.update(ctx, id, teamID, body)
}

func (m *mockNewsService) Delete(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func ctxWithUser() context.Context {
	return auth.ContextWithUser(context.Background(), &auth.UserRow{Id: uuid.New(), Name: "Admin", Email: "a@x.c"})
}

// TestNewsHandler_CreateNews_TitleTooLong_Returns400 regression-tests a bug
// where CreateNews/UpdateNews validated title with validate.Text (10,000-char
// cap), 40x the openapi.yaml contract's maxLength: 255 for title — matching
// the pattern events/handler.go already uses for its own Title field.
func TestNewsHandler_CreateNews_TitleTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := news.NewHandler(&mockNewsService{}, slog.Default())
	longTitle := strings.Repeat("t", 256)
	body := &gen.CreateNewsRequest{Title: longTitle, Body: "text"}
	_, err := h.CreateNews(ctxWithUser(), gen.CreateNewsRequestObject{TeamId: uuid.New(), Body: body})

	require.Error(t, err)
}

func TestNewsHandler_CreateNews_TitleAtLimit_Succeeds(t *testing.T) {
	t.Parallel()

	svc := &mockNewsService{
		create: func(_ context.Context, _, _ uuid.UUID, _ *gen.CreateNewsRequest) (gen.NewsItem, error) {
			return gen.NewsItem{}, nil
		},
	}
	h := news.NewHandler(svc, slog.Default())
	title := strings.Repeat("t", 255)
	body := &gen.CreateNewsRequest{Title: title, Body: "text"}
	_, err := h.CreateNews(ctxWithUser(), gen.CreateNewsRequestObject{TeamId: uuid.New(), Body: body})

	require.NoError(t, err)
}

func TestNewsHandler_UpdateNews_TitleTooLong_Returns400(t *testing.T) {
	t.Parallel()

	h := news.NewHandler(&mockNewsService{}, slog.Default())
	longTitle := strings.Repeat("t", 256)
	body := &gen.UpdateNewsRequest{Title: &longTitle}
	_, err := h.UpdateNews(ctxWithUser(), gen.UpdateNewsRequestObject{TeamId: uuid.New(), NewsId: uuid.New(), Body: body})

	require.Error(t, err)
}
