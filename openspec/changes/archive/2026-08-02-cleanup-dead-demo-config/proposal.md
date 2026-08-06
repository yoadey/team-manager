## Why

`frontend/src/config.ts` still parses `VITE_STORAGE_KEY_PREFIX` and
`VITE_MOCK_DELAY_MIN`/`VITE_MOCK_DELAY_MAX` into `config.storageKeyPrefix`/
`mockDelayMin`/`mockDelayMax`, but nothing reads those fields anymore —
`grep` across `frontend/src` finds no other consumer. They're leftovers
from the localStorage-backed mock that `replace-mock-with-msw` removed;
the current MSW demo backend (`src/mocks/handlers.ts`) uses its own fixed
5 ms delay and doesn't namespace anything by a storage-key prefix. Left in
place, they mislead anyone configuring `frontend/.env` into thinking these
variables still affect demo behavior (this was caught while correcting
CLAUDE.md's env-var table, which had the same drift).

## What Changes

- Remove `storageKeyPrefix`, `mockDelayMin`, `mockDelayMax` and their env
  parsing from `config.ts`.
- Remove the corresponding lines from `frontend/.env.example`.
- Remove the now-obsolete assertions in `config.test.ts`.
- Remove the two now-nonexistent rows from CLAUDE.md's frontend env-var
  table (they currently read "unused" — this change deletes them
  entirely once the code is gone).

## Capabilities

### Added Capabilities
- `demo-mode`: no environment variable is parsed into application config
  unless some code path actually reads it.

## Impact

- `frontend/src/config.ts`, `frontend/src/config.test.ts`,
  `frontend/.env.example`, `CLAUDE.md`.
- No behavior change: both variables are already dead reads.
