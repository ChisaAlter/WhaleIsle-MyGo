package interaction

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	store "whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
)

// A queued reply is a delivered answer whose adoption has not happened yet, so
// the strip carries one sentence that says exactly that and never the wire code.
func TestQueuedReplyNoticeIsLocalizedAndCarriesNoWireCode(t *testing.T) {
	for _, tc := range []struct{ locale, want string }{
		{i18n.LocaleZH, "回答已排队，等主机接收后才会生效；不需要再次提交。"},
		{i18n.LocaleEN, "Your answer is queued; it takes effect once the host takes it up. No need to send it again."},
	} {
		i18n.SetLocale(tc.locale)
		s := store.New()
		t.Cleanup(s.Detach)
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{Store: s, Snapshot: store.ViewState{QuestionStatus: "queued"}, ThemeID: "deepseek.light"})
		}, 600, 900)
		tt.Frame()
		if !tt.HasText(tc.want) {
			t.Fatalf("locale %q: missing queued notice in %q", tc.locale, tt.Texts())
		}
		for _, text := range tt.Texts() {
			if strings.Contains(text, "REPLY_QUEUED") || strings.Contains(text, "already queued") {
				t.Fatalf("locale %q rendered wire wording %q", tc.locale, text)
			}
		}
	}
}
