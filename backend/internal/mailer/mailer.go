// Package mailer abstracts outbound transactional email away from the auth
// package, mirroring how internal/storage abstracts image bytes away from an
// S3-compatible object store: a small interface, a real implementation, and
// an in-memory/logging fake for dev and tests.
package mailer

import "context"

// Mailer sends transactional email. The only email this application sends
// today is the self-registration verification link.
type Mailer interface {
	// SendVerificationEmail sends toEmail a message containing verifyURL, the
	// link the recipient must open to confirm their address.
	SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error
}
