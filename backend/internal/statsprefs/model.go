package statsprefs

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// LastSelection is a member's most recently selected statistics date range
// for a team. FromDate/ToDate are nil when nothing has been saved yet;
// PresetID is set when the selection is a saved Preset rather than an
// ad-hoc custom range.
type LastSelection struct {
	FromDate *time.Time
	ToDate   *time.Time
	PresetID *uuid.UUID
}

// Preset is a named, reusable date range a member saved for themselves.
type Preset struct {
	ID       uuid.UUID
	Name     string
	FromDate time.Time
	ToDate   time.Time
}

// maxPresetsPerTeamUser bounds an otherwise-unbounded per-(team,user)
// collection -- a generous safety rail, not a real constraint on the
// feature's intended use (a handful of named seasons/ranges).
const maxPresetsPerTeamUser = 20

// ErrTooManyPresets is returned by Service.CreatePreset when the caller
// already has maxPresetsPerTeamUser presets saved for this team.
var ErrTooManyPresets = errors.New("maximum number of saved statistics presets reached")

// ErrPresetNotFound is returned by Service.SetLastSelection when the
// caller-supplied PresetID doesn't reference a preset owned by them in this
// team -- prevents a saved selection from ever pointing at another user's
// (or another team's) preset.
var ErrPresetNotFound = errors.New("preset not found")
