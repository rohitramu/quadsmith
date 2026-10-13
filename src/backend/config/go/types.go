package config

import "time"

// APIServerConfig represents configuration for the public API server.
type APIServerConfig struct {
	Environment string         `yaml:"environment"`
	Server      ServerConfig   `yaml:"server"`
	Database    DatabaseConfig `yaml:"database"`
	Auth        AuthConfig     `yaml:"auth"`
}

// WorkerConfig represents configuration for the background worker daemon.
type WorkerConfig struct {
	Environment string             `yaml:"environment"`
	Worker      WorkerDaemonConfig `yaml:"worker"`
	Database    DatabaseConfig     `yaml:"database"`
}

// ServerConfig configures the API HTTP server.
type ServerConfig struct {
	Port            int           `yaml:"port"`
	StaticDir       string        `yaml:"static_dir"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	MaxRequestBytes int64         `yaml:"max_request_bytes"`
}

// WorkerDaemonConfig configures background worker execution and ConnectRPC worker service.
type WorkerDaemonConfig struct {
	Port        int `yaml:"port"`
	Concurrency int `yaml:"concurrency"`
}

// DatabaseConfig configures PostgreSQL connection pooling.
type DatabaseConfig struct {
	URL            string `yaml:"url"`
	MaxConnections int    `yaml:"max_connections"`
	MinConnections int    `yaml:"min_connections"`
}

// AuthConfig configures authentication, sessions, tokens, and verification OTPs.
type AuthConfig struct {
	Secret           string        `yaml:"secret"`
	DevPrintOTP      bool          `yaml:"dev_print_otp"`
	SecureCookies    bool          `yaml:"secure_cookies"`
	TrustedProxyHops int           `yaml:"trusted_proxy_hops"`
	AccessTokenTTL   time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `yaml:"refresh_token_ttl"`
	OtpTTL           time.Duration `yaml:"otp_ttl"`
	MagicLinkTTL     time.Duration `yaml:"magic_link_ttl"`
	EmailCooldown    time.Duration `yaml:"email_cooldown"`
}
