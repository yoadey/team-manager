package members

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/teams"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// memberService is the interface the Handler relies on.
type memberService interface {
	ListMembers(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error)
	AddMember(ctx context.Context, teamID string, params AddMemberParams) (*gen.Member, error)
	UpdateMember(ctx context.Context, membershipID, teamID string, patch MemberPatch) (*gen.Member, error)
	SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string) (*gen.Member, error)
	RemoveMember(ctx context.Context, membershipID, teamID string) error
}

// permissionChecker returns the caller's effective per-module permissions for
// a team, used to enforce the "may only grant permissions you hold yourself"
// ceiling on role assignment.
type permissionChecker interface {
	GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error)
}

// Handler implements the member-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    memberService
	perms  permissionChecker
	logger *slog.Logger
	audit  *audit.Logger
}

// NewHandler creates a new Handler. al is the shared audit logger; when nil a
// log-only logger is created from logger. perms supplies the caller's
// effective permissions for the settings:write ceiling check on role
// assignment during member creation.
func NewHandler(svc memberService, perms permissionChecker, logger *slog.Logger, al *audit.Logger) *Handler {
	if al == nil {
		al = audit.New(logger)
	}
	return &Handler{svc: svc, perms: perms, logger: logger, audit: al}
}

// actor returns the acting user's id for audit records, or "" when absent.
func actor(ctx context.Context) string {
	if u, ok := auth.UserFromContext(ctx); ok {
		return u.Id.String()
	}
	return ""
}

// ListMembers returns paginated members of a team.
func (h *Handler) ListMembers(ctx context.Context, request gen.ListMembersRequestObject) (gen.ListMembersResponseObject, error) {
	limit := pagination.ParseLimit(request.Params.Limit)
	cursor := ""
	if request.Params.Cursor != nil {
		cursor = *request.Params.Cursor
	}
	members, next, err := h.svc.ListMembers(ctx, request.TeamId.String(), limit, cursor)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			return nil, apierror.BadRequest("invalid cursor")
		}
		h.logger.ErrorContext(ctx, "ListMembers failed", "err", err)
		return nil, fmt.Errorf("members.Handler.ListMembers: %w", err)
	}
	return gen.ListMembers200JSONResponse{Items: members, NextCursor: next}, nil
}

// validateAddMemberBody validates the required and optional fields of an
// AddMember request.
func validateAddMemberBody(body *gen.AddMemberRequest) error {
	if err := validate.Name(body.Name); err != nil {
		return fmt.Errorf("%w", err)
	}
	if err := validate.Email(string(body.Email)); err != nil {
		return fmt.Errorf("%w", err)
	}
	if body.Phone != nil {
		if err := validate.MaxLen(*body.Phone, 32, "phone"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	if body.Group != nil {
		if err := validate.MaxLen(*body.Group, 100, "group"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	if body.RoleIds != nil {
		if err := validate.UUIDItems(len(*body.RoleIds), "roleIds"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	return nil
}

// AddMember adds a new member to the team.
func (h *Handler) AddMember(ctx context.Context, request gen.AddMemberRequestObject) (gen.AddMemberResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validateAddMemberBody(request.Body); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	params := AddMemberParams{
		Name:  request.Body.Name,
		Email: string(request.Body.Email),
		Phone: request.Body.Phone,
		Group: request.Body.Group,
	}
	if request.Body.RoleIds != nil && len(*request.Body.RoleIds) > 0 {
		// Assigning roles at creation time is equivalent to SetMemberRoles and
		// must be gated the same way: members:write alone is not enough to
		// hand out settings:write (or any other module) to a new member.
		perms, err := h.perms.GetPermissions(ctx, request.TeamId, user.Id)
		if err != nil {
			h.logger.ErrorContext(ctx, "AddMember: permission check failed", "err", err)
			return nil, fmt.Errorf("members.Handler.AddMember: %w", err)
		}
		if perms.Settings != "write" {
			return nil, apierror.Forbidden("insufficient permissions to assign roles")
		}
		for _, u := range *request.Body.RoleIds {
			params.RoleIDs = append(params.RoleIDs, u.String())
		}
	}

	m, err := h.svc.AddMember(ctx, request.TeamId.String(), params)
	if err != nil {
		if errors.Is(err, ErrRoleNotInTeam) {
			return nil, apierror.UnprocessableEntity("one or more roles do not belong to this team")
		}
		if errors.Is(err, ErrDuplicateMembership) {
			return nil, apierror.Conflict("user is already a member of this team")
		}
		h.logger.ErrorContext(ctx, "AddMember failed", "err", err)
		return nil, fmt.Errorf("members.Handler.AddMember: %w", err)
	}
	h.audit.Record(ctx, audit.EventMemberAdd, audit.Success, actor(ctx),
		slog.String("teamId", request.TeamId.String()), slog.String("membershipId", m.MembershipId.String()))
	metrics.TeamEvents.WithLabelValues("member", "create").Inc()
	return gen.AddMember201JSONResponse(*m), nil
}

// validateMemberPatch validates the optional fields of an UpdateMember
// request body and builds the resulting MemberPatch.
func validateMemberPatch(body *gen.UpdateMemberJSONRequestBody) (MemberPatch, error) {
	patch := MemberPatch{}
	if body.Name != nil {
		if err := validate.Name(*body.Name); err != nil {
			return patch, fmt.Errorf("%w", err)
		}
		patch.Name = body.Name
	}
	if body.Email != nil {
		if err := validate.Email(string(*body.Email)); err != nil {
			return patch, fmt.Errorf("%w", err)
		}
		s := string(*body.Email)
		patch.Email = &s
	}
	if body.Phone != nil {
		if err := validate.MaxLen(*body.Phone, 32, "phone"); err != nil {
			return patch, fmt.Errorf("%w", err)
		}
		patch.Phone = body.Phone
	}
	if body.Address != nil {
		if err := validate.MaxLen(*body.Address, 500, "address"); err != nil {
			return patch, fmt.Errorf("%w", err)
		}
		patch.Address = body.Address
	}
	if body.Birthday != nil {
		t := body.Birthday.Time
		patch.Birthday = &t
	}
	if body.Group != nil {
		if err := validate.MaxLen(*body.Group, 100, "group"); err != nil {
			return patch, fmt.Errorf("%w", err)
		}
		patch.Group = body.Group
	}
	return patch, nil
}

// UpdateMember updates member profile fields.
func (h *Handler) UpdateMember(ctx context.Context, request gen.UpdateMemberRequestObject) (gen.UpdateMemberResponseObject, error) {
	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}

	patch, err := validateMemberPatch(request.Body)
	if err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	m, err := h.svc.UpdateMember(ctx, request.MembershipId.String(), request.TeamId.String(), patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("member not found")
		}
		if errors.Is(err, ErrEmailTaken) {
			return nil, apierror.Conflict("email is already used by another account")
		}
		h.logger.ErrorContext(ctx, "UpdateMember failed", "err", err)
		return nil, fmt.Errorf("members.Handler.UpdateMember: %w", err)
	}
	h.audit.Record(ctx, audit.EventMemberUpdate, audit.Success, actor(ctx),
		slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()))
	metrics.TeamEvents.WithLabelValues("member", "update").Inc()
	return gen.UpdateMember200JSONResponse(*m), nil
}

// SetMemberRoles replaces the member's role assignments.
func (h *Handler) SetMemberRoles(ctx context.Context, request gen.SetMemberRolesRequestObject) (gen.SetMemberRolesResponseObject, error) {
	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.UUIDItems(len(request.Body.RoleIds), "roleIds"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	roleIDs := make([]string, len(request.Body.RoleIds))
	for i, u := range request.Body.RoleIds {
		roleIDs[i] = u.String()
	}

	m, err := h.svc.SetRoles(ctx, request.MembershipId.String(), request.TeamId.String(), roleIDs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("member not found")
		}
		if errors.Is(err, ErrRoleNotInTeam) {
			return nil, apierror.UnprocessableEntity("one or more roles do not belong to this team")
		}
		if errors.Is(err, ErrLastSettingsAdmin) {
			return nil, apierror.Conflict(ErrLastSettingsAdmin.Error())
		}
		h.logger.ErrorContext(ctx, "SetMemberRoles failed", "err", err)
		return nil, fmt.Errorf("members.Handler.SetMemberRoles: %w", err)
	}
	h.audit.Record(ctx, audit.EventMemberRolesChange, audit.Success, actor(ctx),
		slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()),
		slog.Int("roleCount", len(roleIDs)))
	metrics.TeamEvents.WithLabelValues("member", "update").Inc()
	return gen.SetMemberRoles200JSONResponse(*m), nil
}

// RemoveMember removes a member from the team.
func (h *Handler) RemoveMember(ctx context.Context, request gen.RemoveMemberRequestObject) (gen.RemoveMemberResponseObject, error) {
	if err := h.svc.RemoveMember(ctx, request.MembershipId.String(), request.TeamId.String()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("member not found")
		}
		if errors.Is(err, ErrLastSettingsAdmin) {
			return nil, apierror.Conflict(ErrLastSettingsAdmin.Error())
		}
		h.logger.ErrorContext(ctx, "RemoveMember failed", "err", err)
		return nil, fmt.Errorf("members.Handler.RemoveMember: %w", err)
	}
	h.audit.Record(ctx, audit.EventMemberRemove, audit.Success, actor(ctx),
		slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()))
	metrics.TeamEvents.WithLabelValues("member", "delete").Inc()
	return gen.RemoveMember204Response{}, nil
}

// ensure time is used.
var _ = time.Time{}
