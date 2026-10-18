package components

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"whaleisle/internal/harness"
)

func TestHostBundleExactSourcePins(t *testing.T) {
	root := t.TempDir()
	m := HostBundleManifest{SchemaVersion: 1, Application: "whaleisle-mygo-host", Version: "1.2.3", Platform: "darwin", Arch: runtime.GOARCH, RuntimeDirectory: "runtime", HostReadinessGuard: harness.CurrentHostReadinessGuardID}
	check := func(want bool) {
		t.Helper()
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, hostBundleManifest), data, 0600); err != nil {
			t.Fatal(err)
		}
		_, err = readHostBundle(root)
		if (err == nil) != want {
			t.Fatalf("source %s, expected valid=%v: %v", m.HostSourceCommit, want, err)
		}
	}
	for _, pin := range []string{harness.CurrentHostSourceCommit, harness.CurrentHostE9SourceCommit} {
		m.HostSourceCommit = pin
		check(true)
	}
	m.HostSourceCommit = strings.Repeat("f", 40)
	check(false)
	m.HostSourceCommit = harness.CurrentHostE9SourceCommit
	m.HostReadinessGuard = ""
	check(false)
	m.HostReadinessGuard = harness.CurrentHostReadinessGuardID
	m.Platform = "linux"
	check(false)
	m.Platform = "darwin"
	m.Arch = "unsupported"
	check(false)
}
