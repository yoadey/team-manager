package events

import "time"

// computeEffectiveAttendance resolves a member's attendance for an event,
// mirroring the frontend mock reference implementation's effectiveStatus()
// (frontend/src/services/serviceLayer.ts): an explicit attendance record
// always wins -- its Absent flag still reflects a live absence-overlap
// check, so a member who already responded but has since logged an
// overlapping planned absence is still flagged absent. Otherwise, a member
// whose planned absence covers the event's date defaults to an automatic
// "no". Otherwise, an opt_out event defaults a non-responder to an
// automatic "yes". Otherwise the member is "pending".
//
// The synthetic auto rows intentionally carry no reason/reasonId/
// reasonVisibility: the UI renders a dedicated, localized banner for
// Auto=true (events.autoOptOut/autoAbsent) and a dedicated localized
// "absent" badge for Absent=true (events.absent) instead of the raw reason
// field, so there is no user-facing text to synthesize here -- doing so
// would mean hardcoding a single-locale string into an API response.
func computeEffectiveAttendance(
	explicitStatus, reason, reasonID, reasonVisibility *string, at *time.Time,
	absenceCovers bool, responseMode string,
) EffectiveAttendance {
	if explicitStatus != nil {
		return EffectiveAttendance{
			Status:           *explicitStatus,
			Reason:           reason,
			ReasonId:         reasonID,
			ReasonVisibility: reasonVisibility,
			At:               at,
			Auto:             false,
			Absent:           absenceCovers,
		}
	}
	if absenceCovers {
		return EffectiveAttendance{Status: "no", Auto: true, Absent: true}
	}
	if responseMode == "opt_out" {
		return EffectiveAttendance{Status: "yes", Auto: true}
	}
	return EffectiveAttendance{Status: "pending"}
}
