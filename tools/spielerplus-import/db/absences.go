package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// maxAbsenceSpanDays mirrors the absences_span_within_limit CHECK
// constraint (backend/internal/db/migrations/00001_init.sql).
const maxAbsenceSpanDays = 1095

// ErrAbsenceSkipped is returned (wrapped) by InsertAbsence for an absence
// that violates a Teamverwaltung constraint. Per spec ("Invalid absences are
// skipped, not fatal"), callers should log and continue rather than abort
// the run.
var ErrAbsenceSkipped = errors.New("absence skipped")

// InsertAbsence writes one absence for userID in teamID, replicating the
// checks backend/internal/absences/service.go would otherwise enforce
// (this importer writes directly to Postgres, bypassing that service): the
// date range must not be inverted, must not exceed the span cap, and must
// not overlap an absence already recorded for that user in that team.
// Violations return a wrapped ErrAbsenceSkipped instead of failing the run.
func (s *Store) InsertAbsence(ctx context.Context, teamID, userID string, from, to time.Time, reason string) (id string, err error) {
	if to.Before(from) {
		return "", fmt.Errorf("%w: to_date %s before from_date %s", ErrAbsenceSkipped, to.Format("2006-01-02"), from.Format("2006-01-02"))
	}
	if to.Sub(from) > maxAbsenceSpanDays*24*time.Hour {
		return "", fmt.Errorf("%w: span exceeds %d days", ErrAbsenceSkipped, maxAbsenceSpanDays)
	}

	var overlaps bool
	err = s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM absences
			WHERE team_id = $1 AND user_id = $2
			  AND from_date <= $4::date AND to_date >= $3::date
		)
	`, teamID, userID, from.Format("2006-01-02"), to.Format("2006-01-02")).Scan(&overlaps)
	if err != nil {
		return "", fmt.Errorf("db: check overlapping absence (team %s, user %s): %w", teamID, userID, err)
	}
	if overlaps {
		return "", fmt.Errorf("%w: overlaps an existing absence for this member", ErrAbsenceSkipped)
	}

	if s.DryRun {
		return dryRunID, nil
	}

	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO absences (id, user_id, team_id, from_date, to_date, reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
	`, newID, userID, teamID, from.Format("2006-01-02"), to.Format("2006-01-02"), nullIfEmpty(reason))
	if err != nil {
		return "", fmt.Errorf("db: insert absence (team %s, user %s): %w", teamID, userID, err)
	}
	return newID, nil
}
