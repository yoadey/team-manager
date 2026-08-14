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
	HasPhoto     bool
	Group        *string
	// Title is a short, self-service, purely cosmetic label (e.g.
	// "Witzbeauftragter") -- display-only, never interpreted by RBAC.
	Title    *string
	JoinedAt time.Time
	Roles    []teams.RoleRow
	// ExcludeFromStats removes this member from personal-quota-oriented
	// statistics views (overview, single-member, attendance matrix) while
	// leaving event-level turnout aggregates unaffected -- see
	// stats.Repository's doc comments on the roster joins for the exact
	// per-query treatment.
	ExcludeFromStats bool
}

// MemberPatch carries optional fields for an UPDATE on users/memberships.
type MemberPatch struct {
	Name             *string
	Email            *string
	Phone            *string
	Birthday         *time.Time
	Address          *string
	Group            *string
	Title            *string
	ExcludeFromStats *bool
}
