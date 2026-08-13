package statsprefs_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/apierror"
	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/statsprefs"
)

type mockStatsPrefsService struct {
	getLastSelectionFn func(ctx context.Context, teamID, userID uuid.UUID) (statsprefs.LastSelection, error)
	setLastSelectionFn func(ctx context.Context, teamID, userID uuid.UUID, sel statsprefs.LastSelection) error
	listPresetsFn      func(ctx context.Context, teamID, userID uuid.UUID) ([]statsprefs.Preset, error)
	createPresetFn     func(ctx context.Context, teamID, userID uuid.UUID, name string, from, to time.Time) (statsprefs.Preset, error)
	updatePresetFn     func(ctx context.Context, teamID, userID, presetID uuid.UUID, name *string, from, to *time.Time) (statsprefs.Preset, error)
	deletePresetFn     func(ctx context.Context, teamID, userID, presetID uuid.UUID) error
}

func (m *mockStatsPrefsService) GetLastSelection(ctx context.Context, teamID, userID uuid.UUID) (statsprefs.LastSelection, error) {
	return m.getLastSelectionFn(ctx, teamID, userID)
}

func (m *mockStatsPrefsService) SetLastSelection(ctx context.Context, teamID, userID uuid.UUID, sel statsprefs.LastSelection) error {
	return m.setLastSelectionFn(ctx, teamID, userID, sel)
}

func (m *mockStatsPrefsService) ListPresets(ctx context.Context, teamID, userID uuid.UUID) ([]statsprefs.Preset, error) {
	return m.listPresetsFn(ctx, teamID, userID)
}

func (m *mockStatsPrefsService) CreatePreset(ctx context.Context, teamID, userID uuid.UUID, name string, from, to time.Time) (statsprefs.Preset, error) {
	return m.createPresetFn(ctx, teamID, userID, name, from, to)
}

func (m *mockStatsPrefsService) UpdatePreset(ctx context.Context, teamID, userID, presetID uuid.UUID, name *string, from, to *time.Time) (statsprefs.Preset, error) {
	return m.updatePresetFn(ctx, teamID, userID, presetID, name, from, to)
}

func (m *mockStatsPrefsService) DeletePreset(ctx context.Context, teamID, userID, presetID uuid.UUID) error {
	return m.deletePresetFn(ctx, teamID, userID, presetID)
}

var (
	statsPrefsTeamID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	statsPrefsUserID = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
)

func statsPrefsAuthedCtx() context.Context {
	user := &auth.UserRow{Id: statsPrefsUserID, Name: "Test User", Email: "test@example.com", AvatarColor: "#6366f1", CreatedAt: time.Now()}
	return auth.ContextWithUser(context.Background(), user)
}

func TestHandler_GetStatsPreferences_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := statsprefs.NewHandler(&mockStatsPrefsService{}, slog.Default())
	_, err := h.GetStatsPreferences(context.Background(), gen.GetStatsPreferencesRequestObject{TeamId: statsPrefsTeamID})
	require.Error(t, err)
}

func TestHandler_GetStatsPreferences_Success(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := &mockStatsPrefsService{
		getLastSelectionFn: func(_ context.Context, gotTeam, gotUser uuid.UUID) (statsprefs.LastSelection, error) {
			assert.Equal(t, statsPrefsTeamID, gotTeam)
			assert.Equal(t, statsPrefsUserID, gotUser)
			return statsprefs.LastSelection{FromDate: &from}, nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	resp, err := h.GetStatsPreferences(statsPrefsAuthedCtx(), gen.GetStatsPreferencesRequestObject{TeamId: statsPrefsTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitGetStatsPreferencesResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_SetStatsPreferences_MissingBody(t *testing.T) {
	t.Parallel()
	h := statsprefs.NewHandler(&mockStatsPrefsService{}, slog.Default())
	_, err := h.SetStatsPreferences(statsPrefsAuthedCtx(), gen.SetStatsPreferencesRequestObject{TeamId: statsPrefsTeamID, Body: nil})
	require.Error(t, err)
}

func TestHandler_SetStatsPreferences_Success(t *testing.T) {
	t.Parallel()
	from := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	to := openapi_types.Date{Time: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)}
	var gotSel statsprefs.LastSelection
	svc := &mockStatsPrefsService{
		setLastSelectionFn: func(_ context.Context, _, _ uuid.UUID, sel statsprefs.LastSelection) error {
			gotSel = sel
			return nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	resp, err := h.SetStatsPreferences(statsPrefsAuthedCtx(), gen.SetStatsPreferencesRequestObject{
		TeamId: statsPrefsTeamID,
		Body:   &gen.SetStatsPreferencesJSONRequestBody{From: from, To: to},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitSetStatsPreferencesResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, gotSel.FromDate)
	assert.True(t, from.Equal(*gotSel.FromDate))
}

func TestHandler_ListStatsPresets_Success(t *testing.T) {
	t.Parallel()
	presets := []statsprefs.Preset{
		{ID: uuid.New(), Name: "Saison 2026/27", FromDate: time.Now(), ToDate: time.Now().AddDate(0, 9, 0)},
	}
	svc := &mockStatsPrefsService{
		listPresetsFn: func(_ context.Context, _, _ uuid.UUID) ([]statsprefs.Preset, error) { return presets, nil },
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	resp, err := h.ListStatsPresets(statsPrefsAuthedCtx(), gen.ListStatsPresetsRequestObject{TeamId: statsPrefsTeamID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitListStatsPresetsResponse(w))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_CreateStatsPreset_TooMany_ReturnsBadRequest(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		createPresetFn: func(_ context.Context, _, _ uuid.UUID, _ string, _, _ time.Time) (statsprefs.Preset, error) {
			return statsprefs.Preset{}, statsprefs.ErrTooManyPresets
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	_, err := h.CreateStatsPreset(statsPrefsAuthedCtx(), gen.CreateStatsPresetRequestObject{
		TeamId: statsPrefsTeamID,
		Body:   &gen.CreateStatsPresetJSONRequestBody{Name: "One Too Many", From: openapi_types.Date{Time: time.Now()}, To: openapi_types.Date{Time: time.Now()}},
	})
	require.Error(t, err)
}

func TestHandler_UpdateStatsPreset_NotFound(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		updatePresetFn: func(_ context.Context, _, _, _ uuid.UUID, _ *string, _, _ *time.Time) (statsprefs.Preset, error) {
			return statsprefs.Preset{}, pgx.ErrNoRows
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	_, err := h.UpdateStatsPreset(statsPrefsAuthedCtx(), gen.UpdateStatsPresetRequestObject{
		TeamId:   statsPrefsTeamID,
		PresetId: uuid.New(),
		Body:     &gen.UpdateStatsPresetJSONRequestBody{},
	})
	require.Error(t, err)
}

func TestHandler_DeleteStatsPreset_Success(t *testing.T) {
	t.Parallel()
	var gotPresetID uuid.UUID
	svc := &mockStatsPrefsService{
		deletePresetFn: func(_ context.Context, _, _, presetID uuid.UUID) error {
			gotPresetID = presetID
			return nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	presetID := uuid.New()
	resp, err := h.DeleteStatsPreset(statsPrefsAuthedCtx(), gen.DeleteStatsPresetRequestObject{TeamId: statsPrefsTeamID, PresetId: presetID})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	require.NoError(t, resp.VisitDeleteStatsPresetResponse(w))
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, presetID, gotPresetID)
}

func TestHandler_DeleteStatsPreset_Unauthenticated(t *testing.T) {
	t.Parallel()
	h := statsprefs.NewHandler(&mockStatsPrefsService{}, slog.Default())
	_, err := h.DeleteStatsPreset(context.Background(), gen.DeleteStatsPresetRequestObject{TeamId: statsPrefsTeamID, PresetId: uuid.New()})
	require.Error(t, err)
}

func TestHandler_SetStatsPreferences_FromAfterTo_RejectsWithoutCallingService(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		setLastSelectionFn: func(_ context.Context, _, _ uuid.UUID, _ statsprefs.LastSelection) error {
			t.Fatal("SetLastSelection must not be called when from is after to")
			return nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	from := openapi_types.Date{Time: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)}
	to := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	_, err := h.SetStatsPreferences(statsPrefsAuthedCtx(), gen.SetStatsPreferencesRequestObject{
		TeamId: statsPrefsTeamID,
		Body:   &gen.SetStatsPreferencesJSONRequestBody{From: from, To: to},
	})
	require.Error(t, err)
}

func TestHandler_SetStatsPreferences_ForeignPresetId_ReturnsBadRequest(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		setLastSelectionFn: func(_ context.Context, _, _ uuid.UUID, _ statsprefs.LastSelection) error {
			return statsprefs.ErrPresetNotFound
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	from := openapi_types.Date{Time: time.Now()}
	to := openapi_types.Date{Time: time.Now()}
	foreignID := uuid.New()
	_, err := h.SetStatsPreferences(statsPrefsAuthedCtx(), gen.SetStatsPreferencesRequestObject{
		TeamId: statsPrefsTeamID,
		Body:   &gen.SetStatsPreferencesJSONRequestBody{From: from, To: to, PresetId: &foreignID},
	})
	require.Error(t, err)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
}

func TestHandler_CreateStatsPreset_EmptyName_RejectsWithoutCallingService(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		createPresetFn: func(_ context.Context, _, _ uuid.UUID, _ string, _, _ time.Time) (statsprefs.Preset, error) {
			t.Fatal("CreatePreset must not be called with an empty name")
			return statsprefs.Preset{}, nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	_, err := h.CreateStatsPreset(statsPrefsAuthedCtx(), gen.CreateStatsPresetRequestObject{
		TeamId: statsPrefsTeamID,
		Body:   &gen.CreateStatsPresetJSONRequestBody{Name: "   ", From: openapi_types.Date{Time: time.Now()}, To: openapi_types.Date{Time: time.Now()}},
	})
	require.Error(t, err)
}

func TestHandler_CreateStatsPreset_NameTooLong_RejectsWithoutCallingService(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		createPresetFn: func(_ context.Context, _, _ uuid.UUID, _ string, _, _ time.Time) (statsprefs.Preset, error) {
			t.Fatal("CreatePreset must not be called with a too-long name")
			return statsprefs.Preset{}, nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	_, err := h.CreateStatsPreset(statsPrefsAuthedCtx(), gen.CreateStatsPresetRequestObject{
		TeamId: statsPrefsTeamID,
		Body: &gen.CreateStatsPresetJSONRequestBody{
			Name: strings.Repeat("a", 101),
			From: openapi_types.Date{Time: time.Now()},
			To:   openapi_types.Date{Time: time.Now()},
		},
	})
	require.Error(t, err)
}

func TestHandler_CreateStatsPreset_FromAfterTo_RejectsWithoutCallingService(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		createPresetFn: func(_ context.Context, _, _ uuid.UUID, _ string, _, _ time.Time) (statsprefs.Preset, error) {
			t.Fatal("CreatePreset must not be called when from is after to")
			return statsprefs.Preset{}, nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	from := openapi_types.Date{Time: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)}
	to := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	_, err := h.CreateStatsPreset(statsPrefsAuthedCtx(), gen.CreateStatsPresetRequestObject{
		TeamId: statsPrefsTeamID,
		Body:   &gen.CreateStatsPresetJSONRequestBody{Name: "Saison", From: from, To: to},
	})
	require.Error(t, err)
}

func TestHandler_UpdateStatsPreset_FromAfterTo_RejectsWithoutCallingService(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		updatePresetFn: func(_ context.Context, _, _, _ uuid.UUID, _ *string, _, _ *time.Time) (statsprefs.Preset, error) {
			t.Fatal("UpdatePreset must not be called when from is after to")
			return statsprefs.Preset{}, nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	from := openapi_types.Date{Time: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)}
	to := openapi_types.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	_, err := h.UpdateStatsPreset(statsPrefsAuthedCtx(), gen.UpdateStatsPresetRequestObject{
		TeamId:   statsPrefsTeamID,
		PresetId: uuid.New(),
		Body:     &gen.UpdateStatsPresetJSONRequestBody{From: &from, To: &to},
	})
	require.Error(t, err)
}

func TestHandler_UpdateStatsPreset_EmptyName_RejectsWithoutCallingService(t *testing.T) {
	t.Parallel()
	svc := &mockStatsPrefsService{
		updatePresetFn: func(_ context.Context, _, _, _ uuid.UUID, _ *string, _, _ *time.Time) (statsprefs.Preset, error) {
			t.Fatal("UpdatePreset must not be called with an empty name")
			return statsprefs.Preset{}, nil
		},
	}
	h := statsprefs.NewHandler(svc, slog.Default())

	emptyName := "   "
	_, err := h.UpdateStatsPreset(statsPrefsAuthedCtx(), gen.UpdateStatsPresetRequestObject{
		TeamId:   statsPrefsTeamID,
		PresetId: uuid.New(),
		Body:     &gen.UpdateStatsPresetJSONRequestBody{Name: &emptyName},
	})
	require.Error(t, err)
}
