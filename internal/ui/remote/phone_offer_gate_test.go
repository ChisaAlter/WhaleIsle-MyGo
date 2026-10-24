package remote

import (
	"testing"
	"time"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

// The offer block follows the gateway transport, not the relay alone: native
// builds urls[].pairingUrl for both transports and only flags relayDown in relay
// mode (reference/electron/src/main/remote.js:344-360). A listening LAN gateway
// therefore shows its QR/link without a relay session; relay mode stays strict.
func TestPhoneOfferGateFollowsTransport(t *testing.T) {
	pass9Locale(t)
	future := time.Now().Add(10 * time.Minute)
	past := time.Now().Add(-time.Minute)
	for _, tc := range []struct {
		name       string
		mode       string
		runtime    phoneremote.RuntimeState
		wantOffer  bool
		wantNoOffr bool
	}{
		{
			name: "lan without relay shows the offer",
			mode: "lan",
			runtime: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: false,
				PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: future},
			wantOffer: true,
		},
		{
			name: "relay with relay shows the offer",
			mode: "relay",
			runtime: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: true,
				PairingURL: "https://pair.example.invalid/#offer=fixture", ExpiresAt: future},
			wantOffer: true,
		},
		{
			name: "relay without relay hides the offer",
			mode: "relay",
			runtime: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: false,
				PairingURL: "https://pair.example.invalid/#offer=fixture", ExpiresAt: future},
			wantNoOffr: true,
		},
		{
			name: "expired offer is hidden even on lan",
			mode: "lan",
			runtime: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: false,
				PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: past},
			wantNoOffr: true,
		},
		{
			name: "missing url hides the offer",
			mode: "lan",
			runtime: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: false,
				ExpiresAt: future},
			wantNoOffr: true,
		},
		{
			name: "not ready hides the offer",
			mode: "lan",
			runtime: phoneremote.RuntimeState{Status: "pairing-unavailable", Listening: true, RelayConnected: false,
				PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: future},
			wantNoOffr: true,
		},
		{
			name: "not listening hides the offer",
			mode: "lan",
			runtime: phoneremote.RuntimeState{Status: "ready", Listening: false, RelayConnected: false,
				PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: future},
			wantNoOffr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copies := 0
			m := Model{
				ThemeID:             "deepseek",
				GatewayProtocol:     "chisacode-v2",
				RemoteEnabled:       true,
				RemoteMode:          tc.mode,
				RemotePort:          6767,
				RemoteBindAddress:   "127.0.0.1",
				RemoteRelayEndpoint: "ayase.cn:443",
				PhoneRuntime:        tc.runtime,
				OnCopyConnectURL:    func() { copies++ },
				OnOpenConnectURL:    func() {},
			}
			pane := pass9PhonePane(m)
			copyLabel := i18n.T(i18n.KeyMiscRemoteCopyPairingLink)
			openLabel := i18n.T(i18n.KeyMiscRemoteOpenPairingLink)
			noOffer := i18n.T(i18n.KeyMiscRemoteNoUsableOfferEnablingStartsTheInst)
			if tc.wantOffer {
				if !pane.HasText(copyLabel) || !pane.HasText(openLabel) {
					t.Fatalf("offer actions missing: %v", pane.Texts())
				}
				if pane.HasText(noOffer) {
					t.Fatalf("no-offer copy shown with a usable offer: %v", pane.Texts())
				}
				if err := pane.Click(copyLabel); err != nil {
					t.Fatal(err)
				}
				if copies != 1 {
					t.Fatalf("copy action ran %d times", copies)
				}
				if !pane.HasText(i18n.T(i18n.KeyRemotePhoneReady)) {
					t.Fatalf("ready badge missing for %s: %v", tc.mode, pane.Texts())
				}
			} else {
				if pane.HasText(copyLabel) || pane.HasText(openLabel) {
					t.Fatalf("offer actions shown without a usable offer: %v", pane.Texts())
				}
				if !pane.HasText(noOffer) {
					t.Fatalf("missing no-offer state: %v", pane.Texts())
				}
			}
		})
	}
}

func TestOfferUsableTable(t *testing.T) {
	future := time.Now().Add(time.Minute)
	base := phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: false,
		PairingURL: "http://127.0.0.1:3180/#offer=fixture", ExpiresAt: future}
	for _, tc := range []struct {
		name  string
		snap  phoneremote.RuntimeState
		lan   bool
		ready bool
	}{
		{name: "lan no relay", snap: base, lan: true, ready: true},
		{name: "relay no relay", snap: base, lan: false, ready: false},
		{name: "relay cell", snap: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: true, PairingURL: base.PairingURL, ExpiresAt: future}, ready: true},
		{name: "expired", snap: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: true, PairingURL: base.PairingURL, ExpiresAt: time.Now().Add(-time.Second)}, ready: false},
		{name: "no url", snap: phoneremote.RuntimeState{Status: "ready", Listening: true, RelayConnected: true, ExpiresAt: future}, ready: false},
		{name: "listening false", snap: phoneremote.RuntimeState{Status: "ready", Listening: false, RelayConnected: true, PairingURL: base.PairingURL, ExpiresAt: future}, ready: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := offerUsable(tc.snap, tc.snap.Status, tc.lan); got != tc.ready {
				t.Fatalf("offerUsable = %v, want %v", got, tc.ready)
			}
		})
	}
}
