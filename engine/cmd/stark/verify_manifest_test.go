package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GetEvinced/stark-marketplace/engine/internal/provenance"
)

func TestVerifyManifestDigestsOnly(t *testing.T) {
	dir := t.TempDir()
	// committed bytes
	idx := filepath.Join(dir, "index.json")
	if err := os.WriteFile(idx, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	// manifest over those bytes
	m := provenance.Compute(map[string]int{"claude": 1},
		map[string][]byte{"index.json": []byte("hello")})
	mb, _ := m.Marshal()
	mp := filepath.Join(dir, "build-manifest.json")
	if err := os.WriteFile(mp, mb, 0o644); err != nil {
		t.Fatal(err)
	}

	// --skip-signature so the test does not require cosign on PATH; digest layer runs.
	if code := runVerifyManifest(mp, dir, true); code != 0 {
		t.Fatalf("want exit 0 for matching digests, got %d", code)
	}

	// tamper → integrity exit 3 (spec §9.8)
	if err := os.WriteFile(idx, []byte("TAMPERED"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runVerifyManifest(mp, dir, true); code != 3 {
		t.Fatalf("want exit 3 on digest mismatch, got %d", code)
	}
}
