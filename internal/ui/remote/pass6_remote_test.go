package remote

import (
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

// pass6SetLocale pins the page's language for one test and restores the
// previous catalog afterwards.
func pass6SetLocale(t *testing.T, locale string) {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(locale)
	t.Cleanup(func() { i18n.SetLocale(old) })
}

// pass6RemoteModel is the installed gateway's default state: relay mode,
// 6767, loopback only and the built-in TLS relay.
func pass6RemoteModel() Model {
	return Model{
		Embedded:            true,
		SettingsPage:        "remote",
		GatewayProtocol:     "chisacode-v2",
		GatewayEditor:       &GatewayEditor{},
		RemoteMode:          "relay",
		RemotePort:          6767,
		RemoteBindAddress:   remoteGatewayDefaultBind,
		RemoteRelayEndpoint: remoteGatewayDefaultRelay,
		RemoteRelayUseTLS:   true,
		PhoneRuntime:        phoneremote.RuntimeState{Status: "ready"},
		OnSaveGateway:       func(config.RemoteGatewayPreferences, config.RemoteGatewayPreferences) error { return nil },
		OnRotateToken:       func() error { return nil },
		ChannelsContent:     func(c *ui.Context) { ui.Text(c, "fixture channels") },
	}
}

// pass6RemoteTester renders the settings content column: the reference's 800
// wide dialog leaves 556 DIPs for the page body (800 − 180 rail − 16 gap − 48
// pane padding); 560 is the rounded figure other settings galleries use.
func pass6RemoteTester(m *Model) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			ui.Column(c).Width(560).Children(func() { View(c, *m) })
		})
	}, 1000, 800)
}

func pass6RequireTexts(t *testing.T, tt *ui.Tester, wants ...string) {
	t.Helper()
	texts := strings.Join(tt.Texts(), "\n")
	for _, want := range wants {
		if !strings.Contains(texts, want) {
			t.Fatalf("missing %q in page texts:\n%s", want, texts)
		}
	}
}

func pass6RequireBefore(t *testing.T, tt *ui.Tester, first, second string) {
	t.Helper()
	a, ok := tt.Find(first)
	if !ok {
		t.Fatalf("missing %q", first)
	}
	b, ok := tt.Find(second)
	if !ok {
		t.Fatalf("missing %q", second)
	}
	if a.Y >= b.Y {
		t.Fatalf("%q (y=%v) is not above %q (y=%v)", first, a.Y, second, b.Y)
	}
}

// TestPass6RemoteGatewayMatchesReference pins the reference's header, tabs and
// row order, and the status/token blocks the gateway page no longer shows.
func TestPass6RemoteGatewayMatchesReference(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	tt := pass6RemoteTester(&m)
	pass6RequireTexts(t, tt,
		i18n.T(KeyPass6RemoteTitle),
		i18n.T(KeyPass6RemoteIntro),
		i18n.T(KeyPass6RemoteTabGateway),
		i18n.T(KeyPass6RemoteTabChannels),
		i18n.T(KeyPass6RemoteRelay),
		i18n.T(KeyPass6RemoteRelayHint),
		i18n.T(KeyPass6RemoteMode),
		i18n.T(KeyPass6RemoteModeLAN),
		i18n.T(KeyPass6RemoteModeServer),
		i18n.T(KeyPass6RemotePort),
		i18n.T(KeyPass6RemotePortHint),
		i18n.T(KeyPass6RemoteBind),
		i18n.T(KeyPass6RemoteBindHintLocal),
		i18n.T(KeyPass6RemoteEncryption),
		i18n.T(KeyPass6RemoteEncryptionHint),
		i18n.T(KeyPass6RemoteRefresh),
		i18n.T(KeyPass6RemoteRefreshHint),
		i18n.T(KeyPass6RemoteRefreshAction),
	)
	texts := strings.Join(tt.Texts(), "\n")
	for _, removed := range []string{"连接状态", "中继访问令牌", "轮换连接令牌", "旧版配置清理"} {
		if strings.Contains(texts, removed) {
			t.Fatalf("gateway page still shows %q:\n%s", removed, texts)
		}
	}
	// The rows follow the installed order from top to bottom.
	pass6RequireBefore(t, tt, i18n.T(KeyPass6RemoteTitle), i18n.T(KeyPass6RemoteIntro))
	pass6RequireBefore(t, tt, i18n.T(KeyPass6RemoteIntro), i18n.T(KeyPass6RemoteTabGateway))
	pass6RequireBefore(t, tt, i18n.T(KeyPass6RemoteTabGateway), i18n.T(KeyPass6RemoteRelay))
	for _, pair := range [][2]string{
		{i18n.T(KeyPass6RemoteRelay), i18n.T(KeyPass6RemoteMode)},
		{i18n.T(KeyPass6RemoteMode), i18n.T(KeyPass6RemotePort)},
		{i18n.T(KeyPass6RemotePort), i18n.T(KeyPass6RemoteBind)},
		{i18n.T(KeyPass6RemoteBind), i18n.T(KeyPass6RemoteEncryption)},
		{i18n.T(KeyPass6RemoteEncryption), i18n.T(KeyPass6RemoteRefresh)},
	} {
		pass6RequireBefore(t, tt, pair[0], pair[1])
	}
	// The text fields are 220 wide, the bind pill is as wide as its label, and every control ends at the row's edge.
	var right float32
	for _, control := range []string{
		i18n.T(KeyPass6RemoteRelayLabel),
		i18n.T(KeyPass6RemotePortLabel),
		i18n.T(KeyPass6RemoteBindLabel),
	} {
		box, ok := tt.Find(control)
		if !ok {
			t.Fatalf("missing control %q", control)
		}
		minW, maxW := float32(220), float32(220)
		if control == i18n.T(KeyPass6RemoteBindLabel) {
			minW, maxW = 40, 200
		}
		if box.W < minW-.5 || box.W > maxW+.5 {
			t.Fatalf("%q is %v wide, want %v to %v", control, box.W, minW, maxW)
		}
		if right == 0 {
			right = box.X + box.W
		} else if box.X+box.W != right {
			t.Fatalf("%q does not share the control column edge: %v vs %v", control, box.X+box.W, right)
		}
	}
	refresh, ok := tt.Find(i18n.T(KeyPass6RemoteRefreshAction))
	if !ok {
		t.Fatalf("missing refresh action")
	}
	if refresh.X+refresh.W > right+1 {
		t.Fatalf("refresh action overflows the control column: %+v", refresh)
	}
}

// TestPass6RemoteDefaultsSaveTheirEffectiveValues proves the fields read the
// built-in relay and port when the model has not saved an explicit value yet.
func TestPass6RemoteDefaultsSaveTheirEffectiveValues(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	m.RemotePort = 0
	m.RemoteRelayEndpoint = ""
	saves := []struct {
		before, after config.RemoteGatewayPreferences
	}{}
	m.OnSaveGateway = func(before, after config.RemoteGatewayPreferences) error {
		saves = append(saves, struct {
			before, after config.RemoteGatewayPreferences
		}{before, after})
		return nil
	}
	tt := pass6RemoteTester(&m)
	if err := tt.Click(i18n.T(KeyPass6RemoteSaveRelay)); err != nil {
		t.Fatal(err)
	}
	if len(saves) != 1 {
		t.Fatalf("relay save calls = %d", len(saves))
	}
	if saves[0].after.RelayEndpoint != remoteGatewayDefaultRelay {
		t.Fatalf("relay default = %q, want %q", saves[0].after.RelayEndpoint, remoteGatewayDefaultRelay)
	}
	tt.Frame()
	if err := tt.Click(i18n.T(KeyPass6RemoteSavePort)); err != nil {
		t.Fatal(err)
	}
	if len(saves) != 2 || saves[1].after.Port != "6767" {
		t.Fatalf("port default = %+v", saves)
	}
	if saves[1].after.Mode != "relay" || saves[1].after.BindAddress != remoteGatewayDefaultBind {
		t.Fatalf("port save carried another field: %+v", saves[1])
	}
}

// TestPass6RemoteModeSwitchSavesImmediately matches the installed page: the
// segment commits the mode alone, without waiting for a save button.
func TestPass6RemoteModeSwitchSavesImmediately(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	saves := []struct {
		before, after config.RemoteGatewayPreferences
	}{}
	m.OnSaveGateway = func(before, after config.RemoteGatewayPreferences) error {
		saves = append(saves, struct {
			before, after config.RemoteGatewayPreferences
		}{before, after})
		return nil
	}
	tt := pass6RemoteTester(&m)
	if err := tt.Click(i18n.T(KeyPass6RemoteModeLAN)); err != nil {
		t.Fatal(err)
	}
	if len(saves) != 1 {
		t.Fatalf("mode save calls = %d", len(saves))
	}
	if saves[0].before.Mode != "relay" || saves[0].after.Mode != "lan" {
		t.Fatalf("mode change = %+v", saves[0])
	}
	if saves[0].after.Port != saves[0].before.Port ||
		saves[0].after.BindAddress != saves[0].before.BindAddress ||
		saves[0].after.RelayEndpoint != saves[0].before.RelayEndpoint {
		t.Fatalf("mode change carried another field: %+v", saves[0])
	}
}

// TestPass6RemotePortSaveSubmitsOnlyThePort edits the port and saves it, and
// checks the relay, mode and bind stay untouched.
func TestPass6RemotePortSaveSubmitsOnlyThePort(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	saves := []struct {
		before, after config.RemoteGatewayPreferences
	}{}
	m.OnSaveGateway = func(before, after config.RemoteGatewayPreferences) error {
		saves = append(saves, struct {
			before, after config.RemoteGatewayPreferences
		}{before, after})
		return nil
	}
	tt := pass6RemoteTester(&m)
	if err := tt.Click(i18n.T(KeyPass6RemotePortLabel)); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("43180")
	tt.Frame()
	if err := tt.Click(i18n.T(KeyPass6RemoteSavePort)); err != nil {
		t.Fatal(err)
	}
	if len(saves) != 1 {
		t.Fatalf("port save calls = %d", len(saves))
	}
	if saves[0].before.Port != "6767" || saves[0].after.Port != "43180" {
		t.Fatalf("port save = %+v", saves[0])
	}
	if saves[0].after.Mode != saves[0].before.Mode ||
		saves[0].after.BindAddress != saves[0].before.BindAddress ||
		saves[0].after.RelayEndpoint != saves[0].before.RelayEndpoint {
		t.Fatalf("port save carried another field: %+v", saves[0])
	}
}

// TestPass6RemoteRefreshRotatesThePairingCode wires the text action to the
// existing rotation callback and surfaces its failure.
func TestPass6RemoteRefreshRotatesThePairingCode(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	rotations := 0
	m.OnRotateToken = func() error { rotations++; return nil }
	tt := pass6RemoteTester(&m)
	if err := tt.Click(i18n.T(KeyPass6RemoteRefreshAction)); err != nil {
		t.Fatal(err)
	}
	if rotations != 1 {
		t.Fatalf("rotation calls = %d", rotations)
	}
	m.OnRotateToken = func() error { return errors.New("fixture rotation failed") }
	tt.Frame()
	if err := tt.Click(i18n.T(KeyPass6RemoteRefreshAction)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("fixture rotation failed") {
		t.Fatalf("rotation failure missing: %v", tt.Texts())
	}
}

// TestPass6RemoteLegacySectionOnlyWithStoredToken covers the muted section:
// absent without a legacy token, a disclosure with the clear action when one
// is stored, and placed between the relay and mode rows.
func TestPass6RemoteLegacySectionOnlyWithStoredToken(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	tt := pass6RemoteTester(&m)
	if tt.HasText(i18n.T(KeyPass6RemoteLegacySection)) {
		t.Fatal("legacy section shown without a legacy token")
	}

	m.LegacyRelayTokenSet = true
	var cleared []string
	m.OnSetRelayToken = func(value string) error {
		cleared = append(cleared, value)
		return nil
	}
	tt.Frame()
	if !tt.HasText(i18n.T(KeyPass6RemoteLegacySection)) {
		t.Fatalf("legacy section missing with a stored token: %v", tt.Texts())
	}
	pass6RequireBefore(t, tt, i18n.T(KeyPass6RemoteRelay), i18n.T(KeyPass6RemoteLegacySection))
	pass6RequireBefore(t, tt, i18n.T(KeyPass6RemoteLegacySection), i18n.T(KeyPass6RemoteMode))
	if err := tt.Click(i18n.T(KeyPass6RemoteLegacySection)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(i18n.T(KeyPass6RemoteLegacyClear)) {
		t.Fatalf("legacy clear action missing: %v", tt.Texts())
	}
	if err := tt.Click(i18n.T(KeyPass6RemoteLegacyClear)); err != nil {
		t.Fatal(err)
	}
	if len(cleared) != 1 || cleared[0] != "" {
		t.Fatalf("legacy clear calls = %v", cleared)
	}
}

// TestPass6RemoteChannelsTabRequiresContribution keeps the second tab honest:
// the gateway is the only tab until the IM plugin supplies a channels body.
func TestPass6RemoteChannelsTabRequiresContribution(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	m.ChannelsContent = nil
	tt := pass6RemoteTester(&m)
	if tt.HasText(i18n.T(KeyPass6RemoteTabChannels)) {
		t.Fatal("channels tab shown without a contribution")
	}
	m.ChannelsContent = func(c *ui.Context) { ui.Text(c, "fixture channels") }
	tt.Frame()
	if !tt.HasText(i18n.T(KeyPass6RemoteTabChannels)) {
		t.Fatalf("channels tab missing with a contribution: %v", tt.Texts())
	}
}

// TestPass6RemoteRestoreDefaultClearsRelay covers the text action beside the
// relay save: the stored endpoint returns to the built-in TLS relay.
func TestPass6RemoteRestoreDefaultClearsRelay(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	m.RemoteRelayEndpoint = "relay.example.com:443"
	m.RemoteRelayUseTLS = false
	saves := []struct {
		before, after config.RemoteGatewayPreferences
	}{}
	m.OnSaveGateway = func(before, after config.RemoteGatewayPreferences) error {
		saves = append(saves, struct {
			before, after config.RemoteGatewayPreferences
		}{before, after})
		return nil
	}
	tt := pass6RemoteTester(&m)
	if err := tt.Click(i18n.T(KeyPass6RemoteRestoreDefault)); err != nil {
		t.Fatal(err)
	}
	if len(saves) != 1 {
		t.Fatalf("restore calls = %d", len(saves))
	}
	if saves[0].before.RelayEndpoint != "relay.example.com:443" || saves[0].after.RelayEndpoint != "" {
		t.Fatalf("restore default = %+v", saves[0])
	}
	if !saves[0].after.RelayTLS {
		t.Fatalf("restored built-in relay is not TLS: %+v", saves[0].after)
	}
	if saves[0].after.Mode != saves[0].before.Mode || saves[0].after.Port != saves[0].before.Port {
		t.Fatalf("restore carried another field: %+v", saves[0])
	}
}

// TestPass6GalleryRemoteSettings exports the Chinese 1000×800 page for
// evidence/compare/ours-settings-remote.png. The gate keeps it out of normal
// runs; run with SETTINGS_GALLERY_DIR set to regenerate.
func TestPass6GalleryRemoteSettings(t *testing.T) {
	dir := os.Getenv("SETTINGS_GALLERY_DIR")
	if dir == "" {
		t.Skip("set SETTINGS_GALLERY_DIR to export ours-settings-remote.png")
	}
	pass6SetLocale(t, i18n.LocaleZH)
	m := pass6RemoteModel()
	m.LegacyRelayTokenSet = true
	m.OnSetRelayToken = func(string) error { return nil }
	m.OnRotateToken = func() error { return nil }
	m.ChannelsContent = func(c *ui.Context) {
		ui.Text(c, "fixture channels")
	}
	tt := pass6RemoteTester(&m)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, "ours-settings-remote.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
