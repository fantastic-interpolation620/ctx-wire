package filter

import (
	"math"
	"os"
	"strings"
)

// TruncateLevel scales every numeric cap a filter declares (truncate_lines_at,
// head_lines/tail_lines, max_lines, and group_by caps) without editing any
// TOML. The default level is the identity, so filters behave exactly as
// written and the conformance corpus stays binding; the dial only loosens or
// tightens HOW MUCH of recognized output is kept. It never makes a filter act
// on output it would not otherwise touch (pass-through stays pass-through).
type TruncateLevel int

const (
	// LevelDefault applies the TOML caps as written.
	LevelDefault TruncateLevel = iota
	// LevelLight doubles every cap (looser, keeps more output).
	LevelLight
	// LevelAggressive halves every cap, never below 1 (tighter, saves more).
	LevelAggressive
	// LevelNone removes the caps entirely: pattern rules still apply, but no
	// line/length/group truncation happens.
	LevelNone
)

// ParseTruncateLevel parses a user-facing level name. ok is false for an
// unknown (or empty) value so callers can fall back explicitly.
func ParseTruncateLevel(s string) (TruncateLevel, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "default", "normal", "medium", "reasonable":
		return LevelDefault, true
	case "light", "low", "loose":
		return LevelLight, true
	case "aggressive", "high", "tight":
		return LevelAggressive, true
	case "none", "off":
		return LevelNone, true
	default:
		return LevelDefault, false
	}
}

// configuredTruncateLevel is the config.toml value, wired in by main at
// startup (like SetUltraCompact). The CTX_WIRE_TRUNCATE env var wins over it.
var configuredTruncateLevel = LevelDefault

// SetConfiguredTruncateLevel records the [output] truncate config value.
func SetConfiguredTruncateLevel(l TruncateLevel) { configuredTruncateLevel = l }

// ResolveTruncateLevel returns the effective level for this invocation:
// CTX_WIRE_TRUNCATE if set and valid, else the configured value. Only runners
// call this; verify and tests pass an explicit level (zero value = default) so
// the conformance corpus is deterministic regardless of the environment.
func ResolveTruncateLevel() TruncateLevel {
	if l, ok := ParseTruncateLevel(os.Getenv("CTX_WIRE_TRUNCATE")); ok {
		return l
	}
	return configuredTruncateLevel
}

// scaledCap returns the effective value of an optional cap at level: nil stays
// nil, LevelNone uncaps (nil), LevelLight doubles, LevelAggressive halves with
// a floor of 1.
func scaledCap(n *int, level TruncateLevel) *int {
	if n == nil || level == LevelDefault {
		return n
	}
	if level == LevelNone {
		return nil
	}
	v := scaledCount(*n, level)
	return &v
}

// scaledCount scales a required (>= 1) cap. LevelNone returns MaxInt so "no
// cap" needs no special-casing at the call sites.
func scaledCount(n int, level TruncateLevel) int {
	switch level {
	case LevelLight:
		return n * 2
	case LevelAggressive:
		return max(n/2, 1)
	case LevelNone:
		return math.MaxInt
	default:
		return n
	}
}
