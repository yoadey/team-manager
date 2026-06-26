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
