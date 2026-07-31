// Package mailer abstracts outbound transactional email away from the auth
// package, mirroring how internal/storage abstracts image bytes away from an
// S3-compatible object store: a small interface, a real implementation, and
// an in-memory/logging fake for dev and tests.
package mailer

import "context"

// Mailer sends transactional email. Today that's the self-registration
// verification link and the password-reset link.
type Mailer interface {
	// SendVerificationEmail sends toEmail a message containing verifyURL, the
	// link the recipient must open to confirm their address.
	SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error
	// SendPasswordResetEmail sends toEmail a message containing resetURL, the
	// link the recipient must open to set a new password.
	SendPasswordResetEmail(ctx context.Context, toEmail, resetURL string) error
}
