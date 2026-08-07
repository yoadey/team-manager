// Package spielerplus scrapes a logged-in SpielerPlus session for the data
// needed to migrate a club to Teamverwaltung. SpielerPlus has no public API;
// this client talks to the same server-rendered pages a browser would.
package spielerplus

import "time"

// ParticipationStatus is SpielerPlus's own attendance vocabulary. Confirmed
// from a HAR capture of a live ajaxgetparticipation response: members are
// grouped under a `<div id="{code}-parti-collapse">`, one group per status -
// the numeric code (not the group's German label text) is what's matched
// here, so this doesn't depend on the account's display language.
type ParticipationStatus string

const (
	ParticipationAccepted     ParticipationStatus = "1"  // "Zugesagt"
	ParticipationUnsure       ParticipationStatus = "2"  // "Unsicher"
	ParticipationDeclined     ParticipationStatus = "0"  // "Absagen / Abwesend"
	ParticipationNoResponse   ParticipationStatus = "99" // "Noch nicht zu/abgesagt"
	ParticipationNotNominated ParticipationStatus = "3"  // "Nicht nominiert" - maps directly to Teamverwaltung's own not_nominated status
)

// EventType mirrors SpielerPlus's own event type classification.
type EventType string

const (
	EventTraining   EventType = "training"
	EventGame       EventType = "game"
	EventTournament EventType = "tournament"
	EventOther      EventType = "other"
)

// Event is a single SpielerPlus calendar entry.
type Event struct {
	// ID is SpielerPlus's own event id ("typeid" in the participation form),
	// used as the external key for import idempotency.
	ID       string
	Type     EventType
	Title    string
	Location string
	Start    time.Time
	// EndIsEstimated is true when SpielerPlus didn't show an end time and it
	// was estimated (Start + 2h), matching the reference community projects'
	// approach for the same gap in SpielerPlus's markup.
	EndIsEstimated bool
	End            time.Time
	// TimeUnknown is true when the page showed no time information at all
	// for this event (as opposed to a start time with only the end
	// estimated) - Start's time-of-day is a meaningless midnight default in
	// that case and callers should not write it out as a real start time.
	TimeUnknown bool
	// MeetTime is SpielerPlus's separate "Treffen" (meet-up) time, distinct
	// from Start ("Beginn"). Zero if the page didn't show one.
	MeetTime time.Time
}

// Attendance is one member's participation status for one event.
type Attendance struct {
	EventID  string
	MemberID string
	Status   ParticipationStatus
	// Reason is the free-text reason the member gave for a decline/absence,
	// if any (empty otherwise).
	Reason string
}

// Member is a SpielerPlus team member/roster entry.
type Member struct {
	// ID is SpielerPlus's own user id ("user_id" in the participation form).
	ID    string
	Name  string
	Email string
	// Role is the SpielerPlus role name as displayed (e.g. "Trainer",
	// "Co-Trainer", "Spieler"), looked up against the configured role
	// mapping before being written to Teamverwaltung.
	Role string
	// Birthday is the member's date of birth, if shown on their profile page
	// and parseable. Zero if not set/visible.
	Birthday time.Time
	// PhotoURL is the absolute URL of the member's profile photo on
	// SpielerPlus's asset CDN, or "" if they have no custom photo set
	// (SpielerPlus falls back to a generic "default.svg" silhouette in that
	// case, which is deliberately not treated as a photo - see ParseMembers).
	PhotoURL string
}

// Absence is a planned absence entered in SpielerPlus, already expanded to a
// concrete date range (SpielerPlus's "recurring weekday" absences are
// expanded to concrete occurrences by the caller, since Teamverwaltung has
// no recurrence concept for absences).
type Absence struct {
	// ID is SpielerPlus's own absence id, used as the external key for
	// import idempotency. For an occurrence expanded out of a recurring
	// absence, this is synthesized as "<absenceID>:<fromDate>" so each
	// occurrence gets a stable, distinct key.
	ID       string
	MemberID string
	From     time.Time
	To       time.Time
	Reason   string
}
