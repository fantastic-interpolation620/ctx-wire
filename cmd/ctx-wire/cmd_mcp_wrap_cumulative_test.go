package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// appendCumulative must accumulate (append, not overwrite) valid JSONL across
// sessions, so the Phase-0 dataset builds up rather than scattering.
func TestAppendCumulativeAccumulatesJSONL(t *testing.T) {
	dir := t.TempDir()
	m := &mcpMeasure{tools: map[string]*toolStat{
		"navigate_page": {calls: 1, resultByte: 90},
		"take_snapshot": {calls: 2, resultByte: 400},
	}}
	m.appendCumulative(dir) // session 1
	m.appendCumulative(dir) // session 2 must append, not clobber

	data, err := os.ReadFile(filepath.Join(dir, "mcp-measure.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := 0
	for _, ln := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if ln == "" {
			continue
		}
		var rec struct {
			Tool  string `json:"tool"`
			Calls int    `json:"calls"`
			Bytes int    `json:"bytes"`
		}
		if err := json.Unmarshal([]byte(ln), &rec); err != nil {
			t.Fatalf("line is not valid JSON: %v\n%s", err, ln)
		}
		if rec.Tool == "" {
			t.Errorf("entry missing tool name: %s", ln)
		}
		lines++
	}
	if lines != 4 { // 2 tools x 2 appends
		t.Errorf("got %d JSONL records, want 4 (2 tools across 2 sessions)", lines)
	}

	// A non-writable dir must not panic or error out (best-effort).
	m.appendCumulative(filepath.Join(dir, "does", "not", "exist"))
}
