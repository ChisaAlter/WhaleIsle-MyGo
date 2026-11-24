package launcher

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestResolveLauncherPaletteFollowsThemeIDWithoutGlobalMutation(t *testing.T) {
	lightDeepSeek := resolveLauncherPalette("deepseek", false)
	darkDeepSeek := resolveLauncherPalette("deepseek", true)
	if lightDeepSeek.base == darkDeepSeek.base {
		t.Fatalf("light and dark palettes returned identical base color: %+v", lightDeepSeek.base)
	}
	// Find another built-in theme with a distinct accent or background from deepseek.
	var otherID string
	dsLight := theme.Resolve("deepseek").Active(false)
	for _, id := range theme.KnownIDs() {
		cand := theme.Resolve(id).Active(false)
		if cand.Accent != dsLight.Accent || cand.BG != dsLight.BG {
			otherID = id
			break
		}
	}
	if otherID == "" {
		t.Skip("no alternate theme with distinct accent/bg")
	}
	other := resolveLauncherPalette(otherID, false)
	if other.accent == lightDeepSeek.accent && other.base == lightDeepSeek.base {
		t.Fatalf("palette ignored ThemeID=%q and fell back to deepseek", otherID)
	}
}

func TestImportBusyPanelLocalizesZHAndEN(t *testing.T) {
	old := i18n.Locale()
	defer i18n.SetLocale(old)

	i18n.SetLocale(i18n.LocaleEN)
	tt := ui.NewTester(func(c *ui.Context) {
		importPanel(c, Model{ImportBusy: true, ImportSummary: "1/3", OnCancelImport: func() {}})
	}, 800, 500)
	if !tt.HasText("Cancel remaining import") {
		t.Fatalf("missing English cancel label: %v", tt.Texts())
	}
	if tt.HasText("取消后续导入") {
		t.Fatalf("hardcoded Chinese leaked in English locale: %v", tt.Texts())
	}

	i18n.SetLocale(i18n.LocaleZH)
	ttZH := ui.NewTester(func(c *ui.Context) {
		importPanel(c, Model{ImportBusy: true, ImportSummary: "1/3", OnCancelImport: func() {}})
	}, 800, 500)
	if !ttZH.HasText("取消后续导入") {
		t.Fatalf("missing Chinese cancel label: %v", ttZH.Texts())
	}
}
