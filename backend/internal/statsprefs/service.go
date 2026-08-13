package statsprefs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// statsprefsRepo is the interface the Service relies on.
type statsprefsRepo interface {
	GetLastSelection(ctx context.Context, teamID, userID uuid.UUID) (LastSelection, error)
	UpsertLastSelection(ctx context.Context, teamID, userID uuid.UUID, sel LastSelection) error
	PresetExists(ctx context.Context, teamID, userID, presetID uuid.UUID) (bool, error)
	ListPresets(ctx context.Context, teamID, userID uuid.UUID) ([]Preset, error)
	CountPresets(ctx context.Context, teamID, userID uuid.UUID) (int, error)
	CreatePreset(ctx context.Context, teamID, userID uuid.UUID, name string, from, to time.Time) (Preset, error)
	UpdatePreset(ctx context.Context, teamID, userID, presetID uuid.UUID, name *string, from, to *time.Time) (Preset, error)
	DeletePreset(ctx context.Context, teamID, userID, presetID uuid.UUID) error
}

// Service implements statsprefs business logic.
type Service struct {
	repo statsprefsRepo
}

// NewService creates a new Service.
func NewService(repo statsprefsRepo) *Service {
	return &Service{repo: repo}
}

// GetLastSelection returns (teamID, userID)'s last-saved statistics range.
func (s *Service) GetLastSelection(ctx context.Context, teamID, userID uuid.UUID) (LastSelection, error) {
	sel, err := s.repo.GetLastSelection(ctx, teamID, userID)
	if err != nil {
		return LastSelection{}, fmt.Errorf("statsprefs.Service.GetLastSelection: %w", err)
	}
	return sel, nil
}

// SetLastSelection saves (teamID, userID)'s current statistics range. When
// sel.PresetID is set, it must reference a preset (teamID, userID) actually
// owns -- rejected with ErrPresetNotFound otherwise, so a selection can
// never be pointed at another user's or another team's preset (only the
// DB's existence-checking foreign key would otherwise catch a bogus id, and
// it can't check ownership).
func (s *Service) SetLastSelection(ctx context.Context, teamID, userID uuid.UUID, sel LastSelection) error {
	if sel.PresetID != nil {
		ok, err := s.repo.PresetExists(ctx, teamID, userID, *sel.PresetID)
		if err != nil {
			return fmt.Errorf("statsprefs.Service.SetLastSelection: %w", err)
		}
		if !ok {
			return ErrPresetNotFound
		}
	}
	if err := s.repo.UpsertLastSelection(ctx, teamID, userID, sel); err != nil {
		return fmt.Errorf("statsprefs.Service.SetLastSelection: %w", err)
	}
	return nil
}

// ListPresets returns every preset (teamID, userID) has saved.
func (s *Service) ListPresets(ctx context.Context, teamID, userID uuid.UUID) ([]Preset, error) {
	presets, err := s.repo.ListPresets(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("statsprefs.Service.ListPresets: %w", err)
	}
	return presets, nil
}

// CreatePreset saves a new named date-range preset for (teamID, userID),
// rejecting the call once the caller already has maxPresetsPerTeamUser
// saved.
func (s *Service) CreatePreset(ctx context.Context, teamID, userID uuid.UUID, name string, from, to time.Time) (Preset, error) {
	count, err := s.repo.CountPresets(ctx, teamID, userID)
	if err != nil {
		return Preset{}, fmt.Errorf("statsprefs.Service.CreatePreset: %w", err)
	}
	if count >= maxPresetsPerTeamUser {
		return Preset{}, ErrTooManyPresets
	}
	p, err := s.repo.CreatePreset(ctx, teamID, userID, name, from, to)
	if err != nil {
		return Preset{}, fmt.Errorf("statsprefs.Service.CreatePreset: %w", err)
	}
	return p, nil
}

// UpdatePreset applies a partial rename/reschedule to a preset owned by
// (teamID, userID).
func (s *Service) UpdatePreset(ctx context.Context, teamID, userID, presetID uuid.UUID, name *string, from, to *time.Time) (Preset, error) {
	p, err := s.repo.UpdatePreset(ctx, teamID, userID, presetID, name, from, to)
	if err != nil {
		return Preset{}, fmt.Errorf("statsprefs.Service.UpdatePreset: %w", err)
	}
	return p, nil
}

// DeletePreset removes a preset owned by (teamID, userID).
func (s *Service) DeletePreset(ctx context.Context, teamID, userID, presetID uuid.UUID) error {
	if err := s.repo.DeletePreset(ctx, teamID, userID, presetID); err != nil {
		return fmt.Errorf("statsprefs.Service.DeletePreset: %w", err)
	}
	return nil
}
