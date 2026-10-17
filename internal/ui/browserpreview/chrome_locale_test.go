package browserpreview

import (
	"testing"
	"whaleisle/internal/ui/i18n"
)

func TestBrowserChromeLanguageSwitch(t *testing.T) {
	old := i18n.Locale()
	defer i18n.SetLocale(old)
	i18n.SetLocale("en")
	if browserChromeLabel("刷新", "Reload") != "Reload" {
		t.Fatal("English toolbar ignored")
	}
	i18n.SetLocale("zh")
	if browserChromeLabel("刷新", "Reload") != "刷新" {
		t.Fatal("Chinese toolbar ignored")
	}
}
