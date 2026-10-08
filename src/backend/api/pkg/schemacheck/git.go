package schemacheck

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitHelper provides utilities to resolve git baseline revisions and query git objects.
type GitHelper struct {
	RepoRoot string
}

// NewGitHelper creates a GitHelper discovered from the working directory.
func NewGitHelper() (*GitHelper, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to locate git repository: %w", err)
	}
	root := strings.TrimSpace(string(out))
	return &GitHelper{RepoRoot: root}, nil
}

// ResolveBaseRef determines the appropriate git reference to compare against.
// Precedence:
// 1. BREAKING_AGAINST_REF environment variable.
// 2. GITHUB_BASE_REF environment variable (GitHub Actions PR base).
// 3. git merge-base HEAD origin/main.
// 4. git merge-base HEAD main.
// 5. HEAD.
func (g *GitHelper) ResolveBaseRef() (string, error) {
	if envRef := os.Getenv("BREAKING_AGAINST_REF"); envRef != "" {
		return strings.TrimSpace(envRef), nil
	}

	if prBase := os.Getenv("GITHUB_BASE_REF"); prBase != "" {
		// In GitHub Actions PR, check origin/<prBase> first
		if g.refExists("origin/" + prBase) {
			return g.mergeBase("HEAD", "origin/"+prBase)
		}
		if g.refExists(prBase) {
			return g.mergeBase("HEAD", prBase)
		}
	}

	if g.refExists("origin/main") {
		return g.mergeBase("HEAD", "origin/main")
	}

	if g.refExists("main") {
		return g.mergeBase("HEAD", "main")
	}

	return "HEAD", nil
}

func (g *GitHelper) refExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Dir = g.RepoRoot
	return cmd.Run() == nil
}

func (g *GitHelper) mergeBase(refA, refB string) (string, error) {
	cmd := exec.Command("git", "merge-base", refA, refB)
	cmd.Dir = g.RepoRoot
	out, err := cmd.Output()
	if err != nil {
		// Fallback to refB if merge-base fails
		return refB, nil
	}
	base := strings.TrimSpace(string(out))
	if base == "" {
		return refB, nil
	}
	return base, nil
}

// GetFileAtRef retrieves the contents of a relative file path at a specific git ref.
func (g *GitHelper) GetFileAtRef(ref, relativePath string) (string, error) {
	// Normalize path relative to repository root
	cleanPath := filepath.Clean(relativePath)
	spec := fmt.Sprintf("%s:%s", ref, cleanPath)

	cmd := exec.Command("git", "show", spec)
	cmd.Dir = g.RepoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to retrieve %s: %s (%w)", spec, stderr.String(), err)
	}

	return stdout.String(), nil
}

// IsDirty returns true if the specified relative path has uncommitted changes.
func (g *GitHelper) IsDirty(path string) bool {
	cmd := exec.Command("git", "status", "--porcelain", path)
	cmd.Dir = g.RepoRoot
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(out)) > 0
}
