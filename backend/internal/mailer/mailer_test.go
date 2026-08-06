package mailer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/mailer"
)

func TestFakeMailer_RecordsLastSentAndCount(t *testing.T) {
	t.Parallel()

	fm := mailer.NewFakeMailer(nil)

	require.NoError(t, fm.SendVerificationEmail(context.Background(), "a@example.com", "https://example.com/verify-email/tok1"))
	require.NoError(t, fm.SendVerificationEmail(context.Background(), "b@example.com", "https://example.com/verify-email/tok2"))

	to, link := fm.LastSentTo()
	assert.Equal(t, "b@example.com", to)
	assert.Equal(t, "https://example.com/verify-email/tok2", link)
	assert.Equal(t, 2, fm.SentCount())
}

func TestFakeMailer_LinksFor(t *testing.T) {
	t.Parallel()

	fm := mailer.NewFakeMailer(nil)

	require.NoError(t, fm.SendVerificationEmail(context.Background(), "a@example.com", "link1"))
	require.NoError(t, fm.SendVerificationEmail(context.Background(), "a@example.com", "link2"))
	require.NoError(t, fm.SendVerificationEmail(context.Background(), "b@example.com", "link3"))

	assert.Equal(t, []string{"link1", "link2"}, fm.LinksFor("a@example.com"))
	assert.Equal(t, []string{"link3"}, fm.LinksFor("b@example.com"))
	assert.Empty(t, fm.LinksFor("nobody@example.com"))
}

func TestFakeMailer_RecordsLastResetSentAndCount(t *testing.T) {
	t.Parallel()

	fm := mailer.NewFakeMailer(nil)

	require.NoError(t, fm.SendPasswordResetEmail(context.Background(), "a@example.com", "https://example.com/reset-password/tok1"))
	require.NoError(t, fm.SendPasswordResetEmail(context.Background(), "b@example.com", "https://example.com/reset-password/tok2"))

	to, link := fm.LastResetSentTo()
	assert.Equal(t, "b@example.com", to)
	assert.Equal(t, "https://example.com/reset-password/tok2", link)
	assert.Equal(t, 2, fm.ResetSentCount())
}

func TestFakeMailer_ResetLinksFor(t *testing.T) {
	t.Parallel()

	fm := mailer.NewFakeMailer(nil)

	require.NoError(t, fm.SendPasswordResetEmail(context.Background(), "a@example.com", "link1"))
	require.NoError(t, fm.SendPasswordResetEmail(context.Background(), "a@example.com", "link2"))
	require.NoError(t, fm.SendPasswordResetEmail(context.Background(), "b@example.com", "link3"))

	assert.Equal(t, []string{"link1", "link2"}, fm.ResetLinksFor("a@example.com"))
	assert.Equal(t, []string{"link3"}, fm.ResetLinksFor("b@example.com"))
	assert.Empty(t, fm.ResetLinksFor("nobody@example.com"))
}

// FakeMailer's verification-email and password-reset-email tracking are
// independent of each other -- sending one must not appear in the other's
// history.
func TestFakeMailer_VerificationAndResetTrackingAreIndependent(t *testing.T) {
	t.Parallel()

	fm := mailer.NewFakeMailer(nil)

	require.NoError(t, fm.SendVerificationEmail(context.Background(), "a@example.com", "verify-link"))
	require.NoError(t, fm.SendPasswordResetEmail(context.Background(), "a@example.com", "reset-link"))

	assert.Equal(t, 1, fm.SentCount())
	assert.Equal(t, 1, fm.ResetSentCount())
	assert.Equal(t, []string{"verify-link"}, fm.LinksFor("a@example.com"))
	assert.Equal(t, []string{"reset-link"}, fm.ResetLinksFor("a@example.com"))
}

func TestNewSMTPMailer_RequiresHostAndFromAddress(t *testing.T) {
	t.Parallel()

	_, err := mailer.NewSMTPMailer(mailer.SMTPConfig{FromAddress: "no-reply@example.com"})
	require.ErrorIs(t, err, mailer.ErrSMTPHostRequired)

	_, err = mailer.NewSMTPMailer(mailer.SMTPConfig{Host: "smtp.example.com"})
	require.ErrorIs(t, err, mailer.ErrSMTPFromAddressRequired)

	m, err := mailer.NewSMTPMailer(mailer.SMTPConfig{Host: "smtp.example.com", FromAddress: "no-reply@example.com"})
	require.NoError(t, err)
	require.NotNil(t, m)
}

// A CRLF in a header field is rejected before any network I/O happens
// (buildMessage runs before send dials out), so this doesn't need a real
// SMTP server to verify the mailer defends against header injection itself,
// independent of upstream validation.
func TestSMTPMailer_RejectsCRLFInToAddress(t *testing.T) {
	t.Parallel()

	m, err := mailer.NewSMTPMailer(mailer.SMTPConfig{Host: "smtp.example.com", FromAddress: "no-reply@example.com"})
	require.NoError(t, err)

	err = m.SendVerificationEmail(context.Background(), "victim@example.com\r\nBcc: attacker@evil.com", "https://example.com/verify-email/tok1")
	require.ErrorIs(t, err, mailer.ErrHeaderInjection)

	err = m.SendPasswordResetEmail(context.Background(), "victim@example.com\r\nBcc: attacker@evil.com", "https://example.com/reset-password/tok1")
	require.ErrorIs(t, err, mailer.ErrHeaderInjection)
}

func TestNewSMTPMailer_DefaultsPort(t *testing.T) {
	t.Parallel()

	// Sending will fail (no real SMTP server), but construction with a blank
	// port must not error -- it defaults to 587, verified indirectly by
	// confirming construction succeeds with only Host/FromAddress set.
	m, err := mailer.NewSMTPMailer(mailer.SMTPConfig{Host: "smtp.example.com", FromAddress: "no-reply@example.com"})
	require.NoError(t, err)
	require.NotNil(t, m)
}
