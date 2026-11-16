package deeplink

import (
	"strings"
	"testing"
)

func TestParseRejectsUserinfoTraversalControlAndOverlongParams(t *testing.T) {
	rejected := []string{
		"whaleisle://user@session/abc",
		"whaleisle://user:pass@settings?section=general",
		`whaleisle://session/a\b`,
		"whaleisle://session/a/b",
		"whaleisle://session/..",
		"whaleisle://session/a..b",
		"whaleisle://session/abc\x00def",
		"whaleisle://session/abc\x7fdef",
		"whaleisle://session/" + strings.Repeat("s", 129),
		"whaleisle://settings?section=" + strings.Repeat("a", 65),
		"whaleisle://settings?section=a.b",
		"whaleisle://settings?section=../creds",
	}
	for _, raw := range rejected {
		if target, ok := Parse(raw); ok {
			t.Errorf("Parse(%q) = (%+v, true), want rejected", raw, target)
		}
	}

	if target, ok := Parse("whaleisle://session/sess_01.ok-2"); !ok || target.Action != ActionSession || target.SessionID != "sess_01.ok-2" {
		t.Fatalf("Parse valid session = (%+v, %v)", target, ok)
	}
	if target, ok := Parse("whaleisle://settings?section=desktop-1"); !ok || target.Action != ActionSettings || target.Section != "desktop-1" {
		t.Fatalf("Parse valid settings = (%+v, %v)", target, ok)
	}
}
