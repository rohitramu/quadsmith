package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	quadsmith_internal "quadsmith/api/gen/quadsmith_internal"
)

func TestExpandTemplates(t *testing.T) {
	t.Setenv("FOO", "hello")
	t.Setenv("BAR", "world")

	input := []byte("msg: \"${FOO} ${BAR} ${BAZ:-default_val}\"")
	output, err := ExpandTemplates(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "msg: \"hello world default_val\""
	if string(output) != expected {
		t.Errorf("expected %q, got %q", expected, string(output))
	}

	// Missing variable without default should return error
	missing := []byte("secret: \"${MISSING_SECRET_KEY}\"")
	_, err = ExpandTemplates(missing)
	if err == nil {
		t.Fatal("expected error for missing variable, got nil")
	}
	if !strings.Contains(err.Error(), "MISSING_SECRET_KEY") {
		t.Errorf("expected error to mention MISSING_SECRET_KEY, got %v", err)
	}
}

func TestValidateDatabase(t *testing.T) {
	tests := []struct {
		name    string
		db      *quadsmith_internal.DatabaseConfig
		env     string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid database config",
			db: &quadsmith_internal.DatabaseConfig{
				Url:            "postgres://user:pass@127.0.0.1:5432/quadsmith",
				Name:           "quadsmith",
				MaxConnections: 20,
				MinConnections: 5,
			},
			env:     "sandbox",
			wantErr: false,
		},
		{
			name: "missing database name",
			db: &quadsmith_internal.DatabaseConfig{
				Url:            "postgres://user:pass@127.0.0.1:5432/quadsmith",
				Name:           "",
				MaxConnections: 20,
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "database.name cannot be empty",
		},
		{
			name: "database name mismatch with url path",
			db: &quadsmith_internal.DatabaseConfig{
				Url:            "postgres://user:pass@127.0.0.1:5432/quadsmith",
				Name:           "quadsmith_internal",
				MaxConnections: 20,
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "does not match configured database name",
		},
		{
			name: "min connections exceeds max connections",
			db: &quadsmith_internal.DatabaseConfig{
				Url:            "postgres://user:pass@127.0.0.1:5432/quadsmith",
				Name:           "quadsmith",
				MaxConnections: 5,
				MinConnections: 10,
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "cannot exceed max_connections",
		},
		{
			name: "localhost rejected in production",
			db: &quadsmith_internal.DatabaseConfig{
				Url:            "postgres://user:pass@localhost:5432/quadsmith",
				Name:           "quadsmith",
				MaxConnections: 20,
			},
			env:     "prod",
			wantErr: true,
			errMsg:  "cannot point to localhost/127.0.0.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDatabase(tc.db, tc.env)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateDatabase() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr && !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("expected error containing %q, got %q", tc.errMsg, err.Error())
			}
		})
	}
}

func TestValidateServer(t *testing.T) {
	cfg := NewTestAPIServerConfig(t)
	if err := ValidateAPIServerConfig(cfg); err != nil {
		t.Fatalf("unexpected validation error on test config: %v", err)
	}

	cfg.Server.Port = 70000
	if err := ValidateAPIServerConfig(cfg); err == nil {
		t.Error("expected error for invalid port, got nil")
	}

	cfg.Server.Port = 8080
	cfg.Server.StaticDir = ""
	if err := ValidateAPIServerConfig(cfg); err == nil {
		t.Error("expected error for empty static dir, got nil")
	}
}

func TestLoadTextproto(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "server.textproto")

	t.Setenv("PORT_VAR", "9090")
	t.Setenv("SECRET_VAR", "super-secret-key-12345678901234")

	content := `
environment: "sandbox"
server {
  port: ${PORT_VAR:-8080}
  static_dir: "dist"
  read_timeout { seconds: 30 }
  write_timeout { seconds: 30 }
  max_request_bytes: 1048576
}
database {
  url: "postgres://postgres:postgres@127.0.0.1:5432/quadsmith"
  name: "quadsmith"
  max_connections: 10
}
auth {
  secret: "${SECRET_VAR}"
  access_token_ttl { seconds: 3600 }
  refresh_token_ttl { seconds: 86400 }
  otp_ttl { seconds: 300 }
  magic_link_ttl { seconds: 900 }
  email_cooldown { seconds: 1 }
}
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	var cfg quadsmith_internal.ApiServerConfig
	if err := Load(configPath, &cfg); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Auth.Secret != "super-secret-key-12345678901234" {
		t.Errorf("expected secret expanded, got %q", cfg.Auth.Secret)
	}
	if cfg.Database.Name != "quadsmith" {
		t.Errorf("expected database name quadsmith, got %q", cfg.Database.Name)
	}

	if err := ValidateAPIServerConfig(&cfg); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
}

func TestNewTestWorkerConfig(t *testing.T) {
	wcfg := NewTestWorkerConfig(t, WithWorkerPort(9999), WithWorkerConcurrency(8))
	if err := ValidateWorkerConfig(wcfg); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if wcfg.Worker.Port != 9999 {
		t.Errorf("expected port 9999, got %d", wcfg.Worker.Port)
	}
	if wcfg.Worker.Concurrency != 8 {
		t.Errorf("expected concurrency 8, got %d", wcfg.Worker.Concurrency)
	}
}

func TestLoadBinaryConfigFiles(t *testing.T) {
	// Sandbox
	apiSandbox, err := LoadAPIServerConfig("", "sandbox")
	if err != nil {
		t.Fatalf("failed to load api-server sandbox: %v", err)
	}
	if apiSandbox.Server.Port != 8080 || apiSandbox.Database.Name != "quadsmith" {
		t.Errorf("unexpected api-server sandbox config: %+v", apiSandbox)
	}

	workerSandbox, err := LoadWorkerConfig("", "sandbox")
	if err != nil {
		t.Fatalf("failed to load worker sandbox: %v", err)
	}
	if workerSandbox.Worker.Port != 8081 || workerSandbox.Database.Name != "quadsmith" {
		t.Errorf("unexpected worker sandbox config: %+v", workerSandbox)
	}

	// Test
	apiTest, err := LoadAPIServerConfig("", "test")
	if err != nil {
		t.Fatalf("failed to load api-server test: %v", err)
	}
	if apiTest.Server.Port != 8080 {
		t.Errorf("unexpected api-server test config: %+v", apiTest)
	}

	workerTest, err := LoadWorkerConfig("", "test")
	if err != nil {
		t.Fatalf("failed to load worker test: %v", err)
	}
	if workerTest.Worker.Port != 8081 {
		t.Errorf("unexpected worker test config: %+v", workerTest)
	}

	// Staging & Prod (require QS_DATABASE_URL and QS_AUTH_SECRET)
	t.Setenv("QS_DATABASE_URL", "postgres://user:pass@db.example.com:5432/quadsmith")
	t.Setenv("QS_AUTH_SECRET", "super-secret-prod-auth-key-32bytes")

	apiStaging, err := LoadAPIServerConfig("", "staging")
	if err != nil {
		t.Fatalf("failed to load api-server staging: %v", err)
	}
	if apiStaging.Server.Port != 443 {
		t.Errorf("unexpected api-server staging port: %d", apiStaging.Server.Port)
	}

	apiProd, err := LoadAPIServerConfig("", "prod")
	if err != nil {
		t.Fatalf("failed to load api-server prod: %v", err)
	}
	if apiProd.Server.Port != 443 {
		t.Errorf("unexpected api-server prod port: %d", apiProd.Server.Port)
	}

	workerStaging, err := LoadWorkerConfig("", "staging")
	if err != nil {
		t.Fatalf("failed to load worker staging: %v", err)
	}
	if workerStaging.Worker.Port != 443 {
		t.Errorf("unexpected worker staging port: %d", workerStaging.Worker.Port)
	}

	workerProd, err := LoadWorkerConfig("", "prod")
	if err != nil {
		t.Fatalf("failed to load worker prod: %v", err)
	}
	if workerProd.Worker.Port != 443 {
		t.Errorf("unexpected worker prod port: %d", workerProd.Worker.Port)
	}
}
