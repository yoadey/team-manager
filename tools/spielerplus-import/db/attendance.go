package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/tools/spielerplus-import/spielerplus"
)

// attendanceStatus maps SpielerPlus's participation vocabulary onto
// Teamverwaltung's `attendance.status` CHECK
// ('yes','no','maybe','pending','not_nominated'). This importer never
// produces 'not_nominated' - that's a Teamverwaltung-specific concept
// (member not selected for the event) with no SpielerPlus equivalent.
func attendanceStatus(p spielerplus.ParticipationStatus) string {
	switch p {
	case spielerplus.ParticipationAccepted:
		return "yes"
	case spielerplus.ParticipationDeclined:
		return "no"
	case spielerplus.ParticipationUnsure:
		return "maybe"
	default:
		return "pending"
	}
}

// UpsertAttendance writes one member's attendance status for one event.
// Naturally idempotent on the DB's own UNIQUE(event_id, user_id) - no state
// file needed (see design.md).
func (s *Store) UpsertAttendance(ctx context.Context, eventID, userID string, status spielerplus.ParticipationStatus) error {
	if s.DryRun {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO attendance (id, event_id, user_id, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id, user_id) DO UPDATE SET status = EXCLUDED.status
	`, uuid.NewString(), eventID, userID, attendanceStatus(status))
	if err != nil {
		return fmt.Errorf("db: upsert attendance (event %s, user %s): %w", eventID, userID, err)
	}
	return nil
}
