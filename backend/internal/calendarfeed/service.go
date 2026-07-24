package calendarfeed

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/notifications"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// ErrFeedUnavailable is returned by ServeFeed for every failure mode --
// unknown/revoked token, the token holder no longer a team member, or the
// token holder's events permission no longer at least "read". The feed
// handler maps it to a plain 404, matching the design decision not to
// distinguish "token never existed" from "token holder lost access" (an
// unauthenticated caller must not be able to learn anything about a token's
// history from the response).
var ErrFeedUnavailable = errors.New("calendarfeed: feed unavailable")

// tokenRepo is the interface Service relies on for token management.
type tokenRepo interface {
	IssueToken(ctx context.Context, userID, teamID uuid.UUID) (string, error)
	Revoke(ctx context.Context, userID, teamID uuid.UUID) error
	FindActiveByToken(ctx context.Context, token string) (*TokenRow, error)
}

// membershipChecker mirrors middleware.MembershipChecker -- satisfied by
// members.Repository.
type membershipChecker interface {
	IsMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error)
}

// permsChecker mirrors middleware.PermissionChecker -- satisfied by
// members.Repository.
type permsChecker interface {
	GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error)
}

// teamRepo is the interface Service relies on to resolve a team's display
// name for the feed's X-WR-CALNAME.
type teamRepo interface {
	GetTeam(ctx context.Context, teamID string) (*teams.TeamRow, error)
}

// eventLister is the interface Service relies on to fetch a team's events.
type eventLister interface {
	ListEvents(ctx context.Context, teamID string, scope gen.ListEventsParamsScope, limit int, cur *events.ListCursor) ([]events.EventRow, error)
}

// Service implements calendar-feed business logic: token issuance/
// revocation and feed rendering.
type Service struct {
	tokens        tokenRepo
	membership    membershipChecker
	perms         permsChecker
	teamRepo      teamRepo
	eventRepo     eventLister
	publicBaseURL string
}

// NewService creates a new Service. publicBaseURL is the scheme+host issued
// feed URLs are built against (config.PublicBaseURL).
func NewService(tokens tokenRepo, membership membershipChecker, perms permsChecker, teamRepo teamRepo, eventRepo eventLister, publicBaseURL string) *Service {
	return &Service{
		tokens:        tokens,
		membership:    membership,
		perms:         perms,
		teamRepo:      teamRepo,
		eventRepo:     eventRepo,
		publicBaseURL: publicBaseURL,
	}
}

// maxFeedEvents caps how many of a team's events the feed renders --
// defensive backstop, mirroring notifications.Repository's
// maxNotificationRows, against pathologically long-lived teams with an
// unbounded event history.
const maxFeedEvents = 2000

// IssueToken mints (rotating any existing one) a calendar feed token for
// (userID, teamID) and returns the ready-to-use subscription URL.
func (s *Service) IssueToken(ctx context.Context, userID, teamID uuid.UUID) (string, error) {
	token, err := s.tokens.IssueToken(ctx, userID, teamID)
	if err != nil {
		return "", fmt.Errorf("calendarfeed.Service.IssueToken: %w", err)
	}
	return s.publicBaseURL + "/api/v1/calendar-feed/" + token + ".ics", nil
}

// RevokeToken invalidates (userID, teamID)'s active token, if any.
func (s *Service) RevokeToken(ctx context.Context, userID, teamID uuid.UUID) error {
	if err := s.tokens.Revoke(ctx, userID, teamID); err != nil {
		return fmt.Errorf("calendarfeed.Service.RevokeToken: %w", err)
	}
	return nil
}

// ServeFeed resolves token to its (user, team), re-checks the token
// holder's *current* team membership and events read permission, and
// renders that team's non-cancelled events as an iCalendar document.
// Returns ErrFeedUnavailable for every failure mode -- see its doc comment.
func (s *Service) ServeFeed(ctx context.Context, token string) ([]byte, error) {
	row, err := s.tokens.FindActiveByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFeedUnavailable
		}
		return nil, fmt.Errorf("calendarfeed.Service.ServeFeed: find token: %w", err)
	}

	isMember, err := s.membership.IsMember(ctx, row.TeamId, row.UserId)
	if err != nil {
		return nil, fmt.Errorf("calendarfeed.Service.ServeFeed: check membership: %w", err)
	}
	if !isMember {
		return nil, ErrFeedUnavailable
	}

	perms, err := s.perms.GetPermissions(ctx, row.TeamId, row.UserId)
	if err != nil {
		return nil, fmt.Errorf("calendarfeed.Service.ServeFeed: get permissions: %w", err)
	}
	if !notifications.HasReadAccess(perms, "events") {
		return nil, ErrFeedUnavailable
	}

	team, err := s.teamRepo.GetTeam(ctx, row.TeamId.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFeedUnavailable
		}
		return nil, fmt.Errorf("calendarfeed.Service.ServeFeed: get team: %w", err)
	}

	evts, err := s.eventRepo.ListEvents(ctx, row.TeamId.String(), gen.All, maxFeedEvents, nil)
	if err != nil {
		return nil, fmt.Errorf("calendarfeed.Service.ServeFeed: list events: %w", err)
	}

	return Render(team.Name, evts), nil
}
