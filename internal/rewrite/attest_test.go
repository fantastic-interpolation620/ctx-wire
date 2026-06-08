package rewrite

import "testing"

func TestContainsUnattestableConstruct(t *testing.T) {
	flagged := []string{
		"git status $(rm -rf /tmp/x)",         // command substitution
		"git status `rm -rf /tmp/x`",          // backtick substitution
		`git log --pretty="$(rm -rf /tmp/x)"`, // substitution inside double quotes
		"kill $(pgrep foo)",                   // common but still unattestable
		"diff <(a) <(b)",                      // process substitution
		"tee >(cat)",                          // process substitution (output)
		"echo $(( $(id) ))",                   // command sub NESTED in arithmetic still runs
		"echo $((`id`))",                      // backtick nested in arithmetic
		"git status\nrm -rf ~",                // real newline -> a second command
		"git status & rm -rf ~",               // background & then a second command
		"echo $(date) && ls",                  // substitution anywhere in the line
		"ls && cat <(rm -rf ~)",               // process sub in a later segment
	}
	for _, s := range flagged {
		if !ContainsUnattestableConstruct(s) {
			t.Errorf("ContainsUnattestableConstruct(%q) = false, want true", s)
		}
	}

	safe := []string{
		"git status",
		"go test ./... 2>&1",           // fd dup, not a file or command
		"git status > /dev/null",       // /dev/null, hides no command
		"git log > /tmp/out.txt",       // plain file redirect (handled elsewhere)
		"git status &",                 // trailing background, no second command
		"cat log | grep err &",         // trailing background after a pipe
		"echo 'literal $(not a sub)'",  // inside single quotes
		`echo "\$(not a sub)"`,         // escaped substitution
		"git -C $REPO status",          // plain variable expansion
		"echo ${HOME}",                 // parameter expansion
		"a && b || c",                  // ordinary operators
		"git add . && git commit -m x", // && separated, splitter handles it
		"echo $((1 + 2))",              // arithmetic expansion, executes no command
		"head -n $((LINES * 2)) f",     // arithmetic operand, still safe
	}
	for _, s := range safe {
		if ContainsUnattestableConstruct(s) {
			t.Errorf("ContainsUnattestableConstruct(%q) = true, want false", s)
		}
	}
}

func TestLinePassesThroughUnattestable(t *testing.T) {
	// A line that hides a command must come back unchanged, so a hook never
	// auto-allows a wrapped form of it (the agent evaluates the original).
	unchanged := []string{
		"git status $(rm -rf /tmp/x)",
		"git status `rm -rf /tmp/x`",
		`git log --pretty="$(rm -rf /tmp/x)"`,
		"git status\nrm -rf ~",
		"git status & rm -rf ~",
		"diff <(a) <(b)",
	}
	for _, s := range unchanged {
		if got := Line(s); got != s {
			t.Errorf("Line(%q) = %q, want unchanged", s, got)
		}
		if got := LineForAgent(s, "claude"); got != s {
			t.Errorf("LineForAgent(%q) = %q, want unchanged", s, got)
		}
	}

	// Benign commands still get wrapped (the gate must not over-block).
	if got := Line("git status"); got != "ctx-wire run git status" {
		t.Errorf("plain command should wrap: Line = %q", got)
	}
	if got := Line("git status &"); got != "ctx-wire run git status &" {
		t.Errorf("trailing & should still wrap: Line = %q", got)
	}
}

func TestExplainReportsUnattestable(t *testing.T) {
	le := Explain("git log --pretty=$(rm -rf ~)")
	if le.Result != le.Original {
		t.Errorf("Explain result = %q, want unchanged original", le.Result)
	}
	if len(le.Segments) != 1 || le.Segments[0].Wrapped || le.Segments[0].Reason != ReasonUnattestable {
		t.Errorf("Explain segments = %+v, want one unwrapped ReasonUnattestable", le.Segments)
	}
}
