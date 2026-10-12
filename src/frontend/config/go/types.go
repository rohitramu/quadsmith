package config

// Config represents the frontend runtime configuration.
type Config struct {
	Environment string    `json:"environment" yaml:"environment"`
	UI          UIConfig  `json:"ui" yaml:"ui"`
	CLI         CLIConfig `json:"cli" yaml:"cli"`
	Ops         OpsConfig `json:"ops" yaml:"ops"`
}

// UIConfig contains public settings for the Web UI.
type UIConfig struct {
	APIURL  string `json:"api_url" yaml:"api_url"`
	AppName string `json:"app_name" yaml:"app_name"`
}

// CLIConfig contains connection settings for the end-user CLI (qs).
type CLIConfig struct {
	APIURL string `json:"api_url" yaml:"api_url"`
}

// OpsConfig contains connection settings for the operational CLI (qsops).
type OpsConfig struct {
	WorkerURL string `json:"worker_url" yaml:"worker_url"`
}
