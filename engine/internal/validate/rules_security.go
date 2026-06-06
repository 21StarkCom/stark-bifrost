package validate

import (
	"path"
	"strings"

	"github.com/GetEvinced/stark-marketplace/engine/internal/model"
)

func checkSecurity(r *Result, where string, a *model.Artifact) {
	if a.Type != model.TypeMCP || a.MCP == nil {
		return
	}
	m := a.MCP
	if m.Transport == "stdio" {
		base := path.Base(m.Command)
		if base != m.Command && strings.Contains(m.Command, "/") {
			r.Errorf(where, "mcp.command must be a bare allowlisted basename, got path %q", m.Command)
		}
		if !commandAllowlist[base] {
			r.Errorf(where, "mcp.command %q not on the allowlist (spec §7.4)", base)
		}
		for _, arg := range m.Args {
			if inlineEvalFlags[arg] {
				r.Errorf(where, "mcp.args contains inline-eval flag %q", arg)
			}
			if strings.HasPrefix(arg, "npx") && !strings.Contains(arg, "@") {
				r.Warnf(where, "unpinned npx in args %q — pin a version/digest", arg)
			}
		}
	}
	// inline credential scan in args/url (defense-in-depth; env is structurally safe)
	for _, arg := range m.Args {
		scanInlineCred(r, where, arg)
	}
	scanInlineCred(r, where, m.URL)
}

func scanInlineCred(r *Result, where, s string) {
	low := strings.ToLower(s)
	// Patterns that warn ON THEIR OWN (a literal value follows the token).
	for _, p := range []string{"token=", "key=", "--token=", "--password="} {
		if strings.Contains(low, p) {
			r.Warnf(where, "possible inline credential in %q — use secretRef", s)
			return
		}
	}
	// userinfo-style creds (e.g. https://user:pass@host) only matter when an `@` follows.
	for _, p := range []string{"--token", "--password", "://"} {
		if strings.Contains(low, p) && strings.Contains(low, "@") {
			r.Warnf(where, "possible inline credential in %q — use secretRef", s)
			return
		}
	}
}
