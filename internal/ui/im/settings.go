package im

import (
	"context"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/imonboard"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// SettingsView is the dsh-im settings.remote.tab Channels contribution. The
// conversation history and message composer belong to the standalone IM panel.
func SettingsView(c *ui.Context, m Model) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted := chrome.Color(colors.FG), chrome.Color(colors.Muted)
	m.Embedded = true
	if m.Store == nil {
		ui.Text(c, "Connect to a Host to configure channels.").TextColor(muted)
		return
	}
	v := m.Store.Onboarding.Snapshot()
	office := m.Store.Office.Snapshot()
	if !v.Open && !office.Open {
		m.Store.Onboarding.Show("weixin")
		v = m.Store.Onboarding.Snapshot()
	}
	ui.Row(c.Key("channels-settings")).FillWidth().MinWidth(0).Gap(16).Children(func() {
		ui.Column(c).Width(112).Shrink(0).Gap(4).Children(func() {
			for _, id := range []string{"weixin", "feishu", "dingtalk", "wecom", "qq", "slack", "telegram", "discord", "whatsapp", "office"} {
				title := "AI Office"
				if id != "office" {
					title = imonboard.PlatformByID(id).Label
				}
				button := ui.Button(c, title).FillWidth().MinWidth(0).Height(34).FontSize(theme.InterfaceSize(12)).BorderWidth(0)
				selected := v.Channel == id && !office.Open || id == "office" && office.Open
				if selected {
					button = button.Background(chrome.Color(colors.Field)).TextColor(fg)
				} else {
					button = button.Background(ui.Transparent).TextColor(muted)
				}
				if button.Disabled(v.Busy || office.Busy).Clicked() {
					if id == "office" {
						m.Store.Onboarding.Hide()
						m.Store.Office.Show(context.Background())
					} else {
						m.Store.Office.Hide()
						m.Store.Onboarding.Show(id)
					}
				}
			}
		})
		ui.Box(c).Grow(1).MinWidth(0).Height(680).Children(func() {
			if office.Open {
				officeSettingsView(c, m)
			} else {
				onboardingView(c, m)
			}
		})
	})
}
