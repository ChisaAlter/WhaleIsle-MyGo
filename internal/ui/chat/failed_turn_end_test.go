package chat

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// A failed turn/end strip used to read only 「运行失败」and hid the host reason
// behind a disclosure. It now uses the same actionable card as a system failure.
func TestFailedTurnEndShowsTheFriendlyFailureCard(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	msg := session.Message{
		ID: "end", Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Seq: 9,
		Body: "0",
		Raw:  []byte(`{"data":{"reason":{"kind":"error","error":{"message":"llm-deepseek: no API key for provider route \"deepseek-official\"","code":"MISSING_CREDENTIAL"}}}}`),
	}
	opened := false
	tt := messageTester(msg, 800, func(m *Model) { m.OnOpenModelSettings = func() { opened = true } })
	if !tt.HasText(i18n.T(i18n.KeyChatFailureAPIKeyTitle)) {
		t.Fatalf("friendly API-key title missing: %q", tt.Texts())
	}
	if tt.HasText("llm-deepseek: no API key") {
		t.Fatal("raw host error is visible without expanding details")
	}
	if err := tt.Click(i18n.T(i18n.KeyChatFailureDetails)); err != nil {
		t.Fatalf("details action missing: %v", err)
	}
	if !tt.HasText("MISSING_CREDENTIAL") && !tt.HasText("llm-deepseek: no API key for provider route") {
		t.Fatalf("expanded details lost the host reason: %q", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyChatOpenModelSettings)); err != nil {
		t.Fatalf("open-model-settings action missing: %v", err)
	}
	if !opened {
		t.Fatal("open-model-settings did not fire")
	}
	_ = ui.Context{}
}
