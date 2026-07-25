## 1. Frontend
- [x] 1.1 `EventCard` (`components/cards.tsx`) now renders inline accept/maybe/decline icon buttons (same icons/colors as `EventDetailSheet`), highlighting the member's current status via `aria-pressed`
- [x] 1.2 Wired to the existing `setMyStatus` action (the shared `useSetAttendanceMutation` path used by the detail sheet); no new endpoint
- [~] 1.3 Controls are hidden for past/cancelled events and for non-nominated members; server rejection already surfaces via the shared action's toast. The **cancellation-cutoff** gating is deferred to `event-cancellation-lead-time` (that field doesn't exist yet) — noted, not implemented here
- [x] 1.4 Icon buttons carry `aria-label`s (reused `events.rsvpYes/Maybe/No`), grouped under `role="group"` labelled by the event title (new key `events.rsvpGroupLabel` in `de`/`en`)

## 2. Tests
- [x] 2.1 `cards.test.tsx`: clicking the inline "Zusagen" icon calls `setMyStatus('ev1','yes', …)` (mock-backed); mock extended with `setMyStatus`
- [x] 2.2 Past-event row shows no inline controls; current status reflected via `aria-pressed`. (Cutoff-disabled case deferred with 1.3.)

## 3. Verification
- [x] 3.1 `npm run lint` (0 errors) + `npm run typecheck` green
- [x] 3.2 `npm test` green — `cards.test.tsx` 20/20; full suite's only failures are the unrelated intermittent `a11y.test.tsx` axe timeouts under load (pass in isolation)
- [x] 3.3 `npm run build` + `check:bundle` within budget (254.4 KB total); RSVP icon buttons are labelled (a11y EventsPage case renders an empty list, unaffected)
