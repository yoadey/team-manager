package teams

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// teamService is the interface the Handler relies on.
type teamService interface {
	ListForUser(ctx context.Context, userID string) ([]gen.TeamForUser, error)
	CreateTeam(ctx context.Context, userID, name string) (*gen.TeamForUser, error)
	GetTeam(ctx context.Context, teamID string) (*gen.Team, error)
	UpdateTeam(ctx context.Context, teamID string, patch TeamPatch) (*gen.Team, error)
	CreateInvite(ctx context.Context, teamID string) (*gen.Invite, error)
	GetTeamPhotoData(ctx context.Context, teamID string) ([]byte, string, error)
	UpdatePhoto(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error)
	DeletePhoto(ctx context.Context, teamID string) error
	GetTeamLogoData(ctx context.Context, teamID string) ([]byte, string, error)
	UpdateLogo(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error)
	DeleteLogo(ctx context.Context, teamID string) error
}

// Handler implements the team-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    teamService
	logger *slog.Logger
	audit  *audit.Logger
}

// NewHandler creates a new Handler. al is the shared audit logger; when nil a
// log-only logger is created from logger.
func NewHandler(svc teamService, logger *slog.Logger, al *audit.Logger) *Handler {
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

// ListTeams returns all teams the current user belongs to.
func (h *Handler) ListTeams(ctx context.Context, _ gen.ListTeamsRequestObject) (gen.ListTeamsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}

	teams, err := h.svc.ListForUser(ctx, user.Id.String())
	if err != nil {
		h.logger.ErrorContext(ctx, "ListTeams failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.ListTeams: %w", err)
	}

	return gen.ListTeams200JSONResponse(teams), nil
}

// CreateTeam creates a new team for the current user.
func (h *Handler) CreateTeam(ctx context.Context, request gen.CreateTeamRequestObject) (gen.CreateTeamResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.Name(request.Body.Name); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	tfu, err := h.svc.CreateTeam(ctx, user.Id.String(), request.Body.Name)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateTeam failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.CreateTeam: %w", err)
	}

	metrics.TeamEvents.WithLabelValues("team", "create").Inc()
	return gen.CreateTeam201JSONResponse(*tfu), nil
}

// GetTeam returns a single team by ID.
func (h *Handler) GetTeam(ctx context.Context, request gen.GetTeamRequestObject) (gen.GetTeamResponseObject, error) {
	t, err := h.svc.GetTeam(ctx, request.TeamId.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFoundTeamResponse("team not found"), nil
		}
		h.logger.ErrorContext(ctx, "GetTeam failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.GetTeam: %w", err)
	}
	return gen.GetTeam200JSONResponse(*t), nil
}

// validateUpdateTeamBody validates the optional fields of an UpdateTeam
// request. CreateTeam validates Name via validate.Name; UpdateTeam must do
// the same for every field it can patch, since a PATCH could otherwise set an
// empty or unbounded name/short/icon/description.
func validateUpdateTeamBody(body *gen.UpdateTeamRequest) error {
	if body.Name != nil {
		if err := validate.Name(*body.Name); err != nil {
			return fmt.Errorf("name: %w", err)
		}
	}
	fields := []struct {
		val   *string
		max   int
		field string
	}{
		{body.Short, 50, "short"},
		{body.Icon, 50, "icon"},
		{body.IconBg, 50, "iconBg"},
		{body.IconFg, 50, "iconFg"},
		{body.Description, 10_000, "description"},
	}
	for _, f := range fields {
		if f.val == nil {
			continue
		}
		if err := validate.MaxLen(*f.val, f.max, f.field); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	if body.ReasonVisibilityRoleIds != nil {
		if err := validate.UUIDItems(len(*body.ReasonVisibilityRoleIds), "reasonVisibilityRoleIds"); err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	return nil
}

// UpdateTeam applies a patch to the team.
func (h *Handler) UpdateTeam(ctx context.Context, request gen.UpdateTeamRequestObject) (gen.UpdateTeamResponseObject, error) {
	if request.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validateUpdateTeamBody(request.Body); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	patch := TeamPatch{}
	if request.Body.Name != nil {
		patch.Name = request.Body.Name
	}
	if request.Body.Short != nil {
		patch.Short = request.Body.Short
	}
	if request.Body.Icon != nil {
		patch.Icon = request.Body.Icon
	}
	if request.Body.IconBg != nil {
		patch.IconBg = request.Body.IconBg
	}
	if request.Body.IconFg != nil {
		patch.IconFg = request.Body.IconFg
	}
	if request.Body.Description != nil {
		patch.Description = request.Body.Description
	}
	if request.Body.ReasonVisibilityRoleIds != nil {
		ids := make([]string, len(*request.Body.ReasonVisibilityRoleIds))
		for i, u := range *request.Body.ReasonVisibilityRoleIds {
			ids[i] = u.String()
		}
		patch.ReasonVisibilityRoleIDs = ids
	}

	t, err := h.svc.UpdateTeam(ctx, request.TeamId.String(), patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("team not found")
		}
		if errors.Is(err, ErrRoleNotInTeam) {
			return nil, apierror.UnprocessableEntity("one or more roles do not belong to this team")
		}
		h.logger.ErrorContext(ctx, "UpdateTeam failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.UpdateTeam: %w", err)
	}
	h.audit.Record(ctx, audit.EventTeamUpdate, audit.Success, actor(ctx),
		slog.String("teamId", request.TeamId.String()))
	metrics.TeamEvents.WithLabelValues("team", "update").Inc()
	return gen.UpdateTeam200JSONResponse(*t), nil
}

// CreateInvite generates a 7-day invite link for the team.
func (h *Handler) CreateInvite(ctx context.Context, request gen.CreateInviteRequestObject) (gen.CreateInviteResponseObject, error) {
	inv, err := h.svc.CreateInvite(ctx, request.TeamId.String())
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateInvite failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.CreateInvite: %w", err)
	}
	h.audit.Record(ctx, audit.EventTeamInvite, audit.Success, actor(ctx),
		slog.String("teamId", request.TeamId.String()), slog.String("inviteId", inv.Id.String()))
	metrics.TeamEvents.WithLabelValues("team", "invite").Inc()
	return gen.CreateInvite201JSONResponse(*inv), nil
}

// GetTeamPhoto returns the team photo as JPEG.
func (h *Handler) GetTeamPhoto(ctx context.Context, request gen.GetTeamPhotoRequestObject) (gen.GetTeamPhotoResponseObject, error) {
	data, _, err := h.svc.GetTeamPhotoData(ctx, request.TeamId.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFoundPhotoResponse("no team photo"), nil
		}
		h.logger.ErrorContext(ctx, "GetTeamPhoto failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.GetTeamPhoto: %w", err)
	}
	return gen.GetTeamPhoto200ImagejpegResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
	}, nil
}

// readMultipartImage reads the first part of a multipart body, capped at 2 MB,
// and validates it is a JPEG or PNG by sniffing the actual content (not the
// client-supplied Content-Type). label is used only for log messages (e.g.
// "UploadTeamPhoto"). Shared by UploadTeamPhoto and UploadTeamLogo, which
// otherwise differ only in which service method they call afterward.
func (h *Handler) readMultipartImage(ctx context.Context, body *multipart.Reader, label string) (data []byte, contentType string, err error) {
	if body == nil {
		return nil, "", apierror.BadRequest("missing multipart body")
	}

	part, err := body.NextPart()
	if err != nil {
		h.logger.WarnContext(ctx, label+": read multipart failed", "err", err)
		return nil, "", apierror.BadRequest("cannot read multipart body")
	}
	defer func() {
		if cerr := part.Close(); cerr != nil {
			h.logger.WarnContext(ctx, "part.Close failed", "err", cerr)
		}
	}()

	data, err = io.ReadAll(io.LimitReader(part, 2<<20)) // 2 MB max
	if err != nil {
		h.logger.WarnContext(ctx, label+": read file data failed", "err", err)
		return nil, "", apierror.BadRequest("cannot read file data")
	}

	// Detect MIME from actual content; reject anything other than JPEG/PNG.
	ct := http.DetectContentType(data)
	if ct != "image/jpeg" && ct != "image/png" {
		return nil, "", apierror.BadRequest("only JPEG and PNG images are accepted")
	}
	return data, ct, nil
}

// UploadTeamPhoto handles a multipart upload, stores the photo, and returns the updated team.
func (h *Handler) UploadTeamPhoto(ctx context.Context, request gen.UploadTeamPhotoRequestObject) (gen.UploadTeamPhotoResponseObject, error) {
	data, ct, err := h.readMultipartImage(ctx, request.Body, "UploadTeamPhoto")
	if err != nil {
		return nil, err
	}

	t, err := h.svc.UpdatePhoto(ctx, request.TeamId.String(), data, ct)
	if err != nil {
		if errors.Is(err, ErrImageTooLarge) {
			return nil, apierror.BadRequest("image dimensions exceed the allowed maximum")
		}
		h.logger.ErrorContext(ctx, "UploadTeamPhoto failed", "err", err)
		return nil, apierror.Internal("photo update failed")
	}
	metrics.TeamEvents.WithLabelValues("team", "update").Inc()
	return gen.UploadTeamPhoto200JSONResponse(*t), nil
}

// GetTeamLogo returns the team logo as JPEG.
func (h *Handler) GetTeamLogo(ctx context.Context, request gen.GetTeamLogoRequestObject) (gen.GetTeamLogoResponseObject, error) {
	data, _, err := h.svc.GetTeamLogoData(ctx, request.TeamId.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFoundLogoResponse("no team logo"), nil
		}
		h.logger.ErrorContext(ctx, "GetTeamLogo failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.GetTeamLogo: %w", err)
	}
	return gen.GetTeamLogo200ImagejpegResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
	}, nil
}

// UploadTeamLogo handles a multipart upload, stores the logo, and returns the updated team.
func (h *Handler) UploadTeamLogo(ctx context.Context, request gen.UploadTeamLogoRequestObject) (gen.UploadTeamLogoResponseObject, error) {
	data, ct, err := h.readMultipartImage(ctx, request.Body, "UploadTeamLogo")
	if err != nil {
		return nil, err
	}

	t, err := h.svc.UpdateLogo(ctx, request.TeamId.String(), data, ct)
	if err != nil {
		if errors.Is(err, ErrImageTooLarge) {
			return nil, apierror.BadRequest("image dimensions exceed the allowed maximum")
		}
		h.logger.ErrorContext(ctx, "UploadTeamLogo failed", "err", err)
		return nil, apierror.Internal("logo update failed")
	}
	metrics.TeamEvents.WithLabelValues("team", "update").Inc()
	return gen.UploadTeamLogo200JSONResponse(*t), nil
}

// DeleteTeamPhoto removes the team photo, reverting display to the icon fallback.
func (h *Handler) DeleteTeamPhoto(ctx context.Context, request gen.DeleteTeamPhotoRequestObject) (gen.DeleteTeamPhotoResponseObject, error) {
	if err := h.svc.DeletePhoto(ctx, request.TeamId.String()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("team not found")
		}
		h.logger.ErrorContext(ctx, "DeleteTeamPhoto failed", "err", err)
		return nil, apierror.Internal("photo removal failed")
	}
	metrics.TeamEvents.WithLabelValues("team", "update").Inc()
	return gen.DeleteTeamPhoto204Response{}, nil
}

// DeleteTeamLogo removes the team logo, reverting display to the icon fallback.
func (h *Handler) DeleteTeamLogo(ctx context.Context, request gen.DeleteTeamLogoRequestObject) (gen.DeleteTeamLogoResponseObject, error) {
	if err := h.svc.DeleteLogo(ctx, request.TeamId.String()); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("team not found")
		}
		h.logger.ErrorContext(ctx, "DeleteTeamLogo failed", "err", err)
		return nil, apierror.Internal("logo removal failed")
	}
	metrics.TeamEvents.WithLabelValues("team", "update").Inc()
	return gen.DeleteTeamLogo204Response{}, nil
}

// ─── error helpers ───────────────────────────────────────────────────────────

func notFoundTeamResponse(detail string) gen.GetTeamResponseObject {
	title := "Not Found"
	status := 404
	return gen.GetTeam404ApplicationProblemPlusJSONResponse{
		NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
			Title:  &title,
			Detail: &detail,
			Status: &status,
		},
	}
}

func notFoundPhotoResponse(detail string) gen.GetTeamPhotoResponseObject {
	title := "Not Found"
	status := 404
	return gen.GetTeamPhoto404ApplicationProblemPlusJSONResponse{
		NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
			Title:  &title,
			Detail: &detail,
			Status: &status,
		},
	}
}

func notFoundLogoResponse(detail string) gen.GetTeamLogoResponseObject {
	title := "Not Found"
	status := 404
	return gen.GetTeamLogo404ApplicationProblemPlusJSONResponse{
		NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
			Title:  &title,
			Detail: &detail,
			Status: &status,
		},
	}
}
