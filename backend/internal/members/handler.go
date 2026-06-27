package members

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/pagination"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// memberService is the interface the Handler relies on.
type memberService interface {
	ListMembers(ctx context.Context, teamID string, limit int, cursor string) ([]gen.Member, *string, error)
	AddMember(ctx context.Context, teamID string, params AddMemberParams) (*gen.Member, error)
	UpdateMember(ctx context.Context, membershipID string, patch MemberPatch) (*gen.Member, error)
	SetRoles(ctx context.Context, membershipID string, roleIDs []string) (*gen.Member, error)
	RemoveMember(ctx context.Context, membershipID string) error
}

// Handler implements the member-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    memberService
	logger *slog.Logger
	audit  *audit.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc memberService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger, audit: audit.New(logger)}
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

// AddMember adds a new member to the team.
func (h *Handler) AddMember(ctx context.Context, request gen.AddMemberRequestObject) (gen.AddMemberResponseObject, error) {
	if _, ok := auth.UserFromContext(ctx); !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.Name(request.Body.Name); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if err := validate.Email(string(request.Body.Email)); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	params := AddMemberParams{
		Name:  request.Body.Name,
		Email: string(request.Body.Email),
		Phone: request.Body.Phone,
		Group: request.Body.Group,
	}
	if request.Body.RoleIds != nil {
		for _, u := range *request.Body.RoleIds {
			params.RoleIDs = append(params.RoleIDs, u.String())
		}
	}

	m, err := h.svc.AddMember(ctx, request.TeamId.String(), params)
	if err != nil {
		h.logger.ErrorContext(ctx, "AddMember failed", "err", err)
		return nil, fmt.Errorf("members.Handler.AddMember: %w", err)
	}
	h.audit.Record(ctx, audit.EventMemberAdd, audit.Success, actor(ctx),
		slog.String("teamId", request.TeamId.String()), slog.String("membershipId", m.MembershipId.String()))
	metrics.TeamEvents.WithLabelValues("member", "create").Inc()
	return gen.AddMember201JSONResponse(*m), nil
}

// UpdateMember updates member profile fields.
func (h *Handler) UpdateMember(ctx context.Context, request gen.UpdateMemberRequestObject) (gen.UpdateMemberResponseObject, error) {
	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}

	patch := MemberPatch{}
	if request.Body.Name != nil {
		patch.Name = request.Body.Name
	}
	if request.Body.Email != nil {
		s := string(*request.Body.Email)
		patch.Email = &s
	}
	if request.Body.Phone != nil {
		patch.Phone = request.Body.Phone
	}
	if request.Body.Address != nil {
		patch.Address = request.Body.Address
	}
	if request.Body.Birthday != nil {
		t := request.Body.Birthday.Time
		patch.Birthday = &t
	}
	if request.Body.Group != nil {
		patch.Group = request.Body.Group
	}

	m, err := h.svc.UpdateMember(ctx, request.MembershipId.String(), patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("member not found")
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

	roleIDs := make([]string, len(request.Body.RoleIds))
	for i, u := range request.Body.RoleIds {
		roleIDs[i] = u.String()
	}

	m, err := h.svc.SetRoles(ctx, request.MembershipId.String(), roleIDs)
	if err != nil {
		h.logger.ErrorContext(ctx, "SetMemberRoles failed", "err", err)
		return nil, fmt.Errorf("members.Handler.SetMemberRoles: %w", err)
	}
	h.audit.Record(ctx, audit.EventMemberRolesChange, audit.Success, actor(ctx),
		slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()),
		slog.Int("roleCount", len(roleIDs)))
	return gen.SetMemberRoles200JSONResponse(*m), nil
}

// RemoveMember removes a member from the team.
func (h *Handler) RemoveMember(ctx context.Context, request gen.RemoveMemberRequestObject) (gen.RemoveMemberResponseObject, error) {
	if err := h.svc.RemoveMember(ctx, request.MembershipId.String()); err != nil {
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
