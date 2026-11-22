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
// Toolbar glyphs from ui-primitives icons/artwork.ts at regular (1px) weight.
var (
	pluginRefreshArtwork    = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M14.5001 8C14.5 9.28552 14.1188 10.5422 13.4045 11.611C12.6903 12.6799 11.6752 13.5129 10.4875 14.0049C9.29982 14.4968 7.99295 14.6255 6.73212 14.3747C5.4713 14.124 4.31314 13.505 3.4041 12.596C2.49514 11.687 1.87614 10.5288 1.62537 9.26798C1.37459 8.00716 1.50331 6.70028 1.99525 5.51261C2.48719 4.32494 3.32025 3.30981 4.3891 2.59557C5.45795 1.88134 6.71458 1.50008 8.0001 1.5C9.9001 1.5 11.7001 2.3 13.0001 3.6L14.5001 5.1" stroke="currentColor"/><path d="M14.4999 1.5V5.1H10.8999" stroke="currentColor"/></svg>`))
	pluginPlusArtwork       = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M8 2V14" stroke="currentColor"/><path d="M2 8H14" stroke="currentColor"/></svg>`))
	pluginChevronDownArtwork = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1"><path d="M4 6L7.29289 9.29289C7.68342 9.68342 8.31658 9.68342 8.70711 9.29289L12 6" stroke="currentColor"/></svg>`))
)

// pluginToolbarSpec is the page toolbar's .addButton recipe: 32 DIP, 13/20
// text, 0 12 padding on the md radius (PluginManagerPage.module.css:610-618).
func pluginToolbarSpec(variant widgets.ButtonVariant) widgets.ButtonSpec {
	return widgets.ButtonSpec{Variant: variant, Size: widgets.ButtonMedium, Height: 32, PadX: 12, FontSize: 13, LineHeight: 20}
}

// pluginToolbarIcon draws a leading glyph at the toolbar's 13 DIP icon size.
func pluginToolbarIcon(icon *ui.SVG, ink ui.Color) func(*ui.Context) {
	return func(c *ui.Context) { ui.Icon(c, icon).Size(13, 13).TextColor(ink) }
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
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	primaryInk := theme.Color(colors.ButtonFG)
	md := theme.ContextRadii(c).MD
	// .addGroup joins the two primaries: .addPrimary clears its right corners,
	// .addMore is a 32-wide icon square whose left corners clear against a 0.5
	// label-primary-foreground-30% seam (PluginManagerPage.module.css:620-637).
	ui.Row(c.Key("plugin-add-group")).Gap(0).AlignItems(ui.Stretch).Children(func() {
		spec := pluginToolbarSpec(widgets.ButtonPrimary)
		spec.Disabled = !s.ManagementReady()
		spec.Leading = pluginToolbarIcon(pluginPlusArtwork, primaryInk)
		if widgets.Button(c, colors, copyText("添加插件", "Add plugin"), spec).Radius(md, 0, 0, md).Clicked() {
			a.OpenInstall()
		}
		ui.Box(c).Width(0.5).Background(primaryInk.Alpha(.3))
		triggerSpec := spec
		triggerSpec.Leading = pluginToolbarIcon(pluginChevronDownArtwork, primaryInk)
		trigger := widgets.Button(c, colors, "", triggerSpec).Label(copyText("选择添加方式", "Choose add method")).Radius(0, md, md, 0).Width(32).Padding(0)
		opened := ui.Local(trigger, "add-menu", func() bool { return false })
		if trigger.Clicked() {
			*opened = !*opened
			if *opened && m.OnOpenAddMenu != nil {
				m.OnOpenAddMenu()
			}
		}
		w, _ := c.Size()
		ui.PopoverBase(c.Key("plugin-add-menu"), trigger, opened, func(panel ui.Element) {
			panel = chrome.PopoverReveal(c, chrome.MenuSurface(panel.Width(max(0, min(float32(320), w-48))).Padding(4).Gap(2).Role(ui.RoleMenu), c.Theme().Surface, c.Theme().Border, 16))
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
				entry(copyText("安装第三方插件", "Install existing plugin"), copyText("输入插件的包名、GitHub 仓库地址或本地目录路径。", "Enter an npm package, Git repository, or local path."), true, a.OpenInstall)
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
			})
		})
	})
}

// pluginMarketURL is the official dsh plugin marketplace the page links to now
// that the local discovery catalog is out of scope.
const pluginMarketURL = "https://awesome-dsh-plugin.com"
