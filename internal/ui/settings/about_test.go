package settings

import (
	"errors"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"time"
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
	if err := tt.Click("Application data and diagnostics"); err != nil {
		t.Fatal(err)
	}
	if checks != 0 || reads != 1 || !tt.HasText("synthetic metadata error") || !tt.HasText("synthetic update pending") {
		t.Fatal("initial metadata/check/error receipts missing")
	}
	if err := tt.Click(i18n.T(KeyPass4OpenRuntime)); err != nil {
		t.Fatal(err)
	}
	if homes != 0 {
		t.Fatal("pending reveal remained enabled")
	}
	tt.Frame()
	if checks != 0 || reads != 1 {
		t.Fatal("render repeated mount-only I/O")
	}
	a.Loading, a.UpdateBusy, a.HomeBusy = false, false, false
	tt.Frame()
	// Receipts arrive as toasts; let the first ones time out so they do not cover the controls.
	tt.SetTime(time.Now().Add(time.Minute))
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeySettingsDesktopCheckUpdates)); err != nil {
		t.Fatal(err)
	}
	if checks != 1 {
		t.Fatal("explicit update check did not fire")
	}
	if err := tt.Click(i18n.T(KeyPass4OpenRuntime)); err != nil {
		t.Fatal(err)
	}
	if homes != 1 || !tt.HasText("synthetic reveal denied") {
		t.Fatal("reveal error hidden")
	}
	if err := tt.Click(i18n.T(KeyPass4GitHubReleases)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("synthetic link denied") {
		t.Fatal("link error hidden")
	}
	a.OnRevealHome = func() error { homes++; return nil }
	if err := tt.Click(i18n.T(KeyPass4OpenRuntime)); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("synthetic reveal denied") {
		t.Fatal("successful retry retained stale reveal error")
	}
}

func TestAboutUpdateCenterContextualPhasesAndActions(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	downloads, resumes, installs, centerOpens := 0, 0, 0, 0
	a := &AboutAccess{
		AppName:            "Whale Isle",
		Version:            "0.3.9",
		UpdatePhase:        "available",
		UpdateVersion:      "0.4.0",
		OnDownloadUpdate:   func() { downloads++ },
		OnResumeUpdate:     func() { resumes++ },
		OnInstallUpdate:    func() { installs++ },
		OnOpenUpdateCenter: func() { centerOpens++ },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			aboutPane(c, Model{About: a}, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 620, 1000)
	if err := tt.Click("Application data and diagnostics"); err != nil {
		t.Fatal(err)
	}

	if !tt.HasText("New version 0.4.0 is available") {
		t.Fatalf("missing available summary: %v", tt.Texts())
	}
	if err := tt.Click("Download update"); err != nil {
		t.Fatal(err)
	}
	if downloads != 1 {
		t.Fatalf("downloads = %d, want 1", downloads)
	}

	a.UpdatePhase = "downloading"
	a.UpdatePercent = 42
	tt.Frame()
	if !tt.HasText("Downloading 42%") {
		t.Fatalf("missing downloading summary: %v", tt.Texts())
	}
	a.UpdateBusy = true
	tt.Frame()
	if tt.HasText("Checking for updates…") || !tt.HasText("Downloading 42%") {
		t.Fatal("active download was mislabeled as an update check")
	}
	a.UpdateBusy = false

	a.UpdatePhase = "paused"
	a.UpdatePercent = 42
	tt.Frame()
	if !tt.HasText("Download paused 42%") {
		t.Fatalf("missing paused summary: %v", tt.Texts())
	}
	if err := tt.Click("Resume download"); err != nil {
		t.Fatal(err)
	}
	if resumes != 1 {
		t.Fatalf("resumes = %d, want 1", resumes)
	}

	a.UpdatePhase = "ready"
	tt.Frame()
	if !tt.HasText("Update ready to install") {
		t.Fatalf("missing ready summary: %v", tt.Texts())
	}
	if err := tt.Click("Install and restart"); err != nil {
		t.Fatal(err)
	}
	if installs != 1 {
		t.Fatalf("installs = %d, want 1", installs)
	}

	if err := tt.Click("Open Update Center"); err != nil {
		t.Fatal(err)
	}
	if centerOpens != 1 {
		t.Fatalf("centerOpens = %d, want 1", centerOpens)
	}
	a.UpdatePhase = "error"
	a.UpdateStatus = "synthetic persisted update failure"
	tt.Frame()
	if !tt.HasText(a.UpdateStatus) {
		t.Fatal("update error receipt was hidden")
	}
	if err := tt.Click("Resume download"); err != nil {
		t.Fatal(err)
	}
	if resumes != 2 {
		t.Fatal("error state lost its recovery action")
	}
}
