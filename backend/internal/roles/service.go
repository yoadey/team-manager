package roles

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// roleRepo is the interface the Service relies on.
type roleRepo interface {
	ListRoles(ctx context.Context, teamID string) ([]teams.RoleRow, error)
	CreateRole(ctx context.Context, teamID, name string, color *string, permissions teams.PermissionsJSON) (*teams.RoleRow, error)
	UpdateRole(ctx context.Context, roleID string, patch RolePatch) (*teams.RoleRow, error)
	DeleteRole(ctx context.Context, roleID string) error
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
		out = append(out, toGenRole(r))
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
	result := toGenRole(*row)
	return &result, nil
}

// UpdateRole applies a patch to a role.
func (s *Service) UpdateRole(ctx context.Context, roleID uuid.UUID, body *gen.UpdateRoleJSONRequestBody) (*gen.Role, error) {
	patch := RolePatch{Name: body.Name, Color: body.Color}
	if body.Permissions != nil {
		p := toInternalPermissions(*body.Permissions)
		patch.Permissions = &p
	}
	row, err := s.repo.UpdateRole(ctx, roleID.String(), patch)
	if err != nil {
		return nil, fmt.Errorf("roles.Service.UpdateRole: %w", err)
	}
	result := toGenRole(*row)
	return &result, nil
}

// DeleteRole deletes a non-system role.
func (s *Service) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	if err := s.repo.DeleteRole(ctx, roleID.String()); err != nil {
		return fmt.Errorf("roles.Service.DeleteRole: %w", err)
	}
	return nil
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func toGenRole(r teams.RoleRow) gen.Role {
	permBytes, _ := json.Marshal(r.Permissions)
	var perms gen.Permissions
	_ = json.Unmarshal(permBytes, &perms)

	return gen.Role{
		Id:          openapi_types.UUID(r.Id),
		TeamId:      openapi_types.UUID(r.TeamID),
		Name:        r.Name,
		System:      r.System,
		Color:       r.Color,
		Permissions: perms,
	}
}

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
