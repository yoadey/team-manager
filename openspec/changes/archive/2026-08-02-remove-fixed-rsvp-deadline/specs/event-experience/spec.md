## REMOVED Requirements

### Requirement: Configurable RSVP deadline with a countdown
Superseded by the `event-cancellation` capability's lead-time-based cutoff (`cancelLeadMinutes`), which covers recurring series correctly and is the sole cutoff mechanism going forward.

Reason: redundant with `event-cancellation`'s lead-time cutoff; kept both mechanisms only confused organizers and doubled enforcement/UI surface.
Migration: events/series with an `rsvpDeadline` set lose that cutoff; organizers who want one must set a cancellation lead time (`cancelLeadMinutes`) instead.
