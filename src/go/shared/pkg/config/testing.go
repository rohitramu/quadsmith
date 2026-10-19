package config

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/durationpb"
	configpb "quadsmith/shared/gen/quadsmith/config"
)

type APIServerOption func(*configpb.ApiServerConfig)

func WithServerPort(port int32) APIServerOption {
	return func(c *configpb.ApiServerConfig) {
		if c.Server == nil {
			c.Server = &configpb.ServerConfig{}
		}
		c.Server.Port = proto.Int32(port)
	}
}

func WithDatabaseURL(url string) APIServerOption {
	return func(c *configpb.ApiServerConfig) {
		if c.Database == nil {
			c.Database = &configpb.DatabaseConfig{}
		}
		c.Database.Url = proto.String(url)
	}
}

func WithDatabaseName(name string) APIServerOption {
	return func(c *configpb.ApiServerConfig) {
		if c.Database == nil {
			c.Database = &configpb.DatabaseConfig{}
		}
		c.Database.Name = proto.String(name)
	}
}

func WithAuthSecret(secret string) APIServerOption {
	return func(c *configpb.ApiServerConfig) {
		if c.Auth == nil {
			c.Auth = &configpb.AuthConfig{}
		}
		c.Auth.Secret = proto.String(secret)
	}
}

func WithEmailCooldown(d time.Duration) APIServerOption {
	return func(c *configpb.ApiServerConfig) {
		if c.Auth == nil {
			c.Auth = &configpb.AuthConfig{}
		}
		c.Auth.EmailCooldown = durationpb.New(d)
	}
}

func NewTestAPIServerConfig(t testing.TB, opts ...APIServerOption) *configpb.ApiServerConfig {
	t.Helper()
	cfg := &configpb.ApiServerConfig{
		Environment: proto.String("sandbox"),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(8081),
			ReadTimeout:     durationpb.New(30 * time.Second),
			WriteTimeout:    durationpb.New(60 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: proto.Int64(2097152),
		},
		Database: &configpb.DatabaseConfig{
			Url:            proto.String("postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api?sslmode=disable"),
			Name:           proto.String("quadsmith_api"),
			MaxConnections: proto.Int32(25),
			MinConnections: proto.Int32(2),
		},
		Auth: &configpb.AuthConfig{
			Secret:           proto.String("quadsmith-dev-secret-key-32bytes-long!"),
			DevPrintOtp:      proto.Bool(true),
			SecureCookies:    proto.Bool(false),
			TrustedProxyHops: proto.Int32(0),
			AccessTokenTtl:   durationpb.New(1 * time.Hour),
			RefreshTokenTtl:  durationpb.New(120 * time.Hour),
			OtpTtl:           durationpb.New(3 * time.Minute),
			MagicLinkTtl:     durationpb.New(15 * time.Minute),
			EmailCooldown:    durationpb.New(1 * time.Second),
			Email: &configpb.EmailConfig{
				Delivery: &configpb.EmailConfig_Console{
					Console: &configpb.ConsoleEmailConfig{},
				},
			},
			Oauth: &configpb.AuthOAuthConfig{
				WebBaseUrl: proto.String("http://localhost:8080"),
				Mode: &configpb.AuthOAuthConfig_Mock{
					Mock: &configpb.MockOAuthConfig{},
				},
			},
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

type OpsApiServerOption func(*configpb.OpsApiServerConfig)

func WithOpsApiServerPort(port int32) OpsApiServerOption {
	return func(c *configpb.OpsApiServerConfig) {
		if c.Server == nil {
			c.Server = &configpb.ServerConfig{}
		}
		c.Server.Port = proto.Int32(port)
	}
}

func NewTestOpsApiServerConfig(t testing.TB, opts ...OpsApiServerOption) *configpb.OpsApiServerConfig {
	t.Helper()
	cfg := &configpb.OpsApiServerConfig{
		Environment: proto.String("sandbox"),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(8082),
			ReadTimeout:     durationpb.New(30 * time.Second),
			WriteTimeout:    durationpb.New(60 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: proto.Int64(2097152),
		},
		Database: &configpb.DatabaseConfig{
			Url:            proto.String("postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api?sslmode=disable"),
			Name:           proto.String("quadsmith_api"),
			MaxConnections: proto.Int32(25),
			MinConnections: proto.Int32(2),
		},
		InternalDatabase: &configpb.DatabaseConfig{
			Url:            proto.String("postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops?sslmode=disable"),
			Name:           proto.String("quadsmith_ops"),
			MaxConnections: proto.Int32(25),
			MinConnections: proto.Int32(2),
		},
		Groups: []*configpb.InternalGroupConfig{
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
		BootstrapTokens: []*configpb.BootstrapTokenConfig{
			{
				Id:        proto.String("sandbox-operator"),
				Name:      proto.String("Sandbox Operator Token"),
				Token:     proto.String("qs_op_sandbox_token_12345"),
				GroupName: proto.String("operators"),
				Type:      proto.String("HUMAN"),
			},
			{
				Id:        proto.String("sandbox-robot"),
				Name:      proto.String("Sandbox Robot Token"),
				Token:     proto.String("qs_robot_sandbox_token_67890"),
				GroupName: proto.String("robots"),
				Type:      proto.String("ROBOT"),
			},
		},
		Jit: &configpb.JitPolicyConfig{
			Enabled:          proto.Bool(true),
			AutoApprove:      proto.Bool(true),
			DefaultHours:     proto.Int32(2),
			MaxHours:         proto.Int32(12),
			SensitiveMethods: []string{"RevokeInternalToken", "SetUserRole", "LoginAsQuadsmith"},
		},
		Oauth: &configpb.OAuthConfig{
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

func NewTestOpsWebServerConfig(t testing.TB) *configpb.OpsWebServerConfig {
	t.Helper()
	return &configpb.OpsWebServerConfig{
		Environment: proto.String("sandbox"),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(8083),
			ReadTimeout:     durationpb.New(15 * time.Second),
			WriteTimeout:    durationpb.New(15 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: proto.Int64(10485760),
		},
		StaticDir:  proto.String("src/react/app-ops-web/dist"),
		OpsApiUrl:  proto.String("http://127.0.0.1:8082"),
		GrafanaUrl: proto.String("http://127.0.0.1:3000"),
	}
}

func NewTestCronServerConfig(t testing.TB) *configpb.CronServerConfig {
	t.Helper()
	return &configpb.CronServerConfig{
		Environment: proto.String("sandbox"),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(8084),
			ReadTimeout:     durationpb.New(15 * time.Second),
			WriteTimeout:    durationpb.New(15 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: proto.Int64(2097152),
		},
		OpsServerUrl: proto.String("http://127.0.0.1:8082"),
		AuthToken:    proto.String("qs_robot_sandbox_token_67890"),
		CleanupExpiredOtps: &configpb.CleanupExpiredOtpsSchedule{
			Enabled: proto.Bool(true),
			Schedule: &configpb.CleanupExpiredOtpsSchedule_Interval{
				Interval: durationpb.New(6 * time.Hour),
			},
		},
		CleanupExpiredTokens: &configpb.CleanupExpiredTokensSchedule{
			Enabled: proto.Bool(true),
			Schedule: &configpb.CleanupExpiredTokensSchedule_Cron{
				Cron: "0 */12 * * *",
			},
		},
		CleanupOrphanedBlobs: &configpb.CleanupOrphanedBlobsSchedule{
			Enabled: proto.Bool(true),
			Schedule: &configpb.CleanupOrphanedBlobsSchedule_Interval{
				Interval: durationpb.New(10 * time.Minute),
			},
		},
		PurgeSoftDeletedAccounts: &configpb.PurgeSoftDeletedAccountsSchedule{
			Enabled: proto.Bool(true),
			Schedule: &configpb.PurgeSoftDeletedAccountsSchedule_Interval{
				Interval: durationpb.New(8 * time.Hour),
			},
		},
		SendUsernameReleaseWarnings: &configpb.SendUsernameReleaseWarningsSchedule{
			Enabled: proto.Bool(true),
			Schedule: &configpb.SendUsernameReleaseWarningsSchedule_Cron{
				Cron: "0 0 * * *",
			},
		},
	}
}
