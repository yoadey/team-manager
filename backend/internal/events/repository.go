package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yoadey/team-manager/backend/internal/attendance"
	"github.com/yoadey/team-manager/backend/internal/db/sqlbuilder"
	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// pgCheckViolation is the Postgres SQLSTATE for a violated CHECK constraint.
const pgCheckViolation = "23514"

// ErrEndTimeBeforeStartTime is returned when a partial UpdateEvent would
// leave end_time <= start_time (violates the events_end_after_start_time
// CHECK constraint). The handler validates this when both fields are present
// in the same request, but a partial update (only one of startTime/endTime)
// can only be caught here, since the merge happens inside the UPDATE
// statement itself -- see absences' identical ErrInvalidDateRange pattern.
var ErrEndTimeBeforeStartTime = errors.New("endTime: must be after startTime")

// Repository handles all event-related DB operations.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ─── helpers ────────────────────────────────────────────────────────────────

// boolVal dereferences a *bool with a false default — prevents NULL on NOT NULL columns.
func boolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// strVal dereferences a *string with a provided default — prevents NULL on NOT NULL columns.
func strVal(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

// uuidSlice coalesces a nil UUID slice to empty — prevents NULL on NOT NULL array columns.
func uuidSlice(ids []uuid.UUID) []uuid.UUID {
	if ids == nil {
		return []uuid.UUID{}
	}
	return ids
}

const selectEventFields = `
	id, team_id, series_id, type, title, date,
	location, note, result,
	COALESCE(TO_CHAR(meet_time, 'HH24:MI'), '') AS meet_time,
	COALESCE(TO_CHAR(start_time, 'HH24:MI'), '') AS start_time,
	COALESCE(TO_CHAR(end_time, 'HH24:MI'), '') AS end_time,
	meet_time_mandatory, response_mode,
	COALESCE(nominated_role_ids, '{}') AS nominated_role_ids,
	status, created_at
`

// scanEventRow scans a full event row from the DB.
func scanEventRow(row pgx.Row) (*EventRow, error) {
	e := &EventRow{}
	var meetTime, startTime, endTime string
	err := row.Scan(
		&e.Id, &e.TeamId, &e.SeriesId, &e.Type, &e.Title, &e.Date,
		&e.Location, &e.Note, &e.Result,
		&meetTime, &startTime, &endTime,
		&e.MeetTimeMandatory, &e.ResponseMode,
		&e.NominatedRoleIds,
		&e.Status, &e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("events.scanEventRow: %w", err)
	}
	if meetTime != "" {
		e.MeetTime = &meetTime
	}
	if startTime != "" {
		e.StartTime = &startTime
	}
	if endTime != "" {
		e.EndTime = &endTime
	}
	return e, nil
}

// ─── ListEvents ─────────────────────────────────────────────────────────────

// ListCursor is the keyset position for event pagination. The comparison
// direction depends on scope (past is DESC, upcoming/all are ASC).
//
// Known, accepted limitation: date is mutable (UpdateEvent lets an admin
// reschedule). If an event's date changes to fall on the other side of an
// in-progress pagination's cursor while a caller is mid-page, that event can
// be skipped or, less likely, duplicated across pages -- the same tradeoff
// any keyset pagination scheme accepts when sorting by an editable column.
// The window is self-healing (a fresh list call is always fully correct)
// and low-impact (rescheduling mid-pagination is rarer than the equivalent
// window in members/absences), so this is deliberately not being
// architected around.
type ListCursor struct {
	Date time.Time `json:"d"`
	ID   uuid.UUID `json:"i"`
}

// ListEvents returns up to limit events for a team filtered by scope, starting
// after cur (nil = first page). Keyset pagination — no OFFSET.
//
// scope is typed on gen.ListEventsParamsScope (not a plain string)
// specifically so the repo-wide "exhaustive" linter (see .golangci.yml) can
// enforce that every case here is revisited when the enum grows -- a plain
// string switch is invisible to it (see notificationModule's identical
// reasoning in internal/notifications/service.go). The handler already
// rejects an unknown scope value via gen.ListEventsParamsScope.Valid()
// before it ever reaches here, so gen.All's case body and the default case
// are deliberately identical -- the default only guards against a future
// caller bypassing that boundary check, not against a currently-reachable
// input.
func (r *Repository) ListEvents(ctx context.Context, teamID string, scope gen.ListEventsParamsScope, limit int, cur *ListCursor) ([]EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Truncate to date granularity: events.date is a DATE column, which casts
	// to midnight UTC. Comparing it against a mid-day timestamp would push
	// today's events out of the "upcoming" set (and into "past") from 00:00:01
	// onward — exactly on the day they matter. Truncating makes "date >= today"
	// include today and "date < today" exclude it.
	today := time.Now().UTC().Truncate(24 * time.Hour)

	var (
		q    string
		args []any
	)

	switch scope {
	case gen.Past:
		args = []any{teamID, today, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) < ($4, $5)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 AND date < $2 %s ORDER BY date DESC, id DESC LIMIT $3`, selectEventFields, pred)
	case gen.Upcoming:
		args = []any{teamID, today, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) > ($4, $5)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 AND date >= $2 %s ORDER BY date ASC, id ASC LIMIT $3`, selectEventFields, pred)
	case gen.All:
		args = []any{teamID, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) > ($3, $4)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 %s ORDER BY date ASC, id ASC LIMIT $2`, selectEventFields, pred)
	default:
		args = []any{teamID, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) > ($3, $4)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 %s ORDER BY date ASC, id ASC LIMIT $2`, selectEventFields, pred)
	}

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.ListEvents: %w", err)
	}
	defer rows.Close()

	var out []EventRow
	for rows.Next() {
		e, err := scanEventRow(rows)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.ListEvents scan: %w", err)
		}
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.ListEvents: %w", err)
	}
	return out, nil
}

// ─── GetEvent ───────────────────────────────────────────────────────────────

// GetEvent retrieves a single event by ID, scoped to teamID.
func (r *Repository) GetEvent(ctx context.Context, eventID, teamID string) (*EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`SELECT %s FROM events WHERE id = $1 AND team_id = $2`, selectEventFields)
	row := r.pool.QueryRow(ctx, q, eventID, teamID)
	e, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("events.Repository.GetEvent: %w", err)
	}
	return e, nil
}

// validateNominatedRolesInTx verifies every ID in roleIDs is a role belonging
// to teamID, returning ErrInvalidNominatedRoleIDs otherwise. Takes the same
// pg_advisory_xact_lock(hashtextextended(teamID, 0)) key
// roles.DeleteRole/members.SetRoles/teams.UpdateTeam already use, so this
// check can't race with a concurrent role deletion committing (and scrubbing
// nominated_role_ids) between this validation and the caller's write --
// otherwise a role could be deleted right after being validated here,
// re-introducing the dangling reference DeleteRole's scrub just removed.
// Service.validateNominatedRoles (a separate, lock-free pre-check via the
// injected roleChecker) still runs first as a fast UX rejection; this is the
// authoritative, race-free check.
func validateNominatedRolesInTx(ctx context.Context, tx pgx.Tx, teamID string, roleIDs []uuid.UUID) error {
	if len(roleIDs) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return fmt.Errorf("events.Repository: advisory lock: %w", err)
	}
	seen := make(map[uuid.UUID]struct{}, len(roleIDs))
	for _, id := range roleIDs {
		seen[id] = struct{}{}
	}
	var count int
	if err := tx.QueryRow(
		ctx,
		`SELECT COUNT(*)::int FROM roles WHERE id = ANY($1) AND team_id = $2`,
		roleIDs, teamID,
	).Scan(&count); err != nil {
		return fmt.Errorf("events.Repository: check nominated roles: %w", err)
	}
	if count != len(seen) {
		return ErrInvalidNominatedRoleIDs
	}
	return nil
}

// ─── CreateEvent ────────────────────────────────────────────────────────────

// CreateEvent inserts a single event row and returns it.
func (r *Repository) CreateEvent(ctx context.Context, teamID string, params *CreateEventParams) (*EventRow, error) { //nolint:gocritic
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateEvent: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := validateNominatedRolesInTx(ctx, tx, teamID, params.NominatedRoleIds); err != nil {
		return nil, err
	}

	q := fmt.Sprintf(`
		INSERT INTO events (
			team_id, type, title, date, location, note,
			meet_time, start_time, end_time, meet_time_mandatory,
			response_mode, nominated_role_ids, status
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7::time, $8::time, $9::time, $10,
			$11, $12, 'active'
		)
		RETURNING %s
	`, selectEventFields)

	row := tx.QueryRow(
		ctx, q,
		teamID, params.Type, params.Title, params.Date,
		params.Location, params.Note,
		nullableTime(params.MeetTime), nullableTime(params.StartTime), nullableTime(params.EndTime),
		boolVal(params.MeetTimeMandatory),
		strVal(params.ResponseMode, "opt_in"),
		uuidSlice(params.NominatedRoleIds),
	)
	e, err := scanEventRow(row)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateEvent: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("events.Repository.CreateEvent: commit: %w", err)
	}
	return e, nil
}

// CreateSeries creates an event_series row and then one event per week for RepeatWeeks.
func (r *Repository) CreateSeries(ctx context.Context, teamID string, params *CreateEventParams) ([]EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	repeatWeeks := params.RepeatWeeks
	if repeatWeeks < 1 {
		repeatWeeks = 1
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := validateNominatedRolesInTx(ctx, tx, teamID, params.NominatedRoleIds); err != nil {
		return nil, err
	}

	// Insert series row.
	var seriesID uuid.UUID
	seriesQ := `
		INSERT INTO event_series (
			team_id, type, title, location, note,
			meet_time, start_time, end_time, meet_time_mandatory,
			response_mode, nominated_role_ids, repeat_weeks
		) VALUES (
			$1, $2, $3, $4, $5,
			$6::time, $7::time, $8::time, $9,
			$10, $11, $12
		)
		RETURNING id
	`
	err = tx.QueryRow(
		ctx, seriesQ,
		teamID, params.Type, params.Title, params.Location, params.Note,
		nullableTime(params.MeetTime), nullableTime(params.StartTime), nullableTime(params.EndTime),
		boolVal(params.MeetTimeMandatory),
		strVal(params.ResponseMode, "opt_in"),
		uuidSlice(params.NominatedRoleIds),
		repeatWeeks,
	).Scan(&seriesID)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: insert series: %w", err)
	}

	// Insert all event instances in a single round-trip via UNNEST over the
	// computed dates, rather than one INSERT...RETURNING per week. The
	// previous sequential-loop version (up to maxRepeatWeeks=104 round-trips)
	// ran inside this function's fixed 5s context timeout while holding the
	// team-wide advisory lock acquired above -- at repository latencies above
	// ~48ms/round-trip (routine for cloud Postgres across AZs, PgBouncer, or
	// pool contention), a legitimate max-length series request would exceed
	// the timeout and fail with a generic 500, while also serializing every
	// other lock-guarded team mutation for the loop's full duration.
	dates := make([]time.Time, repeatWeeks)
	for i := 0; i < repeatWeeks; i++ {
		dates[i] = params.Date.AddDate(0, 0, i*7)
	}
	eventQ := fmt.Sprintf(`
		INSERT INTO events (
			team_id, series_id, type, title, date, location, note,
			meet_time, start_time, end_time, meet_time_mandatory,
			response_mode, nominated_role_ids, status
		)
		SELECT $1, $2, $3, $4, d, $6, $7,
			$8::time, $9::time, $10::time, $11,
			$12, $13, 'active'
		FROM unnest($5::date[]) AS d
		RETURNING %s
	`, selectEventFields)

	rows, err := tx.Query(
		ctx, eventQ,
		teamID, seriesID, params.Type, params.Title, dates,
		params.Location, params.Note,
		nullableTime(params.MeetTime), nullableTime(params.StartTime), nullableTime(params.EndTime),
		boolVal(params.MeetTimeMandatory),
		strVal(params.ResponseMode, "opt_in"),
		uuidSlice(params.NominatedRoleIds),
	)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: insert events: %w", err)
	}
	defer rows.Close()

	var events []EventRow
	for rows.Next() {
		e, err := scanEventRow(rows)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.CreateSeries: %w", err)
		}
		events = append(events, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: %w", err)
	}
	// unnest() over a plain array does not guarantee row order matches the
	// input array's order (though it does in every version tested); sort by
	// date defensively so callers (CreateEvent reads rows[0]) always get the
	// first occurrence, not whichever row Postgres happened to return first.
	sort.Slice(events, func(i, j int) bool { return events[i].Date.Before(events[j].Date) })

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: commit: %w", err)
	}
	return events, nil
}

// ─── UpdateEvent ────────────────────────────────────────────────────────────

// UpdateEvent updates a single event or all events in its series, scoped to
// teamID. When scope is "series", the series-wide update and the single-event
// update run inside one transaction so a failure between them can never leave
// the series definition and the individual event instance inconsistent.
func (r *Repository) UpdateEvent(ctx context.Context, eventID, teamID string, params *UpdateEventParams, scope string) (*EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.UpdateEvent: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if params.NominatedRoleIds != nil {
		if err := validateNominatedRolesInTx(ctx, tx, teamID, params.NominatedRoleIds); err != nil {
			return nil, err
		}
	}

	if scope == "series" {
		// Get series_id for this event, verified to belong to teamID.
		var seriesID *uuid.UUID
		err := tx.QueryRow(ctx, `SELECT series_id FROM events WHERE id = $1 AND team_id = $2`, eventID, teamID).Scan(&seriesID)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.UpdateEvent: get series_id: %w", err)
		}
		if seriesID != nil {
			if err := updateSeriesEvents(ctx, tx, seriesID.String(), params); err != nil {
				return nil, err
			}
		}
	}

	// Always update the specific event and return it, scoped to teamID.
	e, err := writeOrReadSingleEvent(ctx, tx, eventID, teamID, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation {
			return nil, ErrEndTimeBeforeStartTime
		}
		return nil, fmt.Errorf("events.Repository.UpdateEvent: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("events.Repository.UpdateEvent: commit: %w", err)
	}
	return e, nil
}

// writeOrReadSingleEvent applies params to the single event eventID (scoped
// to teamID) and returns the resulting row. A request that set no field at
// all (buildEventUpdateSets' ok == false) has nothing to write -- this reads
// the row back instead of running a no-op UPDATE (see sqlbuilder's package
// doc for why a SET-clause fallback isn't used here).
func writeOrReadSingleEvent(ctx context.Context, tx pgx.Tx, eventID, teamID string, params *UpdateEventParams) (*EventRow, error) {
	setSQL, args, nextIdx, ok := buildEventUpdateSets(params, 1)
	if !ok {
		q := fmt.Sprintf(`SELECT %s FROM events WHERE id = $1 AND team_id = $2`, selectEventFields)
		return scanEventRow(tx.QueryRow(ctx, q, eventID, teamID))
	}
	args = append(args, eventID, teamID)
	q := fmt.Sprintf(`UPDATE events SET %s WHERE id = $%d AND team_id = $%d RETURNING %s`, setSQL, nextIdx, nextIdx+1, selectEventFields)
	return scanEventRow(tx.QueryRow(ctx, q, args...))
}

// updateSeriesEvents updates every event in seriesID within tx. Date is
// deliberately excluded: it's what makes each occurrence in a series
// distinct, so applying it series-wide would collapse every occurrence onto
// the same date instead of updating only the specific event scope=series was
// invoked on (which UpdateEvent still does afterward with the full params).
func updateSeriesEvents(ctx context.Context, tx pgx.Tx, seriesID string, params *UpdateEventParams) error {
	seriesParams := *params
	seriesParams.Date = nil
	setSQL, args, nextIdx, ok := buildEventUpdateSets(&seriesParams, 1)
	if !ok {
		// Nothing but Date was set (the common "change just this occurrence's
		// date, scope=series" request) — there's nothing series-wide to
		// update; UpdateEvent's subsequent direct update still applies the
		// date to the single targeted event.
		return nil
	}
	q := fmt.Sprintf(`UPDATE events SET %s WHERE series_id = $%d`, setSQL, nextIdx)
	args = append(args, seriesID)
	_, err := tx.Exec(ctx, q, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation {
			return ErrEndTimeBeforeStartTime
		}
		return fmt.Errorf("events.Repository.updateSeriesEvents: %w", err)
	}
	return nil
}

// buildEventUpdateSets builds the dynamic SET clause for a partial
// UpdateEventParams patch via sqlbuilder, numbering placeholders from
// startIdx. ok is false when params sets no field at all -- callers must not
// run an UPDATE in that case (see sqlbuilder's package doc comment).
func buildEventUpdateSets(params *UpdateEventParams, startIdx int) (setSQL string, args []any, nextIdx int, ok bool) {
	b := sqlbuilder.New()

	if params.Type != nil {
		b.Add("type", *params.Type)
	}
	if params.Title != nil {
		b.Add("title", *params.Title)
	}
	if params.Date != nil {
		b.Add("date", *params.Date)
	}
	if params.Location != nil {
		b.Add("location", *params.Location)
	}
	if params.Note != nil {
		b.Add("note", *params.Note)
	}
	if params.MeetTime != nil {
		b.Add("meet_time", nullableTime(params.MeetTime))
	}
	if params.StartTime != nil {
		b.Add("start_time", nullableTime(params.StartTime))
	}
	if params.EndTime != nil {
		b.Add("end_time", nullableTime(params.EndTime))
	}
	if params.MeetTimeMandatory != nil {
		b.Add("meet_time_mandatory", *params.MeetTimeMandatory)
	}
	if params.ResponseMode != nil {
		b.Add("response_mode", *params.ResponseMode)
	}
	if params.NominatedRoleIds != nil {
		b.Add("nominated_role_ids", params.NominatedRoleIds)
	}

	return b.Build(startIdx)
}

// ─── SetStatus ──────────────────────────────────────────────────────────────

// SetStatus updates event status for a single event or all events in its
// series, scoped to teamID. When scope is "series", the series-wide update
// and the single-event update run inside one transaction -- mirroring
// UpdateEvent's identical pattern -- so a failure between them (or a
// concurrent delete of the targeted event) can never leave the series-wide
// status flip committed while the caller sees a 404 for the specific event
// they asked to change.
func (r *Repository) SetStatus(ctx context.Context, eventID, teamID, status, scope string) (*EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.SetStatus: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if scope == "series" {
		var seriesID *uuid.UUID
		err := tx.QueryRow(ctx, `SELECT series_id FROM events WHERE id = $1 AND team_id = $2`, eventID, teamID).Scan(&seriesID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("events.Repository.SetStatus: get series_id: %w", err)
		}
		if seriesID != nil {
			// Only today's and future instances are affected. Bulk-changing the
			// status of already-held (past) occurrences would retroactively
			// rewrite team history — e.g. cancelling "the rest of the series"
			// must not flip completed trainings to cancelled and drop them from
			// stats. The event addressed by eventID is still updated
			// individually below regardless of its date.
			_, err = tx.Exec(ctx, `UPDATE events SET status = $1 WHERE series_id = $2 AND team_id = $3 AND date >= CURRENT_DATE`, status, seriesID, teamID)
			if err != nil {
				return nil, fmt.Errorf("events.Repository.SetStatus: update series: %w", err)
			}
		}
	}

	q := fmt.Sprintf(`UPDATE events SET status = $1 WHERE id = $2 AND team_id = $3 RETURNING %s`, selectEventFields)
	row := tx.QueryRow(ctx, q, status, eventID, teamID)
	e, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("events.Repository.SetStatus: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("events.Repository.SetStatus: commit: %w", err)
	}
	return e, nil
}

// ─── DeleteEvent ────────────────────────────────────────────────────────────

// DeleteEvent deletes a single event, or the entire series (all occurrences,
// past and future, plus their attendance and comments) scoped to teamID.
// events.series_id is ON DELETE SET NULL, not CASCADE, so the individual
// event rows must be deleted explicitly — deleting only the event_series row
// would detach the events instead of removing them.
func (r *Repository) DeleteEvent(ctx context.Context, eventID, teamID, scope string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if scope == "series" {
		var seriesID *uuid.UUID
		err := r.pool.QueryRow(ctx, `SELECT series_id FROM events WHERE id = $1 AND team_id = $2`, eventID, teamID).Scan(&seriesID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("events.Repository.DeleteEvent: get series_id: %w", err)
		}
		if seriesID != nil {
			tx, err := r.pool.Begin(ctx)
			if err != nil {
				return fmt.Errorf("events.Repository.DeleteEvent: begin tx: %w", err)
			}
			defer func() { _ = tx.Rollback(ctx) }()

			if _, err = tx.Exec(ctx, `DELETE FROM events WHERE series_id = $1 AND team_id = $2`, seriesID, teamID); err != nil {
				return fmt.Errorf("events.Repository.DeleteEvent: delete series events: %w", err)
			}
			if _, err = tx.Exec(ctx, `DELETE FROM event_series WHERE id = $1`, seriesID); err != nil {
				return fmt.Errorf("events.Repository.DeleteEvent: delete series: %w", err)
			}
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("events.Repository.DeleteEvent: commit: %w", err)
			}
			return nil
		}
	}

	tag, err := r.pool.Exec(ctx, `DELETE FROM events WHERE id = $1 AND team_id = $2`, eventID, teamID)
	if err != nil {
		return fmt.Errorf("events.Repository.DeleteEvent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ─── Attendance Summary ──────────────────────────────────────────────────────

// absenceCoversExpr and effectiveStatusExpr are defined once in
// internal/attendance and reused here so the event summary and the statistics
// module (internal/stats) can never drift apart on how effective attendance is
// derived. computeEffectiveAttendance mirrors the same precedence in Go.
const (
	absenceCoversExpr   = attendance.AbsenceCoversExpr
	effectiveStatusExpr = attendance.EffectiveStatusExpr
)

// GetAttendanceSummary returns aggregated attendance counts for an event,
// scoped to teamID. Roster-driven (joined from memberships, not attendance):
// every current team member is counted exactly once, with opt_out/absence-
// based defaulting (effectiveStatusExpr) applied to members who never
// explicitly responded -- a departed member (whose attendance row
// RemoveMember intentionally leaves in place as history, since attendance/
// absences are keyed by user_id/team_id rather than membership_id) is
// naturally excluded, since they're no longer a memberships row to join
// from.
func (r *Repository) GetAttendanceSummary(ctx context.Context, eventID, teamID string) (EventSummaryData, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT
			COUNT(*) FILTER (WHERE eff_status = 'yes')           AS yes,
			COUNT(*) FILTER (WHERE eff_status = 'no')            AS no,
			COUNT(*) FILTER (WHERE eff_status = 'maybe')         AS maybe,
			COUNT(*) FILTER (WHERE eff_status = 'pending')       AS pending,
			COUNT(*) FILTER (WHERE eff_status = 'not_nominated') AS not_nominated,
			COUNT(*) FILTER (WHERE eff_status != 'not_nominated') AS nominated,
			COUNT(*)                                              AS total
		FROM (
			SELECT ` + effectiveStatusExpr + ` AS eff_status
			FROM events e
			JOIN memberships m ON m.team_id = e.team_id
			LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = m.user_id
			WHERE e.id = $1 AND e.team_id = $2
		) sub
	`
	var s EventSummaryData
	err := r.pool.QueryRow(ctx, q, eventID, teamID).Scan(
		&s.Yes, &s.No, &s.Maybe, &s.Pending, &s.NotNominated, &s.Nominated, &s.Total,
	)
	if err != nil {
		return s, fmt.Errorf("events.Repository.GetAttendanceSummary: %w", err)
	}
	return s, nil
}

// ─── MyAttendance ───────────────────────────────────────────────────────────

// GetMyAttendance returns the current user's attendance record for an event,
// scoped to teamID, or nil.
func (r *Repository) GetMyAttendance(ctx context.Context, eventID, userID, teamID string) (*AttendanceDBRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT a.id, a.event_id, a.user_id, a.status, a.reason, a.reason_id, a.reason_visibility, a.at
		FROM attendance a
		JOIN events e ON e.id = a.event_id
		WHERE a.event_id = $1 AND a.user_id = $2 AND e.team_id = $3
	`
	row := r.pool.QueryRow(ctx, q, eventID, userID, teamID)
	a := &AttendanceDBRow{}
	err := row.Scan(&a.Id, &a.EventId, &a.UserId, &a.Status, &a.Reason, &a.ReasonId, &a.ReasonVisibility, &a.At)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("events.Repository.GetMyAttendance: %w", err)
	}
	return a, nil
}

// GetMyEffectiveAttendance returns userID's resolved attendance for an
// event scoped to teamID -- an explicit record if one exists, otherwise the
// result of opt_out/absence-based defaulting (computeEffectiveAttendance).
// Unlike GetMyAttendance, this is driven from events (LEFT JOIN attendance),
// not attendance, so it always resolves to a value for an event that exists
// in teamID; nil only means eventID doesn't belong to teamID.
func (r *Repository) GetMyEffectiveAttendance(ctx context.Context, eventID, userID, teamID string) (*EffectiveAttendance, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT a.status, a.reason, a.reason_id, a.reason_visibility, a.at,
		       EXISTS (
		           SELECT 1 FROM absences ab
		           WHERE ab.user_id = $2 AND ab.team_id = e.team_id
		             AND ab.from_date <= e.date AND ab.to_date >= e.date
		       ),
		       e.response_mode
		FROM events e
		LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = $2
		WHERE e.id = $1 AND e.team_id = $3
	`
	var status, reason, reasonID, reasonVisibility *string
	var at *time.Time
	var absenceCovers bool
	var responseMode string
	err := r.pool.QueryRow(ctx, q, eventID, userID, teamID).Scan(
		&status, &reason, &reasonID, &reasonVisibility, &at, &absenceCovers, &responseMode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("events.Repository.GetMyEffectiveAttendance: %w", err)
	}
	eff := computeEffectiveAttendance(status, reason, reasonID, reasonVisibility, at, absenceCovers, responseMode)
	return &eff, nil
}

// ─── Batched attendance lookups (used by ListEvents) ───────────────────────

// GetAttendanceSummaries returns aggregated attendance counts for multiple
// events in a single query, keyed by event ID. Roster-driven like
// GetAttendanceSummary: an event with zero current team members would be the
// only way to be absent from the map (never a real case, since CreateEvent
// requires a team to already exist), so callers can otherwise assume every
// requested eventID is present. Used by ListEvents to avoid issuing one
// GetAttendanceSummary query per event; all eventIDs in a single call
// belong to one team, matching ListEvents' own team-scoped query.
func (r *Repository) GetAttendanceSummaries(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID]EventSummaryData, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out := make(map[uuid.UUID]EventSummaryData, len(eventIDs))
	if len(eventIDs) == 0 {
		return out, nil
	}
	q := `
		SELECT
			id,
			COUNT(*) FILTER (WHERE eff_status = 'yes')            AS yes,
			COUNT(*) FILTER (WHERE eff_status = 'no')             AS no,
			COUNT(*) FILTER (WHERE eff_status = 'maybe')          AS maybe,
			COUNT(*) FILTER (WHERE eff_status = 'pending')        AS pending,
			COUNT(*) FILTER (WHERE eff_status = 'not_nominated')  AS not_nominated,
			COUNT(*) FILTER (WHERE eff_status != 'not_nominated') AS nominated,
			COUNT(*)                                              AS total
		FROM (
			SELECT e.id, ` + effectiveStatusExpr + ` AS eff_status
			FROM events e
			JOIN memberships m ON m.team_id = e.team_id
			LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = m.user_id
			WHERE e.id = ANY($1)
		) sub
		GROUP BY id
	`
	rows, err := r.pool.Query(ctx, q, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.GetAttendanceSummaries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var s EventSummaryData
		if err := rows.Scan(&id, &s.Yes, &s.No, &s.Maybe, &s.Pending, &s.NotNominated, &s.Nominated, &s.Total); err != nil {
			return nil, fmt.Errorf("events.Repository.GetAttendanceSummaries scan: %w", err)
		}
		out[id] = s
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.GetAttendanceSummaries: %w", err)
	}
	return out, nil
}

// GetMyAttendances returns userID's attendance record for multiple events in
// a single query, keyed by event ID. Events with no record for userID are
// absent from the map. Used by ListEvents to avoid issuing one
// GetMyAttendance query per event.
func (r *Repository) GetMyAttendances(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]AttendanceDBRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out := make(map[uuid.UUID]AttendanceDBRow, len(eventIDs))
	if len(eventIDs) == 0 {
		return out, nil
	}
	q := `
		SELECT id, event_id, user_id, status, reason, reason_id, reason_visibility, at
		FROM attendance
		WHERE event_id = ANY($1) AND user_id = $2
	`
	rows, err := r.pool.Query(ctx, q, eventIDs, userID)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.GetMyAttendances: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var a AttendanceDBRow
		if err := rows.Scan(&a.Id, &a.EventId, &a.UserId, &a.Status, &a.Reason, &a.ReasonId, &a.ReasonVisibility, &a.At); err != nil {
			return nil, fmt.Errorf("events.Repository.GetMyAttendances scan: %w", err)
		}
		out[a.EventId] = a
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.GetMyAttendances: %w", err)
	}
	return out, nil
}

// GetMyEffectiveAttendances returns userID's resolved attendance for
// multiple events in a single query, keyed by event ID -- the batched
// counterpart to GetMyEffectiveAttendance, used by ListEvents to avoid one
// query per event. Every eventID present in the DB is present in the
// result map (defaulted if the user has no explicit record); an eventID
// absent from the map means it doesn't exist.
func (r *Repository) GetMyEffectiveAttendances(ctx context.Context, eventIDs []uuid.UUID, userID string) (map[uuid.UUID]EffectiveAttendance, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out := make(map[uuid.UUID]EffectiveAttendance, len(eventIDs))
	if len(eventIDs) == 0 {
		return out, nil
	}
	q := `
		SELECT e.id, a.status, a.reason, a.reason_id, a.reason_visibility, a.at,
		       EXISTS (
		           SELECT 1 FROM absences ab
		           WHERE ab.user_id = $2 AND ab.team_id = e.team_id
		             AND ab.from_date <= e.date AND ab.to_date >= e.date
		       ),
		       e.response_mode
		FROM events e
		LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = $2
		WHERE e.id = ANY($1)
	`
	rows, err := r.pool.Query(ctx, q, eventIDs, userID)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.GetMyEffectiveAttendances: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var status, reason, reasonID, reasonVisibility *string
		var at *time.Time
		var absenceCovers bool
		var responseMode string
		if err := rows.Scan(&id, &status, &reason, &reasonID, &reasonVisibility, &at, &absenceCovers, &responseMode); err != nil {
			return nil, fmt.Errorf("events.Repository.GetMyEffectiveAttendances scan: %w", err)
		}
		out[id] = computeEffectiveAttendance(status, reason, reasonID, reasonVisibility, at, absenceCovers, responseMode)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.GetMyEffectiveAttendances: %w", err)
	}
	return out, nil
}

// ─── ListAttendance ─────────────────────────────────────────────────────────

// maxAttendanceRows caps the attendance list at a size no real team roster
// should ever reach. Unlike history-based lists (transactions, notifications),
// attendance is a complete per-event snapshot that callers rely on seeing in
// full, so this is a defensive backstop against pathological data (e.g. stale
// rows for removed members that were never cleaned up) rather than a real
// pagination cutoff.
const maxAttendanceRows = 5000

// ListAttendance returns up to maxAttendanceRows roster rows for an event
// scoped to teamID, enriched with user data and each member's effective
// attendance (computeEffectiveAttendance). Roster-driven, not
// attendance-driven: every current team member appears exactly once, even
// if they've never explicitly responded -- a departed member simply isn't a
// memberships row here anymore, matching the previous inner-join exclusion.
func (r *Repository) ListAttendance(ctx context.Context, eventID, teamID string) ([]AttendanceEnriched, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT
			m.id,
			m.user_id,
			m."group",
			u.name,
			u.avatar_color,
			(u.photo_data IS NOT NULL AND length(u.photo_data) > 0) AS has_photo,
			a.status,
			a.reason,
			a.reason_id,
			a.reason_visibility,
			a.at,
			` + absenceCoversExpr + `,
			e.response_mode
		FROM events e
		JOIN memberships m ON m.team_id = e.team_id
		JOIN users u ON u.id = m.user_id
		LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = m.user_id
		WHERE e.id = $1 AND e.team_id = $2
		ORDER BY u.name ASC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, eventID, teamID, maxAttendanceRows)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.ListAttendance: %w", err)
	}
	defer rows.Close()

	var out []AttendanceEnriched
	var membershipIDs []string
	for rows.Next() {
		var a AttendanceEnriched
		var status, reason, reasonID, reasonVisibility *string
		var at *time.Time
		var absenceCovers bool
		var responseMode string
		err := rows.Scan(
			&a.MembershipId, &a.UserId, &a.Group,
			&a.Name, &a.AvatarColor, &a.HasPhoto,
			&status, &reason, &reasonID, &reasonVisibility, &at,
			&absenceCovers, &responseMode,
		)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.ListAttendance scan: %w", err)
		}
		eff := computeEffectiveAttendance(status, reason, reasonID, reasonVisibility, at, absenceCovers, responseMode)
		a.Status = eff.Status
		a.Reason = eff.Reason
		a.ReasonId = eff.ReasonId
		a.ReasonVisibility = eff.ReasonVisibility
		a.At = eff.At
		a.Auto = eff.Auto
		a.Absent = eff.Absent
		out = append(out, a)
		membershipIDs = append(membershipIDs, a.MembershipId.String())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.ListAttendance: %w", err)
	}

	if len(out) > 0 {
		primaryRoles, err := r.batchGetPrimaryRoles(ctx, membershipIDs)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.ListAttendance: %w", err)
		}
		for i := range out {
			if role, ok := primaryRoles[out[i].MembershipId.String()]; ok {
				out[i].PrimaryRole = &role
			}
		}
	}
	return out, nil
}

// batchGetPrimaryRoles returns each membership's "primary" role -- the
// lowest-role-id-first convention members.Repository's
// batchGetRoles/getRolesForMembershipQ also ORDER BY, so this attendance
// row's PrimaryRole agrees with the same member's PrimaryRole on the
// members list -- keyed by membership ID. A membership with no roles is
// absent from the map. DISTINCT ON (mr.membership_id) with an ORDER BY on
// role id makes the "first" choice deterministic across calls, rather than
// depending on unspecified row order.
func (r *Repository) batchGetPrimaryRoles(ctx context.Context, membershipIDs []string) (map[string]teams.RoleRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (mr.membership_id)
			mr.membership_id, r.id, r.team_id, r.name, r.system, r.color, r.permissions
		FROM membership_roles mr
		JOIN roles r ON r.id = mr.role_id
		JOIN memberships m ON m.id = mr.membership_id
		WHERE mr.membership_id = ANY($1::uuid[]) AND r.team_id = m.team_id
		ORDER BY mr.membership_id, r.id
	`, membershipIDs)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.batchGetPrimaryRoles: %w", err)
	}
	defer rows.Close()

	result := make(map[string]teams.RoleRow)
	for rows.Next() {
		var membershipID string
		rr := teams.RoleRow{}
		var permJSON []byte
		if err := rows.Scan(&membershipID, &rr.Id, &rr.TeamID, &rr.Name, &rr.System, &rr.Color, &permJSON); err != nil {
			return nil, fmt.Errorf("events.Repository.batchGetPrimaryRoles scan: %w", err)
		}
		if err := json.Unmarshal(permJSON, &rr.Permissions); err != nil {
			return nil, fmt.Errorf("events.Repository.batchGetPrimaryRoles unmarshal: %w", err)
		}
		result[membershipID] = rr
	}
	return result, rows.Err()
}

// GetReasonVisibilityContext returns the team's configured reason-visibility
// role whitelist (teams.reason_visibility_role_ids) and the viewer's own role
// IDs within that team, so the service layer can decide whether to redact a
// declined-attendance reason for a given viewer.
func (r *Repository) GetReasonVisibilityContext(ctx context.Context, teamID, viewerID string) (teamRoleIDs, viewerRoleIDs []string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := r.pool.QueryRow(
		ctx,
		`SELECT reason_visibility_role_ids FROM teams WHERE id = $1`, teamID,
	).Scan(&teamRoleIDs); err != nil {
		return nil, nil, fmt.Errorf("events.Repository.GetReasonVisibilityContext team: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT r.id::text
		FROM roles r
		JOIN membership_roles mr ON mr.role_id = r.id
		JOIN memberships m ON m.id = mr.membership_id
		WHERE m.team_id = $1 AND m.user_id = $2 AND r.team_id = $1
	`, teamID, viewerID)
	if err != nil {
		return nil, nil, fmt.Errorf("events.Repository.GetReasonVisibilityContext viewer roles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, nil, fmt.Errorf("events.Repository.GetReasonVisibilityContext scan: %w", err)
		}
		viewerRoleIDs = append(viewerRoleIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("events.Repository.GetReasonVisibilityContext: %w", err)
	}
	return teamRoleIDs, viewerRoleIDs, nil
}

// ─── SetAttendance ──────────────────────────────────────────────────────────

// SetAttendance upserts an attendance record for an event scoped to teamID.
// callerID is the authenticated caller; when it differs from userID (setting
// another member's attendance), the write itself re-verifies callerID
// currently holds events:write via the WHERE clause below -- not just the
// service layer's earlier, unlocked permChecker.GetPermissions read. Without
// this, a concurrent SetRoles/DeleteRole/UpdateRole revoking the caller's
// events:write between that check and this write could still let the write
// through; folding the check into this statement's own atomic snapshot
// closes that window without needing a shared transaction or advisory lock
// on this very hot path. Returns pgx.ErrNoRows if eventID does not belong to
// teamID, if userID is not a member of teamID (prevents forging attendance
// rows for arbitrary users outside the team), OR -- in that narrow race --
// if callerID no longer holds events:write; these are deliberately not
// distinguished here, matching how every other reason this returns
// pgx.ErrNoRows is already ambiguous by design.
func (r *Repository) SetAttendance(ctx context.Context, eventID, callerID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*AttendanceDBRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		INSERT INTO attendance (event_id, user_id, status, reason, reason_id, reason_visibility, at)
		SELECT $1, $2, $3, $4, $5, $6, now()
		WHERE EXISTS (SELECT 1 FROM events WHERE id = $1 AND team_id = $7)
		  AND EXISTS (SELECT 1 FROM memberships WHERE team_id = $7 AND user_id = $2)
		  AND ($8 = $2 OR EXISTS (
		        SELECT 1 FROM roles r
		        JOIN membership_roles mr ON mr.role_id = r.id
		        JOIN memberships m ON m.id = mr.membership_id
		        WHERE m.team_id = $7 AND m.user_id = $8 AND r.team_id = $7
		          AND r.permissions->>'events' = 'write'
		      ))
		ON CONFLICT (event_id, user_id) DO UPDATE
			SET status = EXCLUDED.status,
			    reason = EXCLUDED.reason,
			    reason_id = EXCLUDED.reason_id,
			    reason_visibility = EXCLUDED.reason_visibility,
			    at = now()
		RETURNING id, event_id, user_id, status, reason, reason_id, reason_visibility, at
	`
	a := &AttendanceDBRow{}
	err := r.pool.QueryRow(ctx, q, eventID, userID, status, reason, reasonID, reasonVisibility, teamID, callerID).Scan(
		&a.Id, &a.EventId, &a.UserId, &a.Status, &a.Reason, &a.ReasonId, &a.ReasonVisibility, &a.At,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("events.Repository.SetAttendance: %w", err)
	}
	return a, nil
}

// ─── SetNomination ──────────────────────────────────────────────────────────

// SetNomination sets or removes nomination for a user on an event scoped to
// teamID. callerID is the authenticated caller; SetNomination is never
// self-service (the service layer requires events:write unconditionally, see
// events.Service.SetNomination), so unlike SetAttendance there is no "acting
// on self" bypass here -- the write itself re-verifies callerID currently
// holds events:write via the EXISTS clause below, closing the same
// concurrent-permission-revocation race SetAttendance's WHERE clause closes,
// rather than relying solely on the service layer's earlier, unlocked
// permChecker.GetPermissions read.
// Returns pgx.ErrNoRows if eventID does not belong to teamID, if userID is
// not a member of teamID, OR -- in that narrow race -- if callerID no longer
// holds events:write; these are deliberately not distinguished in the
// nominated=false branch, matching SetAttendance's identical ambiguity.
// nominated=false → upsert status=not_nominated
// nominated=true  → delete any not_nominated record for this user/event.
func (r *Repository) SetNomination(ctx context.Context, eventID, callerID, userID, teamID string, nominated bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if !nominated {
		// Clears reason/reason_id/reason_visibility on the ON CONFLICT branch,
		// not just status -- otherwise a prior "no" row's private decline
		// reason survives under status='not_nominated', which ListAttendance's
		// redaction only gates on status=="no", so it would leak to every
		// team member on the next GET .../attendance.
		q := `
			INSERT INTO attendance (event_id, user_id, status, at)
			SELECT $1, $2, 'not_nominated', now()
			WHERE EXISTS (SELECT 1 FROM events WHERE id = $1 AND team_id = $3)
			  AND EXISTS (SELECT 1 FROM memberships WHERE team_id = $3 AND user_id = $2)
			  AND EXISTS (
			        SELECT 1 FROM roles r
			        JOIN membership_roles mr ON mr.role_id = r.id
			        JOIN memberships m ON m.id = mr.membership_id
			        WHERE m.team_id = $3 AND m.user_id = $4 AND r.team_id = $3
			          AND r.permissions->>'events' = 'write'
			      )
			ON CONFLICT (event_id, user_id) DO UPDATE
				SET status = 'not_nominated', reason = NULL, reason_id = NULL, reason_visibility = NULL, at = now()
		`
		tag, err := r.pool.Exec(ctx, q, eventID, userID, teamID, callerID)
		if err != nil {
			return fmt.Errorf("events.Repository.SetNomination(false): %w", err)
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	}

	// Remove not_nominated record so the user reverts to pending/default,
	// scoped to teamID via a join back to events, and gated on callerID
	// currently holding events:write via the same EXISTS predicate.
	tag, err := r.pool.Exec(
		ctx,
		`DELETE FROM attendance a USING events e
		 WHERE a.event_id = e.id AND a.event_id = $1 AND a.user_id = $2
		   AND a.status = 'not_nominated' AND e.team_id = $3
		   AND EXISTS (
		         SELECT 1 FROM roles r
		         JOIN membership_roles mr ON mr.role_id = r.id
		         JOIN memberships m ON m.id = mr.membership_id
		         WHERE m.team_id = $3 AND m.user_id = $4 AND r.team_id = $3
		           AND r.permissions->>'events' = 'write'
		       )`,
		eventID, userID, teamID, callerID,
	)
	if err != nil {
		return fmt.Errorf("events.Repository.SetNomination(true): %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Distinguish "event not in team" from "caller lost events:write" from
		// "nothing to delete" -- a no-op delete for an owned event with a
		// permitted caller is not an error, but a cross-team attempt must be,
		// and a caller who has lost events:write must not be told the delete
		// silently succeeded.
		var eventInTeam, callerHasWrite bool
		if err := r.pool.QueryRow(ctx, `
			SELECT
			    EXISTS(SELECT 1 FROM events WHERE id = $1 AND team_id = $2),
			    EXISTS(
			        SELECT 1 FROM roles r
			        JOIN membership_roles mr ON mr.role_id = r.id
			        JOIN memberships m ON m.id = mr.membership_id
			        WHERE m.team_id = $2 AND m.user_id = $3 AND r.team_id = $2
			          AND r.permissions->>'events' = 'write'
			    )
		`, eventID, teamID, callerID).Scan(&eventInTeam, &callerHasWrite); err != nil {
			return fmt.Errorf("events.Repository.SetNomination(true): verify team/permission: %w", err)
		}
		if !eventInTeam || !callerHasWrite {
			return pgx.ErrNoRows
		}
	}
	return nil
}

// ─── Comments ───────────────────────────────────────────────────────────────

// maxCommentsPerEvent caps how many comments a single event can accumulate,
// enforced in Service.AddComment via CountComments -- unlike ListAttendance's
// per-request row cap, comments have no natural bound (any team member can
// add one via the self-service events/comments route, with no RBAC write
// gate), and ListComments' OFFSET-based pagination would otherwise let an
// unbounded comment count grow to where every page pays a proportionally
// larger scan cost, with no ceiling anywhere in the write path.
const maxCommentsPerEvent = 2000

// CountComments returns the number of comments an event has, used to enforce
// maxCommentsPerEvent before an insert.
func (r *Repository) CountComments(ctx context.Context, eventID, teamID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM event_comments c
		JOIN events e ON e.id = c.event_id
		WHERE c.event_id = $1 AND e.team_id = $2
	`, eventID, teamID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("events.Repository.CountComments: %w", err)
	}
	return count, nil
}

// ListComments returns all comments for an event scoped to teamID, enriched
// with user data.
func (r *Repository) ListComments(ctx context.Context, eventID, teamID string, limit, offset int) ([]CommentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT
			c.id, c.event_id, c.user_id, c.text, c.created_at,
			u.name,
			u.avatar_color,
			(u.photo_data IS NOT NULL AND length(u.photo_data) > 0) AS has_photo
		FROM event_comments c
		JOIN users u ON u.id = c.user_id
		JOIN events e ON e.id = c.event_id
		WHERE c.event_id = $1 AND e.team_id = $2
		ORDER BY c.created_at ASC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, q, eventID, teamID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.ListComments: %w", err)
	}
	defer rows.Close()

	var out []CommentRow
	for rows.Next() {
		var c CommentRow
		var hasPhoto bool
		err := rows.Scan(
			&c.Id, &c.EventId, &c.UserId, &c.Text, &c.CreatedAt,
			&c.ActorName, &c.ActorColor, &hasPhoto,
		)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.ListComments scan: %w", err)
		}
		c.HasActorPhoto = &hasPhoto
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.ListComments: %w", err)
	}
	return out, nil
}

// AddComment inserts a new event comment scoped to teamID and returns it
// enriched. Returns pgx.ErrNoRows if eventID does not belong to teamID, or if
// userID is not (or is no longer) a member of teamID -- events/comments is a
// self-service write (see authz.go), so RequireMembership only checks
// membership once at the start of the request; without this re-check here, a
// membership removal racing this call could still attach a permanently
// visible comment to an event from someone no longer on the team, the same
// gap events.SetAttendance/SetNomination already guard against.
func (r *Repository) AddComment(ctx context.Context, eventID, userID, teamID, text string) (*CommentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		WITH inserted AS (
			INSERT INTO event_comments (event_id, user_id, text)
			SELECT $1, $2, $3
			WHERE EXISTS (SELECT 1 FROM events WHERE id = $1 AND team_id = $4)
			  AND EXISTS (SELECT 1 FROM memberships WHERE team_id = $4 AND user_id = $2)
			RETURNING id, event_id, user_id, text, created_at
		)
		SELECT
			i.id, i.event_id, i.user_id, i.text, i.created_at,
			u.name, u.avatar_color,
			(u.photo_data IS NOT NULL AND length(u.photo_data) > 0) AS has_photo
		FROM inserted i
		JOIN users u ON u.id = i.user_id
	`
	c := &CommentRow{}
	var hasPhoto bool
	err := r.pool.QueryRow(ctx, q, eventID, userID, text, teamID).Scan(
		&c.Id, &c.EventId, &c.UserId, &c.Text, &c.CreatedAt,
		&c.ActorName, &c.ActorColor, &hasPhoto,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("events.Repository.AddComment: %w", err)
	}
	c.HasActorPhoto = &hasPhoto
	return c, nil
}

// DeleteComment deletes a comment if the requesting user owns it and it
// belongs to teamID.
func (r *Repository) DeleteComment(ctx context.Context, commentID, userID, teamID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(
		ctx,
		`DELETE FROM event_comments c USING events e
		 WHERE c.event_id = e.id AND c.id = $1 AND c.user_id = $2 AND e.team_id = $3`,
		commentID, userID, teamID,
	)
	if err != nil {
		return fmt.Errorf("events.Repository.DeleteComment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ─── internal helpers ────────────────────────────────────────────────────────

// nullableTime converts a *string "HH:mm" to a value suitable for a Postgres TIME column.
// Returns nil when s is nil or empty, so pgx sends NULL.
func nullableTime(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}
