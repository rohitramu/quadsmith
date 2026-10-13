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

// LoadAPIServerConfig loads and parses an API server configuration from a path or environment name.
func LoadAPIServerConfig(configPath, envName string) (*APIServerConfig, error) {
	resolvedPath, err := resolveConfigPath("api-server", configPath, envName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %q: %w", resolvedPath, err)
	}

	expanded, err := ExpandTemplates(data)
	if err != nil {
		return nil, fmt.Errorf("configuration template error in %q:\n%w", resolvedPath, err)
	}

	var cfg APIServerConfig
	decoder := yaml.NewDecoder(bytes.NewReader(expanded))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %q: %w", resolvedPath, err)
	}

	if cfg.Environment == "" {
		cfg.Environment = inferEnvironment(resolvedPath, envName)
	}

	applyAPIServerEnvOverrides(&cfg)

	return &cfg, nil
}

// LoadWorkerConfig loads and parses a background worker configuration from a path or environment name.
func LoadWorkerConfig(configPath, envName string) (*WorkerConfig, error) {
	resolvedPath, err := resolveConfigPath("worker", configPath, envName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %q: %w", resolvedPath, err)
	}

	expanded, err := ExpandTemplates(data)
	if err != nil {
		return nil, fmt.Errorf("configuration template error in %q:\n%w", resolvedPath, err)
	}

	var cfg WorkerConfig
	decoder := yaml.NewDecoder(bytes.NewReader(expanded))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %q: %w", resolvedPath, err)
	}

	if cfg.Environment == "" {
		cfg.Environment = inferEnvironment(resolvedPath, envName)
	}

	applyWorkerEnvOverrides(&cfg)

	return &cfg, nil
}

func resolveConfigPath(binaryName, configPath, envName string) (string, error) {
	if configPath != "" {
		return configPath, nil
	}

	if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
		return envFile, nil
	}

	if envName == "" {
		envName = os.Getenv("QS_ENV")
		if envName == "" {
			envName = "sandbox"
		}
	}

	return FindBinaryConfigFile(binaryName, envName)
}

func inferEnvironment(filePath, fallbackEnv string) string {
	if envOverride := os.Getenv("QS_ENV"); envOverride != "" {
		return envOverride
	}
	if fallbackEnv != "" {
		return fallbackEnv
	}
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func applyAPIServerEnvOverrides(cfg *APIServerConfig) {
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

	applyDatabaseEnvOverrides(&cfg.Database)

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
}

func applyWorkerEnvOverrides(cfg *WorkerConfig) {
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

	applyDatabaseEnvOverrides(&cfg.Database)
}

func applyDatabaseEnvOverrides(db *DatabaseConfig) {
	if v := os.Getenv("QS_DATABASE_URL"); v != "" {
		db.URL = v
	}
	if v := os.Getenv("QS_DATABASE_MAX_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db.MaxConnections = n
		}
	}
	if v := os.Getenv("QS_DATABASE_MIN_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			db.MinConnections = n
		}
	}
}

// Validate ensures all configuration required by the API server daemon is present and valid.
func (c *APIServerConfig) Validate() error {
	var errs []string

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

	if err := validateDatabase(&c.Database, c.Environment); err != nil {
		errs = append(errs, err.Error())
	}

	if err := validateAuth(&c.Auth, c.Environment); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return formatErrors("API Server", errs)
	}
	return nil
}

// ValidateSeeder validates database settings required by the seeder.
func (c *APIServerConfig) ValidateSeeder() error {
	if err := validateDatabase(&c.Database, c.Environment); err != nil {
		return formatErrors("Database Seeder", []string{err.Error()})
	}
	return nil
}

// Validate ensures all configuration required by the background worker daemon is present and valid.
func (c *WorkerConfig) Validate() error {
	var errs []string

	if c.Worker.Port < 1 || c.Worker.Port > 65535 {
		errs = append(errs, fmt.Sprintf("worker.port must be between 1 and 65535 (got %d)", c.Worker.Port))
	}
	if c.Worker.Concurrency <= 0 {
		errs = append(errs, fmt.Sprintf("worker.concurrency must be greater than 0 (got %d)", c.Worker.Concurrency))
	}

	if err := validateDatabase(&c.Database, c.Environment); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return formatErrors("Background Worker", errs)
	}
	return nil
}

func validateDatabase(db *DatabaseConfig, env string) error {
	if strings.TrimSpace(db.URL) == "" {
		return fmt.Errorf("database.url cannot be empty")
	}
	parsed, err := url.Parse(db.URL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return fmt.Errorf("database.url must be a valid postgres connection string")
	}
	if db.MaxConnections <= 0 {
		return fmt.Errorf("database.max_connections must be greater than 0 (got %d)", db.MaxConnections)
	}
	if isProduction(env) || isStaging(env) {
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" {
			return fmt.Errorf("database.url cannot point to localhost/127.0.0.1 in %s environment", env)
		}
	}
	return nil
}

func validateAuth(auth *AuthConfig, env string) error {
	var errs []string
	if strings.TrimSpace(auth.Secret) == "" {
		errs = append(errs, "auth.secret cannot be empty")
	}
	if auth.AccessTokenTTL <= 0 {
		errs = append(errs, "auth.access_token_ttl must be greater than 0")
	}
	if auth.RefreshTokenTTL <= 0 {
		errs = append(errs, "auth.refresh_token_ttl must be greater than 0")
	}
	if auth.OtpTTL <= 0 {
		errs = append(errs, "auth.otp_ttl must be greater than 0")
	}
	if auth.MagicLinkTTL <= 0 {
		errs = append(errs, "auth.magic_link_ttl must be greater than 0")
	}
	if auth.EmailCooldown <= 0 {
		errs = append(errs, "auth.email_cooldown must be greater than 0")
	}

	if isProduction(env) || isStaging(env) {
		if auth.Secret == "quadsmith-dev-secret-key-32bytes-long!" || auth.Secret == "quadsmith-test-secret-key-32bytes-long!" {
			errs = append(errs, fmt.Sprintf("auth.secret cannot use development default secret in %s environment", env))
		}
		if !auth.SecureCookies {
			errs = append(errs, fmt.Sprintf("auth.secure_cookies must be true in %s environment", env))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func isProduction(env string) bool {
	return strings.EqualFold(env, "prod") || strings.EqualFold(env, "production")
}

func isStaging(env string) bool {
	return strings.EqualFold(env, "staging")
}

func formatErrors(component string, errs []string) error {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("FATAL: %s configuration validation failed:\n", component))
	for _, e := range errs {
		b.WriteString(fmt.Sprintf("  - %s\n", e))
	}
	return fmt.Errorf("%s", strings.TrimRight(b.String(), "\n"))
}

// FindBinaryConfigFile searches for the YAML file matching binaryName and envName.
func FindBinaryConfigFile(binaryName, envName string) (string, error) {
	filename := fmt.Sprintf("%s.yaml", strings.ToLower(envName))

	// Search locations relative to working directory or container root
	candidates := []string{
		filepath.Join("src", "backend", "config", binaryName, filename),
		filepath.Join("backend", "config", binaryName, filename),
		filepath.Join("config", binaryName, filename),
		filepath.Join("..", "src", "backend", "config", binaryName, filename),
		filepath.Join("..", "config", binaryName, filename),
		filepath.Join("..", binaryName, filename),
		filepath.Join("..", "..", "src", "backend", "config", binaryName, filename),
		filepath.Join("..", "..", "..", "src", "backend", "config", binaryName, filename),
		filepath.Join("/app", "src", "backend", "config", binaryName, filename),
		filepath.Join("/app", "backend", "config", binaryName, filename),
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

	return "", fmt.Errorf("unable to locate configuration file %q for binary %q and environment %q", filename, binaryName, envName)
}
