package roles

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
)

// roleService is the interface the Handler relies on.
type roleService interface {
	ListRoles(ctx context.Context, teamID uuid.UUID) ([]gen.Role, error)
	CreateRole(ctx context.Context, teamID uuid.UUID, body *gen.CreateRoleJSONRequestBody) (*gen.Role, error)
	UpdateRole(ctx context.Context, roleID uuid.UUID, body *gen.UpdateRoleJSONRequestBody) (*gen.Role, error)
	DeleteRole(ctx context.Context, roleID uuid.UUID) error
}

// Handler implements the role-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    roleService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc roleService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ListRoles returns all roles defined for the given team.
func (h *Handler) ListRoles(ctx context.Context, req gen.ListRolesRequestObject) (gen.ListRolesResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	roles, err := h.svc.ListRoles(ctx, uuid.UUID(req.TeamId))
	if err != nil {
		h.logger.ErrorContext(ctx, "ListRoles failed", "err", err)
		return nil, apierror.Internal("failed to list roles")
	}
	return gen.ListRoles200JSONResponse(roles), nil
}

// CreateRole creates a new custom role for a team.
func (h *Handler) CreateRole(ctx context.Context, req gen.CreateRoleRequestObject) (gen.CreateRoleResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	role, err := h.svc.CreateRole(ctx, uuid.UUID(req.TeamId), req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateRole failed", "err", err)
		return nil, apierror.Internal("failed to create role")
	}
	return gen.CreateRole201JSONResponse(*role), nil
}

// UpdateRole updates a role's name, color, or permissions.
func (h *Handler) UpdateRole(ctx context.Context, req gen.UpdateRoleRequestObject) (gen.UpdateRoleResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	role, err := h.svc.UpdateRole(ctx, openapi_types.UUID(req.RoleId), req.Body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("role not found")
		}
		h.logger.ErrorContext(ctx, "UpdateRole failed", "err", err)
		return nil, apierror.Internal("failed to update role")
	}
	return gen.UpdateRole200JSONResponse(*role), nil
}

// DeleteRole deletes a role and all its assignments.
func (h *Handler) DeleteRole(ctx context.Context, req gen.DeleteRoleRequestObject) (gen.DeleteRoleResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.DeleteRole(ctx, openapi_types.UUID(req.RoleId)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("role not found")
		}
		h.logger.ErrorContext(ctx, "DeleteRole failed", "err", err)
		return nil, apierror.Internal("failed to delete role")
	}
	return gen.DeleteRole204Response{}, nil
}
