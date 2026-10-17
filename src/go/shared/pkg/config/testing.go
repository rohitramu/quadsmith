package config

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	durationpb "google.golang.org/protobuf/types/known/durationpb"
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
		Environment: proto.String("test"),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(8080),
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
			Secret:           proto.String("quadsmith-test-secret-key-32bytes-long!"),
			DevPrintOtp:      proto.Bool(false),
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

type OpsServerOption func(*configpb.OpsServerConfig)

func WithOpsServerPort(port int32) OpsServerOption {
	return func(c *configpb.OpsServerConfig) {
		if c.Server == nil {
			c.Server = &configpb.ServerConfig{}
		}
		c.Server.Port = proto.Int32(port)
	}
}

func NewTestOpsServerConfig(t testing.TB, opts ...OpsServerOption) *configpb.OpsServerConfig {
	t.Helper()
	cfg := &configpb.OpsServerConfig{
		Environment: proto.String("test"),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(8080),
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
		Jit: &configpb.JitPolicyConfig{
			Enabled:          proto.Bool(true),
			AutoApprove:      proto.Bool(true),
			DefaultHours:     proto.Int32(2),
			MaxHours:         proto.Int32(12),
			SensitiveMethods: []string{"RunDatabaseMaintenance", "SeedDatabase"},
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

func NewTestJobServerConfig(t testing.TB) *configpb.JobServerConfig {
	t.Helper()
	return &configpb.JobServerConfig{
		Environment: proto.String("test"),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(8080),
			ReadTimeout:     durationpb.New(30 * time.Second),
			WriteTimeout:    durationpb.New(60 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: proto.Int64(2097152),
		},
		OpsServerUrl: proto.String("http://127.0.0.1:8082"),
		AuthToken:    proto.String("qs_robot_test_token_67890"),
		InternalDatabase: &configpb.DatabaseConfig{
			Url:            proto.String("postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops?sslmode=disable"),
			Name:           proto.String("quadsmith_ops"),
			MaxConnections: proto.Int32(10),
			MinConnections: proto.Int32(2),
		},
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
				Interval: durationpb.New(5 * time.Minute),
			},
		},
		RunDatabaseMaintenance: &configpb.RunDatabaseMaintenanceSchedule{
			Enabled: proto.Bool(false),
			Schedule: &configpb.RunDatabaseMaintenanceSchedule_Cron{
				Cron: "0 3 * * 0",
			},
			FullVacuum: proto.Bool(false),
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
	}
}
