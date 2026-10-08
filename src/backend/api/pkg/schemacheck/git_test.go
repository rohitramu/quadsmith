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

	t.Logf("Resolved base ref: %s (uncommitted changes: %v)", baseRef, gh.HasUncommittedChanges())

	expectedRef := "HEAD~1"
	if !gh.refExists("HEAD~1") {
		expectedRef = "HEAD"
	}
	if baseRef != expectedRef {
		t.Errorf("Expected base ref to be %q (previous commit relative to currently checked out), got %q", expectedRef, baseRef)
	}

	// Test reading schema.sql from the resolved baseRef
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
