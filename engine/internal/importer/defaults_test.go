package importer

import (
	"testing"

	"github.com/21StarkCom/stark-bifrost/engine/internal/model"
)

func TestDefaultsAppliedAndNoted(t *testing.T) {
	a := &model.Artifact{Name: "x", Type: model.TypeSkill, Description: "d"}
	res := &ImportResult{}
	applyArtifactDefaults(a, res, "demo/skill/x")

	if a.Version != defaultVersion {
		t.Fatalf("version = %q, want %q", a.Version, defaultVersion)
	}
	if a.Maturity != model.MaturityBeta {
		t.Fatalf("maturity = %q, want beta", a.Maturity)
	}
	if len(a.Runtimes) != 2 || a.Runtimes[0] != model.RuntimeClaude || a.Runtimes[1] != model.RuntimeCodex {
		t.Fatalf("runtimes = %+v, want [claude codex]", a.Runtimes)
	}
	// every defaulted field must be recorded for the human checklist
	if len(res.Notes) < 3 {
		t.Fatalf("want >=3 metadata notes, got %d: %+v", len(res.Notes), res.Notes)
	}
}
