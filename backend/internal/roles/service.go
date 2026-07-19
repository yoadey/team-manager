package roles

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// roleRepo is the interface the Service relies on.
type roleRepo interface {
	ListRoles(ctx context.Context, teamID string) ([]teams.RoleRow, error)
	CreateRole(ctx context.Context, teamID, name string, color *string, permissions teams.PermissionsJSON) (*teams.RoleRow, error)
	UpdateRole(ctx context.Context, roleID, teamID, callerUserID string, patch RolePatch) (*teams.RoleRow, error)
	DeleteRole(ctx context.Context, roleID, teamID string) error
}

// Service implements role business logic.
type Service struct {
	repo roleRepo
}

// NewService creates a new Service.
func NewService(repo roleRepo) *Service {
	return &Service{repo: repo}
}

// ListRoles returns all roles for the given team.
func (s *Service) ListRoles(ctx context.Context, teamID uuid.UUID) ([]gen.Role, error) {
	rows, err := s.repo.ListRoles(ctx, teamID.String())
	if err != nil {
		return nil, fmt.Errorf("roles.Service.ListRoles: %w", err)
	}
	out := make([]gen.Role, 0, len(rows))
	for _, r := range rows {
		out = append(out, teams.ToGenRole(r))
	}
	return out, nil
}

// CreateRole creates a new custom role.
func (s *Service) CreateRole(ctx context.Context, teamID uuid.UUID, body *gen.CreateRoleJSONRequestBody) (*gen.Role, error) {
	perms := toInternalPermissions(body.Permissions)
	row, err := s.repo.CreateRole(ctx, teamID.String(), body.Name, body.Color, perms)
	if err != nil {
		return nil, fmt.Errorf("roles.Service.CreateRole: %w", err)
	}
	result := teams.ToGenRole(*row)
	return &result, nil
}

// UpdateRole applies a patch to a role that belongs to teamID. callerUserID
// is the authenticated caller, used to enforce that a permissions patch
// can't grant more than the caller's own ceiling allows (see
// enforceNoRoleEscalation).
func (s *Service) UpdateRole(ctx context.Context, roleID, teamID, callerUserID uuid.UUID, body *gen.UpdateRoleJSONRequestBody) (*gen.Role, error) {
	patch := RolePatch{Name: body.Name, Color: body.Color}
	if body.Permissions != nil {
		p := toInternalPermissions(*body.Permissions)
		patch.Permissions = &p
	}
	row, err := s.repo.UpdateRole(ctx, roleID.String(), teamID.String(), callerUserID.String(), patch)
	if err != nil {
		return nil, fmt.Errorf("roles.Service.UpdateRole: %w", err)
	}
	result := teams.ToGenRole(*row)
	return &result, nil
}

// DeleteRole deletes a non-system role that belongs to teamID.
func (s *Service) DeleteRole(ctx context.Context, roleID, teamID uuid.UUID) error {
	if err := s.repo.DeleteRole(ctx, roleID.String(), teamID.String()); err != nil {
		return fmt.Errorf("roles.Service.DeleteRole: %w", err)
	}
	return nil
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func toInternalPermissions(p gen.Permissions) teams.PermissionsJSON {
	return teams.PermissionsJSON{
		Events:   string(p.Events),
		Members:  string(p.Members),
		Finances: string(p.Finances),
		News:     string(p.News),
		Polls:    string(p.Polls),
		Settings: string(p.Settings),
	}
}
