package finances

import (
	"time"

	"github.com/google/uuid"
)

// TransactionRow is the internal DB representation of a financial transaction.
type TransactionRow struct {
	ID        uuid.UUID
	TeamID    uuid.UUID
	Type      string
	Title     string
	Amount    float64
	Date      time.Time
	Category  *string
	CreatedAt time.Time
}

// PenaltyRow is the internal DB representation of a penalty definition.
type PenaltyRow struct {
	ID     uuid.UUID
	TeamID uuid.UUID
	Label  string
	Amount float64
}

// PenaltyAssignmentRow is the internal DB representation of a penalty assignment.
type PenaltyAssignmentRow struct {
	ID                uuid.UUID
	TeamID            uuid.UUID
	UserID            uuid.UUID
	PenaltyID         uuid.UUID
	Paid              bool
	Date              time.Time
	PenaltyLabel      *string
	PenaltyAmount     *float64
	MemberName        *string
	MemberAvatarColor *string
	HasPhoto          *bool
}

// ContributionRow is the internal DB representation of a monthly contribution.
type ContributionRow struct {
	ID                uuid.UUID
	TeamID            uuid.UUID
	UserID            uuid.UUID
	Month             string // YYYY-MM
	Label             *string
	Amount            float64
	Status            string
	MemberName        *string
	MemberAvatarColor *string
	HasPhoto          *bool
}

// PenaltyPatch carries optional fields for an UPDATE penalties query.
type PenaltyPatch struct {
	Label  *string
	Amount *float64
}

// TransactionPatch carries optional fields for an UPDATE transactions query.
type TransactionPatch struct {
	Type     *string
	Title    *string
	Amount   *float64
	Category *string
}

// ContributionPatch carries optional fields for an UPDATE contributions query.
type ContributionPatch struct {
	Label  *string
	Amount *float64
}
