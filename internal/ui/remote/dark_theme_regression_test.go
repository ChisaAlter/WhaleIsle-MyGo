package remote

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func TestRemoteSettingsDarkThemeButtonsAndBindSelect(t *testing.T) {
	pass6SetLocale(t, i18n.LocaleZH)
	accent := ui.RGB(122, 170, 255)
	onAccent := ui.RGB(15, 17, 21)

	// 1. Remote workspace page: Enable button is compact (not 900px full width)
	// and uses dark-theme primary accent instead of pure white.
	wsModel := Model{
		Embedded:               true,
		SettingsPage:           "workspace",
		ThemeID:                "deepseek.dark",
		SystemDark:             true,
		HarnessReady:           true,
		RemoteWorkspaceEnabled: false,
		OnToggleWorkspace:      func(bool) {},
	}
	ttWS := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		design.Bind(c, design.Dark, accent, onAccent)
		ui.Column(c).Fill().Background(ui.RGB(21, 21, 23)).Padding(20).Children(func() {
			View(c, wsModel)
		})
	}, 960, 500)
	enableBtn, ok := ttWS.Find("启用远程工作区")
	if !ok {
		t.Fatalf("missing Enable remote workspace button: %v", ttWS.Texts())
	}
	if enableBtn.W > 260 {
		t.Fatalf("Enable remote workspace stretched to full width: %+v", enableBtn)
	}
	enablePx := ttWS.Image().RGBAAt(int(enableBtn.X+4), int(enableBtn.Y+4))
	if enablePx.R > 240 && enablePx.G > 240 && enablePx.B > 240 {
		t.Fatalf("Enable remote workspace rendered pure white in dark mode: %+v", enablePx)
	}

	// 2. Phone remote page: Save & active Server buttons are not pure white,
	// and bind address uses coss.Select and switches cleanly.
	var saved []config.RemoteGatewayPreferences
	gwModel := pass6RemoteModel()
	gwModel.ThemeID = "deepseek.dark"
	gwModel.SystemDark = true
	gwModel.OnSaveGateway = func(_, next config.RemoteGatewayPreferences) error {
		saved = append(saved, next)
		gwModel.RemoteBindAddress = next.BindAddress
		return nil
	}
	ttGW := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		design.Bind(c, design.Dark, accent, onAccent)
		ui.Column(c).Fill().Background(ui.RGB(21, 21, 23)).Padding(20).Children(func() {
			ui.Column(c).Width(560).Children(func() { View(c, gwModel) })
		})
	}, 1000, 800)
	for _, name := range []string{i18n.T(KeyPass6RemoteSaveRelay), i18n.T(KeyPass6RemoteModeServer)} {
		box, ok := ttGW.Find(name)
		if !ok {
			t.Fatalf("missing %q control: %v", name, ttGW.Texts())
		}
		px := ttGW.Image().RGBAAt(int(box.X+4), int(box.Y+4))
		if px.R > 240 && px.G > 240 && px.B > 240 {
			t.Fatalf("%q rendered pure white in dark mode: %+v", name, px)
		}
	}

	if err := ttGW.Click(i18n.T(KeyPass6RemoteBindLabel)); err != nil {
		t.Fatalf("bind coss.Select trigger not clickable: %v", err)
	}
	if err := ttGW.Click("整个局域网"); err != nil {
		t.Fatalf("bind coss.Select option not clickable: %v", err)
	}
	if len(saved) != 1 || saved[0].BindAddress != "0.0.0.0" {
		t.Fatalf("bind select did not commit 0.0.0.0: %+v", saved)
	}
}
