package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setConfigPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("CTX_WIRE_CONFIG", path)
	return path
}

func TestSetCaptureFileToolsCreatesMissing(t *testing.T) {
	path := setConfigPath(t)
	got, err := SetCaptureFileTools(true)
	if err != nil || got != path {
		t.Fatalf("SetCaptureFileTools = %q, %v", got, err)
	}
	cfg, err := Load()
	if err != nil || !cfg.Hooks.CaptureFileTools {
		t.Fatalf("Load after set = %+v, %v", cfg.Hooks, err)
	}
}

func TestSetCaptureFileToolsUpsertsExisting(t *testing.T) {
	path := setConfigPath(t)
	orig := "# my config\n[hooks]\n# keep raw\nexclude_commands = [\"curl\"]\n\n[output]\nultra_compact = true\n"
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SetCaptureFileTools(true); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || !cfg.Hooks.CaptureFileTools {
		t.Fatalf("flag not set: %+v, %v", cfg.Hooks, err)
	}
	if len(cfg.Hooks.ExcludeCommands) != 1 || !cfg.Output.UltraCompact {
		t.Errorf("surgical edit lost neighbors: %+v", cfg)
	}
	data, _ := os.ReadFile(path)
	for _, keep := range []string{"# my config", "# keep raw"} {
		if !strings.Contains(string(data), keep) {
			t.Errorf("comment lost: %q\n%s", keep, data)
		}
	}

	// Toggle off replaces the line in place (no duplicates).
	if _, err := SetCaptureFileTools(false); err != nil {
		t.Fatal(err)
	}
	cfg, _ = Load()
	if cfg.Hooks.CaptureFileTools {
		t.Error("flag still on after disable")
	}
	data, _ = os.ReadFile(path)
	if n := strings.Count(string(data), "capture_file_tools"); n != 1 {
		t.Errorf("want exactly one key line, got %d:\n%s", n, data)
	}
}

func TestSetCaptureFileToolsNoHooksSection(t *testing.T) {
	path := setConfigPath(t)
	if err := os.WriteFile(path, []byte("[output]\nultra_compact = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SetCaptureFileTools(true); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil || !cfg.Hooks.CaptureFileTools || !cfg.Output.UltraCompact {
		t.Fatalf("appended section broken: %+v, %v", cfg, err)
	}
}

// TestSetCaptureFileToolsIgnoresForeignSection pins the section-scoping fix:
// a capture_file_tools key under some OTHER table must never be rewritten;
// the real key is inserted under [hooks].
func TestSetCaptureFileToolsIgnoresForeignSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	orig := "[other]\ncapture_file_tools = false # not ours\n\n[hooks]\nexclude_commands = [\"curl\"]\n"
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CTX_WIRE_CONFIG", path)
	if _, err := SetCaptureFileTools(true); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	s := string(got)
	if !strings.Contains(s, "capture_file_tools = false # not ours") {
		t.Errorf("foreign section's key was rewritten:\n%s", s)
	}
	if !strings.Contains(s, "[hooks]\ncapture_file_tools = true") {
		t.Errorf("[hooks] did not gain the key:\n%s", s)
	}
}
