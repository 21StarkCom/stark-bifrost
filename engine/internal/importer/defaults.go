package importer

import "github.com/21-Stark-AI/stark-marketplace/engine/internal/model"

const (
	defaultVersion    = "0.1.0"
	defaultMaturity   = model.MaturityBeta
	defaultOwnerName  = "21 Stark AI"
	defaultOwnerEmail = "engineering@21stark.com"
)

// defaultRuntimes keeps imported Claude Code skills and commands installable on
// OpenAI/Codex by default. Gemini remains an explicit human compatibility decision.
func defaultRuntimes() []model.Runtime {
	return []model.Runtime{model.RuntimeClaude, model.RuntimeCodex}
}

// applyArtifactDefaults fills the canonical-superset fields that stark-skills source
// lacks, and records each as a MetaNote for human review (spec §12).
func applyArtifactDefaults(a *model.Artifact, res *ImportResult, where string) {
	if a.Version == "" {
		a.Version = defaultVersion
		res.note(where, "version", "defaulted to "+defaultVersion+"; set the real semver")
	}
	if a.Maturity == "" {
		a.Maturity = defaultMaturity
		res.note(where, "maturity", "defaulted to beta; confirm stability level")
	}
	if len(a.Runtimes) == 0 {
		a.Runtimes = defaultRuntimes()
		res.note(where, "runtimes", "defaulted to [claude, codex]; add gemini only after verifying support")
	}
	if len(a.Tags) == 0 {
		res.note(where, "tags", "no tags imported; add discovery tags")
	}
	if a.Category == "" {
		res.note(where, "category", "no category; assign one for faceted search")
	}
}
