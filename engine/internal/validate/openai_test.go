package validate

import (
	"testing"

	"github.com/21StarkCom/stark-bifrost/engine/internal/model"
)

func TestOpenAICompatibilityRequiresCodexForClaudeSkills(t *testing.T) {
	a := &model.Artifact{Name: "s", Type: model.TypeSkill, Runtimes: []model.Runtime{model.RuntimeClaude}}
	r := &Result{}
	checkOpenAICompatibility(r, "demo/skill/s", a)
	if !r.HasErrors() {
		t.Fatal("Claude-targeted skills must also target codex")
	}
}

func TestOpenAICompatibilityRequiresCodexForClaudeCommands(t *testing.T) {
	a := &model.Artifact{Name: "cmd", Type: model.TypeCommand, Runtimes: []model.Runtime{model.RuntimeClaude}}
	r := &Result{}
	checkOpenAICompatibility(r, "demo/command/cmd", a)
	if !r.HasErrors() {
		t.Fatal("Claude-targeted commands must also target codex")
	}
}

func TestOpenAICompatibilityAllowsCodexParity(t *testing.T) {
	for _, typ := range []model.ArtifactType{model.TypeSkill, model.TypeCommand} {
		a := &model.Artifact{
			Name:     "x",
			Type:     typ,
			Runtimes: []model.Runtime{model.RuntimeClaude, model.RuntimeCodex},
		}
		r := &Result{}
		checkOpenAICompatibility(r, "demo/"+string(typ)+"/x", a)
		if r.HasErrors() {
			t.Fatalf("%s with codex parity should pass: %+v", typ, r.Errors)
		}
	}
}

func TestOpenAICompatibilityDoesNotApplyToOtherTypes(t *testing.T) {
	a := &model.Artifact{Name: "agent", Type: model.TypeAgent, Runtimes: []model.Runtime{model.RuntimeClaude}}
	r := &Result{}
	checkOpenAICompatibility(r, "demo/agent/agent", a)
	if r.HasErrors() {
		t.Fatalf("non skill/command artifacts are governed by the capability matrix: %+v", r.Errors)
	}
}
