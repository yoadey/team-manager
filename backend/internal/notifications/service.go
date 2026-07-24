package notifications

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// notifRepo is the interface the Service relies on.
type notifRepo interface {
	ListByTeamAndUser(ctx context.Context, teamID, userID uuid.UUID) ([]*NotificationRow, error)
	MarkSeen(ctx context.Context, teamID, userID uuid.UUID) error
}

// permChecker returns the effective per-module permissions for a (team,
// user) pair -- satisfied by members.Repository.
type permChecker interface {
	GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error)
}

// Service implements notifications business logic.
type Service struct {
	repo  notifRepo
	perms permChecker
}

// NewService creates a new Service.
func NewService(repo notifRepo, perms permChecker) *Service {
	return &Service{repo: repo, perms: perms}
}

// NotificationModule returns the RBAC module a notification type belongs to,
// or "" if it's self-standing (not gated by any module permission). The
// /notifications route itself carries no module-level RBAC check (it
// aggregates across events, news, and polls), so without this, a member
// with e.g. events:none would still see event/attendance notices -- exactly
// the "none must hide the module" property enforced everywhere else.
//
// Typed on gen.NotificationType (not a plain string) specifically so the
// repo-wide "exhaustive" linter (see .golangci.yml) can enforce that every
// case here is revisited when a new value is added to that enum -- a plain
// string switch is invisible to it, and would otherwise let a future
// module-gated notification type silently fall through the default case and
// leak to every team member regardless of their permission on that module.
//
// Exported (not just used internally by List) so internal/jobs can apply the
// identical gate before enqueuing a Web Push delivery for the same
// notification -- push must not open a side channel around the same
// module-permission check the in-app feed already enforces.
func NotificationModule(notifType gen.NotificationType) string {
	switch notifType {
	case gen.NotificationTypeAttendance,
		gen.NotificationTypeEventCreated,
		gen.NotificationTypeEventUpdated,
		gen.NotificationTypeEventCancelled,
		gen.NotificationTypeEventReactivated,
		gen.NotificationTypeEventDeleted:
		return "events"
	case gen.NotificationTypeNews:
		return "news"
	case gen.NotificationTypePoll:
		return "polls"
	case gen.NotificationTypeAbsence:
		return ""
	default:
		// Safety net for a value outside the known enum (a malformed/future
		// DB row) -- exhaustive's default-signifies-exhaustive check is off
		// repo-wide, so this default does NOT suppress a missing-case warning
		// when a new gen.NotificationType constant is added; it only covers
		// values that were never valid to begin with.
		return ""
	}
}

// HasReadAccess reports whether p grants at least "read" on module. An empty
// module (self-standing notification types, e.g. "absence") is always
// visible. Every other module string must match one of PermissionsJSON's six
// fields explicitly and fail CLOSED on anything else -- unlike
// NotificationModule's callers-are-trusted default, this function is the
// actual gate deciding whether a notification is shown, so an unrecognized
// module (e.g. NotificationModule is later extended to return "members"/
// "finances"/"settings" for a new notification type, without a matching case
// added here too) must not silently grant access, mirroring
// middleware/authz.go's hasWritePermission/hasAnyPermission, which fail
// closed on the same six module names for the identical reason.
//
// Exported for the same reason NotificationModule is -- internal/jobs
// applies it before enqueuing a Web Push delivery.
func HasReadAccess(p teams.PermissionsJSON, module string) bool {
	if module == "" {
		return true
	}
	var level string
	switch module {
	case "events":
		level = p.Events
	case "members":
		level = p.Members
	case "finances":
		level = p.Finances
	case "news":
		level = p.News
	case "polls":
		level = p.Polls
	case "settings":
		level = p.Settings
	default:
		return false
	}
	return level == "read" || level == "write"
}

// List returns all notifications for the user in the given team that
// originate from a module the user has at least "read" on.
func (s *Service) List(ctx context.Context, teamID, userID uuid.UUID) (gen.NotificationsResult, error) {
	rows, err := s.repo.ListByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		return gen.NotificationsResult{}, fmt.Errorf("notifications.Service.List: %w", err)
	}
	perms, err := s.perms.GetPermissions(ctx, teamID, userID)
	if err != nil {
		return gen.NotificationsResult{}, fmt.Errorf("notifications.Service.List: get permissions: %w", err)
	}

	items := make([]gen.AppNotification, 0, len(rows))
	unreadCount := 0
	for _, row := range rows {
		if !HasReadAccess(perms, NotificationModule(gen.NotificationType(row.Type))) {
			continue
		}
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
//
// Known, accepted limitation: seen_at is a single team-wide timestamp, not
// per-module. If a member with e.g. events:none marks notifications seen
// (advancing seen_at to now, covering only the news/poll items List actually
// showed them), then later gains events:read, event notifications created
// before that seen_at render as already-read even though List's
// hasReadAccess filter hid them at the time. No data is exposed either way
// -- List always re-filters by current permissions -- so this is a minor
// unread-count/UX inconsistency, not a security gap, and not worth a
// per-module seen_at for how rarely a member's module permissions change.
func (s *Service) MarkSeen(ctx context.Context, teamID, userID uuid.UUID) error {
	if err := s.repo.MarkSeen(ctx, teamID, userID); err != nil {
		return fmt.Errorf("notifications.Service.MarkSeen: %w", err)
	}
	return nil
}

// toGenNotification maps a NotificationRow to the generated gen.AppNotification type.
func toGenNotification(row *NotificationRow) gen.AppNotification {
	hasPhoto := row.HasPhoto
	n := gen.AppNotification{
		Id:            row.Id,
		TeamId:        row.TeamId,
		Type:          gen.NotificationType(row.Type),
		CreatedAt:     row.CreatedAt,
		HasActorPhoto: &hasPhoto,
		Unread:        &row.Unread,
	}
	if row.ActorId != nil {
		uid := *row.ActorId
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
		uid := *row.EventId
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
