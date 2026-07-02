package teams

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
)

const inviteTTL = 7 * 24 * time.Hour

// teamRepo is the interface the Service relies on.
type teamRepo interface {
	ListTeamsForUser(ctx context.Context, userID string) ([]TeamRow, error)
	GetTeam(ctx context.Context, teamID string) (*TeamRow, error)
	CreateTeam(ctx context.Context, name, creatorUserID string) (*TeamRow, error)
	UpdateTeam(ctx context.Context, teamID string, patch TeamPatch) (*TeamRow, error)
	GetMemberCount(ctx context.Context, teamID string) (int, error)
	GetMembership(ctx context.Context, teamID, userID string) (*MembershipRow, error)
	GetRolesForMembership(ctx context.Context, membershipID, teamID string) ([]RoleRow, error)
	CreateInvite(ctx context.Context, teamID string, ttl time.Duration) (*InviteRow, error)
	UpdateTeamPhoto(ctx context.Context, teamID string, data []byte, mime string) error
	UpdateTeamLogo(ctx context.Context, teamID string, data []byte, mime string) error
}

// Service implements team business logic.
type Service struct {
	repo teamRepo
	// publicBaseURL is the user-facing frontend origin used to build shareable
	// invite links (no trailing slash).
	publicBaseURL string
}

// NewService creates a new Service. publicBaseURL is the user-facing frontend
// origin (e.g. https://app.example.com) used to build shareable invite links.
func NewService(repo teamRepo, publicBaseURL string) *Service {
	return &Service{repo: repo, publicBaseURL: strings.TrimRight(publicBaseURL, "/")}
}

// ListForUser returns all teams for the given user enriched with member count,
// the user's roles, and merged permissions.
func (s *Service) ListForUser(ctx context.Context, userID string) ([]gen.TeamForUser, error) {
	teams, err := s.repo.ListTeamsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.ListForUser: %w", err)
	}

	out := make([]gen.TeamForUser, 0, len(teams))
	for _, t := range teams {
		tfu, err := s.enrichTeamForUser(ctx, t, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, *tfu)
	}
	return out, nil
}

// CreateTeam creates a new team and returns it enriched for the creator.
func (s *Service) CreateTeam(ctx context.Context, userID, name string) (*gen.TeamForUser, error) {
	tr, err := s.repo.CreateTeam(ctx, name, userID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.CreateTeam: %w", err)
	}
	tfu, err := s.enrichTeamForUser(ctx, *tr, userID)
	if err != nil {
		return nil, err
	}
	return tfu, nil
}

// GetTeam returns the gen.Team for the given ID.
func (s *Service) GetTeam(ctx context.Context, teamID string) (*gen.Team, error) {
	tr, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("teams.Service.GetTeam: %w", err)
	}
	return toGenTeam(tr), nil
}

// UpdateTeam applies a patch and returns the updated gen.Team.
func (s *Service) UpdateTeam(ctx context.Context, teamID string, patch TeamPatch) (*gen.Team, error) {
	tr, err := s.repo.UpdateTeam(ctx, teamID, patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("teams.Service.UpdateTeam: %w", err)
	}
	return toGenTeam(tr), nil
}

// CreateInvite creates a 7-day invite for the team and returns it.
func (s *Service) CreateInvite(ctx context.Context, teamID string) (*gen.Invite, error) {
	inv, err := s.repo.CreateInvite(ctx, teamID, inviteTTL)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.CreateInvite: %w", err)
	}
	link := fmt.Sprintf("%s/join/%s/%s", s.publicBaseURL, teamID, inv.Code)
	return &gen.Invite{
		Id:        inv.Id,
		TeamId:    inv.TeamID,
		Code:      inv.Code,
		Link:      link,
		ExpiresAt: inv.ExpiresAt,
		CreatedAt: inv.CreatedAt,
	}, nil
}

// GetTeamPhotoData returns the raw photo bytes and MIME type for the given team.
func (s *Service) GetTeamPhotoData(ctx context.Context, teamID string) (data []byte, mime string, err error) {
	tr, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", pgx.ErrNoRows
		}
		return nil, "", fmt.Errorf("teams.Service.GetTeamPhotoData: %w", err)
	}
	if len(tr.PhotoData) == 0 {
		return nil, "", pgx.ErrNoRows
	}
	mime = "image/jpeg"
	if tr.PhotoMime != nil && *tr.PhotoMime != "" {
		mime = *tr.PhotoMime
	}
	return tr.PhotoData, mime, nil
}

// UpdatePhoto resizes and stores the team photo, returning the updated gen.Team.
func (s *Service) UpdatePhoto(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error) {
	resized, err := resizeImage(data, mime)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.UpdatePhoto: resize: %w", err)
	}
	if err := s.repo.UpdateTeamPhoto(ctx, teamID, resized, "image/jpeg"); err != nil {
		return nil, fmt.Errorf("teams.Service.UpdatePhoto: store: %w", err)
	}
	tr, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.UpdatePhoto: refresh: %w", err)
	}
	return toGenTeam(tr), nil
}

// GetTeamLogoData returns the raw logo bytes and MIME type for the given team.
func (s *Service) GetTeamLogoData(ctx context.Context, teamID string) (data []byte, mime string, err error) {
	tr, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", pgx.ErrNoRows
		}
		return nil, "", fmt.Errorf("teams.Service.GetTeamLogoData: %w", err)
	}
	if len(tr.LogoData) == 0 {
		return nil, "", pgx.ErrNoRows
	}
	mime = "image/jpeg"
	if tr.LogoMime != nil && *tr.LogoMime != "" {
		mime = *tr.LogoMime
	}
	return tr.LogoData, mime, nil
}

// UpdateLogo resizes and stores the team logo, returning the updated gen.Team.
func (s *Service) UpdateLogo(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error) {
	resized, err := resizeImage(data, mime)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.UpdateLogo: resize: %w", err)
	}
	if err := s.repo.UpdateTeamLogo(ctx, teamID, resized, "image/jpeg"); err != nil {
		return nil, fmt.Errorf("teams.Service.UpdateLogo: store: %w", err)
	}
	tr, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.UpdateLogo: refresh: %w", err)
	}
	return toGenTeam(tr), nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

func (s *Service) enrichTeamForUser(ctx context.Context, tr TeamRow, userID string) (*gen.TeamForUser, error) {
	count, err := s.repo.GetMemberCount(ctx, tr.Id.String())
	if err != nil {
		return nil, fmt.Errorf("teams.Service.enrichTeamForUser count: %w", err)
	}

	m, err := s.repo.GetMembership(ctx, tr.Id.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.enrichTeamForUser membership: %w", err)
	}

	roles, err := s.repo.GetRolesForMembership(ctx, m.Id.String(), tr.Id.String())
	if err != nil {
		return nil, fmt.Errorf("teams.Service.enrichTeamForUser roles: %w", err)
	}

	genRoles := make([]gen.Role, len(roles))
	for i, r := range roles {
		genRoles[i] = toGenRole(r)
	}

	perms := MergePermissions(roles)
	hasPhoto := len(tr.PhotoData) > 0
	hasLogo := len(tr.LogoData) > 0

	tfu := &gen.TeamForUser{
		Id:           tr.Id,
		Name:         tr.Name,
		Short:        tr.Short,
		Icon:         tr.Icon,
		IconBg:       tr.IconBg,
		IconFg:       tr.IconFg,
		Description:  tr.Description,
		HasPhoto:     &hasPhoto,
		HasLogo:      &hasLogo,
		MemberCount:  count,
		MembershipId: m.Id,
		MyRoles:      genRoles,
		MyPerms:      perms,
	}

	if len(tr.ReasonVisibilityRoleIDs) > 0 {
		uuids := make([]openapi_types.UUID, len(tr.ReasonVisibilityRoleIDs))
		copy(uuids, tr.ReasonVisibilityRoleIDs)
		tfu.ReasonVisibilityRoleIds = &uuids
	}

	return tfu, nil
}

func toGenTeam(tr *TeamRow) *gen.Team {
	hasPhoto := len(tr.PhotoData) > 0
	hasLogo := len(tr.LogoData) > 0
	t := &gen.Team{
		Id:          tr.Id,
		Name:        tr.Name,
		Short:       tr.Short,
		Icon:        tr.Icon,
		IconBg:      tr.IconBg,
		IconFg:      tr.IconFg,
		Description: tr.Description,
		HasPhoto:    &hasPhoto,
		HasLogo:     &hasLogo,
	}
	if len(tr.ReasonVisibilityRoleIDs) > 0 {
		uuids := make([]openapi_types.UUID, len(tr.ReasonVisibilityRoleIDs))
		copy(uuids, tr.ReasonVisibilityRoleIDs)
		t.ReasonVisibilityRoleIds = &uuids
	}
	return t
}

func toGenRole(r RoleRow) gen.Role {
	return gen.Role{
		Id:     r.Id,
		TeamId: r.TeamID,
		Name:   r.Name,
		System: r.System,
		Color:  r.Color,
		Permissions: gen.Permissions{
			Events:   gen.PermLevel(r.Permissions.Events),
			Members:  gen.PermLevel(r.Permissions.Members),
			Finances: gen.PermLevel(r.Permissions.Finances),
			News:     gen.PermLevel(r.Permissions.News),
			Polls:    gen.PermLevel(r.Permissions.Polls),
			Settings: gen.PermLevel(r.Permissions.Settings),
		},
	}
}

func resizeImage(data []byte, mime string) ([]byte, error) {
	reader := bytes.NewReader(data)

	var src image.Image
	var decodeErr error

	switch mime {
	case "image/png":
		src, decodeErr = png.Decode(reader)
	default:
		src, decodeErr = jpeg.Decode(reader)
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("decode image: %w", decodeErr)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}
