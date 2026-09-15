package scaffold

import (
	"fmt"
	"sort"
	"strings"
)

// agentPaths maps a selectable agent tool to the template-relative
// paths it owns. A path is written only when its owner is selected;
// a path owned by nobody is always written.
var agentPaths = map[string][]string{
	"claude":  {"CLAUDE.md", ".claude"},
	"cursor":  {".cursor"},
	"copilot": {".github/copilot-instructions.md"},
	"gemini":  {"GEMINI.md"},
	// Codex reads AGENTS.md natively, which is unconditional. The
	// entry exists so `--agents=codex` is accepted rather than
	// rejected as unknown; it contributes no paths.
	"codex": {},
}

// AgentsNone is the --agents value (or an empty flag) that selects no
// agent tooling at all.
const AgentsNone = "none"

// AgentNames returns the recognised agent tool names, sorted.
func AgentNames() []string {
	names := make([]string, 0, len(agentPaths))
	for name := range agentPaths {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ParseAgents parses a comma-separated --agents list into a sorted,
// deduplicated set of recognised tool names. "" and "none" both mean no
// tools; "none" combined with any other name is an ambiguity error.
func ParseAgents(list string) ([]string, error) {
	if list == "" || list == AgentsNone {
		return nil, nil
	}

	var names []string
	for _, raw := range strings.Split(list, ",") {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if name == AgentsNone {
			return nil, fmt.Errorf("--agents=none cannot be combined with other tool names")
		}
		names = append(names, name)
	}

	return ParseAgentNames(names)
}

// ParseAgentNames validates names against the recognised tools and returns
// them sorted and deduplicated. Unlike ParseAgents there is no "none"
// shortcut here: "none" is as unrecognised as any other unknown name.
func ParseAgentNames(names []string) ([]string, error) {
	seen := map[string]bool{}
	for _, name := range names {
		if _, ok := agentPaths[name]; !ok {
			return nil, fmt.Errorf("unknown agent %q, valid names: %s", name, strings.Join(AgentNames(), ", "))
		}
		seen[name] = true
	}

	agents := make([]string, 0, len(seen))
	for name := range seen {
		agents = append(agents, name)
	}
	sort.Strings(agents)
	return agents, nil
}

// skipsAgentPath reports whether rel (slash-separated, relative to
// the template root) belongs to an agent tool that is not in
// selected. A path owned by no tool is never skipped.
func skipsAgentPath(rel string, selected map[string]bool) bool {
	for agent, paths := range agentPaths {
		if selected[agent] {
			continue
		}
		for _, p := range paths {
			if rel == p || strings.HasPrefix(rel, p+"/") {
				return true
			}
		}
	}
	return false
}
