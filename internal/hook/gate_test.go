package hook

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// bashPayload builds a hook payload with a JSON-escaped command, so a real
// newline in cmd is encoded as \n and the adapter decodes it back.
func bashPayload(tool, cmd string) string {
	b, _ := json.Marshal(cmd)
	return `{"tool_name":"` + tool + `","tool_input":{"command":` + string(b) + `}}`
}

// TestHooksDoNotRewriteUnattestable proves no adapter emits a wrapped
// `ctx-wire run` allow for a command that hides another command. Such a command
// must reach the host agent unchanged so the agent applies its own decision.
func TestHooksDoNotRewriteUnattestable(t *testing.T) {
	adapters := []struct {
		name string
		fn   func(io.Reader, io.Writer) error
		tool string
	}{
		{"claude", Claude, "Bash"},
		{"cursor", Cursor, "Shell"},
		{"gemini", Gemini, "run_shell_command"},
		{"codex", Codex, "Bash"},
		{"copilot", Copilot, "Bash"},
	}
	unattestable := []string{
		`git status $(rm -rf /tmp/x)`,
		`git log --pretty="$(rm -rf /tmp/x)"`,
		"git status\nrm -rf ~", // a real newline smuggling a second command
		`git status & rm -rf ~`,
		`diff <(a) <(b)`,
	}
	for _, a := range adapters {
		for _, cmd := range unattestable {
			var out bytes.Buffer
			if err := a.fn(strings.NewReader(bashPayload(a.tool, cmd)), &out); err != nil {
				t.Fatalf("%s(%q): %v", a.name, cmd, err)
			}
			if strings.Contains(out.String(), "ctx-wire run") {
				t.Errorf("%s emitted a rewrite for unattestable %q:\n%s", a.name, cmd, out.String())
			}
		}
		// Control: a benign command IS rewritten, so the adapter works and the
		// gate is what suppresses the unsafe ones.
		var out bytes.Buffer
		if err := a.fn(strings.NewReader(bashPayload(a.tool, "git status")), &out); err != nil {
			t.Fatalf("%s control: %v", a.name, err)
		}
		if !strings.Contains(out.String(), "ctx-wire run") {
			t.Errorf("%s did not rewrite a benign command; output:\n%s", a.name, out.String())
		}
	}
}

// TestHookRejectsOversizePayload proves an over-cap payload fails open
// (passthrough) instead of being truncated into a parseable prefix: a valid JSON
// object padded with whitespace past the cap would otherwise rewrite.
func TestHookRejectsOversizePayload(t *testing.T) {
	payload := `{"tool_name":"Bash","tool_input":{"command":"git status"}}` +
		strings.Repeat(" ", maxHookInput)
	var out bytes.Buffer
	if err := Claude(strings.NewReader(payload), &out); err != nil {
		t.Fatalf("Claude: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("oversize payload must fail open (no output), got %q", out.String())
	}
}

// TestHookStripsBOM proves a UTF-8 BOM-prefixed payload (as some hosts send on
// Windows) is still parsed and rewritten, rather than silently failing to parse.
func TestHookStripsBOM(t *testing.T) {
	bom := string([]byte{0xEF, 0xBB, 0xBF})
	payload := bom + `{"tool_name":"Bash","tool_input":{"command":"git status"}}`
	var out bytes.Buffer
	if err := Claude(strings.NewReader(payload), &out); err != nil {
		t.Fatalf("Claude: %v", err)
	}
	if !strings.Contains(out.String(), "ctx-wire run") {
		t.Errorf("BOM-prefixed payload should still rewrite; got %q", out.String())
	}
}
