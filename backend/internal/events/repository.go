package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

// ListEvents returns events for a team filtered by scope.
func (r *Repository) ListEvents(ctx context.Context, teamID, scope string, limit, offset int) ([]EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	today := time.Now().UTC()

	var (
		q    string
		args []any
	)

	switch scope {
	case "past":
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 AND date < $2 ORDER BY date DESC LIMIT $3 OFFSET $4`, selectEventFields)
		args = []any{teamID, today, limit, offset}
	case "upcoming":
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 AND date >= $2 ORDER BY date ASC LIMIT $3 OFFSET $4`, selectEventFields)
		args = []any{teamID, today, limit, offset}
	default:
		q = fmt.Sprintf(`SELECT %s FROM events WHERE team_id = $1 ORDER BY date ASC LIMIT $2 OFFSET $3`, selectEventFields)
		args = []any{teamID, limit, offset}
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

// GetEvent retrieves a single event by ID.
func (r *Repository) GetEvent(ctx context.Context, eventID string) (*EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := fmt.Sprintf(`SELECT %s FROM events WHERE id = $1`, selectEventFields)
	row := r.pool.QueryRow(ctx, q, eventID)
	e, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("events.Repository.GetEvent: %w", err)
	}
	return e, nil
}

// ─── CreateEvent ────────────────────────────────────────────────────────────

// CreateEvent inserts a single event row and returns it.
func (r *Repository) CreateEvent(ctx context.Context, teamID string, params *CreateEventParams) (*EventRow, error) { //nolint:gocritic
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
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

	row := r.pool.QueryRow(
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

	// Insert event instances.
	eventQ := fmt.Sprintf(`
		INSERT INTO events (
			team_id, series_id, type, title, date, location, note,
			meet_time, start_time, end_time, meet_time_mandatory,
			response_mode, nominated_role_ids, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8::time, $9::time, $10::time, $11,
			$12, $13, 'active'
		)
		RETURNING %s
	`, selectEventFields)

	var events []EventRow
	for i := 0; i < repeatWeeks; i++ {
		eventDate := params.Date.AddDate(0, 0, i*7)
		row := tx.QueryRow(
			ctx, eventQ,
			teamID, seriesID, params.Type, params.Title, eventDate,
			params.Location, params.Note,
			nullableTime(params.MeetTime), nullableTime(params.StartTime), nullableTime(params.EndTime),
			boolVal(params.MeetTimeMandatory),
			strVal(params.ResponseMode, "opt_in"),
			uuidSlice(params.NominatedRoleIds),
		)
		e, err := scanEventRow(row)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.CreateSeries: insert event %d: %w", i, err)
		}
		events = append(events, *e)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("events.Repository.CreateSeries: commit: %w", err)
	}
	return events, nil
}

// ─── UpdateEvent ────────────────────────────────────────────────────────────

// UpdateEvent updates a single event or all events in its series.
func (r *Repository) UpdateEvent(ctx context.Context, eventID string, params *UpdateEventParams, scope string) (*EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if scope == "series" {
		// Get series_id for this event.
		var seriesID *uuid.UUID
		err := r.pool.QueryRow(ctx, `SELECT series_id FROM events WHERE id = $1`, eventID).Scan(&seriesID)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.UpdateEvent: get series_id: %w", err)
		}
		if seriesID != nil {
			if err := r.updateSeriesEvents(ctx, seriesID.String(), params); err != nil {
				return nil, err
			}
		}
	}

	// Always update the specific event and return it.
	sets, args := buildUpdateSets(params, eventID)
	q := fmt.Sprintf(`UPDATE events SET %s WHERE id = $%d RETURNING %s`, sets, len(args), selectEventFields)
	row := r.pool.QueryRow(ctx, q, args...)
	e, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("events.Repository.UpdateEvent: %w", err)
	}
	return e, nil
}

func (r *Repository) updateSeriesEvents(ctx context.Context, seriesID string, params *UpdateEventParams) error {
	sets, args := buildUpdateSets(params, "")
	// Remove last arg (the eventID placeholder we added) — we use series_id instead.
	args = args[:len(args)-1]
	q := fmt.Sprintf(`UPDATE events SET %s WHERE series_id = $%d`, sets, len(args)+1)
	args = append(args, seriesID)
	_, err := r.pool.Exec(ctx, q, args...)
	if err != nil {
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

// SetStatus updates event status for a single event or all events in its series.
func (r *Repository) SetStatus(ctx context.Context, eventID, status, scope string) (*EventRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if scope == "series" {
		var seriesID *uuid.UUID
		err := r.pool.QueryRow(ctx, `SELECT series_id FROM events WHERE id = $1`, eventID).Scan(&seriesID)
		if err != nil {
			return nil, fmt.Errorf("events.Repository.SetStatus: get series_id: %w", err)
		}
		if seriesID != nil {
			_, err = r.pool.Exec(ctx, `UPDATE events SET status = $1 WHERE series_id = $2`, status, seriesID)
			if err != nil {
				return nil, fmt.Errorf("events.Repository.SetStatus: update series: %w", err)
			}
		}
	}

	q := fmt.Sprintf(`UPDATE events SET status = $1 WHERE id = $2 RETURNING %s`, selectEventFields)
	row := r.pool.QueryRow(ctx, q, status, eventID)
	e, err := scanEventRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("events.Repository.SetStatus: %w", err)
	}
	return e, nil
}

// ─── DeleteEvent ────────────────────────────────────────────────────────────

// DeleteEvent deletes a single event or the entire series (cascade).
func (r *Repository) DeleteEvent(ctx context.Context, eventID, scope string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if scope == "series" {
		var seriesID *uuid.UUID
		err := r.pool.QueryRow(ctx, `SELECT series_id FROM events WHERE id = $1`, eventID).Scan(&seriesID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("events.Repository.DeleteEvent: get series_id: %w", err)
		}
		if seriesID != nil {
			_, err = r.pool.Exec(ctx, `DELETE FROM event_series WHERE id = $1`, seriesID)
			if err != nil {
				return fmt.Errorf("events.Repository.DeleteEvent: delete series: %w", err)
			}
			return nil // cascade deletes events
		}
	}

	_, err := r.pool.Exec(ctx, `DELETE FROM events WHERE id = $1`, eventID)
	if err != nil {
		return fmt.Errorf("events.Repository.DeleteEvent: %w", err)
	}
	return nil
}

// ─── Attendance Summary ──────────────────────────────────────────────────────

// GetAttendanceSummary returns aggregated attendance counts for an event.
func (r *Repository) GetAttendanceSummary(ctx context.Context, eventID string) (EventSummaryData, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'yes')           AS yes,
			COUNT(*) FILTER (WHERE status = 'no')            AS no,
			COUNT(*) FILTER (WHERE status = 'maybe')         AS maybe,
			COUNT(*) FILTER (WHERE status = 'pending')       AS pending,
			COUNT(*) FILTER (WHERE status = 'not_nominated') AS not_nominated,
			COUNT(*) FILTER (WHERE status != 'not_nominated') AS nominated,
			COUNT(*)                                          AS total
		FROM attendance
		WHERE event_id = $1
	`
	var s EventSummaryData
	err := r.pool.QueryRow(ctx, q, eventID).Scan(
		&s.Yes, &s.No, &s.Maybe, &s.Pending, &s.NotNominated, &s.Nominated, &s.Total,
	)
	if err != nil {
		return s, fmt.Errorf("events.Repository.GetAttendanceSummary: %w", err)
	}
	return s, nil
}

// ─── MyAttendance ───────────────────────────────────────────────────────────

// GetMyAttendance returns the current user's attendance record for an event, or nil.
func (r *Repository) GetMyAttendance(ctx context.Context, eventID, userID string) (*AttendanceDBRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		SELECT id, event_id, user_id, status, reason, reason_id, reason_visibility, at
		FROM attendance
		WHERE event_id = $1 AND user_id = $2
	`
	row := r.pool.QueryRow(ctx, q, eventID, userID)
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

// ─── ListAttendance ─────────────────────────────────────────────────────────

// ListAttendance returns all attendance rows for an event, enriched with user data.
func (r *Repository) ListAttendance(ctx context.Context, eventID string) ([]AttendanceEnriched, error) {
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
		WHERE a.event_id = $1
		ORDER BY u.name ASC
	`
	rows, err := r.pool.Query(ctx, q, eventID)
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

// ─── SetAttendance ──────────────────────────────────────────────────────────

// SetAttendance upserts an attendance record.
func (r *Repository) SetAttendance(ctx context.Context, eventID, userID string, status, reason, reasonID, reasonVisibility *string) (*AttendanceDBRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		INSERT INTO attendance (event_id, user_id, status, reason, reason_id, reason_visibility, at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (event_id, user_id) DO UPDATE
			SET status = EXCLUDED.status,
			    reason = EXCLUDED.reason,
			    reason_id = EXCLUDED.reason_id,
			    reason_visibility = EXCLUDED.reason_visibility,
			    at = now()
		RETURNING id, event_id, user_id, status, reason, reason_id, reason_visibility, at
	`
	a := &AttendanceDBRow{}
	err := r.pool.QueryRow(ctx, q, eventID, userID, status, reason, reasonID, reasonVisibility).Scan(
		&a.Id, &a.EventId, &a.UserId, &a.Status, &a.Reason, &a.ReasonId, &a.ReasonVisibility, &a.At,
	)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.SetAttendance: %w", err)
	}
	return a, nil
}

// ─── SetNomination ──────────────────────────────────────────────────────────

// SetNomination sets or removes nomination for a user on an event.
// nominated=false → upsert status=not_nominated
// nominated=true  → delete any not_nominated record for this user/event.
func (r *Repository) SetNomination(ctx context.Context, eventID, userID string, nominated bool) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if !nominated {
		q := `
			INSERT INTO attendance (event_id, user_id, status, at)
			VALUES ($1, $2, 'not_nominated', now())
			ON CONFLICT (event_id, user_id) DO UPDATE
				SET status = 'not_nominated', at = now()
		`
		_, err := r.pool.Exec(ctx, q, eventID, userID)
		if err != nil {
			return fmt.Errorf("events.Repository.SetNomination(false): %w", err)
		}
		return nil
	}

	// Remove not_nominated record so the user reverts to pending/default.
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM attendance WHERE event_id = $1 AND user_id = $2 AND status = 'not_nominated'`,
		eventID, userID,
	)
	if err != nil {
		return fmt.Errorf("events.Repository.SetNomination(true): %w", err)
	}
	return nil
}

// ─── Comments ───────────────────────────────────────────────────────────────

// ListComments returns all comments for an event, enriched with user data.
func (r *Repository) ListComments(ctx context.Context, eventID string, limit, offset int) ([]CommentRow, error) {
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
		WHERE c.event_id = $1
		ORDER BY c.created_at ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, q, eventID, limit, offset)
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

// AddComment inserts a new event comment and returns it enriched.
func (r *Repository) AddComment(ctx context.Context, eventID, userID, text string) (*CommentRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := `
		WITH inserted AS (
			INSERT INTO event_comments (event_id, user_id, text)
			VALUES ($1, $2, $3)
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
	err := r.pool.QueryRow(ctx, q, eventID, userID, text).Scan(
		&c.Id, &c.EventId, &c.UserId, &c.Text, &c.CreatedAt,
		&c.ActorName, &c.ActorColor, &hasPhoto,
	)
	if err != nil {
		return nil, fmt.Errorf("events.Repository.AddComment: %w", err)
	}
	c.HasActorPhoto = &hasPhoto
	return c, nil
}

// DeleteComment deletes a comment if the requesting user owns it.
func (r *Repository) DeleteComment(ctx context.Context, commentID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tag, err := r.pool.Exec(
		ctx,
		`DELETE FROM event_comments WHERE id = $1 AND user_id = $2`,
		commentID, userID,
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
