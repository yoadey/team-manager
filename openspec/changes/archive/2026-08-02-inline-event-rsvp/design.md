## Context

The events overview renders a list of events; RSVP lives only in the detail sheet, which already calls the attendance mutation (`useEventActions`/`useAbsenceActions`/`useEventMutations`). The member's own current status is available on the event/attendance data used by the list.

## Goals / Non-Goals

**Goals:**
- One-tap accept/maybe/decline from the overview, with the current status visible.
- Behavior identical to the detail sheet (same mutation, same cutoff rules).

**Non-Goals:**
- Setting attendance for other members from the overview (that stays in the detail/attendance matrix).
- A new endpoint or payload.

## Decisions

- Add a small icon group (accept / maybe / decline) per event row using the shared attendance-status icons; the active status is highlighted.
- Wire to the existing attendance mutation hook with optimistic update + rollback on error, matching the detail sheet.
- Disable the controls when the event's cancellation cutoff has passed (reads the `cancelLeadMinutes`-derived cutoff from `event-cancellation-lead-time`) and for past events; show the server rejection as a toast.
- Provide accessible labels (icon-only buttons need `aria-label`).

## Risks / Trade-offs

- Row density: three icon buttons per row must stay legible on mobile — coordinate with `mobile-layout-polish`.
- Consistency: reuse the same status icons/colors as the detail view and attendance matrix so the overview doesn't diverge.
- Optimistic updates must reconcile with the query cache the list reads from (TanStack Query) to avoid flicker.
