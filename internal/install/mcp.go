package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"

	"ctx-wire/internal/agent"
)

// mcpServerName is the key ctx-wire registers itself under in an mcp.json.
const mcpServerName = "ctx-wire"

// VSCodeMCPPath returns the workspace MCP config path for VS Code Copilot,
// .vscode/mcp.json under the given directory (use the current working dir).
func VSCodeMCPPath(workdir string) string {
	return filepath.Join(workdir, ".vscode", "mcp.json")
}

// VisualStudioMCPPath returns the global MCP config path for Visual Studio
// Copilot, %USERPROFILE%/.mcp.json (mapped to the home dir cross-platform).
func VisualStudioMCPPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mcp.json"), nil
}

// desiredMCPServer is the stdio server entry pointing at `ctx-wire mcp`. Used
// both to write the config and to detect that it is already present.
func desiredMCPServer(agentName string) map[string]any {
	server := map[string]any{
		"type":    "stdio",
		"command": "ctx-wire",
		"args":    []any{"mcp"},
	}
	if ag := agent.Normalize(agentName); ag != "" {
		server["env"] = map[string]any{agent.EnvName: ag}
	}
	return server
}

func isManagedMCPServer(value any) bool {
	server, ok := value.(map[string]any)
	if !ok {
		return false
	}
	if len(server) != 3 && len(server) != 4 {
		return false
	}
	if server["type"] != "stdio" || server["command"] != "ctx-wire" {
		return false
	}
	if !reflect.DeepEqual(server["args"], []any{"mcp"}) {
		return false
	}
	env, hasEnv := server["env"]
	if !hasEnv {
		return true
	}
	envMap, ok := env.(map[string]any)
	if !ok || len(envMap) != 1 {
		return false
	}
	ag, ok := envMap[agent.EnvName].(string)
	return ok && agent.Normalize(ag) != ""
}

// InstallMCP merges the ctx-wire stdio server into the mcp.json at path, under
// the top-level "servers" object. Both VS Code and Visual Studio use this
// format. Idempotent; preserves any other servers and top-level keys; atomic
// write with .bak backup.
func InstallMCP(path, agentName string) (changed bool, err error) {
	root := map[string]any{}
	data, readErr := os.ReadFile(path)
	switch {
	case readErr == nil:
		if len(data) > 0 {
			if err := json.Unmarshal(data, &root); err != nil {
				return false, fmt.Errorf("parse %s: %w", path, err)
			}
		}
	case errors.Is(readErr, fs.ErrNotExist):
		// new file
	default:
		return false, readErr
	}
	if root == nil {
		root = map[string]any{}
	}

	servers, err := ensureJSONObject(root, "servers", path)
	if err != nil {
		return false, err
	}

	desired := desiredMCPServer(agentName)
	if existing, ok := servers[mcpServerName]; ok && reflect.DeepEqual(existing, desired) {
		return false, nil
	}
	servers[mcpServerName] = desired

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, err
	}
	if err := writeAtomic(path, append(out, '\n'), len(data) > 0); err != nil {
		return false, err
	}
	return true, nil
}
