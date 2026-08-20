package audit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/audit"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestRecord_EmitsStableSchema(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	audit.New(logger).Record(
		context.Background(),
		audit.EventLogin,
		audit.Success,
		"user-1",
		slog.String("email", "a@b.c"),
	)

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, true, rec["audit"])
	assert.Equal(t, "auth.login", rec["event"])
	assert.Equal(t, "success", rec["outcome"])
	assert.Equal(t, "user-1", rec["actor"])
	assert.Equal(t, "a@b.c", rec["email"])
}

func TestRecord_FailureOutcomeAndEmptyActor(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	audit.New(logger).Record(context.Background(), audit.EventLogout, audit.Failure, "")

	var rec map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rec))
	assert.Equal(t, "auth.logout", rec["event"])
	assert.Equal(t, "failure", rec["outcome"])
	assert.Equal(t, "", rec["actor"])
}

func TestNew_NilLoggerFallsBack(t *testing.T) {
	// Should not panic when constructed with a nil logger.
	assert.NotPanics(t, func() {
		audit.New(nil).Record(context.Background(), audit.EventLogin, audit.Success, "u")
	})
}

// TestRecord_PersistsToDB exercises the actual persistToDB codepath (attrs
// JSON marshaling + INSERT INTO audit_log) against a real Postgres, since
// TestRecord_EmitsStableSchema and TestRecord_FailureOutcomeAndEmptyActor
// above only exercise Record without a DB pool attached. This is the
// compliance-retention persistence path described as authoritative by the
// package doc comment.
func TestRecord_PersistsToDB(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	audit.New(logger).WithDB(pool).Record(
		ctx,
		audit.EventRoleUpdate,
		audit.Success,
		"user-42",
		slog.String("role_id", "role-1"),
		slog.Float64("permission_count", 3),
	)

	var event, outcome string
	var actorID *string
	var attrsJSON []byte
	err := pool.QueryRow(ctx,
		`SELECT event, outcome, actor_id, attrs FROM audit_log WHERE event = $1`,
		string(audit.EventRoleUpdate),
	).Scan(&event, &outcome, &actorID, &attrsJSON)
	require.NoError(t, err)

	assert.Equal(t, string(audit.EventRoleUpdate), event)
	assert.Equal(t, string(audit.Success), outcome)
	require.NotNil(t, actorID)
	assert.Equal(t, "user-42", *actorID)

	var attrs map[string]any
	require.NoError(t, json.Unmarshal(attrsJSON, &attrs))
	assert.Equal(t, "role-1", attrs["role_id"])
	assert.Equal(t, float64(3), attrs["permission_count"])
}

// TestRecord_PersistsToDB_EmptyActorIsNull confirms the "" actor sentinel used
// for unauthenticated/unknown actors is stored as SQL NULL, not the literal
// empty string, since actorVal in persistToDB is nil-vs-pointer branching
// on actor == "".
func TestRecord_PersistsToDB_EmptyActorIsNull(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))

	audit.New(logger).WithDB(pool).Record(ctx, audit.EventLogout, audit.Failure, "")

	var outcome string
	var actorID *string
	err := pool.QueryRow(ctx,
		`SELECT outcome, actor_id FROM audit_log WHERE event = $1`,
		string(audit.EventLogout),
	).Scan(&outcome, &actorID)
	require.NoError(t, err)

	assert.Equal(t, string(audit.Failure), outcome)
	assert.Nil(t, actorID)
}
