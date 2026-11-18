package settings

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"

	"whaleisle/internal/ui/i18n"
)

func zhForTest(t *testing.T) {
	t.Helper()
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })
}

// TestDiscoveryMessageNeverPassesRawTextThrough covers the one provider path
// whose translator returns its input unchanged when it does not know the
// message: the pane must still explain the failure instead of quoting it.
func TestDiscoveryMessageNeverPassesRawTextThrough(t *testing.T) {
	zhForTest(t)
	known := discoveryMessage(errors.New("context canceled"))
	if !strings.Contains(known, "已取消") {
		t.Fatalf("known controller message = %q", known)
	}
	raw := errors.New("rpc error: code = Unavailable desc = dial tcp 10.0.0.1:443: connect: connection refused")
	got := discoveryMessage(raw)
	if !strings.Contains(got, "网络") {
		t.Fatalf("unknown controller message = %q", got)
	}
	if strings.Contains(got, "dial tcp") || strings.Contains(got, "rpc error") {
		t.Fatalf("unknown controller message leaked the raw text: %q", got)
	}
	if got := discoveryMessage(nil); got != "" {
		t.Fatalf("nil error = %q", got)
	}
}

// paintedCarrier matches the fields this package paints. A local variable that
// keeps the raw text for the classifier is fine; a carrier is not.
var paintedCarrier = regexp.MustCompile(`Title:|\.Error = |\*[A-Za-z]+ = |draft\.Err|g\.Status|e\.Status|sourceError|state\.message`)

func TestSettingsCarriersNeverTakeARawError(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for index, line := range strings.Split(string(source), "\n") {
			if strings.Contains(line, ".Error()") && paintedCarrier.MatchString(line) {
				t.Errorf("%s:%d paints a raw error: %s", name, index+1, strings.TrimSpace(line))
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}
