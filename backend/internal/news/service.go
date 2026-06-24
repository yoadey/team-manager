package news

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

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
		return nil, err
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
		return gen.NewsItem{}, err
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
		return gen.NewsItem{}, err
	}
	return toGenNewsItem(row), nil
}

// Delete removes a news item by ID.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// toGenNewsItem maps a NewsRow to the generated gen.NewsItem type.
func toGenNewsItem(row *NewsRow) gen.NewsItem {
	hasPhoto := len(row.PhotoData) > 0
	ni := gen.NewsItem{
		Id:             openapi_types.UUID(row.Id),
		TeamId:         openapi_types.UUID(row.TeamId),
		AuthorId:       openapi_types.UUID(row.AuthorId),
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
