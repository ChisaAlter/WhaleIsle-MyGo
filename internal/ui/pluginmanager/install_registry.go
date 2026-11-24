package pluginmanager

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/pagekit"
)

func registryName(registry, resolved *string) string {
	key := manager.RegistryKey(registry, resolved)
	switch key {
	case manager.OfficialNPMRegistry:
		return copyText("npm npm 官方源", "npm official registry")
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

// installOptionsRow holds the guide's toggle at the left and the registry
// trigger at the right. It returns the guide toggle, the registry anchor
// the choices float from, and whether each is open.
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
	ui.Row(c).FillWidth().Wrap().Gap(design.Sp(2)).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
		guide, guideOpen = installGuideToggle(c, i)
		registry, registryOpen = installRegistryToggle(c, i, chosen, disabled, a)
	})
	return guide, registry, guideOpen, registryOpen
}

// installChevronSettle is the reference's `transition: transform 160ms ease`
// on the disclosure chevron.
const installChevronSettle = 160 * time.Millisecond

type installChevronRotation struct {
	angle   float32
	from    float32
	to      float32
	since   time.Time
	started bool
}

// installChevronAngle eases a disclosure chevron between closed and open, so
// unfolding the guide turns the arrow over instead of swapping glyphs.
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

// installGuideToggle unfolds the install guide: a bare text link whose
// chevron turns over as the guide opens, as the reference's .guideToggle.
func installGuideToggle(c *ui.Context, i manager.InstallState) (ui.Element, bool) {
	t := design.For(c)
	control := ui.ButtonBase(c.Key("install-guide-toggle")).Shrink(0).Height(design.Sp(4.5)).Gap(design.Sp(1)).
		Padding(0).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	open := ui.Local(control, "guide-open", func() bool { return i.MirrorRecovery })
	ink := t.MutedForeground
	if control.Hovered() {
		ink = t.Foreground
	}
	label := guideToggleLabel(*open)
	control.Label(label).Expanded(*open)
	control.Children(func() {
		ui.Icon(c, coss.IconChevronDown.SVG()).Size(16, 16).Shrink(0).TextColor(ink).
			Rotate(installChevronAngle(c, control, *open))
		ui.Text(c, label).FontSize(design.TextXS.Size).TextColor(ink).SingleLine()
	})
	if control.Clicked() {
		*open = !*open
	}
	return control, *open
}

// guideToggleLabel is the guide toggle's two names.
func guideToggleLabel(open bool) string {
	if open {
		return copyText("收起引导", "Hide the guide")
	}
	return copyText("插件安装引导和示例", "Install guide and examples")
}

// installRegistryToggle names the registry the install asks first and
// opens the choices.
func installRegistryToggle(c *ui.Context, i manager.InstallState, chosen string, disabled bool, a manager.Actions) (ui.Element, bool) {
	open := i.RegistryOpen && i.Phase == "idle"
	chip := coss.Button(c, copyText("安装源：", "Registry: ")+chosen, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, TrailingIcon: coss.IconChevronsUpDown,
		Pressed: open, Disabled: disabled}).Label(copyText("安装源", "Registry") + " " + chosen).Expanded(open)
	if chip.Clicked() && !disabled {
		a.ToggleRegistryOptions()
	}
	return chip, open
}

// installGuideExample is the guide's one example: the npm package name, the
// hint under it and the example the row fills in.
func installGuideExample() (title, hint, example string) {
	return copyText("填入插件 npm 包名", "Enter the plugin's npm package name"),
		copyText("插件包名即 npm 包名（如 dsh-xxx 或 @作者/插件名），社区插件的 README 安装命令中 dsh plugin add 或 pnpm add 之后的部分。",
			"Use the npm package name, such as dsh-xxx or @author/plugin: the part after dsh plugin add or pnpm add in the plugin README."),
		"dsh-plugin-whale-pet"
}

// installGuidePanel is the unfolded guide: a muted card with the example
// and the button that fills it in.
func installGuidePanel(c *ui.Context, i manager.InstallState, a manager.Actions, open bool) {
	if !open {
		return
	}
	t := design.For(c)
	title, hint, example := installGuideExample()
	ui.Row(c).FillWidth().Gap(design.Sp(3)).AlignItems(ui.Center).Padding(design.Sp(3), design.Sp(4)).Radius(design.RadiusXL).Background(t.Muted).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(1)).Children(func() {
			pagekit.Strong(c, title).FillWidth()
			pagekit.Caption(c, hint).FillWidth()
			ui.Row(c).FillWidth().Gap(design.Sp(1)).Children(func() {
				pagekit.Caption(c, copyText("示例：", "Example: ")).Shrink(0)
				pagekit.Code(c, example)
			})
		})
		fill := copyText("填入示例", "Fill the example")
		if coss.Button(c, fill, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: i.Phase != "idle"}).Shrink(0).Label(fill + " " + example).Clicked() {
			a.EditInstall(example, i.Registry)
		}
	})
	if strings.TrimSpace(i.Spec) == "" && i.MirrorRecovery {
		pagekit.Muted(c, copyText("请用 npm 包名尝试其他安装来源。", "Try another installation source using the npm package name.")).FillWidth()
	}
}

// installRegistryPanel floats the registry choices from anchor: expanding
// them does not resize the install form, and Escape first dismisses this
// inner editing scope.
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
	c.Key("plugin-install-registry-options")
	coss.Popover(c, anchor, &open, coss.PopoverOptions{Align: coss.AlignEnd, Width: max(0, min(design.Sp(105), w-design.Sp(16)))}, func() {
		ui.Column(c).FillWidth().Gap(design.Sp(3)).Role(ui.RoleDialog).Label(copyText("安装源", "Installation registry")).Children(func() {
			choice := "custom"
			if i.RegistrySelection.Kind != "custom" {
				choice = "offered:"
				if r := i.RegistrySelection.Registry; r != nil {
					choice += *r
				}
			}
			was := choice
			coss.RadioGroup(c, copyText("安装源", "Installation registry"), func() {
				for _, registry := range manager.OfferedRegistries(i.RegistryInfo) {
					label := registryName(registry, resolved)
					value := "offered:"
					if registry != nil {
						value += *registry
						if label != *registry {
							label += " · " + *registry
						}
					}
					coss.Radio(c, &choice, value, label, coss.CheckOptions{Key: value})
				}
				coss.Radio(c, &choice, "custom", copyText("自定义地址", "Custom address"), coss.CheckOptions{Key: "custom"})
			})
			if choice != was {
				chooseRegistry(i, a, choice, *draft)
			}
			errText := ""
			if i.RegistryError {
				errText = copyText("请输入以 http:// 或 https:// 开头的地址", "Enter an address starting with http:// or https://")
			}
			coss.Field(c, coss.FieldOptions{Label: copyText("自定义安装源 URL", "Custom registry URL"), Error: errText,
				Description: copyText("填写内网或私有 npm 源地址。需要登录时，请将凭据保存在本机 ~/.npmrc。", "Enter an internal or private npm registry. Store required login credentials in this machine's ~/.npmrc.")}, func() {
				if coss.Input(c, draft, coss.InputOptions{Placeholder: "https://npm.example.com/"}).Changed() {
					a.ChooseRegistry(manager.RegistryChoice{Kind: "custom", URL: *draft})
				}
			})
		})
	})
	if shown && !open {
		a.ToggleRegistryOptions()
	}
}

// chooseRegistry applies a radio choice: an offered registry by its value,
// or the custom address with the draft.
func chooseRegistry(i manager.InstallState, a manager.Actions, choice, draft string) {
	if choice == "custom" {
		a.ChooseRegistry(manager.RegistryChoice{Kind: "custom", URL: draft})
		return
	}
	for _, registry := range manager.OfferedRegistries(i.RegistryInfo) {
		value := "offered:"
		if registry != nil {
			value += *registry
		}
		if value == choice {
			a.ChooseRegistry(manager.RegistryChoice{Kind: "offered", Registry: registry})
			return
		}
	}
}
