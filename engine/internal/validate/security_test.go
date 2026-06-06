package validate

import (
	"testing"

	"github.com/GetEvinced/stark-marketplace/engine/internal/model"
)

func mcp(cmd string, args ...string) *model.Artifact {
	return &model.Artifact{Name: "m", Type: model.TypeMCP, Runtimes: model.AllRuntimes(),
		MCP: &model.MCPConfig{Transport: "stdio", Command: cmd, Args: args}}
}

func TestCommandAllowlist(t *testing.T) {
	r := &Result{}
	checkSecurity(r, "demo/mcp/m", mcp("totally-evil-binary"))
	if !r.HasErrors() {
		t.Fatal("expected allowlist rejection")
	}
	r2 := &Result{}
	checkSecurity(r2, "demo/mcp/m", mcp("node", "server.js"))
	if r2.HasErrors() {
		t.Fatalf("node should be allowed: %+v", r2.Errors)
	}
}

func TestInlineEvalRejected(t *testing.T) {
	r := &Result{}
	checkSecurity(r, "demo/mcp/m", mcp("node", "-e", "doEvil()"))
	if !r.HasErrors() {
		t.Fatal("expected inline-eval rejection")
	}
}

func TestInlineCredWarnsWithoutAt(t *testing.T) {
	// `key=` and `--token=` carry a literal value — they must WARN even with no `@`.
	cases := [][]string{
		{"--api-base", "https://api.example.com?key=abc123"},
		{"--token=ghp_deadbeef"},
	}
	for i, args := range cases {
		r := &Result{}
		checkSecurity(r, "demo/mcp/m", mcp("node", args...))
		if len(r.Warnings) == 0 {
			t.Fatalf("case %d: expected inline-credential warning for %v", i, args)
		}
	}
}
