package rewrite

import "testing"

// cat was removed from the shim default set to stop it corrupting
// command-substitution captures. That must NOT cost the agent-read savings: an
// agent's explicit `cat file` is still rewritten through the hook (it is
// attestable and its output is displayed, not eval'd). And a `$(cat ...)`
// capture must stay un-rewritten by the hook (the attestation gate), which is
// why de-shimming cat is what actually closes the corruption path.
func TestCatHookWrapInvariantAfterDeShim(t *testing.T) {
	// Agent-explicit read: still wrapped -> savings preserved.
	if got, want := LineForAgent("cat foo.txt", "claude"), "ctx-wire run --agent claude cat foo.txt"; got != want {
		t.Errorf("explicit cat read must still be hook-wrapped: got %q, want %q", got, want)
	}
	// Command-substitution capture: hook leaves it alone (attestation gate), so
	// the only thing that ever filtered it was the shim, now removed.
	if got := LineForAgent("data=$(cat foo.txt)", "claude"); got != "data=$(cat foo.txt)" {
		t.Errorf("a $(cat ...) capture must stay un-rewritten by the hook, got %q", got)
	}
}
