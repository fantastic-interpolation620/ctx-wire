package rewrite

import "strings"

// SegmentExplain describes the hook-rewrite decision for one command segment.
type SegmentExplain struct {
	Command   string // the trimmed segment text
	Wrapped   bool   // true if the hook would wrap it in `ctx-wire run`
	Reason    string // when not wrapped, why (a Reason* constant)
	Inner     string // the command `ctx-wire run` receives (for time: the timed command)
	Rewritten string // the full rewritten segment (when Wrapped)
}

// LineExplain is the full hook-rewrite explanation for a command line.
type LineExplain struct {
	Original string
	Result   string // the rewritten line (equals Original when nothing is wrapped)
	Segments []SegmentExplain
}

// Explain reports, segment by segment, whether the hook would rewrite a command
// line and why. It reuses the same passReason logic the rewriter uses, so the
// explanation can never drift from the actual rewrite behavior.
func Explain(line string) LineExplain {
	le := LineExplain{Original: line, Result: Line(line)}
	// A line with an unattestable construct is passed through whole (lineWith
	// refuses to rewrite it), so report it as one passthrough rather than per
	// segment, keeping Explain consistent with the actual rewrite.
	if ContainsUnattestableConstruct(line) {
		le.Segments = append(le.Segments, SegmentExplain{
			Command: strings.TrimSpace(line),
			Reason:  ReasonUnattestable,
		})
		return le
	}
	segs, _ := splitTopLevel(line)
	for _, seg := range segs {
		core := strings.TrimSpace(seg)
		if core == "" {
			continue
		}
		rewritten, inner, wrapped, reason := rewriteCore(core, prefix)
		le.Segments = append(le.Segments, SegmentExplain{
			Command:   core,
			Wrapped:   wrapped,
			Reason:    reason,
			Inner:     inner,
			Rewritten: rewritten,
		})
	}
	if len(le.Segments) == 0 {
		le.Segments = append(le.Segments, SegmentExplain{Reason: ReasonEmpty})
	}
	return le
}
