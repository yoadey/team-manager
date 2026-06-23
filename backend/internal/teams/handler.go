package teams

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
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
}

// Handler implements the team-related methods of gen.StrictServerInterface.
type Handler struct {
	svc    teamService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc teamService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// ListTeams returns all teams the current user belongs to.
func (h *Handler) ListTeams(ctx context.Context, _ gen.ListTeamsRequestObject) (gen.ListTeamsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("teams.Handler.ListTeams: not authenticated")
	}

	teams, err := h.svc.ListForUser(ctx, user.Id.String())
	if err != nil {
		h.logger.ErrorContext(ctx, "ListTeams failed", "err", err)
		return nil, err
	}

	return gen.ListTeams200JSONResponse(teams), nil
}

// CreateTeam creates a new team for the current user.
func (h *Handler) CreateTeam(ctx context.Context, request gen.CreateTeamRequestObject) (gen.CreateTeamResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("teams.Handler.CreateTeam: not authenticated")
	}
	if request.Body == nil {
		return nil, fmt.Errorf("teams.Handler.CreateTeam: missing body")
	}

	tfu, err := h.svc.CreateTeam(ctx, user.Id.String(), request.Body.Name)
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateTeam failed", "err", err)
		return nil, err
	}

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
		return nil, err
	}
	return gen.GetTeam200JSONResponse(*t), nil
}

// UpdateTeam applies a patch to the team.
func (h *Handler) UpdateTeam(ctx context.Context, request gen.UpdateTeamRequestObject) (gen.UpdateTeamResponseObject, error) {
	if request.Body == nil {
		return nil, fmt.Errorf("teams.Handler.UpdateTeam: missing body")
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
			ids[i] = openapi_types.UUID(u).String()
		}
		patch.ReasonVisibilityRoleIDs = ids
	}

	t, err := h.svc.UpdateTeam(ctx, request.TeamId.String(), patch)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("team not found")
		}
		h.logger.ErrorContext(ctx, "UpdateTeam failed", "err", err)
		return nil, err
	}
	return gen.UpdateTeam200JSONResponse(*t), nil
}

// CreateInvite generates a 7-day invite link for the team.
func (h *Handler) CreateInvite(ctx context.Context, request gen.CreateInviteRequestObject) (gen.CreateInviteResponseObject, error) {
	inv, err := h.svc.CreateInvite(ctx, request.TeamId.String())
	if err != nil {
		h.logger.ErrorContext(ctx, "CreateInvite failed", "err", err)
		return nil, err
	}
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
		return nil, err
	}
	return gen.GetTeamPhoto200ImagejpegResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
	}, nil
}

// UploadTeamPhoto handles a multipart upload, stores the photo, and returns the updated team.
func (h *Handler) UploadTeamPhoto(ctx context.Context, request gen.UploadTeamPhotoRequestObject) (gen.UploadTeamPhotoResponseObject, error) {
	if request.Body == nil {
		return nil, fmt.Errorf("teams.Handler.UploadTeamPhoto: missing multipart body")
	}

	part, err := request.Body.NextPart()
	if err != nil {
		return nil, fmt.Errorf("teams.Handler.UploadTeamPhoto: read multipart: %w", err)
	}
	defer part.Close()

	data, err := io.ReadAll(part)
	if err != nil {
		return nil, fmt.Errorf("teams.Handler.UploadTeamPhoto: read file data: %w", err)
	}

	ct := part.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}

	t, err := h.svc.UpdatePhoto(ctx, request.TeamId.String(), data, ct)
	if err != nil {
		h.logger.ErrorContext(ctx, "UploadTeamPhoto failed", "err", err)
		return nil, err
	}
	return gen.UploadTeamPhoto200JSONResponse(*t), nil
}

// GetTeamLogo returns the team logo as JPEG.
func (h *Handler) GetTeamLogo(ctx context.Context, request gen.GetTeamLogoRequestObject) (gen.GetTeamLogoResponseObject, error) {
	title := "Not Found"
	detail := "no team logo"
	status := 404
	return gen.GetTeamLogo404ApplicationProblemPlusJSONResponse{
		NotFoundApplicationProblemPlusJSONResponse: gen.NotFoundApplicationProblemPlusJSONResponse{
			Title:  &title,
			Detail: &detail,
			Status: &status,
		},
	}, nil
}

// UploadTeamLogo handles a multipart upload and stores the logo for the team.
func (h *Handler) UploadTeamLogo(ctx context.Context, request gen.UploadTeamLogoRequestObject) (gen.UploadTeamLogoResponseObject, error) {
	return nil, fmt.Errorf("teams.Handler.UploadTeamLogo: not implemented")
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
