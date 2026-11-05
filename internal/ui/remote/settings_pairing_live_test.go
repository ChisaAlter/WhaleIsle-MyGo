package remote

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

// A ready LAN gateway must not warn about remoteAppBaseUrl: its phone page is the
// daemon's SPA, so the entry is derived from the live pairing URL and the
// readiness row reads as verified. The unverified wording stays for no runtime.
func TestRemotePageLiveEntryAndReadiness(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	future := time.Now().Add(10 * time.Minute)
	for _, tc := range []struct {
		name        string
		mode        string
		appBase     string
		snap        phoneremote.RuntimeState
		wantEntry   string
		wantREADME  string
		wantWarning bool
	}{
		{
			name: "lan ready derives the phone page",
			mode: "lan",
			snap: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: false,
				PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: future},
			wantEntry:  "http://127.0.0.1:3180",
			wantREADME: i18n.T(i18n.KeyRemotePhoneServiceReady),
		},
		{
			name: "relay ready uses the app base url",
			mode: "relay", appBase: "https://app.example.com/dshd",
			snap: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: true,
				PairingURL: "https://app.example.com/dshd/#offer=fixture", ExpiresAt: future},
			wantEntry:  "https://app.example.com/dshd",
			wantREADME: i18n.T(i18n.KeyRemotePhoneServiceReady),
		},
		{
			name:        "no runtime stays unverified",
			mode:        "lan",
			snap:        phoneremote.RuntimeState{Status: "offline"},
			wantREADME:  i18n.T(i18n.KeyRemotePhoneIncomplete),
			wantWarning: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			active := "gateway"
			m := Model{
				Embedded: true, SettingsPage: "remote", GatewayProtocol: "chisacode-v2", ActiveTab: &active,
				GatewayEditor: &GatewayEditor{}, RemoteEnabled: true, RemoteMode: tc.mode,
				RemotePort: 3499, RemoteBindAddress: "127.0.0.1", RemoteAppBaseURL: tc.appBase,
				OnSaveGateway: func(config.RemoteGatewayPreferences, config.RemoteGatewayPreferences) error { return nil },
				PhoneRuntime:  tc.snap,
			}
			page, _ := phonePopupTester(&m)
			text := strings.Join(page.Texts(), "\n")
			if tc.wantEntry != "" && phoneSnapshot(m).ConnectURL != tc.wantEntry {
				t.Fatalf("entry %q missing: %v", tc.wantEntry, page.Texts())
			}
			if !strings.Contains(text, tc.wantREADME) {
				t.Fatalf("readiness %q missing: %v", tc.wantREADME, page.Texts())
			}
			usable := offerUsable(m.PhoneRuntime, m.PhoneRuntime.Status, lanMode(m.RemoteMode))
			if usable == tc.wantWarning {
				t.Fatal("offer readiness diverged from runtime", page.Texts())
			}

		})
	}
}

// TestRemotePairingSurfaceSnapshot renders the mounted block with a live-shaped
// LAN snapshot so the QR image itself is available as visual evidence
// (SURFACES_GALLERY_DIR=… writes the PNG).
func TestRemotePairingSurfaceSnapshot(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	active := "gateway"
	m := Model{
		Embedded: true, SettingsPage: "remote", GatewayProtocol: "chisacode-v2", ActiveTab: &active,
		GatewayEditor: &GatewayEditor{}, RemoteEnabled: true, RemoteMode: "lan",
		RemotePort: 3499, RemoteBindAddress: "127.0.0.1",
		OnSaveGateway:    func(config.RemoteGatewayPreferences, config.RemoteGatewayPreferences) error { return nil },
		OnCopyConnectURL: func() {},
		OnOpenConnectURL: func() {},
		PhoneRuntime: phoneremote.RuntimeState{
			Status: "ready", Listening: true, RelayConnected: false,
			PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: time.Now().Add(10 * time.Minute),
			PairingImage: pairingFixturePNG(t),
		},
	}
	tt, _ := phonePopupTester(&m)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	saveSurfaceSnapshot(t, tt, "remote-panel-block-headless.png")
}
