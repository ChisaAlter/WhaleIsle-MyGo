package main

import (
	"strings"
	"testing"

	"whaleisle/internal/product"
	"whaleisle/internal/ui/i18n"
)

// TestPanelWindowTitleCarriesOurIdentity covers AYA-36: a plugin panel window is
// this product, so its caption starts with this product's name. The page it hosts
// is the upstream Harness page, whose <title> is the other product's name, and the
// host pins this caption so the page cannot rename the window.
func TestPanelWindowTitleCarriesOurIdentity(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	got := panelWindowTitle("DeepSeek 应用接口")
	if !strings.HasPrefix(got, product.Name) || !strings.Contains(got, "DeepSeek 应用接口") {
		t.Fatalf("panel caption = %q, want %q first", got, product.Name)
	}
	if strings.Contains(got, "Whale Isle") {
		t.Fatalf("panel caption carries the upstream product name: %q", got)
	}
	if unnamed := panelWindowTitle("   "); !strings.HasPrefix(unnamed, product.Name) || strings.TrimSpace(strings.TrimPrefix(unnamed, product.Name)) == "—" {
		t.Fatalf("unnamed panel caption = %q", unnamed)
	}
}
