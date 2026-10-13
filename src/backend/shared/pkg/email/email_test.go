package email

import (
	"context"
	"strings"
	"testing"
	"time"

	config "quadsmith/shared/gen/quadsmith/config"
)

func TestConsoleEmailSender(t *testing.T) {
	sender := NewConsoleEmailSender()
	ctx := context.Background()

	err := sender.SendVerification(ctx, "pilot@example.com", "123456", "https://quadsmith.net/verify?token=abc", "login", 3*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = sender.SendSecurityAlert(ctx, "pilot@example.com", "New login from unknown IP", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewEmailSender(t *testing.T) {
	// Nil config -> ConsoleEmailSender
	s1 := NewEmailSender(nil)
	if _, ok := s1.(*ConsoleEmailSender); !ok {
		t.Errorf("expected *ConsoleEmailSender, got %T", s1)
	}

	// Console provider -> ConsoleEmailSender
	s2 := NewEmailSender(&config.EmailConfig{Provider: "console"})
	if _, ok := s2.(*ConsoleEmailSender); !ok {
		t.Errorf("expected *ConsoleEmailSender, got %T", s2)
	}

	// SMTP provider -> SMTPEmailSender
	s3 := NewEmailSender(&config.EmailConfig{
		Provider: "smtp",
		Smtp: &config.SmtpConfig{
			Host: "smtp.example.com",
			Port: 587,
		},
	})
	if smtpSender, ok := s3.(*SMTPEmailSender); !ok {
		t.Errorf("expected *SMTPEmailSender, got %T", s3)
	} else {
		if smtpSender.host != "smtp.example.com" || smtpSender.port != 587 {
			t.Errorf("unexpected smtp sender config: %+v", smtpSender)
		}
	}
}

func TestBuildVerificationHTML(t *testing.T) {
	html := buildVerificationHTML("Login Verification Code", "654321", "https://quadsmith.net/verify?token=xyz", 3)
	if !strings.Contains(html, "654321") {
		t.Errorf("expected html to contain code 654321")
	}
	if !strings.Contains(html, "https://quadsmith.net/verify?token=xyz") {
		t.Errorf("expected html to contain magic link")
	}
	if !strings.Contains(html, "3 minutes") {
		t.Errorf("expected html to contain 3 minutes")
	}
}

func TestSMTPEmailSender_ExtractEmail(t *testing.T) {
	sender := &SMTPEmailSender{}
	tests := []struct {
		input    string
		expected string
	}{
		{"Quadsmith Auth <auth@quadsmith.net>", "auth@quadsmith.net"},
		{"auth@quadsmith.net", "auth@quadsmith.net"},
		{"  support@quadsmith.net  ", "support@quadsmith.net"},
	}

	for _, tc := range tests {
		got := sender.extractEmailAddress(tc.input)
		if got != tc.expected {
			t.Errorf("for input %q expected %q, got %q", tc.input, tc.expected, got)
		}
	}
}
