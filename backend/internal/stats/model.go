package stats

import (
	"errors"

	"github.com/google/uuid"
)

// Threshold modes for a group's (role's) attendance threshold. Mirrors the
// OpenAPI StatsThresholdMode enum and role_stats_thresholds.mode's CHECK.
const (
	ThresholdModePercent = "percent"
	ThresholdModeCount   = "count"
)

// DefaultThresholdMode/DefaultThresholdValue are what a role resolves to
// while it has no role_stats_thresholds row: "below 50 %", i.e. exactly the
// cut-off EventStat.enough (pct >= 0.5) has always used for the team as a
// whole, so an unconfigured team sees no change in what counts as too low.
const (
	DefaultThresholdMode  = ThresholdModePercent
	DefaultThresholdValue = 50
)

// Upper bounds for a threshold value per mode -- mirrored by
// role_stats_thresholds' CHECK constraints (the DB backstop) and by
// StatsGroupThreshold.value's maximum in openapi.yaml.
const (
	maxThresholdPercent = 100
	maxThresholdCount   = 10000
)

// ErrInvalidThreshold is returned by Service.SetGroupThresholds for an
// unknown mode or an out-of-range value. Mapped to 400 by the handler.
var ErrInvalidThreshold = errors.New("invalid attendance threshold")

// ErrRoleNotInTeam is returned by Repository.SetGroupThresholds when a
// listed role does not belong to the team being updated. Mapped to 422 by
// the handler (same status teams.ErrRoleNotInTeam maps to).
var ErrRoleNotInTeam = errors.New("role does not belong to team")

// MemberStatRow holds the raw attendance aggregation for one member.
type MemberStatRow struct {
	UserID      uuid.UUID
	Name        string
	AvatarColor string
	HasPhoto    bool
	Yes         int
	Counted     int
}

// EventStatRow holds per-event attendance counts.
type EventStatRow struct {
	EventID uuid.UUID
	Title   string
	Type    string
	Date    string // YYYY-MM-DD
	Yes     int
	Counted int
	// Pending is the number of roster members with no response at all (see
	// EventGroupStatRow.Pending).
	Pending int
}

// MatrixColumnRow is one event column of the attendance matrix.
type MatrixColumnRow struct {
	EventID uuid.UUID
	Title   string
	Type    string
	Date    string // YYYY-MM-DD
}

// MatrixCellRow is one member's effective status for one event. EventID is nil
// for a member who has no events in range (a LEFT JOIN placeholder row that
// still lets the member appear as an empty matrix row).
type MatrixCellRow struct {
	UserID      uuid.UUID
	Name        string
	AvatarColor string
	HasPhoto    bool
	EventID     *uuid.UUID
	Eff         string // yes | no | maybe | pending
}

// EventGroupStatRow holds one group's (role's) attendance counts for one
// event, plus that role's resolved threshold (default already applied).
type EventGroupStatRow struct {
	EventID uuid.UUID
	RoleID  uuid.UUID
	Name    string
	Color   *string
	Yes     int
	Counted int
	// Pending is the number of the group's members who were expected at the
	// event but gave no response: effective status "pending" (normally no
	// stored response, no covering absence, not an opt-out event; a stored
	// 'pending' row -- a reset response -- counts too), excluding members who
	// are not nominated or flagged exclude_from_stats. Kept apart from
	// Counted (yes/no/maybe) because the quotas deliberately ignore such
	// members, while the per-event line-up counts them as not present.
	Pending        int
	ThresholdMode  string // percent | count
	ThresholdValue int
}

// GroupThresholdRow is one role's attendance threshold.
type GroupThresholdRow struct {
	RoleID uuid.UUID
	Mode   string // percent | count
	Value  int
}
