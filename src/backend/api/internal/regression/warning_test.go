package regression

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"quadsmith/api/pkg/schemacheck"
)

var reportMu sync.Mutex

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

	// Warning mode: log to test output and record
	t.Logf("⚠️  WARNING: %s", msg)

	if gh != nil && gh.RepoRoot != "" {
		writeReportFile(gh.RepoRoot, category, msg, true)
	}
}

// reportCleanStatus records that a layer evaluated cleanly without breaking changes.
func reportCleanStatus(gh *schemacheck.GitHelper, category, baseRef, statusMsg string) {
	msg := fmt.Sprintf("[%s vs %s]\n  %s\n", category, baseRef, statusMsg)
	if gh != nil && gh.RepoRoot != "" {
		writeReportFile(gh.RepoRoot, category, msg, false)
	}
}

func writeReportFile(repoRoot, category, content string, isBreaking bool) {
	reportMu.Lock()
	defer reportMu.Unlock()

	tmpDir := filepath.Join(repoRoot, ".tmp")
	_ = os.MkdirAll(tmpDir, 0755)

	slug := strings.ToLower(category)
	slug = strings.ReplaceAll(slug, " ", "_")
	slug = strings.ReplaceAll(slug, "/", "_")
	slug = strings.ReplaceAll(slug, "(", "")
	slug = strings.ReplaceAll(slug, ")", "")

	if isBreaking {
		_ = os.Remove(filepath.Join(tmpDir, fmt.Sprintf("clean_%s.log", slug)))
		_ = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("breaking_%s.log", slug)), []byte(content), 0644)
	} else {
		_ = os.Remove(filepath.Join(tmpDir, fmt.Sprintf("breaking_%s.log", slug)))
		_ = os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("clean_%s.log", slug)), []byte(content), 0644)
	}

	rebuildWarningLogLocked(tmpDir)
}

func rebuildWarningLogLocked(tmpDir string) {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}

	hasBreaking := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "breaking_") && strings.HasSuffix(entry.Name(), ".log") {
			hasBreaking = true
			break
		}
	}

	logPath := filepath.Join(tmpDir, "breaking_warning.log")
	if !hasBreaking {
		_ = os.Remove(logPath)
		return
	}

	// Order of sections: API, Database (SQL) Schema, Seed Data
	sections := []string{"api_protobuf", "database_sql_schema", "seed_data"}
	var combined strings.Builder

	for _, sec := range sections {
		breakingPath := filepath.Join(tmpDir, fmt.Sprintf("breaking_%s.log", sec))
		cleanPath := filepath.Join(tmpDir, fmt.Sprintf("clean_%s.log", sec))

		var sectionText string
		if data, err := os.ReadFile(breakingPath); err == nil && len(data) > 0 {
			sectionText = strings.TrimSpace(string(data))
		} else if data, err := os.ReadFile(cleanPath); err == nil && len(data) > 0 {
			sectionText = strings.TrimSpace(string(data))
		}

		if sectionText != "" {
			if combined.Len() > 0 {
				combined.WriteString("\n\n")
			}
			combined.WriteString(sectionText)
		}
	}

	if combined.Len() > 0 {
		combined.WriteString("\n")
	}

	_ = os.WriteFile(logPath, []byte(combined.String()), 0644)
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
