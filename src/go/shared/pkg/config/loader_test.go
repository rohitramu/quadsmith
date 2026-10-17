package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/durationpb"
	configpb "quadsmith/shared/gen/quadsmith/config"
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
		db      *configpb.DatabaseConfig
		env     string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid database config",
			db: &configpb.DatabaseConfig{
				Url:            proto.String("postgres://user:pass@127.0.0.1:5432/quadsmith_api"),
				Name:           proto.String("quadsmith_api"),
				MaxConnections: proto.Int32(20),
				MinConnections: proto.Int32(5),
			},
			env:     "sandbox",
			wantErr: false,
		},
		{
			name: "missing database name",
			db: &configpb.DatabaseConfig{
				Url:            proto.String("postgres://user:pass@127.0.0.1:5432/quadsmith_api"),
				Name:           proto.String(""),
				MaxConnections: proto.Int32(20),
				MinConnections: proto.Int32(5),
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "database.name cannot be empty",
		},
		{
			name: "database name mismatch with url path",
			db: &configpb.DatabaseConfig{
				Url:            proto.String("postgres://user:pass@127.0.0.1:5432/quadsmith_api"),
				Name:           proto.String("quadsmith_ops"),
				MaxConnections: proto.Int32(20),
				MinConnections: proto.Int32(5),
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "does not match configured database name",
		},
		{
			name: "min connections exceeds max connections",
			db: &configpb.DatabaseConfig{
				Url:            proto.String("postgres://user:pass@127.0.0.1:5432/quadsmith_api"),
				Name:           proto.String("quadsmith_api"),
				MaxConnections: proto.Int32(5),
				MinConnections: proto.Int32(10),
			},
			env:     "sandbox",
			wantErr: true,
			errMsg:  "cannot exceed max_connections",
		},
		{
			name: "localhost rejected in production",
			db: &configpb.DatabaseConfig{
				Url:            proto.String("postgres://user:pass@localhost:5432/quadsmith_api"),
				Name:           proto.String("quadsmith_api"),
				MaxConnections: proto.Int32(20),
				MinConnections: proto.Int32(5),
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

	cfg.Server.Port = proto.Int32(70000)
	if err := ValidateAPIServerConfig(cfg); err == nil {
		t.Error("expected error for invalid port, got nil")
	}

	webCfg := &configpb.WebServerConfig{
		Environment: proto.String("sandbox"),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(3000),
			ReadTimeout:     durationpb.New(15 * time.Second),
			WriteTimeout:    durationpb.New(15 * time.Second),
			IdleTimeout:     durationpb.New(60 * time.Second),
			MaxRequestBytes: proto.Int64(10485760),
		},
		StaticDir: proto.String(""),
		ApiUrl:    proto.String("http://localhost:8080"),
	}
	if err := ValidateWebServerConfig(webCfg); err == nil {
		t.Error("expected error for empty static dir on web server, got nil")
	}
}

func TestValidateNoUnsetFields(t *testing.T) {
	cfg := NewTestAPIServerConfig(t)
	if err := ValidateNoUnsetFields(cfg); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	// Unset a scalar field
	cfg.Auth.DevPrintOtp = nil
	if err := ValidateNoUnsetFields(cfg); err == nil {
		t.Fatal("expected error when scalar field is unset, got nil")
	}

	// Restore and unset a nested message field
	cfg.Auth.DevPrintOtp = proto.Bool(false)
	cfg.Server.IdleTimeout = nil
	if err := ValidateNoUnsetFields(cfg); err == nil {
		t.Fatal("expected error when message field is unset, got nil")
	}

	// Restore and unset oneof
	cfg.Server.IdleTimeout = durationpb.New(120 * time.Second)
	cfg.Auth.Email.Delivery = nil
	if err := ValidateNoUnsetFields(cfg); err == nil {
		t.Fatal("expected error when oneof is unset, got nil")
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
  idle_timeout { seconds: 60 }
  max_request_bytes: 1048576
}
database {
  url: "postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api"
  name: "quadsmith_api"
  max_connections: 10
  min_connections: 2
}
auth {
  secret: "${SECRET_VAR}"
  dev_print_otp: false
  secure_cookies: false
  trusted_proxy_hops: 0
  access_token_ttl { seconds: 3600 }
  refresh_token_ttl { seconds: 86400 }
  otp_ttl { seconds: 300 }
  magic_link_ttl { seconds: 900 }
  email_cooldown { seconds: 1 }
  email {
    console {}
  }
  oauth {
    web_base_url: "http://localhost:5173"
    mock {}
  }
}
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	var cfg configpb.ApiServerConfig
	if err := Load(configPath, &cfg); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.GetServer().GetPort() != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.GetServer().GetPort())
	}
	if cfg.GetAuth().GetSecret() != "super-secret-key-12345678901234" {
		t.Errorf("expected secret expanded, got %q", cfg.GetAuth().GetSecret())
	}
	if cfg.GetDatabase().GetName() != "quadsmith_api" {
		t.Errorf("expected database name quadsmith_api, got %q", cfg.GetDatabase().GetName())
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
	if apiSandbox.Server.GetPort() != 8080 || apiSandbox.Database.GetName() != "quadsmith_api" {
		t.Errorf("unexpected api-server sandbox config: %+v", apiSandbox)
	}

	opsSandboxPath := filepath.Join("..", "..", "..", "server-ops", "config", "sandbox.textproto")
	opsSandbox, err := LoadOpsServerConfig(opsSandboxPath)
	if err != nil {
		t.Fatalf("failed to load ops-server sandbox: %v", err)
	}
	if opsSandbox.Server.GetPort() != 8080 || opsSandbox.Database.GetName() != "quadsmith_api" || opsSandbox.InternalDatabase.GetName() != "quadsmith_ops" {
		t.Errorf("unexpected ops-server sandbox config: %+v", opsSandbox)
	}

	jobSandboxPath := filepath.Join("..", "..", "..", "server-jobs", "config", "sandbox.textproto")
	jobSandbox, err := LoadJobServerConfig(jobSandboxPath)
	if err != nil {
		t.Fatalf("failed to load job-server sandbox: %v", err)
	}
	if jobSandbox.GetOpsServerUrl() != "http://127.0.0.1:8082" || !jobSandbox.GetCleanupExpiredOtps().GetEnabled() {
		t.Errorf("unexpected job-server sandbox config: %+v", jobSandbox)
	}

	webSandboxPath := filepath.Join("..", "..", "..", "server-web", "config", "sandbox.textproto")
	webSandbox, err := LoadWebServerConfig(webSandboxPath)
	if err != nil {
		t.Fatalf("failed to load web-server sandbox: %v", err)
	}
	if webSandbox.Server.GetPort() != 8080 || webSandbox.GetStaticDir() != "src/react/app-web/dist" {
		t.Errorf("unexpected web-server sandbox config: %+v", webSandbox)
	}

	opsCliSandbox, err := LoadOpsCliConfig("", "sandbox")
	if err != nil {
		t.Fatalf("failed to load ops cli sandbox: %v", err)
	}
	if opsCliSandbox.GetOpsServerUrl() != "http://127.0.0.1:8082" {
		t.Errorf("unexpected ops cli sandbox config: %+v", opsCliSandbox)
	}

	cliSandbox, err := LoadCliConfig("", "sandbox")
	if err != nil {
		t.Fatalf("failed to load cli sandbox: %v", err)
	}
	if cliSandbox.GetApiUrl() != "http://127.0.0.1:8081" {
		t.Errorf("unexpected cli sandbox config: %+v", cliSandbox)
	}

	// Staging & Prod (require QS_DATABASE_URL, QS_INTERNAL_DATABASE_URL, QS_JOB_SERVER_AUTH_TOKEN, and QS_AUTH_SECRET)
	t.Setenv("QS_DATABASE_URL", "postgres://user:pass@db.example.com:5432/quadsmith_api")
	t.Setenv("QS_INTERNAL_DATABASE_URL", "postgres://user:pass@db.example.com:5432/quadsmith_ops")
	t.Setenv("QS_JOB_SERVER_AUTH_TOKEN", "qs_robot_prod_token_secret")
	t.Setenv("QS_AUTH_SECRET", "super-secret-prod-auth-key-32bytes")
	t.Setenv("QS_OAUTH_CLIENT_ID", "dummy-client-id")
	t.Setenv("QS_OAUTH_CLIENT_SECRET", "dummy-client-secret")
	t.Setenv("QS_SMTP_PASSWORD", "dummy-smtp-password")
	t.Setenv("QS_GOOGLE_CLIENT_ID", "dummy-google-id")
	t.Setenv("QS_GOOGLE_CLIENT_SECRET", "dummy-google-secret")
	t.Setenv("QS_GITHUB_CLIENT_ID", "dummy-github-id")
	t.Setenv("QS_GITHUB_CLIENT_SECRET", "dummy-github-secret")
	t.Setenv("QS_DISCORD_CLIENT_ID", "dummy-discord-id")
	t.Setenv("QS_DISCORD_CLIENT_SECRET", "dummy-discord-secret")

	apiStagingPath := filepath.Join("..", "..", "..", "server-api", "config", "staging.textproto")
	apiStaging, err := LoadAPIServerConfig(apiStagingPath)
	if err != nil {
		t.Fatalf("failed to load api-server staging: %v", err)
	}
	if apiStaging.Server.GetPort() != 8080 {
		t.Errorf("unexpected api-server staging port: %d", apiStaging.Server.GetPort())
	}

	apiProdPath := filepath.Join("..", "..", "..", "server-api", "config", "prod.textproto")
	apiProd, err := LoadAPIServerConfig(apiProdPath)
	if err != nil {
		t.Fatalf("failed to load api-server prod: %v", err)
	}
	if apiProd.Server.GetPort() != 8080 {
		t.Errorf("unexpected api-server prod port: %d", apiProd.Server.GetPort())
	}

	webStagingPath := filepath.Join("..", "..", "..", "server-web", "config", "staging.textproto")
	webStaging, err := LoadWebServerConfig(webStagingPath)
	if err != nil {
		t.Fatalf("failed to load web-server staging: %v", err)
	}
	if webStaging.Server.GetPort() != 8080 {
		t.Errorf("unexpected web-server staging port: %d", webStaging.Server.GetPort())
	}

	webProdPath := filepath.Join("..", "..", "..", "server-web", "config", "prod.textproto")
	webProd, err := LoadWebServerConfig(webProdPath)
	if err != nil {
		t.Fatalf("failed to load web-server prod: %v", err)
	}
	if webProd.Server.GetPort() != 8080 {
		t.Errorf("unexpected web-server prod port: %d", webProd.Server.GetPort())
	}

	opsStagingPath := filepath.Join("..", "..", "..", "server-ops", "config", "staging.textproto")
	opsStaging, err := LoadOpsServerConfig(opsStagingPath)
	if err != nil {
		t.Fatalf("failed to load ops-server staging: %v", err)
	}
	if opsStaging.Server.GetPort() != 8080 {
		t.Errorf("unexpected ops-server staging port: %d", opsStaging.Server.GetPort())
	}

	opsProdPath := filepath.Join("..", "..", "..", "server-ops", "config", "prod.textproto")
	opsProd, err := LoadOpsServerConfig(opsProdPath)
	if err != nil {
		t.Fatalf("failed to load ops-server prod: %v", err)
	}
	if opsProd.Server.GetPort() != 8080 {
		t.Errorf("unexpected ops-server prod port: %d", opsProd.Server.GetPort())
	}

	jobStagingPath := filepath.Join("..", "..", "..", "server-jobs", "config", "staging.textproto")
	jobStaging, err := LoadJobServerConfig(jobStagingPath)
	if err != nil {
		t.Fatalf("failed to load job-server staging: %v", err)
	}
	if jobStaging.GetOpsServerUrl() != "https://staging-ops.quadsmith.com" {
		t.Errorf("unexpected job-server staging ops_server_url: %s", jobStaging.GetOpsServerUrl())
	}

	jobProdPath := filepath.Join("..", "..", "..", "server-jobs", "config", "prod.textproto")
	jobProd, err := LoadJobServerConfig(jobProdPath)
	if err != nil {
		t.Fatalf("failed to load job-server prod: %v", err)
	}
	if jobProd.GetOpsServerUrl() != "https://ops.quadsmith.com" {
		t.Errorf("unexpected job-server prod ops_server_url: %s", jobProd.GetOpsServerUrl())
	}

	opsCliStaging, err := LoadOpsCliConfig("", "staging")
	if err != nil {
		t.Fatalf("failed to load ops cli staging: %v", err)
	}
	if opsCliStaging.GetOpsServerUrl() != "https://staging-ops.quadsmith.com" {
		t.Errorf("unexpected ops cli staging config: %+v", opsCliStaging)
	}

	opsCliProd, err := LoadOpsCliConfig("", "prod")
	if err != nil {
		t.Fatalf("failed to load ops cli prod: %v", err)
	}
	if opsCliProd.GetOpsServerUrl() != "https://ops.quadsmith.com" {
		t.Errorf("unexpected ops cli prod config: %+v", opsCliProd)
	}
}
