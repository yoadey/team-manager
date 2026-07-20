// Package audit provides structured logging for security-sensitive actions
// (authentication, permission and role changes, financial mutations), distinct
// from the general request log. Every record carries a stable schema —
// audit=true, event, outcome, actor — so the stream can be filtered out of the
// application logs and shipped to a SIEM or retained separately for compliance.
//
// Records intentionally never include secrets (passwords, tokens, cookie
// values); identifiers (user id, email, target id) are expected and required
// for the trail to be useful.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Outcome describes whether the audited action succeeded.
type Outcome string

const (
	// Success marks a completed action.
	Success Outcome = "success"
	// Failure marks a rejected or errored action.
	Failure Outcome = "failure"
)

// Event names for audited actions. Add new constants here as more modules emit
// audit records (e.g. roles, finances).
const (
	EventLogin              = "auth.login"
	EventLogout             = "auth.logout"
	EventAccountErase       = "auth.account_erase"
	EventRegister           = "auth.register"
	EventEmailVerify        = "auth.email_verify"
	EventResendVerification = "auth.resend_verification"

	EventRoleCreate = "role.create"
	EventRoleUpdate = "role.update"
	EventRoleDelete = "role.delete"

	EventMemberUpdate      = "member.update"
	EventMemberRemove      = "member.remove"
	EventMemberRolesChange = "member.roles_change"

	// EventFinanceMutation covers all financial write operations; the specific
	// action is carried in an "operation" attribute (e.g. transaction.create).
	EventFinanceMutation = "finance.mutation"

	EventTeamCreate       = "team.create"
	EventTeamUpdate       = "team.update"
	EventTeamInvite       = "team.invite_create"
	EventTeamInviteAccept = "team.invite_accept"

	// EventTeamBrandingUpdate covers photo/logo upload and delete; the
	// specific action is carried in an "operation" attribute (e.g.
	// photo.upload), mirroring EventFinanceMutation's shape.
	EventTeamBrandingUpdate = "team.branding_update"
)

// Logger emits audit records to the application log and, when a DB pool is
// provided, persists them to the audit_log table for compliance retention.
type Logger struct {
	l    *slog.Logger
	pool *pgxpool.Pool
}

// New returns an audit Logger writing to l. A nil l falls back to slog.Default.
// Call WithDB to also persist records to the database.
func New(l *slog.Logger) *Logger {
	if l == nil {
		l = slog.Default()
	}
	return &Logger{l: l}
}

// WithDB returns a copy of the Logger that also writes audit records to the
// audit_log table. DB write failures are logged but do not return errors to
// callers — the structured log remains the authoritative record.
func (a *Logger) WithDB(pool *pgxpool.Pool) *Logger {
	return &Logger{l: a.l, pool: pool}
}

// Record emits one audit record. actor is the acting user id ("" when unknown
// or unauthenticated); attrs carry event-specific, non-secret context.
func (a *Logger) Record(ctx context.Context, event string, outcome Outcome, actor string, attrs ...slog.Attr) {
	all := make([]slog.Attr, 0, len(attrs)+4)
	all = append(all,
		slog.Bool("audit", true),
		slog.String("event", event),
		slog.String("outcome", string(outcome)),
		slog.String("actor", actor),
	)
	all = append(all, attrs...)
	a.l.LogAttrs(ctx, slog.LevelInfo, "audit", all...)

	if a.pool != nil {
		a.persistToDB(ctx, event, outcome, actor, attrs)
	}
}

// persistToDB writes the audit record to the audit_log table. Errors are
// logged as warnings and do not propagate — the structured log is the primary
// record and must not be blocked by DB unavailability.
func (a *Logger) persistToDB(ctx context.Context, event string, outcome Outcome, actor string, attrs []slog.Attr) {
	attrsMap := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		attrsMap[attr.Key] = attr.Value.Any()
	}
	attrsJSON, err := json.Marshal(attrsMap)
	if err != nil {
		a.l.WarnContext(ctx, "audit: failed to marshal attrs for DB persistence", "err", err)
		return
	}

	var actorVal *string
	if actor != "" {
		actorVal = &actor
	}

	_, err = a.pool.Exec(ctx,
		`INSERT INTO audit_log (event, outcome, actor_id, attrs) VALUES ($1, $2, $3, $4)`,
		event, string(outcome), actorVal, attrsJSON,
	)
	if err != nil {
		a.l.WarnContext(ctx, "audit: failed to persist record to DB", "event", event, "err", err)
	}
}
