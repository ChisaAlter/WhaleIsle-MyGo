package chat

import (
	"strings"
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
	painted := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{i18n.T(i18n.KeyChatRunFailed), "No model credentials are set. Add an API key in Settings → Models, then try again.", "MISSING_CREDENTIAL"} {
		if !strings.Contains(painted, want) {
			t.Fatalf("failed turn lost %q: %q", want, painted)
		}
	}
	for _, unwanted := range []string{"llm-deepseek", "provider route", "DEEPSEEK_API_KEY"} {
		if strings.Contains(painted, unwanted) {
			t.Fatalf("failed turn painted %q: %q", unwanted, painted)
		}
	}
	if tt.HasText("Details") || tt.HasText("Open model settings") {
		t.Fatal("failed turn added actions absent from reference")
	}
}
