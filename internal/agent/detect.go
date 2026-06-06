package agent

import "strings"

// procInfo is one process in the ancestor walk.
type procInfo struct {
	ppid int
	cmd  string
}

// detectMaxDepth bounds the ancestor walk so a pathological tree can never spin.
const detectMaxDepth = 16

// detectFrom walks the process-ancestor chain starting at startPid, returning
// the first recognized agent (closest ancestor wins, so codex run inside an
// editor attributes to codex, not the editor). It is the pure core of detect,
// kept separate so it can be tested with a synthetic process map.
func detectFrom(startPid int, procs map[int]procInfo) string {
	pid := startPid
	for depth := 0; depth < detectMaxDepth && pid > 1; depth++ {
		p, ok := procs[pid]
		if !ok {
			return ""
		}
		if name := matchAgent(p.cmd); name != "" {
			return name
		}
		pid = p.ppid
	}
	return ""
}

type detectPattern struct {
	name     string
	patterns []string
}

var detectPatterns = []detectPattern{
	{name: "claude", patterns: []string{"claude"}},
	{name: "codex", patterns: []string{"codex"}},
	{name: "cursor", patterns: []string{"cursor"}},
	{name: "gemini", patterns: []string{"gemini"}},
	{name: "copilot", patterns: []string{"copilot"}},
	{name: "windsurf", patterns: []string{"windsurf"}},
	{name: "cline", patterns: []string{"cline"}},
	{name: "kilocode", patterns: []string{"kilocode"}},
	{name: "antigravity", patterns: []string{"antigravity"}},
	{name: "opencode", patterns: []string{"opencode"}},
	{name: "pi", patterns: []string{"pi-coding-agent", "pi coding agent", "/.pi/agent"}},
	{name: "hermes", patterns: []string{"hermes"}},
	{name: "vscode", patterns: []string{"vscode", "visual studio code"}},
	{name: "visualstudio", patterns: []string{"visualstudio", "visual studio"}},
}

// matchAgent returns the canonical agent name whose token appears in a process
// command, or "" for none. The list is ordered, so the first matching agent wins
// on the rare command that mentions two.
func matchAgent(cmd string) string {
	low := strings.ToLower(cmd)
	for _, item := range detectPatterns {
		for _, pattern := range item.patterns {
			if strings.Contains(low, pattern) {
				return item.name
			}
		}
	}
	return ""
}
