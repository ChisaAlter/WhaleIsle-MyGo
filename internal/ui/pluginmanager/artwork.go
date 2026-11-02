package pluginmanager

import (
	"embed"
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	marketui "whaleisle/internal/ui/market"
	"whaleisle/internal/ui/svgimage"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
	source := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(data)
	outer := ui.Column(c).Key("plugin-preview/" + name).FillWidth().MinWidth(0).Gap(6)
	attempt := ui.Local(outer, "attempt", func() int { return 0 })
	outer.Children(func() {
		// The image keeps its decode with its element: a new attempt asks for a
		// new element, and so for a fresh decode, while the skeleton stands in
		// for the picture — loading or failed — until one arrives.
		try := ui.Column(c).Key("plugin-preview-try/" + name + "/" + strconv.Itoa(*attempt)).FillWidth().MinWidth(0)
		try.Children(func() {
			if marketui.PluginImage(c, source, 640, 0, owner) {
				return
			}
			previewSkeleton(c, func() { *attempt++ })
		})
	})
}

// previewSkeleton stands in for artwork that is still decoding or could not be:
// the shape of the preview, and a control that asks for it again. Nothing here
// is a dead end, so an unavailable image never leaves a sentence with no action.
func previewSkeleton(c *ui.Context, retry func()) {
	t := c.Theme()
	ui.Box(c).FillWidth().MinWidth(0).Height(160).Radius(10).
		Background(t.Surface).Border(.5, t.Border).Children(func() {
		ui.Column(c).Fill().Center().Gap(10).Children(func() {
			ui.Box(c).Width(180).Height(12).Radius(6).Background(t.Border.Alpha(.6))
			ui.Box(c).Width(120).Height(10).Radius(5).Background(t.Border.Alpha(.45))
			if ui.Button(c, i18n.T(i18n.KeyPluginPreviewRetry)).Clicked() {
				retry()
			}
		})
	})
}

func addPluginMenu(c *ui.Context, m Model, s manager.State, a manager.Actions) {
	ui.Row(c.Key("plugin-add-group")).Gap(0).Children(func() {
		if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), copyText("添加插件", "Add plugin"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: !s.ManagementReady()}).Clicked() {
			a.OpenInstall()
		}
		trigger := ui.Button(c, "⌄").Label(copyText("选择添加方式", "Choose add method")).Disabled(!s.ManagementReady())
		opened := ui.Local(trigger, "add-menu", func() bool { return false })
		if trigger.Clicked() {
			*opened = !*opened
			if *opened && m.OnOpenAddMenu != nil {
				m.OnOpenAddMenu()
			}
		}
		w, _ := c.Size()
		ui.PopoverBase(c.Key("plugin-add-menu"), trigger, opened, func(panel ui.Element) {
			panel = chrome.PopoverReveal(c, chrome.MenuSurface(panel.Width(max(0, min(float32(320), w-48))).Padding(6).Gap(2).Role(ui.RoleMenu), c.Theme().Background, c.Theme().Border, 12))
			if m.OnNativeOverlay != nil {
				m.OnNativeOverlay(true)
			}
			panel.Children(func() {
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
