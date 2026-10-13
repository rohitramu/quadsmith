package config

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	quadsmith_internal "quadsmith/api/gen/quadsmith_internal"
)

type APIServerOption func(*quadsmith_internal.ApiServerConfig)

func WithServerPort(port int32) APIServerOption {
	return func(c *quadsmith_internal.ApiServerConfig) {
		if c.Server == nil {
			c.Server = &quadsmith_internal.ServerConfig{}
		}
		c.Server.Port = port
	}
}

func WithDatabaseURL(url string) APIServerOption {
	return func(c *quadsmith_internal.ApiServerConfig) {
		if c.Database == nil {
			c.Database = &quadsmith_internal.DatabaseConfig{}
		}
		c.Database.Url = url
	}
}

func WithDatabaseName(name string) APIServerOption {
	return func(c *quadsmith_internal.ApiServerConfig) {
		if c.Database == nil {
			c.Database = &quadsmith_internal.DatabaseConfig{}
		}
		c.Database.Name = name
	}
}

func WithAuthSecret(secret string) APIServerOption {
	return func(c *quadsmith_internal.ApiServerConfig) {
		if c.Auth == nil {
			c.Auth = &quadsmith_internal.AuthConfig{}
		}
		c.Auth.Secret = secret
	}
}

func WithEmailCooldown(d time.Duration) APIServerOption {
	return func(c *quadsmith_internal.ApiServerConfig) {
		if c.Auth == nil {
			c.Auth = &quadsmith_internal.AuthConfig{}
		}
		c.Auth.EmailCooldown = durationpb.New(d)
	}
}

type WorkerOption func(*quadsmith_internal.WorkerConfig)

func WithWorkerPort(port int32) WorkerOption {
	return func(c *quadsmith_internal.WorkerConfig) {
		if c.Worker == nil {
			c.Worker = &quadsmith_internal.WorkerDaemonConfig{}
		}
		c.Worker.Port = port
	}
}

func WithWorkerConcurrency(concurrency int32) WorkerOption {
	return func(c *quadsmith_internal.WorkerConfig) {
		if c.Worker == nil {
			c.Worker = &quadsmith_internal.WorkerDaemonConfig{}
		}
		c.Worker.Concurrency = concurrency
	}
}

func WithWorkerDatabaseURL(url string) WorkerOption {
	return func(c *quadsmith_internal.WorkerConfig) {
		if c.Database == nil {
			c.Database = &quadsmith_internal.DatabaseConfig{}
		}
		c.Database.Url = url
	}
}

func WithWorkerDatabaseName(name string) WorkerOption {
	return func(c *quadsmith_internal.WorkerConfig) {
		if c.Database == nil {
			c.Database = &quadsmith_internal.DatabaseConfig{}
		}
		c.Database.Name = name
	}
}

func WithWorkerInternalDatabaseURL(url string) WorkerOption {
	return func(c *quadsmith_internal.WorkerConfig) {
		if c.InternalDatabase == nil {
			c.InternalDatabase = &quadsmith_internal.DatabaseConfig{}
		}
		c.InternalDatabase.Url = url
	}
}

func WithWorkerInternalDatabaseName(name string) WorkerOption {
	return func(c *quadsmith_internal.WorkerConfig) {
		if c.InternalDatabase == nil {
			c.InternalDatabase = &quadsmith_internal.DatabaseConfig{}
		}
		c.InternalDatabase.Name = name
	}
}

func NewTestAPIServerConfig(t testing.TB, opts ...APIServerOption) *quadsmith_internal.ApiServerConfig {
	t.Helper()
	cfg := &quadsmith_internal.ApiServerConfig{
		Environment: "test",
		Server: &quadsmith_internal.ServerConfig{
			Port:            8080,
			StaticDir:       "src/frontend/web/dist",
			ReadTimeout:     durationpb.New(30 * time.Second),
			WriteTimeout:    durationpb.New(60 * time.Second),
			IdleTimeout:     durationpb.New(120 * time.Second),
			MaxRequestBytes: 2097152,
		},
		Database: &quadsmith_internal.DatabaseConfig{
			Url:            "postgres://postgres:postgres@127.0.0.1:5432/quadsmith?sslmode=disable",
			Name:           "quadsmith",
			MaxConnections: 25,
			MinConnections: 2,
		},
		Auth: &quadsmith_internal.AuthConfig{
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

func NewTestWorkerConfig(t testing.TB, opts ...WorkerOption) *quadsmith_internal.WorkerConfig {
	t.Helper()
	cfg := &quadsmith_internal.WorkerConfig{
		Environment: "test",
		Worker: &quadsmith_internal.WorkerDaemonConfig{
			Port:        8081,
			Concurrency: 2,
		},
		Database: &quadsmith_internal.DatabaseConfig{
			Url:            "postgres://postgres:postgres@127.0.0.1:5432/quadsmith?sslmode=disable",
			Name:           "quadsmith",
			MaxConnections: 25,
			MinConnections: 2,
		},
		InternalDatabase: &quadsmith_internal.DatabaseConfig{
			Url:            "postgres://postgres:postgres@127.0.0.1:5432/quadsmith_internal?sslmode=disable",
			Name:           "quadsmith_internal",
			MaxConnections: 25,
			MinConnections: 2,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
