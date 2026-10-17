package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestViewPromptModeLabels(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)

	draft := ""
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Snapshot: session.ViewState{PromptMode: session.PromptModeQueue},
			Draft:    &draft,
			OnPromptModeChange: func(session.PromptMode) {},
		})
	}, 800, 200)

	want := i18n.T(i18n.KeyComposerPermissionDefault)
	if !tt.HasText(want) {
		t.Fatalf("missing %q; texts = %q", want, tt.Texts())
	}
}
