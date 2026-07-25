package polls

import (
	"time"

	"github.com/google/uuid"
)

// PollRow mirrors the DB polls table.
type PollRow struct {
	Id        uuid.UUID
	TeamId    uuid.UUID
	CreatorId uuid.UUID
	Question  string
	Multiple  bool
	Anonymous bool
	CreatedAt time.Time
}

// PollOptionRow mirrors the DB poll_options table.
type PollOptionRow struct {
	Id        uuid.UUID
	PollId    uuid.UUID
	Text      string
	SortOrder int
}

// PollVoteRow represents a row from the poll_votes joined with user info.
type PollVoteRow struct {
	PollId   uuid.UUID
	OptionId uuid.UUID
	UserId   uuid.UUID
	// MembershipId is the voter's membership in the poll's team. Nil when the
	// voter is no longer a member (membership removed after voting) -- the vote
	// row itself survives, so it must still count, just without a photo URL.
	MembershipId *uuid.UUID
	UserName     *string
	UserColor    *string
	HasPhoto     bool
}
