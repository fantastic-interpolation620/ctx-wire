package install

import (
	"os"
	"path/filepath"
	"testing"
)

// makeConfigDir creates a directory that passes isClaudeConfigDir (has both
// settings.json and projects/).
func makeConfigDir(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(dir, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestClaudeConfigDirsAlwaysIncludesDefault verifies that ~/.claude is always
// returned even when it has no settings.json or projects/ yet (fresh install).
func TestClaudeConfigDirsAlwaysIncludesDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	dirs, err := ClaudeConfigDirs()
	if err != nil {
		t.Fatalf("ClaudeConfigDirs: %v", err)
	}
	if len(dirs) == 0 {
		t.Fatal("expected at least one dir, got none")
	}
	want := filepath.Join(home, ".claude")
	if dirs[0] != want {
		t.Errorf("dirs[0] = %q, want %q", dirs[0], want)
	}
}

// TestClaudeConfigDirsEnvAlwaysIncluded verifies that CLAUDE_CONFIG_DIR is
// always included, even when the dir does not pass the isClaudeConfigDir check
// (no settings.json, no projects/).
func TestClaudeConfigDirsEnvAlwaysIncluded(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	custom := filepath.Join(t.TempDir(), "custom-cfg")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", custom)

	dirs, err := ClaudeConfigDirs()
	if err != nil {
		t.Fatalf("ClaudeConfigDirs: %v", err)
	}
	found := false
	for _, d := range dirs {
		if d == custom {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CLAUDE_CONFIG_DIR %q not in dirs %v", custom, dirs)
	}
	// env dir is first.
	if dirs[0] != custom {
		t.Errorf("env dir should be first; got %v", dirs)
	}
}

// TestClaudeConfigDirsDeduplicatesEnvAndDefault verifies that when
// CLAUDE_CONFIG_DIR == ~/.claude the dir appears only once.
func TestClaudeConfigDirsDeduplicatesEnvAndDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))

	dirs, err := ClaudeConfigDirs()
	if err != nil {
		t.Fatalf("ClaudeConfigDirs: %v", err)
	}
	seen := map[string]int{}
	for _, d := range dirs {
		seen[d]++
	}
	if seen[filepath.Join(home, ".claude")] != 1 {
		t.Errorf("~/.claude appears %d times in dirs %v, want 1", seen[filepath.Join(home, ".claude")], dirs)
	}
}

// TestClaudeConfigDirsIncludesRealSiblings verifies that siblings with both
// settings.json and projects/ are included, while dirs that lack either are not.
func TestClaudeConfigDirsIncludesRealSiblings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	// Real sibling: should be included.
	realSibling := makeConfigDir(t, home, ".claude-main")

	// settings only (no projects/): should NOT be included.
	settingsOnly := filepath.Join(home, ".claude-mem")
	if err := os.MkdirAll(settingsOnly, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settingsOnly, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// projects/ only (no settings.json): should NOT be included.
	projectsOnly := filepath.Join(home, ".claude-proj")
	if err := os.MkdirAll(filepath.Join(projectsOnly, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Neither: should NOT be included.
	backupDir := filepath.Join(home, ".claude-settings-backup-20260101")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dirs, err := ClaudeConfigDirs()
	if err != nil {
		t.Fatalf("ClaudeConfigDirs: %v", err)
	}

	inDirs := map[string]bool{}
	for _, d := range dirs {
		inDirs[d] = true
	}

	if !inDirs[realSibling] {
		t.Errorf("real sibling %q not found in dirs %v", realSibling, dirs)
	}
	if inDirs[settingsOnly] {
		t.Errorf("settings-only dir %q should NOT be in dirs %v", settingsOnly, dirs)
	}
	if inDirs[projectsOnly] {
		t.Errorf("projects-only dir %q should NOT be in dirs %v", projectsOnly, dirs)
	}
	if inDirs[backupDir] {
		t.Errorf("backup dir %q should NOT be in dirs %v", backupDir, dirs)
	}
}

// TestClaudeConfigDirsDeduplicatesEnvAndSibling verifies that when
// CLAUDE_CONFIG_DIR points at a dir that is also a sibling (e.g. ~/.claude-main)
// it appears only once.
func TestClaudeConfigDirsDeduplicatesEnvAndSibling(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sibling := makeConfigDir(t, home, ".claude-ship")
	t.Setenv("CLAUDE_CONFIG_DIR", sibling)

	dirs, err := ClaudeConfigDirs()
	if err != nil {
		t.Fatalf("ClaudeConfigDirs: %v", err)
	}
	seen := map[string]int{}
	for _, d := range dirs {
		seen[d]++
	}
	if seen[sibling] != 1 {
		t.Errorf("%q appears %d times in dirs %v, want 1", sibling, seen[sibling], dirs)
	}
}

// TestClaudeConfigDirsOrder verifies the documented order: env, ~/.claude, sorted siblings.
func TestClaudeConfigDirsOrder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	env := filepath.Join(t.TempDir(), "env-cfg")
	if err := os.MkdirAll(env, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", env)

	// Two siblings: .claude-ship comes after .claude-main alphabetically.
	makeConfigDir(t, home, ".claude-ship")
	makeConfigDir(t, home, ".claude-main")

	dirs, err := ClaudeConfigDirs()
	if err != nil {
		t.Fatalf("ClaudeConfigDirs: %v", err)
	}

	if len(dirs) < 4 {
		t.Fatalf("expected at least 4 dirs; got %v", dirs)
	}
	if dirs[0] != env {
		t.Errorf("dirs[0] = %q, want env dir %q", dirs[0], env)
	}
	if dirs[1] != filepath.Join(home, ".claude") {
		t.Errorf("dirs[1] = %q, want ~/.claude", dirs[1])
	}
	// .claude-main < .claude-ship alphabetically.
	foundMain, foundShip := -1, -1
	for i, d := range dirs {
		if d == filepath.Join(home, ".claude-main") {
			foundMain = i
		}
		if d == filepath.Join(home, ".claude-ship") {
			foundShip = i
		}
	}
	if foundMain < 0 || foundShip < 0 {
		t.Fatalf("missing siblings in dirs %v", dirs)
	}
	if foundMain >= foundShip {
		t.Errorf("expected .claude-main (i=%d) before .claude-ship (i=%d)", foundMain, foundShip)
	}
}

// TestIsClaudeConfigDir covers the helper directly.
func TestIsClaudeConfigDir(t *testing.T) {
	home := t.TempDir()

	// Directory with both: true.
	both := filepath.Join(home, "both")
	if err := os.MkdirAll(filepath.Join(both, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(both, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !isClaudeConfigDir(both) {
		t.Error("both: expected isClaudeConfigDir = true")
	}

	// settings.json is a directory, not a file: false.
	settingsIsDir := filepath.Join(home, "settings-is-dir")
	if err := os.MkdirAll(filepath.Join(settingsIsDir, "settings.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(settingsIsDir, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if isClaudeConfigDir(settingsIsDir) {
		t.Error("settings-is-dir: expected isClaudeConfigDir = false")
	}

	// Missing projects/: false.
	noProjects := filepath.Join(home, "no-projects")
	if err := os.MkdirAll(noProjects, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noProjects, "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isClaudeConfigDir(noProjects) {
		t.Error("no-projects: expected isClaudeConfigDir = false")
	}

	// Missing settings.json: false.
	noSettings := filepath.Join(home, "no-settings")
	if err := os.MkdirAll(filepath.Join(noSettings, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if isClaudeConfigDir(noSettings) {
		t.Error("no-settings: expected isClaudeConfigDir = false")
	}
}
