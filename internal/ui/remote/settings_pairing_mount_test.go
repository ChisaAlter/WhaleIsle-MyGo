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

// The 远程 page must mount the pairing surface: the QR/status/actions block was
// previously unreachable (viewPhoneSection had no production caller), so a user
// could never scan the code the gateway produced.
func TestRemotePageMountsPairingSurface(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	var qr bytes.Buffer
	if err := png.Encode(&qr, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(10 * time.Minute)
	copies, opens := 0, 0
	active := "gateway"
	m := Model{
		Embedded:            true,
		SettingsPage:        "remote",
		GatewayProtocol:     "chisacode-v2",
		ActiveTab:           &active,
		GatewayEditor:       &GatewayEditor{},
		RemoteEnabled:       true,
		RemoteMode:          "lan",
		RemotePort:          3499,
		RemoteBindAddress:   "127.0.0.1",
		RemoteRelayEndpoint: "ayase.cn:443",
		OnSaveGateway:       func(config.RemoteGatewayPreferences, config.RemoteGatewayPreferences) error { return nil },
		OnCopyConnectURL:    func() { copies++ },
		OnOpenConnectURL:    func() { opens++ },
		PhoneRuntime: phoneremote.RuntimeState{
			Status: "ready", Listening: true, RelayConnected: false,
			PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: now,
			PairingImage: decodePNG(t, qr.Bytes()), DevicesLoaded: true,
		},
	}
	page := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Gap(12).Children(func() { View(c, m) })
	}, 780, 1600)
	text := strings.Join(page.Texts(), "\n")
	if !strings.Contains(text, "offer v2") || !strings.Contains(text, "ready") {
		t.Fatalf("pairing status missing: %v", page.Texts())
	}
	if !page.HasText(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)) || !page.HasText(i18n.T(i18n.KeyMiscRemoteOpenPairingLink)) {
		t.Fatalf("pairing actions missing: %v", page.Texts())
	}
	if page.HasText(i18n.T(i18n.KeyMiscRemoteNoUsableOfferEnablingStartsTheInst)) {
		t.Fatalf("no-offer copy shown with a usable offer: %v", page.Texts())
	}
	if !page.HasText(i18n.T(i18n.KeyMiscRemotePairingLinkValid) + now.Local().Format("15:04:05")) {
		t.Fatalf("expiry line missing: %v", page.Texts())
	}
	if err := page.Click(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)); err != nil {
		t.Fatal(err)
	}
	if copies != 1 {
		t.Fatalf("copy action ran %d times", copies)
	}
	if err := page.Click(i18n.T(i18n.KeyMiscRemoteOpenPairingLink)); err != nil {
		t.Fatal(err)
	}
	if opens != 1 {
		t.Fatalf("open action ran %d times", opens)
	}

	// The status line is the live runtime snapshot, not a cached action message.
	m.PhoneRuntime.Status = "starting"
	page = ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Gap(12).Children(func() { View(c, m) })
	}, 780, 1600)
	text = strings.Join(page.Texts(), "\n")
	if !strings.Contains(text, "offer v2") || !strings.Contains(text, "starting") {
		t.Fatalf("status line is not live: %v", page.Texts())
	}
	if page.HasText(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)) {
		t.Fatalf("actions shown while starting: %v", page.Texts())
	}
}

func TestRemotePagePairingSurfaceNoOffer(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	for _, tc := range []struct {
		name string
		mode string
		snap phoneremote.RuntimeState
	}{
		{name: "lan not listening", mode: "lan", snap: phoneremote.RuntimeState{Status: "offline"}},
		{name: "relay without relay", mode: "relay", snap: phoneremote.RuntimeState{Status: "ready", Listening: true, PairingURL: "https://pair.invalid/#offer=x", ExpiresAt: time.Now().Add(time.Minute)}},
		{name: "expired", mode: "lan", snap: phoneremote.RuntimeState{Status: "ready", Listening: true, PairingURL: "http://127.0.0.1:3180/#offer=x", ExpiresAt: time.Now().Add(-time.Minute)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			active := "gateway"
			m := Model{
				Embedded: true, SettingsPage: "remote", GatewayProtocol: "chisacode-v2", ActiveTab: &active,
				GatewayEditor: &GatewayEditor{}, RemoteEnabled: true, RemoteMode: tc.mode,
				RemotePort: 3499, RemoteBindAddress: "127.0.0.1",
				OnSaveGateway: func(config.RemoteGatewayPreferences, config.RemoteGatewayPreferences) error { return nil },
				PhoneRuntime:  tc.snap,
			}
			page := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).FillWidth().Gap(12).Children(func() { View(c, m) })
			}, 780, 1200)
			if !page.HasText(i18n.T(i18n.KeyMiscRemoteNoUsableOfferEnablingStartsTheInst)) {
				t.Fatalf("missing no-offer copy: %v", page.Texts())
			}
			if page.HasText(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)) {
				t.Fatalf("offer actions shown without a usable offer: %v", page.Texts())
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
