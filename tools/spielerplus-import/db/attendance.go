package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/yoadey/team-manager/tools/spielerplus-import/spielerplus"
)

// attendanceStatus maps SpielerPlus's participation-group codes (see
// spielerplus.ParticipationStatus) onto Teamverwaltung's `attendance.status`
// CHECK ('yes','no','maybe','pending','not_nominated') - confirmed 1:1 from
// a HAR capture of a live ajaxgetparticipation response, including
// SpielerPlus's own "Nicht nominiert" group mapping directly onto
// Teamverwaltung's not_nominated.
func attendanceStatus(p spielerplus.ParticipationStatus) string {
	switch p {
	case spielerplus.ParticipationAccepted:
		return "yes"
	case spielerplus.ParticipationDeclined:
		return "no"
	case spielerplus.ParticipationUnsure:
		return "maybe"
	case spielerplus.ParticipationNotNominated:
		return "not_nominated"
	default: // ParticipationNoResponse, or anything unrecognized
		return "pending"
	}
}

// UpsertAttendance writes one member's attendance status (and optional
// reason) for one event. Naturally idempotent on the DB's own
// UNIQUE(event_id, user_id) - no state file needed (see design.md).
func (s *Store) UpsertAttendance(ctx context.Context, eventID, userID string, status spielerplus.ParticipationStatus, reason string) error {
	if s.DryRun {
		return nil
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO attendance (id, event_id, user_id, status, reason)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id, user_id) DO UPDATE SET status = EXCLUDED.status, reason = EXCLUDED.reason
	`, uuid.NewString(), eventID, userID, attendanceStatus(status), nullIfEmpty(reason))
	if err != nil {
		return fmt.Errorf("db: upsert attendance (event %s, user %s): %w", eventID, userID, err)
	}
	return nil
}
