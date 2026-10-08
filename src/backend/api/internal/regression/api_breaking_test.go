package regression

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"quadsmith/api/pkg/schemacheck"
)

func findBufBinary(repoRoot string) (string, error) {
	candidates := []string{
		filepath.Join(repoRoot, "bin/buf"),
		filepath.Join(repoRoot, "../bin/buf"),
		filepath.Join(repoRoot, "../../bin/buf"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	// Fallback to PATH
	if p, err := exec.LookPath("buf"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("buf binary not found in bin/buf or PATH. Please run 'make generate' or 'make build'")
}

func TestAPI_NoBreakingChangesAgainstGitBase(t *testing.T) {
	gh, err := schemacheck.NewGitHelper()
	if err != nil {
		t.Fatalf("Failed to initialize GitHelper: %v", err)
	}

	bufBin, err := findBufBinary(gh.RepoRoot)
	if err != nil {
		t.Skipf("Skipping API breaking test: %v", err)
	}

	baseRef, err := gh.ResolveBaseRef()
	if err != nil {
		t.Fatalf("Failed to resolve git base ref: %v", err)
	}

	t.Logf("Running buf breaking against base ref: %s", baseRef)

	// buf breaking proto --against '.git#ref=<baseRef>,subdir=proto'
	againstArg := fmt.Sprintf(".git#ref=%s,subdir=proto", baseRef)

	cmd := exec.Command(bufBin, "breaking", "proto", "--against", againstArg)
	cmd.Dir = gh.RepoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr != nil {
		output := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
		reportBreakingChange(t, gh, "API (Protobuf)", baseRef, fmt.Sprintf("%s\n\nRun 'buf breaking' to inspect, or restore backwards compatibility.", output))
		return
	}

	t.Log("API Protobuf backwards compatibility check passed: 0 breaking changes detected.")
}

func TestAPI_BreakingDetectorCatchesViolations(t *testing.T) {
	gh, err := schemacheck.NewGitHelper()
	if err != nil {
		t.Fatalf("Failed to initialize GitHelper: %v", err)
	}

	bufBin, err := findBufBinary(gh.RepoRoot)
	if err != nil {
		t.Skipf("Skipping synthetic test: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "proto_breaking_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	v1Dir := filepath.Join(tmpDir, "v1")
	v2Dir := filepath.Join(tmpDir, "v2")
	if err := os.MkdirAll(filepath.Join(v1Dir, "testpkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(v2Dir, "testpkg"), 0755); err != nil {
		t.Fatal(err)
	}

	bufConfig := `version: v1
breaking:
  use:
    - FILE
lint:
  use:
    - DEFAULT
`
	if err := os.WriteFile(filepath.Join(v1Dir, "testpkg/buf.yaml"), []byte(bufConfig), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v2Dir, "testpkg/buf.yaml"), []byte(bufConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// Baseline proto
	v1Proto := `syntax = "proto3";
package testpkg;
message Device {
  string id = 1;
  string name = 2;
}
service DeviceService {
  rpc GetDevice(Device) returns (Device);
}
`
	if err := os.WriteFile(filepath.Join(v1Dir, "testpkg/test.proto"), []byte(v1Proto), 0644); err != nil {
		t.Fatal(err)
	}

	// Breaking proto: field 2 removed, RPC GetDevice removed
	v2Proto := `syntax = "proto3";
package testpkg;
message Device {
  string id = 1;
  // Field 2 removed
  int32 voltage = 3;
}
service DeviceService {
  // GetDevice removed
}
`
	if err := os.WriteFile(filepath.Join(v2Dir, "testpkg/test.proto"), []byte(v2Proto), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bufBin, "breaking", v2Dir, "--against", v1Dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err == nil {
		t.Fatal("Expected buf breaking to detect breaking changes (field deleted, RPC removed), but it succeeded")
	}

	output := stdout.String() + stderr.String()
	if !strings.Contains(output, "deleted") {
		t.Fatalf("Expected output to report deleted field/RPC, got:\n%s", output)
	}
	t.Logf("Synthetic breaking change was correctly caught:\n%s", strings.TrimSpace(output))
}
