// Package codex is the Codex (OpenAI) adapter target (spec §6, corrected matrix).
// Codex has NATIVE Skills at .agents/skills/<name>/SKILL.md (name+description
// required). Prompts are deprecated, so prompt/command map to a Codex skill.
// agent → emulated Codex skill. mcp → ~/.codex/config.toml [mcp_servers.<name>].
// Render is the canonical bundle-level entry point (CC-1): it iterates the bundle's
// artifacts, resolves each body via merge.Resolve (which runs fence.Strip) internally,
// and emits per-runtime output.
package codex

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GetEvinced/stark-marketplace/engine/internal/adapter"
	"github.com/GetEvinced/stark-marketplace/engine/internal/adapter/emulate"
	"github.com/GetEvinced/stark-marketplace/engine/internal/fieldmap"
	"github.com/GetEvinced/stark-marketplace/engine/internal/merge"
	"github.com/GetEvinced/stark-marketplace/engine/internal/model"
	"github.com/pelletier/go-toml/v2"
)

// version is the independently-versioned target identity (spec §7.7).
const version = "codex@1"

type Target struct{}

func New() *Target { return &Target{} }

func (t *Target) Runtime() model.Runtime { return model.RuntimeCodex }
func (t *Target) Version() string        { return version }

// modelMap translates canonical model ids → Codex model ids (§6.2 ActionMap).
func modelMap(canonical string) (string, bool) {
	switch canonical {
	case "opus", "sonnet":
		return "gpt-5-codex", true
	case "haiku":
		return "gpt-5-mini", true
	default:
		return "", false
	}
}

// Render emits Codex output for every artifact in the bundle that targets Codex.
// Per CC-1 it owns body resolution: merge.Resolve(a, RuntimeCodex) runs fence.Strip
// internally — the target never receives a pre-stripped body. merge.Resolve returns
// (Resolved, Findings, error); the resolved body is res.Body and merge findings are
// folded into the flat []adapter.Finding (CC-1).
func (t *Target) Render(b *model.Bundle) ([]adapter.OutputFile, []adapter.Finding, error) {
	var files []adapter.OutputFile
	var findings []adapter.Finding
	for _, a := range b.Artifacts {
		if !targetsRuntime(a, model.RuntimeCodex) {
			continue
		}
		res, mf, err := merge.Resolve(a, model.RuntimeCodex)
		if err != nil {
			return nil, nil, fmt.Errorf("codex: resolve %s/%s: %w", b.Name, a.Name, err)
		}
		findings = append(findings, foldFindings(b.Name, a, mf)...)
		out, err := t.emitArtifact(a, res.Body)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, out...)
	}
	return files, findings, nil
}

func targetsRuntime(a *model.Artifact, rt model.Runtime) bool {
	for _, r := range a.Runtimes {
		if r == rt {
			return true
		}
	}
	return false
}

// foldFindings translates merge.Findings into the canonical flat []adapter.Finding
// (CC-1): array foot-guns and author-divergence surface as warn-level findings.
func foldFindings(bundle string, a *model.Artifact, mf merge.Findings) []adapter.Finding {
	var out []adapter.Finding
	for _, field := range mf.ArrayDrops {
		out = append(out, adapter.Finding{
			Where: fmt.Sprintf("%s/%s@codex", bundle, a.Name),
			Level: "warn",
			Msg:   fmt.Sprintf("override array %q drops a base prefix (likely accidental — spec §4.3)", field),
		})
	}
	if mf.Diverged {
		out = append(out, adapter.Finding{
			Where: fmt.Sprintf("%s/%s@codex", bundle, a.Name),
			Level: "warn",
			Msg:   "diverged: " + mf.DivergedReason,
		})
	}
	return out
}

func (t *Target) emitArtifact(a *model.Artifact, body string) ([]adapter.OutputFile, error) {
	switch a.Type {
	case model.TypeSkill, model.TypePrompt, model.TypeCommand:
		return t.emitSkill(a, body, false), nil
	case model.TypeAgent:
		return t.emitSkill(a, body, true), nil // emulated
	case model.TypeMCP:
		return t.emitMCP(a)
	default:
		return nil, fmt.Errorf("codex: unsupported artifact type %q", a.Type)
	}
}

// emitSkill writes .agents/skills/<name>/SKILL.md. emulated=true prepends the
// §6.1 fidelity header (agents have no Codex primitive).
func (t *Target) emitSkill(a *model.Artifact, body string, emulated bool) []adapter.OutputFile {
	res := fieldmap.Apply(a, model.RuntimeCodex, modelMap)

	var fm strings.Builder
	fm.WriteString("---\n")
	// name + description are REQUIRED by Codex skills.
	fm.WriteString("name: " + a.Name + "\n")
	fm.WriteString("description: " + a.Description + "\n")
	// carried fields, sorted by key for determinism (§7.6).
	for _, k := range sortedKeys(res.Carried) {
		fm.WriteString(k + ": " + res.Carried[k] + "\n")
	}
	fm.WriteString("---\n")

	var b strings.Builder
	if emulated {
		b.WriteString(emulate.Header(a.Bundle, a.Name, "<!-- ", " -->"))
	}
	// derived fields render as usage prose (§6.2: argument-hint → usage note).
	if hint, ok := res.Derived["argument-hint"]; ok {
		b.WriteString("Usage: " + a.Name + " " + hint + "\n\n")
	}
	b.WriteString(body)

	return []adapter.OutputFile{{
		Path:    ".agents/skills/" + a.Name + "/SKILL.md",
		Content: []byte(fm.String() + b.String()),
	}}
}

func sortedKeys(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// codexMCPDoc is an ordered struct (NOT a map) so go-toml emits deterministic
// key order (§7.6). One server per emitted fragment; install merges by key (§9.2).
type codexMCPDoc struct {
	MCPServers map[string]codexMCPServer `toml:"mcp_servers"`
}

type codexMCPServer struct {
	Command string            `toml:"command,omitempty"`
	Args    []string          `toml:"args,omitempty"`
	URL     string            `toml:"url,omitempty"`
	Env     map[string]string `toml:"env,omitempty"`
}

func (t *Target) emitMCP(a *model.Artifact) ([]adapter.OutputFile, error) {
	if a.MCP == nil {
		return nil, fmt.Errorf("codex: mcp artifact %q has no mcp config", a.Name)
	}
	srv := codexMCPServer{
		Command: a.MCP.Command,
		Args:    a.MCP.Args,
		URL:     a.MCP.URL,
	}
	if len(a.MCP.Env) > 0 {
		srv.Env = map[string]string{}
		// secretRef → ${KEY} placeholder; never the secret value (§4.4).
		for k := range a.MCP.Env {
			srv.Env[k] = "${" + k + "}"
		}
	}
	doc := codexMCPDoc{MCPServers: map[string]codexMCPServer{a.Name: srv}}
	out, err := toml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("codex: marshal mcp toml: %w", err)
	}
	return []adapter.OutputFile{{Path: "config.toml", Content: out}}, nil
}
