## Why

Teams that share facilities want to see each other's schedule to avoid clashes — but only the *when* and *where*, not the internals. There's currently no way for team A to let team B see its calendar, and the event payload carries far more than a scheduling view should expose (attendance, comments, notes).

## What Changes

- Let a team **grant read-only calendar visibility** to another team (managed by someone with `settings` write on the sharing team).
- A grantee team's members get a **redacted** view of the shared calendar: each event's **time and location** (and title/type), and **nothing else** — no attendance, no participants, no comments, no notes.
- This is one-directional per grant and does not make the grantee a participant (contrast `cross-team-events`, where members merge and RSVP).

## Capabilities

### New Capabilities
- `shared-calendar`: read-only, redacted cross-team visibility of a team's event schedule.

## Impact

- Spec/backend: `openapi.yaml` — endpoints to grant/revoke/list calendar shares (settings-gated) and a read endpoint returning a **redacted** event projection (`SharedCalendarEvent`: time, location, title, type only); a `calendar_shares(owner_team_id, viewer_team_id)` table + migration; authorization allowing a viewer-team member to read only the redacted projection of a granting team; regenerate clients; tests.
- Frontend: a settings UI to manage which teams a calendar is shared with; a way to view shared calendars (overlay/section) rendering only redacted fields; `frontend/src/i18n/{de,en}.ts`; MSW.
- CI: openapi-drift, migration gates, backend + frontend gates. **API + migration + RBAC-affecting.**
