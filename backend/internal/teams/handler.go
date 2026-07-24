package teams

import (
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
	CreateTeam(ctx context.Context, userID, name string, icon, iconBg, iconFg *string) (*gen.TeamForUser, error)
	GetTeam(ctx context.Context, teamID string) (*gen.Team, error)
	UpdateTeam(ctx context.Context, teamID string, patch TeamPatch) (*gen.Team, error)
	CreateInvite(ctx context.Context, teamID string) (*gen.Invite, error)
	AcceptInvite(ctx context.Context, code, userID string) (*gen.AcceptInviteResponse, error)
	GetTeamPhotoURL(ctx context.Context, teamID string) (string, error)
	GetTeamPhotoBytes(ctx context.Context, teamID string) (io.ReadCloser, string, error)
	UpdatePhoto(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error)
	DeletePhoto(ctx context.Context, teamID string) error
	GetTeamLogoURL(ctx context.Context, teamID string) (string, error)
	GetTeamLogoBytes(ctx context.Context, teamID string) (io.ReadCloser, string, error)
	UpdateLogo(ctx context.Context, teamID string, data []byte, mime string) (*gen.Team, error)
	DeleteLogo(ctx context.Context, teamID string) error
}

// Handler implements the team-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    teamService
	logger *slog.Logger
	audit  *audit.Logger
	// imageDeliveryProxyEnabled mirrors config.Config.ImageDeliveryProxyEnabled
	// (wired in by cmd/server/main.go via SetImageDeliveryProxyEnabled).
	// Defaults to false: GetTeamPhoto/GetTeamLogo redirect (302) to a
	// presigned object-store URL, unchanged from before this flag existed.
	imageDeliveryProxyEnabled bool
}

// NewHandler creates a new Handler. al is the shared audit logger; when nil a
// log-only logger is created from logger.
func NewHandler(svc teamService, logger *slog.Logger, al *audit.Logger) *Handler {
	if al == nil {
		al = audit.New(logger)
	}
	return &Handler{svc: svc, logger: logger, audit: al}
}

// SetImageDeliveryProxyEnabled configures whether GetTeamPhoto/GetTeamLogo
// stream image bytes directly through the backend (proxy mode) instead of
// redirecting to a presigned object-store URL (the default). See
// config.Config.ImageDeliveryProxyEnabled.
func (h *Handler) SetImageDeliveryProxyEnabled(enabled bool) {
	h.imageDeliveryProxyEnabled = enabled
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
	// Same bounds validateUpdateTeamBody applies to these fields on the PATCH
	// path -- CreateTeam must validate them too, since they reach the same
	// icon/icon_bg/icon_fg columns.
	for _, f := range []struct {
		val   *string
		field string
	}{
		{request.Body.Icon, "icon"},
		{request.Body.IconBg, "iconBg"},
		{request.Body.IconFg, "iconFg"},
	} {
		if f.val == nil {
			continue
		}
		if err := validate.MaxLen(*f.val, 50, f.field); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}

	tfu, err := h.svc.CreateTeam(ctx, user.Id.String(), request.Body.Name, request.Body.Icon, request.Body.IconBg, request.Body.IconFg)
	if err != nil {
		if errors.Is(err, ErrTooManyTeams) {
			return nil, apierror.UnprocessableEntity(err.Error())
		}
		h.logger.ErrorContext(ctx, "CreateTeam failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.CreateTeam: %w", err)
	}

	// CreateTeam mints a new Admin role with full write permissions and
	// assigns it to the caller -- more privilege than UpdateTeam/CreateInvite/
	// AcceptInvite grant, all three of which are already audited below.
	h.audit.Record(ctx, audit.EventTeamCreate, audit.Success, actor(ctx),
		slog.String("teamId", tfu.Id.String()))
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

// AcceptInvite redeems an invite code, adding the authenticated caller to the
// invite's team.
func (h *Handler) AcceptInvite(ctx context.Context, request gen.AcceptInviteRequestObject) (gen.AcceptInviteResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := validate.MaxLen(request.Code, 64, "code"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}

	tfu, err := h.svc.AcceptInvite(ctx, request.Code, user.Id.String())
	if err != nil {
		if errors.Is(err, ErrInviteNotFound) {
			return notFoundInviteResponse("invite not found or expired"), nil
		}
		h.logger.ErrorContext(ctx, "AcceptInvite failed", "err", err)
		return nil, fmt.Errorf("teams.Handler.AcceptInvite: %w", err)
	}
	h.audit.Record(ctx, audit.EventTeamInviteAccept, audit.Success, actor(ctx),
		slog.String("teamId", tfu.Id.String()))
	metrics.TeamEvents.WithLabelValues("team", "invite_accept").Inc()
	return gen.AcceptInvite200JSONResponse(*tfu), nil
}

// deliverImage implements the shared redirect-vs-proxy branching behind
// GetTeamPhoto and GetTeamLogo, which otherwise differ only in which service
// methods and generated response types they plug in -- factored out instead
// of duplicated so the mode switch (SetImageDeliveryProxyEnabled /
// config.Config.ImageDeliveryProxyEnabled) lives in exactly one place.
// notFound/bytesResponse/redirectResponse construct the operation-specific
// gen.Get*ResponseObject.
func deliverImage[R any](
	ctx context.Context,
	h *Handler,
	opName string,
	getBytes func(context.Context) (io.ReadCloser, string, error),
	getURL func(context.Context) (string, error),
	notFound func() R,
	bytesResponse func(data io.ReadCloser, contentType string) R,
	redirectResponse func(url string) R,
) (R, error) {
	var zero R
	if h.imageDeliveryProxyEnabled {
		data, contentType, err := getBytes(ctx)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return notFound(), nil
			}
			h.logger.ErrorContext(ctx, opName+" failed", "err", err)
			return zero, fmt.Errorf("teams.Handler.%s: %w", opName, err)
		}
		return bytesResponse(data, contentType), nil
	}

	url, err := getURL(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound(), nil
		}
		h.logger.ErrorContext(ctx, opName+" failed", "err", err)
		return zero, fmt.Errorf("teams.Handler.%s: %w", opName, err)
	}
	return redirectResponse(url), nil
}

// GetTeamPhoto returns the team photo: a short-lived presigned URL to
// redirect to (default), or the image bytes streamed directly when the
// deployment is configured for proxy image delivery
// (SetImageDeliveryProxyEnabled / config.Config.ImageDeliveryProxyEnabled).
//
// deliverImage call sites over their respective gen types) -- the shared
// logic already lives in deliverImage; this is the irreducible per-operation
// wiring, not copy-paste that should be merged further.
//
//nolint:dupl // structurally mirrors GetTeamLogo below (both are thin
func (h *Handler) GetTeamPhoto(ctx context.Context, request gen.GetTeamPhotoRequestObject) (gen.GetTeamPhotoResponseObject, error) {
	teamID := request.TeamId.String()
	return deliverImage(
		ctx, h, "GetTeamPhoto",
		func(ctx context.Context) (io.ReadCloser, string, error) { return h.svc.GetTeamPhotoBytes(ctx, teamID) },
		func(ctx context.Context) (string, error) { return h.svc.GetTeamPhotoURL(ctx, teamID) },
		func() gen.GetTeamPhotoResponseObject { return notFoundPhotoResponse("no team photo") },
		func(data io.ReadCloser, contentType string) gen.GetTeamPhotoResponseObject {
			return gen.GetTeamPhoto200ImageResponse{
				PhotoBytesImageResponse: gen.PhotoBytesImageResponse{Body: data, ContentType: contentType},
			}
		},
		func(url string) gen.GetTeamPhotoResponseObject {
			return gen.GetTeamPhoto302Response{Headers: gen.PhotoRedirectResponseHeaders{Location: &url}}
		},
	)
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

	const maxImageBytes = 2 << 20 // 2 MB max
	data, err = io.ReadAll(io.LimitReader(part, maxImageBytes+1))
	if err != nil {
		h.logger.WarnContext(ctx, label+": read file data failed", "err", err)
		return nil, "", apierror.BadRequest("cannot read file data")
	}
	// io.LimitReader silently truncates rather than erroring once the cap is
	// reached, so io.ReadAll alone can't distinguish "exactly at the limit"
	// from "oversized" -- reading one extra byte lets us detect the latter
	// and reject it explicitly (413) instead of letting the truncated data
	// fail image decoding downstream and fall through to a generic 500.
	if len(data) > maxImageBytes {
		return nil, "", apierror.New(http.StatusRequestEntityTooLarge, "Payload Too Large", "image exceeds the 2 MB upload limit")
	}

	// Detect MIME from actual content; reject anything other than JPEG/PNG.
	ct := http.DetectContentType(data)
	if ct != "image/jpeg" && ct != "image/png" {
		return nil, "", apierror.BadRequest("only JPEG and PNG images are accepted")
	}
	return data, ct, nil
}

// recordBrandingUpdate emits the audit record and metric shared by every
// successful photo/logo mutation (upload or delete).
func (h *Handler) recordBrandingUpdate(ctx context.Context, teamID, operation string) {
	h.audit.Record(ctx, audit.EventTeamBrandingUpdate, audit.Success, actor(ctx),
		slog.String("teamId", teamID), slog.String("operation", operation))
	metrics.TeamEvents.WithLabelValues("team", "update").Inc()
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
	h.recordBrandingUpdate(ctx, request.TeamId.String(), "photo.upload")
	return gen.UploadTeamPhoto200JSONResponse(*t), nil
}

// GetTeamLogo returns the team logo: a short-lived presigned URL to redirect
// to (default), or the image bytes streamed directly when the deployment is
// configured for proxy image delivery (SetImageDeliveryProxyEnabled /
// config.Config.ImageDeliveryProxyEnabled).
//
//nolint:dupl // structurally mirrors GetTeamPhoto above; see its comment.
func (h *Handler) GetTeamLogo(ctx context.Context, request gen.GetTeamLogoRequestObject) (gen.GetTeamLogoResponseObject, error) {
	teamID := request.TeamId.String()
	return deliverImage(
		ctx, h, "GetTeamLogo",
		func(ctx context.Context) (io.ReadCloser, string, error) { return h.svc.GetTeamLogoBytes(ctx, teamID) },
		func(ctx context.Context) (string, error) { return h.svc.GetTeamLogoURL(ctx, teamID) },
		func() gen.GetTeamLogoResponseObject { return notFoundLogoResponse("no team logo") },
		func(data io.ReadCloser, contentType string) gen.GetTeamLogoResponseObject {
			return gen.GetTeamLogo200ImageResponse{
				PhotoBytesImageResponse: gen.PhotoBytesImageResponse{Body: data, ContentType: contentType},
			}
		},
		func(url string) gen.GetTeamLogoResponseObject {
			return gen.GetTeamLogo302Response{Headers: gen.PhotoRedirectResponseHeaders{Location: &url}}
		},
	)
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
	h.recordBrandingUpdate(ctx, request.TeamId.String(), "logo.upload")
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
	h.recordBrandingUpdate(ctx, request.TeamId.String(), "photo.delete")
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
	h.recordBrandingUpdate(ctx, request.TeamId.String(), "logo.delete")
	return gen.DeleteTeamLogo204Response{}, nil
}

// ─── error helpers ───────────────────────────────────────────────────────────

func notFoundTeamResponse(detail string) gen.GetTeamResponseObject {
	e := apierror.NotFound(detail)
	return gen.GetTeam404ApplicationProblemPlusJSONResponse{
		NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
			Title:  &e.Title,
			Detail: &detail,
			Status: &e.Status,
			Type:   &e.Type,
		},
	}
}

func notFoundInviteResponse(detail string) gen.AcceptInviteResponseObject {
	e := apierror.NotFound(detail)
	return gen.AcceptInvite404ApplicationProblemPlusJSONResponse{
		NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
			Title:  &e.Title,
			Detail: &detail,
			Status: &e.Status,
			Type:   &e.Type,
		},
	}
}

func notFoundPhotoResponse(detail string) gen.GetTeamPhotoResponseObject {
	e := apierror.NotFound(detail)
	return gen.GetTeamPhoto404ApplicationProblemPlusJSONResponse{
		NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
			Title:  &e.Title,
			Detail: &detail,
			Status: &e.Status,
			Type:   &e.Type,
		},
	}
}

func notFoundLogoResponse(detail string) gen.GetTeamLogoResponseObject {
	e := apierror.NotFound(detail)
	return gen.GetTeamLogo404ApplicationProblemPlusJSONResponse{
		NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
			Title:  &e.Title,
			Detail: &detail,
			Status: &e.Status,
			Type:   &e.Type,
		},
	}
}
