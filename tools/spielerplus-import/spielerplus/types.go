// Package spielerplus scrapes a logged-in SpielerPlus session for the data
// needed to migrate a club to Teamverwaltung. SpielerPlus has no public API;
// this client talks to the same server-rendered pages a browser would.
package spielerplus

import "time"

// ParticipationStatus is SpielerPlus's own attendance vocabulary, read off
// the "selected" participation button's title attribute on an event page.
type ParticipationStatus string

const (
	ParticipationAccepted  ParticipationStatus = "accepted" // "Zugesagt"
	ParticipationUnsure    ParticipationStatus = "unsure"   // "Unsicher"
	ParticipationDeclined  ParticipationStatus = "declined" // "Absagen / Abwesend"
	ParticipationNoResonse ParticipationStatus = "no_response"
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
}

// Attendance is one member's participation status for one event.
type Attendance struct {
	EventID  string
	MemberID string
	Status   ParticipationStatus
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
