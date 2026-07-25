## Context

Events store team-local wall-clock date/start/end. Attendance is set via `POST .../attendance` (`SetAttendanceRequest`). Series exist (recurring events). There is no deadline field today, so this adds one; expressing it as a relative lead time (not an absolute instant) lets a single series definition apply the same cutoff to every occurrence.

## Goals / Non-Goals

**Goals:**
- Organizers set the cutoff once, as "N hours M minutes before start".
- Server rejects late self-service RSVP changes; the effective absolute cutoff is derived per occurrence from its start.

**Non-Goals:**
- Per-member exceptions or waitlists.
- Changing how RSVP itself works (see `inline-event-rsvp`).

## Decisions

- Store a single integer `cancelLeadMinutes` (nullable = no cutoff). UI splits it into hours + minutes; the API stays one field.
- Effective cutoff = event start instant − `cancelLeadMinutes`. Enforcement lives in `events.Service` on attendance mutation: reject self-service changes past the cutoff with a 409/422 problem+json; callers with `write` on `events` bypass.
- Series carry the field on the series definition; each generated occurrence inherits it and computes its own cutoff.
- Frontend disables RSVP controls and shows the cutoff once passed; keeps the server as source of truth.

## Risks / Trade-offs

- Timezone correctness: the cutoff must be computed from the event's absolute start instant (Europe/Berlin wall-clock → UTC), reusing existing date helpers, not the browser's local reinterpretation.
- Backward data: existing events get `NULL` (no cutoff) — no behavior change until an organizer sets one.
- Must not block organizers from correcting attendance after the cutoff — the `write`-on-`events` bypass is required.
