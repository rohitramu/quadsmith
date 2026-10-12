package config

import "strings"

// DefaultEnv is the default environment baked into frontend binaries.
// Can be changed at build time via -ldflags "-X quadsmith/frontend/config.DefaultEnv=sandbox"
var DefaultEnv = "prod"

// BakedConfigs holds the compile-time baked presets for all environments.
var BakedConfigs = map[string]Config{
	"sandbox": {
		Environment: "sandbox",
		UI: UIConfig{
			APIURL:  "http://localhost:8080",
			AppName: "Quadsmith (Sandbox)",
		},
		CLI: CLIConfig{
			APIURL: "http://localhost:8080",
		},
		Ops: OpsConfig{
			WorkerURL: "http://localhost:8081",
		},
	},
	"test": {
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
	},
	"staging": {
		Environment: "staging",
		UI: UIConfig{
			APIURL:  "https://staging.quadsmith.com",
			AppName: "Quadsmith (Staging)",
		},
		CLI: CLIConfig{
			APIURL: "https://staging.quadsmith.com",
		},
		Ops: OpsConfig{
			WorkerURL: "https://staging.quadsmith.com",
		},
	},
	"prod": {
		Environment: "prod",
		UI: UIConfig{
			APIURL:  "https://quadsmith.com",
			AppName: "Quadsmith",
		},
		CLI: CLIConfig{
			APIURL: "https://quadsmith.com",
		},
		Ops: OpsConfig{
			WorkerURL: "https://quadsmith.com",
		},
	},
}

// GetBaked retrieves the compile-time baked configuration for an environment.
func GetBaked(envName string) (Config, bool) {
	lower := strings.ToLower(strings.TrimSpace(envName))
	if lower == "production" {
		lower = "prod"
	}
	cfg, ok := BakedConfigs[lower]
	return cfg, ok
}
