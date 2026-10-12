package config

import (
	"path/filepath"
	"testing"
)

func TestLoadBakedConfigs(t *testing.T) {
	envs := []string{"sandbox", "test", "staging", "prod"}
	for _, env := range envs {
		cfg, err := Load(env)
		if err != nil {
			t.Fatalf("failed to load baked config for %q: %v", env, err)
		}
		if err := cfg.ValidateCLI(); err != nil {
			t.Errorf("CLI validation failed for %q: %v", env, err)
		}
		if err := cfg.ValidateOps(); err != nil {
			t.Errorf("Ops validation failed for %q: %v", env, err)
		}
	}

	// Verify prod defaults to quadsmith.com
	prodCfg, _ := Load("prod")
	if prodCfg.CLI.APIURL != "https://quadsmith.com" {
		t.Errorf("expected prod CLI APIURL https://quadsmith.com, got %q", prodCfg.CLI.APIURL)
	}
	if prodCfg.Ops.WorkerURL != "https://quadsmith.com" {
		t.Errorf("expected prod Ops WorkerURL https://quadsmith.com, got %q", prodCfg.Ops.WorkerURL)
	}
}

func TestLoadFromFile(t *testing.T) {
	rootPath, err := filepath.Abs("../sandbox.yaml")
	if err != nil {
		t.Fatalf("failed to get path: %v", err)
	}

	cfg, err := LoadFromFile(rootPath)
	if err != nil {
		t.Fatalf("failed to load sandbox file: %v", err)
	}

	if cfg.CLI.APIURL != "http://localhost:8080" {
		t.Errorf("expected sandbox CLI APIURL http://localhost:8080, got %q", cfg.CLI.APIURL)
	}
	if cfg.Ops.WorkerURL != "http://localhost:8081" {
		t.Errorf("expected sandbox Ops WorkerURL http://localhost:8081, got %q", cfg.Ops.WorkerURL)
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("QS_CLI_API_URL", "http://custom-api:9000")
	t.Setenv("QS_OPS_WORKER_URL", "http://custom-worker:9001")

	cfg, err := Load("prod")
	if err != nil {
		t.Fatalf("failed to load prod config: %v", err)
	}

	if cfg.CLI.APIURL != "http://custom-api:9000" {
		t.Errorf("expected CLI override http://custom-api:9000, got %q", cfg.CLI.APIURL)
	}
	if cfg.Ops.WorkerURL != "http://custom-worker:9001" {
		t.Errorf("expected Ops override http://custom-worker:9001, got %q", cfg.Ops.WorkerURL)
	}
}

func TestValidationFailures(t *testing.T) {
	cfg := &Config{}
	if err := cfg.ValidateCLI(); err == nil {
		t.Errorf("expected ValidateCLI to fail on empty config")
	}
	if err := cfg.ValidateOps(); err == nil {
		t.Errorf("expected ValidateOps to fail on empty config")
	}

	cfg.CLI.APIURL = "ftp://invalid-scheme"
	if err := cfg.ValidateCLI(); err == nil {
		t.Errorf("expected ValidateCLI to fail on invalid scheme")
	}
}
