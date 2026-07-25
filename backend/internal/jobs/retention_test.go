package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// TestRetentionWorker_DeletesInBatches seeds more rows than a single
// retention batch (1000) to verify deleteBatched loops until the table is
// fully exhausted, rather than only removing the first batch.
func TestRetentionWorker_DeletesInBatches(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Retention Team')`, teamID)
	require.NoError(t, err)

	const oldRowCount = 1500
	oldCutoff := time.Now().Add(-100 * 24 * time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO notifications (team_id, type, created_at)
		SELECT $1, 'news', $2
		FROM generate_series(1, $3)
	`, teamID, oldCutoff, oldRowCount)
	require.NoError(t, err)

	// One recent notification must survive retention.
	_, err = pool.Exec(ctx,
		`INSERT INTO notifications (team_id, type, created_at) VALUES ($1, 'news', now())`, teamID)
	require.NoError(t, err)

	worker := jobs.NewRetentionWorker(pool, 90, 30, 365, 7)
	require.NoError(t, worker.Work(ctx, &river.Job[jobs.RetentionArgs]{}))

	var remaining int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE team_id = $1`, teamID).Scan(&remaining))
	assert.Equal(t, 1, remaining, "only the recent notification should survive retention")
}

// TestRetentionWorker_TimeoutExceedsRiverDefault is a regression test for a
// bug where RetentionWorker relied on WorkerDefaults' zero-value Timeout(),
// so River applied its own JobTimeoutDefault (1 minute) to the job's outer
// context. That outer deadline caps every phase's own context.WithTimeout
// (contexts can only get an earlier deadline, never a later one), so with
// six sequential 30s phase budgets the last phase(s) -- including
// audit_log's compliance-mandated cleanup -- could be starved or cancelled
// mid-batch on a run with a large backlog, silently defeating the whole
// point of giving each phase an independent budget (see
// retentionPhaseTimeout's doc comment).
func TestRetentionWorker_TimeoutExceedsRiverDefault(t *testing.T) {
	t.Parallel()

	worker := jobs.NewRetentionWorker(nil, 90, 30, 365, 7)
	timeout := worker.Timeout(&river.Job[jobs.RetentionArgs]{})
	assert.Greater(t, timeout, river.JobTimeoutDefault,
		"RetentionWorker.Timeout must exceed River's default JobTimeout, or the outer job context caps all six phase timeouts to a shared budget shorter than their sum")
}

// TestRetentionWorker_KeepsStillValidLongLivedSession is a regression test
// for a bug where session retention deleted rows based on created_at instead
// of expires_at: a session created long ago but with a long TTL (still
// valid, expires_at in the future) must survive retention even though its
// created_at is older than the retention window. Only sessions that have
// actually expired (and stayed expired past the retention grace period)
// should be purged.
func TestRetentionWorker_KeepsStillValidLongLivedSession(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Retention User', 'retention@example.com', '#123456')`,
		userID)
	require.NoError(t, err)

	// Created 60 days ago (older than the 30-day retention window) but still
	// valid for another 30 days — must NOT be deleted.
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES ($1, 'still-valid-long-lived', now() - interval '60 days', now() + interval '30 days')
	`, userID)
	require.NoError(t, err)

	// Expired 100 days ago — must be deleted.
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES ($1, 'long-expired', now() - interval '130 days', now() - interval '100 days')
	`, userID)
	require.NoError(t, err)

	worker := jobs.NewRetentionWorker(pool, 90, 30, 365, 7)
	require.NoError(t, worker.Work(ctx, &river.Job[jobs.RetentionArgs]{}))

	var tokens []string
	rows, err := pool.Query(ctx, `SELECT token_hash FROM sessions WHERE user_id = $1`, userID)
	require.NoError(t, err)
	for rows.Next() {
		var tok string
		require.NoError(t, rows.Scan(&tok))
		tokens = append(tokens, tok)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []string{"still-valid-long-lived"}, tokens, "only the still-valid session should survive retention")
}

// Regression test: invites accumulated unboundedly with no cleanup
// mechanism at all -- CreateInvite is called every time the invite sheet is
// opened, with no reuse of unexpired codes. An invite still within its
// (fixed, 7-day) validity window must survive retention even if the row is
// old-ish; only one that expired more than the retention grace period ago
// should be purged.
func TestRetentionWorker_DeletesLongExpiredInvites(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Invite Retention Team')`, teamID)
	require.NoError(t, err)

	// Still within its validity window -- must NOT be deleted.
	_, err = pool.Exec(ctx, `
		INSERT INTO invites (team_id, code, expires_at)
		VALUES ($1, 'still-valid-code', now() + interval '3 days')
	`, teamID)
	require.NoError(t, err)

	// Expired 100 days ago -- must be deleted.
	_, err = pool.Exec(ctx, `
		INSERT INTO invites (team_id, code, expires_at)
		VALUES ($1, 'long-expired-code', now() - interval '100 days')
	`, teamID)
	require.NoError(t, err)

	worker := jobs.NewRetentionWorker(pool, 90, 30, 365, 7)
	require.NoError(t, worker.Work(ctx, &river.Job[jobs.RetentionArgs]{}))

	var codes []string
	rows, err := pool.Query(ctx, `SELECT code FROM invites WHERE team_id = $1`, teamID)
	require.NoError(t, err)
	for rows.Next() {
		var code string
		require.NoError(t, rows.Scan(&code))
		codes = append(codes, code)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []string{"still-valid-code"}, codes, "only the still-valid invite should survive retention")
}

// TestRetentionWorker_DeletesNeverVerifiedAccountsPastCutoff is a regression
// test for the self-registration retention phase: an account that never
// completed email verification must be purged once created_at is older than
// unverifiedAccountRetention, so a squatted email address eventually becomes
// registrable again -- but a recently-created unverified account (still
// within its grace period) and a verified account (regardless of age) must
// both survive. Also verifies that deleting the never-verified user cascades
// to its email_verification_tokens row (ON DELETE CASCADE).
func TestRetentionWorker_DeletesNeverVerifiedAccountsPastCutoff(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	// Never verified, created long before the retention cutoff -- must be
	// deleted, cascading to its verification token.
	staleUnverifiedID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color, email_verified_at, created_at)
		VALUES ($1, 'Stale Unverified', 'stale-unverified@example.com', '#111111', NULL, now() - interval '100 days')
	`, staleUnverifiedID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, 'stale-unverified-token-hash', now() + interval '1 day')
	`, staleUnverifiedID)
	require.NoError(t, err)

	// Never verified but created recently -- must survive (still within the
	// grace period).
	recentUnverifiedID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color, email_verified_at, created_at)
		VALUES ($1, 'Recent Unverified', 'recent-unverified@example.com', '#222222', NULL, now())
	`, recentUnverifiedID)
	require.NoError(t, err)

	// Verified long ago -- must survive regardless of age.
	staleVerifiedID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color, email_verified_at, created_at)
		VALUES ($1, 'Stale Verified', 'stale-verified@example.com', '#333333', now() - interval '100 days', now() - interval '100 days')
	`, staleVerifiedID)
	require.NoError(t, err)

	worker := jobs.NewRetentionWorker(pool, 90, 30, 365, 7)
	require.NoError(t, worker.Work(ctx, &river.Job[jobs.RetentionArgs]{}))

	var remainingEmails []string
	rows, err := pool.Query(ctx, `SELECT email FROM users WHERE id = ANY($1)`,
		[]uuid.UUID{staleUnverifiedID, recentUnverifiedID, staleVerifiedID})
	require.NoError(t, err)
	for rows.Next() {
		var email string
		require.NoError(t, rows.Scan(&email))
		remainingEmails = append(remainingEmails, email)
	}
	require.NoError(t, rows.Err())

	assert.ElementsMatch(t, []string{"recent-unverified@example.com", "stale-verified@example.com"}, remainingEmails,
		"only the never-verified account past the retention cutoff should be deleted")

	var tokenCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM email_verification_tokens WHERE user_id = $1`, staleUnverifiedID,
	).Scan(&tokenCount))
	assert.Equal(t, 0, tokenCount, "deleting the never-verified user must cascade to its verification token")
}

// TestRetentionWorker_DeletesExpiredVerificationTokensButKeepsUnexpired is a
// regression test for the second self-registration retention phase: an
// email_verification_token past its own expires_at must be purged
// independent of the unverified-account grace period above (e.g. a verified
// user's stale token, or a superseded resend), while a still-valid token
// must survive.
func TestRetentionWorker_DeletesExpiredVerificationTokensButKeepsUnexpired(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, avatar_color, email_verified_at, created_at)
		VALUES ($1, 'Token Retention User', 'token-retention@example.com', '#444444', now(), now())
	`, userID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, 'expired-token-hash', now() - interval '1 hour')
	`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, 'still-valid-token-hash', now() + interval '1 hour')
	`, userID)
	require.NoError(t, err)

	worker := jobs.NewRetentionWorker(pool, 90, 30, 365, 7)
	require.NoError(t, worker.Work(ctx, &river.Job[jobs.RetentionArgs]{}))

	var remainingHashes []string
	rows, err := pool.Query(ctx, `SELECT token_hash FROM email_verification_tokens WHERE user_id = $1`, userID)
	require.NoError(t, err)
	for rows.Next() {
		var hash string
		require.NoError(t, rows.Scan(&hash))
		remainingHashes = append(remainingHashes, hash)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []string{"still-valid-token-hash"}, remainingHashes,
		"only the still-valid verification token should survive retention")
}

// TestRetentionWorker_ContinuesLaterPhasesWhenAnEarlierPhaseFails is a
// regression test for a bug where Work returned immediately on the first
// phase's error, skipping sessions/invites/audit_log entirely -- exactly the
// "one table's backlog starves the phases after it" failure mode
// retentionPhaseTimeout's own independent per-phase budgets were built to
// prevent, just triggered by a phase's own error/timeout propagating up
// instead of a shared budget being exhausted. Forces the notifications
// phase to fail with a genuine SQL error (dropping its table), which
// exercises the identical deleteBatched error-return path a real
// retentionPhaseTimeout timeout would, then verifies the sessions and
// invites phases still ran and cleaned up despite that failure.
func TestRetentionWorker_ContinuesLaterPhasesWhenAnEarlierPhaseFails(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Retention Continue User', 'retention-continue@example.com', '#654321')`,
		userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES ($1, 'phase-continue-expired', now() - interval '130 days', now() - interval '100 days')
	`, userID)
	require.NoError(t, err)

	teamID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Retention Continue Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		INSERT INTO invites (team_id, code, expires_at)
		VALUES ($1, 'phase-continue-code', now() - interval '100 days')
	`, teamID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `DROP TABLE notifications CASCADE`)
	require.NoError(t, err)

	worker := jobs.NewRetentionWorker(pool, 90, 30, 365, 7)
	err = worker.Work(ctx, &river.Job[jobs.RetentionArgs]{})
	require.Error(t, err, "Work must report the notifications phase's failure")
	assert.Contains(t, err.Error(), "delete notifications")

	var sessionCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessions WHERE user_id = $1`, userID).Scan(&sessionCount))
	assert.Equal(t, 0, sessionCount, "sessions phase must still run and delete the expired session even though notifications failed first")

	var inviteCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM invites WHERE team_id = $1`, teamID).Scan(&inviteCount))
	assert.Equal(t, 0, inviteCount, "invites phase must still run and delete the expired invite even though notifications failed first")
}
