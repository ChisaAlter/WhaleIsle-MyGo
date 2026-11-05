package remote

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

// The installed gateway settings do not duplicate the launcher's anchored phone popup.
func phonePopupTester(m *Model) (*ui.Tester, *bool) {
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		anchor := ui.Button(c, "Phone launcher").Width(160).Height(28)
		PhonePopup(c, anchor, &open, *m)
	}, 400, 650)
	return tt, &open
}

func TestLauncherPhonePopupMountsPairingSurface(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	var qr bytes.Buffer
	if err := png.Encode(&qr, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	copies, opens := 0, 0
	active := "gateway"
	m := Model{
		Embedded: true, SettingsPage: "remote", GatewayProtocol: "chisacode-v2", ActiveTab: &active,
		GatewayEditor: &GatewayEditor{}, RemoteEnabled: true, RemoteMode: "lan", RemotePort: 3499,
		RemoteBindAddress: "127.0.0.1", RemoteRelayEndpoint: "ayase.cn:443",
		OnSaveGateway:    func(config.RemoteGatewayPreferences, config.RemoteGatewayPreferences) error { return nil },
		OnCopyConnectURL: func() { copies++ }, OnOpenConnectURL: func() { opens++ },
		PhoneRuntime: phoneremote.RuntimeState{Status: "ready", Listening: true, PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: time.Now().Add(10 * time.Minute), PairingImage: decodePNG(t, qr.Bytes()), DevicesLoaded: true},
	}
	gateway := ui.NewTester(func(c *ui.Context) { View(c, m) }, 780, 1000)
	if gateway.HasText("Phone connection") || gateway.HasText("Connect phone…") || gateway.HasText("Copy link") {
		t.Fatal("gateway duplicated the launcher popup", gateway.Texts())
	}
	page, open := phonePopupTester(&m)
	if !page.HasText(i18n.T(i18n.KeyRemotePhoneServiceReady)) || page.HasText(i18n.T(i18n.KeyRemotePhoneReady)) {
		t.Fatal("verified live runtime still claimed pairing was unverified", page.Texts())
	}
	if !page.HasText("Phone connection") || !page.HasText("Scan to connect") || !strings.Contains(strings.Join(page.Texts(), "\n"), "Valid for ") {
		t.Fatal("live QR popup absent", page.Texts())
	}
	if err := page.Click("Copy link"); err != nil {
		t.Fatal(err)
	}
	if err := page.Click("Open"); err != nil {
		t.Fatal(err)
	}
	if copies != 1 || opens != 1 {
		t.Fatal("pairing actions did not use the current owner", copies, opens)
	}
	if err := page.Click("Paired devices (0)"); err != nil {
		t.Fatal(err)
	}
	page.Key(0, ui.KeyEscape)
	if !*open || !page.HasText("Scan to connect") {
		t.Fatal("device dismissal closed the connection owner")
	}
	m.PhoneRuntime.Status = "starting"
	page.Frame()
	if !page.HasText(phoneReadinessLabel(phoneSnapshot(m).Readiness)) {
		t.Fatal("runtime readiness not updated", page.Texts())
	}
	_ = page.Click("Copy link")
	if copies != 1 {
		t.Fatal("starting runtime exposed a usable offer")
	}
	page.Key(0, ui.KeyEscape)
	if !*open {
		t.Fatal("busy connection popup dismissed")
	}
	m.PhoneRuntime.Status = "ready"
	page.Frame()
	page.Key(0, ui.KeyEscape)
	if *open {
		t.Fatal("idle connection popup did not dismiss")
	}
}

func TestPhonePopupNoUsableOffer(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for _, tc := range []struct {
		name, mode string
		snap       phoneremote.RuntimeState
		message    string
	}{
		{"lan not listening", "lan", phoneremote.RuntimeState{Status: "offline"}, "Waiting for a pairing code…"},
		{"relay without relay", "relay", phoneremote.RuntimeState{Status: "ready", Listening: true, PairingURL: "https://pair.invalid/#offer=x", ExpiresAt: time.Now().Add(time.Minute)}, "Waiting for a pairing code…"},
		{"expired", "lan", phoneremote.RuntimeState{Status: "ready", Listening: true, PairingURL: "http://127.0.0.1:3180/#offer=x", ExpiresAt: time.Now().Add(-time.Minute)}, "Pairing code expired. Refresh to continue."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copies, opens := 0, 0
			m := Model{GatewayProtocol: "chisacode-v2", RemoteEnabled: true, RemoteMode: tc.mode, PhoneRuntime: tc.snap, OnCopyConnectURL: func() { copies++ }, OnOpenConnectURL: func() { opens++ }}
			page, _ := phonePopupTester(&m)
			if !page.HasText(tc.message) {
				t.Fatal("missing no-offer state", page.Texts())
			}
			_ = page.Click("Copy link")
			_ = page.Click("Open")
			if copies != 0 || opens != 0 {
				t.Fatal("unusable offer executed a link action")
			}
		})
	}
}

func decodePNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}
