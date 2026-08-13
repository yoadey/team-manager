package statsprefs_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/statsprefs"
)

type mockRepo struct {
	getLastSelectionFn    func(ctx context.Context, teamID, userID uuid.UUID) (statsprefs.LastSelection, error)
	upsertLastSelectionFn func(ctx context.Context, teamID, userID uuid.UUID, sel statsprefs.LastSelection) error
	listPresetsFn         func(ctx context.Context, teamID, userID uuid.UUID) ([]statsprefs.Preset, error)
	countPresetsFn        func(ctx context.Context, teamID, userID uuid.UUID) (int, error)
	createPresetFn        func(ctx context.Context, teamID, userID uuid.UUID, name string, from, to time.Time) (statsprefs.Preset, error)
	updatePresetFn        func(ctx context.Context, teamID, userID, presetID uuid.UUID, name *string, from, to *time.Time) (statsprefs.Preset, error)
	deletePresetFn        func(ctx context.Context, teamID, userID, presetID uuid.UUID) error
}

func (m *mockRepo) GetLastSelection(ctx context.Context, teamID, userID uuid.UUID) (statsprefs.LastSelection, error) {
	return m.getLastSelectionFn(ctx, teamID, userID)
}

func (m *mockRepo) UpsertLastSelection(ctx context.Context, teamID, userID uuid.UUID, sel statsprefs.LastSelection) error {
	return m.upsertLastSelectionFn(ctx, teamID, userID, sel)
}

func (m *mockRepo) ListPresets(ctx context.Context, teamID, userID uuid.UUID) ([]statsprefs.Preset, error) {
	return m.listPresetsFn(ctx, teamID, userID)
}

func (m *mockRepo) CountPresets(ctx context.Context, teamID, userID uuid.UUID) (int, error) {
	return m.countPresetsFn(ctx, teamID, userID)
}

func (m *mockRepo) CreatePreset(ctx context.Context, teamID, userID uuid.UUID, name string, from, to time.Time) (statsprefs.Preset, error) {
	return m.createPresetFn(ctx, teamID, userID, name, from, to)
}

func (m *mockRepo) UpdatePreset(ctx context.Context, teamID, userID, presetID uuid.UUID, name *string, from, to *time.Time) (statsprefs.Preset, error) {
	return m.updatePresetFn(ctx, teamID, userID, presetID, name, from, to)
}

func (m *mockRepo) DeletePreset(ctx context.Context, teamID, userID, presetID uuid.UUID) error {
	return m.deletePresetFn(ctx, teamID, userID, presetID)
}

func TestService_GetLastSelection_DelegatesToRepository(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	from := time.Now()
	want := statsprefs.LastSelection{FromDate: &from}
	repo := &mockRepo{
		getLastSelectionFn: func(_ context.Context, gotTeam, gotUser uuid.UUID) (statsprefs.LastSelection, error) {
			assert.Equal(t, teamID, gotTeam)
			assert.Equal(t, userID, gotUser)
			return want, nil
		},
	}

	svc := statsprefs.NewService(repo)
	got, err := svc.GetLastSelection(context.Background(), teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestService_CreatePreset_UnderCap_Succeeds(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	from, to := time.Now(), time.Now().AddDate(0, 3, 0)
	created := false
	repo := &mockRepo{
		countPresetsFn: func(_ context.Context, _, _ uuid.UUID) (int, error) { return 19, nil },
		createPresetFn: func(_ context.Context, _, _ uuid.UUID, name string, _, _ time.Time) (statsprefs.Preset, error) {
			created = true
			return statsprefs.Preset{Name: name}, nil
		},
	}

	svc := statsprefs.NewService(repo)
	_, err := svc.CreatePreset(context.Background(), teamID, userID, "Saison", from, to)
	require.NoError(t, err)
	assert.True(t, created)
}

// TestService_CreatePreset_AtCap_RejectsWithoutInserting verifies the
// maxPresetsPerTeamUser guard rejects the call before ever reaching the
// repository's insert -- a caller already at the cap must not be able to
// create an unbounded number of presets by racing the check.
func TestService_CreatePreset_AtCap_RejectsWithoutInserting(t *testing.T) {
	t.Parallel()

	teamID, userID := uuid.New(), uuid.New()
	from, to := time.Now(), time.Now().AddDate(0, 3, 0)
	repo := &mockRepo{
		countPresetsFn: func(_ context.Context, _, _ uuid.UUID) (int, error) { return 20, nil },
		createPresetFn: func(_ context.Context, _, _ uuid.UUID, _ string, _, _ time.Time) (statsprefs.Preset, error) {
			t.Fatal("CreatePreset must not be called once the cap is reached")
			return statsprefs.Preset{}, nil
		},
	}

	svc := statsprefs.NewService(repo)
	_, err := svc.CreatePreset(context.Background(), teamID, userID, "One Too Many", from, to)
	require.ErrorIs(t, err, statsprefs.ErrTooManyPresets)
}

func TestService_DeletePreset_PropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("db error")
	repo := &mockRepo{
		deletePresetFn: func(_ context.Context, _, _, _ uuid.UUID) error { return repoErr },
	}

	svc := statsprefs.NewService(repo)
	err := svc.DeletePreset(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.Error(t, err)
}
