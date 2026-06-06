package codex

import (
	"testing"

	"github.com/GetEvinced/stark-marketplace/engine/internal/model"
)

func TestCodexEmitsMCPConfigToml(t *testing.T) {
	a := &model.Artifact{
		Name: "bigquery", Type: model.TypeMCP, Bundle: "stark-data",
		Description: "BQ MCP.", Version: "1.2.0",
		Runtimes: []model.Runtime{model.RuntimeCodex},
		MCP: &model.MCPConfig{
			Transport: "stdio", Command: "stark-bq-mcp",
			Args: []string{"--project", "${BQ_PROJECT}"},
			Env:  map[string]model.SecretRef{"BQ_PROJECT": {SecretRef: "bq-project-id"}},
		},
	}
	files, _, err := New().Render(bundleWith(a))
	if err != nil {
		t.Fatal(err)
	}
	body, ok := findFile(files, "config.toml")
	if !ok {
		t.Fatalf("expected config.toml; got %v", files)
	}
	// go-toml/v2 emits literal (single-quoted) strings for values with no special
	// chars — valid, deterministic TOML. The Codex MCP key is [mcp_servers.<name>].
	for _, want := range []string{
		"[mcp_servers.bigquery]",
		`command = 'stark-bq-mcp'`,
		`args = ['--project', '${BQ_PROJECT}']`,
		`[mcp_servers.bigquery.env]`,
		`BQ_PROJECT = '${BQ_PROJECT}'`,
	} {
		if !contains(body, want) {
			t.Fatalf("config.toml missing %q in:\n%s", want, body)
		}
	}
}
