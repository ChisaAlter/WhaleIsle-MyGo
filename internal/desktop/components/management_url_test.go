package components

import (
	"strings"
	"testing"
)

func TestBridgeManagementUIURL(t *testing.T) {
	key := strings.Repeat("a", 32)
	valid := "http://127.0.0.1:8765/?k=" + key
	u, err := bridgeManagementURL(valid)
	if err != nil || u.String() != valid {
		t.Fatalf("valid UI address: %v %v", u, err)
	}
	for _, raw := range []string{"file:///tmp/program", "javascript:alert(1)", "https://example.com/?k=" + key, "http://127.0.0.1/?k=" + key, "http://user@127.0.0.1:8765/?k=" + key, "http://127.0.0.1:8765/?k=invalid"} {
		if _, err := bridgeManagementURL(raw); err == nil {
			t.Fatal("accepted non-management address")
		}
	}
}
