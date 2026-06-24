package teams

import (
	"time"

	"github.com/google/uuid"
)

// TeamRow mirrors the DB teams table.
type TeamRow struct {
	Id                      uuid.UUID
	Name                    string
	Short                   *string
	Icon                    *string
	IconBg                  *string
	IconFg                  *string
	PhotoData               []byte
	PhotoMime               *string
	LogoData                []byte
	LogoMime                *string
	Description             *string
	ReasonVisibilityRoleIDs []uuid.UUID
	CreatedAt               time.Time
}

// MembershipRow mirrors the DB memberships table.
type MembershipRow struct {
	Id       uuid.UUID
	TeamID   uuid.UUID
	UserID   uuid.UUID
	Group    *string
	JoinedAt time.Time
}

// RoleRow mirrors the DB roles table.
type RoleRow struct {
	Id          uuid.UUID
	TeamID      uuid.UUID
	Name        string
	System      bool
	Color       *string
	Permissions PermissionsJSON
}

// PermissionsJSON is the in-memory representation of the JSONB permissions column.
type PermissionsJSON struct {
	Events   string `json:"events"`
	Members  string `json:"members"`
	Finances string `json:"finances"`
	News     string `json:"news"`
	Polls    string `json:"polls"`
	Settings string `json:"settings"`
}

// InviteRow mirrors the DB invites table.
type InviteRow struct {
	Id        uuid.UUID
	TeamID    uuid.UUID
	Code      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// MemberRow is a composite row used in list queries, joining memberships + users + roles.
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
	Roles        []RoleRow
}
