package roles

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// validPermissions reports whether every per-module level in p is a known
// PermLevel value (none|read|write). Unrecognized strings currently rank as
// "none" everywhere permissions are evaluated, so this isn't a privilege
// escalation, but it's still garbage we shouldn't persist in a
// security-relevant column.
func validPermissions(p gen.Permissions) bool {
	return p.Events.Valid() && p.Members.Valid() && p.Finances.Valid() &&
		p.News.Valid() && p.Polls.Valid() && p.Settings.Valid()
}

// roleService is the interface the Handler relies on.
type roleService interface {
	ListRoles(ctx context.Context, teamID uuid.UUID) ([]gen.Role, error)
	CreateRole(ctx context.Context, teamID uuid.UUID, body *gen.CreateRoleJSONRequestBody) (*gen.Role, error)
	UpdateRole(ctx context.Context, roleID, teamID uuid.UUID, body *gen.UpdateRoleJSONRequestBody) (*gen.Role, error)
	DeleteRole(ctx context.Context, roleID, teamID uuid.UUID) error
}

// Handler implements the role-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    roleService
	logger *slog.Logger
	audit  *audit.Logger
}

// NewHandler creates a new Handler. al is the shared audit logger; when nil a
// log-only logger is created from logger.
func NewHandler(svc roleService, logger *slog.Logger, al *audit.Logger) *Handler {
	if al == nil {
		al = audit.New(logger)
	}
	return &Handler{svc: svc, logger: logger, audit: al}
}

// ListRoles returns all roles defined for the given team.
func (h *Handler) ListRoles(ctx context.Context, req gen.ListRolesRequestObject) (gen.ListRolesResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	roles, err := h.svc.ListRoles(ctx, req.TeamId)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListRoles failed", "err", err)
		return nil, apierror.Internal("failed to list roles")
	}
	return gen.ListRoles200JSONResponse(roles), nil
}

// CreateRole creates a new custom role for a team.
func (h *Handler) CreateRole(ctx context.Context, req gen.CreateRoleRequestObject) (gen.CreateRoleResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.Name(req.Body.Name); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if req.Body.Color != nil {
		if err := validate.MaxLen(*req.Body.Color, 32, "color"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if !validPermissions(req.Body.Permissions) {
		return nil, apierror.BadRequest("permissions must each be one of none, read, write")
	}
	role, err := h.svc.CreateRole(ctx, req.TeamId, req.Body)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateRole failed", "err", err)
		return nil, apierror.Internal("failed to create role")
	}
	h.audit.Record(ctx, audit.EventRoleCreate, audit.Success, user.Id.String(),
		slog.String("teamId", req.TeamId.String()), slog.String("roleId", role.Id.String()))
	metrics.TeamEvents.WithLabelValues("role", "create").Inc()
	return gen.CreateRole201JSONResponse(*role), nil
}

// UpdateRole updates a role's name, color, or permissions.
func (h *Handler) UpdateRole(ctx context.Context, req gen.UpdateRoleRequestObject) (gen.UpdateRoleResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if req.Body.Name != nil {
		if err := validate.Name(*req.Body.Name); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if req.Body.Color != nil {
		if err := validate.MaxLen(*req.Body.Color, 32, "color"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	if req.Body.Permissions != nil && !validPermissions(*req.Body.Permissions) {
		return nil, apierror.BadRequest("permissions must each be one of none, read, write")
	}
	role, err := h.svc.UpdateRole(ctx, req.RoleId, req.TeamId, req.Body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("role not found")
		}
		if errors.Is(err, ErrSystemRole) {
			h.audit.Record(ctx, audit.EventRoleUpdate, audit.Failure, user.Id.String(),
				slog.String("roleId", req.RoleId.String()), slog.String("reason", "system_role"))
			return nil, apierror.Forbidden("cannot change the name or permissions of a built-in system role")
		}
		if errors.Is(err, ErrLastSettingsAdmin) {
			h.audit.Record(ctx, audit.EventRoleUpdate, audit.Failure, user.Id.String(),
				slog.String("roleId", req.RoleId.String()), slog.String("reason", "last_settings_admin"))
			return nil, apierror.Conflict(ErrLastSettingsAdmin.Error())
		}
		h.logger.ErrorContext(ctx, "UpdateRole failed", "err", err)
		return nil, apierror.Internal("failed to update role")
	}
	h.audit.Record(ctx, audit.EventRoleUpdate, audit.Success, user.Id.String(),
		slog.String("roleId", req.RoleId.String()))
	metrics.TeamEvents.WithLabelValues("role", "update").Inc()
	return gen.UpdateRole200JSONResponse(*role), nil
}

// DeleteRole deletes a role and all its assignments.
func (h *Handler) DeleteRole(ctx context.Context, req gen.DeleteRoleRequestObject) (gen.DeleteRoleResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.DeleteRole(ctx, req.RoleId, req.TeamId); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("role not found")
		}
		if errors.Is(err, ErrSystemRole) {
			h.audit.Record(ctx, audit.EventRoleDelete, audit.Failure, user.Id.String(),
				slog.String("roleId", req.RoleId.String()), slog.String("reason", "system_role"))
			return nil, apierror.Forbidden("cannot delete a built-in system role")
		}
		if errors.Is(err, ErrLastSettingsAdmin) {
			h.audit.Record(ctx, audit.EventRoleDelete, audit.Failure, user.Id.String(),
				slog.String("roleId", req.RoleId.String()), slog.String("reason", "last_settings_admin"))
			return nil, apierror.Conflict(ErrLastSettingsAdmin.Error())
		}
		h.logger.ErrorContext(ctx, "DeleteRole failed", "err", err)
		return nil, apierror.Internal("failed to delete role")
	}
	h.audit.Record(ctx, audit.EventRoleDelete, audit.Success, user.Id.String(),
		slog.String("roleId", req.RoleId.String()))
	metrics.TeamEvents.WithLabelValues("role", "delete").Inc()
	return gen.DeleteRole204Response{}, nil
}
