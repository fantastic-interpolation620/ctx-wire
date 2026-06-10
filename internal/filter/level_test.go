package filter

import (
	"strings"
	"testing"
)

func TestParseTruncateLevel(t *testing.T) {
	cases := []struct {
		in   string
		want TruncateLevel
		ok   bool
	}{
		{"default", LevelDefault, true},
		{"normal", LevelDefault, true},
		{"reasonable", LevelDefault, true},
		{"light", LevelLight, true},
		{"low", LevelLight, true},
		{"AGGRESSIVE", LevelAggressive, true},
		{"high", LevelAggressive, true},
		{"none", LevelNone, true},
		{"off", LevelNone, true},
		{" none ", LevelNone, true},
		{"", LevelDefault, false},
		{"bogus", LevelDefault, false},
	}
	for _, c := range cases {
		got, ok := ParseTruncateLevel(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseTruncateLevel(%q) = %v, %v; want %v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestScaledCap(t *testing.T) {
	ten := 10
	one := 1
	if got := scaledCap(nil, LevelAggressive); got != nil {
		t.Errorf("nil cap must stay nil, got %v", *got)
	}
	if got := scaledCap(&ten, LevelDefault); got == nil || *got != 10 {
		t.Errorf("default must be identity, got %v", got)
	}
	if got := scaledCap(&ten, LevelLight); got == nil || *got != 20 {
		t.Errorf("light must double, got %v", got)
	}
	if got := scaledCap(&ten, LevelAggressive); got == nil || *got != 5 {
		t.Errorf("aggressive must halve, got %v", got)
	}
	if got := scaledCap(&one, LevelAggressive); got == nil || *got != 1 {
		t.Errorf("aggressive must floor at 1, got %v", got)
	}
	if got := scaledCap(&ten, LevelNone); got != nil {
		t.Errorf("none must uncap, got %v", *got)
	}
}

func TestResolveTruncateLevelPrecedence(t *testing.T) {
	t.Cleanup(func() { SetConfiguredTruncateLevel(LevelDefault) })

	t.Setenv("CTX_WIRE_TRUNCATE", "")
	SetConfiguredTruncateLevel(LevelLight)
	if got := ResolveTruncateLevel(); got != LevelLight {
		t.Errorf("config value should apply, got %v", got)
	}
	t.Setenv("CTX_WIRE_TRUNCATE", "aggressive")
	if got := ResolveTruncateLevel(); got != LevelAggressive {
		t.Errorf("env must win over config, got %v", got)
	}
	t.Setenv("CTX_WIRE_TRUNCATE", "bogus")
	if got := ResolveTruncateLevel(); got != LevelLight {
		t.Errorf("invalid env must fall back to config, got %v", got)
	}
}

// TestApplyHonorsTruncateLevel proves the dial changes real Apply output at
// every level, and that the zero value is exactly today's behavior.
func TestApplyHonorsTruncateLevel(t *testing.T) {
	cf := firstFilter(t, `
schema_version = 1
[filters.cap]
match_command = "^cap\\b"
max_lines = 4
`)
	input := strings.TrimSpace(strings.Repeat("line\n", 8))

	lineCount := func(lvl TruncateLevel) int {
		out := ApplyWithMetaOptions(cf, input, ApplyOptions{TruncateLevel: lvl}).Output
		return len(strings.Split(out, "\n"))
	}

	// Default: 4 kept + 1 "truncated" marker.
	if got := lineCount(LevelDefault); got != 5 {
		t.Errorf("default = %d lines, want 5 (4 kept + marker)", got)
	}
	// Light doubles the cap to 8: all lines fit, no marker.
	if got := lineCount(LevelLight); got != 8 {
		t.Errorf("light = %d lines, want 8 (uncut)", got)
	}
	// Aggressive halves to 2: 2 kept + marker.
	if got := lineCount(LevelAggressive); got != 3 {
		t.Errorf("aggressive = %d lines, want 3 (2 kept + marker)", got)
	}
	// None removes the cap entirely.
	if got := lineCount(LevelNone); got != 8 {
		t.Errorf("none = %d lines, want 8 (uncapped)", got)
	}
}

func TestApplyTruncateLevelScalesGroupCaps(t *testing.T) {
	cf := firstFilter(t, `
schema_version = 1
[filters.grp]
match_command = "^grp\\b"
[filters.grp.group_by]
key = "^([a-z]+):"
max_per_group = 2
max_groups = 1
omit_label = "... (+%d more in %s)"
`)
	input := "a:1\na:2\na:3\na:4\nb:1\nb:2"

	out := ApplyWithMetaOptions(cf, input, ApplyOptions{}).Output
	if !strings.Contains(out, "more groups") || strings.Contains(out, "b:1") {
		t.Errorf("default should cap groups:\n%s", out)
	}
	out = ApplyWithMetaOptions(cf, input, ApplyOptions{TruncateLevel: LevelNone}).Output
	for _, want := range []string{"a:1", "a:4", "b:1", "b:2"} {
		if !strings.Contains(out, want) {
			t.Errorf("none should keep everything; missing %q:\n%s", want, out)
		}
	}
	out = ApplyWithMetaOptions(cf, input, ApplyOptions{TruncateLevel: LevelLight}).Output
	if !strings.Contains(out, "a:4") || !strings.Contains(out, "b:2") {
		t.Errorf("light (per-group 4, groups 2) should keep all:\n%s", out)
	}
}
