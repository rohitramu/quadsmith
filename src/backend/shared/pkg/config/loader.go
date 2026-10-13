package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	config "quadsmith/shared/gen/quadsmith/config"
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

// Load reads a textproto file, expands environment variable templates, and unmarshals into msg.
func Load[M proto.Message](path string, msg M) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read configuration file %q: %w", path, err)
	}

	expanded, err := ExpandTemplates(raw)
	if err != nil {
		return fmt.Errorf("configuration template error in %q:\n%w", path, err)
	}

	opts := prototext.UnmarshalOptions{
		DiscardUnknown: false,
	}
	if err := opts.Unmarshal(expanded, msg); err != nil {
		return fmt.Errorf("failed to parse textproto in %q: %w", path, err)
	}

	return nil
}

// LoadAPIServerConfig loads and validates the API server configuration from an explicit file path.
func LoadAPIServerConfig(configPath string) (*config.ApiServerConfig, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			return nil, fmt.Errorf("configuration file path must be explicitly provided (use --config flag or QS_CONFIG env var)")
		}
	}

	var cfg config.ApiServerConfig
	if err := Load(configPath, &cfg); err != nil {
		return nil, err
	}

	applyAPIServerEnvOverrides(&cfg)

	if err := ValidateAPIServerConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadOpsServerConfig loads and validates the internal ops server configuration from an explicit file path.
func LoadOpsServerConfig(configPath string) (*config.OpsServerConfig, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			return nil, fmt.Errorf("configuration file path must be explicitly provided (use --config flag or QS_CONFIG env var)")
		}
	}

	var cfg config.OpsServerConfig
	if err := Load(configPath, &cfg); err != nil {
		return nil, err
	}

	applyOpsServerEnvOverrides(&cfg)

	if err := ValidateOpsServerConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadJobServerConfig loads and validates the job scheduler configuration from an explicit file path.
func LoadJobServerConfig(configPath string) (*config.JobServerConfig, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			return nil, fmt.Errorf("configuration file path must be explicitly provided (use --config flag or QS_CONFIG env var)")
		}
	}

	var cfg config.JobServerConfig
	if err := Load(configPath, &cfg); err != nil {
		return nil, err
	}

	applyJobServerEnvOverrides(&cfg)

	if err := ValidateJobServerConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadWebServerConfig loads and validates the web server configuration from an explicit file path.
func LoadWebServerConfig(configPath string) (*config.WebServerConfig, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			return nil, fmt.Errorf("configuration file path must be explicitly provided (use --config flag or QS_CONFIG env var)")
		}
	}

	var cfg config.WebServerConfig
	if err := Load(configPath, &cfg); err != nil {
		return nil, err
	}

	applyWebServerEnvOverrides(&cfg)

	if err := ValidateWebServerConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadOpsCliConfig loads and validates configuration for the qsops operations CLI.
func LoadOpsCliConfig(configPath, envName string) (*config.OpsCliConfig, error) {
	resolvedPath, err := resolveConfigPath("cli-ops", configPath, envName)
	if err != nil {
		resolvedPath, err = resolveConfigPath("qsops", configPath, envName)
	}

	var cfg config.OpsCliConfig
	if err == nil {
		_ = Load(resolvedPath, &cfg)
	}

	if err := ValidateOpsCliConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadCliConfig loads and validates configuration for the customer CLI.
func LoadCliConfig(configPath, envName string) (*config.CliConfig, error) {
	resolvedPath, err := resolveConfigPath("cli", configPath, envName)
	if err != nil {
		return nil, err
	}

	var cfg config.CliConfig
	if err := Load(resolvedPath, &cfg); err != nil {
		return nil, err
	}

	if err := ValidateCliConfig(&cfg); err != nil {
		return nil, err
	}

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

// FindBinaryConfigFile searches for the textproto config file matching binaryName and envName.
func FindBinaryConfigFile(binaryName, envName string) (string, error) {
	filename := fmt.Sprintf("%s.textproto", strings.ToLower(envName))

	candidates := []string{
		filepath.Join("src", "backend", binaryName, "config", filename),
		filepath.Join("src", "frontend", binaryName, "config", filename),
		filepath.Join("backend", binaryName, "config", filename),
		filepath.Join("frontend", binaryName, "config", filename),
		filepath.Join(binaryName, "config", filename),
		filepath.Join("config", filename),
		filepath.Join("..", "config", filename),
		filepath.Join("..", binaryName, "config", filename),
		filepath.Join("..", "..", binaryName, "config", filename),
		filepath.Join("..", "..", "..", binaryName, "config", filename),
		filepath.Join("..", "src", "backend", binaryName, "config", filename),
		filepath.Join("..", "src", "frontend", binaryName, "config", filename),
		filepath.Join("..", "..", "src", "backend", binaryName, "config", filename),
		filepath.Join("..", "..", "src", "frontend", binaryName, "config", filename),
		filepath.Join("..", "..", "..", "src", "backend", binaryName, "config", filename),
		filepath.Join("..", "..", "..", "src", "frontend", binaryName, "config", filename),
		filepath.Join("/app", "src", "backend", binaryName, "config", filename),
		filepath.Join("/app", "src", "frontend", binaryName, "config", filename),
		filepath.Join("/app", binaryName, "config", filename),
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

	// Search upwards from cwd for repo root containing src/backend/<binaryName>/config/<filename> or src/frontend/<binaryName>/config/<filename>
	if cwd, err := os.Getwd(); err == nil {
		curr := cwd
		for i := 0; i < 6; i++ {
			targetBackend := filepath.Join(curr, "src", "backend", binaryName, "config", filename)
			if info, err := os.Stat(targetBackend); err == nil && !info.IsDir() {
				return targetBackend, nil
			}
			targetFrontend := filepath.Join(curr, "src", "frontend", binaryName, "config", filename)
			if info, err := os.Stat(targetFrontend); err == nil && !info.IsDir() {
				return targetFrontend, nil
			}
			parent := filepath.Dir(curr)
			if parent == curr {
				break
			}
			curr = parent
		}
	}

	return "", fmt.Errorf("unable to locate configuration file %q for binary %q and environment %q", filename, binaryName, envName)
}

func applyAPIServerEnvOverrides(cfg *config.ApiServerConfig) {
	if cfg.Database != nil {
		applyDatabaseEnvOverrides(cfg.Database)
	}
	if cfg.Auth != nil {
		if v := os.Getenv("QS_AUTH_SECRET"); v != "" {
			cfg.Auth.Secret = v
		}
	}
}

func applyOpsServerEnvOverrides(cfg *config.OpsServerConfig) {
	if cfg.Database != nil {
		applyDatabaseEnvOverrides(cfg.Database)
	}
	if cfg.InternalDatabase != nil {
		applyInternalDatabaseEnvOverrides(cfg.InternalDatabase)
	}
	if cfg.Oauth != nil {
		if v := os.Getenv("QS_OAUTH_CLIENT_ID"); v != "" {
			cfg.Oauth.ClientId = v
		}
		if v := os.Getenv("QS_OAUTH_CLIENT_SECRET"); v != "" {
			cfg.Oauth.ClientSecret = v
		}
	}
}

func applyJobServerEnvOverrides(cfg *config.JobServerConfig) {
	if v := os.Getenv("QS_JOB_SERVER_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	}
}

func applyWebServerEnvOverrides(cfg *config.WebServerConfig) {
	if cfg.Database != nil {
		applyDatabaseEnvOverrides(cfg.Database)
	}
}

func applyInternalDatabaseEnvOverrides(db *config.DatabaseConfig) {
	if v := os.Getenv("QS_INTERNAL_DATABASE_URL"); v != "" {
		db.Url = v
	}
}

func applyDatabaseEnvOverrides(db *config.DatabaseConfig) {
	if v := os.Getenv("QS_DATABASE_URL"); v != "" {
		db.Url = v
	}
}

// ValidateAPIServerConfig ensures all configuration required by the API server daemon is present and valid.
func ValidateAPIServerConfig(c *config.ApiServerConfig) error {
	var errs []string

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, test, staging, prod)")
	}

	if c.Server == nil {
		errs = append(errs, "server configuration section is missing")
	} else {
		if c.Server.Port < 1 || c.Server.Port > 65535 {
			errs = append(errs, fmt.Sprintf("server.port must be between 1 and 65535 (got %d)", c.Server.Port))
		}
		if c.Server.ReadTimeout != nil && c.Server.ReadTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.read_timeout must be greater than 0")
		}
		if c.Server.WriteTimeout != nil && c.Server.WriteTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.write_timeout must be greater than 0")
		}
		if c.Server.IdleTimeout != nil && c.Server.IdleTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.idle_timeout must be greater than 0")
		}
		if c.Server.MaxRequestBytes <= 0 {
			errs = append(errs, "server.max_request_bytes must be greater than 0")
		}
	}

	if c.Database == nil {
		errs = append(errs, "database configuration section is missing")
	} else if err := ValidateDatabase(c.Database, c.GetEnvironment()); err != nil {
		errs = append(errs, err.Error())
	}

	if c.Auth == nil {
		errs = append(errs, "auth configuration section is missing")
	} else if err := ValidateAuth(c.Auth, c.GetEnvironment()); err != nil {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return formatErrors("API Server", errs)
	}
	return nil
}

// ValidateWebServerConfig ensures all configuration required by the web server daemon is present and valid.
func ValidateWebServerConfig(c *config.WebServerConfig) error {
	var errs []string

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, test, staging, prod)")
	}

	if c.Server == nil {
		errs = append(errs, "server configuration section is missing")
	} else {
		if c.Server.Port < 1 || c.Server.Port > 65535 {
			errs = append(errs, fmt.Sprintf("server.port must be between 1 and 65535 (got %d)", c.Server.Port))
		}
		if strings.TrimSpace(c.Server.StaticDir) == "" {
			errs = append(errs, "server.static_dir cannot be empty")
		}
		if c.Server.ReadTimeout != nil && c.Server.ReadTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.read_timeout must be greater than 0")
		}
		if c.Server.WriteTimeout != nil && c.Server.WriteTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.write_timeout must be greater than 0")
		}
		if c.Server.IdleTimeout != nil && c.Server.IdleTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.idle_timeout must be greater than 0")
		}
	}

	if strings.TrimSpace(c.ApiUrl) == "" {
		errs = append(errs, "api_url cannot be empty")
	} else {
		u, err := url.Parse(c.ApiUrl)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			errs = append(errs, fmt.Sprintf("api_url must be a valid HTTP or HTTPS URL (got %q)", c.ApiUrl))
		}
	}

	if strings.TrimSpace(c.AppName) == "" {
		errs = append(errs, "app_name cannot be empty")
	}

	if c.Database != nil && c.Database.Url != "" {
		if err := ValidateDatabase(c.Database, c.GetEnvironment()); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return formatErrors("Web Server", errs)
	}
	return nil
}

// ValidateOpsServerConfig ensures all configuration required by the internal ops server is present and valid.
func ValidateOpsServerConfig(c *config.OpsServerConfig) error {
	var errs []string

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, test, staging, prod)")
	}

	if c.Server == nil {
		errs = append(errs, "server configuration section is missing")
	} else {
		if c.Server.Port < 1 || c.Server.Port > 65535 {
			errs = append(errs, fmt.Sprintf("server.port must be between 1 and 65535 (got %d)", c.Server.Port))
		}
		if c.Server.ReadTimeout != nil && c.Server.ReadTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.read_timeout must be greater than 0")
		}
		if c.Server.WriteTimeout != nil && c.Server.WriteTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.write_timeout must be greater than 0")
		}
		if c.Server.IdleTimeout != nil && c.Server.IdleTimeout.AsDuration() <= 0 {
			errs = append(errs, "server.idle_timeout must be greater than 0")
		}
	}

	if c.Database == nil {
		errs = append(errs, "database configuration section is missing")
	} else if err := ValidateDatabase(c.Database, c.GetEnvironment()); err != nil {
		errs = append(errs, "database: "+err.Error())
	}

	if c.InternalDatabase == nil {
		errs = append(errs, "internal_database configuration section is missing")
	} else if err := ValidateDatabase(c.InternalDatabase, c.GetEnvironment()); err != nil {
		errs = append(errs, "internal_database: "+err.Error())
	}

	if len(c.Groups) == 0 {
		errs = append(errs, "at least one IAM group must be defined in groups")
	}

	if len(errs) > 0 {
		return formatErrors("Ops Server", errs)
	}
	return nil
}

// ValidateJobServerConfig ensures all configuration required by the job server is present and valid.
func ValidateJobServerConfig(c *config.JobServerConfig) error {
	var errs []string

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, test, staging, prod)")
	}

	if strings.TrimSpace(c.GetOpsServerUrl()) == "" {
		errs = append(errs, "ops_server_url cannot be empty")
	} else {
		u, err := url.Parse(c.GetOpsServerUrl())
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			errs = append(errs, fmt.Sprintf("ops_server_url must be a valid HTTP or HTTPS URL (got %q)", c.GetOpsServerUrl()))
		}
	}

	if strings.TrimSpace(c.GetAuthToken()) == "" {
		errs = append(errs, "auth_token cannot be empty")
	}

	if len(c.Schedules) == 0 {
		errs = append(errs, "at least one schedule must be defined in schedules")
	}

	if len(errs) > 0 {
		return formatErrors("Job Server", errs)
	}
	return nil
}

// ValidateOpsCliConfig ensures configuration required by qsops is present and valid.
func ValidateOpsCliConfig(c *config.OpsCliConfig) error {
	var errs []string

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, test, staging, prod)")
	}

	if strings.TrimSpace(c.GetOpsServerUrl()) == "" {
		errs = append(errs, "ops_server_url cannot be empty")
	} else {
		u, err := url.Parse(c.GetOpsServerUrl())
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			errs = append(errs, fmt.Sprintf("ops_server_url must be a valid HTTP or HTTPS URL (got %q)", c.GetOpsServerUrl()))
		}
	}

	if c.Timeout != nil && c.Timeout.AsDuration() <= 0 {
		errs = append(errs, "timeout must be greater than 0")
	}

	if len(errs) > 0 {
		return formatErrors("Ops CLI", errs)
	}
	return nil
}

// ValidateCliConfig ensures configuration required by the customer CLI is present and valid.
func ValidateCliConfig(c *config.CliConfig) error {
	var errs []string

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, test, staging, prod)")
	}

	if strings.TrimSpace(c.GetApiUrl()) == "" {
		errs = append(errs, "api_url cannot be empty")
	} else {
		u, err := url.Parse(c.GetApiUrl())
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			errs = append(errs, fmt.Sprintf("api_url must be a valid HTTP or HTTPS URL (got %q)", c.GetApiUrl()))
		}
	}

	if c.Timeout != nil && c.Timeout.AsDuration() <= 0 {
		errs = append(errs, "timeout must be greater than 0")
	}

	if len(errs) > 0 {
		return formatErrors("CLI", errs)
	}
	return nil
}

// ValidateSeeder validates database settings required by the seeder.
func ValidateSeeder(db *config.DatabaseConfig, env string) error {
	if db == nil {
		return formatErrors("Database Seeder", []string{"database configuration is missing"})
	}
	if err := ValidateDatabase(db, env); err != nil {
		return formatErrors("Database Seeder", []string{err.Error()})
	}
	return nil
}

// ValidateDatabase checks that the database configuration is well-formed and valid for the environment.
func ValidateDatabase(db *config.DatabaseConfig, env string) error {
	if strings.TrimSpace(db.GetUrl()) == "" {
		return fmt.Errorf("database.url cannot be empty")
	}
	if strings.TrimSpace(db.GetName()) == "" {
		return fmt.Errorf("database.name cannot be empty")
	}
	parsed, err := url.Parse(db.GetUrl())
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		return fmt.Errorf("database.url must be a valid postgres connection string")
	}

	// Verify database URL path matches configured name
	pathDb := strings.TrimPrefix(parsed.Path, "/")
	if pathDb != "" && pathDb != db.GetName() {
		return fmt.Errorf("database.url path %q does not match configured database name %q", pathDb, db.GetName())
	}

	if db.GetMaxConnections() <= 0 {
		return fmt.Errorf("database.max_connections must be greater than 0 (got %d)", db.GetMaxConnections())
	}
	if db.GetMinConnections() > 0 && db.GetMinConnections() > db.GetMaxConnections() {
		return fmt.Errorf("database.min_connections (%d) cannot exceed max_connections (%d)", db.GetMinConnections(), db.GetMaxConnections())
	}
	if isProduction(env) || isStaging(env) {
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" {
			return fmt.Errorf("database.url cannot point to localhost/127.0.0.1 in %s environment", env)
		}
	}
	return nil
}

// ValidateAuth checks authentication configuration settings.
func ValidateAuth(auth *config.AuthConfig, env string) error {
	var errs []string
	if strings.TrimSpace(auth.GetSecret()) == "" {
		errs = append(errs, "auth.secret cannot be empty")
	}
	if auth.GetAccessTokenTtl() != nil && auth.GetAccessTokenTtl().AsDuration() <= 0 {
		errs = append(errs, "auth.access_token_ttl must be greater than 0")
	}
	if auth.GetRefreshTokenTtl() != nil && auth.GetRefreshTokenTtl().AsDuration() <= 0 {
		errs = append(errs, "auth.refresh_token_ttl must be greater than 0")
	}
	if auth.GetOtpTtl() != nil && auth.GetOtpTtl().AsDuration() <= 0 {
		errs = append(errs, "auth.otp_ttl must be greater than 0")
	}
	if auth.GetMagicLinkTtl() != nil && auth.GetMagicLinkTtl().AsDuration() <= 0 {
		errs = append(errs, "auth.magic_link_ttl must be greater than 0")
	}
	if auth.GetEmailCooldown() != nil && auth.GetEmailCooldown().AsDuration() <= 0 {
		errs = append(errs, "auth.email_cooldown must be greater than 0")
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
