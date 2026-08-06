package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/tools/spielerplus-import/spielerplus"
)

// ErrFinanceRecordSkipped is returned (wrapped) for a finance record that
// violates a Teamverwaltung constraint (a non-positive amount, mainly -
// transactions/penalties/contributions all CHECK amount > 0). Per the same
// "skipped, not fatal" convention as ErrAbsenceSkipped, callers should log
// and continue rather than abort the run.
var ErrFinanceRecordSkipped = errors.New("finance record skipped")

// InsertTransaction writes one Kasse (cashbox) ledger entry. Not idempotent
// by itself - callers must consult the local state file (State.Transactions),
// since transactions has no external-id column.
func (s *Store) InsertTransaction(ctx context.Context, teamID string, tx spielerplus.Transaction) (id string, err error) {
	if tx.AmountCents <= 0 {
		return "", fmt.Errorf("%w: transaction %q amount must be positive, got %d cents", ErrFinanceRecordSkipped, tx.Title, tx.AmountCents)
	}
	if s.DryRun {
		return dryRunID, nil
	}

	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO transactions (id, team_id, type, title, amount, date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
	`, newID, teamID, tx.Type, tx.Title, tx.AmountCents, tx.Date.Format("2006-01-02"))
	if err != nil {
		return "", fmt.Errorf("db: insert transaction %q (%s): %w", tx.Title, tx.ID, err)
	}
	return newID, nil
}

// InsertPenalty creates a Teamverwaltung penalty-catalog entry. Not
// idempotent by itself - callers must consult the local state file
// (State.PenaltyCatalog), since penalties has no external-id column.
func (s *Store) InsertPenalty(ctx context.Context, teamID, label string, amountCents int64) (id string, err error) {
	if amountCents <= 0 {
		return "", fmt.Errorf("%w: penalty %q amount must be positive, got %d cents", ErrFinanceRecordSkipped, label, amountCents)
	}
	if s.DryRun {
		return dryRunID, nil
	}

	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO penalties (id, team_id, label, amount, updated_at)
		VALUES ($1, $2, $3, $4, now())
	`, newID, teamID, label, amountCents)
	if err != nil {
		return "", fmt.Errorf("db: insert penalty %q: %w", label, err)
	}
	return newID, nil
}

// InsertPenaltyAssignment writes one punishment assigned to a member.
// penaltyID may be "" (no matching catalog entry was found for the
// assignment's reason text) - penalty_assignments.penalty_id is nullable
// for exactly this case, snapshotting amount/label directly instead. Not
// idempotent by itself - callers must consult the local state file
// (State.PenaltyAssignments), since penalty_assignments has no
// external-id column.
func (s *Store) InsertPenaltyAssignment(ctx context.Context, teamID, userID, penaltyID string, amountCents int64, label string, paid bool, date time.Time) (id string, err error) {
	if amountCents <= 0 {
		return "", fmt.Errorf("%w: penalty assignment (user %s) amount must be positive, got %d cents", ErrFinanceRecordSkipped, userID, amountCents)
	}
	if s.DryRun {
		return dryRunID, nil
	}

	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO penalty_assignments (id, team_id, user_id, penalty_id, paid, date, amount, label)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, newID, teamID, userID, nullIfEmpty(penaltyID), paid, date.Format("2006-01-02"), amountCents, label)
	if err != nil {
		return "", fmt.Errorf("db: insert penalty assignment (team %s, user %s): %w", teamID, userID, err)
	}
	return newID, nil
}

func contributionStatus(paid bool) string {
	if paid {
		return "paid"
	}
	return "open"
}

// UpsertContribution writes one member's status for one SpielerPlus due
// column, keyed by (teamID, userID, month). Naturally idempotent on the
// DB's own UNIQUE(team_id, user_id, month) - no state file needed. month
// is a synthetic placeholder assigned by importrun (SpielerPlus's due
// columns carry no real month of their own - see design.md), not a real
// due date.
func (s *Store) UpsertContribution(ctx context.Context, teamID, userID, month, label string, amountCents int64, paid bool) (id string, err error) {
	if amountCents <= 0 {
		return "", fmt.Errorf("%w: contribution %q (user %s) amount must be positive, got %d cents", ErrFinanceRecordSkipped, label, userID, amountCents)
	}
	if s.DryRun {
		return dryRunID, nil
	}

	newID := uuid.NewString()
	err = s.Pool.QueryRow(ctx, `
		INSERT INTO contributions (id, team_id, user_id, month, label, amount, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (team_id, user_id, month) DO UPDATE
			SET label = EXCLUDED.label, amount = EXCLUDED.amount, status = EXCLUDED.status, updated_at = now()
		RETURNING id
	`, newID, teamID, userID, month, nullIfEmpty(label), amountCents, contributionStatus(paid)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("db: upsert contribution (team %s, user %s, month %s): %w", teamID, userID, month, err)
	}
	return id, nil
}
