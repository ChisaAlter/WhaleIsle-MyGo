package deeplink

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		raw    string
		action Action
		sid    string
		sec    string
		ok     bool
	}{
		{"whaleisle://open", ActionOpen, "", "", true},
		{"whaleisle://launcher", ActionLauncher, "", "", true},
		{"whaleisle://settings?section=credentials", ActionSettings, "", "credentials", true},
		{"whaleisle://session/abc", ActionSession, "abc", "", true},
		{"whaleisle://usage", ActionUsage, "", "", true},
		{"whaleisle://market", ActionMarket, "", "", true},
		{"whaleisle://schedule", ActionSchedule, "", "", true},
		{"whaleisle://agents", ActionAgents, "", "", true},
		{"whaleisle://auth", ActionAuth, "", "", true},
		{"whaleisle:///?action=settings&section=desktop", ActionSettings, "", "desktop", true},
		{"https://example.com", "", "", "", false},
		{"", "", "", "", false},
	}
	for _, tc := range cases {
		got, ok := Parse(tc.raw)
		if ok != tc.ok {
			t.Fatalf("%s ok=%v want %v", tc.raw, ok, tc.ok)
		}
		if !ok {
			continue
		}
		if got.Action != tc.action || got.SessionID != tc.sid || got.Section != tc.sec {
			t.Fatalf("%s -> %+v", tc.raw, got)
		}
	}
}
