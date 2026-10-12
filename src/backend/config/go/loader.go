package config

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

var placeholderRegex = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)(?::-([^}]*))?\}`)

// ExpandTemplates replaces ${VAR} and ${VAR:-default} placeholders with environment values.
// If a required variable ${VAR} is unset and has no default, an error is recorded.
func ExpandTemplates(content []byte) ([]byte, error) {
	var missingVars []string
	expanded := placeholderRegex.ReplaceAllFunc(content, func(match []byte) []byte {
		submatches := placeholderRegex.FindSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		varName := string(submatches[1])
		hasDefault := len(submatches) >= 3 && submatches[2] != nil
		defaultValue := ""
		if hasDefault {
			defaultValue = string(submatches[2])
		}

		val, found := os.LookupEnv(varName)
		if found && val != "" {
			return []byte(val)
		}
		if hasDefault {
			return []byte(defaultValue)
		}
		missingVars = append(missingVars, varName)
		return match
	})

	if len(missingVars) > 0 {
		var errMsgs []string
		for _, v := range missingVars {
			errMsgs = append(errMsgs, fmt.Sprintf("Missing required secret environment variable: %s", v))
		}
		return nil, fmt.Errorf("%s", strings.Join(errMsgs, "\n"))
	}

	return expanded, nil
}

// Load loads and parses a backend configuration from the given file path.
// If configPath is empty, it locates the default file for the active environment (QS_ENV or "sandbox").
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			envName := os.Getenv("QS_ENV")
			if envName == "" {
				envName = "sandbox"
			}
			resolved, err := FindBackendConfigFile(envName)
			if err != nil {
				return nil, err
			}
			configPath = resolved
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %q: %w", configPath, err)
	}

	expanded, err := ExpandTemplates(data)
	if err != nil {
		return nil, fmt.Errorf("configuration template error in %q:\n%w", configPath, err)
	}

	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(expanded))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %q: %w", configPath, err)
	}

	// Infer environment from filename if not explicitly defined
	if cfg.Environment == "" {
		base := filepath.Base(configPath)
		ext := filepath.Ext(base)
		cfg.Environment = strings.TrimSuffix(base, ext)
	}
	if envOverride := os.Getenv("QS_ENV"); envOverride != "" {
		cfg.Environment = envOverride
	}

	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	// Server overrides
	if v := os.Getenv("QS_SERVER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("QS_SERVER_STATIC_DIR"); v != "" {
		cfg.Server.StaticDir = v
	}
	if v := os.Getenv("QS_SERVER_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv("QS_SERVER_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}

	// Database overrides
	if v := os.Getenv("QS_DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("QS_DATABASE_MAX_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxConnections = n
		}
	}
	if v := os.Getenv("QS_DATABASE_MIN_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MinConnections = n
		}
	}

	// Auth overrides
	if v := os.Getenv("QS_AUTH_SECRET"); v != "" {
		cfg.Auth.Secret = v
	}
	if v := os.Getenv("QS_AUTH_DEV_PRINT_OTP"); v != "" {
		cfg.Auth.DevPrintOTP = v == "true" || v == "1"
	}
	if v := os.Getenv("QS_AUTH_SECURE_COOKIES"); v != "" {
		cfg.Auth.SecureCookies = v == "true" || v == "1"
	}
	if v := os.Getenv("QS_AUTH_TRUSTED_PROXY_HOPS"); v != "" {
		if hops, err := strconv.Atoi(v); err == nil {
			cfg.Auth.TrustedProxyHops = hops
		}
	}
	if v := os.Getenv("QS_AUTH_ACCESS_TOKEN_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.AccessTokenTTL = d
		}
	}
	if v := os.Getenv("QS_AUTH_REFRESH_TOKEN_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.RefreshTokenTTL = d
		}
	}
	if v := os.Getenv("QS_AUTH_OTP_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.OtpTTL = d
		}
	}
	if v := os.Getenv("QS_AUTH_MAGIC_LINK_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.MagicLinkTTL = d
		}
	}
	if v := os.Getenv("QS_AUTH_EMAIL_COOLDOWN"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.EmailCooldown = d
		}
	}

	// Worker overrides
	if v := os.Getenv("QS_WORKER_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Worker.Port = p
		}
	}
	if v := os.Getenv("QS_WORKER_CONCURRENCY"); v != "" {
		if c, err := strconv.Atoi(v); err == nil {
			cfg.Worker.Concurrency = c
		}
	}
}

// ValidateServer ensures all configuration required by the API server daemon is present and valid.
func (c *Config) ValidateServer() error {
	var errs []string

	// Server validation
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port must be between 1 and 65535 (got %d)", c.Server.Port))
	}
	if c.Server.StaticDir == "" {
		errs = append(errs, "server.static_dir cannot be empty")
	}
	if c.Server.ReadTimeout <= 0 {
		errs = append(errs, "server.read_timeout must be greater than 0")
	}
	if c.Server.WriteTimeout <= 0 {
		errs = append(errs, "server.write_timeout must be greater than 0")
	}
	if c.Server.MaxRequestBytes <= 0 {
		errs = append(errs, "server.max_request_bytes must be greater than 0")
	}

	// Database validation
	if err := c.validateDatabase(); err != nil {
		errs = append(errs, err.Error())
	}

	// Auth validation
	if err := c.validateAuth(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return formatErrors("API Server", errs)
	}
	return nil
}

// ValidateWorker ensures all configuration required by the background worker daemon is present and valid.
func (c *Config) ValidateWorker() error {
	var errs []string

	if c.Worker.Port < 1 || c.Worker.Port > 65535 {
		errs = append(errs, fmt.Sprintf("worker.port must be between 1 and 65535 (got %d)", c.Worker.Port))
	}
	if c.Worker.Concurrency <= 0 {
		errs = append(errs, fmt.Sprintf("worker.concurrency must be greater than 0 (got %d)", c.Worker.Concurrency))
	}

	if err := c.validateDatabase(); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return formatErrors("Background Worker", errs)
	}
	return nil
}

// ValidateSeeder ensures all configuration required by the database seeder is present and valid.
func (c *Config) ValidateSeeder() error {
	if err := c.validateDatabase(); err != nil {
		return formatErrors("Database Seeder", []string{err.Error()})
	}
	return nil
}

func (c *Config) validateDatabase() error {
	if strings.TrimSpace(c.Database.URL) == "" {
		return fmt.Errorf("database.url cannot be empty")
	}
	parsed, err := url.Parse(c.Database.URL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return fmt.Errorf("database.url must be a valid postgres connection string")
	}
	if c.Database.MaxConnections <= 0 {
		return fmt.Errorf("database.max_connections must be greater than 0 (got %d)", c.Database.MaxConnections)
	}
	if c.IsProduction() || c.IsStaging() {
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" {
			return fmt.Errorf("database.url cannot point to localhost/127.0.0.1 in %s environment", c.Environment)
		}
	}
	return nil
}

func (c *Config) validateAuth() error {
	var errs []string
	if strings.TrimSpace(c.Auth.Secret) == "" {
		errs = append(errs, "auth.secret cannot be empty")
	}
	if c.Auth.AccessTokenTTL <= 0 {
		errs = append(errs, "auth.access_token_ttl must be greater than 0")
	}
	if c.Auth.RefreshTokenTTL <= 0 {
		errs = append(errs, "auth.refresh_token_ttl must be greater than 0")
	}
	if c.Auth.OtpTTL <= 0 {
		errs = append(errs, "auth.otp_ttl must be greater than 0")
	}
	if c.Auth.MagicLinkTTL <= 0 {
		errs = append(errs, "auth.magic_link_ttl must be greater than 0")
	}
	if c.Auth.EmailCooldown <= 0 {
		errs = append(errs, "auth.email_cooldown must be greater than 0")
	}

	if c.IsProduction() || c.IsStaging() {
		if c.Auth.Secret == "quadsmith-dev-secret-key-32bytes-long!" || c.Auth.Secret == "quadsmith-test-secret-key-32bytes-long!" {
			errs = append(errs, fmt.Sprintf("auth.secret cannot use development default secret in %s environment", c.Environment))
		}
		if !c.Auth.SecureCookies {
			errs = append(errs, fmt.Sprintf("auth.secure_cookies must be true in %s environment", c.Environment))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "prod") || strings.EqualFold(c.Environment, "production")
}

func (c *Config) IsStaging() bool {
	return strings.EqualFold(c.Environment, "staging")
}

func formatErrors(component string, errs []string) error {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("FATAL: %s configuration validation failed:\n", component))
	for _, e := range errs {
		b.WriteString(fmt.Sprintf("  - %s\n", e))
	}
	return fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
}

// FindBackendConfigFile searches for the YAML file matching envName.
func FindBackendConfigFile(envName string) (string, error) {
	filename := fmt.Sprintf("%s.yaml", strings.ToLower(envName))

	// Search locations relative to working directory or container root
	candidates := []string{
		filepath.Join("src", "backend", "config", filename),
		filepath.Join("backend", "config", filename),
		filepath.Join("config", filename),
		filepath.Join("..", "src", "backend", "config", filename),
		filepath.Join("..", "config", filename),
		filepath.Join("..", "..", "src", "backend", "config", filename),
		filepath.Join("..", "..", "..", "src", "backend", "config", filename),
		filepath.Join("/app", "src", "backend", "config", filename),
		filepath.Join("/app", "backend", "config", filename),
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs, nil
			}
			return c, nil
		}
	}

	return "", fmt.Errorf("unable to locate configuration file %q for environment %q (specify with --config <path>)", filename, envName)
}
