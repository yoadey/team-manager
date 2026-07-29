## Context

Today's data-access layer is a deliberate three-way split from the sqlc adoption (`openspec/changes/archive/2026-07-18-adopt-sqlc`):

| Pattern | Modules | Notes |
|---|---|---|
| sqlc-generated (`internal/db/gen`) | `news`, `roles` (static queries only) | 2/14 |
| `internal/db/sqlbuilder` (dynamic `SET`) | `events`, `finances`, `roles` | introduced specifically to remove the no-op-fallback SET-builder bug |
| Raw hand-written pgx | remaining 11 modules, 35 files total | never migrated (`members`/`polls` explicitly deferred as follow-up; `events`/`finances` intentionally excluded from sqlc per that change's own design) |

Total repository-layer surface: 14 `repository.go` files, ~6,760 lines. `fmt.Sprintf`/`sqlbuilder.` occurrence counts per module (a rough proxy for dynamic-query risk) are 0 for `stats`/`notifications`/`push`/`calendarfeed`, 1–3 for `polls`/`auth`/`absences`/`calendarshare`, 7–10 for `members`/`teams`, and heaviest (already on `sqlbuilder`) for `events`/`finances`.

The team has decided the value of one consistent pattern outweighs sqlc's compile-time query checking. This design's job is to make sure that decision doesn't reopen the two concrete bugs the sqlc change was written to close.

## Goals / Non-Goals

**Goals:**
- One data-access pattern (GORM) across all 14 repositories; no leftover sqlc-generated code or `sqlbuilder` usage after this change is fully rolled out.
- **Structural**, not just conventional, tenant-scoping enforcement: a team-scoped query issued without an explicit team filter must fail (test failure at minimum, ideally at query-build time), not silently return cross-team rows.
- **Structurally remove** the original SET-builder bug class: no repository code manually tracks a SQL placeholder index. Dynamic updates are built as an explicit `map[string]any` of only the changed columns, mirroring `sqlbuilder.Builder`'s "no implicit fallback" contract but relying on GORM's own parameter binding instead of hand-rolled `$N` arithmetic.
- No `fmt.Sprintf`-composed SQL text with variable data anywhere in the migrated code.
- Preserve existing operational behavior: pool sizing/limits, the slow-query tracer, and the Prometheus pool-stats collector must keep working exactly as they do today — the migration must not force a switch to a differently-instrumented connection pool.
- Keep the exact regression intent of `TestEventRepository_UpdateEvent_Series_OnlyDateSet_DoesNotCorruptSQL` alive, ported to GORM.

**Non-Goals:**
- GORM `AutoMigrate` is not adopted. Goose + `internal/db/migrations/*.sql` remain the single schema source of truth; GORM struct tags describe existing tables, they don't generate them.
- Not a multi-database-engine migration. Postgres remains the only target; this change does not attempt DB portability.
- No rewrite of `service.go`/`handler.go` business logic beyond the call-site changes forced by new repository method signatures/error types.
- River's own DB access (`riverpgxv5`, a separate pgx pool consumer for the job queue) is untouched — it doesn't go through repositories and isn't part of this migration.

## Decisions

### 1. Driver wiring: GORM on top of the existing `*pgxpool.Pool`, not a separate connection pool
`gorm.io/driver/postgres` normally opens its own `database/sql.DB`. Instead, reuse the pattern `internal/db.RunMigrations` already uses for goose:

```go
sqlDB := stdlib.OpenDBFromPool(pool)         // *pgxpool.Pool -> *sql.DB, same pattern as RunMigrations
gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
```

This keeps `internal/db.Connect`'s pool config (`MaxConns: 25`, `MinConns: 2`, lifetime/idle settings) and — critically — the `SlowQueryTracer` (`internal/db/tracer.go`, a `pgx.QueryTracer` attached to `cfg.ConnConfig.Tracer`) and the Prometheus `poolStatsCollector` (`internal/db/metrics.go`, reads `pgxpool.Stat()`) working unchanged, because every query GORM issues still ultimately runs through the same `*pgxpool.Pool` connections. No new metrics/tracing code is needed.

### 2. Tenant scoping: a registered callback, not just a convention
sqlc's guarantee was that `AND team_id = $N` was reviewable, static text in a `.sql` file. GORM queries are built at runtime, so the equivalent guarantee has to be structural:

- `internal/db/gormx` defines a `TeamScoped` interface (`TeamID() uuid.UUID` via an embedded `TeamID uuid.UUID` field) that every team-scoped GORM model embeds.
- `gormx.ForTeam(teamID uuid.UUID) func(*gorm.DB) *gorm.DB` is the **only** sanctioned way to add team scoping to a query; repositories call `db.Scopes(gormx.ForTeam(teamID))` on every by-id/by-team query.
- `gormx.RegisterScopeGuard(gdb *gorm.DB)` registers `Before("gorm:query")`/`Before("gorm:update")`/`Before("gorm:delete")` callbacks that inspect `stmt.Model` — if the model implements `TeamScoped` and the built statement's `WHERE` clause doesn't reference `team_id`, the callback sets `stmt.Error` (aborting the query) instead of letting it run unscoped. This turns "forgot the scope" into a runtime error surfaced by the first integration test that exercises the code path, rather than a silent cross-tenant leak.
- Every migrated repository keeps (or gains) the cross-team-isolation test already required by the `data-access` spec's "Tenant scoping preserved" scenario — now asserting against the GORM-based repository instead of the sqlc/pgx one.

### 3. Dynamic updates: explicit column maps, never struct-based `.Save()`/`.Updates(struct)`
GORM's struct-based partial update (`db.Model(&row).Updates(patchStruct)`) silently skips Go zero-values (`""`, `0`, `false`, zero `time.Time`) — a *different* bug shape that can just as easily corrupt data as the original `SET id = $N` fallback (a caller intentionally clearing a field to `""` would see the change silently dropped). The replacement:

```go
p := gormx.NewPatch()
if req.Title != nil { p.Set("title", *req.Title) }
if req.Pinned != nil { p.Set("pinned", *req.Pinned) }
if p.Empty() { return ErrNothingToUpdate }   // explicit, same contract as sqlbuilder.Builder.Empty()
err := db.Scopes(gormx.ForTeam(teamID)).Model(&Row{}).Where("id = ?", id).Updates(p.Values()).Error
```

`gormx.Patch` is a thin wrapper around `map[string]any` — it exists only to (a) keep the same "explicit empty-set signal, no fallback" contract `sqlbuilder.Builder` had, and (b) give call sites a single, greppable pattern for partial updates so a reviewer can check "is this a whitelisted map, not a struct" at a glance. No index arithmetic exists anywhere in this path — GORM's own placeholder binding makes the original bug (wrong-column binding via a shared running counter) structurally unreachable, not just tested against.

### 4. Type mapping
- `uuid.UUID` (`github.com/google/uuid`) already implements `sql.Scanner`/`driver.Valuer` (`func (uuid *UUID) Scan(src any) error` / `func (uuid UUID) Value() (driver.Value, error)`) — confirmed via `go doc`. No custom GORM type needed; existing `Row` structs (e.g. `teams.TeamRow`, `roles.RoleRow`) can gain `gorm:"..."` tags with minimal change.
- `time.Time` for `timestamptz` columns: handled natively by `database/sql`/pgx, no override needed.
- `roles.PermissionsJSON` (JSONB): gains `Scan`/`Value` methods (`encoding/json` marshal/unmarshal to `[]byte`), replacing the sqlc column-type override that previously did the equivalent mapping at the codegen layer.
- Not-found translation: sqlc/pgx code translates `pgx.ErrNoRows` to domain sentinel errors (pattern used in 57 files across repositories/services/handlers/tests). GORM returns `gorm.ErrRecordNotFound` instead — every migrated repository's translation point changes from `errors.Is(err, pgx.ErrNoRows)` to `errors.Is(err, gorm.ErrRecordNotFound)`; the domain sentinel errors themselves (and everything above the repository layer) are unchanged.

### 5. No associations/preload for cross-table reads
GORM's `Preload`/association auto-loading is a common source of N+1 queries and hides the actual SQL shape. Decision: don't use them. Multi-table reads (e.g. `stats` attendance aggregation, `events` series + attendance + comments) are written as explicit `Joins(...)`/raw `Select` queries through the GORM builder, kept just as readable/reviewable as today's hand-written pgx SQL — GORM is adopted for its builder/binding/scoping ergonomics, not its object-graph loading.

### 6. Rollout order (risk-ascending, both stacks coexist mid-migration)
1. **Pilot** — `news`, `roles` (already sqlc; smallest, lowest dynamic-query surface; proves the scoping callback + `Patch` helper + type mapping before touching untouched modules).
2. **Wave 2** — `stats`, `notifications`, `push`, `calendarfeed` (zero `fmt.Sprintf`/`sqlbuilder` hits today — closest to pure static CRUD).
3. **Wave 3** — `polls`, `auth`, `absences`, `calendarshare` (1–3 dynamic-construction hits each).
4. **Wave 4** — `members`, `teams` (7–10 hits each; `members` was explicitly deferred in the original sqlc rollout for this reason).
5. **Wave 5** — `finances`, `events` (heaviest dynamic surface, already on `sqlbuilder` today; carries the original `DoesNotCorruptSQL` regression test, ported here as the hard gate for this wave).

sqlc/`sqlbuilder` stay installed and functional throughout waves 1–4 so CI is green after every module; they're only removed in the cleanup task once wave 5 (their only remaining consumers) is done.

### 7. Coverage/lint treatment
Unlike `internal/gen`/`internal/db/gen`, `internal/db/gormx` and the migrated repositories are hand-written, reviewed code, not generated output — they stay **inside** the coverage gate (no exclusion), same as any other `internal/<module>` package today.

## Risks / Trade-offs

- **Loss of compile-time query checking.** sqlc caught a typo'd column name at `make generate` time; GORM catches the same mistake only via test execution. Mitigated by requiring integration-test coverage for every migrated repository method (already implied by the existing coverage gate) and by keeping `gorm:"column:..."` struct tags as the single, reviewable source of column names, checked against `internal/db/migrations`.
- **GORM footguns beyond the two addressed above** (soft-delete conventions via `DeletedAt`, global scopes interacting unexpectedly with `gormx.ForTeam`, `Session(&gorm.Session{})` reuse pitfalls) are real but not exercised by this codebase's schema (no soft-delete columns exist today) — flagged here so a future change doesn't introduce one without re-reading this design.
- **Dependency-weight**: `openspec/config.yaml` states the project is "deliberately dependency-light; justify new runtime deps." `gorm.io/gorm` + `gorm.io/driver/postgres` are a materially larger addition than sqlc (a build-time-only tool with zero runtime footprint). This is the team's explicit, informed trade-off for this change — recorded here as the required justification, not something this change tries to minimize away.
- **Big-bang exposure across 14 modules** mitigated by the risk-ascending, coexisting-stacks rollout in Decision 6 — each wave is independently shippable/revertable, and `events`/`finances` (the modules with actual production-corruption history) are migrated last, with the most test scrutiny.
- **River queue** uses its own `riverpgxv5` pool/driver, untouched by this change — called out explicitly so a reviewer doesn't assume it needs migrating too.
