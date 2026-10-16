package config

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/durationpb"
	"quadsmith/shared/gen/quadsmith/config"
)

type APIServerOption func(*config.ApiServerConfig)

func WithServerPort(port int32) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Server == nil {
			c.Server = &config.ServerConfig{}
		}
		c.Server.Port = proto.Int32(port)
	}
}

func WithDatabaseURL(url string) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Database == nil {
			c.Database = &config.DatabaseConfig{}
		}
		c.Database.Url = proto.String(url)
	}
}

func WithDatabaseName(name string) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Database == nil {
			c.Database = &config.DatabaseConfig{}
		}
		c.Database.Name = proto.String(name)
	}
}

func WithAuthSecret(secret string) APIServerOption {
	return func(c *config.ApiServerConfig) {
		if c.Auth == nil {
			c.Auth = &config.AuthConfig{}
		}
		c.Auth.Secret = proto.String(secret)
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
		Environment: proto.String("test"),
		Server: &config.ServerConfig{
			Port:            proto.Int32(8080),
			ReadTimeout:     durationpb.New(30 * time.Second),
			WriteTimeout:    durationpb.New(60 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: proto.Int64(2097152),
		},
		Database: &config.DatabaseConfig{
			Url:            proto.String("postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api?sslmode=disable"),
			Name:           proto.String("quadsmith_api"),
			MaxConnections: proto.Int32(25),
			MinConnections: proto.Int32(2),
		},
		Auth: &config.AuthConfig{
			Secret:           proto.String("quadsmith-test-secret-key-32bytes-long!"),
			DevPrintOtp:      proto.Bool(false),
			SecureCookies:    proto.Bool(false),
			TrustedProxyHops: proto.Int32(0),
			AccessTokenTtl:   durationpb.New(1 * time.Hour),
			RefreshTokenTtl:  durationpb.New(120 * time.Hour),
			OtpTtl:           durationpb.New(3 * time.Minute),
			MagicLinkTtl:     durationpb.New(15 * time.Minute),
			EmailCooldown:    durationpb.New(1 * time.Second),
			Email: &config.EmailConfig{
				Delivery: &config.EmailConfig_Console{
					Console: &config.ConsoleEmailConfig{},
				},
			},
			Oauth: &config.AuthOAuthConfig{
				WebBaseUrl: proto.String("http://localhost:8080"),
				Mode: &config.AuthOAuthConfig_Mock{
					Mock: &config.MockOAuthConfig{},
				},
			},
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
		c.Server.Port = proto.Int32(port)
	}
}

func NewTestOpsServerConfig(t testing.TB, opts ...OpsServerOption) *config.OpsServerConfig {
	t.Helper()
	cfg := &config.OpsServerConfig{
		Environment: proto.String("test"),
		Server: &config.ServerConfig{
			Port:            proto.Int32(8080),
			ReadTimeout:     durationpb.New(30 * time.Second),
			WriteTimeout:    durationpb.New(60 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: proto.Int64(2097152),
		},
		Database: &config.DatabaseConfig{
			Url:            proto.String("postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api?sslmode=disable"),
			Name:           proto.String("quadsmith_api"),
			MaxConnections: proto.Int32(25),
			MinConnections: proto.Int32(2),
		},
		InternalDatabase: &config.DatabaseConfig{
			Url:            proto.String("postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops?sslmode=disable"),
			Name:           proto.String("quadsmith_ops"),
			MaxConnections: proto.Int32(25),
			MinConnections: proto.Int32(2),
		},
		Groups: []*config.InternalGroupConfig{
			{
				Name:           proto.String("operators"),
				Description:    proto.String("Full developer ops permissions"),
				AllowedMethods: []string{"*"},
			},
			{
				Name:           proto.String("robots"),
				Description:    proto.String("Automated scheduler and cron permissions"),
				AllowedMethods: []string{"*"},
			},
			{
				Name:           proto.String("viewers"),
				Description:    proto.String("Read-only internal operator permissions"),
				AllowedMethods: []string{"Get*", "List*"},
			},
		},
		BootstrapTokens: []*config.BootstrapTokenConfig{
			{
				Id:        proto.String("test-operator-token"),
				Name:      proto.String("Test Operator Token"),
				Token:     proto.String("qs_op_test_token_12345"),
				GroupName: proto.String("operators"),
				Type:      proto.String("HUMAN"),
			},
			{
				Id:        proto.String("test-robot-token"),
				Name:      proto.String("Test Robot Token"),
				Token:     proto.String("qs_robot_test_token_67890"),
				GroupName: proto.String("robots"),
				Type:      proto.String("ROBOT"),
			},
		},
		Jit: &config.JitPolicyConfig{
			Enabled:          proto.Bool(true),
			AutoApprove:      proto.Bool(true),
			DefaultHours:     proto.Int32(2),
			MaxHours:         proto.Int32(12),
			SensitiveMethods: []string{"RunDatabaseMaintenance", "SeedDatabase"},
		},
		Oauth: &config.OAuthConfig{
			Provider:     proto.String("github"),
			ClientId:     proto.String("dummy-id"),
			ClientSecret: proto.String("dummy-secret"),
			AllowedOrgs:  []string{"quadsmith"},
			DefaultGroup: proto.String("viewers"),
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
		Environment:  proto.String("test"),
		OpsServerUrl: proto.String("http://127.0.0.1:8082"),
		AuthToken:    proto.String("qs_robot_test_token_67890"),
		Schedules: []*config.JobScheduleConfig{
			{
				Name:      proto.String("cleanup_expired_otps"),
				Procedure: proto.String("CleanupExpiredOtps"),
				Interval:  proto.String("6h"),
			},
		},
	}
}
