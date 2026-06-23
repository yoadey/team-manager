package members

import (
	"time"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/backend/internal/teams"
)

// MemberRow is a composite row joining memberships + users + roles.
type MemberRow struct {
	MembershipID uuid.UUID
	UserID       uuid.UUID
	Name         string
	Email        string
	Phone        *string
	Birthday     *time.Time
	Address      *string
	AvatarColor  string
	PhotoData    []byte
	Group        *string
	JoinedAt     time.Time
	Roles        []teams.RoleRow
}

// MemberPatch carries optional fields for an UPDATE on users/memberships.
type MemberPatch struct {
	Name     *string
	Email    *string
	Phone    *string
	Birthday *time.Time
	Address  *string
	Group    *string
}

// AddMemberParams holds the fields needed to add a member to a team.
type AddMemberParams struct {
	Name    string
	Email   string
	Phone   *string
	Group   *string
	RoleIDs []string
}
