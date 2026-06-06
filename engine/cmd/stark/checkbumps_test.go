package main

import (
	"path/filepath"
	"testing"
)

// On a clean repo (committed index matches the catalog with no un-bumped source
// edits), check-bumps must exit 0.
func TestCheckBumpsCleanRepoExitsZero(t *testing.T) {
	root := repoRoot(t)
	// ensure committed output is current so the previous index is coherent
	if code := runBuild(filepath.Join(root, "catalog"), root, false); code != 0 {
		t.Fatalf("pre-build want 0, got %d", code)
	}
	if code := runCheckBumps(filepath.Join(root, "catalog"), root); code != 0 {
		t.Fatalf("clean check-bumps want 0, got %d", code)
	}
}
