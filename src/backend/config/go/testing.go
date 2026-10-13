package config

import (
	"testing"
	"time"
)

// APIServerOption allows customizing a test API server configuration.
type APIServerOption func(*APIServerConfig)

func WithServerPort(port int) APIServerOption {
	return func(c *APIServerConfig) {
		c.Server.Port = port
	}
}

func WithDatabaseURL(url string) APIServerOption {
	return func(c *APIServerConfig) {
		c.Database.URL = url
	}
}

func WithAuthSecret(secret string) APIServerOption {
	return func(c *APIServerConfig) {
		c.Auth.Secret = secret
	}
}

func WithEmailCooldown(d time.Duration) APIServerOption {
	return func(c *APIServerConfig) {
		c.Auth.EmailCooldown = d
	}
}

// WorkerOption allows customizing a test worker configuration.
type WorkerOption func(*WorkerConfig)

func WithWorkerPort(port int) WorkerOption {
	return func(c *WorkerConfig) {
		c.Worker.Port = port
	}
}

func WithWorkerConcurrency(concurrency int) WorkerOption {
	return func(c *WorkerConfig) {
		c.Worker.Concurrency = concurrency
	}
}

func WithWorkerDatabaseURL(url string) WorkerOption {
	return func(c *WorkerConfig) {
		c.Database.URL = url
	}
}

// NewTestAPIServerConfig creates an isolated test API server configuration.
func NewTestAPIServerConfig(t testing.TB, opts ...APIServerOption) *APIServerConfig {
	t.Helper()
	cfg := &APIServerConfig{
		Environment: "test",
		Server: ServerConfig{
			Port:            8080,
			StaticDir:       "src/frontend/web/dist",
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    60 * time.Second,
			IdleTimeout:     120 * time.Second,
			MaxRequestBytes: 2097152,
		},
		Database: DatabaseConfig{
			URL:            "postgres://postgres:postgres@127.0.0.1:5432/quadsmith?sslmode=disable",
			MaxConnections: 25,
			MinConnections: 2,
		},
		Auth: AuthConfig{
			Secret:           "quadsmith-test-secret-key-32bytes-long!",
			DevPrintOTP:      false,
			SecureCookies:    false,
			TrustedProxyHops: 0,
			AccessTokenTTL:   1 * time.Hour,
			RefreshTokenTTL:  120 * time.Hour,
			OtpTTL:           3 * time.Minute,
			MagicLinkTTL:     15 * time.Minute,
			EmailCooldown:    1 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// NewTestWorkerConfig creates an isolated test worker configuration.
func NewTestWorkerConfig(t testing.TB, opts ...WorkerOption) *WorkerConfig {
	t.Helper()
	cfg := &WorkerConfig{
		Environment: "test",
		Worker: WorkerDaemonConfig{
			Port:        8081,
			Concurrency: 2,
		},
		Database: DatabaseConfig{
			URL:            "postgres://postgres:postgres@127.0.0.1:5432/quadsmith?sslmode=disable",
			MaxConnections: 25,
			MinConnections: 2,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
