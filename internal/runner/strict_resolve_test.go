package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// When a command resolves only to a ctx-wire shim with no real binary behind it
// (e.g. a stripped PATH missing /usr/bin), Run must fail cleanly with 127 rather
// than exec the bare name and bounce back into the shim. Regression for the
// runner re-entry bug surfaced by Yosif's logs.
func TestRunFailsCleanWhenOnlyShim(t *testing.T) {
	reg := mustRegistry(t)
	shimDir := t.TempDir()
	t.Setenv("PATH", shimDir)

	// A managed shim with no real binary anywhere on PATH.
	managed := "#!/bin/sh\n# ctx-wire shim v1\nexec true\n"
	if err := os.WriteFile(filepath.Join(shimDir, "onlyshim"), []byte(managed), 0o755); err != nil {
		t.Fatal(err)
	}

	code, err := Run(context.Background(), reg, "onlyshim", nil)
	if err != nil {
		t.Fatalf("Run should not return a launch error for an only-shim command, got %v", err)
	}
	if code != 127 {
		t.Fatalf("Run on an only-shim command must exit 127 (clean, no re-entry), got %d", code)
	}
}
