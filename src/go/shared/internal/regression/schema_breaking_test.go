package regression

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quadsmith/shared/pkg/schemacheck"
)

func TestSchema_NoBreakingChangesAgainstGitBase(t *testing.T) {
	gh, err := schemacheck.NewGitHelper()
	if err != nil {
		t.Fatalf("Failed to initialize GitHelper: %v", err)
	}

	liveSchemaDir := strings.TrimSpace(os.Getenv("QS_LIVE_SCHEMA_DIR"))
	baseRef := "live-db"
	if liveSchemaDir == "" {
		baseRef, err = gh.ResolveBaseRef()
		if err != nil {
			t.Fatalf("Failed to resolve git base ref: %v", err)
		}
		t.Logf("Checking database schema backwards compatibility against base ref: %s", baseRef)
	} else {
		t.Logf("Checking database schema backwards compatibility against live schema dir: %s", liveSchemaDir)
	}

	schemaFiles := []string{
		"quadsmith_api.sql",
		"quadsmith_ops.sql",
	}

	var allBreaking []string
	for _, filename := range schemaFiles {
		relPath := filepath.Join("src/db", filename)
		currPath := filepath.Join(gh.RepoRoot, relPath)
		currContent, err := os.ReadFile(currPath)
		if err != nil {
			t.Fatalf("Failed to read current %s: %v", relPath, err)
		}

		currSchema, err := schemacheck.ParseSQL(string(currContent))
		if err != nil {
			t.Fatalf("Failed to parse current %s: %v", relPath, err)
		}

		var baseContent string
		if liveSchemaDir != "" {
			livePath := filepath.Join(liveSchemaDir, filename)
			rawBase, err := os.ReadFile(livePath)
			if err != nil {
				t.Fatalf("Failed to read live schema %s: %v", livePath, err)
			}
			baseContent = string(rawBase)
		} else {
			baseContent, err = gh.GetFileAtRef(baseRef, relPath)
			if err != nil {
				if filename == "quadsmith_api.sql" {
					for _, legacyPath := range []string{
						"src/go/db/quadsmith_api.sql",
						"src/backend/db/quadsmith_api.sql",
						"src/backend/db/quadsmith.sql",
						"src/backend/db/schema.sql",
					} {
						if content, legacyErr := gh.GetFileAtRef(baseRef, legacyPath); legacyErr == nil {
							baseContent = content
							err = nil
							break
						}
					}
				}
				if err != nil {
					t.Logf("Baseline %s not found at %s; skipping baseline comparison for %s", relPath, baseRef, filename)
					continue
				}
			}
		}

		baseSchema, err := schemacheck.ParseSQL(baseContent)
		if err != nil {
			t.Fatalf("Failed to parse baseline %s from %s: %v", filename, baseRef, err)
		}

		breakingChanges := schemacheck.Compare(baseSchema, currSchema)
		for _, bc := range breakingChanges {
			allBreaking = append(allBreaking, fmt.Sprintf("  - [%s] %s", filename, bc.DetailedString()))
		}
	}

	if len(allBreaking) > 0 {
		reportBreakingChange(t, gh, "Database (SQL) Schema", baseRef, fmt.Sprintf("%s\nEnsure table/column modifications are backwards-compatible, or write pre-migration scripts.", strings.Join(allBreaking, "\n")))
		return
	}

	reportCleanStatus(gh, "Database (SQL) Schema", baseRef, "✓ No breaking changes detected (PostgreSQL schemas quadsmith_api.sql and quadsmith_ops.sql are backwards-compatible).")
	t.Log("Database schema backwards compatibility check passed: 0 breaking changes detected.")
}
