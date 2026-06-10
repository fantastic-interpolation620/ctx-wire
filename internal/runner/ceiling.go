package runner

import (
	"fmt"
	"io"
	"sync"

	"ctx-wire/internal/filter"
)

// The passthrough ceiling bounds what an unfiltered command can stream into
// the agent's context: the head passes through live, the tail of each stream
// is kept, and everything between is omitted with an explicit marker, never
// summarized. It generalizes the JSON passthrough ceiling (jsonGuard) to all
// passthrough output, under the same recovery rule the MCP relay uses: the
// ceiling only ever fires alongside a kept spool, so the full output is always
// one path away. Output at or under head+tail per stream is byte-exact.
var (
	// passthroughHeadBytes is the shared live budget across stdout and stderr.
	// Generous on purpose: normal command output must never see the ceiling.
	passthroughHeadBytes = 48 << 10
	// passthroughTailBytes is the per-stream tail kept after the head budget is
	// spent. Failures put the actionable signal (summaries, stack frames) at
	// the end, so the tail is what the agent actually needs.
	passthroughTailBytes = 16 << 10
)

// passthroughCeiling resolves the effective head/tail for the user's truncate
// level: none disables the ceiling entirely, light doubles it, aggressive
// halves it.
func passthroughCeiling() (head, tail int, enabled bool) {
	switch filter.ResolveTruncateLevel() {
	case filter.LevelNone:
		return 0, 0, false
	case filter.LevelLight:
		return passthroughHeadBytes * 2, passthroughTailBytes * 2, true
	case filter.LevelAggressive:
		return max(passthroughHeadBytes/2, 1), max(passthroughTailBytes/2, 1), true
	default:
		return passthroughHeadBytes, passthroughTailBytes, true
	}
}

// streamCeiling is the shared limiter behind the per-stream facets. The head
// budget is shared (the agent's context does not care which stream spent it);
// each facet keeps its own bounded tail so stdout and stderr stay separated.
type streamCeiling struct {
	mu    sync.Mutex
	head  int // remaining shared live budget
	tail  int // per-facet tail capacity
	limit int // effective head+tail, named in the omission marker
}

func newStreamCeiling(head, tail int) *streamCeiling {
	return &streamCeiling{head: head, tail: tail, limit: head + tail}
}

// ceilWriter is one stream's facet: live until the shared head budget is
// spent, then a bounded tail ring. flush emits the omission marker and the
// kept tail.
type ceilWriter struct {
	c       *streamCeiling
	dst     io.Writer
	tail    []byte
	omitted int // bytes diverted that will NOT be flushed from the ring
}

func (c *streamCeiling) writer(dst io.Writer) *ceilWriter {
	return &ceilWriter{c: c, dst: dst}
}

func (w *ceilWriter) Write(p []byte) (int, error) {
	w.c.mu.Lock()
	defer w.c.mu.Unlock()
	n := len(p)
	// Live while the shared head budget lasts; a write crossing the boundary
	// is split so the head stays byte-exact.
	if w.c.head > 0 {
		live := min(len(p), w.c.head)
		if _, err := w.dst.Write(p[:live]); err != nil {
			return 0, err
		}
		w.c.head -= live
		p = p[live:]
	}
	// Beyond the head: keep only the last tail-cap bytes of this stream.
	if len(p) > 0 {
		w.tail = append(w.tail, p...)
		if over := len(w.tail) - w.c.tail; over > 0 {
			w.omitted += over
			w.tail = append(w.tail[:0], w.tail[over:]...)
		}
	}
	return n, nil
}

// flush emits the held tail. When bytes were actually omitted (the diverted
// run exceeded the tail cap) the tail is preceded by an explicit marker; when
// everything diverted still fits the ring, the output is complete and no
// marker is shown. Returns whether this stream was truncated.
func (w *ceilWriter) flush() (bool, error) {
	w.c.mu.Lock()
	defer w.c.mu.Unlock()
	if len(w.tail) == 0 && w.omitted == 0 {
		return false, nil
	}
	if w.omitted > 0 {
		marker := fmt.Sprintf("\n[ctx-wire: %d bytes omitted (over the %d-byte passthrough ceiling); tail follows, full output spooled]\n",
			w.omitted, w.c.limit)
		if _, err := w.dst.Write([]byte(marker)); err != nil {
			return true, err
		}
	}
	_, err := w.dst.Write(w.tail)
	w.tail = nil
	return w.omitted > 0, err
}
