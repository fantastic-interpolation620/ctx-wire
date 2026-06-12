package main

import (
	"path/filepath"
	"testing"
)

// TestClaudeMCPConfigPath pins the per-config MCP path rule (verified on disk):
// the default ~/.claude uses the SIBLING ~/.claude.json (its in-dir copy is an
// empty stub), while a custom CLAUDE_CONFIG_DIR keeps the file IN the dir. The
// original `configDir + ".json"` got the default right by luck and missed every
// custom config.
func TestClaudeMCPConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	def := filepath.Join(home, ".claude")
	if got, want := claudeMCPConfigPath(def), filepath.Join(home, ".claude.json"); got != want {
		t.Errorf("default ~/.claude: got %q, want sibling %q", got, want)
	}
	for _, name := range []string{".claude-main", ".claude-ship"} {
		dir := filepath.Join(home, name)
		if got, want := claudeMCPConfigPath(dir), filepath.Join(dir, ".claude.json"); got != want {
			t.Errorf("custom %s: got %q, want in-dir %q", name, got, want)
		}
	}
}
