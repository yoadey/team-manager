package events

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
type ListCursor struct {
	Date time.Time `json:"d"`
	ID   uuid.UUID `json:"i"`
}

// ListEvents returns up to limit events for a team filtered by scope, starting
// after cur (nil = first page). Keyset pagination — no OFFSET.
func (r *Repository) ListEvents(ctx context.Context, teamID, scope string, limit int, cur *ListCursor) ([]EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	today := time.Now().UTC()

	var (
		q    string
		args []any
	)

	switch scope {
	case "past":
		args = []any{teamID, today, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) < ($4, $5)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 AND date < $2 %s ORDER BY date DESC, id DESC LIMIT $3`, selectEventFields, pred)
	case "upcoming":
		args = []any{teamID, today, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) > ($4, $5)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 AND date >= $2 %s ORDER BY date ASC, id ASC LIMIT $3`, selectEventFields, pred)
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
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, teamID); err != nil {
		return fmt.Errorf("events.Repository: advisory lock: %w", err)
	}
	if len(roleIDs) == 0 {
		return nil
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
	sets, args := buildUpdateSets(params, eventID)
	args = append(args, teamID)
	q := fmt.Sprintf(`UPDATE events SET %s WHERE id = $%d AND team_id = $%d RETURNING %s`, sets, len(args)-1, len(args), selectEventFields)
	row := tx.QueryRow(ctx, q, args...)
	e, err := scanEventRow(row)
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

// updateSeriesEvents updates every event in seriesID within tx. Date is
// deliberately excluded: it's what makes each occurrence in a series
// distinct, so applying it series-wide would collapse every occurrence onto
// the same date instead of updating only the specific event scope=series was
// invoked on (which UpdateEvent still does afterward with the full params).
func updateSeriesEvents(ctx context.Context, tx pgx.Tx, seriesID string, params *UpdateEventParams) error {
	seriesParams := *params
	seriesParams.Date = nil
	sets, args := buildUpdateSets(&seriesParams, "")
	// Remove last arg (the eventID placeholder we added) — we use series_id instead.
	args = args[:len(args)-1]
	if len(args) == 0 {
		// Nothing but Date was set (the common "change just this occurrence's
		// date, scope=series" request) — buildUpdateSets' no-op fallback
		// ("SET id = $1") assumes the last arg is the primary key of the row
		// being updated, which held for the direct single-event UPDATE this
		// function's sibling call site does, but not here: with args empty,
		// the fallback's placeholder would bind to series_id in the WHERE
		// clause too, producing `UPDATE events SET id = $1 WHERE series_id =
		// $1` — overwriting every event's primary key with the series ID.
		// There's nothing series-wide to update, so skip the query entirely;
		// UpdateEvent's subsequent direct update still applies the date to
		// the single targeted event.
		return nil
	}
	q := fmt.Sprintf(`UPDATE events SET %s WHERE series_id = $%d`, sets, len(args)+1)
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

// buildUpdateSets constructs a SET clause and args slice for an UPDATE query.
// The event ID is appended as the last arg (placeholder = len(args)).
func buildUpdateSets(params *UpdateEventParams, eventID string) (setSQL string, args []interface{}) {
	var sets []string
	idx := 1

	add := func(col string, val interface{}) {
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}

	if params.Type != nil {
		add("type", *params.Type)
	}
	if params.Title != nil {
		add("title", *params.Title)
	}
	if params.Date != nil {
		add("date", *params.Date)
	}
	if params.Location != nil {
		add("location", *params.Location)
	}
	if params.Note != nil {
		add("note", *params.Note)
	}
	if params.MeetTime != nil {
		add("meet_time", nullableTime(params.MeetTime))
	}
	if params.StartTime != nil {
		add("start_time", nullableTime(params.StartTime))
	}
	if params.EndTime != nil {
		add("end_time", nullableTime(params.EndTime))
	}
	if params.MeetTimeMandatory != nil {
		add("meet_time_mandatory", *params.MeetTimeMandatory)
	}
	if params.ResponseMode != nil {
		add("response_mode", *params.ResponseMode)
	}
	if params.NominatedRoleIds != nil {
		add("nominated_role_ids", params.NominatedRoleIds)
	}

	// Append event ID as the last arg (used by the WHERE clause of the direct event update).
	args = append(args, eventID)

	if len(sets) == 0 {
		// If nothing to update, set a no-op to keep SQL valid.
		sets = append(sets, fmt.Sprintf("id = $%d", idx))
	}

	setStr := ""
	for i, s := range sets {
		if i > 0 {
			setStr += ", "
		}
		setStr += s
	}
	return setStr, args
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
			_, err = tx.Exec(ctx, `UPDATE events SET status = $1 WHERE series_id = $2 AND team_id = $3`, status, seriesID, teamID)
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

// GetAttendanceSummary returns aggregated attendance counts for an event,
// scoped to teamID.
func (r *Repository) GetAttendanceSummary(ctx context.Context, eventID, teamID string) (EventSummaryData, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT
			COUNT(*) FILTER (WHERE a.status = 'yes')           AS yes,
			COUNT(*) FILTER (WHERE a.status = 'no')            AS no,
			COUNT(*) FILTER (WHERE a.status = 'maybe')         AS maybe,
			COUNT(*) FILTER (WHERE a.status = 'pending')       AS pending,
			COUNT(*) FILTER (WHERE a.status = 'not_nominated') AS not_nominated,
			COUNT(*) FILTER (WHERE a.status != 'not_nominated') AS nominated,
			COUNT(*)                                            AS total
		FROM attendance a
		JOIN events e ON e.id = a.event_id
		WHERE a.event_id = $1 AND e.team_id = $2
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

// ─── Batched attendance lookups (used by ListEvents) ───────────────────────

// GetAttendanceSummaries returns aggregated attendance counts for multiple
// events in a single query, keyed by event ID. Events with no attendance
// rows are absent from the map (callers should treat that as a zero-value
// EventSummaryData). Used by ListEvents to avoid issuing one
// GetAttendanceSummary query per event.
func (r *Repository) GetAttendanceSummaries(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID]EventSummaryData, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out := make(map[uuid.UUID]EventSummaryData, len(eventIDs))
	if len(eventIDs) == 0 {
		return out, nil
	}
	q := `
		SELECT
			event_id,
			COUNT(*) FILTER (WHERE status = 'yes')            AS yes,
			COUNT(*) FILTER (WHERE status = 'no')             AS no,
			COUNT(*) FILTER (WHERE status = 'maybe')          AS maybe,
			COUNT(*) FILTER (WHERE status = 'pending')        AS pending,
			COUNT(*) FILTER (WHERE status = 'not_nominated')  AS not_nominated,
			COUNT(*) FILTER (WHERE status != 'not_nominated') AS nominated,
			COUNT(*)                                          AS total
		FROM attendance
		WHERE event_id = ANY($1)
		GROUP BY event_id
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

// ─── ListAttendance ─────────────────────────────────────────────────────────

// maxAttendanceRows caps the attendance list at a size no real team roster
// should ever reach. Unlike history-based lists (transactions, notifications),
// attendance is a complete per-event snapshot that callers rely on seeing in
// full, so this is a defensive backstop against pathological data (e.g. stale
// rows for removed members that were never cleaned up) rather than a real
// pagination cutoff.
const maxAttendanceRows = 5000

// ListAttendance returns up to maxAttendanceRows attendance rows for an event
// scoped to teamID, enriched with user data.
func (r *Repository) ListAttendance(ctx context.Context, eventID, teamID string) ([]AttendanceEnriched, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT
			a.user_id,
			a.status,
			a.reason,
			a.reason_id,
			a.reason_visibility,
			a.at,
			u.name,
			u.avatar_color,
			(u.photo_data IS NOT NULL AND length(u.photo_data) > 0) AS has_photo
		FROM attendance a
		JOIN users u ON u.id = a.user_id
		JOIN events e ON e.id = a.event_id
		WHERE a.event_id = $1 AND e.team_id = $2
		ORDER BY u.name ASC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, eventID, teamID, maxAttendanceRows)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.ListAttendance: %w", err)
	}
	defer rows.Close()

	var out []AttendanceEnriched
	for rows.Next() {
		var a AttendanceEnriched
		err := rows.Scan(
			&a.UserId, &a.Status, &a.Reason, &a.ReasonId, &a.ReasonVisibility, &a.At,
			&a.Name, &a.AvatarColor, &a.HasPhoto,
		)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.ListAttendance scan: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.ListAttendance: %w", err)
	}
	return out, nil
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
		WHERE m.team_id = $1 AND m.user_id = $2
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
// Returns pgx.ErrNoRows if eventID does not belong to teamID, or if userID is
// not a member of teamID (prevents forging attendance rows for arbitrary
// users outside the team).
func (r *Repository) SetAttendance(ctx context.Context, eventID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*AttendanceDBRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		INSERT INTO attendance (event_id, user_id, status, reason, reason_id, reason_visibility, at)
		SELECT $1, $2, $3, $4, $5, $6, now()
		WHERE EXISTS (SELECT 1 FROM events WHERE id = $1 AND team_id = $7)
		  AND EXISTS (SELECT 1 FROM memberships WHERE team_id = $7 AND user_id = $2)
		ON CONFLICT (event_id, user_id) DO UPDATE
			SET status = EXCLUDED.status,
			    reason = EXCLUDED.reason,
			    reason_id = EXCLUDED.reason_id,
			    reason_visibility = EXCLUDED.reason_visibility,
			    at = now()
		RETURNING id, event_id, user_id, status, reason, reason_id, reason_visibility, at
	`
	a := &AttendanceDBRow{}
	err := r.pool.QueryRow(ctx, q, eventID, userID, status, reason, reasonID, reasonVisibility, teamID).Scan(
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
// teamID. Returns pgx.ErrNoRows if eventID does not belong to teamID.
// nominated=false → upsert status=not_nominated
// nominated=true  → delete any not_nominated record for this user/event.
func (r *Repository) SetNomination(ctx context.Context, eventID, userID, teamID string, nominated bool) error {
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
			ON CONFLICT (event_id, user_id) DO UPDATE
				SET status = 'not_nominated', reason = NULL, reason_id = NULL, reason_visibility = NULL, at = now()
		`
		tag, err := r.pool.Exec(ctx, q, eventID, userID, teamID)
		if err != nil {
			return fmt.Errorf("events.Repository.SetNomination(false): %w", err)
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	}

	// Remove not_nominated record so the user reverts to pending/default,
	// scoped to teamID via a join back to events.
	tag, err := r.pool.Exec(
		ctx,
		`DELETE FROM attendance a USING events e
		 WHERE a.event_id = e.id AND a.event_id = $1 AND a.user_id = $2
		   AND a.status = 'not_nominated' AND e.team_id = $3`,
		eventID, userID, teamID,
	)
	if err != nil {
		return fmt.Errorf("events.Repository.SetNomination(true): %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Distinguish "event not in team" from "nothing to delete" by checking
		// the event exists in the team; a no-op delete for an owned event is
		// not an error, but a cross-team attempt must be.
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM events WHERE id = $1 AND team_id = $2)`, eventID, teamID).Scan(&exists); err != nil {
			return fmt.Errorf("events.Repository.SetNomination(true): verify team: %w", err)
		}
		if !exists {
			return pgx.ErrNoRows
		}
	}
	return nil
}

// ─── Comments ───────────────────────────────────────────────────────────────

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
