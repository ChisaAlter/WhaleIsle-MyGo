package credentials

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStorageStatusDescribesWithoutUnlocking(t *testing.T) {
	dir := t.TempDir()
	if s := InspectStorage(dir); s.Mode != "missing" {
		t.Fatal(s)
	}
	for _, tc := range []struct{ body, mode string }{{`{"apiKey":"never-display-this"}`, "plaintext"}, {`{"version":"safeStorage-v1","payload":"djEw"}`, "encrypted"}, {`{"version":"unexpected","payload":"secret"}`, "unknown"}} {
		if err := os.WriteFile(filepath.Join(dir, FileName), []byte(tc.body), 0600); err != nil {
			t.Fatal(err)
		}
		s := InspectStorage(dir)
		if s.Mode != tc.mode || strings.Contains(s.Error, "secret") || strings.Contains(s.Error, "never-display-this") {
			t.Fatalf("unexpected metadata: %+v", s)
		}
	}
}
