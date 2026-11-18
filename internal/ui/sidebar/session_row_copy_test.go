package sidebar

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestSessionRowNeverShowsTheProviderEnvelope keeps the row's label on the same
// sentence as the failure row: a Session titled with the failure that ended its
// turn names a route and an environment variable, which is neither copy nor a
// title.
func TestSessionRowNeverShowsTheProviderEnvelope(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	want := "未配置模型凭据。请在 设置 → 模型 里填写 API Key 后重试。"
	for _, title := range []string{
		`llm-deepseek: no API key for provider route "deepseek-official"`,
		`llm-deepseek: no API key for provider route "deepseek-official"; store DEEPSEEK_API_KEY through the credentials service (the web Models page writes it), or export DEEPSEEK_API_KEY in the launching environment`,
		`llm-openai: no API key for provider route "openai-responses"`,
	} {
		tt := ui.NewTester(func(c *ui.Context) {
			m := Model{Snapshot: session.ViewState{SelectedID: "other"}}
			rowSession(c.Key("a"), session.Item{SessionID: "a", Title: title}, 0, m, ui.RGB(0, 0, 0), ui.RGB(120, 120, 120), ui.Transparent)
		}, 280, 120)
		painted := strings.Join(tt.Texts(), "\n")
		if strings.Contains(painted, "llm-") || strings.Contains(painted, "provider route") || strings.Contains(painted, "_API_KEY") {
			t.Fatalf("row painted the provider envelope: %q", painted)
		}
		if !strings.Contains(painted, want) {
			t.Fatalf("row = %q, want the sentence %q", painted, want)
		}
	}
}

// TestSessionRowKeepsTitlesAPersonWrote keeps the guard off ordinary titles,
// including one that only mentions a provider.
func TestSessionRowKeepsTitlesAPersonWrote(t *testing.T) {
	for _, title := range []string{"修复 llm-deepseek 重试", "DeepSeek 官方", "新会话", ""} {
		if got := sessionRowLabel(title); got != title {
			t.Fatalf("sessionRowLabel(%q) = %q", title, got)
		}
	}
}
