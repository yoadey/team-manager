package db

import (
	"testing"

	"github.com/yoadey/team-manager/tools/spielerplus-import/spielerplus"
)

func TestEventType(t *testing.T) {
	cases := map[spielerplus.EventType]string{
		spielerplus.EventTraining:   "training",
		spielerplus.EventGame:       "auftritt",
		spielerplus.EventTournament: "auftritt",
		spielerplus.EventOther:      "event",
	}
	for in, want := range cases {
		if got := eventType(in); got != want {
			t.Errorf("eventType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAttendanceStatus(t *testing.T) {
	cases := map[spielerplus.ParticipationStatus]string{
		spielerplus.ParticipationAccepted:     "yes",
		spielerplus.ParticipationDeclined:     "no",
		spielerplus.ParticipationUnsure:       "maybe",
		spielerplus.ParticipationNoResponse:   "pending",
		spielerplus.ParticipationNotNominated: "not_nominated",
	}
	for in, want := range cases {
		if got := attendanceStatus(in); got != want {
			t.Errorf("attendanceStatus(%q) = %q, want %q", in, got, want)
		}
	}
}
