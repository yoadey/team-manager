package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ExportData is the GDPR Art. 15 personal-data export for one user. It is
// marshaled to JSON as the export document; amounts and dates are rendered as
// strings for a clean, locale-independent dump.
type ExportData struct {
	ExportedAt         time.Time                 `json:"exportedAt"`
	Profile            ExportProfile             `json:"profile"`
	Memberships        []ExportMembership        `json:"memberships"`
	Attendance         []ExportAttendance        `json:"attendance"`
	Comments           []ExportComment           `json:"comments"`
	Absences           []ExportAbsence           `json:"absences"`
	AuthoredNews       []ExportNews              `json:"authoredNews"`
	CreatedPolls       []ExportPoll              `json:"createdPolls"`
	PollVotes          []ExportPollVote          `json:"pollVotes"`
	PenaltyAssignments []ExportPenaltyAssignment `json:"penaltyAssignments"`
	Contributions      []ExportContribution      `json:"contributions"`
}

type ExportProfile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       *string   `json:"phone,omitempty"`
	AvatarColor string    `json:"avatarColor"`
	Birthday    *string   `json:"birthday,omitempty"`
	Address     *string   `json:"address,omitempty"`
	HasPhoto    bool      `json:"hasPhoto"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ExportMembership struct {
	TeamID   string    `json:"teamId"`
	TeamName string    `json:"teamName"`
	JoinedAt time.Time `json:"joinedAt"`
	Roles    []string  `json:"roles"`
}

type ExportAttendance struct {
	EventID    string  `json:"eventId"`
	EventTitle string  `json:"eventTitle"`
	EventDate  string  `json:"eventDate"`
	Status     string  `json:"status"`
	Reason     *string `json:"reason,omitempty"`
}

type ExportComment struct {
	EventID   string    `json:"eventId"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExportAbsence struct {
	TeamID    string    `json:"teamId"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Reason    *string   `json:"reason,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExportNews struct {
	TeamID    string    `json:"teamId"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExportPoll struct {
	TeamID    string    `json:"teamId"`
	Question  string    `json:"question"`
	CreatedAt time.Time `json:"createdAt"`
}

type ExportPollVote struct {
	PollID   string `json:"pollId"`
	Question string `json:"question"`
	Option   string `json:"option"`
}

type ExportPenaltyAssignment struct {
	TeamID string `json:"teamId"`
	Label  string `json:"label"`
	Amount string `json:"amount"`
	Paid   bool   `json:"paid"`
	Date   string `json:"date"`
}

type ExportContribution struct {
	TeamID     string  `json:"teamId"`
	Name       string  `json:"name"`
	DueDate    *string `json:"dueDate,omitempty"`
	Amount     string  `json:"amount"`
	PaidAmount string  `json:"paidAmount"`
	Status     string  `json:"status"`
}

// queryExportRows runs a single-parameter ($1 = userID) query and collects each
// row via scan. The result is always a non-nil slice so the export renders [].
func queryExportRows[T any](ctx context.Context, q func(context.Context, string, ...any) (pgx.Rows, error), sql, userID string, scan func(pgx.Rows) (T, error)) ([]T, error) {
	rows, err := q(ctx, sql, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	out := make([]T, 0)
	for rows.Next() {
		v, scanErr := scan(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan: %w", scanErr)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

// ExportUserData gathers every personal-data record tied to userID into a single
// export document (GDPR Art. 15). All queries are read-only.
func (r *Repository) ExportUserData(ctx context.Context, userID string) (*ExportData, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	out := &ExportData{ExportedAt: time.Now().UTC()}

	if err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, email, phone, avatar_color, birthday::text, address,
		       (photo_object_key IS NOT NULL AND length(photo_object_key) > 0), created_at
		FROM users WHERE id = $1
	`, userID).Scan(
		&out.Profile.ID, &out.Profile.Name, &out.Profile.Email, &out.Profile.Phone,
		&out.Profile.AvatarColor, &out.Profile.Birthday, &out.Profile.Address,
		&out.Profile.HasPhoto, &out.Profile.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: profile: %w", err)
	}

	var err error
	if out.Memberships, err = queryExportRows(ctx, r.pool.Query, `
		SELECT m.team_id::text, t.name, m.joined_at,
		       COALESCE(array_agg(r.name) FILTER (WHERE r.name IS NOT NULL), '{}')
		FROM memberships m
		JOIN teams t ON t.id = m.team_id
		LEFT JOIN membership_roles mr ON mr.membership_id = m.id
		LEFT JOIN roles r ON r.id = mr.role_id
		WHERE m.user_id = $1
		GROUP BY m.team_id, t.name, m.joined_at
		ORDER BY m.joined_at
	`, userID, func(rows pgx.Rows) (ExportMembership, error) {
		var m ExportMembership
		return m, rows.Scan(&m.TeamID, &m.TeamName, &m.JoinedAt, &m.Roles)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: memberships: %w", err)
	}

	if out.Attendance, err = queryExportRows(ctx, r.pool.Query, `
		SELECT a.event_id::text, e.title, e.date::text, a.status, a.reason
		FROM attendance a JOIN events e ON e.id = a.event_id
		WHERE a.user_id = $1 ORDER BY e.date
	`, userID, func(rows pgx.Rows) (ExportAttendance, error) {
		var a ExportAttendance
		return a, rows.Scan(&a.EventID, &a.EventTitle, &a.EventDate, &a.Status, &a.Reason)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: attendance: %w", err)
	}

	if out.Comments, err = queryExportRows(ctx, r.pool.Query, `
		SELECT event_id::text, text, created_at FROM event_comments
		WHERE user_id = $1 ORDER BY created_at
	`, userID, func(rows pgx.Rows) (ExportComment, error) {
		var c ExportComment
		return c, rows.Scan(&c.EventID, &c.Text, &c.CreatedAt)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: comments: %w", err)
	}

	if out.Absences, err = queryExportRows(ctx, r.pool.Query, `
		SELECT team_id::text, from_date::text, to_date::text, reason, created_at
		FROM absences WHERE user_id = $1 ORDER BY from_date
	`, userID, func(rows pgx.Rows) (ExportAbsence, error) {
		var a ExportAbsence
		return a, rows.Scan(&a.TeamID, &a.From, &a.To, &a.Reason, &a.CreatedAt)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: absences: %w", err)
	}

	if out.AuthoredNews, err = queryExportRows(ctx, r.pool.Query, `
		SELECT team_id::text, title, body, created_at FROM news
		WHERE author_id = $1 ORDER BY created_at
	`, userID, func(rows pgx.Rows) (ExportNews, error) {
		var n ExportNews
		return n, rows.Scan(&n.TeamID, &n.Title, &n.Body, &n.CreatedAt)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: news: %w", err)
	}

	if out.CreatedPolls, err = queryExportRows(ctx, r.pool.Query, `
		SELECT team_id::text, question, created_at FROM polls
		WHERE creator_id = $1 ORDER BY created_at
	`, userID, func(rows pgx.Rows) (ExportPoll, error) {
		var p ExportPoll
		return p, rows.Scan(&p.TeamID, &p.Question, &p.CreatedAt)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: polls: %w", err)
	}

	if out.PollVotes, err = queryExportRows(ctx, r.pool.Query, `
		SELECT pv.poll_id::text, p.question, o.text
		FROM poll_votes pv
		JOIN polls p ON p.id = pv.poll_id
		JOIN poll_options o ON o.id = pv.option_id
		WHERE pv.user_id = $1
	`, userID, func(rows pgx.Rows) (ExportPollVote, error) {
		var v ExportPollVote
		return v, rows.Scan(&v.PollID, &v.Question, &v.Option)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: votes: %w", err)
	}

	// paid mirrors finances.Repository.ListAssignments/assignmentPaid's
	// derivation (live sum of linked income transactions vs. the assignment's
	// own snapshotted amount, since migration
	// 00020_penalty_assignment_linked_payment.sql dropped the stored `paid`
	// column) -- kept as a duplicated raw query here rather than importing
	// the finances package, consistent with the contributions query below.
	if out.PenaltyAssignments, err = queryExportRows(ctx, r.pool.Query, `
		SELECT pa.team_id::text, pa.label, pa.amount::text,
		       COALESCE(paidtx.paid_amount, 0) >= pa.amount AS paid,
		       pa.date::text
		FROM penalty_assignments pa
		LEFT JOIN LATERAL (
			SELECT SUM(t.amount) AS paid_amount
			FROM transactions t
			WHERE t.penalty_assignment_id = pa.id AND t.type = 'income'
		) paidtx ON true
		WHERE pa.user_id = $1 ORDER BY pa.date
	`, userID, func(rows pgx.Rows) (ExportPenaltyAssignment, error) {
		var p ExportPenaltyAssignment
		return p, rows.Scan(&p.TeamID, &p.Label, &p.Amount, &p.Paid, &p.Date)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: penalties: %w", err)
	}

	// paidAmount/status mirror finances.Repository.ListContributions'
	// derivation (live sum of linked income transactions, never a stored
	// column) -- kept as a duplicated raw query here rather than importing
	// the finances package, consistent with this file's existing pattern of
	// standalone export queries for every other module (e.g. penalties above).
	if out.Contributions, err = queryExportRows(ctx, r.pool.Query, `
		SELECT c.team_id::text, c.name, c.due_date::text, c.amount::text,
		       COALESCE(pa.paid_amount, 0)::text AS paid_amount,
		       CASE
		         WHEN COALESCE(pa.paid_amount, 0) = 0 THEN 'open'
		         WHEN COALESCE(pa.paid_amount, 0) >= c.amount THEN 'paid'
		         ELSE 'partial'
		       END AS status
		FROM contributions c
		LEFT JOIN LATERAL (
			SELECT SUM(t.amount) AS paid_amount
			FROM transactions t
			WHERE t.contribution_id = c.id AND t.type = 'income'
		) pa ON true
		WHERE c.user_id = $1 ORDER BY c.due_date NULLS LAST, c.name
	`, userID, func(rows pgx.Rows) (ExportContribution, error) {
		var c ExportContribution
		return c, rows.Scan(&c.TeamID, &c.Name, &c.DueDate, &c.Amount, &c.PaidAmount, &c.Status)
	}); err != nil {
		return nil, fmt.Errorf("auth.Repository.ExportUserData: contributions: %w", err)
	}

	return out, nil
}
