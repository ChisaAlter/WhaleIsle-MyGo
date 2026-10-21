package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestClassifyTurnFailureRecognizesAPIKeyErrors(t *testing.T) {
	cases := []struct {
		body string
		code string
		kind turnFailureKind
	}{
		{"llm-deepseek: no API key for provider route deepseek/deepseek-chat", "", turnFailureAPIKey},
		{"no API key configured", "MISSING_API_KEY", turnFailureAPIKey},
		{"请求失败：未配置 API Key", "", turnFailureAPIKey},
		{"invalid API key", "AUTH", turnFailureAuth},
		{"account quota exhausted", "QUOTA", turnFailureQuota},
		{"run interrupted by user", "INTERRUPTED", turnFailureInterrupted},
		{"HTTP 502 from upstream", "", turnFailureGeneric},
	}
	for _, tc := range cases {
		got := classifyTurnFailure(tc.body, tc.code)
		if got.Kind != tc.kind {
			t.Fatalf("classify(%q, %q) = %v, want %v", tc.body, tc.code, got.Kind, tc.kind)
		}
		if got.Title == "" {
			t.Fatalf("classify(%q) produced no headline", tc.body)
		}
		if got.Raw != strings.TrimSpace(tc.body) {
			t.Fatalf("classify(%q) dropped the raw text: %q", tc.body, got.Raw)
		}
		if tc.kind == turnFailureGeneric && got.Action != "" {
			t.Fatalf("generic failure must not offer an action: %q", got.Action)
		}
		if tc.kind == turnFailureAPIKey && got.Action != i18n.T(i18n.KeyChatOpenModelSettings) {
			t.Fatalf("api-key failure action = %q", got.Action)
		}
	}
}

func TestSplitTurnFailureKeepsTheHostCode(t *testing.T) {
	text, code := splitTurnFailure("llm-deepseek: no API key for provider route x MISSING_API_KEY")
	if strings.Contains(text, "MISSING_API_KEY") {
		t.Fatalf("code leaked into the headline text: %q", text)
	}
	if code != "MISSING_API_KEY" {
		t.Fatalf("code = %q", code)
	}
}

// The card leads with a localized sentence and an action; the raw Host error
// only appears after the details toggle and is never dropped.
func TestTurnFailureCardLocalizesAndKeepsRawDetails(t *testing.T) {
	raw := "llm-deepseek: no API key for provider route deepseek/deepseek-chat"
	opened := 0
	palette := theme.Resolve("deepseek.light").Light
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(12).Children(func() {
			paintTurnFailure(c, "m1", classifyTurnFailure(raw, "MISSING_API_KEY"), palette,
				ui.RGB(20, 20, 20), ui.RGB(110, 110, 110), func() { opened++ })
		})
	}, 760, 320)
	if !tt.HasText(i18n.T(i18n.KeyChatFailureAPIKeyTitle)) {
		t.Fatalf("localized headline missing: %q", tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeyChatFailureAPIKeyBody)) {
		t.Fatalf("localized explanation missing: %q", tt.Texts())
	}
	if tt.HasText(raw) {
		t.Fatal("raw Host error is visible before the details toggle")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatFailureDetails)); err != nil {
		t.Fatalf("details toggle missing: %v", err)
	}
	if !tt.HasText(raw) {
		t.Fatalf("raw Host error lost after expanding: %q", tt.Texts())
	}
	if !tt.HasText("MISSING_API_KEY") {
		t.Fatalf("host code lost: %q", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyChatOpenModelSettings)); err != nil {
		t.Fatalf("model settings action missing: %v", err)
	}
	if opened != 1 {
		t.Fatalf("open model settings calls = %d", opened)
	}
}

// Without a wired action the card still explains the failure and keeps details.
func TestTurnFailureCardWithoutActionStaysReadable(t *testing.T) {
	raw := "HTTP 502 from upstream"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(12).Children(func() {
			paintTurnFailure(c, "m2", classifyTurnFailure(raw, ""), theme.Resolve("deepseek.light").Light,
				ui.RGB(20, 20, 20), ui.RGB(110, 110, 110), nil)
		})
	}, 760, 240)
	if !tt.HasText(i18n.T(i18n.KeyChatRunFailed)) {
		t.Fatalf("generic headline missing: %q", tt.Texts())
	}
	if tt.HasText(i18n.T(i18n.KeyChatOpenModelSettings)) {
		t.Fatal("generic failure must not advertise model settings")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatFailureDetails)); err != nil {
		t.Fatalf("details toggle missing: %v", err)
	}
	if !tt.HasText(raw) {
		t.Fatalf("raw error lost: %q", tt.Texts())
	}
}
