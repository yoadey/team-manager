package roles

import (
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// RolePatch carries optional fields for an UPDATE roles query.
type RolePatch struct {
	Name        *string
	Color       *string
	Permissions *teams.PermissionsJSON
}
