package config

import (
	"testing"
	"time"
)

// Option allows customizing a test configuration.
type Option func(*Config)

func WithServerPort(port int) Option {
	return func(c *Config) {
		c.Server.Port = port
	}
}

func WithDatabaseURL(url string) Option {
	return func(c *Config) {
		c.Database.URL = url
	}
}

func WithAuthSecret(secret string) Option {
	return func(c *Config) {
		c.Auth.Secret = secret
	}
}

func WithEmailCooldown(d time.Duration) Option {
	return func(c *Config) {
		c.Auth.EmailCooldown = d
	}
}

func WithWorkerPort(port int) Option {
	return func(c *Config) {
		c.Worker.Port = port
	}
}

func WithWorkerConcurrency(concurrency int) Option {
	return func(c *Config) {
		c.Worker.Concurrency = concurrency
	}
}

// NewTestBackendConfig creates an isolated test configuration initialized from test defaults with options applied.
func NewTestBackendConfig(t testing.TB, opts ...Option) *Config {
	t.Helper()
	cfg := &Config{
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
		Worker: WorkerConfig{
			Port:        8081,
			Concurrency: 2,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
