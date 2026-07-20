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
	"golang.org/x/image/draw"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/storage"
)

const inviteTTL = 7 * 24 * time.Hour

// ErrImageTooLarge is returned by resizeImage when the uploaded image's
// dimensions exceed maxDecodePixels.
var ErrImageTooLarge = errors.New("teams.resizeImage: image dimensions exceed the allowed maximum")

// ErrTooManyTeams is returned once a user hits maxTeamsPerUser.
var ErrTooManyTeams = fmt.Errorf("account has reached the maximum of %d teams", maxTeamsPerUser)

// maxTeamsPerUser caps how many teams a single account can create/join.
// ListForUser's own doc comment notes it batches enrichment across "an
// unbounded number of teams" in 3 queries total rather than per-team --
// that batching keeps per-team overhead low, but every one of those queries
// (plus ListTeamsForUser itself) still runs under the same fixed 5s
// timeout, against the same shared connection pool, that every other query
// in this codebase uses to bound worst-case cost. GET /teams is hit on
// essentially every session, so without a cap, one account creating an
// unbounded number of teams turns its own login into a query that holds a
// pool connection for up to 5s each -- and since the pool is shared across
// every team and every user, enough concurrent requests from that one
// account can exhaust it for everyone, not just the account that caused it.
// 500 comfortably covers any real user; this exists to stop runaway/
// malicious creation, not to constrain legitimate multi-team membership.
const maxTeamsPerUser = 500

// maxLogoDim caps the longest edge of a resized team photo/logo.
const maxLogoDim = 800

// maxDecodePixels caps the total pixel count of an uploaded image before it
// is fully decoded, bounding peak memory against decompression-bomb inputs
// (a small compressed file can declare enormous dimensions that blow up
// memory when fully decoded — the 2 MB upload-size limit in readMultipartImage
// does not bound the decoded pixel count). 50 MP comfortably covers
// legitimate phone-camera photos. Mirrors internal/auth/service.go's
// resizeImage, which guards user profile photo uploads the same way.
const maxDecodePixels = 50_000_000

// teamRepo is the interface the Service relies on.
type teamRepo interface {
	ListTeamsForUser(ctx context.Context, userID string) ([]TeamRow, error)
	CountTeamsForUser(ctx context.Context, userID string) (int, error)
	GetTeam(ctx context.Context, teamID string) (*TeamRow, error)
	CreateTeam(ctx context.Context, name, creatorUserID string, icon, iconBg, iconFg *string) (*TeamRow, error)
	UpdateTeam(ctx context.Context, teamID string, patch TeamPatch) (*TeamRow, error)
	GetMemberCount(ctx context.Context, teamID string) (int, error)
	GetMembership(ctx context.Context, teamID, userID string) (*MembershipRow, error)
	GetRolesForMembership(ctx context.Context, membershipID, teamID string) ([]RoleRow, error)
	GetMemberCounts(ctx context.Context, teamIDs []string) (map[string]int, error)
	GetMembershipsForUser(ctx context.Context, teamIDs []string, userID string) (map[string]MembershipRow, error)
	GetRolesForMemberships(ctx context.Context, membershipIDs []string) (map[string][]RoleRow, error)
	CreateInvite(ctx context.Context, teamID string, ttl time.Duration) (*InviteRow, error)
	AcceptInvite(ctx context.Context, code, userID string) (*TeamRow, bool, error)
	GetTeamPhotoKey(ctx context.Context, teamID string) (string, error)
	GetTeamLogoKey(ctx context.Context, teamID string) (string, error)
	UpdateTeamPhoto(ctx context.Context, teamID, objectKey string) error
	UpdateTeamLogo(ctx context.Context, teamID, objectKey string) error
	DeleteTeamPhoto(ctx context.Context, teamID string) error
	DeleteTeamLogo(ctx context.Context, teamID string) error
}

// Service implements team business logic.
type Service struct {
	repo  teamRepo
	store storage.ObjectStore
	// publicBaseURL is the user-facing frontend origin used to build shareable
	// invite links (no trailing slash).
	publicBaseURL string
}

// NewService creates a new Service. publicBaseURL is the user-facing frontend
// origin (e.g. https://app.example.com) used to build shareable invite links.
func NewService(repo teamRepo, store storage.ObjectStore, publicBaseURL string) *Service {
	return &Service{repo: repo, store: store, publicBaseURL: strings.TrimRight(publicBaseURL, "/")}
}

// teamPhotoKey/teamLogoKey return the object store key for a team's photo/logo.
func teamPhotoKey(teamID string) string { return "teams/" + teamID + "/photo" }
func teamLogoKey(teamID string) string  { return "teams/" + teamID + "/logo" }

// ListForUser returns all teams for the given user enriched with member count,
// the user's roles, and merged permissions. Enrichment is batched across all
// of the user's teams (3 queries total) rather than per-team, since this
// backs GET /teams -- hit on essentially every session -- and a user can
// belong to an unbounded number of teams.
func (s *Service) ListForUser(ctx context.Context, userID string) ([]gen.TeamForUser, error) {
	teams, err := s.repo.ListTeamsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.ListForUser: %w", err)
	}
	if len(teams) == 0 {
		return []gen.TeamForUser{}, nil
	}

	teamIDs := make([]string, len(teams))
	for i, t := range teams {
		teamIDs[i] = t.Id.String()
	}

	counts, err := s.repo.GetMemberCounts(ctx, teamIDs)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.ListForUser counts: %w", err)
	}
	memberships, err := s.repo.GetMembershipsForUser(ctx, teamIDs, userID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.ListForUser memberships: %w", err)
	}

	membershipIDs := make([]string, 0, len(memberships))
	for _, m := range memberships {
		membershipIDs = append(membershipIDs, m.Id.String())
	}
	rolesByMembership, err := s.repo.GetRolesForMemberships(ctx, membershipIDs)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.ListForUser roles: %w", err)
	}

	out := make([]gen.TeamForUser, 0, len(teams))
	for _, t := range teams {
		m, ok := memberships[t.Id.String()]
		if !ok {
			return nil, fmt.Errorf("teams.Service.ListForUser: %w for team %s", pgx.ErrNoRows, t.Id)
		}
		roles := rolesByMembership[m.Id.String()]
		out = append(out, *buildTeamForUser(t, m, counts[t.Id.String()], roles))
	}
	return out, nil
}

// CreateTeam creates a new team and returns it enriched for the creator.
func (s *Service) CreateTeam(ctx context.Context, userID, name string, icon, iconBg, iconFg *string) (*gen.TeamForUser, error) {
	count, err := s.repo.CountTeamsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.CreateTeam: %w", err)
	}
	if count >= maxTeamsPerUser {
		return nil, ErrTooManyTeams
	}

	tr, err := s.repo.CreateTeam(ctx, name, userID, icon, iconBg, iconFg)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.CreateTeam: %w", err)
	}
	tfu, err := s.enrichTeamForUser(ctx, *tr, userID)
	if err != nil {
		return nil, err
	}
	return tfu, nil
}

// AcceptInvite redeems a 7-day invite code, adding userID to the invite's
// team, and returns that team enriched for the caller, plus whether the
// caller was already a member (a no-op join-wise).
func (s *Service) AcceptInvite(ctx context.Context, code, userID string) (*gen.AcceptInviteResponse, error) {
	tr, alreadyMember, err := s.repo.AcceptInvite(ctx, code, userID)
	if err != nil {
		if errors.Is(err, ErrInviteNotFound) {
			return nil, ErrInviteNotFound
		}
		return nil, fmt.Errorf("teams.Service.AcceptInvite: %w", err)
	}
	tfu, err := s.enrichTeamForUser(ctx, *tr, userID)
	if err != nil {
		return nil, err
	}
	return &gen.AcceptInviteResponse{
		Id:                      tfu.Id,
		Name:                    tfu.Name,
		Short:                   tfu.Short,
		Icon:                    tfu.Icon,
		IconBg:                  tfu.IconBg,
		IconFg:                  tfu.IconFg,
		Description:             tfu.Description,
		HasPhoto:                tfu.HasPhoto,
		HasLogo:                 tfu.HasLogo,
		MemberCount:             tfu.MemberCount,
		MembershipId:            tfu.MembershipId,
		MyRoles:                 tfu.MyRoles,
		MyPerms:                 tfu.MyPerms,
		ReasonVisibilityRoleIds: tfu.ReasonVisibilityRoleIds,
		AlreadyMember:           alreadyMember,
	}, nil
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
		if errors.Is(err, ErrRoleNotInTeam) {
			return nil, ErrRoleNotInTeam
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

// GetTeamPhotoURL returns a short-lived presigned URL for the team's photo,
// or pgx.ErrNoRows if the team has no photo set.
func (s *Service) GetTeamPhotoURL(ctx context.Context, teamID string) (string, error) {
	key, err := s.repo.GetTeamPhotoKey(ctx, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("teams.Service.GetTeamPhotoURL: %w", err)
	}
	url, err := s.store.PresignGet(ctx, key, storage.PresignTTL)
	if err != nil {
		return "", fmt.Errorf("teams.Service.GetTeamPhotoURL: %w", err)
	}
	return url, nil
}

// UpdatePhoto resizes the image, uploads it to the object store, stores the
// key, and returns the updated gen.Team. Upload order is S3 put before the DB
// write; if the DB write fails, the just-uploaded object is deleted
// (best-effort) so it doesn't linger orphaned.
func (s *Service) UpdatePhoto(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error) {
	resized, err := resizeImage(data, mime)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.UpdatePhoto: resize: %w", err)
	}
	key := teamPhotoKey(teamID)
	if err := s.store.Put(ctx, key, resized, "image/jpeg"); err != nil {
		return nil, fmt.Errorf("teams.Service.UpdatePhoto: upload: %w", err)
	}
	if err := s.repo.UpdateTeamPhoto(ctx, teamID, key); err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, fmt.Errorf("teams.Service.UpdatePhoto: store: %w", err)
	}
	tr, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.UpdatePhoto: refresh: %w", err)
	}
	return toGenTeam(tr), nil
}

// DeletePhoto clears the team photo, reverting display to the icon fallback,
// and best-effort deletes the underlying object.
func (s *Service) DeletePhoto(ctx context.Context, teamID string) error {
	key, keyErr := s.repo.GetTeamPhotoKey(ctx, teamID)
	if err := s.repo.DeleteTeamPhoto(ctx, teamID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("teams.Service.DeletePhoto: %w", err)
	}
	if keyErr == nil && key != "" {
		_ = s.store.Delete(ctx, key)
	}
	return nil
}

// GetTeamLogoURL returns a short-lived presigned URL for the team's logo, or
// pgx.ErrNoRows if the team has no logo set.
func (s *Service) GetTeamLogoURL(ctx context.Context, teamID string) (string, error) {
	key, err := s.repo.GetTeamLogoKey(ctx, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("teams.Service.GetTeamLogoURL: %w", err)
	}
	url, err := s.store.PresignGet(ctx, key, storage.PresignTTL)
	if err != nil {
		return "", fmt.Errorf("teams.Service.GetTeamLogoURL: %w", err)
	}
	return url, nil
}

// UpdateLogo resizes the image, uploads it to the object store, stores the
// key, and returns the updated gen.Team.
func (s *Service) UpdateLogo(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error) {
	resized, err := resizeImage(data, mime)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.UpdateLogo: resize: %w", err)
	}
	key := teamLogoKey(teamID)
	if err := s.store.Put(ctx, key, resized, "image/jpeg"); err != nil {
		return nil, fmt.Errorf("teams.Service.UpdateLogo: upload: %w", err)
	}
	if err := s.repo.UpdateTeamLogo(ctx, teamID, key); err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, fmt.Errorf("teams.Service.UpdateLogo: store: %w", err)
	}
	tr, err := s.repo.GetTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("teams.Service.UpdateLogo: refresh: %w", err)
	}
	return toGenTeam(tr), nil
}

// DeleteLogo clears the team logo, reverting display to the icon fallback,
// and best-effort deletes the underlying object.
func (s *Service) DeleteLogo(ctx context.Context, teamID string) error {
	key, keyErr := s.repo.GetTeamLogoKey(ctx, teamID)
	if err := s.repo.DeleteTeamLogo(ctx, teamID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("teams.Service.DeleteLogo: %w", err)
	}
	if keyErr == nil && key != "" {
		_ = s.store.Delete(ctx, key)
	}
	return nil
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

	return buildTeamForUser(tr, *m, count, roles), nil
}

// buildTeamForUser assembles the API shape from already-fetched pieces, so
// both the single-team path (enrichTeamForUser) and the batched list path
// (ListForUser) share one assembly implementation.
func buildTeamForUser(tr TeamRow, m MembershipRow, count int, roles []RoleRow) *gen.TeamForUser {
	genRoles := make([]gen.Role, len(roles))
	for i, r := range roles {
		genRoles[i] = ToGenRole(r)
	}

	perms := MergePermissions(roles)
	hasPhoto := tr.HasPhoto
	hasLogo := tr.HasLogo

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

	return tfu
}

func toGenTeam(tr *TeamRow) *gen.Team {
	hasPhoto := tr.HasPhoto
	hasLogo := tr.HasLogo
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

func ToGenRole(r RoleRow) gen.Role {
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

// resizeImage decodes a JPEG or PNG, scales it proportionally so neither
// dimension exceeds maxLogoDim, and re-encodes as JPEG.
func resizeImage(data []byte, mime string) ([]byte, error) {
	// Read the header first (cheap) and reject oversized images before a full
	// decode — see maxDecodePixels.
	var cfg image.Config
	var cfgErr error
	switch mime {
	case "image/png":
		cfg, cfgErr = png.DecodeConfig(bytes.NewReader(data))
	default:
		cfg, cfgErr = jpeg.DecodeConfig(bytes.NewReader(data))
	}
	if cfgErr != nil {
		return nil, fmt.Errorf("decode image config: %w", cfgErr)
	}
	if cfg.Width*cfg.Height > maxDecodePixels {
		return nil, fmt.Errorf("%w (%dx%d)", ErrImageTooLarge, cfg.Width, cfg.Height)
	}

	var src image.Image
	var decodeErr error

	switch mime {
	case "image/png":
		src, decodeErr = png.Decode(bytes.NewReader(data))
	default:
		src, decodeErr = jpeg.Decode(bytes.NewReader(data))
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("decode image: %w", decodeErr)
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= maxLogoDim && h <= maxLogoDim {
		// No resize needed — still re-encode as JPEG for consistency.
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("encode image: %w", err)
		}
		return buf.Bytes(), nil
	}

	// Compute new dimensions preserving aspect ratio.
	var newW, newH int
	if w > h {
		newH = h * maxLogoDim / w
		newW = maxLogoDim
	} else {
		newW = w * maxLogoDim / h
		newH = maxLogoDim
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode resized image: %w", err)
	}
	return buf.Bytes(), nil
}
