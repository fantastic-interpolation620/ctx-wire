package gain

import (
	"strings"
	"testing"
)

// The by-source breakdown is the instrument for "are hook-path and shim-path
// savings comparable?" It must bucket entries by source, sum correctly, sort by
// savings with untagged last, and surface in the summary output.
func TestBySourceBreakdown(t *testing.T) {
	useTempLog(t)
	rec := func(cmd, agent, source string, raw, emitted int) {
		if err := RecordWithMeta(cmd, "f", "filtered", agent, source, raw, emitted, 0); err != nil {
			t.Fatal(err)
		}
	}
	rec("c1", "claude", "hook", 1000, 100) // hook saved 900
	rec("c2", "claude", "hook", 2000, 200) // hook saved 1800 (hook total: 2 cmds, 2700)
	rec("c3", "cline", "shim", 500, 50)    // shim saved 450
	rec("c4", "", "run", 100, 10)          // run saved 90
	rec("c5", "", "", 300, 30)             // untagged (pre-tag) saved 270

	s, err := Summarize()
	if err != nil {
		t.Fatal(err)
	}

	by := map[string]SourceStat{}
	for _, st := range s.BySource {
		by[st.Source] = st
	}
	if h := by["hook"]; h.Commands != 2 || h.SavedBytes != 2700 {
		t.Errorf("hook bucket: got %d cmds / %d saved, want 2 / 2700", h.Commands, h.SavedBytes)
	}
	if sh := by["shim"]; sh.Commands != 1 || sh.SavedBytes != 450 {
		t.Errorf("shim bucket: got %d cmds / %d saved, want 1 / 450", sh.Commands, sh.SavedBytes)
	}
	if r := by["run"]; r.SavedBytes != 90 {
		t.Errorf("run bucket: got %d saved, want 90", r.SavedBytes)
	}
	// Sorted by savings desc; the untagged ("") bucket must sink last.
	if s.BySource[0].Source != "hook" {
		t.Errorf("by-source must lead with the biggest saver (hook); got %q", s.BySource[0].Source)
	}
	if last := s.BySource[len(s.BySource)-1].Source; last != "" {
		t.Errorf("untagged source must sort last; got %q", last)
	}

	out := Format(s)
	if !strings.Contains(out, "By Source") || !strings.Contains(out, "hook") || !strings.Contains(out, "shim") {
		t.Errorf("summary output missing the By Source section:\n%s", out)
	}
}
