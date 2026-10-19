package remote

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/ui/i18n"
)

func TestGatewayBindLabelsDistinguishUnknownObservedAndStaleNIC(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	for _, check := range []struct {
		address  string
		detected []string
		want     string
	}{
		{"0.0.0.0", nil, "Whole LAN"}, {"127.0.0.1", []string{}, "This machine only"},
		{"10.0.0.9", nil, "10.0.0.9"}, {"10.0.0.9", []string{}, "10.0.0.9 · unavailable"},
		{"10.0.0.9", []string{"10.0.0.2"}, "10.0.0.9 · unavailable"}, {"10.0.0.9", []string{"10.0.0.9"}, "10.0.0.9"},
	} {
		if got := gatewayBindLabel(check.address, check.detected); got != check.want {
			t.Fatal("bind state mislabelled", check.address, check.detected, got, check.want)
		}
	}
}

func TestGatewayBindSelectorSavesRawAddressWithoutTextDrafts(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
	e := &GatewayEditor{Open: true, Draft: config.RemoteGatewayPreferences{RelayEndpoint: "unsubmitted-relay.invalid:9443", Port: "43180"}}
	m := Model{GatewayEditor: e, RemoteMode: "relay", RemotePort: 3180, RemoteBindAddress: "10.0.0.9", RemoteRelayEndpoint: "saved-relay.invalid:443", PhoneRuntime: phoneremote.RuntimeState{BindAddresses: []string{"10.0.0.2"}}}
	type change struct {
		before, after config.RemoteGatewayPreferences
	}
	changes := []change{}
	m.OnSaveGateway = func(before, after config.RemoteGatewayPreferences) error {
		changes = append(changes, change{before, after})
		m.RemoteBindAddress = after.BindAddress
		return nil
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() { gatewaySettingsFields(c, m, ui.RGB(20, 20, 20), ui.RGB(80, 80, 80)) })
	}, 800, 1000)
	if !tt.HasText("10.0.0.9 · unavailable") {
		t.Fatal("saved missing NIC was not retained visibly")
	}
	if err := tt.Click("Gateway bind address"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Whole LAN"); err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].before.BindAddress != "10.0.0.9" || changes[0].after.BindAddress != "0.0.0.0" {
		t.Fatal("bind label was submitted instead of raw IP", changes)
	}
	if changes[0].after.Port != "3180" || changes[0].after.RelayEndpoint != "saved-relay.invalid:443" || e.Draft.Port != "43180" || e.Draft.RelayEndpoint != "unsubmitted-relay.invalid:9443" {
		t.Fatal("bind selection submitted or erased pending text drafts", changes, e.Draft)
	}
	if err := tt.Click("Gateway bind address"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("10.0.0.2"); err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || changes[1].after.BindAddress != "10.0.0.2" {
		t.Fatal("detected NIC label did not map back to its raw address", changes)
	}
}
