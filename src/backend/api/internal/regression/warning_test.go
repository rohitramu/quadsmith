package regression

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quadsmith/api/pkg/schemacheck"
)

// isStrictBreaking returns whether breaking changes should fail tests as errors.
// Currently (pre-launch), breaking changes are treated as warnings by default
// so developers are not blocked during rapid schema and proto iteration.
// Set BREAKING_CHECK_STRICT=1 or STRICT_BREAKING=1 to enforce hard errors.
//
// TODO(launch): When getting closer to launch, update the default behavior or
// enforce STRICT_BREAKING=1 in 'make test'.
func isStrictBreaking() bool {
	v := os.Getenv("STRICT_BREAKING")
	if v == "1" || strings.ToLower(v) == "true" {
		return true
	}
	v = os.Getenv("BREAKING_CHECK_STRICT")
	if v == "1" || strings.ToLower(v) == "true" {
		return true
	}
	return false
}

// reportBreakingChange reports a breaking change either as a test failure (if strict)
// or as a warning (if pre-launch warning mode). In warning mode, details are logged
// and written to <repo_root>/.tmp/breaking_warning.log so Makefile can surface them.
func reportBreakingChange(t *testing.T, gh *schemacheck.GitHelper, category, baseRef, details string) {
	t.Helper()
	msg := fmt.Sprintf("[%s Breaking Changes vs %s]\n%s", category, baseRef, details)

	if isStrictBreaking() {
		t.Fatalf("BREAKING %s CHANGE DETECTED vs %s:\n%s", category, baseRef, details)
		return
	}

	// Warning mode: log to test output and record to .tmp/breaking_warning.log
	t.Logf("⚠️  WARNING: %s", msg)

	if gh != nil && gh.RepoRoot != "" {
		tmpDir := filepath.Join(gh.RepoRoot, ".tmp")
		_ = os.MkdirAll(tmpDir, 0755)
		logPath := filepath.Join(tmpDir, "breaking_warning.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			_, _ = f.WriteString(msg + "\n\n")
		}
	}
}

func TestIsStrictBreaking(t *testing.T) {
	origStrict := os.Getenv("STRICT_BREAKING")
	origBreakingStrict := os.Getenv("BREAKING_CHECK_STRICT")
	defer func() {
		os.Setenv("STRICT_BREAKING", origStrict)
		os.Setenv("BREAKING_CHECK_STRICT", origBreakingStrict)
	}()

	os.Unsetenv("STRICT_BREAKING")
	os.Unsetenv("BREAKING_CHECK_STRICT")
	if isStrictBreaking() {
		t.Errorf("Expected isStrictBreaking() to be false by default in pre-launch mode")
	}

	os.Setenv("STRICT_BREAKING", "1")
	if !isStrictBreaking() {
		t.Errorf("Expected isStrictBreaking() to be true when STRICT_BREAKING=1")
	}

	os.Setenv("STRICT_BREAKING", "true")
	if !isStrictBreaking() {
		t.Errorf("Expected isStrictBreaking() to be true when STRICT_BREAKING=true")
	}

	os.Setenv("STRICT_BREAKING", "0")
	if isStrictBreaking() {
		t.Errorf("Expected isStrictBreaking() to be false when STRICT_BREAKING=0")
	}

	os.Unsetenv("STRICT_BREAKING")
	os.Setenv("BREAKING_CHECK_STRICT", "1")
	if !isStrictBreaking() {
		t.Errorf("Expected isStrictBreaking() to be true when BREAKING_CHECK_STRICT=1")
	}
}
