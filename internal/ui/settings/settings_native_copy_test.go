package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestModelsPaneUsesInstalledIntroCopy pins the Models tab's one-line intro to
// the installed dictionary wording, which differs from the earlier local copy.
func TestModelsPaneUsesInstalledIntroCopy(t *testing.T) {
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })
	i18n.SetLocale(i18n.LocaleEN)
	tt := ui.NewTester(func(c *ui.Context) {
		modelsPane(c, Model{Models: &ModelsAccess{}}, ui.RGB(15, 17, 21), ui.RGB(129, 133, 140))
	}, 900, 600)
	if !tt.HasText("Enter your API keys to use models from the following providers.") {
		t.Fatalf("models intro is not the installed copy: %v", tt.Texts())
	}
	i18n.SetLocale(i18n.LocaleZH)
	tt.Frame()
	if !tt.HasText("填入各提供商的 API 密钥即可使用其模型。") {
		t.Fatalf("models intro lost its installed Chinese copy: %v", tt.Texts())
	}
}
