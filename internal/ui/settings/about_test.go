package settings

import (
	"errors"
	"github.com/egoist/mygo/ui"
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

func TestAboutPendingActionsAndErrorReceipts(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	checks, homes, reads := 0, 0, 0
	a := &AboutAccess{AppName: "Fixture", Version: "1.2.3", Loading: true, UpdateBusy: true, HomeBusy: true, Err: "synthetic metadata error", UpdateStatus: "synthetic update pending", HomePath: "fixture-runtime", OnCheckUpdates: func() { checks++ }, OnRevealHome: func() error { homes++; return errors.New("synthetic reveal denied") }, ReadCredentialStorage: func() credentials.StorageStatus { reads++; return credentials.StorageStatus{Mode: "unknown"} }, OnOpenLink: func(string) error { return errors.New("synthetic link denied") }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() { aboutPane(c, Model{About: a}, c.Theme().Text, c.Theme().TextMuted) })
	}, 620, 1000)
	if checks != 1 || reads != 1 || !tt.HasText("synthetic metadata error") || !tt.HasText("synthetic update pending") {
		t.Fatal("initial metadata/check/error receipts missing")
	}
	if err := tt.Click("Open runtime directory"); err != nil {
		t.Fatal(err)
	}
	if homes != 0 {
		t.Fatal("pending reveal remained enabled")
	}
	tt.Frame()
	if checks != 1 || reads != 1 {
		t.Fatal("render repeated mount-only I/O")
	}
	a.Loading, a.UpdateBusy, a.HomeBusy = false, false, false
	tt.Frame()
	if err := tt.Click("Open runtime directory"); err != nil {
		t.Fatal(err)
	}
	if homes != 1 || !tt.HasText("synthetic reveal denied") {
		t.Fatal("reveal error hidden")
	}
	if err := tt.Click("Releases"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("synthetic link denied") {
		t.Fatal("link error hidden")
	}
	a.OnRevealHome = func() error { homes++; return nil }
	if err := tt.Click("Open runtime directory"); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("synthetic reveal denied") {
		t.Fatal("successful retry retained stale reveal error")
	}
}
