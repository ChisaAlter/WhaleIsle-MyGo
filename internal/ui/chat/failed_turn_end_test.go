package chat

import (
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestFailedTurnEndShowsInlineReasonAndCode(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })

	msg := session.Message{
		ID: "end", Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Seq: 9,
		Body: "0",
		Raw:  []byte(`{"data":{"reason":{"kind":"error","error":{"message":"llm-deepseek: no API key for provider route \"deepseek-official\"","code":"MISSING_CREDENTIAL"}}}}`),
	}
	tt := messageTester(msg, 800, nil)
	for _, want := range []string{i18n.T(i18n.KeyChatRunFailed), `llm-deepseek: no API key for provider route "deepseek-official"`, "MISSING_CREDENTIAL"} {
		if !tt.HasText(want) {
			t.Fatalf("failed turn lost %q: %q", want, tt.Texts())
		}
	}
	if tt.HasText("Details") || tt.HasText("Open model settings") {
		t.Fatal("failed turn added actions absent from reference")
	}
}
