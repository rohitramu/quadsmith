package config

import "testing"

// Option allows customizing a test frontend configuration.
type Option func(*Config)

func WithCLIAPIURL(apiURL string) Option {
	return func(c *Config) {
		c.CLI.APIURL = apiURL
	}
}

func WithOpsWorkerURL(workerURL string) Option {
	return func(c *Config) {
		c.Ops.WorkerURL = workerURL
	}
}

// NewTestFrontendConfig creates an isolated test frontend configuration.
func NewTestFrontendConfig(t testing.TB, opts ...Option) *Config {
	t.Helper()
	cfg := &Config{
		Environment: "test",
		UI: UIConfig{
			APIURL:  "http://127.0.0.1:8080",
			AppName: "Quadsmith (Test)",
		},
		CLI: CLIConfig{
			APIURL: "http://127.0.0.1:8080",
		},
		Ops: OpsConfig{
			WorkerURL: "http://127.0.0.1:8081",
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}
