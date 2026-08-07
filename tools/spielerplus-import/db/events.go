package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/tools/spielerplus-import/spielerplus"
)

// eventType maps SpielerPlus's event classification onto Teamverwaltung's
// `events.type` CHECK ('training', 'auftritt', 'event') - 'auftritt' is the
// sport-neutral term Teamverwaltung uses for a fixture/competition
// appearance (game/match/tournament), 'event' is the catch-all.
func eventType(t spielerplus.EventType) string {
	switch t {
	case spielerplus.EventTraining:
		return "training"
	case spielerplus.EventGame, spielerplus.EventTournament:
		return "auftritt"
	default:
		return "event"
	}
}

// InsertEvent creates a Teamverwaltung event for teamID from a scraped
// SpielerPlus event. Not idempotent by itself - callers must consult the
// local state file (see mapping.State) to avoid re-inserting an event
// that's already been imported, since `events` has no external-id column.
func (s *Store) InsertEvent(ctx context.Context, teamID string, ev spielerplus.Event) (id string, err error) {
	if s.DryRun {
		return dryRunID, nil
	}

	newID := uuid.NewString()
	var startTime, endTime, meetTime any
	// A page that gave no time information at all has a meaningless
	// midnight Start/End placeholder (see spielerplus.parseDateTime) -
	// leave both columns NULL rather than importing a fake 00:00-02:00 slot.
	if !ev.TimeUnknown {
		if !ev.Start.IsZero() {
			startTime = ev.Start.Format("15:04:05")
		}
		if !ev.End.IsZero() {
			endTime = ev.End.Format("15:04:05")
		}
	}
	if !ev.MeetTime.IsZero() {
		meetTime = ev.MeetTime.Format("15:04:05")
	}

	_, err = s.Pool.Exec(ctx, `
		INSERT INTO events (id, team_id, type, title, date, location, meet_time, start_time, end_time, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())
	`, newID, teamID, eventType(ev.Type), ev.Title, ev.Start.Format("2006-01-02"), nullIfEmpty(ev.Location), meetTime, startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("db: insert event %q (%s): %w", ev.Title, ev.ID, err)
	}
	return newID, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
