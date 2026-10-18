package settings

import (
	"strings"
	"testing"
	"whaleisle/internal/credentials"
	"whaleisle/internal/ui/i18n"
)

func TestAboutStorageWarningsAndLinks(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	text := credentialStorageDescription(credentials.StorageStatus{Mode: "plaintext", SupportedPlatform: true})
	if !strings.Contains(text, "Warning") || !strings.Contains(text, "plaintext") {
		t.Fatal(text)
	}
	text = credentialStorageDescription(credentials.StorageStatus{Mode: "encrypted", SupportedPlatform: true})
	if !strings.Contains(text, "not been verified") {
		t.Fatal("unverified OS key advertised as available")
	}
	links := aboutLinks()
	if len(links) != 2 || links[0].URL != "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases" || links[1].URL != "https://github.com/deepseek-ai/deepseek-harness" {
		t.Fatal(links)
	}
}
