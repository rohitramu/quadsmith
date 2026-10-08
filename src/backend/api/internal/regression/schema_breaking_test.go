package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"quadsmith/api/pkg/schemacheck"
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

	// 1. Read current schema.sql
	currPath := filepath.Join(gh.RepoRoot, "src/backend/db/schema.sql")
	currContent, err := os.ReadFile(currPath)
	if err != nil {
		t.Fatalf("Failed to read current schema.sql at %s: %v", currPath, err)
	}

	currSchema, err := schemacheck.ParseSQL(string(currContent))
	if err != nil {
		t.Fatalf("Failed to parse current schema.sql: %v", err)
	}

	// 2. Read baseline schema.sql from git
	baseContent, err := gh.GetFileAtRef(baseRef, "src/backend/db/schema.sql")
	if err != nil {
		t.Fatalf("Failed to read baseline schema.sql from git ref %s: %v", baseRef, err)
	}

	baseSchema, err := schemacheck.ParseSQL(baseContent)
	if err != nil {
		t.Fatalf("Failed to parse baseline schema.sql from %s: %v", baseRef, err)
	}

	// 3. Compare schemas
	breakingChanges := schemacheck.Compare(baseSchema, currSchema)
	if len(breakingChanges) > 0 {
		var report strings.Builder
		report.WriteString(baseRef + ":\n")
		for _, bc := range breakingChanges {
			report.WriteString("  - " + bc.String() + "\n")
		}
		t.Fatalf("BREAKING DATABASE CHANGES DETECTED vs %s\n\nEnsure table/column modifications are backwards-compatible, or write pre-migration scripts.", report.String())
	}

	t.Log("Database schema backwards compatibility check passed: 0 breaking changes detected.")
}
