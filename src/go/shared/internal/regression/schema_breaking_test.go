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

	baseRef, err := gh.ResolveBaseRef()
	if err != nil {
		t.Fatalf("Failed to resolve git base ref: %v", err)
	}

	t.Logf("Checking database schema backwards compatibility against base ref: %s", baseRef)

	// 1. Read current quadsmith_api.sql
	currPath := filepath.Join(gh.RepoRoot, "src/db/quadsmith_api.sql")
	currContent, err := os.ReadFile(currPath)
	if err != nil {
		currPath = filepath.Join(gh.RepoRoot, "src/go/db/quadsmith_api.sql")
		currContent, err = os.ReadFile(currPath)
		if err != nil {
			currPath = filepath.Join(gh.RepoRoot, "src/backend/db/quadsmith_api.sql")
			currContent, err = os.ReadFile(currPath)
			if err != nil {
				currPath = filepath.Join(gh.RepoRoot, "src/backend/db/quadsmith.sql")
				currContent, err = os.ReadFile(currPath)
				if err != nil {
					t.Fatalf("Failed to read current quadsmith_api.sql: %v", err)
				}
			}
		}
	}

	currSchema, err := schemacheck.ParseSQL(string(currContent))
	if err != nil {
		t.Fatalf("Failed to parse current schema: %v", err)
	}

	// 2. Read baseline schema from git (src/db, src/go/db, src/backend/db, etc.)
	baseContent, err := gh.GetFileAtRef(baseRef, "src/db/quadsmith_api.sql")
	if err != nil {
		baseContent, err = gh.GetFileAtRef(baseRef, "src/go/db/quadsmith_api.sql")
		if err != nil {
			baseContent, err = gh.GetFileAtRef(baseRef, "src/backend/db/quadsmith_api.sql")
			if err != nil {
				baseContent, err = gh.GetFileAtRef(baseRef, "src/backend/db/quadsmith.sql")
				if err != nil {
					baseContent, err = gh.GetFileAtRef(baseRef, "src/backend/db/schema.sql")
					if err != nil {
						t.Fatalf("Failed to read baseline schema from git ref %s: %v", baseRef, err)
					}
				}
			}
		}
	}

	baseSchema, err := schemacheck.ParseSQL(baseContent)
	if err != nil {
		t.Fatalf("Failed to parse baseline schema.sql from %s: %v", baseRef, err)
	}

	// 3. Compare schemas
	breakingChanges := schemacheck.Compare(baseSchema, currSchema)
	if len(breakingChanges) > 0 {
		var report strings.Builder
		for _, bc := range breakingChanges {
			report.WriteString("  - " + bc.DetailedString() + "\n")
		}
		reportBreakingChange(t, gh, "Database (SQL) Schema", baseRef, fmt.Sprintf("%s\nEnsure table/column modifications are backwards-compatible, or write pre-migration scripts.", strings.TrimRight(report.String(), "\n")))
		return
	}

	reportCleanStatus(gh, "Database (SQL) Schema", baseRef, "✓ No breaking changes detected (PostgreSQL schema is backwards-compatible).")
	t.Log("Database schema backwards compatibility check passed: 0 breaking changes detected.")
}
