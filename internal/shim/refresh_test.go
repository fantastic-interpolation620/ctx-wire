package shim

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRefreshManaged(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	ctxWire := filepath.Join(t.TempDir(), "ctx-wire")
	absCtxWire, _ := filepath.Abs(ctxWire)

	write := func(name, content string) string {
		p := filepath.Join(dir, shimFileName(name))
		if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// A stale managed shim: carries the marker but a pre-guard body.
	stale := write("git", "#!/bin/sh\n"+marker+"\n# stale pre-guard shim body\nexec git \"$@\"\n")
	// An already-current managed shim: must be left as-is.
	current := write("rg", shimScript("rg", absCtxWire))
	// A non-ctx-wire file: must never be touched.
	foreignBody := "#!/bin/sh\necho not ours\n"
	foreign := write("ls", foreignBody)

	if n := RefreshManaged(ctxWire); n != 1 {
		t.Fatalf("RefreshManaged rewrote %d shims, want 1 (only the stale one)", n)
	}
	if got, _ := os.ReadFile(stale); string(got) != shimScript("git", absCtxWire) {
		t.Errorf("stale shim was not regenerated to the current template:\n%s", got)
	}
	if got, _ := os.ReadFile(current); string(got) != shimScript("rg", absCtxWire) {
		t.Error("an up-to-date shim should be left unchanged")
	}
	if got, _ := os.ReadFile(foreign); string(got) != foreignBody {
		t.Errorf("a non-ctx-wire file must not be touched, got:\n%s", got)
	}
}

func TestRefreshManagedHealsDuplicateDirs(t *testing.T) {
	dirA, dirB := t.TempDir(), t.TempDir()
	t.Setenv("PATH", dirA+string(os.PathListSeparator)+dirB)
	ctxWire := filepath.Join(t.TempDir(), "ctx-wire")
	absCtxWire, _ := filepath.Abs(ctxWire)

	stale := "#!/bin/sh\n" + marker + "\n# stale pre-guard body\nexec git \"$@\"\n"
	a := filepath.Join(dirA, shimFileName("git"))
	b := filepath.Join(dirB, shimFileName("git"))
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte(stale), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A foreign file in one of the dirs must survive.
	foreign := filepath.Join(dirB, shimFileName("ls"))
	foreignBody := "#!/bin/sh\necho not ours\n"
	if err := os.WriteFile(foreign, []byte(foreignBody), 0o755); err != nil {
		t.Fatal(err)
	}

	if n := RefreshManaged(ctxWire); n != 2 {
		t.Fatalf("RefreshManaged healed %d shims, want 2 (both duplicate dirs)", n)
	}
	for _, p := range []string{a, b} {
		if got, _ := os.ReadFile(p); string(got) != shimScript("git", absCtxWire) {
			t.Errorf("shim %s was not healed to the guarded template", p)
		}
	}
	if got, _ := os.ReadFile(foreign); string(got) != foreignBody {
		t.Error("a foreign file in a duplicate dir must not be touched")
	}
}
