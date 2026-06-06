package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GetEvinced/stark-marketplace/engine/internal/load"
)

// TestGoldenSeedBundle renders the committed catalog's stark-gh bundle and
// compares each file to a checked-in golden. Run with UPDATE_GOLDEN=1 to regenerate.
var update = os.Getenv("UPDATE_GOLDEN") == "1"

func TestGoldenSeedBundle(t *testing.T) {
	cat, err := load.Load("../../../../catalog")
	if err != nil {
		t.Fatal(err)
	}
	bundle := cat.Bundles[0] // sorted: stark-gh
	files, _, err := New().Render(bundle)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("testdata", "golden", bundle.Name)
	for _, f := range files {
		gp := filepath.Join(dir, f.Path)
		if update {
			_ = os.MkdirAll(filepath.Dir(gp), 0o755)
			if err := os.WriteFile(gp, f.Content, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(gp)
		if err != nil {
			t.Fatalf("missing golden %s (run UPDATE_GOLDEN=1): %v", gp, err)
		}
		if string(want) != string(f.Content) {
			t.Fatalf("golden mismatch %s:\n--- want ---\n%s\n--- got ---\n%s", f.Path, want, f.Content)
		}
	}
}
