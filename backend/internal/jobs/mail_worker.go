package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/codes"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/mailer"
)

// mailSendTimeout bounds a single mail-delivery job's SMTP round trip (dial +
// STARTTLS + auth + transmit). mailer.SMTPMailer's own dial/conn deadline is
// 10s (see internal/mailer/smtp.go's smtpDialTimeout); this gives real margin
// above that rather than racing it, mirroring how PushDeliveryWorker's own
// timeout exceeds push.Pusher's internal budget.
const mailSendTimeout = 20 * time.Second

// SendVerificationEmailWorker delivers a single self-registration
// verification email via the configured mailer.Mailer. auth.Repository.
// CreateEmailVerificationToken enqueues the auth.SendVerificationEmailArgs
// jobs this worker consumes (in the same DB transaction as the token-row
// insert -- see that method's doc comment). Any error is returned as-is so
// River's built-in retry/backoff applies -- a transient SMTP failure (relay
// hiccup, DNS blip) gets retried instead of silently losing the email,
// unlike the old synchronous best-effort "log and swallow" behavior this
// replaces.
type SendVerificationEmailWorker struct {
	river.WorkerDefaults[auth.SendVerificationEmailArgs]
	mailer mailer.Mailer
}

// NewSendVerificationEmailWorker constructs a SendVerificationEmailWorker.
func NewSendVerificationEmailWorker(m mailer.Mailer) *SendVerificationEmailWorker {
	return &SendVerificationEmailWorker{mailer: m}
}

// Work sends the verification email described by job.Args.
func (w *SendVerificationEmailWorker) Work(ctx context.Context, job *river.Job[auth.SendVerificationEmailArgs]) (err error) {
	ctx, span := tracer.Start(ctx, "send_verification_email.work")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	ctx, cancel := context.WithTimeout(ctx, mailSendTimeout)
	defer cancel()

	if err := w.mailer.SendVerificationEmail(ctx, job.Args.Email, job.Args.VerifyURL); err != nil {
		return fmt.Errorf("jobs.SendVerificationEmailWorker: %w", err)
	}
	return nil
}

// SendPasswordResetEmailWorker delivers a single password-reset email via
// the configured mailer.Mailer. auth.Repository.CreatePasswordResetToken
// enqueues the auth.SendPasswordResetEmailArgs jobs this worker consumes.
// See SendVerificationEmailWorker's doc comment for the retry/backoff
// rationale.
type SendPasswordResetEmailWorker struct {
	river.WorkerDefaults[auth.SendPasswordResetEmailArgs]
	mailer mailer.Mailer
}

// NewSendPasswordResetEmailWorker constructs a SendPasswordResetEmailWorker.
func NewSendPasswordResetEmailWorker(m mailer.Mailer) *SendPasswordResetEmailWorker {
	return &SendPasswordResetEmailWorker{mailer: m}
}

// Work sends the password-reset email described by job.Args.
func (w *SendPasswordResetEmailWorker) Work(ctx context.Context, job *river.Job[auth.SendPasswordResetEmailArgs]) (err error) {
	ctx, span := tracer.Start(ctx, "send_password_reset_email.work")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	ctx, cancel := context.WithTimeout(ctx, mailSendTimeout)
	defer cancel()

	if err := w.mailer.SendPasswordResetEmail(ctx, job.Args.Email, job.Args.ResetURL); err != nil {
		return fmt.Errorf("jobs.SendPasswordResetEmailWorker: %w", err)
	}
	return nil
}
