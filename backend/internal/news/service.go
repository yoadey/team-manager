package news

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/jobs"
)

// newsRepo is the interface the Service relies on.
type newsRepo interface {
	ListByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*NewsRow, error)
	Create(ctx context.Context, teamID, authorID uuid.UUID, title, body string, pinned bool) (*NewsRow, error)
	Update(ctx context.Context, id uuid.UUID, title, body *string, pinned *bool) (*NewsRow, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// jobEnqueuer is satisfied by *jobs.Client.
type jobEnqueuer interface {
	EnqueueNotification(ctx context.Context, args jobs.NotificationArgs) error
}

// Service implements news business logic.
type Service struct {
	repo newsRepo
	jobs jobEnqueuer
}

// NewService creates a new Service.
func NewService(repo newsRepo, enq jobEnqueuer) *Service {
	return &Service{repo: repo, jobs: enq}
}

// ListByTeam returns paginated news items for the given team.
func (s *Service) ListByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]gen.NewsItem, error) {
	rows, err := s.repo.ListByTeam(ctx, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("news.Service.ListByTeam: %w", err)
	}
	result := make([]gen.NewsItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, toGenNewsItem(row))
	}
	return result, nil
}

// Create adds a new news item and enqueues a notification job.
func (s *Service) Create(ctx context.Context, teamID, authorID uuid.UUID, body *gen.CreateNewsRequest) (gen.NewsItem, error) {
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
		_ = s.jobs.EnqueueNotification(ctx, jobs.NotificationArgs{
			TeamID:  teamID,
			Type:    "news",
			ActorID: authorID,
			Title:   &title,
		})
	}
	return toGenNewsItem(row), nil
}

// Update modifies an existing news item.
func (s *Service) Update(ctx context.Context, id uuid.UUID, body *gen.UpdateNewsRequest) (gen.NewsItem, error) {
	row, err := s.repo.Update(ctx, id, body.Title, body.Body, body.Pinned)
	if err != nil {
		return gen.NewsItem{}, fmt.Errorf("news.Service.Update: %w", err)
	}
	return toGenNewsItem(row), nil
}

// Delete removes a news item by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("news.Service.Delete: %w", err)
	}
	return nil
}

// toGenNewsItem maps a NewsRow to the generated gen.NewsItem type.
func toGenNewsItem(row *NewsRow) gen.NewsItem {
	hasPhoto := len(row.PhotoData) > 0
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
