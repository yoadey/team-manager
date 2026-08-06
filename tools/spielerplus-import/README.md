# spielerplus-import

A standalone, one-off migration tool that imports a club's members, events
(trainings/games), attendance, and planned absences (including past ones)
from [SpielerPlus](https://www.spielerplus.de) into Teamverwaltung, ahead of
real users logging in for the first time.

This is its own Go module, separate from the main `backend`/`frontend`
modules and not part of their build or CI - see
`openspec/changes/spielerplus-data-migration/` (in the repo root) for the
full design and rationale.

## Important caveats before you run this

- **SpielerPlus has no public API.** This tool scrapes the same
  server-rendered HTML pages/ajax fragments a browser would. All four data
  sources (events, attendance, members, absences) are now grounded in real
  HAR captures that included response bodies, not guesses:
  - **Events and attendance** (`spielerplus/events.go`,
    `spielerplus/attendance.go`): the events list (`GET /events` plus
    `POST /events/ajaxgetevents`, JSON-enveloped, to page into history) and
    per-event attendance (`POST /events/ajaxgetparticipation`, also
    JSON-enveloped) match confirmed real markup, including that
    `ajaxgetparticipation` does return every team member's status grouped
    by status code (not just the logged-in user's own). Only "training"
    events were present in the capture (game/tournament/event types are
    still a guess) and no location field was observed.
  - **Members** (`spielerplus/members.go`): `GET /team` is the roster, but
    it does **not** show email addresses - each member's email only shows
    on their own profile page (`GET /user/view?id=...`), so the importer
    fetches one profile page per member. Only the account owner's own
    profile was captured, so it's unverified whether viewing *other*
    members' profiles shows their email the same way (a member with no
    visible email is skipped and logged, not imported without one).
  - **Absences** (`spielerplus/absences.go`): `GET /absence` renders every
    absence type as tabs in one page load; the importer reads the
    type-specific tabs (which together hold the full history) and skips the
    "Aktuell" (current) tab, which is a filtered subset. The "1 Tag pro
    Woche" (weekly-recurring) absence type had no real examples in the
    capture, so a recurring entry is expanded by weekday only if one can be
    detected in its reason text (e.g. "montags") - otherwise it's imported
    as a single literal date range and logged, rather than guessed wrong.
  - If a page's expected elements still aren't found on your account (a
    genuinely different SpielerPlus layout, a different display language,
    etc.), the tool fails loudly with an error naming the selector to fix,
    rather than silently importing nothing.
- **This writes directly to the Teamverwaltung database**, bypassing the
  backend's HTTP API. Always run with `--dry-run` first, and test against a
  disposable/staging database before running against production.

## Setup

1. **Capture a SpielerPlus session cookie.** Log into
   `https://www.spielerplus.de` in your browser, open DevTools -> Network
   (or Application -> Cookies), and copy the full `Cookie:` header value (or
   concatenate the relevant `name=value` pairs) sent with a request to
   spielerplus.de. Your SpielerPlus password is never entered into this
   tool.
2. **Create the target team in Teamverwaltung first** (through the normal
   UI), along with any roles you want to map SpielerPlus roles onto (e.g. a
   "Trainer" role with `events:write`).
3. Copy `role-mapping.example.yaml` and adjust it to your team's roles.
4. Set the required environment variables:

   | Variable                    | Purpose                                                        |
   |------------------------------|-----------------------------------------------------------------|
   | `DATABASE_URL`               | Same Postgres DSN as the Teamverwaltung backend                 |
   | `TEAM_ID`                    | UUID of the already-created target team                         |
   | `SPIELERPLUS_SESSION_COOKIE` | The cookie header value captured above                          |
   | `ROLE_MAPPING_PATH`          | Path to your role-mapping YAML file                             |
   | `STATE_PATH`                 | Optional; defaults to `./spielerplus-import-state.json`         |
   | `SPIELERPLUS_REQUEST_DELAY`  | Optional; minimum gap between requests to SpielerPlus, as a Go duration (e.g. `1s`). Defaults to `500ms`. Set `0` to disable throttling entirely (not recommended - a full import issues one request per member's profile and per event's attendance, so this is what keeps the tool from hammering SpielerPlus in a tight loop). |
   | `SPIELERPLUS_EXPECTED_TEAM_NAME` | Optional; see "Which SpielerPlus team gets imported?" below. |

## Which SpielerPlus team gets imported?

SpielerPlus scopes every page this tool reads (events, attendance, roster,
absences) by whichever team is currently *active* in your SpielerPlus
session - there's no team id in any of the URLs to double check against, so
an account that manages more than one team could have the wrong one active
when its session cookie was captured.

Before writing anything, the tool fetches the active team's display name
(shown in SpielerPlus's own sidebar "Team wechseln" card) and:

- if `SPIELERPLUS_EXPECTED_TEAM_NAME` is set, compares it directly and
  aborts on a mismatch, without prompting - useful for repeat/non-interactive
  runs once you've confirmed the name once;
- otherwise, prints the detected name and asks for an explicit `y`/`yes` on
  stdin before continuing.

If it's the wrong team, switch at
[`https://www.spielerplus.de/site/select-team`](https://www.spielerplus.de/site/select-team)
in your browser, capture a fresh session cookie from that session, and start
over.

## Usage

```sh
cd tools/spielerplus-import
go build ./...

# Always start with a dry run: scrapes and validates everything, writes nothing.
DATABASE_URL=... TEAM_ID=... SPIELERPLUS_SESSION_COOKIE=... ROLE_MAPPING_PATH=./role-mapping.yaml \
  go run . --dry-run

# Once the dry-run summary looks right:
DATABASE_URL=... TEAM_ID=... SPIELERPLUS_SESSION_COOKIE=... ROLE_MAPPING_PATH=./role-mapping.yaml \
  go run .
```

The tool prints a summary of created/skipped records per entity type, and a
list of skip reasons (e.g. an absence that overlapped an existing one, or
attendance for a member not found on the imported roster).

It's safe to run more than once: users are deduplicated by email,
attendance by the database's own per-event-per-user uniqueness, and events/
absences via a local state file (`STATE_PATH`) mapping SpielerPlus IDs to
Teamverwaltung IDs - keep that file around between runs.

## What imported users can do afterwards

Each imported member is matched to a Teamverwaltung account **by email**: if
one already exists (created by any means - a previous import run, an invite,
self-registration) it's reused as-is and just gets a membership in the target
team if it doesn't have one yet; only members with no matching account get a
new one. A newly created account has no usable password but an
already-verified email, so its owner can log in by using **"forgot
password"** with their email on the normal Teamverwaltung login page - no
self-registration needed. This matching is why fetching each member's email
from their profile page (see the members caveat above) matters even for
people who might already have an account.

## Testing

```sh
go test ./...
```

HTML parsing (`spielerplus/`) is tested against inline HTML fixtures. Once
you've captured real markup while fixing the members/absences selectors
(see caveats above), consider adding it as a fixture-backed test case too.
