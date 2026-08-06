package finances

import (
	"time"

	"github.com/google/uuid"
)

// TransactionRow is the internal DB representation of a financial transaction.
// Amount is stored as integer cents (e.g. 1050 = 10.50) to avoid the binary
// floating-point imprecision that float64 introduces at the API boundary.
type TransactionRow struct {
	ID       uuid.UUID
	TeamID   uuid.UUID
	Type     string
	Title    string
	Amount   int64
	Date     time.Time
	Category *string
	// ContributionID is the membership fee this transaction (fully or
	// partially) pays, or nil if it isn't a fee payment. ON DELETE SET NULL
	// on the FK -- deleting the contribution never deletes booked income.
	// Mutually exclusive with PenaltyAssignmentID.
	ContributionID *uuid.UUID
	// PenaltyAssignmentID is the penalty assignment this transaction pays,
	// or nil if it isn't a fine payment. Same ON DELETE SET NULL/mutual-
	// exclusivity reasoning as ContributionID.
	PenaltyAssignmentID *uuid.UUID
	CreatedAt           time.Time
}

// PenaltyRow is the internal DB representation of a penalty definition.
// Amount is stored as integer cents.
type PenaltyRow struct {
	ID     uuid.UUID
	TeamID uuid.UUID
	Label  string
	Amount int64
}

// PenaltyAssignmentRow is the internal DB representation of a penalty
// assignment. PenaltyAmount/PaidAmount are stored as integer cents.
// PaidAmount is never stored on the penalty_assignments table itself -- like
// ContributionRow.PaidAmount, it's computed live as the sum of linked
// transactions (see Repository.ListAssignments) so it can never drift from
// the ledger that is its source of truth.
type PenaltyAssignmentRow struct {
	ID     uuid.UUID
	TeamID uuid.UUID
	UserID uuid.UUID
	// PenaltyID is nullable: it becomes NULL when the source penalty catalog
	// entry is deleted (ON DELETE SET NULL, migration 00027). The assignment's
	// snapshotted PenaltyLabel/PenaltyAmount remain the authoritative record.
	PenaltyID         *uuid.UUID
	PaidAmount        int64
	Date              time.Time
	Note              *string
	PenaltyLabel      *string
	PenaltyAmount     *int64
	MemberName        *string
	MemberAvatarColor *string
	HasPhoto          *bool
}

// ContributionRow is the internal DB representation of a membership fee, one
// row per member it applies to. Amount/PaidAmount are stored as integer
// cents. PaidAmount is never stored on the contributions table itself -- it
// is computed live as the sum of linked transactions (see
// Repository.ListContributions) so it can never drift from the ledger that
// is its source of truth.
type ContributionRow struct {
	ID                uuid.UUID
	TeamID            uuid.UUID
	UserID            uuid.UUID
	Name              string
	Amount            int64
	DueDate           *time.Time
	PaidAmount        int64
	MemberName        *string
	MemberAvatarColor *string
	HasPhoto          *bool
}

// PenaltyPatch carries optional fields for an UPDATE penalties query.
type PenaltyPatch struct {
	Label  *string
	Amount *int64
}

// TransactionPatch carries optional fields for an UPDATE transactions query.
type TransactionPatch struct {
	Type     *string
	Title    *string
	Amount   *int64
	Category *string
	Date     *time.Time
}

// ContributionPatch carries optional fields for an UPDATE contributions query.
type ContributionPatch struct {
	Name    *string
	Amount  *int64
	DueDate *time.Time
}
