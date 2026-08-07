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
// for exactly this case, snapshotting amount/label directly instead
// (mirroring what the backend's own CreateAssignment snapshots from the
// catalog entry it looked up, minus the requirement that a catalog entry
// exist at all - this importer writes directly to Postgres, bypassing that
// requirement, same as everywhere else it bypasses the app's services).
// Whether the assignment was paid on SpielerPlus is not written anywhere:
// since migration 00020_penalty_assignment_linked_payment,
// penalty_assignments has no paid column at all - paid state is derived
// from income transactions linked via transactions.penalty_assignment_id,
// which this importer deliberately does not attempt to set (see
// design.md's "paid state is not linked" decision). Not idempotent by
// itself - callers must consult the local state file
// (State.PenaltyAssignments), since penalty_assignments has no
// external-id column.
func (s *Store) InsertPenaltyAssignment(ctx context.Context, teamID, userID, penaltyID string, amountCents int64, label string, date time.Time) (id string, err error) {
	if amountCents <= 0 {
		return "", fmt.Errorf("%w: penalty assignment (user %s) amount must be positive, got %d cents", ErrFinanceRecordSkipped, userID, amountCents)
	}
	if s.DryRun {
		return dryRunID, nil
	}

	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO penalty_assignments (id, team_id, user_id, penalty_id, date, amount, label)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, newID, teamID, userID, nullIfEmpty(penaltyID), date.Format("2006-01-02"), amountCents, label)
	if err != nil {
		return "", fmt.Errorf("db: insert penalty assignment (team %s, user %s): %w", teamID, userID, err)
	}
	return newID, nil
}

// InsertContribution writes one member's SpielerPlus due column as a
// Teamverwaltung contribution. Since migration
// 00018_flexible_membership_fees, contributions no longer has a "month"
// concept at all (name is free text, due_date is an optional real date, and
// there is no UNIQUE(team_id, user_id, month) constraint to upsert
// against) - so each SpielerPlus due column simply becomes its own row,
// with no synthetic month juggling needed (see design.md; this replaced an
// earlier, more awkward approach that spread columns across made-up
// consecutive months to work around the old schema's one-row-per-month
// limit). dueDate is nil: SpielerPlus's due columns carry no date of their
// own to map it from. Whether the due was paid on SpielerPlus is not
// written anywhere, for the same reason as InsertPenaltyAssignment - paid
// state is now derived from a linked transaction, which this importer
// doesn't attempt to set. Not idempotent by itself - callers must consult
// the local state file (State.Dues), since contributions has no
// external-id column (and, post-migration, no other natural unique key to
// dedupe on either).
func (s *Store) InsertContribution(ctx context.Context, teamID, userID, name string, amountCents int64) (id string, err error) {
	if amountCents <= 0 {
		return "", fmt.Errorf("%w: contribution %q (user %s) amount must be positive, got %d cents", ErrFinanceRecordSkipped, name, userID, amountCents)
	}
	if s.DryRun {
		return dryRunID, nil
	}

	newID := uuid.NewString()
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO contributions (id, team_id, user_id, name, amount, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
	`, newID, teamID, userID, name, amountCents)
	if err != nil {
		return "", fmt.Errorf("db: insert contribution %q (user %s): %w", name, userID, err)
	}
	return newID, nil
}
