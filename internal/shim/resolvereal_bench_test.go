package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkResolveRealRead contrasts the bounded head read ResolveReal now uses
// against the previous whole-file read, on a 5MB real-binary stand-in with no
// shim marker (the common case: a path argument that is a real binary).
func BenchmarkResolveRealRead(b *testing.B) {
	big := filepath.Join(b.TempDir(), "bigbin")
	if err := os.WriteFile(big, make([]byte, 5<<20), 0o755); err != nil {
		b.Fatal(err)
	}
	b.Run("head", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = isManagedShimFile(big)
		}
	})
	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			d, _ := os.ReadFile(big)
			_ = strings.Contains(string(d), marker)
		}
	})
}
