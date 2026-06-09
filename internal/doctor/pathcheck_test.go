package doctor

import (
	"runtime"
	"testing"
)

// A PATH without /usr/bin or /bin is the environment fault behind "no real X on
// PATH"; doctor must flag it so the user fixes their shell rather than blaming
// ctx-wire.
func TestMissingSystemPathDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix system dirs")
	}
	t.Setenv("PATH", "/opt/homebrew/bin:/usr/local/bin") // no /usr/bin, no /bin
	if got := missingSystemPathDirs(); len(got) != 2 {
		t.Fatalf("a PATH without /usr/bin or /bin must report both missing, got %v", got)
	}
	t.Setenv("PATH", "/usr/local/bin:/usr/bin:/bin")
	if got := missingSystemPathDirs(); len(got) != 0 {
		t.Fatalf("a PATH with the system dirs must report none missing, got %v", got)
	}
	t.Setenv("PATH", "/usr/bin:/opt/homebrew/bin") // /bin missing only
	if got := missingSystemPathDirs(); len(got) != 1 || got[0] != "/bin" {
		t.Fatalf("expected only /bin missing, got %v", got)
	}
}
