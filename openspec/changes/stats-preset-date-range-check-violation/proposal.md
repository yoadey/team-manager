## Why

`stats_view_presets` (migration `00030_stats_view_preferences.sql`) has a DB
`CHECK (to_date >= from_date)` constraint. `UpdateStatsPresetRequest`'s
`from`/`to` fields are both optional (a PATCH), so a single-bound request can
invert a preset's stored range — e.g. a preset with
`from=2026-01-01,to=2026-06-01` PATCHed with only `{"from":"2026-07-01"}`.
`backend/internal/statsprefs/handler.go`'s `UpdateStatsPreset` only validates
ordering when *both* bounds arrive in the same PATCH (mirroring
`absences/handler.go`'s identical both-fields-present-only check, per that
handler's own comment) — a single-bound PATCH that inverts the range is left
entirely to the DB constraint to catch.

Unlike the sibling `backend/internal/absences/repository.go`'s `Update`
method, which explicitly detects the CHECK-violation SQLSTATE (`23514`,
`pgCheckViolation`) and maps it to a typed `ErrInvalidDateRange` →
`apierror.BadRequest` (400), `backend/internal/statsprefs/repository.go`'s
`UpdatePreset` never checks for this SQLSTATE — the violation surfaces as a
generic wrapped error, which `Handler.UpdateStatsPreset` has no case for, so
it falls through to a logged `apierror.Internal("failed to update stats
preset")` (500). A client mistake (an unremarkable, foreseeable PATCH shape)
gets treated and logged as a server fault, which also pollutes error-rate
alerting/dashboards with client-caused noise.

## What Changes

- `statsprefs/repository.go`'s `UpdatePreset` detects a Postgres CHECK
  violation (SQLSTATE `23514`) on the `UPDATE stats_view_presets` statement
  and returns a new typed `statsprefs.ErrInvalidDateRange`, mirroring
  `absences.ErrInvalidDateRange`/its detection code exactly (same SQLSTATE
  constant name and `errors.As(err, &pgErr)` shape; `stats_view_presets` has
  only the one CHECK, so no constraint-name branch like absences' span-vs-
  range distinction is needed).
- `statsprefs/handler.go`'s `UpdateStatsPreset` gains a case mapping
  `ErrInvalidDateRange` to `apierror.BadRequest` (400), placed alongside its
  existing `pgx.ErrNoRows` → 404 case, before the generic `apierror.Internal`
  fallback.
- No OpenAPI, migration, or frontend changes — this only fixes an
  already-specified but under-implemented error path (the DB constraint and
  the optional PATCH fields are both pre-existing).

## Capabilities

### Modified Capabilities
- `stats-view-preferences`: a single-bound `UpdateStatsPreset` PATCH that
  would invert a preset's stored date range now fails with 400 Bad Request
  instead of 500 Internal Server Error.

## Impact

- `backend/internal/statsprefs/repository.go` (`UpdatePreset`) + test.
- `backend/internal/statsprefs/handler.go` (`UpdateStatsPreset`) + test.
- No API contract change (`openapi.yaml` unchanged — `UpdateStatsPresetRequest`
  already documents `from`/`to` as optional; only the response status for
  this specific bad input changes from 500 to 400).
