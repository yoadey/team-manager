package statsprefs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgCheckViolation is the Postgres SQLSTATE for a violated CHECK constraint.
// Mirrors absences.pgCheckViolation.
const pgCheckViolation = "23514"

// Repository handles all stats_last_selection / stats_view_presets DB
// operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetLastSelection returns (teamID, userID)'s last-saved statistics
// selection, or a zero-value LastSelection (all fields nil) if the member
// has never saved one -- mirroring push.Repository.GetPreferences' "no row
// yet" handling.
func (r *Repository) GetLastSelection(ctx context.Context, teamID, userID uuid.UUID) (LastSelection, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var sel LastSelection
	err := r.pool.QueryRow(ctx, `
		SELECT from_date, to_date, preset_id
		FROM stats_last_selection
		WHERE team_id = $1 AND user_id = $2
	`, teamID, userID).Scan(&sel.FromDate, &sel.ToDate, &sel.PresetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return LastSelection{}, nil
	}
	if err != nil {
		return LastSelection{}, fmt.Errorf("statsprefs.Repository.GetLastSelection: %w", err)
	}
	return sel, nil
}

// UpsertLastSelection saves (teamID, userID)'s current statistics
// selection, creating the row on first save.
func (r *Repository) UpsertLastSelection(ctx context.Context, teamID, userID uuid.UUID, sel LastSelection) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO stats_last_selection (team_id, user_id, from_date, to_date, preset_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (team_id, user_id) DO UPDATE SET
			from_date  = EXCLUDED.from_date,
			to_date    = EXCLUDED.to_date,
			preset_id  = EXCLUDED.preset_id,
			updated_at = now()
	`, teamID, userID, sel.FromDate, sel.ToDate, sel.PresetID)
	if err != nil {
		return fmt.Errorf("statsprefs.Repository.UpsertLastSelection: %w", err)
	}
	return nil
}

// PresetExists reports whether presetID is a preset owned by (teamID,
// userID) -- used by Service.SetLastSelection to validate a caller-supplied
// presetId before persisting it, so a saved selection can never reference
// another user's or another team's preset.
func (r *Repository) PresetExists(ctx context.Context, teamID, userID, presetID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM stats_view_presets WHERE id = $1 AND team_id = $2 AND user_id = $3)
	`, presetID, teamID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("statsprefs.Repository.PresetExists: %w", err)
	}
	return exists, nil
}

// ListPresets returns every preset (teamID, userID) has saved, newest first.
func (r *Repository) ListPresets(ctx context.Context, teamID, userID uuid.UUID) ([]Preset, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, from_date, to_date
		FROM stats_view_presets
		WHERE team_id = $1 AND user_id = $2
		ORDER BY created_at DESC
	`, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("statsprefs.Repository.ListPresets: %w", err)
	}
	defer rows.Close()

	var result []Preset
	for rows.Next() {
		var p Preset
		if err := rows.Scan(&p.ID, &p.Name, &p.FromDate, &p.ToDate); err != nil {
			return nil, fmt.Errorf("statsprefs.Repository.ListPresets scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// CountPresets returns how many presets (teamID, userID) currently has
// saved, used by the service layer to enforce maxPresetsPerTeamUser before
// inserting a new one.
func (r *Repository) CountPresets(ctx context.Context, teamID, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM stats_view_presets WHERE team_id = $1 AND user_id = $2
	`, teamID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("statsprefs.Repository.CountPresets: %w", err)
	}
	return count, nil
}

// CreatePreset inserts a new preset for (teamID, userID).
func (r *Repository) CreatePreset(ctx context.Context, teamID, userID uuid.UUID, name string, from, to time.Time) (Preset, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	p := Preset{Name: name, FromDate: from, ToDate: to}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO stats_view_presets (team_id, user_id, name, from_date, to_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, teamID, userID, name, from, to).Scan(&p.ID)
	if err != nil {
		return Preset{}, fmt.Errorf("statsprefs.Repository.CreatePreset: %w", err)
	}
	return p, nil
}

// UpdatePreset applies a partial patch (nil fields left unchanged) to a
// preset owned by (teamID, userID), returning pgx.ErrNoRows if no such
// preset exists -- scoping the WHERE clause to team+user (not just id)
// means a caller can never rename/reschedule another member's preset, since
// the query simply matches no row for them.
func (r *Repository) UpdatePreset(ctx context.Context, teamID, userID, presetID uuid.UUID, name *string, from, to *time.Time) (Preset, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var p Preset
	err := r.pool.QueryRow(ctx, `
		UPDATE stats_view_presets
		SET name      = COALESCE($4, name),
			from_date = COALESCE($5, from_date),
			to_date   = COALESCE($6, to_date)
		WHERE id = $1 AND team_id = $2 AND user_id = $3
		RETURNING id, name, from_date, to_date
	`, presetID, teamID, userID, name, from, to).Scan(&p.ID, &p.Name, &p.FromDate, &p.ToDate)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation {
			return Preset{}, ErrInvalidDateRange
		}
		return Preset{}, fmt.Errorf("statsprefs.Repository.UpdatePreset: %w", err)
	}
	return p, nil
}

// DeletePreset removes a preset owned by (teamID, userID). Deleting a
// preset that doesn't exist, or belongs to a different user, affects no
// rows and is not an error -- delete is idempotent, matching
// push.Repository.Delete's convention.
func (r *Repository) DeletePreset(ctx context.Context, teamID, userID, presetID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := r.pool.Exec(ctx, `
		DELETE FROM stats_view_presets WHERE id = $1 AND team_id = $2 AND user_id = $3
	`, presetID, teamID, userID)
	if err != nil {
		return fmt.Errorf("statsprefs.Repository.DeletePreset: %w", err)
	}
	return nil
}
