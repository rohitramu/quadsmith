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
	"google.golang.org/protobuf/reflect/protoreflect"

	"google.golang.org/protobuf/types/known/durationpb"
	configpb "quadsmith/shared/gen/quadsmith/config"
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
func LoadAPIServerConfig(configPath string) (*configpb.ApiServerConfig, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			return nil, fmt.Errorf("configuration file path must be explicitly provided (use --config flag or QS_CONFIG env var)")
		}
	}

	var cfg configpb.ApiServerConfig
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
func LoadOpsServerConfig(configPath string) (*configpb.OpsServerConfig, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			return nil, fmt.Errorf("configuration file path must be explicitly provided (use --config flag or QS_CONFIG env var)")
		}
	}

	var cfg configpb.OpsServerConfig
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
func LoadJobServerConfig(configPath string) (*configpb.JobServerConfig, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			return nil, fmt.Errorf("configuration file path must be explicitly provided (use --config flag or QS_CONFIG env var)")
		}
	}

	var cfg configpb.JobServerConfig
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
func LoadWebServerConfig(configPath string) (*configpb.WebServerConfig, error) {
	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			return nil, fmt.Errorf("configuration file path must be explicitly provided (use --config flag or QS_CONFIG env var)")
		}
	}

	var cfg configpb.WebServerConfig
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
func LoadOpsCliConfig(configPath, envName string) (*configpb.OpsCliConfig, error) {
	resolvedPath, err := resolveConfigPath("cli-ops", configPath, envName)
	if err != nil {
		resolvedPath, err = resolveConfigPath("qsops", configPath, envName)
		if err != nil {
			return nil, err
		}
	}

	var cfg configpb.OpsCliConfig
	if err := Load(resolvedPath, &cfg); err != nil {
		return nil, err
	}

	applyOpsCliEnvOverrides(&cfg)

	if err := ValidateOpsCliConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadCliConfig loads and validates configuration for the customer CLI.
func LoadCliConfig(configPath, envName string) (*configpb.CliConfig, error) {
	resolvedPath, err := resolveConfigPath("cli", configPath, envName)
	if err != nil {
		return nil, err
	}

	var cfg configpb.CliConfig
	if err := Load(resolvedPath, &cfg); err != nil {
		return nil, err
	}

	applyCliEnvOverrides(&cfg)

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
		filepath.Join("src", "go", binaryName, "config", filename),
		filepath.Join("go", binaryName, "config", filename),
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
		filepath.Join("..", "src", "go", binaryName, "config", filename),
		filepath.Join("..", "src", "backend", binaryName, "config", filename),
		filepath.Join("..", "src", "frontend", binaryName, "config", filename),
		filepath.Join("..", "..", "src", "go", binaryName, "config", filename),
		filepath.Join("..", "..", "src", "backend", binaryName, "config", filename),
		filepath.Join("..", "..", "src", "frontend", binaryName, "config", filename),
		filepath.Join("..", "..", "..", "src", "go", binaryName, "config", filename),
		filepath.Join("..", "..", "..", "src", "backend", binaryName, "config", filename),
		filepath.Join("..", "..", "..", "src", "frontend", binaryName, "config", filename),
		filepath.Join("/app", "src", "go", binaryName, "config", filename),
		filepath.Join("/app", "src", "backend", binaryName, "config", filename),
		filepath.Join("/app", "src", "frontend", binaryName, "config", filename),
		filepath.Join("/app", binaryName, "config", filename),
		filepath.Join("/app", "config", filename),
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

	// Search upwards from cwd for repo root containing src/go/<binaryName>/config/<filename>
	if cwd, err := os.Getwd(); err == nil {
		curr := cwd
		for i := 0; i < 6; i++ {
			targetGo := filepath.Join(curr, "src", "go", binaryName, "config", filename)
			if info, err := os.Stat(targetGo); err == nil && !info.IsDir() {
				return targetGo, nil
			}
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

func applyAPIServerEnvOverrides(cfg *configpb.ApiServerConfig) {
	if cfg.Database != nil {
		applyDatabaseEnvOverrides(cfg.Database)
	}
	if cfg.Auth != nil {
		if v := os.Getenv("QS_AUTH_SECRET"); v != "" {
			cfg.Auth.Secret = &v
		}
		if cfg.Auth.Email != nil && cfg.Auth.Email.GetSmtp() != nil {
			if v := os.Getenv("QS_SMTP_PASSWORD"); v != "" {
				cfg.Auth.Email.GetSmtp().Password = &v
			}
		}
		if cfg.Auth.Oauth != nil {
			if v := os.Getenv("QS_WEB_BASE_URL"); v != "" {
				cfg.Auth.Oauth.WebBaseUrl = &v
			}
		}
	}
}

func applyOpsServerEnvOverrides(cfg *configpb.OpsServerConfig) {
	if cfg.Database != nil {
		applyDatabaseEnvOverrides(cfg.Database)
	}
	if cfg.InternalDatabase != nil {
		applyInternalDatabaseEnvOverrides(cfg.InternalDatabase)
	}
	if cfg.Oauth != nil {
		if v := os.Getenv("QS_OAUTH_CLIENT_ID"); v != "" {
			cfg.Oauth.ClientId = &v
		}
		if v := os.Getenv("QS_OAUTH_CLIENT_SECRET"); v != "" {
			cfg.Oauth.ClientSecret = &v
		}
	}
}

func applyJobServerEnvOverrides(cfg *configpb.JobServerConfig) {
	if v := os.Getenv("QS_JOB_SERVER_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = &v
	}
	if v := os.Getenv("QS_OPS_SERVER_URL"); v != "" {
		cfg.OpsServerUrl = &v
	}
}

func applyWebServerEnvOverrides(cfg *configpb.WebServerConfig) {
	if v := os.Getenv("QS_API_URL"); v != "" {
		cfg.ApiUrl = &v
	}
}

func applyOpsCliEnvOverrides(cfg *configpb.OpsCliConfig) {
	if v := os.Getenv("QS_OPS_SERVER_URL"); v != "" {
		cfg.OpsServerUrl = &v
	}
}

func applyCliEnvOverrides(cfg *configpb.CliConfig) {
	if v := os.Getenv("QS_SERVER_URL"); v != "" {
		cfg.ApiUrl = &v
	}
}

func applyInternalDatabaseEnvOverrides(db *configpb.DatabaseConfig) {
	if v := os.Getenv("QS_INTERNAL_DATABASE_URL"); v != "" {
		db.Url = &v
	}
}

func applyDatabaseEnvOverrides(db *configpb.DatabaseConfig) {
	if v := os.Getenv("QS_DATABASE_URL"); v != "" {
		db.Url = &v
	}
}

// ValidateNoUnsetFields recursively inspects msg using Protobuf reflection.
// It verifies that:
// 1. Every non-oneof scalar field (which has presence enabled via 'optional') is explicitly set.
// 2. Every non-oneof singular message field is explicitly set, and its sub-fields are recursively validated.
// 3. For any oneof group, exactly one branch is set, and if that branch is a message, it is recursively validated.
// 4. Repeated message fields: every element in the list is recursively validated.
// 5. Special handling for Google Well-Known Types (like google.protobuf.Duration): verified to be non-nil.
func ValidateNoUnsetFields(msg proto.Message) error {
	if msg == nil {
		return fmt.Errorf("configuration message is nil")
	}
	var missing []string
	validateMessageFields(msg.ProtoReflect(), string(msg.ProtoReflect().Descriptor().Name()), &missing)
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration field(s):\n  - %s", strings.Join(missing, "\n  - "))
	}
	return nil
}

func validateMessageFields(m protoreflect.Message, path string, missing *[]string) {
	if !m.IsValid() {
		*missing = append(*missing, fmt.Sprintf("%s is invalid/nil", path))
		return
	}

	md := m.Descriptor()
	fields := md.Fields()

	checkedOneofs := make(map[protoreflect.Name]bool)

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)

		// Check oneof groups
		if oo := fd.ContainingOneof(); oo != nil {
			if !checkedOneofs[oo.Name()] {
				checkedOneofs[oo.Name()] = true
				which := m.WhichOneof(oo)
				if which == nil {
					*missing = append(*missing, fmt.Sprintf("%s: oneof %q must have an active branch set", path, oo.Name()))
				} else {
					val := m.Get(which)
					if which.Message() != nil && !which.IsList() && !which.IsMap() {
						if which.Message().FullName() != "google.protobuf.Duration" && which.Message().FullName() != "google.protobuf.Timestamp" {
							validateMessageFields(val.Message(), fmt.Sprintf("%s.%s", path, which.Name()), missing)
						}
					}
				}
			}
			continue
		}

		// Non-oneof repeated fields: validate each element if it's a message
		if fd.IsList() {
			if fd.Message() != nil {
				list := m.Get(fd).List()
				for j := 0; j < list.Len(); j++ {
					validateMessageFields(list.Get(j).Message(), fmt.Sprintf("%s.%s[%d]", path, fd.Name(), j), missing)
				}
			}
			continue
		}

		// Non-oneof singular field: must be explicitly set
		if !m.Has(fd) {
			*missing = append(*missing, fmt.Sprintf("%s.%s", path, fd.Name()))
			continue
		}

		// If it's a sub-message, recursively validate
		if fd.Message() != nil && !fd.IsMap() {
			if fd.Message().FullName() != "google.protobuf.Duration" && fd.Message().FullName() != "google.protobuf.Timestamp" {
				validateMessageFields(m.Get(fd).Message(), fmt.Sprintf("%s.%s", path, fd.Name()), missing)
			}
		}
	}
}

// ValidateAPIServerConfig ensures all configuration required by the API server daemon is present and valid.
func ValidateAPIServerConfig(c *configpb.ApiServerConfig) error {
	var errs []string

	if err := ValidateNoUnsetFields(c); err != nil {
		errs = append(errs, err.Error())
	}

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, staging, prod)")
	}

	if c.Server == nil {
		errs = append(errs, "server configuration section is missing")
	} else {
		if c.Server.GetPort() < 1 || c.Server.GetPort() > 65535 {
			errs = append(errs, fmt.Sprintf("server.port must be between 1 and 65535 (got %d)", c.Server.GetPort()))
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
		if c.Server.GetMaxRequestBytes() <= 0 {
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
func ValidateWebServerConfig(c *configpb.WebServerConfig) error {
	var errs []string

	if err := ValidateNoUnsetFields(c); err != nil {
		errs = append(errs, err.Error())
	}

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, staging, prod)")
	}

	if c.Server == nil {
		errs = append(errs, "server configuration section is missing")
	} else {
		if c.Server.GetPort() < 1 || c.Server.GetPort() > 65535 {
			errs = append(errs, fmt.Sprintf("server.port must be between 1 and 65535 (got %d)", c.Server.GetPort()))
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
		if c.Server.GetMaxRequestBytes() <= 0 {
			errs = append(errs, "server.max_request_bytes must be greater than 0")
		}
	}

	if strings.TrimSpace(c.GetStaticDir()) == "" {
		errs = append(errs, "static_dir cannot be empty")
	}

	if strings.TrimSpace(c.GetApiUrl()) == "" {
		errs = append(errs, "api_url cannot be empty")
	} else {
		u, err := url.Parse(c.GetApiUrl())
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			errs = append(errs, fmt.Sprintf("api_url must be a valid HTTP or HTTPS URL (got %q)", c.GetApiUrl()))
		}
	}

	if len(errs) > 0 {
		return formatErrors("Web Server", errs)
	}
	return nil
}

// ValidateOpsServerConfig ensures all configuration required by the internal ops server is present and valid.
func ValidateOpsServerConfig(c *configpb.OpsServerConfig) error {
	var errs []string

	if err := ValidateNoUnsetFields(c); err != nil {
		errs = append(errs, err.Error())
	}

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, staging, prod)")
	}

	if c.Server == nil {
		errs = append(errs, "server configuration section is missing")
	} else {
		if c.Server.GetPort() < 1 || c.Server.GetPort() > 65535 {
			errs = append(errs, fmt.Sprintf("server.port must be between 1 and 65535 (got %d)", c.Server.GetPort()))
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
		if c.Server.GetMaxRequestBytes() <= 0 {
			errs = append(errs, "server.max_request_bytes must be greater than 0")
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
func ValidateJobServerConfig(c *configpb.JobServerConfig) error {
	var errs []string

	if err := ValidateNoUnsetFields(c); err != nil {
		errs = append(errs, err.Error())
	}

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, staging, prod)")
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

	// Helper to validate a task's schedule branch
	validateTaskSchedule := func(name string, hasCron bool, cronStr string, hasInterval bool, dur *durationpb.Duration) {
		if hasCron {
			if strings.TrimSpace(cronStr) == "" {
				errs = append(errs, fmt.Sprintf("%s.cron cannot be empty", name))
			} else if err := ValidateCronExpression(cronStr); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			}
		} else if hasInterval {
			if dur == nil {
				errs = append(errs, fmt.Sprintf("%s.interval cannot be nil", name))
			} else if _, err := DurationPbToCron(dur); err != nil {
				errs = append(errs, fmt.Sprintf("%s: invalid interval: %v", name, err))
			}
		} else {
			errs = append(errs, fmt.Sprintf("%s: schedule (cron or interval) must be set", name))
		}
	}

	if s := c.GetCleanupExpiredOtps(); s != nil {
		_, hasCron := s.GetSchedule().(*configpb.CleanupExpiredOtpsSchedule_Cron)
		_, hasInterval := s.GetSchedule().(*configpb.CleanupExpiredOtpsSchedule_Interval)
		validateTaskSchedule("cleanup_expired_otps", hasCron, s.GetCron(), hasInterval, s.GetInterval())
	}
	if s := c.GetCleanupExpiredTokens(); s != nil {
		_, hasCron := s.GetSchedule().(*configpb.CleanupExpiredTokensSchedule_Cron)
		_, hasInterval := s.GetSchedule().(*configpb.CleanupExpiredTokensSchedule_Interval)
		validateTaskSchedule("cleanup_expired_tokens", hasCron, s.GetCron(), hasInterval, s.GetInterval())
	}
	if s := c.GetCleanupOrphanedBlobs(); s != nil {
		_, hasCron := s.GetSchedule().(*configpb.CleanupOrphanedBlobsSchedule_Cron)
		_, hasInterval := s.GetSchedule().(*configpb.CleanupOrphanedBlobsSchedule_Interval)
		validateTaskSchedule("cleanup_orphaned_blobs", hasCron, s.GetCron(), hasInterval, s.GetInterval())
	}
	if s := c.GetPurgeSoftDeletedAccounts(); s != nil {
		_, hasCron := s.GetSchedule().(*configpb.PurgeSoftDeletedAccountsSchedule_Cron)
		_, hasInterval := s.GetSchedule().(*configpb.PurgeSoftDeletedAccountsSchedule_Interval)
		validateTaskSchedule("purge_soft_deleted_accounts", hasCron, s.GetCron(), hasInterval, s.GetInterval())
	}
	if s := c.GetSendUsernameReleaseWarnings(); s != nil {
		_, hasCron := s.GetSchedule().(*configpb.SendUsernameReleaseWarningsSchedule_Cron)
		_, hasInterval := s.GetSchedule().(*configpb.SendUsernameReleaseWarningsSchedule_Interval)
		validateTaskSchedule("send_username_release_warnings", hasCron, s.GetCron(), hasInterval, s.GetInterval())
	}

	if len(errs) > 0 {
		return formatErrors("Job Server", errs)
	}
	return nil
}

// ValidateOpsCliConfig ensures configuration required by qsops is present and valid.
func ValidateOpsCliConfig(c *configpb.OpsCliConfig) error {
	var errs []string

	if err := ValidateNoUnsetFields(c); err != nil {
		errs = append(errs, err.Error())
	}

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, staging, prod)")
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
func ValidateCliConfig(c *configpb.CliConfig) error {
	var errs []string

	if err := ValidateNoUnsetFields(c); err != nil {
		errs = append(errs, err.Error())
	}

	if strings.TrimSpace(c.GetEnvironment()) == "" {
		errs = append(errs, "environment must be explicitly specified (e.g. sandbox, staging, prod)")
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
func ValidateSeeder(db *configpb.DatabaseConfig, env string) error {
	if db == nil {
		return formatErrors("Database Seeder", []string{"database configuration is missing"})
	}
	if err := ValidateDatabase(db, env); err != nil {
		return formatErrors("Database Seeder", []string{err.Error()})
	}
	return nil
}

// ValidateDatabase checks that the database configuration is well-formed and valid for the environment.
func ValidateDatabase(db *configpb.DatabaseConfig, env string) error {
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
func ValidateAuth(auth *configpb.AuthConfig, env string) error {
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
	if auth.GetEmail() != nil {
		if auth.GetEmail().GetSmtp() == nil && auth.GetEmail().GetConsole() == nil {
			errs = append(errs, "auth.email must specify either 'console' or 'smtp'")
		}
		if smtp := auth.GetEmail().GetSmtp(); smtp != nil {
			if strings.TrimSpace(smtp.GetHost()) == "" {
				errs = append(errs, "auth.email.smtp.host cannot be empty")
			}
			if smtp.GetPort() <= 0 || smtp.GetPort() > 65535 {
				errs = append(errs, "auth.email.smtp.port must be between 1 and 65535")
			}
			if strings.TrimSpace(smtp.GetFromAddress()) == "" {
				errs = append(errs, "auth.email.smtp.from_address cannot be empty")
			}
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
