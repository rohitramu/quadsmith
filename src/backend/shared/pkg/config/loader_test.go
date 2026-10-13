package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "quadsmith/shared/gen/quadsmith/config"
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
		db      *config.DatabaseConfig
		env     string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid database config",
			db: &config.DatabaseConfig{
				Url:            "postgres://user:pass@127.0.0.1:5432/quadsmith_api",
				Name:           "quadsmith_api",
				MaxConnections: 20,
				MinConnections: 5,
			},
			env:     "sandbox",
			wantErr: false,
		},
		{
			name: "missing database name",
			db: &config.DatabaseConfig{
				Url:            "postgres://user:pass@127.0.0.1:5432/quadsmith_api",
				Name:           "",
				MaxConnections: 20,
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "database.name cannot be empty",
		},
		{
			name: "database name mismatch with url path",
			db: &config.DatabaseConfig{
				Url:            "postgres://user:pass@127.0.0.1:5432/quadsmith_api",
				Name:           "quadsmith_ops",
				MaxConnections: 20,
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "does not match configured database name",
		},
		{
			name: "min connections exceeds max connections",
			db: &config.DatabaseConfig{
				Url:            "postgres://user:pass@127.0.0.1:5432/quadsmith_api",
				Name:           "quadsmith_api",
				MaxConnections: 5,
				MinConnections: 10,
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "cannot exceed max_connections",
		},
		{
			name: "localhost rejected in production",
			db: &config.DatabaseConfig{
				Url:            "postgres://user:pass@localhost:5432/quadsmith_api",
				Name:           "quadsmith_api",
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

	webCfg := &config.WebServerConfig{
		Environment: "sandbox",
		Server: &config.ServerConfig{
			Port: 3000,
		},
		StaticDir: "",
		ApiUrl:    "http://localhost:8080",
	}
	if err := ValidateWebServerConfig(webCfg); err == nil {
		t.Error("expected error for empty static dir on web server, got nil")
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
  read_timeout { seconds: 30 }
  write_timeout { seconds: 30 }
  max_request_bytes: 1048576
}
database {
  url: "postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api"
  name: "quadsmith_api"
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

	var cfg config.ApiServerConfig
	if err := Load(configPath, &cfg); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Auth.Secret != "super-secret-key-12345678901234" {
		t.Errorf("expected secret expanded, got %q", cfg.Auth.Secret)
	}
	if cfg.Database.Name != "quadsmith_api" {
		t.Errorf("expected database name quadsmith_api, got %q", cfg.Database.Name)
	}

	if err := ValidateAPIServerConfig(&cfg); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
}

func TestLoadServerConfigs_ExplicitPathRequired(t *testing.T) {
	t.Setenv("QS_CONFIG", "")
	if _, err := LoadAPIServerConfig(""); err == nil {
		t.Fatal("expected error when configPath is empty for LoadAPIServerConfig")
	}
	if _, err := LoadOpsServerConfig(""); err == nil {
		t.Fatal("expected error when configPath is empty for LoadOpsServerConfig")
	}
	if _, err := LoadJobServerConfig(""); err == nil {
		t.Fatal("expected error when configPath is empty for LoadJobServerConfig")
	}
	if _, err := LoadWebServerConfig(""); err == nil {
		t.Fatal("expected error when configPath is empty for LoadWebServerConfig")
	}
}

func TestLoadBinaryConfigFiles(t *testing.T) {
	// Sandbox
	apiSandboxPath := filepath.Join("..", "..", "..", "server-api", "config", "sandbox.textproto")
	apiSandbox, err := LoadAPIServerConfig(apiSandboxPath)
	if err != nil {
		t.Fatalf("failed to load api-server sandbox: %v", err)
	}
	if apiSandbox.Server.Port != 8080 || apiSandbox.Database.Name != "quadsmith_api" {
		t.Errorf("unexpected api-server sandbox config: %+v", apiSandbox)
	}

	opsSandboxPath := filepath.Join("..", "..", "..", "server-ops", "config", "sandbox.textproto")
	opsSandbox, err := LoadOpsServerConfig(opsSandboxPath)
	if err != nil {
		t.Fatalf("failed to load ops-server sandbox: %v", err)
	}
	if opsSandbox.Server.Port != 8081 || opsSandbox.Database.Name != "quadsmith_api" || opsSandbox.InternalDatabase.Name != "quadsmith_ops" {
		t.Errorf("unexpected ops-server sandbox config: %+v", opsSandbox)
	}

	jobSandboxPath := filepath.Join("..", "..", "..", "server-jobs", "config", "sandbox.textproto")
	jobSandbox, err := LoadJobServerConfig(jobSandboxPath)
	if err != nil {
		t.Fatalf("failed to load job-server sandbox: %v", err)
	}
	if jobSandbox.OpsServerUrl != "http://127.0.0.1:8081" || len(jobSandbox.Schedules) != 3 {
		t.Errorf("unexpected job-server sandbox config: %+v", jobSandbox)
	}

	webSandboxPath := filepath.Join("..", "..", "..", "..", "frontend", "web", "config", "sandbox.textproto")
	webSandbox, err := LoadWebServerConfig(webSandboxPath)
	if err != nil {
		t.Fatalf("failed to load web-server sandbox: %v", err)
	}
	if webSandbox.Server.Port != 3000 || webSandbox.StaticDir != "src/frontend/web/app/dist" {
		t.Errorf("unexpected web-server sandbox config: %+v", webSandbox)
	}

	opsCliSandbox, err := LoadOpsCliConfig("", "sandbox")
	if err != nil {
		t.Fatalf("failed to load ops cli sandbox: %v", err)
	}
	if opsCliSandbox.OpsServerUrl != "http://127.0.0.1:8081" {
		t.Errorf("unexpected ops cli sandbox config: %+v", opsCliSandbox)
	}

	cliSandbox, err := LoadCliConfig("", "sandbox")
	if err != nil {
		t.Fatalf("failed to load cli sandbox: %v", err)
	}
	if cliSandbox.ApiUrl != "http://127.0.0.1:8080" {
		t.Errorf("unexpected cli sandbox config: %+v", cliSandbox)
	}

	// Test
	apiTestPath := filepath.Join("..", "..", "..", "server-api", "config", "test.textproto")
	apiTest, err := LoadAPIServerConfig(apiTestPath)
	if err != nil {
		t.Fatalf("failed to load api-server test: %v", err)
	}
	if apiTest.Server.Port != 8080 {
		t.Errorf("unexpected api-server test config: %+v", apiTest)
	}

	opsTestPath := filepath.Join("..", "..", "..", "server-ops", "config", "test.textproto")
	opsTest, err := LoadOpsServerConfig(opsTestPath)
	if err != nil {
		t.Fatalf("failed to load ops-server test: %v", err)
	}
	if opsTest.Server.Port != 8081 || opsTest.InternalDatabase.Name != "quadsmith_ops" {
		t.Errorf("unexpected ops-server test config: %+v", opsTest)
	}

	// Staging & Prod (require QS_DATABASE_URL, QS_INTERNAL_DATABASE_URL, and QS_AUTH_SECRET)
	t.Setenv("QS_DATABASE_URL", "postgres://user:pass@db.example.com:5432/quadsmith_api")
	t.Setenv("QS_INTERNAL_DATABASE_URL", "postgres://user:pass@db.example.com:5432/quadsmith_ops")
	t.Setenv("QS_AUTH_SECRET", "super-secret-prod-auth-key-32bytes")
	t.Setenv("QS_OAUTH_CLIENT_ID", "dummy-client-id")
	t.Setenv("QS_OAUTH_CLIENT_SECRET", "dummy-client-secret")

	apiStagingPath := filepath.Join("..", "..", "..", "server-api", "config", "staging.textproto")
	apiStaging, err := LoadAPIServerConfig(apiStagingPath)
	if err != nil {
		t.Fatalf("failed to load api-server staging: %v", err)
	}
	if apiStaging.Server.Port != 443 {
		t.Errorf("unexpected api-server staging port: %d", apiStaging.Server.Port)
	}

	apiProdPath := filepath.Join("..", "..", "..", "server-api", "config", "prod.textproto")
	apiProd, err := LoadAPIServerConfig(apiProdPath)
	if err != nil {
		t.Fatalf("failed to load api-server prod: %v", err)
	}
	if apiProd.Server.Port != 443 {
		t.Errorf("unexpected api-server prod port: %d", apiProd.Server.Port)
	}

	opsStagingPath := filepath.Join("..", "..", "..", "server-ops", "config", "staging.textproto")
	opsStaging, err := LoadOpsServerConfig(opsStagingPath)
	if err != nil {
		t.Fatalf("failed to load ops-server staging: %v", err)
	}
	if opsStaging.Server.Port != 8081 {
		t.Errorf("unexpected ops-server staging port: %d", opsStaging.Server.Port)
	}

	opsProdPath := filepath.Join("..", "..", "..", "server-ops", "config", "prod.textproto")
	opsProd, err := LoadOpsServerConfig(opsProdPath)
	if err != nil {
		t.Fatalf("failed to load ops-server prod: %v", err)
	}
	if opsProd.Server.Port != 8081 {
		t.Errorf("unexpected ops-server prod port: %d", opsProd.Server.Port)
	}
}
