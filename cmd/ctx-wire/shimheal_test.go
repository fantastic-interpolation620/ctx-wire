package main

import "testing"

// The shim self-heal must never point shims at a dev/temp binary: a dev build
// (version "dev", the default under `go test`) must yield no stable path, so
// RefreshManaged is never invoked.
func TestShimHealSkipsDevBuild(t *testing.T) {
	if version != "dev" {
		t.Skipf("test assumes the default dev version, got %q", version)
	}
	if _, ok := stableCurrentBinaryPath(); ok {
		t.Error("stableCurrentBinaryPath must return false for a dev build")
	}
	// Hot-path commands must never trigger the heal (no-op, must not panic).
	maybeRefreshManagedShims("run")
	maybeRefreshManagedShims("hook")
	maybeRefreshManagedShims("rewrite")
}
