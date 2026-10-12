package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Load resolves the frontend configuration for the specified environment.
// If envName is empty, it checks QS_ENV and falls back to DefaultEnv ("prod").
// It first attempts to use compile-time baked configurations, then applies any environment variable overrides.
func Load(envName string) (*Config, error) {
	if envName == "" {
		envName = os.Getenv("QS_ENV")
	}
	if envName == "" {
		envName = DefaultEnv
	}

	var cfg Config
	if baked, ok := GetBaked(envName); ok {
		cfg = baked
	} else {
		// Attempt to load from YAML file if a custom environment is requested
		filePath, err := findFrontendConfigFile(envName)
		if err != nil {
			return nil, fmt.Errorf("unknown environment %q (valid: sandbox, test, staging, prod): %w", envName, err)
		}
		loaded, err := LoadFromFile(filePath)
		if err != nil {
			return nil, err
		}
		cfg = *loaded
	}

	applyEnvOverrides(&cfg)
	return &cfg, nil
}

// LoadFromFile reads and parses a frontend YAML configuration file.
func LoadFromFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read frontend config file %q: %w", filePath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %q: %w", filePath, err)
	}

	if cfg.Environment == "" {
		base := filepath.Base(filePath)
		ext := filepath.Ext(base)
		cfg.Environment = strings.TrimSuffix(base, ext)
	}

	applyEnvOverrides(&cfg)
	return &cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("QS_CLI_API_URL"); v != "" {
		cfg.CLI.APIURL = v
	} else if v := os.Getenv("QS_API_URL"); v != "" {
		cfg.CLI.APIURL = v
	}
	if v := os.Getenv("QS_OPS_WORKER_URL"); v != "" {
		cfg.Ops.WorkerURL = v
	} else if v := os.Getenv("QS_WORKER_URL"); v != "" {
		cfg.Ops.WorkerURL = v
	}
	if v := os.Getenv("QS_UI_API_URL"); v != "" {
		cfg.UI.APIURL = v
	}
	if v := os.Getenv("QS_UI_APP_NAME"); v != "" {
		cfg.UI.AppName = v
	}
}

// ValidateCLI ensures configuration needed for the end-user CLI (qs) is valid.
func (c *Config) ValidateCLI() error {
	if strings.TrimSpace(c.CLI.APIURL) == "" {
		return fmt.Errorf("FATAL: CLI configuration validation failed: cli.api_url cannot be empty")
	}
	u, err := url.Parse(c.CLI.APIURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("FATAL: CLI configuration validation failed: cli.api_url must be a valid HTTP or HTTPS URL (got %q)", c.CLI.APIURL)
	}
	return nil
}

// ValidateOps ensures configuration needed for the ops CLI (qsops) is valid.
func (c *Config) ValidateOps() error {
	if strings.TrimSpace(c.Ops.WorkerURL) == "" {
		return fmt.Errorf("FATAL: Ops configuration validation failed: ops.worker_url cannot be empty")
	}
	u, err := url.Parse(c.Ops.WorkerURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("FATAL: Ops configuration validation failed: ops.worker_url must be a valid HTTP or HTTPS URL (got %q)", c.Ops.WorkerURL)
	}
	return nil
}

func findFrontendConfigFile(envName string) (string, error) {
	filename := fmt.Sprintf("%s.yaml", strings.ToLower(envName))
	candidates := []string{
		filepath.Join("src", "frontend", "config", filename),
		filepath.Join("frontend", "config", filename),
		filepath.Join("..", "src", "frontend", "config", filename),
		filepath.Join("..", "config", filename),
		filepath.Join("..", "..", "src", "frontend", "config", filename),
		filepath.Join("/app", "src", "frontend", "config", filename),
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return filepath.Abs(c)
		}
	}

	return "", fmt.Errorf("file %q not found", filename)
}
