package notifications

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
)

// notifRepo is the interface the Service relies on.
type notifRepo interface {
	ListByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) ([]*NotificationRow, error)
	MarkSeen(ctx context.Context, teamID, userID uuid.UUID) error
}

// Service implements notifications business logic.
type Service struct {
	repo notifRepo
}

// NewService creates a new Service.
func NewService(repo notifRepo) *Service {
	return &Service{repo: repo}
}

// List returns all notifications for the user in the given team.
func (s *Service) List(ctx context.Context, teamID, userID uuid.UUID) (gen.NotificationsResult, error) {
	rows, err := s.repo.ListByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		return gen.NotificationsResult{}, err
	}

	items := make([]gen.AppNotification, 0, len(rows))
	unreadCount := 0
	for _, row := range rows {
		n := toGenNotification(row)
		items = append(items, n)
		if row.Unread {
			unreadCount++
		}
	}
	return gen.NotificationsResult{
		Items:       items,
		UnreadCount: unreadCount,
	}, nil
}

// MarkSeen records that the user has seen all notifications.
func (s *Service) MarkSeen(ctx context.Context, teamID, userID uuid.UUID) error {
	return s.repo.MarkSeen(ctx, teamID, userID)
}

// toGenNotification maps a NotificationRow to the generated gen.AppNotification type.
func toGenNotification(row *NotificationRow) gen.AppNotification {
	hasPhoto := len(row.PhotoData) > 0
	n := gen.AppNotification{
		Id:            openapi_types.UUID(row.Id),
		TeamId:        openapi_types.UUID(row.TeamId),
		Type:          gen.NotificationType(row.Type),
		CreatedAt:     row.CreatedAt,
		HasActorPhoto: &hasPhoto,
		Unread:        &row.Unread,
	}
	if row.ActorId != nil {
		uid := openapi_types.UUID(*row.ActorId)
		n.ActorId = &uid
	}
	if row.ActorName != nil {
		n.ActorName = row.ActorName
	}
	if row.ActorColor != nil {
		n.ActorColor = row.ActorColor
	}
	if row.Status != nil {
		s := gen.AttendanceStatus(*row.Status)
		n.Status = &s
	}
	if row.Title != nil {
		n.Title = row.Title
	}
	if row.EventId != nil {
		uid := openapi_types.UUID(*row.EventId)
		n.EventId = &uid
	}
	if row.EventTitle != nil {
		n.EventTitle = row.EventTitle
	}
	if row.EventDate != nil {
		d := openapi_types.Date{Time: *row.EventDate}
		n.EventDate = &d
	}
	if row.Note != nil {
		n.Note = row.Note
	}
	return n
}
