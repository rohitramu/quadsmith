package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"

	"quadsmith/shared/pkg/config"

	pb "quadsmith/shared/gen/quadsmith/config"
)

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)(?::-([^}]*))?\}`)

type configFileCheck struct {
	path      string
	component string
	env       string
}

func main() {
	var checks []configFileCheck
	components := []struct {
		relDir    string
		component string
	}{
		{"src/go/server-api/config", "server-api"},
		{"src/go/server-ops-api/config", "server-ops-api"},
		{"src/go/server-ops-web/config", "server-ops-web"},
		{"src/go/server-cron/config", "server-cron"},
		{"src/go/server-web/config", "server-web"},
		{"src/go/cli/config", "cli"},
		{"src/go/cli-ops/config", "cli-ops"},
	}

	environments := []string{"sandbox", "staging", "prod"}

	for _, c := range components {
		for _, env := range environments {
			p := filepath.Join(c.relDir, fmt.Sprintf("%s.textproto", env))
			checks = append(checks, configFileCheck{
				path:      p,
				component: c.component,
				env:       env,
			})
		}
	}

	for _, env := range []string{"staging", "prod"} {
		checks = append(checks, configFileCheck{
			path:      filepath.Join("deploy/config", fmt.Sprintf("%s.textproto", env)),
			component: "deploy",
			env:       env,
		})
	}

	var errors []string
	verifiedCount := 0

	for _, check := range checks {
		raw, err := os.ReadFile(check.path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to read file: %v", check.path, err))
			continue
		}

		if err := validateConfigFile(check, raw); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", check.path, err))
			continue
		}
		verifiedCount++
	}

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "❌ Configuration Validation Failed (%d error(s)):\n", len(errors))
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  • %s\n", e)
		}
		os.Exit(1)
	}

	fmt.Printf("✓ Validated all %d configuration files across %d applications and deploy topology (all fields explicitly declared, zero non-secret environment variables)\n",
		verifiedCount, len(components))
}

func validateConfigFile(check configFileCheck, raw []byte) error {
	// First, check raw text for unauthorized environment variable placeholders.
	// Only secret fields are permitted to use ${...}.
	lines := strings.Split(string(raw), "\n")
	for lineIdx, line := range lines {
		if envVarPattern.MatchString(line) {
			trimmed := strings.TrimSpace(line)
			if !isAllowedSecretPlaceholder(check.component, trimmed) {
				return fmt.Errorf("line %d: non-secret field contains environment variable placeholder %q (only secrets may be set via environment variable)",
					lineIdx+1, trimmed)
			}
		}
	}

	// Expand templates with dummy values for validation if needed
	expanded, err := expandTemplatesForCheck(raw)
	if err != nil {
		return fmt.Errorf("template expansion failed: %w", err)
	}

	opts := prototext.UnmarshalOptions{DiscardUnknown: false}

	switch check.component {
	case "server-api":
		var cfg pb.ApiServerConfig
		if err := opts.Unmarshal(expanded, &cfg); err != nil {
			return fmt.Errorf("failed to parse ApiServerConfig: %w", err)
		}
		if cfg.GetEnvironment() != check.env {
			return fmt.Errorf("environment field %q does not match filename %q", cfg.GetEnvironment(), check.env)
		}
		if err := config.ValidateAPIServerConfig(&cfg); err != nil {
			return err
		}
		if cfg.Server == nil || cfg.Server.IdleTimeout == nil || cfg.Server.GetMaxRequestBytes() <= 0 {
			return fmt.Errorf("server timeouts and limits must be explicitly specified")
		}
		if cfg.Database == nil || cfg.Database.GetMinConnections() <= 0 || cfg.Database.GetMaxConnections() <= 0 {
			return fmt.Errorf("database connection limits must be explicitly specified")
		}

	case "server-ops-api":
		var cfg pb.OpsApiServerConfig
		if err := opts.Unmarshal(expanded, &cfg); err != nil {
			return fmt.Errorf("failed to parse OpsApiServerConfig: %w", err)
		}
		if cfg.GetEnvironment() != check.env {
			return fmt.Errorf("environment field %q does not match filename %q", cfg.GetEnvironment(), check.env)
		}
		if err := config.ValidateOpsApiServerConfig(&cfg); err != nil {
			return err
		}
		if cfg.Server == nil || cfg.Server.IdleTimeout == nil || cfg.Server.GetMaxRequestBytes() <= 0 {
			return fmt.Errorf("server timeouts and limits must be explicitly specified")
		}
		if cfg.Database == nil || cfg.Database.GetMinConnections() <= 0 || cfg.Database.GetMaxConnections() <= 0 {
			return fmt.Errorf("database connection limits must be explicitly specified")
		}
		if cfg.InternalDatabase == nil || cfg.InternalDatabase.GetMinConnections() <= 0 || cfg.InternalDatabase.GetMaxConnections() <= 0 {
			return fmt.Errorf("internal database connection limits must be explicitly specified")
		}

	case "server-ops-web":
		var cfg pb.OpsWebServerConfig
		if err := opts.Unmarshal(expanded, &cfg); err != nil {
			return fmt.Errorf("failed to parse OpsWebServerConfig: %w", err)
		}
		if cfg.GetEnvironment() != check.env {
			return fmt.Errorf("environment field %q does not match filename %q", cfg.GetEnvironment(), check.env)
		}
		if err := config.ValidateOpsWebServerConfig(&cfg); err != nil {
			return err
		}
		if cfg.Server == nil || cfg.Server.IdleTimeout == nil || cfg.Server.GetMaxRequestBytes() <= 0 {
			return fmt.Errorf("server timeouts and limits must be explicitly specified")
		}
		if strings.TrimSpace(cfg.GetStaticDir()) == "" {
			return fmt.Errorf("static_dir must be explicitly specified")
		}

	case "server-cron":
		var cfg pb.CronServerConfig
		if err := opts.Unmarshal(expanded, &cfg); err != nil {
			return fmt.Errorf("failed to parse CronServerConfig: %w", err)
		}
		if cfg.GetEnvironment() != check.env {
			return fmt.Errorf("environment field %q does not match filename %q", cfg.GetEnvironment(), check.env)
		}
		if err := config.ValidateCronServerConfig(&cfg); err != nil {
			return err
		}
		if cfg.Server == nil || cfg.Server.IdleTimeout == nil || cfg.Server.GetMaxRequestBytes() <= 0 {
			return fmt.Errorf("server timeouts and limits must be explicitly specified")
		}

	case "server-web":
		var cfg pb.WebServerConfig
		if err := opts.Unmarshal(expanded, &cfg); err != nil {
			return fmt.Errorf("failed to parse WebServerConfig: %w", err)
		}
		if cfg.GetEnvironment() != check.env {
			return fmt.Errorf("environment field %q does not match filename %q", cfg.GetEnvironment(), check.env)
		}
		if err := config.ValidateWebServerConfig(&cfg); err != nil {
			return err
		}
		if cfg.Server == nil || cfg.Server.IdleTimeout == nil || cfg.Server.GetMaxRequestBytes() <= 0 {
			return fmt.Errorf("server timeouts and limits must be explicitly specified")
		}
		if strings.TrimSpace(cfg.GetStaticDir()) == "" {
			return fmt.Errorf("static_dir must be explicitly specified")
		}

	case "cli":
		var cfg pb.CliConfig
		if err := opts.Unmarshal(expanded, &cfg); err != nil {
			return fmt.Errorf("failed to parse CliConfig: %w", err)
		}
		if cfg.GetEnvironment() != check.env {
			return fmt.Errorf("environment field %q does not match filename %q", cfg.GetEnvironment(), check.env)
		}
		if err := config.ValidateCliConfig(&cfg); err != nil {
			return err
		}
		if cfg.Timeout == nil || cfg.Timeout.AsDuration() <= 0 {
			return fmt.Errorf("timeout must be explicitly specified")
		}

	case "cli-ops":
		var cfg pb.OpsCliConfig
		if err := opts.Unmarshal(expanded, &cfg); err != nil {
			return fmt.Errorf("failed to parse OpsCliConfig: %w", err)
		}
		if cfg.GetEnvironment() != check.env {
			return fmt.Errorf("environment field %q does not match filename %q", cfg.GetEnvironment(), check.env)
		}
		if err := config.ValidateOpsCliConfig(&cfg); err != nil {
			return err
		}
		if cfg.Timeout == nil || cfg.Timeout.AsDuration() <= 0 {
			return fmt.Errorf("timeout must be explicitly specified")
		}

	case "deploy":
		var cfg pb.DeployConfig
		if err := opts.Unmarshal(expanded, &cfg); err != nil {
			return fmt.Errorf("failed to parse DeployConfig: %w", err)
		}
		if cfg.GetEnvironment() != check.env {
			return fmt.Errorf("environment field %q does not match filename %q", cfg.GetEnvironment(), check.env)
		}
		if err := config.ValidateDeployConfig(&cfg); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown component %q", check.component)
	}

	return nil
}

func isAllowedSecretPlaceholder(component, line string) bool {
	// Secret fields that may use environment variables:
	// - database.url (contains password / credentials)
	// - auth.secret
	// - job_server.auth_token
	// - oauth.client_id, oauth.client_secret
	if strings.HasPrefix(line, "url:") {
		return true
	}
	if strings.HasPrefix(line, "secret:") {
		return true
	}
	if strings.HasPrefix(line, "auth_token:") {
		return true
	}
	if strings.HasPrefix(line, "client_id:") || strings.HasPrefix(line, "client_secret:") {
		return true
	}
	if strings.HasPrefix(line, "password:") {
		return true
	}
	return false
}

func expandTemplatesForCheck(content []byte) ([]byte, error) {
	// Expand placeholders for schema validation; provide sensible dummy values if unset
	expanded := envVarPattern.ReplaceAllFunc(content, func(match []byte) []byte {
		submatches := envVarPattern.FindSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		varName := string(submatches[1])
		hasDefault := len(submatches) >= 3 && submatches[2] != nil
		if hasDefault {
			return submatches[2]
		}
		if val, found := os.LookupEnv(varName); found && val != "" {
			return []byte(val)
		}
		// Dummy values for required secrets during check
		switch {
		case strings.Contains(varName, "INTERNAL_DATABASE_URL"):
			return []byte("postgres://postgres:postgres@db.quadsmith.internal:5432/quadsmith_ops")
		case strings.Contains(varName, "DATABASE_URL"):
			return []byte("postgres://postgres:postgres@db.quadsmith.internal:5432/quadsmith_api")
		case strings.Contains(varName, "AUTH_SECRET"):
			return []byte("dummy-secret-key-for-validation-32bytes!")
		case strings.Contains(varName, "AUTH_TOKEN"):
			return []byte("dummy-auth-token-for-validation")
		case strings.Contains(varName, "CLIENT_ID"):
			return []byte("dummy-client-id")
		case strings.Contains(varName, "CLIENT_SECRET"):
			return []byte("dummy-client-secret")
		case strings.Contains(varName, "SMTP_PASSWORD") || strings.Contains(varName, "SMTP_PASS"):
			return []byte("dummy-smtp-password")
		default:
			return []byte("dummy-val")
		}
	})
	return expanded, nil
}
