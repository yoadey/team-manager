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
	"log/slog"
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
	EventLogin        = "auth.login"
	EventLogout       = "auth.logout"
	EventAccountErase = "auth.account_erase"

	EventRoleCreate = "role.create"
	EventRoleUpdate = "role.update"
	EventRoleDelete = "role.delete"

	EventMemberAdd         = "member.add"
	EventMemberUpdate      = "member.update"
	EventMemberRemove      = "member.remove"
	EventMemberRolesChange = "member.roles_change"

	// EventFinanceMutation covers all financial write operations; the specific
	// action is carried in an "operation" attribute (e.g. transaction.create).
	EventFinanceMutation = "finance.mutation"
)

// Logger emits audit records over an slog.Logger.
type Logger struct {
	l *slog.Logger
}

// New returns an audit Logger writing to l. A nil l falls back to slog.Default.
func New(l *slog.Logger) *Logger {
	if l == nil {
		l = slog.Default()
	}
	return &Logger{l: l}
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
}
