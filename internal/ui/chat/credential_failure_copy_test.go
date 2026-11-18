package chat

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// missingCredentialSentence is the one sentence a person reads for this failure
// in both locales the shell ships.
var missingCredentialSentence = map[string]string{
	i18n.LocaleZH: "未配置模型凭据。请在 设置 → 模型 里填写 API Key 后重试。",
	i18n.LocaleEN: "No model credentials are set. Add an API key in Settings → Models, then try again.",
}

// providerFailureShapes are the ways the Host reports one cause: the route, the
// provider and the environment variable all change, the cause does not.
var providerFailureShapes = []struct{ code, raw string }{
	{"MISSING_CREDENTIAL", `llm-deepseek: no API key for provider route "deepseek-official"; store DEEPSEEK_API_KEY through the credentials service (the web Models page writes it), or export DEEPSEEK_API_KEY in the launching environment`},
	{"", `llm-deepseek: no API key for provider route "deepseek-official"`},
	{"", `llm-openai: no API key for provider route "openai-responses"`},
	{"MISSING_API_KEY", `llm-moonshot: no API key for provider route "moonshot"`},
}

// TestTurnFailureNeverShowsTheProviderEnvelope covers every surface that paints
// a failed turn: the transcript's failure row, the turn-end strip's detail and
// the legacy system row. All three read the same sentence, and the raw text only
// survives where it is retained for the log.
func TestTurnFailureNeverShowsTheProviderEnvelope(t *testing.T) {
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
			colors := theme.Resolve("deepseek.light").Light
			for _, shape := range providerFailureShapes {
				want := missingCredentialSentence[locale]
				envelope, _ := json.Marshal(map[string]any{"data": map[string]any{"reason": map[string]any{"kind": "error", "error": map[string]any{"message": shape.raw, "code": shape.code}}}})
				msg := session.Message{ID: "turn", Role: "status", Kind: session.MessageKindText, EventType: "turn/end", Seq: 3, Body: "0", Raw: envelope}

				failure, ok := terminalFailure(msg)
				if !ok || failure.Message != want {
					t.Fatalf("%q read as %+v, want %q", shape.raw, failure, want)
				}
				tt := ui.NewTester(func(c *ui.Context) {
					ui.Column(c).FillWidth().Padding(12).Children(func() {
						paintTurnFailure(c, failure, colors)
					})
				}, 760, 200)
				assertNoTransportCopy(t, tt.Texts(), shape.raw, shape.code)
				if painted := strings.Join(tt.Texts(), "\n"); !strings.Contains(painted, want) {
					t.Fatalf("failure row = %q, want %q", painted, want)
				}

				assertNoTransportCopy(t, []string{terminalReasonDetail(msg)}, shape.raw, shape.code)

				text, code := splitTurnFailure(shape.raw + " MISSING_CREDENTIAL")
				if code != "MISSING_CREDENTIAL" || text != want {
					t.Fatalf("legacy system row = %q %q, want %q", text, code, want)
				}
				assertNoTransportCopy(t, []string{text}, shape.raw, "MISSING_CREDENTIAL")

				if string(msg.Raw) != string(envelope) {
					t.Fatal("the retained failure changed")
				}
			}
		})
	}
}

// TestRetryAttemptNeverShowsTheProviderEnvelope keeps the retry row's attempt
// list on the same sentence: it reads the same Host failure as the failure row.
func TestRetryAttemptNeverShowsTheProviderEnvelope(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	attempt := session.RetryAttempt{RetryID: "r1", Retry: 1, MaxRetries: 3, DelayMs: 2000, Seq: 7, State: "scheduled",
		Failure: json.RawMessage(`{"message":"llm-deepseek: no API key for provider route \"deepseek-official\"","code":"MISSING_CREDENTIAL"}`)}
	expanded := map[string]bool{"retry::s1:r1:7": true}
	m := Model{Snapshot: session.ViewState{SelectedID: "s1"}, ThinkingExpanded: &expanded}
	msg := session.Message{ID: "retry", Seq: 7, Retry: &session.RetryStatus{Attempts: []session.RetryAttempt{attempt}}}
	tt := ui.NewTester(func(c *ui.Context) {
		RetryStatusView(c, m, msg, ui.RGB(0, 0, 0), ui.RGB(90, 90, 90))
	}, 760, 320)
	assertNoTransportCopy(t, tt.Texts(), `llm-deepseek: no API key for provider route "deepseek-official"`, "MISSING_CREDENTIAL")
	if want := missingCredentialSentence[i18n.LocaleZH]; !strings.Contains(strings.Join(tt.Texts(), "\n"), want) {
		t.Fatalf("retry row lost the sentence %q: %q", want, tt.Texts())
	}
}

// TestUnrecognisedFailureStillSpeaksASentence is the fallback rule: a failure
// this build cannot name gets the generic sentence, never its own wording.
func TestUnrecognisedFailureStillSpeaksASentence(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	message, code := displayFailure(json.RawMessage(`{"message":"gateway/definition-unavailable: strict definition withdrawn","code":"UNKNOWN"}`))
	if code != "UNKNOWN" || message == "" || strings.Contains(message, "gateway") {
		t.Fatalf("unrecognised failure = %q %q", message, code)
	}
	if message, code := displayFailure(json.RawMessage(`not json at all`)); code != "" || message == "" {
		t.Fatalf("unreadable failure = %q %q", message, code)
	}
}

var (
	envNamePattern = regexp.MustCompile(`\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*_API_KEY\b`)
	quotedPattern  = regexp.MustCompile(`"([^"]+)"`)
)

// assertsNoTransportCopy fails when the painted copy carries the transport's own
// words: the provider envelope, the route it names, or an environment variable.
// The stable code the Host also sends is not an environment variable, so the
// code the failure keeps is excluded from the search.
func assertNoTransportCopy(t *testing.T, texts []string, raw, code string) {
	t.Helper()
	forbidden := []string{"provider route"}
	for _, name := range envNamePattern.FindAllString(raw, -1) {
		if name != code {
			forbidden = append(forbidden, name)
		}
	}
	for _, route := range quotedPattern.FindAllStringSubmatch(raw, -1) {
		forbidden = append(forbidden, route[1])
	}
	for _, text := range texts {
		if strings.HasPrefix(strings.TrimSpace(text), "llm-") {
			t.Fatalf("painted the provider envelope: %q", text)
		}
		for _, word := range forbidden {
			if strings.Contains(text, word) {
				t.Fatalf("painted %q in %q", word, text)
			}
		}
	}
}
