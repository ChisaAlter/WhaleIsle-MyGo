package browserpreview

import (
	"strings"
	"testing"
	"whaleisle/internal/ui/i18n"
)

func TestBrowserToolEnglishSafetyCopy(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(old)
	title, hint, result := browserDevToolsCopy("darwin")
	if title != "Allow Safari inspection" || !strings.Contains(hint, "does not automatically") || !strings.Contains(result, "revoke") {
		t.Fatal(title, hint, result)
	}
	if !strings.Contains(browserSiteDataNotice(), "cannot be undone") || !strings.Contains(browserCookieNotice(), "signed out") {
		t.Fatal("lost clear-data consequences")
	}
}
