package members

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/storage"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// memberRepo is the interface the Service relies on.
type memberRepo interface {
	ListMembers(ctx context.Context, teamID string, limit int, cur *ListCursor) ([]MemberRow, error)
	GetMemberPhotoKey(ctx context.Context, teamID, membershipID string) (string, error)
	UpdateMember(ctx context.Context, membershipID, teamID, callerUserID string, patch MemberPatch) (*MemberRow, error)
	SetMemberTitle(ctx context.Context, membershipID, teamID, callerUserID string, title *string) (*MemberRow, error)
	SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string, callerUserID string) (*MemberRow, error)
	RemoveMember(ctx context.Context, membershipID, teamID, callerUserID string) error
}

// Service implements member business logic.
type Service struct {
	repo  memberRepo
	store storage.ObjectStore
	pager *pagination.Paginator
}

// NewService creates a new Service. pager may be nil, in which case a default
// (unsigned) Paginator is used.
func NewService(repo memberRepo, store storage.ObjectStore, pager *pagination.Paginator) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, store: store, pager: pager}
}

// GetMemberPhotoURL returns a short-lived presigned URL for the given
// membership's photo, or pgx.ErrNoRows if the membership doesn't belong to
// teamID or the member has no photo set.
func (s *Service) GetMemberPhotoURL(ctx context.Context, teamID, membershipID string) (string, error) {
	key, err := s.repo.GetMemberPhotoKey(ctx, teamID, membershipID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("members.Service.GetMemberPhotoURL: %w", err)
	}
	url, err := s.store.PresignGet(ctx, key, storage.PresignTTL)
	if err != nil {
		return "", fmt.Errorf("members.Service.GetMemberPhotoURL: %w", err)
	}
	return url, nil
}

// GetMemberPhotoBytes returns the given membership's photo as a byte stream
// plus its content type, or pgx.ErrNoRows if the membership doesn't belong to
// teamID or the member has no photo set. Used instead of GetMemberPhotoURL
// when the deployment is configured for proxy image delivery
// (config.Config.ImageDeliveryProxyEnabled) -- the caller streams the bytes
// through the backend rather than redirecting to a presigned URL. The caller
// must Close the returned ReadCloser.
func (s *Service) GetMemberPhotoBytes(ctx context.Context, teamID, membershipID string) (io.ReadCloser, string, error) {
	key, err := s.repo.GetMemberPhotoKey(ctx, teamID, membershipID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", pgx.ErrNoRows
		}
		return nil, "", fmt.Errorf("members.Service.GetMemberPhotoBytes: %w", err)
	}
	data, contentType, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, "", fmt.Errorf("members.Service.GetMemberPhotoBytes: %w", err)
	}
	return data, contentType, nil
}

// ListMembers returns a keyset page of members plus the cursor for the next
// page (nil on the last page). cursor is the opaque token from a prior page
// ("" = first page).
func (s *Service) ListMembers(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error) {
	var cur *ListCursor
	var decoded ListCursor
	if ok, err := s.pager.Decode(cursor, &decoded); err != nil {
		return nil, nil, fmt.Errorf("members.Service.ListMembers: %w", err)
	} else if ok {
		cur = &decoded
	}

	rows, err := s.repo.ListMembers(ctx, teamID, limit+1, cur)
	if err != nil {
		return nil, nil, fmt.Errorf("members.Service.ListMembers: %w", err)
	}

	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		token, err := s.pager.Encode(ListCursor{Name: last.Name, ID: last.MembershipID})
		if err != nil {
			return nil, nil, fmt.Errorf("members.Service.ListMembers: %w", err)
		}
		next = &token
	}

	out := make([]gen.Member, len(rows))
	for i, r := range rows {
		out[i] = toGenMember(r)
	}
	return out, next, nil
}

// UpdateMember updates member profile and returns the updated gen.Member.
// callerUserID is the authenticated caller, used to require settings:write
// when the patch changes another member's email (see ErrCannotChangeOthersEmail).
func (s *Service) UpdateMember(ctx context.Context, membershipID, teamID, callerUserID string, patch MemberPatch) (*gen.Member, error) {
	mr, err := s.repo.UpdateMember(ctx, membershipID, teamID, callerUserID, patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("members.Service.UpdateMember: %w", err)
	}
	m := toGenMember(*mr)
	return &m, nil
}

// SetMemberTitle sets or clears the caller's own title on membershipID.
func (s *Service) SetMemberTitle(ctx context.Context, membershipID, teamID, callerUserID string, title *string) (*gen.Member, error) {
	mr, err := s.repo.SetMemberTitle(ctx, membershipID, teamID, callerUserID, title)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		if errors.Is(err, ErrCannotSetOthersTitle) {
			return nil, ErrCannotSetOthersTitle
		}
		return nil, fmt.Errorf("members.Service.SetMemberTitle: %w", err)
	}
	m := toGenMember(*mr)
	return &m, nil
}

// SetRoles replaces the member's role assignments. callerUserID is the
// acting user, used to enforce that they cannot grant a permission level
// they do not themselves hold (see enforceNoPermissionEscalation).
func (s *Service) SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string, callerUserID string) (*gen.Member, error) {
	mr, err := s.repo.SetRoles(ctx, membershipID, teamID, roleIDs, callerUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		if errors.Is(err, ErrRoleNotInTeam) {
			return nil, ErrRoleNotInTeam
		}
		if errors.Is(err, ErrLastSettingsAdmin) {
			return nil, ErrLastSettingsAdmin
		}
		if errors.Is(err, ErrInsufficientPermissionToGrant) {
			return nil, ErrInsufficientPermissionToGrant
		}
		return nil, fmt.Errorf("members.Service.SetRoles: %w", err)
	}
	m := toGenMember(*mr)
	return &m, nil
}

// RemoveMember removes a member from the team.
func (s *Service) RemoveMember(ctx context.Context, membershipID, teamID, callerUserID string) error {
	if err := s.repo.RemoveMember(ctx, membershipID, teamID, callerUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		if errors.Is(err, ErrLastSettingsAdmin) {
			return ErrLastSettingsAdmin
		}
		if errors.Is(err, ErrCannotRemoveSettingsAdmin) {
			return ErrCannotRemoveSettingsAdmin
		}
		return fmt.Errorf("members.Service.RemoveMember: %w", err)
	}
	return nil
}

// ─── mapper ──────────────────────────────────────────────────────────────────

func toGenMember(mr MemberRow) gen.Member {
	hasPhoto := mr.HasPhoto

	genRoles := make([]gen.Role, len(mr.Roles))
	for i, r := range mr.Roles {
		genRoles[i] = teams.ToGenRole(r)
	}

	// Primary role = first role, ordered by role id (arbitrary but
	// deterministic -- repository.go's batchGetRoles/getRolesForMembershipQ
	// both ORDER BY r.id so this agrees with events.batchGetPrimaryRoles'
	// identical convention for the same membership).
	var primaryRole *gen.Role
	if len(genRoles) > 0 {
		primaryRole = &genRoles[0]
	}

	// Merged permissions.
	perms := teams.MergePermissions(mr.Roles)

	m := gen.Member{
		MembershipId:     mr.MembershipID,
		UserId:           mr.UserID,
		Name:             mr.Name,
		Email:            openapi_types.Email(mr.Email),
		Phone:            mr.Phone,
		AvatarColor:      mr.AvatarColor,
		HasPhoto:         &hasPhoto,
		Group:            mr.Group,
		Title:            mr.Title,
		JoinedAt:         mr.JoinedAt,
		Roles:            genRoles,
		PrimaryRole:      primaryRole,
		Perms:            &perms,
		ExcludeFromStats: mr.ExcludeFromStats,
	}

	if mr.Address != nil {
		m.Address = mr.Address
	}
	if mr.Birthday != nil {
		d := openapi_types.Date{Time: *mr.Birthday}
		m.Birthday = &d
	}

	return m
}
