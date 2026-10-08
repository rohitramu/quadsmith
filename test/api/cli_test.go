package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func resolveQSPath(t *testing.T) string {
	t.Helper()
	// Try relative from test/api/
	qsPath, err := filepath.Abs("../../bin/qs")
	if err == nil {
		if _, err := os.Stat(qsPath); err == nil {
			return qsPath
		}
	}
	// Try relative from test/
	qsPath, err = filepath.Abs("../bin/qs")
	if err == nil {
		if _, err := os.Stat(qsPath); err == nil {
			return qsPath
		}
	}
	// Try relative from repo root
	qsPath, err = filepath.Abs("bin/qs")
	if err == nil {
		if _, err := os.Stat(qsPath); err == nil {
			return qsPath
		}
	}
	t.Fatalf("Failed to resolve qs binary path (checked ../../bin/qs, ../bin/qs, and bin/qs)")
	return ""
}

func TestCLI_ListFrames(t *testing.T) {
	// Fail if we can't reach the sandbox
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Sandbox not running at 127.0.0.1:8080: %v", err)
	}

	qsPath := resolveQSPath(t)

	cmd := exec.Command(qsPath, "frames", "list", "--json")
	cmd.Env = append(cmd.Env, "QS_API_URL=http://127.0.0.1:8080")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI command failed: %v\nStderr: %s", err, stderr.String())
	}

	// Parse JSON
	var frames []map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &frames); err != nil {
		t.Fatalf("Failed to parse CLI JSON output: %v\nOutput: %s", err, stdout.String())
	}

	if len(frames) == 0 {
		t.Log("CLI returned empty frames list")
	} else {
		t.Logf("CLI returned %d frames", len(frames))
	}
}

func TestCLI_ListBuilds(t *testing.T) {
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get("http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("Sandbox not running at 127.0.0.1:8080: %v", err)
	}

	qsPath := resolveQSPath(t)

	cmd := exec.Command(qsPath, "builds", "list", "--json")
	cmd.Env = append(cmd.Env, "QS_API_URL=http://127.0.0.1:8080")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("CLI command failed: %v\nStderr: %s", err, stderr.String())
	}

	var builds []interface{} // can be empty or list of maps
	if err := json.Unmarshal(stdout.Bytes(), &builds); err != nil {
		t.Fatalf("Failed to parse CLI JSON output: %v\nOutput: %s", err, stdout.String())
	}

	t.Logf("CLI returned %d builds", len(builds))
}
