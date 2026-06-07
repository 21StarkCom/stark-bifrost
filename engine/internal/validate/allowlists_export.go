package validate

import "sort"

// CommandAllowlist returns the sorted list of MCP `command` basenames that
// MAY be spawned on a developer's machine. Exposed so `stark allowlist` can
// surface the authoritative list into docs without that doc drifting from the
// source of truth. To widen the list, edit allowlist.go (CODEOWNERS-gated).
func CommandAllowlist() []string {
	out := make([]string, 0, len(commandAllowlist))
	for k := range commandAllowlist {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// AgentToolAllowlist returns the sorted list of agent.tools grants treated as
// known-safe. Unknown grants surface as a lint warning (toolsallow.go), not a
// hard error — but the canonical list is here so docs and CI gates have one
// source of truth.
func AgentToolAllowlist() []string {
	out := make([]string, 0, len(agentToolAllowlist))
	for k := range agentToolAllowlist {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
