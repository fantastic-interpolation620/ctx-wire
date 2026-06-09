package mcpcompress

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

func loadFixtureText(t *testing.T, name string) string {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Skipf("fixture %s not present: %v", name, err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var msg struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.NewDecoder(gz).Decode(&msg); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, c := range msg.Result.Content {
		b.WriteString(c.Text)
	}
	return b.String()
}

var uidRe = regexp.MustCompile(`uid=\S+`)

// THE CENTERPIECE. The reducer must be purely subtractive: every line it keeps is
// byte-identical to a line in the input, and every uid in the output is an
// unchanged uid from the input. A renumbered uid hands the agent a valid-looking
// reference pointing at the WRONG element, a silent wrong-click, and is the one
// bug this test exists to catch.
func TestReduceNeverRenumbers(t *testing.T) {
	in := loadFixtureText(t, "github_pr_snapshot.json.gz")
	out, dropped := ReduceSnapshotText(in)

	if dropped == 0 || len(out) >= len(in) {
		t.Fatalf("expected real reduction; got dropped=%d, len in=%d out=%d", dropped, len(in), len(out))
	}
	// Purely subtractive: every output line exists byte-identical in the input.
	inLines := map[string]int{}
	for _, l := range strings.Split(in, "\n") {
		inLines[l]++
	}
	for _, l := range strings.Split(out, "\n") {
		if inLines[l] == 0 {
			t.Fatalf("reducer emitted a line not present byte-identical in the input (not subtractive):\n%q", l)
		}
	}
	// No new or renumbered uids: every output uid is an input uid.
	inUids := map[string]bool{}
	for _, u := range uidRe.FindAllString(in, -1) {
		inUids[u] = true
	}
	for _, u := range uidRe.FindAllString(out, -1) {
		if !inUids[u] {
			t.Fatalf("reducer produced a uid not in the input (renumbered/regenerated): %s", u)
		}
	}
}

func TestReduceReducesMeaningfully(t *testing.T) {
	in := loadFixtureText(t, "github_pr_snapshot.json.gz")
	out, dropped := ReduceSnapshotText(in)
	pct := 100 * float64(len(in)-len(out)) / float64(len(in))
	t.Logf("reduced %d -> %d chars (%.1f%% smaller), %d lines dropped", len(in), len(out), pct, dropped)
	if len(out) >= len(in) {
		t.Fatal("expected the reducer to shrink a real snapshot")
	}
	// Regression guard: if this drops well below the established ratio, the reducer
	// quietly got weaker (or the fixture changed). Conservative floor.
	if pct < 5 {
		t.Errorf("reduction %.1f%% is suspiciously low; reducer may have regressed", pct)
	}
}

func TestReduceSafeNoOpOnNonSnapshot(t *testing.T) {
	for _, s := range []string{"", "hello world", `{"some":"json"}`, "no uid lines here\njust text"} {
		out, dropped := ReduceSnapshotText(s)
		if out != s || dropped != 0 {
			t.Errorf("non-snapshot input must pass through unchanged: in=%q out=%q dropped=%d", s, out, dropped)
		}
	}
}

func TestReduceIdempotent(t *testing.T) {
	in := loadFixtureText(t, "github_pr_snapshot.json.gz")
	once, _ := ReduceSnapshotText(in)
	twice, d2 := ReduceSnapshotText(once)
	if twice != once {
		t.Errorf("reduce must be idempotent; a second pass changed the output (dropped=%d)", d2)
	}
}
