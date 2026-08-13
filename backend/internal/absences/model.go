package absences

import (
	"time"

	"github.com/google/uuid"
)

// AbsenceRow mirrors the DB absences table.
type AbsenceRow struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	TeamId    uuid.UUID
	FromDate  time.Time
	ToDate    time.Time
	Reason    *string
	CreatedAt time.Time
	// NotRelevantForStats excludes this absence's covered event dates from
	// the owner's attendance statistics entirely, instead of counting them
	// as absent.
	NotRelevantForStats bool
	// NotRelevantSetBy is who last set NotRelevantForStats -- nil if it has
	// never been touched (still at its false default).
	NotRelevantSetBy *uuid.UUID
	// Joined from users
	MemberName        *string
	MemberAvatarColor *string
	HasPhoto          bool
	// Joined from memberships (nil if the user is no longer a member of TeamId)
	MembershipId *uuid.UUID
	// Joined from roles (primary role)
	RoleName  *string
	RoleColor *string
}
