package members

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
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
	GetMemberPhotoURL(ctx context.Context, teamID, membershipID string) (string, error)
	UpdateMember(ctx context.Context, membershipID, teamID, callerUserID string, patch MemberPatch) (*gen.Member, error)
	SetRoles(ctx context.Context, membershipID, teamID string, roleIDs []string, callerUserID string) (*gen.Member, error)
	RemoveMember(ctx context.Context, membershipID, teamID, callerUserID string) error
}

// Handler implements the member-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    memberService
	logger *slog.Logger
	audit  *audit.Logger
}

// NewHandler creates a new Handler. al is the shared audit logger; when nil a
// log-only logger is created from logger.
func NewHandler(svc memberService, logger *slog.Logger, al *audit.Logger) *Handler {
	if al == nil {
		al = audit.New(logger)
	}
	return &Handler{svc: svc, logger: logger, audit: al}
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

// GetMemberPhoto redirects to a short-lived presigned URL for the member's
// profile photo.
func (h *Handler) GetMemberPhoto(ctx context.Context, request gen.GetMemberPhotoRequestObject) (gen.GetMemberPhotoResponseObject, error) {
	url, err := h.svc.GetMemberPhotoURL(ctx, request.TeamId.String(), request.MembershipId.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("no profile photo")
		}
		h.logger.ErrorContext(ctx, "GetMemberPhoto failed", "err", err)
		return nil, fmt.Errorf("members.Handler.GetMemberPhoto: %w", err)
	}
	return gen.GetMemberPhoto302Response{
		Headers: gen.PhotoRedirectResponseHeaders{Location: &url},
	}, nil
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
		// Normalized to lowercase before it ever reaches the DB's UNIQUE
		// constraint on users.email, which is case-sensitive -- without
		// this, Bob@Example.com and bob@example.com collide on every real
		// mail provider but not on this constraint, so ErrEmailTaken would
		// never fire and the app would end up with two accounts for what is
		// really one address. Login (auth.Repository.FindUserByEmail) also
		// normalizes its lookup key to match.
		s := strings.ToLower(strings.TrimSpace(string(*body.Email)))
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
		if err := validate.Birthday(t); err != nil {
			return patch, fmt.Errorf("birthday %w", err)
		}
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
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}

	patch, err := validateMemberPatch(request.Body)
	if err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	m, err := h.svc.UpdateMember(ctx, request.MembershipId.String(), request.TeamId.String(), user.Id.String(), patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("member not found")
		}
		if errors.Is(err, ErrCannotChangeOthersEmail) {
			h.audit.Record(ctx, audit.EventMemberUpdate, audit.Failure, actor(ctx),
				slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()),
				slog.String("reason", "cannot_change_others_email"))
			return nil, apierror.Forbidden(ErrCannotChangeOthersEmail.Error())
		}
		if errors.Is(err, ErrEmailTaken) {
			// users.email is a global (not per-team) UNIQUE constraint. A
			// caller only needs members:write on ANY team (trivially
			// obtained by creating one) to submit this patch, so a
			// distinguishable response here -- even with generic wording --
			// would let them probe whether an arbitrary, unrelated email
			// address belongs to a registered account anywhere on the
			// platform, independent of any relationship to that account.
			// Reusing validate.Email's exact status/message makes a
			// well-formed-but-taken address structurally indistinguishable
			// from a malformed one from the client's perspective, closing
			// that oracle rather than just softening its wording.
			return nil, apierror.BadRequest(validate.ErrEmailInvalid.Error())
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
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
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

	m, err := h.svc.SetRoles(ctx, request.MembershipId.String(), request.TeamId.String(), roleIDs, user.Id.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("member not found")
		}
		if errors.Is(err, ErrRoleNotInTeam) {
			return nil, apierror.UnprocessableEntity("one or more roles do not belong to this team")
		}
		if errors.Is(err, ErrLastSettingsAdmin) {
			h.audit.Record(ctx, audit.EventMemberRolesChange, audit.Failure, actor(ctx),
				slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()),
				slog.String("reason", "last_settings_admin"))
			return nil, apierror.Conflict(ErrLastSettingsAdmin.Error())
		}
		if errors.Is(err, ErrInsufficientPermissionToGrant) {
			h.audit.Record(ctx, audit.EventMemberRolesChange, audit.Failure, actor(ctx),
				slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()),
				slog.String("reason", "insufficient_permission_to_grant"))
			return nil, apierror.Forbidden(ErrInsufficientPermissionToGrant.Error())
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
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.RemoveMember(ctx, request.MembershipId.String(), request.TeamId.String(), user.Id.String()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("member not found")
		}
		if errors.Is(err, ErrLastSettingsAdmin) {
			h.audit.Record(ctx, audit.EventMemberRemove, audit.Failure, actor(ctx),
				slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()),
				slog.String("reason", "last_settings_admin"))
			return nil, apierror.Conflict(ErrLastSettingsAdmin.Error())
		}
		if errors.Is(err, ErrCannotRemoveSettingsAdmin) {
			h.audit.Record(ctx, audit.EventMemberRemove, audit.Failure, actor(ctx),
				slog.String("teamId", request.TeamId.String()), slog.String("membershipId", request.MembershipId.String()),
				slog.String("reason", "cannot_remove_settings_admin"))
			return nil, apierror.Forbidden(ErrCannotRemoveSettingsAdmin.Error())
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
