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
	root := ui.Column(c.Key("channels-settings")).FillWidth().MinWidth(0)
	width := root.Bounds().W
	if width <= 0 {
		width, _ = c.Size()
	}
	root.Draw(func(_ *ui.Painter, bounds ui.Rect) {
		if bounds.W != width {
			c.Invalidate()
		}
	})
	windowWidth, _ := c.Size()
	narrow := windowWidth <= 840
	rail := func() {
		buttons := func() {
			for _, id := range []string{"weixin", "feishu", "dingtalk", "wecom", "qq", "slack", "telegram", "discord", "whatsapp", "office"} {
				title := "AI Office"
				if id != "office" {
					title = imonboard.PlatformByID(id).Label
				}
				button := ui.Button(c.Key("channel-"+id), "").Label(title).MinWidth(0).Height(48).Padding(8, 12).Border(1, chrome.Color(colors.Border)).Radius(12).Role(ui.RoleTab)
				if narrow {
					buttonWidth := max(float32(0), (width-8)/2)
					if windowWidth <= 560 {
						buttonWidth = width
					}
					button.Width(buttonWidth)
				} else {
					button.FillWidth()
				}
				selected := v.Channel == id && !office.Open || id == "office" && office.Open
				if selected {
					button = button.Background(chrome.Color(colors.Field)).TextColor(fg)
				} else {
					button = button.Background(ui.Transparent).TextColor(muted)
				}
				button.Selected(selected).Children(func() {
					ui.Row(c).FillWidth().MinWidth(0).Gap(10).AlignItems(ui.Center).Children(func() {
						channelRailLogo(c, id)
						ui.Column(c).Grow(1).MinWidth(0).Children(func() {
							ui.Text(c, title).FontSize(theme.InterfaceSize(14)).FontWeight(600).FillWidth().MinWidth(0).SingleLine().Ellipsis("…")
							if id == "office" {
								ui.Text(c, channelText("（实验功能）", "(Experimental)")).FontSize(theme.InterfaceSize(10)).TextColor(muted).SingleLine()
							}
						})
					})
				})
				if button.Disabled(office.Busy).Clicked() && !selected {
					if id == "office" {
						m.Store.Onboarding.Hide()
						m.Store.Office.Show(context.Background())
					} else {
						m.Store.Office.Hide()
						m.Store.Onboarding.Show(id)
					}
				}
			}
		}
		if narrow {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Role(ui.RoleTabList).Children(buttons)
		} else {
			ui.Scroll(c).Width(174).Height(520).Shrink(0).Padding(1, 4, 1, 1).Children(func() { ui.Column(c).FillWidth().Gap(8).Role(ui.RoleTabList).Children(buttons) })
		}
	}
	body := func() {
		column := ui.Column(c).MinWidth(0)
		if narrow {
			column.FillWidth()
		} else {
			column.Grow(1)
		}
		column.Children(func() {
			if office.Open {
				officeSettingsView(c, m)
			} else {
				onboardingView(c, m)
			}
		})
	}
	root.Children(func() {
		if narrow {
			rail()
			ui.Box(c).Height(18)
			body()
			return
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(24).AlignItems(ui.Start).Children(func() {
			rail()
			ui.Box(c).Width(1).Height(520).Shrink(0).Background(chrome.Color(colors.Border))
			body()
		})
	})
}
