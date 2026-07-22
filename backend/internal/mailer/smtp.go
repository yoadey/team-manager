package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// ErrSMTPHostRequired is returned by NewSMTPMailer when Host is empty.
var ErrSMTPHostRequired = errors.New("mailer.NewSMTPMailer: Host is required")

// ErrSMTPFromAddressRequired is returned by NewSMTPMailer when FromAddress is empty.
var ErrSMTPFromAddressRequired = errors.New("mailer.NewSMTPMailer: FromAddress is required")

// ErrSTARTTLSUnsupported is returned by SMTPMailer.send when the relay
// doesn't advertise STARTTLS -- send never falls back to plaintext.
var ErrSTARTTLSUnsupported = errors.New("mailer.SMTPMailer.send: server does not support STARTTLS")

// SMTPConfig holds the settings needed to send mail via an SMTP relay with
// STARTTLS.
type SMTPConfig struct {
	Host        string
	Port        string
	Username    string
	Password    string
	FromAddress string
}

// SMTPMailer sends mail via an SMTP relay using explicit STARTTLS. Unlike
// smtp.SendMail (which only upgrades to TLS when the server advertises
// STARTTLS and otherwise silently sends in plaintext), this implementation
// requires the upgrade to succeed, so credentials and message content are
// never sent over an unencrypted connection.
type SMTPMailer struct {
	cfg SMTPConfig
}

// NewSMTPMailer validates cfg and returns an SMTPMailer.
func NewSMTPMailer(cfg SMTPConfig) (*SMTPMailer, error) {
	if cfg.Host == "" {
		return nil, ErrSMTPHostRequired
	}
	if cfg.FromAddress == "" {
		return nil, ErrSMTPFromAddressRequired
	}
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	return &SMTPMailer{cfg: cfg}, nil
}

// smtpDialTimeout bounds the connection + handshake phase so a hung/black-holed
// relay can't stall the request that triggered the send indefinitely.
const smtpDialTimeout = 10 * time.Second

// SendVerificationEmail sends a plain-text message containing verifyURL to
// toEmail via the configured SMTP relay.
func (m *SMTPMailer) SendVerificationEmail(ctx context.Context, toEmail, verifyURL string) error {
	subject := "Confirm your email address"
	body := fmt.Sprintf(
		"Please confirm your email address by opening the link below:\r\n\r\n%s\r\n\r\nIf you did not request this, you can ignore this message.\r\n",
		verifyURL,
	)
	msg := buildMessage(m.cfg.FromAddress, toEmail, subject, body)
	return m.send(ctx, toEmail, msg)
}

// buildMessage assembles a minimal RFC 5322 message with the headers required
// for it to be accepted and rendered sanely by mail clients.
func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}

// send dials the SMTP relay, upgrades to TLS via STARTTLS, authenticates (if
// credentials are configured), and delivers msg to toEmail.
func (m *SMTPMailer) send(ctx context.Context, toEmail string, msg []byte) (err error) {
	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)

	dialer := net.Dialer{Timeout: smtpDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mailer.SMTPMailer.send: dial: %w", err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(smtpDialTimeout))

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("mailer.SMTPMailer.send: new client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return ErrSTARTTLSUnsupported
	}
	if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("mailer.SMTPMailer.send: starttls: %w", err)
	}

	if m.cfg.Username != "" {
		auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mailer.SMTPMailer.send: auth: %w", err)
		}
	}

	if err := client.Mail(m.cfg.FromAddress); err != nil {
		return fmt.Errorf("mailer.SMTPMailer.send: mail from: %w", err)
	}
	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("mailer.SMTPMailer.send: rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer.SMTPMailer.send: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("mailer.SMTPMailer.send: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer.SMTPMailer.send: close data: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("mailer.SMTPMailer.send: quit: %w", err)
	}
	return nil
}
