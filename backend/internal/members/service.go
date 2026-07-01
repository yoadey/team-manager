package members

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// memberRepo is the interface the Service relies on.
type memberRepo interface {
	ListMembers(ctx context.Context, teamID string, limit int, cur *ListCursor) ([]MemberRow, error)
	AddMember(ctx context.Context, teamID string, params AddMemberParams) (*MemberRow, error)
	UpdateMember(ctx context.Context, membershipID, teamID string, patch MemberPatch) (*MemberRow, error)
	SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string) (*MemberRow, error)
	RemoveMember(ctx context.Context, membershipID, teamID string) error
}

// Service implements member business logic.
type Service struct {
	repo  memberRepo
	pager *pagination.Paginator
}

// NewService creates a new Service. pager may be nil, in which case a default
// (unsigned) Paginator is used.
func NewService(repo memberRepo, pager *pagination.Paginator) *Service {
	if pager == nil {
		pager = pagination.New(nil)
	}
	return &Service{repo: repo, pager: pager}
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

// AddMember adds a member and returns the gen.Member.
func (s *Service) AddMember(ctx context.Context, teamID string, params AddMemberParams) (*gen.Member, error) {
	mr, err := s.repo.AddMember(ctx, teamID, params)
	if err != nil {
		return nil, fmt.Errorf("members.Service.AddMember: %w", err)
	}
	m := toGenMember(*mr)
	return &m, nil
}

// UpdateMember updates member profile and returns the updated gen.Member.
func (s *Service) UpdateMember(ctx context.Context, membershipID, teamID string, patch MemberPatch) (*gen.Member, error) {
	mr, err := s.repo.UpdateMember(ctx, membershipID, teamID, patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("members.Service.UpdateMember: %w", err)
	}
	m := toGenMember(*mr)
	return &m, nil
}

// SetRoles replaces the member's role assignments.
func (s *Service) SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string) (*gen.Member, error) {
	mr, err := s.repo.SetRoles(ctx, membershipID, teamID, roleIDs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		if errors.Is(err, ErrRoleNotInTeam) {
			return nil, ErrRoleNotInTeam
		}
		return nil, fmt.Errorf("members.Service.SetRoles: %w", err)
	}
	m := toGenMember(*mr)
	return &m, nil
}

// RemoveMember removes a member from the team.
func (s *Service) RemoveMember(ctx context.Context, membershipID, teamID string) error {
	if err := s.repo.RemoveMember(ctx, membershipID, teamID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return fmt.Errorf("members.Service.RemoveMember: %w", err)
	}
	return nil
}

// ─── mapper ──────────────────────────────────────────────────────────────────

func toGenMember(mr MemberRow) gen.Member {
	hasPhoto := len(mr.PhotoData) > 0

	genRoles := make([]gen.Role, len(mr.Roles))
	for i, r := range mr.Roles {
		genRoles[i] = toGenRole(r)
	}

	// Primary role = first role (arbitrary ordering).
	var primaryRole *gen.Role
	if len(genRoles) > 0 {
		primaryRole = &genRoles[0]
	}

	// Merged permissions.
	perms := teams.MergePermissions(mr.Roles)

	m := gen.Member{
		MembershipId: mr.MembershipID,
		UserId:       mr.UserID,
		Name:         mr.Name,
		Email:        openapi_types.Email(mr.Email),
		Phone:        mr.Phone,
		AvatarColor:  mr.AvatarColor,
		HasPhoto:     &hasPhoto,
		Group:        mr.Group,
		JoinedAt:     mr.JoinedAt,
		Roles:        genRoles,
		PrimaryRole:  primaryRole,
		Perms:        &perms,
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

func toGenRole(r teams.RoleRow) gen.Role {
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

// ensure time is used.
var _ = time.Time{}
