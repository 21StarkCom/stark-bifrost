package validate

import (
	"testing"

	"github.com/GetEvinced/stark-marketplace/engine/internal/model"
)

func TestOutputUniquenessCollision(t *testing.T) {
	// On Codex, both a skill and a command named "x" could map to the same
	// prompts/x namespace — collision must be caught.
	b := &model.Bundle{Name: "demo", Runtimes: model.AllRuntimes(), Artifacts: []*model.Artifact{
		{Name: "x", Type: model.TypeCommand, Runtimes: model.AllRuntimes()},
		{Name: "x", Type: model.TypePrompt, Runtimes: model.AllRuntimes()},
	}}
	r := &Result{}
	checkOutputUniqueness(r, b)
	if !r.HasErrors() {
		t.Fatal("expected output-namespace collision error")
	}
}
