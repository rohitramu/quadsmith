package schemacheck

import (
	"testing"
)

func TestGitHelper(t *testing.T) {
	gh, err := NewGitHelper()
	if err != nil {
		t.Fatalf("Failed to initialize GitHelper: %v", err)
	}

	if gh.RepoRoot == "" {
		t.Fatal("RepoRoot should not be empty")
	}

	baseRef, err := gh.ResolveBaseRef()
	if err != nil {
		t.Fatalf("Failed to resolve base ref: %v", err)
	}

	t.Logf("Resolved base ref: %s", baseRef)

	// Test reading schema.sql from HEAD
	content, err := gh.GetFileAtRef(baseRef, "src/backend/db/schema.sql")
	if err != nil {
		t.Fatalf("Failed to read schema.sql at base ref %s: %v", baseRef, err)
	}

	if len(content) == 0 {
		t.Fatal("Retrieved schema content should not be empty")
	}

	schema, err := ParseSQL(content)
	if err != nil {
		t.Fatalf("Failed to parse retrieved schema: %v", err)
	}

	if len(schema.Tables) == 0 {
		t.Fatal("Parsed schema should have tables")
	}
}
