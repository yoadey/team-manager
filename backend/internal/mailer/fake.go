package mailer

import (
	"context"
	"log/slog"
	"sync"
)

// FakeMailer is an in-memory Mailer for dev and tests: it logs the
// verification link instead of sending real mail, and records the last sent
// message so tests can assert on it without a real mail server.
type FakeMailer struct {
	logger *slog.Logger

	mu               sync.Mutex
	lastTo           string
	lastVerify       string
	sentCount        int
	sentToLinks      map[string][]string
	lastResetTo      string
	lastReset        string
	resetSentCount   int
	sentToResetLinks map[string][]string
}

// NewFakeMailer creates a FakeMailer that logs via logger.
func NewFakeMailer(logger *slog.Logger) *FakeMailer {
	if logger == nil {
		logger = slog.Default()
	}
	return &FakeMailer{logger: logger, sentToLinks: map[string][]string{}, sentToResetLinks: map[string][]string{}}
}

// SendVerificationEmail logs verifyURL instead of sending it, and records it
// for test assertions.
func (f *FakeMailer) SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error {
	f.logger.InfoContext(ctx, "mailer: verification email (SMTP not configured, logging instead)",
		"to", toEmail, "link", verifyURL)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastTo = toEmail
	f.lastVerify = verifyURL
	f.sentCount++
	f.sentToLinks[toEmail] = append(f.sentToLinks[toEmail], verifyURL)
	return nil
}

// SendPasswordResetEmail logs resetURL instead of sending it, and records it
// for test assertions.
func (f *FakeMailer) SendPasswordResetEmail(ctx context.Context, toEmail, resetURL string) error {
	f.logger.InfoContext(ctx, "mailer: password reset email (SMTP not configured, logging instead)",
		"to", toEmail, "link", resetURL)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastResetTo = toEmail
	f.lastReset = resetURL
	f.resetSentCount++
	f.sentToResetLinks[toEmail] = append(f.sentToResetLinks[toEmail], resetURL)
	return nil
}

// LastResetSentTo returns the recipient and reset link of the most recently
// sent password-reset message. Test helper.
func (f *FakeMailer) LastResetSentTo() (toEmail, resetURL string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastResetTo, f.lastReset
}

// ResetSentCount returns the total number of password-reset messages sent.
// Test helper.
func (f *FakeMailer) ResetSentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resetSentCount
}

// ResetLinksFor returns every password-reset link ever sent to toEmail, in
// send order. Test helper.
func (f *FakeMailer) ResetLinksFor(toEmail string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	links := f.sentToResetLinks[toEmail]
	out := make([]string, len(links))
	copy(out, links)
	return out
}

// LastSentTo returns the recipient and verification link of the most
// recently sent message. Test helper.
func (f *FakeMailer) LastSentTo() (toEmail, verifyURL string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastTo, f.lastVerify
}

// SentCount returns the total number of messages sent. Test helper.
func (f *FakeMailer) SentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sentCount
}

// LinksFor returns every verification link ever sent to toEmail, in send
// order. Test helper.
func (f *FakeMailer) LinksFor(toEmail string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	links := f.sentToLinks[toEmail]
	out := make([]string, len(links))
	copy(out, links)
	return out
}
