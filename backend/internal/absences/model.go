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
	// Joined from users
	MemberName        *string
	MemberAvatarColor *string
	HasPhoto          bool
	// Joined from roles (primary role)
	RoleName  *string
	RoleColor *string
}
