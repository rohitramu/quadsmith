package config

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	config "quadsmith/shared/gen/quadsmith/config"
)

type APIServerOption func(*config.ApiServerConfig)

func WithServerPort(port int32) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Server == nil {
			c.Server = &config.ServerConfig{}
		}
		c.Server.Port = port
	}
}

func WithDatabaseURL(url string) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Database == nil {
			c.Database = &config.DatabaseConfig{}
		}
		c.Database.Url = url
	}
}

func WithDatabaseName(name string) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Database == nil {
			c.Database = &config.DatabaseConfig{}
		}
		c.Database.Name = name
	}
}

func WithAuthSecret(secret string) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Auth == nil {
			c.Auth = &config.AuthConfig{}
		}
		c.Auth.Secret = secret
	}
}

func WithEmailCooldown(d time.Duration) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Auth == nil {
			c.Auth = &config.AuthConfig{}
		}
		c.Auth.EmailCooldown = durationpb.New(d)
	}
}

func NewTestAPIServerConfig(t testing.TB, opts ...APIServerOption) *config.ApiServerConfig {
	t.Helper()
	cfg := &config.ApiServerConfig{
		Environment: "test",
		Server: &config.ServerConfig{
			Port:            8080,
			ReadTimeout:     durationpb.New(30 * time.Second),
			WriteTimeout:    durationpb.New(60 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: 2097152,
		},
		Database: &config.DatabaseConfig{
			Url:            "postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api?sslmode=disable",
			Name:           "quadsmith_api",
			MaxConnections: 25,
			MinConnections: 2,
		},
		Auth: &config.AuthConfig{
			Secret:           "quadsmith-test-secret-key-32bytes-long!",
			DevPrintOtp:      false,
			SecureCookies:    false,
			TrustedProxyHops: 0,
			AccessTokenTtl:   durationpb.New(1 * time.Hour),
			RefreshTokenTtl:  durationpb.New(120 * time.Hour),
			OtpTtl:           durationpb.New(3 * time.Minute),
			MagicLinkTtl:     durationpb.New(15 * time.Minute),
			EmailCooldown:    durationpb.New(1 * time.Second),
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type OpsServerOption func(*config.OpsServerConfig)

func WithOpsServerPort(port int32) OpsServerOption {
	return func(c *config.OpsServerConfig) {
		if c.Server == nil {
			c.Server = &config.ServerConfig{}
		}
		c.Server.Port = port
	}
}

func NewTestOpsServerConfig(t testing.TB, opts ...OpsServerOption) *config.OpsServerConfig {
	t.Helper()
	cfg := &config.OpsServerConfig{
		Environment: "test",
		Server: &config.ServerConfig{
			Port: 8081,
		},
		Database: &config.DatabaseConfig{
			Url:            "postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api?sslmode=disable",
			Name:           "quadsmith_api",
			MaxConnections: 25,
			MinConnections: 2,
		},
		InternalDatabase: &config.DatabaseConfig{
			Url:            "postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops?sslmode=disable",
			Name:           "quadsmith_ops",
			MaxConnections: 25,
			MinConnections: 2,
		},
		Groups: []*config.InternalGroupConfig{
			{
				Name:           "operators",
				Description:    "Full developer ops permissions",
				AllowedMethods: []string{"*"},
			},
			{
				Name:           "robots",
				Description:    "Automated scheduler and cron permissions",
				AllowedMethods: []string{"*"},
			},
			{
				Name:           "viewers",
				Description:    "Read-only internal operator permissions",
				AllowedMethods: []string{"Get*", "List*"},
			},
		},
		BootstrapTokens: []*config.BootstrapTokenConfig{
			{
				Id:        "test-operator-token",
				Name:      "Test Operator Token",
				Token:     "qs_op_test_token_12345",
				GroupName: "operators",
				Type:      "HUMAN",
			},
			{
				Id:        "test-robot-token",
				Name:      "Test Robot Token",
				Token:     "qs_robot_test_token_67890",
				GroupName: "robots",
				Type:      "ROBOT",
			},
		},
		Jit: &config.JitPolicyConfig{
			Enabled:          true,
			AutoApprove:      true,
			DefaultHours:     2,
			MaxHours:         12,
			SensitiveMethods: []string{"RunDatabaseMaintenance", "SeedDatabase"},
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

func NewTestJobServerConfig(t testing.TB) *config.JobServerConfig {
	t.Helper()
	return &config.JobServerConfig{
		Environment:  "test",
		OpsServerUrl: "http://127.0.0.1:8081",
		AuthToken:    "qs_robot_test_token_67890",
		Schedules: []*config.JobScheduleConfig{
			{
				Name:      "cleanup_expired_otps",
				Procedure: "CleanupExpiredOtps",
				Interval:  "6h",
			},
		},
	}
}
