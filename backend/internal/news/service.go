package news

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/pagination"
)

// newsRepo is the interface the Service relies on.
type newsRepo interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cur *ListCursor) ([]*NewsRow, error)
	CountByTeam(ctx context.Context, teamID uuid.UUID) (int, error)
	Create(ctx context.Context, teamID, authorID uuid.UUID, title, body string, pinned bool) (*NewsRow, error)
	Update(ctx context.Context, id, teamID uuid.UUID, title, body *string, pinned *bool) (*NewsRow, error)
	Delete(ctx context.Context, id, teamID uuid.UUID) error
}

// ErrTooManyNewsItems is returned once a team hits maxNewsPerTeam. Unlike
// finances' per-team caps, ListByTeam here is already properly
// keyset-paginated (O(limit), not O(table size)), so this isn't closing an
// availability bug -- it's a cheap defense-in-depth cap against unbounded
// storage growth from a scripted or careless news:write caller.
var ErrTooManyNewsItems = fmt.Errorf("team has reached the maximum of %d news items", maxNewsPerTeam)

const maxNewsPerTeam = 50_000

// jobEnqueuer is satisfied by *jobs.Client.
type jobEnqueuer interface {
	EnqueueNotification(ctx context.Context, args jobs.NotificationArgs) error
}

// Service implements news business logic.
type Service struct {
	repo   newsRepo
	jobs   jobEnqueuer
	pager  *pagination.Paginator
	logger *slog.Logger
}

// NewService creates a new Service. pager may be nil, in which case a default
// (unsigned) Paginator is used.
func NewService(repo newsRepo, enq jobEnqueuer, pager *pagination.Paginator, logger *slog.Logger) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, jobs: enq, pager: pager, logger: logger}
}

// ListByTeam returns a keyset page of news items plus the cursor for the next
// page (nil when the last page is reached). cursor is the opaque token from a
// prior page ("" = first page).
func (s *Service) ListByTeam(ctx context.Context, teamID uuid.UUID, limit int, cursor string) ([]gen.NewsItem, *string, error) {
	var cur *ListCursor
	var decoded ListCursor
	if ok, err := s.pager.Decode(cursor, &decoded); err != nil {
		return nil, nil, fmt.Errorf("news.Service.ListByTeam: %w", err)
	} else if ok {
		cur = &decoded
	}

	// Fetch one extra row to detect whether a further page exists.
	rows, err := s.repo.ListByTeam(ctx, teamID, limit+1, cur)
	if err != nil {
		return nil, nil, fmt.Errorf("news.Service.ListByTeam: %w", err)
	}

	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		token, err := s.pager.Encode(ListCursor{Pinned: last.Pinned, CreatedAt: last.CreatedAt, ID: last.Id})
		if err != nil {
			return nil, nil, fmt.Errorf("news.Service.ListByTeam: %w", err)
		}
		next = &token
	}

	result := make([]gen.NewsItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, toGenNewsItem(row))
	}
	return result, next, nil
}

// Create adds a new news item and enqueues a notification job.
func (s *Service) Create(ctx context.Context, teamID, authorID uuid.UUID, body *gen.CreateNewsRequest) (gen.NewsItem, error) {
	count, err := s.repo.CountByTeam(ctx, teamID)
	if err != nil {
		return gen.NewsItem{}, fmt.Errorf("news.Service.Create: %w", err)
	}
	if count >= maxNewsPerTeam {
		return gen.NewsItem{}, ErrTooManyNewsItems
	}

	pinned := false
	if body.Pinned != nil {
		pinned = *body.Pinned
	}
	row, err := s.repo.Create(ctx, teamID, authorID, body.Title, body.Body, pinned)
	if err != nil {
		return gen.NewsItem{}, fmt.Errorf("news.Service.Create: %w", err)
	}
	// Fire notification via River (best-effort; ignore error so it doesn't fail the request).
	if s.jobs != nil {
		title := body.Title
		if err := s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
			TeamID:  teamID,
			Type:    "news",
			ActorID: authorID,
			Title:   &title,
		}); err != nil {
			s.logger.Warn("news: failed to enqueue notification", slog.String("newsId", row.Id.String()), slog.String("error", err.Error()))
		}
	}
	return toGenNewsItem(row), nil
}

// Update modifies an existing news item that belongs to teamID.
func (s *Service) Update(ctx context.Context, id, teamID uuid.UUID, body *gen.UpdateNewsRequest) (gen.NewsItem, error) {
	row, err := s.repo.Update(ctx, id, teamID, body.Title, body.Body, body.Pinned)
	if err != nil {
		return gen.NewsItem{}, fmt.Errorf("news.Service.Update: %w", err)
	}
	return toGenNewsItem(row), nil
}

// Delete removes a news item by ID that belongs to teamID.
func (s *Service) Delete(ctx context.Context, id, teamID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id, teamID); err != nil {
		return fmt.Errorf("news.Service.Delete: %w", err)
	}
	return nil
}

// toGenNewsItem maps a NewsRow to the generated gen.NewsItem type.
func toGenNewsItem(row *NewsRow) gen.NewsItem {
	hasPhoto := row.HasPhoto
	ni := gen.NewsItem{
		Id:             row.Id,
		TeamId:         row.TeamId,
		AuthorId:       row.AuthorId,
		Title:          row.Title,
		Body:           row.Body,
		Pinned:         row.Pinned,
		CreatedAt:      row.CreatedAt,
		HasAuthorPhoto: &hasPhoto,
	}
	if row.AuthorName != nil {
		ni.AuthorName = row.AuthorName
	}
	if row.AuthorColor != nil {
		ni.AuthorColor = row.AuthorColor
	}
	return ni
}
