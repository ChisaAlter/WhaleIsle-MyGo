package chat

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestTerminalFailureUsesStableCodeAndKeepsUncodedReason(t *testing.T) {
	for _, tc := range []struct {
		code, raw, want string
	}{
		{"", "llm-deepseek: no API key for provider route deepseek-official", "llm-deepseek: no API key for provider route deepseek-official"},
		{"AUTH", "credential fragment must not reach Chat", i18n.T(i18n.KeyChatRetryAuthFailed)},
		{"ACCOUNT_SIGNED_OUT", "raw sign-out", i18n.T(i18n.KeyChatRetrySignedOut)},
		{"ACCOUNT_SIGN_IN_REQUIRED", "raw sign-in", i18n.T(i18n.KeyChatRetrySignIn)},
		{"QUOTA", "raw quota", i18n.T(i18n.KeyChatRetryQuota)},
		{"ACCOUNT_QUOTA", "raw account quota", i18n.T(i18n.KeyChatRetryQuota)},
		{"TIMEOUT", "upstream timeout", "upstream timeout"},
	} {
		raw, _ := json.Marshal(map[string]any{"data": map[string]any{"reason": map[string]any{"kind": "error", "error": map[string]any{"message": tc.raw, "code": tc.code}}}})
		msg := session.Message{Raw: raw}
		got, ok := terminalFailure(msg)
		if !ok || got.Message != tc.want || got.Code != tc.code {
			t.Fatalf("failure %q = %+v, %v", tc.code, got, ok)
		}
		if string(msg.Raw) != string(raw) || !strings.Contains(string(msg.Raw), tc.raw) {
			t.Fatal("presentation changed retained failure")
		}
	}
}

func TestTurnFailureIsInlineWithoutExtraActions(t *testing.T) {
	raw := "llm-deepseek: no API key for provider route deepseek-official"
	for _, width := range []int{280, 760} {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Padding(12).Children(func() {
				paintTurnFailure(c, turnFailure{Message: raw, Code: "MISSING_CREDENTIAL"}, theme.Resolve("deepseek.light").Light)
			})
		}, width, 320)
		for _, want := range []string{i18n.T(i18n.KeyChatRunFailed), raw, "MISSING_CREDENTIAL"} {
			if !tt.HasText(want) {
				t.Fatalf("width %d lost %q: %q", width, want, tt.Texts())
			}
		}
		for _, unwanted := range []string{"还没有配置 API Key", "No API key yet", "打开模型设置", "Open model settings", "详情", "Details"} {
			if tt.HasText(unwanted) {
				t.Fatalf("extra failure action/copy %q", unwanted)
			}
		}
		if _, ok := tt.Find("MISSING_CREDENTIAL"); !ok {
			t.Fatal("provider code was merged into the message")
		}
	}
}

func TestAccountSignOutIsATerminalNotice(t *testing.T) {
	msg := session.Message{Raw: json.RawMessage(`{"data":{"reason":{"kind":"aborted","reason":{"kind":"hook","reason":"deepseek-account/signed-out"}}}}`)}
	failure, ok := terminalFailure(msg)
	if !ok || failure.Code != "ACCOUNT_SIGNED_OUT" {
		t.Fatalf("sign-out notice lost: %+v", failure)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		paintTurnFailure(c, failure, theme.Resolve("deepseek.light").Light)
	}, 760, 160)
	if !tt.HasText(i18n.T(i18n.KeyChatAccountStopped)) || !tt.HasText(i18n.T(i18n.KeyChatRetrySignedOut)) {
		t.Fatalf("sign-out copy lost: %q", tt.Texts())
	}
	for _, reason := range []string{`{"kind":"complete"}`, `{"kind":"max-tokens"}`, `{"kind":"aborted","reason":{"kind":"user"}}`} {
		if _, ok := terminalFailure(session.Message{Raw: json.RawMessage(`{"data":{"reason":` + reason + `}}`)}); ok {
			t.Fatalf("nonfailure %s became a terminal error", reason)
		}
	}
}

func TestSplitTurnFailureKeepsLegacyHostCode(t *testing.T) {
	text, code := splitTurnFailure("llm-deepseek: no API key for provider route x MISSING_API_KEY")
	if strings.Contains(text, "MISSING_API_KEY") || code != "MISSING_API_KEY" {
		t.Fatalf("legacy failure = %q %q", text, code)
	}
}
