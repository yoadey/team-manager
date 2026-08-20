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

// eventsEndAfterStartTimeConstraint is the name of the CHECK constraint
// added by migration 00012. events/event_series also carry other CHECK
// constraints (type IN (...), response_mode IN (...)) on the same
// pgCheckViolation SQLSTATE -- matching on the SQLSTATE alone, without this
// name check, would misreport any of those as "endTime must be after
// startTime" too, mirroring the mistake absences' own CHECK-violation
// mapping is deliberately guarded against via ConstraintName.
const eventsEndAfterStartTimeConstraint = "events_end_after_start_time"

// ErrEndTimeBeforeStartTime is returned when a partial UpdateEvent would
// leave end_time <= start_time (violates the events_end_after_start_time
// CHECK constraint). The handler validates this when both fields are present
// in the same request, but a partial update (only one of startTime/endTime)
// can only be caught here, since the merge happens inside the UPDATE
// statement itself -- see absences' identical ErrInvalidDateRange pattern.
var ErrEndTimeBeforeStartTime = errors.New("endTime: must be after startTime")

// eventsEndDateAfterDateConstraint is the CHECK constraint added by
// migration 00025, guarding end_date >= date -- mirrors
// eventsEndAfterStartTimeConstraint's reasoning above.
const eventsEndDateAfterDateConstraint = "events_end_date_after_date"

// eventsMultiDaySpanWithinLimitConstraint is the CHECK constraint added by
// migration 00025, capping end_date - date at maxMultiDaySpanDays -- the
// partial-update backstop for the same cap Service.CreateEvent enforces
// early (see ErrMultiDaySpanTooLong).
const eventsMultiDaySpanWithinLimitConstraint = "events_multiday_span_within_limit"

// ErrMultiDayEndDateBeforeDate is returned when a create or partial update
// would leave end_date < date. Service.CreateEvent catches the common case
// early (both fields present in the same request); a partial UpdateEvent
// that only changes one of date/multiDayEndDate can only be caught here,
// once the merge happens inside the UPDATE statement itself and violates the
// events_end_date_after_date CHECK constraint -- see
// ErrEndTimeBeforeStartTime's identical reasoning.
var ErrMultiDayEndDateBeforeDate = errors.New("multiDayEndDate: must not be before date")

// ErrMultiDayEndDateOnSeriesEvent is returned when a caller sets
// multiDayEndDate on an event that belongs to a recurring series --
// multi-day spans are only meaningful for standalone events (see design.md's
// "Mutually exclusive with recurring" decision), and series occurrences are
// generated one per calendar day, so allowing one occurrence to span several
// would silently overlap with the next occurrence's own date.
var ErrMultiDayEndDateOnSeriesEvent = errors.New("multiDayEndDate: cannot be set on an event that belongs to a recurring series")

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

// ─── event_teams (cross-team events) ───────────────────────────────────────

// eventScopedByAnyTargetTeam is a WHERE-clause fragment: true when teamID
// (the placeholder given as teamPlaceholder, e.g. "$2") is one of the teams
// event alias "e" targets, via the event_teams join table (migration
// 00035). Every event has at least one event_teams row -- its owning team,
// always inserted by CreateEvent/CreateSeries -- so for a single-team event
// this is a strict superset of (in fact, identical to) the old
// "e.team_id = $N" scope check it replaces across every read/RSVP path
// (GetEvent, ListEvents, comments, attendance list/RSVP/nominations, event
// status). UpdateEvent and DeleteEvent deliberately keep the old
// owning-team-only "e.team_id = $N" check instead of this relaxation --
// editing/deleting a cross-team event is only ever done via its owning
// team's URL (see events.Service's create/update authorization), and
// deletion is furthermore never extended to non-owning targets at all (see
// migration 00035's comment).
func eventScopedByAnyTargetTeam(teamPlaceholder string) string {
	return "EXISTS (SELECT 1 FROM event_teams et WHERE et.event_id = e.id AND et.team_id = " + teamPlaceholder + ")"
}

// dedupTeamIDs returns owning plus every id in extra, deduplicated. Order is
// not meaningful (event_teams' PK also tolerates duplicates via ON CONFLICT
// in insertEventTeams, so this is a courtesy, not a correctness
// requirement).
func dedupTeamIDs(owning uuid.UUID, extra []uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{owning: {}}
	out := []uuid.UUID{owning}
	for _, id := range extra {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// insertEventTeams inserts one event_teams row for every (event, team) pair
// in the cross product of eventIDs × teamIDs, run inside tx. Used
// identically by CreateEvent (a single eventID) and CreateSeries (one row
// per generated occurrence -- every occurrence in a series shares the same
// target team set).
func insertEventTeams(ctx context.Context, tx pgx.Tx, eventIDs, teamIDs []uuid.UUID) error {
	if len(eventIDs) == 0 || len(teamIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO event_teams (event_id, team_id)
		SELECT e, t FROM unnest($1::uuid[]) AS e CROSS JOIN unnest($2::uuid[]) AS t
		ON CONFLICT (event_id, team_id) DO NOTHING
	`, eventIDs, teamIDs)
	if err != nil {
		return fmt.Errorf("events.Repository.insertEventTeams: %w", err)
	}
	return nil
}

// TeamsExist reports whether every id in teamIDs is a real team. An empty
// teamIDs always returns true. Used by Service to validate crossTeamIds
// before the more expensive per-team permission check, outside of any
// write transaction -- validateCrossTeamIDsInTx duplicates this same
// existence check inside CreateEvent/CreateSeries/UpdateEvent's own
// transaction as the authoritative, final guard (the same
// pre-check-then-authoritative-recheck split validateNominatedRoles/
// validateNominatedRolesInTx already establishes for nominated_role_ids).
func (r *Repository) TeamsExist(ctx context.Context, teamIDs []uuid.UUID) (bool, error) {
	if len(teamIDs) == 0 {
		return true, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	seen := make(map[uuid.UUID]struct{}, len(teamIDs))
	for _, id := range teamIDs {
		seen[id] = struct{}{}
	}
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*)::int FROM teams WHERE id = ANY($1)`, teamIDs).Scan(&count); err != nil {
		return false, fmt.Errorf("events.Repository.TeamsExist: %w", err)
	}
	return count == len(seen), nil
}

// ListEventTeamsBatch is GetEventTeams' batched counterpart, keyed by event
// ID -- used by ListEvents to avoid one query per event.
func (r *Repository) ListEventTeamsBatch(ctx context.Context, eventIDs []uuid.UUID) (map[uuid.UUID][]EventTeamRow, error) {
	out := make(map[uuid.UUID][]EventTeamRow, len(eventIDs))
	if len(eventIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT et.event_id, t.id, t.name
		FROM event_teams et
		JOIN teams t ON t.id = et.team_id
		WHERE et.event_id = ANY($1)
		ORDER BY t.name ASC
	`, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.ListEventTeamsBatch: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventID uuid.UUID
		var row EventTeamRow
		if err := rows.Scan(&eventID, &row.TeamID, &row.TeamName); err != nil {
			return nil, fmt.Errorf("events.Repository.ListEventTeamsBatch scan: %w", err)
		}
		out[eventID] = append(out[eventID], row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.ListEventTeamsBatch: %w", err)
	}
	return out, nil
}

// validateCrossTeamIDsInTx verifies every ID in crossTeamIDs is a real team,
// returning ErrInvalidCrossTeamIds otherwise. Unlike
// validateNominatedRolesInTx, this takes no advisory lock: a team, unlike a
// role, is never deleted as part of an ordinary in-team admin action (team
// deletion is not a feature this codebase has), so there is no equivalent
// concurrent-deletion race to close here.
func validateCrossTeamIDsInTx(ctx context.Context, tx pgx.Tx, crossTeamIDs []uuid.UUID) error {
	if len(crossTeamIDs) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(crossTeamIDs))
	for _, id := range crossTeamIDs {
		seen[id] = struct{}{}
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*)::int FROM teams WHERE id = ANY($1)`, crossTeamIDs).Scan(&count); err != nil {
		return fmt.Errorf("events.Repository: check cross-team ids: %w", err)
	}
	if count != len(seen) {
		return ErrInvalidCrossTeamIds
	}
	return nil
}

// GetEventTeams returns every team an event targets (its owning team plus
// any crossTeamIds), each with its name, ordered by name ascending. Every
// event has at least one row (see migration 00035) -- a single-team event's
// slice always has length 1.
func (r *Repository) GetEventTeams(ctx context.Context, eventID string) ([]EventTeamRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name
		FROM event_teams et
		JOIN teams t ON t.id = et.team_id
		WHERE et.event_id = $1
		ORDER BY t.name ASC
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.GetEventTeams: %w", err)
	}
	defer rows.Close()

	var out []EventTeamRow
	for rows.Next() {
		var row EventTeamRow
		if err := rows.Scan(&row.TeamID, &row.TeamName); err != nil {
			return nil, fmt.Errorf("events.Repository.GetEventTeams scan: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.GetEventTeams: %w", err)
	}
	return out, nil
}

// ListEventMemberTeams returns, for every user who belongs to at least one
// of an event's targeted teams, the subset of those targeted teams they
// belong to (team_id + name) -- the raw material
// Service.ListAttendance/enrichEvent use to compute each attendee's team
// badge (display rule: no badge if the viewer's own team is in this set,
// else the alphabetically-first team name in it). Only meaningful for a
// cross-team event with more than one targeted team; callers skip calling
// this at all for a single-team event, since its roster never needs a
// badge -- see GetEventTeams' length.
func (r *Repository) ListEventMemberTeams(ctx context.Context, eventID string) (map[uuid.UUID][]EventTeamRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := r.pool.Query(ctx, `
		SELECT m.user_id, t.id, t.name
		FROM event_teams et
		JOIN teams t ON t.id = et.team_id
		JOIN memberships m ON m.team_id = et.team_id
		WHERE et.event_id = $1
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.ListEventMemberTeams: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID][]EventTeamRow)
	for rows.Next() {
		var userID uuid.UUID
		var row EventTeamRow
		if err := rows.Scan(&userID, &row.TeamID, &row.TeamName); err != nil {
			return nil, fmt.Errorf("events.Repository.ListEventMemberTeams scan: %w", err)
		}
		out[userID] = append(out[userID], row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.ListEventMemberTeams: %w", err)
	}
	return out, nil
}

const selectEventFields = `
	id, team_id, series_id, type, title, date, end_date,
	location, note, result,
	COALESCE(TO_CHAR(meet_time, 'HH24:MI'), '') AS meet_time,
	COALESCE(TO_CHAR(start_time, 'HH24:MI'), '') AS start_time,
	COALESCE(TO_CHAR(end_time, 'HH24:MI'), '') AS end_time,
	meet_time_mandatory, response_mode,
	COALESCE(nominated_role_ids, '{}') AS nominated_role_ids,
	status, created_at, cancel_lead_minutes, exclude_from_stats
`

// scanEventRow scans a full event row from the DB.
func scanEventRow(row pgx.Row) (*EventRow, error) {
	e := &EventRow{}
	var meetTime, startTime, endTime string
	err := row.Scan(
		&e.Id, &e.TeamId, &e.SeriesId, &e.Type, &e.Title, &e.Date, &e.EndDate,
		&e.Location, &e.Note, &e.Result,
		&meetTime, &startTime, &endTime,
		&e.MeetTimeMandatory, &e.ResponseMode,
		&e.NominatedRoleIds,
		&e.Status, &e.CreatedAt, &e.CancelLeadMinutes, &e.ExcludeFromStats,
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

	teamScope := eventScopedByAnyTargetTeam("$1")
	switch scope {
	case gen.Past:
		args = []any{teamID, today, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) < ($4, $5)"
			args = append(args, cur.Date, cur.ID)
		}
		// COALESCE(end_date, date): a multi-day event that has started but not
		// yet finished (end_date >= today) stays out of "past" even though its
		// start date has already gone by -- it's still ongoing.
		q = fmt.Sprintf(`SELECT %s FROM events e WHERE %s AND COALESCE(end_date, date) < $2 %s ORDER BY date DESC, id DESC LIMIT $3`, selectEventFields, teamScope, pred)
	case gen.Upcoming:
		args = []any{teamID, today, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) > ($4, $5)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events e WHERE %s AND COALESCE(end_date, date) >= $2 %s ORDER BY date ASC, id ASC LIMIT $3`, selectEventFields, teamScope, pred)
	case gen.All:
		args = []any{teamID, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) > ($3, $4)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events e WHERE %s %s ORDER BY date ASC, id ASC LIMIT $2`, selectEventFields, teamScope, pred)
	default:
		args = []any{teamID, limit}
		pred := ""
		if cur != nil {
			pred = "AND (date, id) > ($3, $4)"
			args = append(args, cur.Date, cur.ID)
		}
		q = fmt.Sprintf(`SELECT %s FROM events e WHERE %s %s ORDER BY date ASC, id ASC LIMIT $2`, selectEventFields, teamScope, pred)
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

// ListUpcomingForReminders returns every non-cancelled event, across all
// teams, whose date falls within [from, to] -- used by
// jobs.EventReminderWorker to find candidate events for its periodic
// reminder scan. Unlike ListEvents, this is deliberately not scoped to a
// single team: the periodic job discovers which team each candidate belongs
// to from the returned rows themselves. from/to bound the query to a coarse,
// day-granularity window (cheap index range scan on date); the caller is
// responsible for the precise, timezone-aware "is this event's start
// instant actually due" check via EventStartInstant.
func (r *Repository) ListUpcomingForReminders(ctx context.Context, from, to time.Time) ([]EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`SELECT %s FROM events WHERE date BETWEEN $1 AND $2 AND status != 'cancelled'`, selectEventFields)
	rows, err := r.pool.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.ListUpcomingForReminders: %w", err)
	}
	defer rows.Close()

	var out []EventRow
	for rows.Next() {
		e, err := scanEventRow(rows)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.ListUpcomingForReminders scan: %w", err)
		}
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("events.Repository.ListUpcomingForReminders: %w", err)
	}
	return out, nil
}

// ─── GetEvent ───────────────────────────────────────────────────────────────

// GetEvent retrieves a single event by ID, scoped to teamID -- teamID must
// be one of the event's targeted teams (event_teams; see
// eventScopedByAnyTargetTeam), not necessarily its owning team.
func (r *Repository) GetEvent(ctx context.Context, eventID, teamID string) (*EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`SELECT %s FROM events e WHERE e.id = $1 AND %s`, selectEventFields, eventScopedByAnyTargetTeam("$2"))
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
	if err := validateCrossTeamIDsInTx(ctx, tx, params.CrossTeamIds); err != nil {
		return nil, err
	}
	owningTeamID, err := uuid.Parse(teamID)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateEvent: parse teamID: %w", err)
	}

	q := fmt.Sprintf(`
		INSERT INTO events (
			team_id, type, title, date, end_date, location, note,
			meet_time, start_time, end_time, meet_time_mandatory,
			response_mode, nominated_role_ids, status, cancel_lead_minutes,
			exclude_from_stats
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8::time, $9::time, $10::time, $11,
			$12, $13, 'active', $14, $15
		)
		RETURNING %s
	`, selectEventFields)

	row := tx.QueryRow(
		ctx, q,
		teamID, params.Type, params.Title, params.Date, params.EndDate,
		params.Location, params.Note,
		nullableTime(params.MeetTime), nullableTime(params.StartTime), nullableTime(params.EndTime),
		boolVal(params.MeetTimeMandatory),
		strVal(params.ResponseMode, "opt_in"),
		uuidSlice(params.NominatedRoleIds),
		params.CancelLeadMinutes,
		params.ExcludeFromStats,
	)
	e, err := scanEventRow(row)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateEvent: %w", err)
	}

	if err := insertEventTeams(ctx, tx, []uuid.UUID{e.Id}, dedupTeamIDs(owningTeamID, params.CrossTeamIds)); err != nil {
		return nil, fmt.Errorf("events.Repository.CreateEvent: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("events.Repository.CreateEvent: commit: %w", err)
	}
	return e, nil
}

// seriesDates computes the weekly occurrence dates for a series, branching on
// whichever of RepeatWeeks/RepeatEndDate the caller set. When RepeatEndDate
// is set it takes precedence: occurrences run weekly from params.Date up to
// and including RepeatEndDate. Otherwise RepeatWeeks (defaulting to 1) gives
// a fixed count, as before. Both paths are already capped at maxRepeatWeeks
// by Service.CreateEvent before this is ever called; the loop below still
// stops at maxRepeatWeeks defensively so a future caller that skips that
// pre-check can't turn this into an unbounded loop inside an open
// transaction.
func seriesDates(params *CreateEventParams) []time.Time {
	if params.RepeatEndDate != nil {
		var dates []time.Time
		for d := params.Date; !d.After(*params.RepeatEndDate) && len(dates) < maxRepeatWeeks; d = d.AddDate(0, 0, 7) {
			dates = append(dates, d)
		}
		if len(dates) == 0 {
			dates = []time.Time{params.Date}
		}
		return dates
	}
	repeatWeeks := params.RepeatWeeks
	if repeatWeeks < 1 {
		repeatWeeks = 1
	}
	if repeatWeeks > maxRepeatWeeks {
		repeatWeeks = maxRepeatWeeks
	}
	var dates []time.Time
	for i := 0; i < repeatWeeks; i++ {
		dates = append(dates, params.Date.AddDate(0, 0, i*7))
	}
	return dates
}

// CreateSeries creates an event_series row and then one event per week for
// RepeatWeeks, or -- when RepeatEndDate is set instead -- weekly up to and
// including that end date (see seriesDates).
func (r *Repository) CreateSeries(ctx context.Context, teamID string, params *CreateEventParams) ([]EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	dates := seriesDates(params)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := validateNominatedRolesInTx(ctx, tx, teamID, params.NominatedRoleIds); err != nil {
		return nil, err
	}
	if err := validateCrossTeamIDsInTx(ctx, tx, params.CrossTeamIds); err != nil {
		return nil, err
	}
	owningTeamID, err := uuid.Parse(teamID)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: parse teamID: %w", err)
	}

	// Insert series row. repeat_weeks is always populated with the derived
	// occurrence count (len(dates)), even in end-date mode, so read paths
	// that only ever consulted repeat_weeks keep working unchanged;
	// repeat_end_date is only set in end-date mode.
	var seriesID uuid.UUID
	seriesQ := `
		INSERT INTO event_series (
			team_id, type, title, location, note,
			meet_time, start_time, end_time, meet_time_mandatory,
			response_mode, nominated_role_ids, repeat_weeks, repeat_end_date, cancel_lead_minutes,
			exclude_from_stats
		) VALUES (
			$1, $2, $3, $4, $5,
			$6::time, $7::time, $8::time, $9,
			$10, $11, $12, $13, $14, $15
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
		len(dates),
		params.RepeatEndDate,
		params.CancelLeadMinutes,
		params.ExcludeFromStats,
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
	eventQ := fmt.Sprintf(`
		INSERT INTO events (
			team_id, series_id, type, title, date, location, note,
			meet_time, start_time, end_time, meet_time_mandatory,
			response_mode, nominated_role_ids, status, cancel_lead_minutes,
			exclude_from_stats
		)
		SELECT $1, $2, $3, $4, d, $6, $7,
			$8::time, $9::time, $10::time, $11,
			$12, $13, 'active', $14, $15
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
		params.CancelLeadMinutes,
		params.ExcludeFromStats,
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

	eventIDs := make([]uuid.UUID, len(events))
	for i := range events {
		eventIDs[i] = events[i].Id
	}
	if err := insertEventTeams(ctx, tx, eventIDs, dedupTeamIDs(owningTeamID, params.CrossTeamIds)); err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: %w", err)
	}

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
	if params.CrossTeamIds != nil {
		if err := validateCrossTeamIDsInTx(ctx, tx, params.CrossTeamIds); err != nil {
			return nil, err
		}
	}

	if scope == "series" || params.EndDate != nil {
		if err := applySeriesScopedUpdate(ctx, tx, eventID, teamID, scope, params); err != nil {
			return nil, err
		}
	}

	// Always update the specific event and return it, scoped to teamID.
	e, err := writeOrReadSingleEvent(ctx, tx, eventID, teamID, params)
	if err != nil {
		return nil, mapUpdateEventWriteError(err)
	}

	// crossTeamIds present in the patch replaces the full additional-target
	// set. writeOrReadSingleEvent above already verified eventID belongs to
	// teamID (the owning team -- crossTeamIds is only ever changed via the
	// owning team's own URL, mirroring create), so teamID is exactly the
	// owning team to reinsert here.
	if params.CrossTeamIds != nil {
		if err := replaceEventTeams(ctx, tx, e.Id, teamID, params.CrossTeamIds); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("events.Repository.UpdateEvent: commit: %w", err)
	}
	return e, nil
}

// applySeriesScopedUpdate looks up eventID's series_id (verified to belong
// to teamID) and, if it belongs to a series, either rejects a multi-day
// EndDate on it (ErrMultiDayEndDateOnSeriesEvent) or -- for scope="series"
// -- applies the series-wide fields via updateSeriesEvents, plus (when
// CrossTeamIds is present in the patch) replaces event_teams for every
// occurrence in the series via replaceEventTeamsForSeries, not just the
// single occurrence eventID addresses -- otherwise sharing (or un-sharing)
// "the whole series" from one occurrence would silently apply to that one
// occurrence only, leaving the rest of the series either not shared with a
// newly-added team or still readable by a removed one. A standalone
// (non-series) event, or a scope="single" request with no EndDate, is a
// no-op here; UpdateEvent's subsequent writeOrReadSingleEvent (and its own
// single-event replaceEventTeams call) always applies the full params to the
// specific event regardless, which for scope="series" redundantly but
// harmlessly re-applies the same event_teams rows this function already set
// for that one occurrence.
func applySeriesScopedUpdate(ctx context.Context, tx pgx.Tx, eventID, teamID, scope string, params *UpdateEventParams) error {
	var seriesID *uuid.UUID
	err := tx.QueryRow(ctx, `SELECT series_id FROM events WHERE id = $1 AND team_id = $2`, eventID, teamID).Scan(&seriesID)
	if err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: get series_id: %w", err)
	}
	if seriesID == nil {
		return nil
	}
	if params.EndDate != nil {
		return ErrMultiDayEndDateOnSeriesEvent
	}
	if scope != "series" {
		return nil
	}
	if err := updateSeriesEvents(ctx, tx, seriesID.String(), params); err != nil {
		return err
	}
	if params.CrossTeamIds != nil {
		if err := replaceEventTeamsForSeries(ctx, tx, seriesID.String(), teamID, params.CrossTeamIds); err != nil {
			return err
		}
	}
	return nil
}

// writeOrReadSingleEvent applies params to the single event eventID (scoped
// to teamID) and returns the resulting row. A request that set no field at
// all (buildEventUpdateSets' ok == false) has nothing to write -- this reads
// mapUpdateEventWriteError translates writeOrReadSingleEvent's raw error
// into the sentinel UpdateEvent's callers expect -- split out of UpdateEvent
// itself purely to keep that function's cognitive complexity under the
// repo's golangci-lint threshold (see events_end_after_start_time and
// siblings' identical CHECK-violation constants for what each case maps).
func mapUpdateEventWriteError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return pgx.ErrNoRows
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation {
		switch pgErr.ConstraintName {
		case eventsEndAfterStartTimeConstraint:
			return ErrEndTimeBeforeStartTime
		case eventsEndDateAfterDateConstraint:
			return ErrMultiDayEndDateBeforeDate
		case eventsMultiDaySpanWithinLimitConstraint:
			return ErrMultiDaySpanTooLong
		}
	}
	return fmt.Errorf("events.Repository.UpdateEvent: %w", err)
}

// replaceEventTeams drops every existing event_teams row for eventID and
// reinserts {owningTeamID} ∪ crossTeamIDs -- split out of UpdateEvent purely
// to keep that function's cognitive complexity under the repo's
// golangci-lint threshold. owningTeamID is parsed from the caller-supplied
// team ID string (UpdateEvent's own teamID param, verified by
// writeOrReadSingleEvent to be the event's owning team).
func replaceEventTeams(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, owningTeamIDStr string, crossTeamIDs []uuid.UUID) error {
	owningTeamID, err := uuid.Parse(owningTeamIDStr)
	if err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: parse teamID: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_teams WHERE event_id = $1`, eventID); err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: clear event_teams: %w", err)
	}
	if err := insertEventTeams(ctx, tx, []uuid.UUID{eventID}, dedupTeamIDs(owningTeamID, crossTeamIDs)); err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: %w", err)
	}
	return nil
}

// replaceEventTeamsForSeries replaces the event_teams rows for every event in
// seriesID with {owningTeamID} ∪ crossTeamIDs, keeping a whole recurring
// series' cross-team targets in lockstep with each other -- mirrors
// replaceEventTeams but applied series-wide instead of to a single event; see
// applySeriesScopedUpdate for why this is needed.
func replaceEventTeamsForSeries(ctx context.Context, tx pgx.Tx, seriesID, owningTeamIDStr string, crossTeamIDs []uuid.UUID) error {
	owningTeamID, err := uuid.Parse(owningTeamIDStr)
	if err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: parse teamID: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT id FROM events WHERE series_id = $1`, seriesID)
	if err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: list series events: %w", err)
	}
	var eventIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("events.Repository.UpdateEvent: scan series event id: %w", err)
		}
		eventIDs = append(eventIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: list series events: %w", err)
	}
	if len(eventIDs) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_teams WHERE event_id = ANY($1)`, eventIDs); err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: clear series event_teams: %w", err)
	}
	if err := insertEventTeams(ctx, tx, eventIDs, dedupTeamIDs(owningTeamID, crossTeamIDs)); err != nil {
		return fmt.Errorf("events.Repository.UpdateEvent: %w", err)
	}
	return nil
}

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
	seriesParams.EndDate = nil
	seriesParams.ClearEndDate = false
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
		if errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation && pgErr.ConstraintName == eventsEndAfterStartTimeConstraint {
			return ErrEndTimeBeforeStartTime
		}
		return fmt.Errorf("events.Repository.updateSeriesEvents: %w", err)
	}
	return nil
}

// addEventTimeFields adds the meet/start/end time trio and the
// meet-time-mandatory flag -- split out of buildEventUpdateSets purely to
// keep that function's cyclomatic complexity under the repo's golangci-lint
// threshold.
func addEventTimeFields(b *sqlbuilder.Builder, params *UpdateEventParams) {
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
	if params.ClearEndDate {
		b.Add("end_date", nil)
	} else if params.EndDate != nil {
		b.Add("end_date", *params.EndDate)
	}
	if params.Location != nil {
		b.Add("location", *params.Location)
	}
	if params.Note != nil {
		b.Add("note", *params.Note)
	}
	addEventTimeFields(b, params)
	if params.ResponseMode != nil {
		b.Add("response_mode", *params.ResponseMode)
	}
	if params.NominatedRoleIds != nil {
		b.Add("nominated_role_ids", params.NominatedRoleIds)
	}
	if params.CancelLeadMinutes != nil {
		b.Add("cancel_lead_minutes", *params.CancelLeadMinutes)
	}
	if params.ExcludeFromStats != nil {
		b.Add("exclude_from_stats", *params.ExcludeFromStats)
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
		seriesQ := fmt.Sprintf(`SELECT series_id FROM events e WHERE e.id = $1 AND %s`, eventScopedByAnyTargetTeam("$2"))
		err := tx.QueryRow(ctx, seriesQ, eventID, teamID).Scan(&seriesID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("events.Repository.SetStatus: get series_id: %w", err)
		}
		if seriesID != nil {
			// Only today's and future instances are affected. Bulk-changing the
			// status of already-held (past) occurrences would retroactively
			// rewrite team history — e.g. cancelling "the rest of the series"
			// must not flip completed trainings to cancelled and drop them from
			// stats. The event addressed by eventID is still updated
			// individually below regardless of its date. Scoped via
			// eventScopedByAnyTargetTeam rather than team_id = $3 so a caller
			// reaching this event through a non-owning targeted team's URL
			// still cascades across the rest of the (shared) series.
			seriesUpdateQ := fmt.Sprintf(`UPDATE events e SET status = $1 WHERE e.series_id = $2 AND e.date >= CURRENT_DATE AND %s`, eventScopedByAnyTargetTeam("$3"))
			_, err = tx.Exec(ctx, seriesUpdateQ, status, seriesID, teamID)
			if err != nil {
				return nil, fmt.Errorf("events.Repository.SetStatus: update series: %w", err)
			}
		}
	}

	q := fmt.Sprintf(`UPDATE events e SET status = $1 WHERE e.id = $2 AND %s RETURNING %s`, eventScopedByAnyTargetTeam("$3"), selectEventFields)
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

// eventTeamMembersLateral is a LATERAL join producing one row per distinct
// user across every team event alias "e" targets (via event_teams, aliased
// "m" to match absenceCoversExpr/effectiveStatusExpr's expected alias),
// deduplicated by user_id -- a person in two targeted teams is counted
// once, matching attendance's own UNIQUE(event_id, user_id). When a user
// belongs to more than one of the event's targeted teams, the membership
// matching viewingTeamPlaceholder (the caller's own viewing team) is
// preferred -- so the absence lookup inside absenceCoversExpr, keyed off
// m.team_id, resolves against the viewer's own team whenever possible;
// otherwise an arbitrary one of the targeted teams is picked. For a
// single-team event (exactly one event_teams row, its owning team -- see
// migration 00035) this always reduces to exactly that team's membership
// list, identical to the old "JOIN memberships m ON m.team_id = e.team_id"
// it replaces across GetAttendanceSummary/GetAttendanceSummaries.
func eventTeamMembersLateral(viewingTeamPlaceholder string) string {
	return `
		JOIN LATERAL (
			SELECT DISTINCT ON (ms.user_id) ms.user_id, ms.team_id
			FROM memberships ms
			JOIN event_teams et ON et.team_id = ms.team_id AND et.event_id = e.id
			ORDER BY ms.user_id, (ms.team_id = ` + viewingTeamPlaceholder + `) DESC, ms.team_id
		) m ON true
	`
}

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
			` + eventTeamMembersLateral("$2") + `
			LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = m.user_id
			WHERE e.id = $1 AND ` + eventScopedByAnyTargetTeam("$2") + `
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
	// ab.team_id = $3 (the viewing team), not e.team_id (the event's owning
	// team): for a cross-team event viewed through a non-owning targeted
	// team, the caller's planned absence is recorded against their own
	// team, not the event's owning team. For a single-team event $3 can
	// only ever be e.team_id anyway (the WHERE scope below admits no other
	// value), so this is unchanged behavior there.
	q := `
		SELECT a.status, a.reason, a.reason_id, a.reason_visibility, a.at,
		       EXISTS (
		           SELECT 1 FROM absences ab
		           WHERE ab.user_id = $2 AND ab.team_id = $3
		             AND ab.from_date <= COALESCE(e.end_date, e.date) AND ab.to_date >= e.date
		       ),
		       e.response_mode
		FROM events e
		LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = $2
		WHERE e.id = $1 AND ` + eventScopedByAnyTargetTeam("$3") + `
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
// GetAttendanceSummary query per event; all eventIDs in a single call are
// visible to teamID (ListEvents' own team-scoped query already guarantees
// this), which GetAttendanceSummaries uses only to break ties in
// eventTeamMembersLateral for a cross-team event's multi-team members --
// not as an additional visibility filter.
func (r *Repository) GetAttendanceSummaries(ctx context.Context, eventIDs []uuid.UUID, teamID string) (map[uuid.UUID]EventSummaryData, error) {
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
			` + eventTeamMembersLateral("$2") + `
			LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = m.user_id
			WHERE e.id = ANY($1)
		) sub
		GROUP BY id
	`
	rows, err := r.pool.Query(ctx, q, eventIDs, teamID)
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
func (r *Repository) GetMyEffectiveAttendances(ctx context.Context, eventIDs []uuid.UUID, userID, teamID string) (map[uuid.UUID]EffectiveAttendance, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out := make(map[uuid.UUID]EffectiveAttendance, len(eventIDs))
	if len(eventIDs) == 0 {
		return out, nil
	}
	// ab.team_id = $3 (the viewing team), not e.team_id (the event's owning
	// team) -- mirrors GetMyEffectiveAttendance's identical reasoning: for a
	// cross-team event viewed through a non-owning targeted team, the
	// caller's planned absence is recorded against their own team.
	q := `
		SELECT e.id, a.status, a.reason, a.reason_id, a.reason_visibility, a.at,
		       EXISTS (
		           SELECT 1 FROM absences ab
		           WHERE ab.user_id = $2 AND ab.team_id = $3
		             AND ab.from_date <= COALESCE(e.end_date, e.date) AND ab.to_date >= e.date
		       ),
		       e.response_mode
		FROM events e
		LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = $2
		WHERE e.id = ANY($1)
	`
	rows, err := r.pool.Query(ctx, q, eventIDs, userID, teamID)
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
	// Roster-driven off event_teams rather than a single team_id: for a
	// cross-team event this is the union of every targeted team's
	// membership list, deduplicated by user_id via DISTINCT ON (a person in
	// two targeted teams appears once, matching attendance's own
	// UNIQUE(event_id, user_id)) -- for a single-team event (exactly one
	// event_teams row, its owning team) this reduces to exactly the old
	// "JOIN memberships m ON m.team_id = e.team_id" roster, unchanged. When
	// a user has more than one targeted-team membership, the one matching
	// teamID ($2, the viewer's own team) is preferred as the "identity" row
	// (group/title/membershipId/absence lookup) whenever it exists; the
	// outer query still orders the final roster by name, independent of
	// that tie-break.
	q := `
		SELECT * FROM (
			SELECT DISTINCT ON (m.user_id)
				m.id,
				m.user_id,
				m."group",
				m.title,
				u.name,
				u.avatar_color,
				(u.photo_object_key IS NOT NULL) AS has_photo,
				a.status,
				a.reason,
				a.reason_id,
				a.reason_visibility,
				a.at,
				` + absenceCoversExpr + `,
				e.response_mode
			FROM events e
			JOIN event_teams et ON et.event_id = e.id
			JOIN memberships m ON m.team_id = et.team_id
			JOIN users u ON u.id = m.user_id
			LEFT JOIN attendance a ON a.event_id = e.id AND a.user_id = m.user_id
			WHERE e.id = $1 AND ` + eventScopedByAnyTargetTeam("$2") + `
			ORDER BY m.user_id, (m.team_id = $2) DESC, m.id
		) roster
		ORDER BY name ASC
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
			&a.MembershipId, &a.UserId, &a.Group, &a.Title,
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
// on this very hot path. The events EXISTS clause also re-checks status !=
// 'cancelled' for the same reason: the service layer's earlier GetEvent read
// of the event's status is not atomic with this write, so a concurrent
// SetStatus(cancelled) committing between that read and this write must not
// be able to still let attendance be recorded/rewritten against an
// already-cancelled event. The same clause also re-checks the event's
// cancel_lead_minutes cutoff: the service layer's earlier deadline check
// (Service.SetAttendance) is likewise not atomic with this write, so a
// request that raced past the cutoff (or a concurrent SetRoles granting
// events:write mid-request) must not slip through here even if it slipped
// past that earlier check -- caller_write (computed once, below) backs both
// the "acting on another member" bypass and the deadline bypass, since both
// are "does callerID currently hold events:write". The cancel_lead_minutes
// cutoff is computed the same way as EventStartInstant/ZonedTimeToUTC (Go):
// date + COALESCE(start_time, meet_time, '18:00') interpreted as
// Europe/Berlin wall-clock, converted to UTC, minus cancel_lead_minutes.
// Returns pgx.ErrNoRows if eventID does not belong to teamID, if the event
// is cancelled, if the deadline has passed and callerID lacks events:write,
// if userID is not a member of teamID (prevents forging attendance rows for
// arbitrary users outside the team), OR -- in that narrow race -- if
// callerID no longer holds events:write; these are deliberately not
// distinguished here, matching how every other reason this returns
// pgx.ErrNoRows is already ambiguous by design.
func (r *Repository) SetAttendance(ctx context.Context, eventID, callerID, userID, teamID string, status, reason, reasonID, reasonVisibility *string) (*AttendanceDBRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		WITH caller_write AS (
			SELECT EXISTS (
			        SELECT 1 FROM roles r
			        JOIN membership_roles mr ON mr.role_id = r.id
			        JOIN memberships m ON m.id = mr.membership_id
			        WHERE m.team_id = $7 AND m.user_id = $8 AND r.team_id = $7
			          AND r.permissions->>'events' = 'write'
			      ) AS has_write
		)
		INSERT INTO attendance (event_id, user_id, status, reason, reason_id, reason_visibility, at)
		SELECT $1, $2, $3, $4, $5, $6, now()
		FROM caller_write
		WHERE EXISTS (
		        SELECT 1 FROM events e
		        WHERE e.id = $1 AND ` + eventScopedByAnyTargetTeam("$7") + `AND e.status != 'cancelled'
		          AND (
		                e.cancel_lead_minutes IS NULL
		                OR now() <= (
		                     (e.date::timestamp + COALESCE(e.start_time, e.meet_time, '18:00'::time))
		                     AT TIME ZONE 'Europe/Berlin'
		                   ) - (e.cancel_lead_minutes * INTERVAL '1 minute')
		                OR caller_write.has_write
		              )
		      )
		  AND EXISTS (SELECT 1 FROM memberships WHERE team_id = $7 AND user_id = $2)
		  AND ($8 = $2 OR caller_write.has_write)
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
			WHERE EXISTS (SELECT 1 FROM events e WHERE e.id = $1 AND ` + eventScopedByAnyTargetTeam("$3") + `)
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
		   AND a.status = 'not_nominated' AND `+eventScopedByAnyTargetTeam("$3")+`
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
			    EXISTS(SELECT 1 FROM events e WHERE e.id = $1 AND `+eventScopedByAnyTargetTeam("$2")+`),
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
		WHERE c.event_id = $1 AND `+eventScopedByAnyTargetTeam("$2")+`
	`, eventID, teamID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("events.Repository.CountComments: %w", err)
	}
	return count, nil
}

// CommentCursor is the keyset position for comment pagination. It matches the
// ORDER BY created_at ASC, id ASC ordering used by ListComments, so
// (created_at, id) is a unique, stable sort key even when several comments
// share the same created_at timestamp.
type CommentCursor struct {
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"i"`
}

// ListComments returns up to limit comments for an event scoped to teamID,
// oldest-first, starting after cur (nil = first page). It is a keyset query:
// no OFFSET, so a busy event's later pages stay as cheap as the first.
func (r *Repository) ListComments(ctx context.Context, eventID, teamID string, limit int, cur *CommentCursor) ([]CommentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var (
		hasCursor  bool
		curCreated time.Time
		curID      uuid.UUID
	)
	if cur != nil {
		hasCursor = true
		curCreated = cur.CreatedAt
		curID = cur.ID
	}

	// author_membership_id is deliberately joined against $2 (the viewing
	// team), not the event's owning team: it must only resolve to a
	// membership the viewer's own team can navigate to a profile for, never
	// a foreign commenter's membership in some other targeted team (see
	// design.md's "no profile access" rule, applied here too even though
	// comments themselves aren't a display-rule-badged view).
	q := `
		SELECT
			c.id, c.event_id, c.user_id, c.text, c.created_at,
			u.name,
			u.avatar_color,
			(u.photo_object_key IS NOT NULL) AS has_photo,
			m.id AS author_membership_id
		FROM event_comments c
		JOIN users u ON u.id = c.user_id
		JOIN events e ON e.id = c.event_id
		LEFT JOIN memberships m ON m.user_id = c.user_id AND m.team_id = $2
		WHERE c.event_id = $1 AND ` + eventScopedByAnyTargetTeam("$2") + `
		  AND ($3::boolean IS FALSE
		       OR (c.created_at, c.id) > ($4::timestamptz, $5::uuid))
		ORDER BY c.created_at ASC, c.id ASC
		LIMIT $6
	`
	rows, err := r.pool.Query(ctx, q, eventID, teamID, hasCursor, curCreated, curID, limit)
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
			&c.ActorName, &c.ActorColor, &hasPhoto, &c.AuthorMembershipId,
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
			WHERE EXISTS (SELECT 1 FROM events e WHERE e.id = $1 AND ` + eventScopedByAnyTargetTeam("$4") + `)
			  AND EXISTS (SELECT 1 FROM memberships WHERE team_id = $4 AND user_id = $2)
			RETURNING id, event_id, user_id, text, created_at
		)
		SELECT
			i.id, i.event_id, i.user_id, i.text, i.created_at,
			u.name, u.avatar_color,
			(u.photo_object_key IS NOT NULL) AS has_photo,
			m.id AS author_membership_id
		FROM inserted i
		JOIN users u ON u.id = i.user_id
		LEFT JOIN memberships m ON m.user_id = i.user_id AND m.team_id = $4
	`
	c := &CommentRow{}
	var hasPhoto bool
	err := r.pool.QueryRow(ctx, q, eventID, userID, text, teamID).Scan(
		&c.Id, &c.EventId, &c.UserId, &c.Text, &c.CreatedAt,
		&c.ActorName, &c.ActorColor, &hasPhoto, &c.AuthorMembershipId,
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
		 WHERE c.event_id = e.id AND c.id = $1 AND c.user_id = $2 AND `+eventScopedByAnyTargetTeam("$3"),
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
