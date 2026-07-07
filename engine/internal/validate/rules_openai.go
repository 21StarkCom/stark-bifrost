package validate

import "github.com/21-Stark-AI/stark-marketplace/engine/internal/model"

// checkOpenAICompatibility enforces the marketplace policy that Claude-facing
// skills and slash commands must also be installable on the OpenAI/Codex surface.
func checkOpenAICompatibility(r *Result, where string, a *model.Artifact) {
	if a.Type != model.TypeSkill && a.Type != model.TypeCommand {
		return
	}
	if hasRuntime(a.Runtimes, model.RuntimeClaude) && !hasRuntime(a.Runtimes, model.RuntimeCodex) {
		r.Errorf(where, "Claude-targeted %s must also target codex for OpenAI compatibility", a.Type)
	}
}

func hasRuntime(runtimes []model.Runtime, want model.Runtime) bool {
	for _, rt := range runtimes {
		if rt == want {
			return true
		}
	}
	return false
}
