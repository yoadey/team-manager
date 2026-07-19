package finances

import (
	"time"

	"github.com/google/uuid"
)

// TransactionRow is the internal DB representation of a financial transaction.
// Amount is stored as integer cents (e.g. 1050 = 10.50) to avoid the binary
// floating-point imprecision that float64 introduces at the API boundary.
type TransactionRow struct {
	ID        uuid.UUID
	TeamID    uuid.UUID
	Type      string
	Title     string
	Amount    int64
	Date      time.Time
	Category  *string
	CreatedAt time.Time
}

// PenaltyRow is the internal DB representation of a penalty definition.
// Amount is stored as integer cents.
type PenaltyRow struct {
	ID     uuid.UUID
	TeamID uuid.UUID
	Label  string
	Amount int64
}

// PenaltyAssignmentRow is the internal DB representation of a penalty assignment.
// PenaltyAmount is stored as integer cents.
type PenaltyAssignmentRow struct {
	ID     uuid.UUID
	TeamID uuid.UUID
	UserID uuid.UUID
	// PenaltyID is nullable: it becomes NULL when the source penalty catalog
	// entry is deleted (ON DELETE SET NULL, migration 00027). The assignment's
	// snapshotted PenaltyLabel/PenaltyAmount remain the authoritative record.
	PenaltyID         *uuid.UUID
	Paid              bool
	Date              time.Time
	PenaltyLabel      *string
	PenaltyAmount     *int64
	MemberName        *string
	MemberAvatarColor *string
	HasPhoto          *bool
}

// ContributionRow is the internal DB representation of a monthly contribution.
// Amount is stored as integer cents.
type ContributionRow struct {
	ID                uuid.UUID
	TeamID            uuid.UUID
	UserID            uuid.UUID
	Month             string // YYYY-MM
	Label             *string
	Amount            int64
	Status            string
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
	Label  *string
	Amount *int64
}
