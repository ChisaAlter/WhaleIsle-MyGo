package pluginmanager

import (
	"strings"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// The reference's install card ends in a fixed footer, not in the scrolling
// content: `PluginManagerPage.tsx:960-974` hands `.installFooter` to the Modal's
// `footer` prop, so the trust note and the primary action stay put while the
// examples above them scroll (`PluginManagerPage.module.css:518-522` scrolls
// `.installContent` alone). Its metrics: `.footer` pads 0 24, `.installFooter` is
// a 20 DIP column, `.installSafety` is a warn panel with 10/12 padding and a 12
// DIP radius, and `.wide` is a full-width 40 DIP primary
// (`PluginManagerPage.module.css:531-537, 637-661, 864-869`).
const (
	installFooterGap    = 20
	installFooterInset  = 24
	installSafetyRadius = 12
	installSafetyPadY   = 10
	installSafetyPadX   = 12
	installSafetyIconSz = 14
	installActionHeight = 40
)

// installWarnInk is --dsw-alias-state-warn-primary, the colour of the note's
// warning glyph.
var installWarnInk = ui.RGB(245, 158, 11)

// installSafetyIcon is the reference's IconWarningOutlineRegular: the circle and
// exclamation of the 16 DIP icon set, at the regular 1 DIP stroke.
var installSafetyIcon = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1" aria-hidden="true"><path d="M8 14.5C11.5899 14.5 14.5 11.5899 14.5 8C14.5 4.41015 11.5899 1.5 8 1.5C4.41015 1.5 1.5 4.41015 1.5 8C1.5 11.5899 4.41015 14.5 8 14.5Z"/><path d="M8 4.29199V9.79199"/><path d="M8 10.708V11.708"/></svg>`))

// installWarnFill is --dsw-alias-state-warn-tertiary: amber-100 in the light
// scheme and the same tint turned over in the dark one.
func installWarnFill(dark bool) ui.Color {
	if dark {
		return ui.RGB(39, 36, 31)
	}
	return ui.RGB(254, 245, 231)
}

// installSafetyInk is --dsw-alias-label-secondary, the note's text colour.
func installSafetyInk(dark bool) ui.Color {
	return ui.Hex(theme.SecondaryInk(dark))
}

// installSafetyCopy is the reference's two lines of trust guidance
// (`installGuideSafety` and `installUpgradeNotice`), the second about upgrades.
func installSafetyCopy() (safety, upgrade string) {
	return copyText("请确认插件来源可信。插件在本机以你的权限运行，来源不明的插件可能损坏 鲸屿 MyGo，或读取和泄露你的数据。",
			"Install only plugins you trust: they run with your permissions and can damage Whale Isle MyGo or leak your data."),
		copyText("插件安装后，暂不支持自动更新。若需升级，请先卸载再安装新版，后续版本会持续改善升级体验。",
			"Installed plugins do not update automatically yet. To upgrade a plugin, uninstall it and install the new version. Later releases will keep improving the upgrade experience.")
}

// installSafetyNote draws `.installSafety`: the warn panel's glyph above the
// note's first line, with the two 12/18 lines stacked beside it.
func installSafetyNote(c *ui.Context) {
	dark := c.Theme().Dark
	ink := installSafetyInk(dark)
	safety, upgrade := installSafetyCopy()
	note := ui.Row(c).FillWidth().Shrink(0).Padding(installSafetyPadY, installSafetyPadX).Gap(8).
		Radius(installSafetyRadius).AlignItems(ui.Start).Background(installWarnFill(dark))
	note.Children(func() {
		ui.Icon(c, installSafetyIcon).Size(installSafetyIconSz, installSafetyIconSz).Shrink(0).
			Margin(2, 0, 0, 0).TextColor(installWarnInk)
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, safety).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(ink)
			ui.Text(c, upgrade).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(18).TextColor(ink)
		})
	})
}

// installFooter draws the spec screen's fixed footer: the trust note, then the
// full-width primary action, `安装` or, while the spec is being checked, a
// disabled `正在检查…` with the running dot the reference puts in it
// (`PluginManagerPage.tsx:969-972`).
func installFooter(c *ui.Context, i manager.InstallState, a manager.Actions, colors theme.Colors, busy bool) {
	checking := i.Phase == "checking"
	label := copyText("安装", "Install")
	if checking {
		label = copyText("正在检查…", "Checking…")
	}
	spec := widgets.ButtonSpec{
		Variant:  widgets.ButtonPrimary,
		Size:     widgets.ButtonMedium,
		Disabled: busy || checking || strings.TrimSpace(i.Spec) == "",
	}
	if checking {
		spec.Leading = func(c *ui.Context) { widgets.StateDot(c, widgets.StateRunning) }
	}
	ui.Column(c).FillWidth().Shrink(0).Gap(installFooterGap).
		Padding(installFooterGap, installFooterInset, installFooterInset, installFooterInset).Children(func() {
		installSafetyNote(c)
		if widgets.Button(c, colors, label, spec).FillWidth().Height(installActionHeight).Clicked() {
			a.RetryInstall()
		}
	})
}
