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
  server-rendered HTML pages/ajax fragments a browser would. The endpoints
  below are confirmed from a real HAR capture (`GET`/`POST` URLs, form
  fields, headers) - but that capture didn't include response *bodies*, so
  the actual HTML/CSS structure each page returns is still unverified, and
  the row selectors in `spielerplus/*.go` remain best-effort guesses that
  will likely need adjusting against your real account before a run
  succeeds. If a page's expected elements aren't found, the tool fails
  loudly with an error naming the selector to fix, rather than silently
  importing nothing.
  - Confirmed: login (`POST /site/login`), the events list (`GET /events`
    plus `POST /events/ajaxgetevents` to page into history), per-event
    attendance (`POST /events/ajaxgetparticipation`), the absences list
    (`GET /absence`), and the roster (`GET /team`, though a `GET /site/team`
    was also seen and might be the real one instead).
  - **Important open question**: `ajaxgetparticipation` was only observed
    being called from a single training's own detail page, so it's
    unconfirmed whether it returns *every* team member's status (what this
    importer needs) or only the logged-in user's own. If it turns out to be
    self-only, full-roster attendance needs a different source - check this
    first before trusting attendance import results.
  - If you can capture a **HAR with response bodies included** (in Chrome
    DevTools: right-click the Network tab's request list -> "Save all as
    HAR with content", not just "Save all as HAR") while browsing the
    roster, an absence's detail view, and a training's participation list,
    that would let the actual selectors be filled in with confidence
    instead of guessed.
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

Each imported member gets a Teamverwaltung account (matched by email) with
no usable password but an already-verified email. They can log in by using
**"forgot password"** with their email on the normal Teamverwaltung login
page - no self-registration needed.

## Testing

```sh
go test ./...
```

HTML parsing (`spielerplus/`) is tested against inline HTML fixtures. Once
you've captured real markup while fixing the members/absences selectors
(see caveats above), consider adding it as a fixture-backed test case too.
