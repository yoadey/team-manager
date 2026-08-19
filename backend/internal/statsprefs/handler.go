package statsprefs

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/validate"
)

// maxPresetNameLen mirrors CreateStatsPresetRequest/UpdateStatsPresetRequest's
// `maxLength: 100` in openapi.yaml -- there's no request-validation
// middleware in this codebase (see e.g. absences/handler.go's identical
// validate.MaxLen calls), so the OpenAPI-declared bound must be re-enforced
// here to actually take effect.
const maxPresetNameLen = 100

// statsprefsService is the interface the Handler relies on.
type statsprefsService interface {
	GetLastSelection(ctx context.Context, teamID, userID uuid.UUID) (LastSelection, error)
	SetLastSelection(ctx context.Context, teamID, userID uuid.UUID, sel LastSelection) error
	ListPresets(ctx context.Context, teamID, userID uuid.UUID) ([]Preset, error)
	CreatePreset(ctx context.Context, teamID, userID uuid.UUID, name string, from, to time.Time) (Preset, error)
	UpdatePreset(ctx context.Context, teamID, userID, presetID uuid.UUID, name *string, from, to *time.Time) (Preset, error)
	DeletePreset(ctx context.Context, teamID, userID, presetID uuid.UUID) error
}

// Handler implements the stats-preferences/stats-presets methods of
// gen.StrictServerInterface. Every operation here is self-service and
// x-rbac-module: public -- the caller always acts on their own selection/
// presets, identified from the auth context, never a path/body user id.
type Handler struct {
	svc    statsprefsService
	logger *slog.Logger
}

// NewHandler creates a new Handler.
func NewHandler(svc statsprefsService, logger *slog.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// GetStatsPreferences returns the caller's last-selected statistics date
// range for teamId.
func (h *Handler) GetStatsPreferences(ctx context.Context, req gen.GetStatsPreferencesRequestObject) (gen.GetStatsPreferencesResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	sel, err := h.svc.GetLastSelection(ctx, req.TeamId, user.Id)
	if err != nil {
		h.logger.ErrorContext(ctx, "GetStatsPreferences failed", "err", err)
		return nil, apierror.Internal("failed to get stats preferences")
	}
	return gen.GetStatsPreferences200JSONResponse(toGenPreferences(sel)), nil
}

// SetStatsPreferences saves the caller's last-selected statistics date range
// for teamId.
func (h *Handler) SetStatsPreferences(ctx context.Context, req gen.SetStatsPreferencesRequestObject) (gen.SetStatsPreferencesResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if req.Body.From.After(req.Body.To.Time) {
		return nil, apierror.BadRequest("'from' must not be after 'to'")
	}
	from := req.Body.From.Time
	to := req.Body.To.Time
	sel := LastSelection{FromDate: &from, ToDate: &to, PresetID: req.Body.PresetId}
	if err := h.svc.SetLastSelection(ctx, req.TeamId, user.Id, sel); err != nil {
		if errors.Is(err, ErrPresetNotFound) {
			return nil, apierror.BadRequest("presetId does not reference a preset you own in this team")
		}
		h.logger.ErrorContext(ctx, "SetStatsPreferences failed", "err", err)
		return nil, apierror.Internal("failed to set stats preferences")
	}
	return gen.SetStatsPreferences204Response{}, nil
}

// ListStatsPresets lists the caller's saved statistics date-range presets
// for teamId.
func (h *Handler) ListStatsPresets(ctx context.Context, req gen.ListStatsPresetsRequestObject) (gen.ListStatsPresetsResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	presets, err := h.svc.ListPresets(ctx, req.TeamId, user.Id)
	if err != nil {
		h.logger.ErrorContext(ctx, "ListStatsPresets failed", "err", err)
		return nil, apierror.Internal("failed to list stats presets")
	}
	items := make([]gen.StatsPreset, 0, len(presets))
	for _, p := range presets {
		items = append(items, toGenPreset(p))
	}
	return gen.ListStatsPresets200JSONResponse{Items: items}, nil
}

// CreateStatsPreset saves a new named statistics date-range preset for the
// caller in teamId.
func (h *Handler) CreateStatsPreset(ctx context.Context, req gen.CreateStatsPresetRequestObject) (gen.CreateStatsPresetResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if err := validate.RequireNonEmpty(req.Body.Name, "name"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if err := validate.MaxLen(req.Body.Name, maxPresetNameLen, "name"); err != nil {
		return nil, apierror.BadRequest(err.Error())
	}
	if req.Body.From.After(req.Body.To.Time) {
		return nil, apierror.BadRequest("'from' must not be after 'to'")
	}
	p, err := h.svc.CreatePreset(ctx, req.TeamId, user.Id, req.Body.Name, req.Body.From.Time, req.Body.To.Time)
	if err != nil {
		if errors.Is(err, ErrTooManyPresets) {
			return nil, apierror.BadRequest(err.Error())
		}
		h.logger.ErrorContext(ctx, "CreateStatsPreset failed", "err", err)
		return nil, apierror.Internal("failed to create stats preset")
	}
	return gen.CreateStatsPreset201JSONResponse(toGenPreset(p)), nil
}

// UpdateStatsPreset renames or reschedules a preset the caller owns.
func (h *Handler) UpdateStatsPreset(ctx context.Context, req gen.UpdateStatsPresetRequestObject) (gen.UpdateStatsPresetResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if req.Body == nil {
		return nil, apierror.BadRequest("missing request body")
	}
	if req.Body.Name != nil {
		if err := validate.RequireNonEmpty(*req.Body.Name, "name"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
		if err := validate.MaxLen(*req.Body.Name, maxPresetNameLen, "name"); err != nil {
			return nil, apierror.BadRequest(err.Error())
		}
	}
	var from, to *time.Time
	if req.Body.From != nil {
		from = &req.Body.From.Time
	}
	if req.Body.To != nil {
		to = &req.Body.To.Time
	}
	// Ordering is only checkable here (no extra DB read) when both bounds
	// arrive in the same PATCH -- mirrors absences/handler.go's UpdateAbsence
	// doing the identical both-fields-present-only check for the same reason.
	if from != nil && to != nil && from.After(*to) {
		return nil, apierror.BadRequest("'from' must not be after 'to'")
	}
	p, err := h.svc.UpdatePreset(ctx, req.TeamId, user.Id, req.PresetId, req.Body.Name, from, to)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.NotFound("preset not found")
		}
		if errors.Is(err, ErrInvalidDateRange) {
			return nil, apierror.BadRequest("'from' must not be after 'to'")
		}
		h.logger.ErrorContext(ctx, "UpdateStatsPreset failed", "err", err)
		return nil, apierror.Internal("failed to update stats preset")
	}
	return gen.UpdateStatsPreset200JSONResponse(toGenPreset(p)), nil
}

// DeleteStatsPreset deletes a preset the caller owns. Idempotent: deleting
// an already-gone or foreign preset id still returns 204.
func (h *Handler) DeleteStatsPreset(ctx context.Context, req gen.DeleteStatsPresetRequestObject) (gen.DeleteStatsPresetResponseObject, error) {
	user, ok := auth.UserFromContext(ctx)
	if !ok {
		return nil, apierror.Unauthorized("not authenticated")
	}
	if err := h.svc.DeletePreset(ctx, req.TeamId, user.Id, req.PresetId); err != nil {
		h.logger.ErrorContext(ctx, "DeleteStatsPreset failed", "err", err)
		return nil, apierror.Internal("failed to delete stats preset")
	}
	return gen.DeleteStatsPreset204Response{}, nil
}

// toGenPreferences maps a LastSelection to the generated wire type.
func toGenPreferences(sel LastSelection) gen.StatsPreferences {
	var p gen.StatsPreferences
	if sel.FromDate != nil {
		p.From = &openapi_types.Date{Time: *sel.FromDate}
	}
	if sel.ToDate != nil {
		p.To = &openapi_types.Date{Time: *sel.ToDate}
	}
	p.PresetId = sel.PresetID
	return p
}

// toGenPreset maps a Preset to the generated wire type.
func toGenPreset(p Preset) gen.StatsPreset {
	return gen.StatsPreset{
		Id:   p.ID,
		Name: p.Name,
		From: openapi_types.Date{Time: p.FromDate},
		To:   openapi_types.Date{Time: p.ToDate},
	}
}
