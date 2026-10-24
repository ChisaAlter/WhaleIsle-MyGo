package remote

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/ui/i18n"
)

func pass9Locale(t *testing.T) {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(old) })
}

func pass9PhonePane(m Model) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		viewPhoneSection(c, m, c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Border, c.Theme().Accent)
	}, 900, 2000)
}

// TestPhoneOfferRequiresLiveRelay: a cached pairing URL is not evidence of a
// working relay. Without relayConnected the page must hide the copy/open
// actions and say that no usable offer exists.
func TestPhoneOfferRequiresLiveRelay(t *testing.T) {
	pass9Locale(t)
	copies := 0
	m := Model{
		ThemeID:             "deepseek",
		GatewayProtocol:     "chisacode-v2",
		RemoteEnabled:       true,
		RemoteMode:          "relay",
		RemotePort:          6767,
		RemoteBindAddress:   "127.0.0.1",
		RemoteRelayEndpoint: "ayase.cn:443",
		PhoneRuntime: phoneremote.RuntimeState{
			Status:         "ready",
			Listening:      true,
			RelayConnected: false,
			PairingURL:     "https://pair.example.invalid/#offer=fixture",
			ExpiresAt:      time.Now().Add(10 * time.Minute),
			DevicesLoaded:  true,
		},
		OnCopyConnectURL: func() { copies++ },
		OnOpenConnectURL: func() {},
	}
	offline := pass9PhonePane(m)
	if offline.HasText(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)) || offline.HasText(i18n.T(i18n.KeyMiscRemoteOpenPairingLink)) {
		t.Fatalf("offer actions shown without a relay: %v", offline.Texts())
	}
	if !offline.HasText(i18n.T(i18n.KeyMiscRemoteNoUsableOfferEnablingStartsTheInst)) {
		t.Fatalf("missing no-offer state: %v", offline.Texts())
	}
	if !offline.HasText(i18n.T(i18n.KeyRemotePhoneIncomplete)) {
		t.Fatalf("readiness claims ready without relay: %v", offline.Texts())
	}

	m.PhoneRuntime.RelayConnected = true
	live := pass9PhonePane(m)
	if !live.HasText(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)) || !live.HasText(i18n.T(i18n.KeyMiscRemoteOpenPairingLink)) {
		t.Fatalf("live offer actions missing: %v", live.Texts())
	}
	if !live.HasText(i18n.T(i18n.KeyRemotePhoneServiceReady)) {
		t.Fatalf("live readiness missing: %v", live.Texts())
	}
	if err := live.Click(i18n.T(i18n.KeyMiscRemoteCopyPairingLink)); err != nil {
		t.Fatal(err)
	}
	if copies != 1 {
		t.Fatalf("copy action ran %d times", copies)
	}
}

// TestGatewayRefreshShowsRuntimeActionState: rotating the pairing code is
// asynchronous, so the page must show the runtime's busy state and its real
// result instead of leaving the click silent.
func TestGatewayRefreshShowsRuntimeActionState(t *testing.T) {
	pass9Locale(t)
	pane := func(action phoneremote.DeviceActionState) *ui.Tester {
		m := pass6RemoteModel()
		m.PhoneRuntime.DeviceAction = action
		return ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Width(560).Children(func() { View(c, m) })
		}, 1000, 800)
	}
	if tt := pane(phoneremote.DeviceActionState{Operation: "rotate", Busy: true}); !tt.HasText("正在更新配对码…") {
		t.Fatalf("missing busy state: %v", tt.Texts())
	}
	if tt := pane(phoneremote.DeviceActionState{Operation: "rotate", Error: "fixture rotation rejected"}); !tt.HasText("fixture rotation rejected") {
		t.Fatalf("missing real action error: %v", tt.Texts())
	}
	if tt := pane(phoneremote.DeviceActionState{Operation: "rotate", Message: "Pairing offer renewed."}); !tt.HasText("Pairing offer renewed.") {
		t.Fatalf("missing real action result: %v", tt.Texts())
	}
}

// TestSSHSummaryNamesReadinessNotStubs pins the real readiness descriptions;
// the page must not claim a live tunnel or an unimplemented feature.
func TestSSHSummaryNamesReadinessNotStubs(t *testing.T) {
	pass9Locale(t)
	cases := map[sshremote.Readiness]string{
		sshremote.ReadinessReady:      i18n.KeyRemoteSSHSummaryReady,
		sshremote.ReadinessIncomplete: i18n.KeyRemoteSSHSummaryIncomplete,
		sshremote.ReadinessDisabled:   i18n.KeyRemoteSSHSummaryDisabled,
		sshremote.ReadinessInactive:   i18n.KeyRemoteSSHSummaryInactive,
	}
	for readiness, key := range cases {
		got := sshWorkspaceSummary(readiness)
		if got != i18n.T(key) {
			t.Fatalf("summary for %q = %q, want %q", readiness, got, i18n.T(key))
		}
		for _, banned := range []string{"未实现", "无实时", "stub"} {
			if strings.Contains(got, banned) {
				t.Fatalf("summary %q still contains %q", got, banned)
			}
		}
	}
	if got := sshReadinessLabel(sshremote.ReadinessInactive); got != i18n.T(i18n.KeyRemoteSSHStatusOff) {
		t.Fatalf("inactive label = %q", got)
	}
}
