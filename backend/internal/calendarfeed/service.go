package calendarfeed

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/events"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/members"
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

// ErrNoActiveToken is returned by GetSettings/UpdateSettings when the
// caller has no active feed token yet -- the content selection is stored on
// the token row, so there's nothing to read or update until one is issued.
var ErrNoActiveToken = errors.New("calendarfeed: no active token")

// ErrInvalidEventType is returned by UpdateSettings when the requested
// content selection includes a value outside the current EventType enum.
var ErrInvalidEventType = errors.New("calendarfeed: invalid event type")

// tokenRepo is the interface Service relies on for token management.
type tokenRepo interface {
	IssueToken(ctx context.Context, userID, teamID uuid.UUID) (string, error)
	Revoke(ctx context.Context, userID, teamID uuid.UUID) error
	FindActiveByToken(ctx context.Context, token string) (*TokenRow, error)
	GetSettings(ctx context.Context, userID, teamID uuid.UUID) (types []string, includeBirthdays bool, err error)
	UpdateSettings(ctx context.Context, userID, teamID uuid.UUID, types []string, includeBirthdays bool) error
}

// memberLister is the interface Service relies on to fetch a team's members
// (for birthday rendering). Satisfied by members.Repository.
type memberLister interface {
	ListMembers(ctx context.Context, teamID string, limit int, cur *members.ListCursor) ([]members.MemberRow, error)
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
// revocation, content-selection management, and feed rendering.
type Service struct {
	tokens        tokenRepo
	membership    membershipChecker
	perms         permsChecker
	teamRepo      teamRepo
	eventRepo     eventLister
	memberRepo    memberLister
	publicBaseURL string
}

// NewService creates a new Service. publicBaseURL is the scheme+host issued
// feed URLs are built against (config.PublicBaseURL).
func NewService(tokens tokenRepo, membership membershipChecker, perms permsChecker, teamRepo teamRepo, eventRepo eventLister, memberRepo memberLister, publicBaseURL string) *Service {
	return &Service{
		tokens:        tokens,
		membership:    membership,
		perms:         perms,
		teamRepo:      teamRepo,
		eventRepo:     eventRepo,
		memberRepo:    memberRepo,
		publicBaseURL: publicBaseURL,
	}
}

// maxFeedEvents caps how many of a team's events the feed renders --
// defensive backstop, mirroring notifications.Repository's
// maxNotificationRows, against pathologically long-lived teams with an
// unbounded event history.
const maxFeedEvents = 2000

// maxFeedMembers mirrors maxFeedEvents for the birthday-source member list.
const maxFeedMembers = 2000

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

// GetSettings returns (userID, teamID)'s current feed content selection.
// Falls back to defaultFeedTypes + birthdays-on if no token has been issued
// yet, rather than erroring -- there is nothing wrong to report, just
// nothing customized yet.
func (s *Service) GetSettings(ctx context.Context, userID, teamID uuid.UUID) (types []string, includeBirthdays bool, err error) {
	types, includeBirthdays, err = s.tokens.GetSettings(ctx, userID, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultFeedTypes, true, nil
		}
		return nil, false, fmt.Errorf("calendarfeed.Service.GetSettings: %w", err)
	}
	return types, includeBirthdays, nil
}

// UpdateSettings validates and stores (userID, teamID)'s feed content
// selection, applying to the existing subscription URL. Returns
// ErrInvalidEventType if types contains a value outside the current
// EventType enum, or ErrNoActiveToken if the caller has no active token to
// attach the selection to.
func (s *Service) UpdateSettings(ctx context.Context, userID, teamID uuid.UUID, types []string, includeBirthdays bool) error {
	for _, t := range types {
		if !gen.EventType(t).Valid() {
			return fmt.Errorf("%w: %q", ErrInvalidEventType, t)
		}
	}
	if err := s.tokens.UpdateSettings(ctx, userID, teamID, types, includeBirthdays); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoActiveToken
		}
		return fmt.Errorf("calendarfeed.Service.UpdateSettings: %w", err)
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
	filtered := filterEventsByType(evts, row.Types)

	birthdays, err := s.loadBirthdays(ctx, row, perms)
	if err != nil {
		return nil, fmt.Errorf("calendarfeed.Service.ServeFeed: %w", err)
	}

	return Render(team.Name, filtered, birthdays), nil
}

// filterEventsByType keeps only the events whose Type is in allowedTypes.
func filterEventsByType(evts []events.EventRow, allowedTypes []string) []events.EventRow {
	allowed := make(map[string]bool, len(allowedTypes))
	for _, t := range allowedTypes {
		allowed[t] = true
	}
	filtered := make([]events.EventRow, 0, len(evts))
	for _, e := range evts {
		if allowed[e.Type] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// loadBirthdays returns row's team's member birthdays, or nil if the feed
// isn't configured to include them or the token holder lacks the "members"
// module read access birthdays live behind -- same as the in-app member
// list would show them nothing, not an error.
func (s *Service) loadBirthdays(ctx context.Context, row *TokenRow, perms teams.PermissionsJSON) ([]Birthday, error) {
	if !row.IncludeBirthdays || !notifications.HasReadAccess(perms, "members") {
		return nil, nil
	}
	memberRows, err := s.memberRepo.ListMembers(ctx, row.TeamId.String(), maxFeedMembers, nil)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	var birthdays []Birthday
	for _, m := range memberRows {
		if m.Birthday != nil {
			birthdays = append(birthdays, Birthday{MemberID: m.UserID, Name: m.Name, Date: *m.Birthday})
		}
	}
	return birthdays, nil
}
