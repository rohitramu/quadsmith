package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExpandTemplates(t *testing.T) {
	// Set test environment variable
	t.Setenv("TEST_VAR", "injected_value")

	tests := []struct {
		name        string
		input       string
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:  "standard substitution",
			input: "secret: ${TEST_VAR}",
			want:  "secret: injected_value",
		},
		{
			name:  "substitution with fallback used when unset",
			input: "secret: ${UNSET_VAR:-fallback_val}",
			want:  "secret: fallback_val",
		},
		{
			name:  "substitution with fallback ignored when set",
			input: "secret: ${TEST_VAR:-fallback_val}",
			want:  "secret: injected_value",
		},
		{
			name:        "missing required variable error",
			input:       "secret: ${REQUIRED_MISSING_VAR}",
			wantErr:     true,
			errContains: "Missing required secret environment variable: REQUIRED_MISSING_VAR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandTemplates([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, string(got))
			}
		})
	}
}

func TestLoadSandboxConfig(t *testing.T) {
	rootPath, err := filepath.Abs("../sandbox.yaml")
	if err != nil {
		t.Fatalf("failed to get path: %v", err)
	}

	cfg, err := Load(rootPath)
	if err != nil {
		t.Fatalf("failed to load sandbox config: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected server.port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Database.URL == "" {
		t.Errorf("expected non-empty database.url")
	}
	if cfg.Auth.Secret == "" {
		t.Errorf("expected non-empty auth.secret")
	}
	if cfg.Auth.OtpTTL != 3*time.Minute {
		t.Errorf("expected otp_ttl 3m, got %v", cfg.Auth.OtpTTL)
	}
	if cfg.Auth.MagicLinkTTL != 15*time.Minute {
		t.Errorf("expected magic_link_ttl 15m, got %v", cfg.Auth.MagicLinkTTL)
	}

	if err := cfg.ValidateServer(); err != nil {
		t.Errorf("sandbox server validation failed: %v", err)
	}
	if err := cfg.ValidateWorker(); err != nil {
		t.Errorf("sandbox worker validation failed: %v", err)
	}
	if err := cfg.ValidateSeeder(); err != nil {
		t.Errorf("sandbox seeder validation failed: %v", err)
	}
}

func TestLoadProdConfigRequiresSecrets(t *testing.T) {
	rootPath, err := filepath.Abs("../prod.yaml")
	if err != nil {
		t.Fatalf("failed to get path: %v", err)
	}

	// Without env vars set, loading prod.yaml must fail
	_ = os.Unsetenv("QS_AUTH_SECRET")
	_ = os.Unsetenv("QS_DATABASE_URL")

	_, err = Load(rootPath)
	if err == nil {
		t.Fatalf("expected loading prod.yaml to fail without secrets, but it succeeded")
	}
	if !strings.Contains(err.Error(), "QS_AUTH_SECRET") || !strings.Contains(err.Error(), "QS_DATABASE_URL") {
		t.Fatalf("expected error mentioning missing secret env vars, got: %v", err)
	}
}

func TestProdValidationRejectsDevSecrets(t *testing.T) {
	cfg := NewTestBackendConfig(t)
	cfg.Environment = "prod"
	cfg.Database.URL = "postgres://user:pass@db.quadsmith.internal:5432/quadsmith"
	cfg.Auth.SecureCookies = true

	// Using dev secret in prod must fail validation
	cfg.Auth.Secret = "quadsmith-dev-secret-key-32bytes-long!"
	if err := cfg.ValidateServer(); err == nil {
		t.Errorf("expected validation to fail when using dev secret in prod")
	}

	// Using valid unique secret in prod must succeed
	cfg.Auth.Secret = "production-real-super-secret-key-64chars-long-minimum-entropy!"
	if err := cfg.ValidateServer(); err != nil {
		t.Errorf("expected validation to pass, got: %v", err)
	}
}

func TestProdValidationRejectsLocalhostDatabase(t *testing.T) {
	cfg := NewTestBackendConfig(t)
	cfg.Environment = "prod"
	cfg.Auth.SecureCookies = true
	cfg.Auth.Secret = "production-real-super-secret-key-64chars-long-minimum-entropy!"

	cfg.Database.URL = "postgres://postgres:postgres@localhost:5432/quadsmith"
	if err := cfg.ValidateServer(); err == nil {
		t.Errorf("expected validation to fail when database.url is localhost in prod")
	}

	cfg.Database.URL = "postgres://postgres:postgres@127.0.0.1:5432/quadsmith"
	if err := cfg.ValidateServer(); err == nil {
		t.Errorf("expected validation to fail when database.url is 127.0.0.1 in prod")
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("QS_SERVER_PORT", "9999")
	t.Setenv("QS_WORKER_CONCURRENCY", "42")
	t.Setenv("QS_AUTH_OTP_TTL", "10m")

	rootPath, err := filepath.Abs("../sandbox.yaml")
	if err != nil {
		t.Fatalf("failed to get path: %v", err)
	}

	cfg, err := Load(rootPath)
	if err != nil {
		t.Fatalf("failed to load sandbox config: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("expected server port 9999 from override, got %d", cfg.Server.Port)
	}
	if cfg.Worker.Concurrency != 42 {
		t.Errorf("expected worker concurrency 42 from override, got %d", cfg.Worker.Concurrency)
	}
	if cfg.Auth.OtpTTL != 10*time.Minute {
		t.Errorf("expected otp_ttl 10m from override, got %v", cfg.Auth.OtpTTL)
	}
}
