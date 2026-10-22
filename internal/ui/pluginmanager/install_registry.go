package pluginmanager

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func registryRadio(c *ui.Context, label string, selected bool, choose func()) {
	row := chrome.MenuRowColors(c, c.Theme().SurfaceHover, c.Theme().SurfacePressed).
		Label(label).Justify(ui.Start).Role(ui.RoleRadio).Selected(selected)
	row.Children(func() {
		ui.Text(c, label).Grow(1).SingleLine().FontSize(theme.InterfaceSize(13)).TextColor(c.Theme().Text)
	})
	if row.Clicked() {
		choose()
	}
}

func registryName(registry, resolved *string) string {
	key := manager.RegistryKey(registry, resolved)
	switch key {
	case manager.OfficialNPMRegistry:
		return copyText("npm 官方源", "npm official registry")
	case manager.NPMMirrorRegistry:
		return copyText("中国大陆镜像源", "Mainland China mirror")
	}
	if registry == nil {
		if resolved != nil {
			return *resolved
		}
		return copyText("pnpm 配置的源", "pnpm configured registry")
	}
	return *registry
}

// installChevronIcon is the reference's IconChevronDownOutlineRegular: the
// rounded V of a 16 DIP box, drawn at the regular 1 DIP stroke.
var installChevronIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true"><path d="M4 6L7.29289 9.29289C7.68342 9.68342 8.31658 9.68342 8.70711 9.29289L12 6" stroke="currentColor" stroke-width="1"/></svg>`))

// installChevronSettle is the reference's `transition: transform 160ms ease` on
// the disclosure chevron.
const installChevronSettle = 160 * time.Millisecond

type installChevronRotation struct {
	angle   float32
	from    float32
	to      float32
	since   time.Time
	started bool
}

// installChevronAngle eases a disclosure chevron between closed and open, so
// unfolding the guide or the registry choices turns the arrow over instead of
// swapping glyphs.
func installChevronAngle(c *ui.Context, owner ui.Element, open bool) float32 {
	state := ui.Local(owner, "install-chevron", func() installChevronRotation { return installChevronRotation{} })
	target := float32(0)
	if open {
		target = 180
	}
	if !state.started {
		state.started, state.angle, state.to = true, target, target
	}
	if state.to != target {
		state.from, state.to, state.since = state.angle, target, c.Now()
	}
	if state.angle != state.to {
		progress := float32(c.Now().Sub(state.since)) / float32(installChevronSettle)
		if progress >= 1 {
			state.angle = state.to
		} else {
			inverted := 1 - progress
			state.angle = state.from + (state.to-state.from)*(1-inverted*inverted*inverted)
			c.After(16 * time.Millisecond)
		}
	}
	return state.angle
}

// installDisclosureChevron draws the 16 DIP chevron of a disclosure control,
// turned by the control's open state.
func installDisclosureChevron(c *ui.Context, owner ui.Element, open bool, color ui.Color) {
	icon := ui.Icon(c, installChevronIcon).Size(16, 16).Shrink(0).TextColor(color)
	icon.Rotate(installChevronAngle(c, owner, open))
}

// installOptionsRow is the reference's `.optionsRow`: the guide's text toggle at
// the left and the registry chip at the right of one wrapping row. It returns
// the registry anchor the choices float from, and whether the guide is open.
func installOptionsRow(c *ui.Context, i manager.InstallState, a manager.Actions, disabled bool) (ui.Element, ui.Element, bool, bool) {
	var resolved *string
	if i.RegistryInfo != nil {
		resolved = i.RegistryInfo.Resolved
	}
	chosen := registryName(i.RegistrySelection.Registry, resolved)
	if i.RegistrySelection.Kind == "custom" {
		chosen = copyText("自定义地址", "Custom address")
	}
	var guide, registry ui.Element
	var guideOpen, registryOpen bool
	ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		guide, guideOpen = installGuideToggle(c, i, a)
		registry, registryOpen = installRegistryToggle(c, i, chosen, disabled, a)
	})
	return guide, registry, guideOpen, registryOpen
}

// installGuideToggle draws the reference's `.guideToggle`: a 16 DIP chevron and
// a 12.5 DIP label in the secondary color, with no border or fill, that takes
// the primary color under the pointer and renames itself while open.
func installGuideToggle(c *ui.Context, i manager.InstallState, a manager.Actions) (ui.Element, bool) {
	t := c.Theme()
	control := ui.Button(c, "").Shrink(0).Height(18).Padding(0).Gap(4).BorderWidth(0).
		Background(ui.Transparent).AlignItems(ui.Center)
	open := ui.Local(control, "guide-open", func() bool { return i.MirrorRecovery })
	color := t.TextMuted
	if control.Hovered() {
		color = t.Text
	}
	label := guideToggleLabel(*open)
	control.Label(label)
	control.Children(func() {
		installDisclosureChevron(c, control, *open, color)
		ui.Text(c, label).FontSize(theme.InterfaceSize(12.5)).TextColor(color).SingleLine()
	})
	if control.Clicked() {
		*open = !*open
	}
	return control, *open
}

// guideToggleLabel is the reference's two names for the guide toggle.
func guideToggleLabel(open bool) string {
	if open {
		return copyText("收起引导", "Hide the guide")
	}
	return copyText("插件安装引导和示例", "Install guide and examples")
}

// installRegistryToggle draws the reference's `.registryToggle`: a 28 DIP chip
// with a hairline border and a layer-2 fill that names the registry the install
// asks first, so the choice reads without unfolding.
func installRegistryToggle(c *ui.Context, i manager.InstallState, chosen string, disabled bool, a manager.Actions) (ui.Element, bool) {
	t := c.Theme()
	open := i.RegistryOpen && i.Phase == "idle"
	chip := ui.Button(c, "").Shrink(0).Height(28).Padding(0, 8, 0, 10).Gap(6).Radius(10).
		Border(.5, t.Border).Background(t.Surface).AlignItems(ui.Center).Disabled(disabled).
		Label(copyText("安装源", "Registry") + " " + chosen)
	if open {
		chip.Background(t.SurfaceHover)
	}
	labelColor := t.TextMuted
	if chip.Hovered() && !disabled {
		labelColor = t.Text
	}
	chip.Children(func() {
		ui.Text(c, copyText("安装源", "Registry")).FontSize(theme.InterfaceSize(12.5)).TextColor(labelColor).SingleLine()
		ui.Text(c, chosen).FontSize(theme.InterfaceSize(12.5)).FontWeight(500).TextColor(t.Text).SingleLine()
		installDisclosureChevron(c, chip, open, labelColor)
	})
	if chip.Clicked() && !disabled {
		a.ToggleRegistryOptions()
	}
	return chip, open
}

// installGuidePanel is the unfolded guide, below the options row.
func installGuidePanel(c *ui.Context, i manager.InstallState, a manager.Actions, open bool) {
	if !open {
		return
	}
	ui.Text(c, copyText("填入插件 npm 包名", "Enter the plugin's npm package name")).FontSize(theme.InterfaceSize(13)).TextColor(c.Theme().Text)
	ui.Text(c, copyText("插件包名即 npm 包名，如 dsh-xxx 或 @作者/插件名，是社区插件 README 安装命令中 dsh plugin add 或 pnpm add 之后的部分。", "Use the npm package name, such as dsh-xxx or @author/plugin: the part after dsh plugin add or pnpm add in the plugin README.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
	if ui.Button(c, "dsh-plugin-whale-pet").Disabled(i.Phase != "idle").Clicked() {
		a.EditInstall("dsh-plugin-whale-pet", i.Registry)
	}
	if strings.TrimSpace(i.Spec) == "" && i.MirrorRecovery {
		ui.Text(c, copyText("请用 npm 包名尝试其他安装来源。", "Try another installation source using the npm package name.")).FillWidth().TextColor(c.Theme().TextMuted)
	}
}

// Keep registry choices in an anchored popover: expanding them does not resize
// the install form, and Escape first dismisses this inner editing scope.
func installRegistryPanel(c *ui.Context, anchor ui.Element, i manager.InstallState, a manager.Actions) {
	var resolved *string
	if i.RegistryInfo != nil {
		resolved = i.RegistryInfo.Resolved
	}
	draft := ui.Local(anchor, "custom-registry-draft", func() string { return "" })
	if i.RegistrySelection.Kind == "custom" {
		*draft = i.RegistrySelection.URL
	}
	open := i.RegistryOpen && i.Phase == "idle"
	shown := open
	w, _ := c.Size()
	ui.PopoverBase(c.Key("plugin-install-registry-options"), anchor, &open, func(panel ui.Element) {
		panel.Width(max(0, min(float32(420), w-64))).Padding(12).Gap(6).Role(ui.RoleDialog).Label(copyText("安装源", "Installation registry")).Background(c.Theme().Background).Border(.5, c.Theme().Border).Radius(8).Children(func() {
			for _, registry := range manager.OfferedRegistries(i.RegistryInfo) {
				selected := i.RegistrySelection.Kind != "custom" && (registry == nil && i.RegistrySelection.Registry == nil || registry != nil && i.RegistrySelection.Registry != nil && *registry == *i.RegistrySelection.Registry)
				label := registryName(registry, resolved)
				if registry != nil && label != *registry {
					label += " · " + *registry
				}
				registryRadio(c, label, selected, func() {
					a.ChooseRegistry(manager.RegistryChoice{Kind: "offered", Registry: registry})
				})
			}
			registryRadio(c, copyText("自定义地址", "Custom address"), i.RegistrySelection.Kind == "custom", func() {
				a.ChooseRegistry(manager.RegistryChoice{Kind: "custom", URL: *draft})
			})
			if ui.TextInput(c, draft).FillWidth().Label(copyText("自定义安装源 URL", "Custom registry URL")).Placeholder("https://npm.example.com/").Changed() {
				a.ChooseRegistry(manager.RegistryChoice{Kind: "custom", URL: *draft})
			}
			if i.RegistryError {
				ui.Text(c, copyText("请输入以 http:// 或 https:// 开头的地址", "Enter an address starting with http:// or https://")).FillWidth().Role(ui.RoleStatus).TextColor(c.Theme().Danger)
			}
			ui.Text(c, copyText("填写内网或私有 npm 源地址。需要登录时，请将凭据保存在本机 ~/.npmrc。", "Enter an internal or private npm registry. Store required login credentials in this machine's ~/.npmrc.")).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
		})
	})
	if shown && !open {
		a.ToggleRegistryOptions()
	}
}
