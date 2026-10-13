package email

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	config "quadsmith/shared/gen/quadsmith/config"
)

// EmailSender defines the standard interface for outbound transactional emails.
type EmailSender interface {
	SendVerification(ctx context.Context, toEmail, code, magicLinkURL, purpose string, ttl time.Duration) error
	SendSecurityAlert(ctx context.Context, toEmail, eventDescription string, eventTime time.Time) error
}

// NewEmailSender constructs an EmailSender based on declarative configuration.
func NewEmailSender(cfg *config.EmailConfig) EmailSender {
	if cfg != nil && cfg.GetSmtp() != nil {
		return NewSMTPEmailSender(cfg.GetSmtp())
	}
	return NewConsoleEmailSender()
}

// -----------------------------------------------------------------------------
// ConsoleEmailSender
// -----------------------------------------------------------------------------

// ConsoleEmailSender formats and prints outbound emails directly to stdout.
// Useful for hermetic test execution, local development, and sandboxes.
type ConsoleEmailSender struct{}

func NewConsoleEmailSender() *ConsoleEmailSender {
	return &ConsoleEmailSender{}
}

func (c *ConsoleEmailSender) SendVerification(ctx context.Context, toEmail, code, magicLinkURL, purpose string, ttl time.Duration) error {
	fmt.Printf("\n=======================================================\n")
	fmt.Printf("[EMAIL] Outbound Verification to: %s\n", toEmail)
	fmt.Printf("[EMAIL] Purpose: %s\n", purpose)
	fmt.Printf("[EMAIL] 6-Digit Code: %s (TTL: %v)\n", code, ttl)
	if magicLinkURL != "" {
		fmt.Printf("[EMAIL] 1-Click Magic Link: %s\n", magicLinkURL)
	}
	fmt.Printf("=======================================================\n\n")
	return nil
}

func (c *ConsoleEmailSender) SendSecurityAlert(ctx context.Context, toEmail, eventDescription string, eventTime time.Time) error {
	fmt.Printf("\n=======================================================\n")
	fmt.Printf("[EMAIL] Security Alert to: %s\n", toEmail)
	fmt.Printf("[EMAIL] Event: %s (Time: %s)\n", eventDescription, eventTime.Format(time.RFC3339))
	fmt.Printf("=======================================================\n\n")
	return nil
}

// -----------------------------------------------------------------------------
// SMTPEmailSender
// -----------------------------------------------------------------------------

// SMTPEmailSender delivers emails over standard SMTP with TLS/STARTTLS support.
type SMTPEmailSender struct {
	host        string
	port        int
	username    string
	password    string
	fromAddress string
	replyTo     string
	requireTLS  bool
}

func NewSMTPEmailSender(cfg *config.SmtpConfig) *SMTPEmailSender {
	if cfg == nil {
		return &SMTPEmailSender{
			port: 587,
		}
	}
	from := cfg.GetFromAddress()
	if from == "" {
		from = "Quadsmith <auth@quadsmith.net>"
	}
	return &SMTPEmailSender{
		host:        cfg.GetHost(),
		port:        int(cfg.GetPort()),
		username:    cfg.GetUsername(),
		password:    cfg.GetPassword(),
		fromAddress: from,
		replyTo:     cfg.GetReplyTo(),
		requireTLS:  cfg.GetRequireTls(),
	}
}

func (s *SMTPEmailSender) SendVerification(ctx context.Context, toEmail, code, magicLinkURL, purpose string, ttl time.Duration) error {
	subject := "Verify your Quadsmith email"
	actionTitle := "Verification Code"
	if strings.Contains(strings.ToLower(purpose), "delete") {
		subject = "Confirm Quadsmith Account Deletion"
		actionTitle = "Deletion Confirmation Code"
	} else if strings.Contains(strings.ToLower(purpose), "add") {
		subject = "Confirm New Email Address"
		actionTitle = "Email Addition Code"
	}

	ttlMin := int(ttl.Minutes())
	if ttlMin < 1 {
		ttlMin = 3
	}

	plainText := fmt.Sprintf(
		"Your %s is: %s\n\nThis code will expire in %d minutes.\n",
		actionTitle, code, ttlMin,
	)
	if magicLinkURL != "" {
		plainText += fmt.Sprintf("\nAlternatively, verify instantly by clicking this link:\n%s\n", magicLinkURL)
	}
	plainText += "\nIf you did not request this email, please disregard it.\n"

	htmlBody := buildVerificationHTML(actionTitle, code, magicLinkURL, ttlMin)

	return s.sendMIME(ctx, toEmail, subject, plainText, htmlBody)
}

func (s *SMTPEmailSender) SendSecurityAlert(ctx context.Context, toEmail, eventDescription string, eventTime time.Time) error {
	subject := "Security Alert for your Quadsmith account"
	plainText := fmt.Sprintf(
		"Security Alert:\n\n%s\n\nEvent occurred at: %s.\nIf this wasn't you, please secure your account immediately.\n",
		eventDescription, eventTime.Format(time.RFC1123),
	)
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #f8fafc; padding: 40px 20px;">
  <div style="max-width: 520px; margin: 0 auto; background: #1e293b; border-radius: 12px; padding: 32px; border: 1px solid #334155;">
    <h2 style="color: #f43f5e; margin-top: 0;">Security Alert</h2>
    <p style="color: #cbd5e1; font-size: 15px; line-height: 1.6;">%s</p>
    <p style="color: #64748b; font-size: 13px;">Event occurred at: <strong>%s</strong></p>
    <hr style="border: none; border-top: 1px solid #334155; margin: 24px 0;" />
    <p style="color: #94a3b8; font-size: 12px;">If you did not perform this action, please sign in and verify your linked accounts immediately.</p>
  </div>
</body>
</html>`, eventDescription, eventTime.Format(time.RFC1123))

	return s.sendMIME(ctx, toEmail, subject, plainText, htmlBody)
}

func (s *SMTPEmailSender) sendMIME(ctx context.Context, toEmail, subject, plainText, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	if s.host == "" {
		return fmt.Errorf("smtp host is not configured")
	}

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to dial smtp server at %s: %w", addr, err)
	}

	// Direct TLS on port 465
	if s.port == 465 {
		tlsConfig := &tls.Config{
			ServerName: s.host,
			MinVersion: tls.VersionTLS12,
		}
		conn = tls.Client(conn, tlsConfig)
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to initialize smtp client: %w", err)
	}
	defer client.Close()

	// STARTTLS on port 587 (or when requireTLS is enabled)
	if s.port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok || s.requireTLS {
			tlsConfig := &tls.Config{
				ServerName: s.host,
				MinVersion: tls.VersionTLS12,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("failed to negotiate STARTTLS: %w", err)
			}
		}
	}

	// Authenticate if credentials are provided
	if s.username != "" && s.password != "" {
		var auth smtp.Auth
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			// Try LOGIN auth fallback if PLAIN fails
			auth = loginAuth(s.username, s.password)
			if err := client.Auth(auth); err != nil {
				return fmt.Errorf("smtp authentication failed: %w", err)
			}
		}
	}

	// Envelope sender
	fromMail := s.extractEmailAddress(s.fromAddress)
	if err := client.Mail(fromMail); err != nil {
		return fmt.Errorf("smtp MAIL FROM failed: %w", err)
	}

	// Envelope recipient
	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("smtp RCPT TO failed: %w", err)
	}

	// Message data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA command failed: %w", err)
	}
	defer w.Close()

	var msgBuf strings.Builder
	msgBuf.WriteString(fmt.Sprintf("From: %s\r\n", s.fromAddress))
	msgBuf.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	if s.replyTo != "" {
		msgBuf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", s.replyTo))
	}
	msgBuf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msgBuf.WriteString("MIME-Version: 1.0\r\n")

	mpWriter := multipart.NewWriter(&msgBuf)
	msgBuf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", mpWriter.Boundary()))

	// Plaintext part
	textHeader := make(textproto.MIMEHeader)
	textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	textHeader.Set("Content-Transfer-Encoding", "base64")
	textPart, err := mpWriter.CreatePart(textHeader)
	if err != nil {
		return fmt.Errorf("failed to create plaintext mime part: %w", err)
	}
	textPart.Write([]byte(base64.StdEncoding.EncodeToString([]byte(plainText))))

	// HTML part
	htmlHeader := make(textproto.MIMEHeader)
	htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlHeader.Set("Content-Transfer-Encoding", "base64")
	htmlPart, err := mpWriter.CreatePart(htmlHeader)
	if err != nil {
		return fmt.Errorf("failed to create html mime part: %w", err)
	}
	htmlPart.Write([]byte(base64.StdEncoding.EncodeToString([]byte(htmlBody))))

	if err := mpWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize multipart mime writer: %w", err)
	}

	if _, err := w.Write([]byte(msgBuf.String())); err != nil {
		return fmt.Errorf("failed to write email message bytes: %w", err)
	}

	return nil
}

func (s *SMTPEmailSender) extractEmailAddress(from string) string {
	if start := strings.Index(from, "<"); start != -1 {
		if end := strings.Index(from[start:], ">"); end != -1 {
			return strings.TrimSpace(from[start+1 : start+end])
		}
	}
	return strings.TrimSpace(from)
}

func buildVerificationHTML(actionTitle, code, magicLinkURL string, ttlMin int) string {
	var buttonHTML string
	if magicLinkURL != "" {
		buttonHTML = fmt.Sprintf(`
      <div style="margin: 32px 0; text-align: center;">
        <a href="%s" style="background: #3b82f6; color: #ffffff; padding: 14px 28px; border-radius: 8px; text-decoration: none; font-weight: 600; font-size: 15px; display: inline-block;">
          Verify &amp; Continue
        </a>
      </div>
      <p style="color: #94a3b8; font-size: 13px; text-align: center; margin-bottom: 24px;">
        Or enter this 6-digit verification code:
      </p>`, magicLinkURL)
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Quadsmith Verification</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #0f172a; color: #f8fafc; margin: 0; padding: 40px 20px;">
  <div style="max-width: 500px; margin: 0 auto; background-color: #1e293b; border-radius: 12px; padding: 36px; border: 1px solid #334155; box-shadow: 0 4px 12px rgba(0,0,0,0.3);">
    <div style="text-align: center; margin-bottom: 28px;">
      <h1 style="color: #38bdf8; font-size: 24px; margin: 0; font-weight: 700;">Quadsmith</h1>
      <p style="color: #94a3b8; font-size: 14px; margin-top: 4px;">FPV Engineering &amp; Drone Systems</p>
    </div>

    <h2 style="font-size: 18px; color: #e2e8f0; margin: 0 0 16px 0; font-weight: 600;">%s</h2>
    <p style="color: #cbd5e1; font-size: 14px; line-height: 1.5; margin: 0 0 24px 0;">
      Please use the following verification code to proceed with your request. This code is single-use and will expire in <strong>%d minutes</strong>.
    </p>

    %s

    <div style="background-color: #0f172a; border-radius: 8px; border: 1px solid #334155; padding: 20px; text-align: center; margin-bottom: 28px;">
      <div style="font-family: 'SF Mono', Consolas, 'Liberation Mono', Menlo, monospace; font-size: 32px; font-weight: 700; letter-spacing: 8px; color: #38bdf8;">
        %s
      </div>
    </div>

    <div style="border-top: 1px solid #334155; padding-top: 20px; margin-top: 28px; text-align: center;">
      <p style="color: #64748b; font-size: 12px; line-height: 1.5; margin: 0;">
        If you did not request this email, you can safely ignore it. Never share your verification codes or links with anyone.
      </p>
    </div>
  </div>
</body>
</html>`, actionTitle, ttlMin, buttonHTML, code)
}

// loginAuth implements the LOGIN authentication mechanism used by some SMTP servers.
type loginAuthStruct struct {
	username, password string
}

func loginAuth(username, password string) smtp.Auth {
	return &loginAuthStruct{username, password}
}

func (a *loginAuthStruct) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuthStruct) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:", "username:":
			return []byte(a.username), nil
		case "Password:", "password:":
			return []byte(a.password), nil
		default:
			return nil, fmt.Errorf("unexpected smtp login server challenge: %s", string(fromServer))
		}
	}
	return nil, nil
}
