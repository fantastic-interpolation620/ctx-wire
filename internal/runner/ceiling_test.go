package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"ctx-wire/internal/tee"
)

// The passthrough-ceiling contract: unfiltered output at or under the ceiling
// is byte-exact; over it, the head streams live, the tail survives, the middle
// is omitted with a marker, and the full scrubbed output is KEPT in the spool
// (the recovery rule). CTX_WIRE_TRUNCATE=none disables the ceiling entirely.

// shrinkCeiling makes the ceiling small enough to exercise without megabytes.
func shrinkCeiling(t *testing.T, head, tail int) {
	t.Helper()
	oldH, oldT := passthroughHeadBytes, passthroughTailBytes
	passthroughHeadBytes, passthroughTailBytes = head, tail
	t.Cleanup(func() { passthroughHeadBytes, passthroughTailBytes = oldH, oldT })
}

func runStream(t *testing.T, script string) (stdout, stderr string, code int) {
	t.Helper()
	t.Setenv("CTX_WIRE_TEE_DIR", t.TempDir())
	var out, errBuf bytes.Buffer
	spool := tee.NewSpool("ceiling-test")
	code, err := streamLive(context.Background(), "sh", []string{"-c", script}, "sh -c test", spool, &out, &errBuf)
	if err != nil {
		t.Fatalf("streamLive: %v", err)
	}
	return out.String(), errBuf.String(), code
}

func TestCeilingUnderLimitIsByteExact(t *testing.T) {
	shrinkCeiling(t, 4096, 1024)
	stdout, stderr, code := runStream(t, `seq 1 100`)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	want := ""
	for i := 1; i <= 100; i++ {
		want += fmt.Sprintf("%d\n", i)
	}
	if stdout != want {
		t.Errorf("under-ceiling output not byte-exact:\n got %d bytes\nwant %d bytes", len(stdout), len(want))
	}
	if strings.Contains(stdout, "ctx-wire:") || strings.Contains(stderr, "[full output:") {
		t.Errorf("under-ceiling run must have no marker and no kept spool:\nstdout=%q\nstderr=%q", stdout, stderr)
	}
}

func TestCeilingOverLimitKeepsHeadTailMarkerAndSpool(t *testing.T) {
	shrinkCeiling(t, 2048, 512)
	// Numbered lines so head/tail content is checkable: ~7 bytes per line,
	// 3000 lines is ~20 KB, far over head+tail.
	stdout, stderr, code := runStream(t, `seq 1 3000`)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(stdout, "1\n2\n3\n") {
		t.Errorf("head must stream byte-exact from the start, got %q", stdout[:24])
	}
	if !strings.Contains(stdout, "bytes omitted (over the 2560-byte passthrough ceiling)") {
		t.Errorf("omission marker missing or wrong:\n%q", stdout)
	}
	if !strings.HasSuffix(stdout, "3000\n") {
		t.Errorf("tail must end with the final line, got %q", stdout[len(stdout)-24:])
	}
	if len(stdout) > 2048+512+256 {
		t.Errorf("emitted %d bytes, want roughly head+tail+marker", len(stdout))
	}
	// The recovery rule: the spool is KEPT on a successful-but-truncated run.
	if !strings.Contains(stderr, "[full output:") {
		t.Errorf("expected the kept-spool hint on stderr, got %q", stderr)
	}
	path := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(stderr), "[full output: "), "]")
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = home + path[1:]
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("spool not readable at %q: %v", path, err)
	}
	if !strings.Contains(string(disk), "\n1500\n") {
		t.Errorf("spool must hold the omitted middle (line 1500)")
	}
}

func TestCeilingNoneDisables(t *testing.T) {
	shrinkCeiling(t, 1024, 256)
	t.Setenv("CTX_WIRE_TRUNCATE", "none")
	stdout, _, code := runStream(t, `seq 1 3000`)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(stdout, "ctx-wire:") {
		t.Errorf("CTX_WIRE_TRUNCATE=none must disable the ceiling:\n%q", stdout[:200])
	}
	if !strings.HasSuffix(stdout, "3000\n") || !strings.HasPrefix(stdout, "1\n") {
		t.Error("with the ceiling off, output must be complete")
	}
}

func TestCeilingFailureKeepsFullSpoolAndTail(t *testing.T) {
	shrinkCeiling(t, 2048, 512)
	stdout, stderr, code := runStream(t, `seq 1 3000; echo "FATAL: the actual error" >&2; exit 7`)
	if code != 7 {
		t.Fatalf("exit %d, want 7", code)
	}
	// stderr is small: it must arrive intact (the shared head budget may be
	// spent by stdout, but the tail ring preserves it).
	if !strings.Contains(stderr, "FATAL: the actual error") {
		t.Errorf("the failure signal was lost:\n%q", stderr)
	}
	if !strings.HasSuffix(stdout, "3000\n") {
		t.Error("failed run must still keep the stdout tail")
	}
	if !strings.Contains(stderr, "[full output:") {
		t.Error("failed run must keep the spool")
	}
}

func TestCeilingBothStreamsShareHeadBudget(t *testing.T) {
	shrinkCeiling(t, 1024, 256)
	// stdout spends the whole head; stderr lands after it and must still
	// surface through its own tail ring.
	stdout, stderr, code := runStream(t, `seq 1 2000; echo "stderr-after-budget" >&2`)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "stderr-after-budget") {
		t.Errorf("stderr tail lost after stdout spent the head budget:\n%q", stderr)
	}
	total := len(stdout) + len(stderr)
	if total > 1024+2*256+512 {
		t.Errorf("combined emit %d bytes, want bounded by head + both tails + markers", total)
	}
}
