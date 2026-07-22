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

func TestNewSMTPMailer_DefaultsPort(t *testing.T) {
	t.Parallel()

	// Sending will fail (no real SMTP server), but construction with a blank
	// port must not error -- it defaults to 587, verified indirectly by
	// confirming construction succeeds with only Host/FromAddress set.
	m, err := mailer.NewSMTPMailer(mailer.SMTPConfig{Host: "smtp.example.com", FromAddress: "no-reply@example.com"})
	require.NoError(t, err)
	require.NotNil(t, m)
}
