package pluginmanager

import (
	"embed"
	"encoding/base64"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/widgets"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
	marketui "whaleisle/internal/ui/market"
	"whaleisle/internal/ui/svgimage"
	"whaleisle/internal/ui/theme"
)

//go:embed assets/*.svg
var previews embed.FS

func metaIcon(meta *manager.Meta) string {
	if meta != nil {
		return meta.Icon
	}
	return ""
}
func builtInArtwork(name string) string {
	data, err := previews.ReadFile("assets/artwork-" + name + ".svg")
	if err != nil {
		return ""
	}
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data)
}
func itemArtwork(id string) string {
	switch id {
	case "shell":
		return builtInArtwork("terminal")
	case "agent-loop":
		return builtInArtwork("loop")
	case "subagent":
		return builtInArtwork("subagent")
	case "web-search":
		return searchArtwork()
	}
	return builtInArtwork("default")
}
func packagePreview(c *ui.Context, name string, owner *svgimage.Owner) {
	asset := ""
	switch name {
	case "@deepseek-ai/dsh-experimental-badge-skill-bundle":
		asset = "badge-skill-preview"
	case "@deepseek-ai/dsh-experimental-voice-input-bundle":
		asset = "voice-input-preview"
	default:
		return
	}
	if strings.HasPrefix(i18n.Locale(), "zh") {
		asset += ".zh"
	}
	data, err := previews.ReadFile("assets/" + asset + ".svg")
	if err != nil {
		return
	}
	if !marketui.PluginImage(c, "data:image/svg+xml;base64,"+base64.StdEncoding.EncodeToString(data), 640, 0, owner) {
		ui.Text(c, copyText("预览暂不可用。", "Preview unavailable.")).FillWidth().TextColor(c.Theme().TextMuted)
	}
}

func addPluginMenu(c *ui.Context, m Model, s manager.State, a manager.Actions) {
	ui.Row(c.Key("plugin-add-group")).Gap(0).Children(func() {
		if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), copyText("添加插件", "Add plugin"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: !s.ManagementReady()}).Clicked() {
			a.OpenInstall()
		}
		trigger := ui.PrimaryButton(c, "⌄").Label(copyText("选择添加方式", "Choose add method")).Disabled(!s.ManagementReady())
		opened := ui.Local(trigger, "add-menu", func() bool { return false })
		if trigger.Clicked() {
			*opened = !*opened
			if *opened && m.OnOpenAddMenu != nil {
				m.OnOpenAddMenu()
			}
		}
		w, _ := c.Size()
		ui.PopoverBase(c.Key("plugin-add-menu"), trigger, opened, func(panel ui.Element) {
			panel.Width(max(0, min(float32(320), w-48))).Padding(4).Gap(2).Role(ui.RoleMenu).Background(c.Theme().Background).Border(.5, c.Theme().Border).Radius(8).Children(func() {
				entryWithHint := func(title, description string, enabled bool, hint string, action func()) {
					row := ui.ButtonBase(c).FillWidth().Height(72).Padding(6, 12).Role(ui.RoleMenuItem).Label(title).Disabled(!enabled)
					if hint != "" {
						row.Tooltip(hint)
					}
					row.Children(func() {
						ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
							ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(14)).TextColor(c.Theme().Text)
							ui.Text(c, description).FillWidth().MaxLines(2).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
						})
					})
					if row.Clicked() {
						*opened = false
						action()
					}
				}
				entry := func(title, description string, enabled bool, action func()) {
					entryWithHint(title, description, enabled, "", action)
				}
				entry(copyText("安装已有插件", "Install existing plugin"), copyText("输入 npm 包名、Git 仓库或本地路径。", "Enter an npm package, Git repository, or local path."), true, a.OpenInstall)
				// The marketplace entry only opens the official site in a browser:
				// browsing and installing stay with the Host's plugin runtime.
				marketHint := ""
				if m.OnOpenURL == nil {
					marketHint = copyText("当前 Host 未提供浏览器打开能力。", "This Host cannot open a browser.")
				}
				entryWithHint(copyText("插件市场", "Plugin marketplace"),
					copyText("在 dsh 官方市场浏览插件（awesome-dsh-plugin.com）", "Browse plugins in the official dsh marketplace (awesome-dsh-plugin.com)"),
					m.OnOpenURL != nil, marketHint, func() {
						if m.OnOpenURL != nil {
							m.OnOpenURL(pluginMarketURL)
						}
					})
				if m.OnCreatePlugin != nil {
					entry(copyText("让 Agent 创建插件", "Let the agent create a plugin"), m.CreatePluginDescription, m.CreatePluginReady, m.OnCreatePlugin)
				}
				if m.AddActions != nil {
					m.AddActions(c)
				}
			})
		})
	})
}

// pluginMarketURL is the official dsh plugin marketplace the page links to now
// that the local discovery catalog is out of scope.
const pluginMarketURL = "https://awesome-dsh-plugin.com"
