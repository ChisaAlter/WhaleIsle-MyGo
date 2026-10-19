package settings

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"time"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func themeFontPane(c *ui.Context, e *theme.Editor) {
	if e == nil {
		return
	}
	themeLibraryPane(c, e, c.Theme().Text, c.Theme().TextMuted)
	themeTypographyPane(c, e, c.Theme().Text, c.Theme().TextMuted)
}
func themeLibraryPane(c *ui.Context, e *theme.Editor, fg, muted ui.Color) {
	if e == nil {
		return
	}
	bodyHeading(c, providerCopy("主题库", "Theme library"), providerCopy("分别为浅色与深色外观选择主题。", "Choose a theme family for each color scheme."), fg, muted)
	ids := []string{}
	for _, f := range e.Draft.Families() {
		ids = append(ids, f.ID)
	}
	for _, half := range []struct {
		label string
		value *string
	}{{providerCopy("浅色主题", "Light theme"), &e.Draft.Light}, {providerCopy("深色主题", "Dark theme"), &e.Draft.Dark}} {
		bodyPreferenceRow(c, "theme-half-"+half.label, half.label, "", fg, muted, func() { ui.Select(c, half.value, ids).FillWidth().Label(half.label) })
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		for _, f := range e.Draft.Families() {
			ui.Column(c.Key("theme-family-" + f.ID)).Width(124).MinWidth(0).Gap(4).Children(func() {
				ui.Row(c).Gap(2).Children(func() {
					for _, s := range []theme.Seed{f.Light, f.Dark} {
						ui.Text(c, "Aa").Width(60).Height(42).FontSize(20).Background(chrome.Color(s.Background)).TextColor(chrome.Color(s.Foreground)).Padding(8)
					}
				})
				ui.Text(c, f.Name).FillWidth().SingleLine().Tooltip(f.Name).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			})
		}
	})
	bodyDisclosure(c, "custom-theme-editor", providerCopy("自定义主题", "Custom themes"), func() {
		customIDs := []string{""}
		for _, f := range e.Draft.Custom {
			customIDs = append(customIDs, f.ID)
		}
		ui.Select(c, &e.Selected, customIDs).FillWidth().Label(providerCopy("选择自定义主题", "Select custom theme"))
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if ui.Button(c, providerCopy("复制浅色主题", "Duplicate light theme")).Clicked() {
				f := e.Draft.Family(e.Draft.Light)
				f.ID = fmt.Sprintf("custom-%d", time.Now().UnixNano())
				f.Name += " copy"
				e.Draft.Custom = append(e.Draft.Custom, f)
				e.Selected = f.ID
			}
			if ui.Button(c, providerCopy("删除自定义主题", "Delete custom theme")).Disabled(e.Selected == "").Clicked() {
				e.DeleteSelected()
			}
		})
		for i := range e.Draft.Custom {
			f := &e.Draft.Custom[i]
			if f.ID != e.Selected {
				continue
			}
			ui.TextInput(c, &f.Name).FillWidth().Label(providerCopy("主题名称", "Theme name"))
			for _, variant := range []struct {
				label string
				seed  *theme.Seed
			}{{providerCopy("浅色", "Light"), &f.Light}, {providerCopy("深色", "Dark"), &f.Dark}} {
				bodyHeading(c, variant.label, "", fg, muted)
				for _, color := range []struct {
					label string
					value *string
				}{{providerCopy("背景", "Background"), &variant.seed.Background}, {providerCopy("文字", "Foreground"), &variant.seed.Foreground}, {providerCopy("强调色", "Accent"), &variant.seed.Accent}} {
					bodyPreferenceRow(c, "color-"+variant.label+color.label, color.label, "#RRGGBB", fg, muted, func() { ui.TextInput(c, color.value).FillWidth().Label(variant.label + " " + color.label) })
				}
			}
		}
	})
}
func bodyFontSizePreference(c *ui.Context, e *theme.Editor, fg, muted ui.Color) {
	if e == nil {
		return
	}
	bodyPreferenceRow(c, "general-font-size", providerCopy("界面字号", "Interface font size"), "", fg, muted, func() {
		value := float64(e.Draft.Interface.Size)
		if ui.StepSlider(c, &value, 8, 48, 1).FillWidth().Label(providerCopy("界面字号", "Interface font size")).Changed() {
			e.Draft.Interface.Size = float32(value)
		}
		ui.Text(c, fmt.Sprintf("%.0f px", value)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		if ui.Button(c, providerCopy("应用字号", "Apply font size")).Disabled(e.OnSave == nil).Clicked() {
			e.Save()
			c.Invalidate()
		}
	})
}
func themeTypographyPane(c *ui.Context, e *theme.Editor, fg, muted ui.Color) {
	if e == nil {
		return
	}
	bodyHeading(c, providerCopy("字体与字号", "Typography"), providerCopy("字体名称可逗号分隔；留空使用系统字体。", "Use comma-separated fallback families; leave blank for the system font."), fg, muted)
	render := func(label string, f *theme.FontPreference) {
		bodyPreferenceRow(c, "font-"+label, label, "", fg, muted, func() { ui.TextInput(c, &f.Family).FillWidth().Label(label + providerCopy("字体", " font")) })
		value := float64(f.Size)
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			if ui.StepSlider(c, &value, 8, 48, 1).Grow(1).MinWidth(0).Label(label + providerCopy("字号", " size")).Changed() {
				f.Size = float32(value)
			}
			ui.Text(c, fmt.Sprintf("%.0f px", f.Size)).Width(44).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		})
		ui.Text(c, providerCopy("预览 字体 Aa 0123", "Preview Aa 0123")).FillWidth().Font(f.Family).FontSize(f.Size).TextColor(fg)
	}
	render(providerCopy("界面", "Interface"), &e.Draft.Interface)
	render(providerCopy("代码", "Code"), &e.Draft.Code)
	bodyDisclosure(c, "typography-advanced", providerCopy("高级字体设置", "Advanced typography"), func() {
		render(providerCopy("聊天", "Chat"), &e.Draft.Chat)
		render(providerCopy("输入框", "Composer"), &e.Draft.Composer)
		render(providerCopy("终端", "Terminal"), &e.Draft.Terminal)
	})
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		if ui.Button(c, providerCopy("恢复默认字体", "Reset fonts")).Clicked() {
			d := theme.Defaults()
			e.Draft.Interface = d.Interface
			e.Draft.Chat = d.Chat
			e.Draft.Code = d.Code
			e.Draft.Composer = d.Composer
			e.Draft.Terminal = d.Terminal
		}
		if ui.Button(c, providerCopy("预览", "Preview")).Clicked() {
			e.ApplyPreview()
			c.Invalidate()
		}
		if ui.Button(c, providerCopy("取消预览", "Cancel preview")).Clicked() {
			e.Cancel()
			c.Invalidate()
		}
		if ui.Button(c, providerCopy("保存主题与字体", "Save themes and fonts")).Disabled(e.OnSave == nil).Clicked() {
			e.Save()
			c.Invalidate()
		}
	})
	if e.Status != "" {
		ui.Text(c, e.Status).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
	}
}
