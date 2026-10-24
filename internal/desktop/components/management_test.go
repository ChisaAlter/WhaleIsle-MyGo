package components

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWhaleBridgeManagementAddress pins what the embedded management window may
// load: only a ready instance's validated loopback address.
func TestWhaleBridgeManagementAddress(t *testing.T) {
	userData := t.TempDir()
	stateDir := filepath.Join(userData, "components", "whalebridge", "data")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	statePath := filepath.Join(stateDir, "state.json")
	key := strings.Repeat("a", 32)
	valid := "http://127.0.0.1:8765/?k=" + key
	writeState := func(url string, ready bool) {
		body := `{"url":` + quote(url) + `,"ready":` + boolText(ready) + `}`
		if err := os.WriteFile(statePath, []byte(body), 0o600); err != nil {
			t.Fatalf("write state: %v", err)
		}
	}

	if _, err := WhaleBridgeManagementAddress(userData); err == nil {
		t.Fatal("missing state file accepted")
	}
	writeState(valid, false)
	if _, err := WhaleBridgeManagementAddress(userData); err == nil {
		t.Fatal("instance that is not ready accepted")
	}
	for _, raw := range []string{
		"https://example.com/?k=" + key,
		"http://127.0.0.1/?k=" + key,
		"http://127.0.0.1:8765/",
		"file:///tmp/page",
	} {
		writeState(raw, true)
		if _, err := WhaleBridgeManagementAddress(userData); err == nil {
			t.Fatalf("non-management address accepted: %s", raw)
		}
	}
	writeState(valid, true)
	got, err := WhaleBridgeManagementAddress(userData)
	if err != nil || got != valid {
		t.Fatalf("address = %q (%v)", got, err)
	}
	if _, err := WhaleBridgeManagementAddress(""); err == nil {
		t.Fatal("empty user data directory accepted")
	}
}

func quote(value string) string { return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"` }

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
