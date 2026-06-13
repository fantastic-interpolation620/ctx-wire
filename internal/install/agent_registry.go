package install

import (
	"fmt"
	"os"
	"path/filepath"
)

// agentDescriptor describes one AI coding agent ctx-wire knows how to wire and
// unwire. Only Uninstall is populated in this slice; future slices add Install,
// ConfigPaths, and DoctorCheck fields.
type agentDescriptor struct {
	Name      string
	Uninstall func(workdir string, r *IntegrationUninstallReport) error
}

// agentRegistry is the single source of truth for per-agent uninstall behavior.
// Both UninstallIntegrations (iterate all) and UninstallAgent (look up by name)
// consume this table, so each agent's uninstall logic lives in exactly one place.
// Order matches the previous implicit iteration order in UninstallIntegrations.
var agentRegistry = []agentDescriptor{
	{Name: "claude", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		dirs, err := ClaudeConfigDirs()
		if err != nil {
			return nil // path unavailable on this OS/setup
		}
		for _, dir := range dirs {
			path := filepath.Join(dir, "settings.json")
			changed, cerr := UninstallClaude(path)
			if cerr != nil {
				return cerr
			}
			if changed {
				r.Removed = append(r.Removed, "claude:"+dir)
			}
			memPath := filepath.Join(dir, "CLAUDE.md")
			if err := r.removeInstr("claude instructions:"+dir, memPath); err != nil {
				return err
			}
		}
		return nil
	}},

	{Name: "cursor", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		path, err := CursorHooksPath()
		if err != nil {
			return nil
		}
		changed, err := UninstallCursor(path)
		if err != nil {
			return err
		}
		if changed {
			r.Removed = append(r.Removed, "cursor")
		}
		return nil
	}},

	{Name: "codex", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		if path, err := CodexHooksPath(); err == nil {
			changed, err := UninstallCodexHooks(path)
			if err != nil {
				return err
			}
			if changed {
				r.Removed = append(r.Removed, "codex")
			}
		}
		if path, err := CodexConfigPath(); err == nil {
			res, err := UninstallCodexAgentEnv(path)
			if err != nil {
				return err
			}
			switch res {
			case CodexEnvUpdated:
				r.Removed = append(r.Removed, "codex agent env")
			case CodexEnvManual:
				r.Skipped = append(r.Skipped, path)
			}
		}
		if p, err := CodexAgentsPath(); err == nil {
			if err := r.removeInstr("codex instructions", p); err != nil {
				return err
			}
		}
		return nil
	}},

	{Name: "gemini", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		hookPath, err := GeminiHookPath()
		if err != nil {
			return nil
		}
		if settingsPath, err := GeminiSettingsPath(); err == nil {
			changed, err := UninstallGeminiSettings(settingsPath, hookPath)
			if err != nil {
				return err
			}
			if changed {
				r.Removed = append(r.Removed, "gemini settings")
			}
		}
		removed, skipped, err := UninstallGeminiHook(hookPath)
		if err != nil {
			return err
		}
		if removed {
			r.Removed = append(r.Removed, "gemini hook")
		}
		if skipped {
			r.Skipped = append(r.Skipped, hookPath)
		}
		if p, err := GeminiMemoryPath(); err == nil {
			if err := r.removeInstr("gemini instructions", p); err != nil {
				return err
			}
		}
		return nil
	}},

	{Name: "cline", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		return r.removeInstr("cline rules", ClineRulesPath(workdir))
	}},

	{Name: "windsurf", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		return r.removeInstr("windsurf rules", WindsurfRulesPath(workdir))
	}},

	{Name: "copilot", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		changed, err := UninstallCopilotHook(CopilotHookPath(workdir))
		if err != nil {
			return err
		}
		if changed {
			r.Removed = append(r.Removed, "copilot hook")
		}
		return r.removeInstr("copilot instructions", CopilotInstructionsPath(workdir))
	}},

	{Name: "kilocode", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		return r.removeInstr("kilocode rules", KilocodeRulesPath(workdir))
	}},

	{Name: "antigravity", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		return r.removeInstr("antigravity rules", AntigravityRulesPath(workdir))
	}},

	{Name: "vscode", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		changed, err := UninstallMCP(VSCodeMCPPath(workdir))
		if err != nil {
			return err
		}
		if changed {
			r.Removed = append(r.Removed, "vscode mcp")
		}
		return nil
	}},

	{Name: "visualstudio", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		path, err := VisualStudioMCPPath()
		if err != nil {
			return nil
		}
		changed, err := UninstallMCP(path)
		if err != nil {
			return err
		}
		if changed {
			r.Removed = append(r.Removed, "visualstudio mcp")
		}
		return nil
	}},

	{Name: "opencode", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		path, err := OpenCodePluginPath()
		if err != nil {
			return nil
		}
		if removeFileIfContent(path, opencodePlugin) {
			r.Removed = append(r.Removed, "opencode plugin")
		}
		return nil
	}},

	{Name: "pi", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		path, err := PiPluginPath()
		if err != nil {
			return nil
		}
		if removeFileIfContent(path, piPlugin) {
			r.Removed = append(r.Removed, "pi extension")
		}
		return nil
	}},

	{Name: "hermes", Uninstall: func(workdir string, r *IntegrationUninstallReport) error {
		dir, err := HermesPluginDir()
		if err != nil {
			return nil
		}
		if removeFileIfContent(filepath.Join(dir, "__init__.py"), hermesPluginInit) {
			_ = os.RemoveAll(dir)
			r.Removed = append(r.Removed, "hermes plugin")
		}
		return nil
	}},
}

// registryByName looks up a descriptor by agent name. Returns (desc, true) when
// found, or a zero value and false when the name is not in the table.
func registryByName(name string) (agentDescriptor, bool) {
	for _, a := range agentRegistry {
		if a.Name == name {
			return a, true
		}
	}
	return agentDescriptor{}, false
}

// errUnknownAgent returns the standard error for an unrecognized agent name.
func errUnknownAgent(name string) error {
	return fmt.Errorf("unknown agent %q", name)
}
