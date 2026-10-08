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
//  1. BREAKING_AGAINST_REF environment variable (explicit override).
//  2. GITHUB_BASE_REF environment variable (GitHub Actions PR base).
//  3. If there are uncommitted changes: compare against HEAD.
//  4. If there are no uncommitted changes: compare against HEAD~1 (or HEAD if HEAD~1 does not exist).
func (g *GitHelper) ResolveBaseRef() (string, error) {
	if envRef := os.Getenv("BREAKING_AGAINST_REF"); envRef != "" {
		return strings.TrimSpace(envRef), nil
	}

	if prBase := os.Getenv("GITHUB_BASE_REF"); prBase != "" {
		if g.refExists("origin/" + prBase) {
			return g.mergeBase("HEAD", "origin/"+prBase)
		}
		if g.refExists(prBase) {
			return g.mergeBase("HEAD", prBase)
		}
	}

	// 1. If we have uncommitted changes, check against HEAD
	if g.HasUncommittedChanges() {
		return "HEAD", nil
	}

	// 2. If we don't have uncommitted changes, check against HEAD~1
	if g.refExists("HEAD~1") {
		return "HEAD~1", nil
	}

	return "HEAD", nil
}

func (g *GitHelper) refExists(ref string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", ref)
	cmd.Dir = g.RepoRoot
	return cmd.Run() == nil
}

func (g *GitHelper) commitHash(ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Dir = g.RepoRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *GitHelper) mergeBase(refA, refB string) (string, error) {
	cmd := exec.Command("git", "merge-base", refA, refB)
	cmd.Dir = g.RepoRoot
	out, err := cmd.Output()
	if err != nil {
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

// HasUncommittedChanges returns true if there are staged or unstaged changes in tracked files.
func (g *GitHelper) HasUncommittedChanges() bool {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	cmd.Dir = g.RepoRoot
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(bytes.TrimSpace(out)) > 0
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
