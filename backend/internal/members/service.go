package members

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// memberRepo is the interface the Service relies on.
type memberRepo interface {
	ListMembers(ctx context.Context, teamID string, limit, offset int) ([]MemberRow, error)
	AddMember(ctx context.Context, teamID string, params AddMemberParams) (*MemberRow, error)
	UpdateMember(ctx context.Context, membershipID string, patch MemberPatch) (*MemberRow, error)
	SetRoles(ctx context.Context, membershipID string, roleIDs []string) (*MemberRow, error)
	RemoveMember(ctx context.Context, membershipID string) error
}

// Service implements member business logic.
type Service struct {
	repo memberRepo
}

// NewService creates a new Service.
func NewService(repo memberRepo) *Service {
	return &Service{repo: repo}
}

// ListMembers returns paginated members of a team as gen.Member objects.
func (s *Service) ListMembers(ctx context.Context, teamID string, limit, offset int) ([]gen.Member, error) {
	rows, err := s.repo.ListMembers(ctx, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("members.Service.ListMembers: %w", err)
	}
	out := make([]gen.Member, len(rows))
	for i, r := range rows {
		out[i] = toGenMember(r)
	}
	return out, nil
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
func (s *Service) UpdateMember(ctx context.Context, membershipID string, patch MemberPatch) (*gen.Member, error) {
	mr, err := s.repo.UpdateMember(ctx, membershipID, patch)
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
func (s *Service) SetRoles(ctx context.Context, membershipID string, roleIDs []string) (*gen.Member, error) {
	mr, err := s.repo.SetRoles(ctx, membershipID, roleIDs)
	if err != nil {
		return nil, fmt.Errorf("members.Service.SetRoles: %w", err)
	}
	m := toGenMember(*mr)
	return &m, nil
}

// RemoveMember removes a member from the team.
func (s *Service) RemoveMember(ctx context.Context, membershipID string) error {
	if err := s.repo.RemoveMember(ctx, membershipID); err != nil {
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
		MembershipId: openapi_types.UUID(mr.MembershipID),
		UserId:       openapi_types.UUID(mr.UserID),
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
		Id:     openapi_types.UUID(r.Id),
		TeamId: openapi_types.UUID(r.TeamID),
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
