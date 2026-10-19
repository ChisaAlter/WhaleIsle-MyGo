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
	group := ui.Column(c.Key("general-font-settings")).FillWidth().MinWidth(0).Gap(0)
	open := ui.Local(group, "expanded", func() bool { return false })
	group.Children(func() {
		generalFontSizeRow(c, e, "text", providerCopy("字号大小", "Font size"), providerCopy("只影响会话内容的字号", "Only affects conversation content"), e.Saved.Chat.Size, 10, 22, open, fg, muted)
		if *open {
			generalFontFamilyRow(c, e, "text", providerCopy("文本字体", "Text font"), providerCopy("用于界面与会话正文。多个字体用逗号分隔，留空使用默认", "Used for the interface and conversation text. Separate fonts with commas; leave empty for the default"), e.Saved.Interface.Family, fg, muted)
			generalFontFamilyRow(c, e, "code", providerCopy("代码字体", "Code font"), providerCopy("用于代码块、行内代码与工具输出。多个字体用逗号分隔，留空使用默认", "Used for code blocks, inline code, and tool output. Separate fonts with commas; leave empty for the default"), e.Saved.Code.Family, fg, muted)
			generalFontSizeRow(c, e, "code", providerCopy("代码字号", "Code font size"), providerCopy("用于代码块、行内代码与工具输出", "Used for code blocks, inline code, and tool output"), e.Saved.Code.Size, 10, 20, nil, fg, muted)
			generalFontFamilyRow(c, e, "terminal", providerCopy("终端字体", "Terminal font"), providerCopy("用于侧栏终端。多个字体用逗号分隔，留空使用默认", "Used for the sidebar terminal. Separate fonts with commas; leave empty for the default"), e.Saved.Terminal.Family, fg, muted)
			generalFontSizeRow(c, e, "terminal", providerCopy("终端字号", "Terminal font size"), providerCopy("用于侧栏终端", "Used for the sidebar terminal"), e.Saved.Terminal.Size, 10, 20, nil, fg, muted)
		}
		if e.Status != "" {
			ui.Text(c, e.Status).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		}
	})
}

func generalFontRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, control func()) {
	ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Padding(16, 0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Padding(0, 48, 0, 0).Gap(4).Children(func() {
				ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(fg)
				ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
			})
			control()
		})
		ui.Box(c).FillWidth().Height(.5).Background(c.Theme().Border)
	})
}

func generalFontSizeRow(c *ui.Context, e *theme.Editor, role, title, hint string, value, lo, hi float32, open *bool, fg, muted ui.Color) {
	generalFontRow(c, "general-size-"+role, title, hint, fg, muted, func() {
		ui.Row(c).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			pill := ui.Row(c).Width(72).Height(36).Radius(12).Background(c.Theme().Surface).Center()
			pill.Children(func() {
				ui.Text(c, fmt.Sprintf("%.0f", value)).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).FontFeatures("tnum").TextColor(fg)
				arrows := ui.Column(c).Absolute().Right(8).Top(5).Gap(2)
				if !pill.Hovered() && !pill.FocusWithin() {
					arrows.Opacity(0)
				}
				arrows.Children(func() {
					for _, direction := range []int{1, -1} {
						label := providerCopy("增大", "Increase") + " " + title
						disabled := value >= hi
						if direction < 0 {
							label = providerCopy("减小", "Decrease") + " " + title
							disabled = value <= lo
						}
						button := ui.Button(c.Key(label), "").Label(label).Tooltip(label).Size(17, 12).Padding(0).Radius(4).BorderWidth(0).Background(c.Theme().Background.Alpha(.75)).Disabled(e.OnSave == nil || disabled)
						button.Children(func() {
							icon := ui.Icon(c, bodyIcon("down")).Size(9, 9).TextColor(fg)
							if direction > 0 {
								icon.Rotate(180)
							}
						})
						if button.Clicked() {
							e.SaveFontSize(role, min(hi, max(lo, value+float32(direction))))
							c.Invalidate()
						}
					}
				})
			})
			ui.Text(c, "px").FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(muted)
			if open != nil {
				button := ui.Button(c, "").Label(providerCopy("更多字体设置", "More font settings")).Tooltip(providerCopy("更多字体设置", "More font settings")).Size(24, 24).Padding(0).Radius(4).BorderWidth(0).Background(ui.Transparent)
				button.Children(func() {
					icon := ui.Icon(c, bodyIcon("down")).Size(12, 12).TextColor(muted)
					if *open {
						icon.Rotate(180)
					}
				})
				if button.Clicked() {
					if *open {
						e.FlushGeneralFontEdits(true)
					}
					*open = !*open
				}
			}
		})
	})
}

func generalFontFamilyRow(c *ui.Context, e *theme.Editor, role, title, hint, saved string, fg, muted ui.Color) {
	generalFontRow(c, "general-family-"+role, title, hint, fg, muted, func() {
		anchor := ui.Column(c).Width(240).Shrink(1).MinWidth(0)
		type familyDraft struct {
			Value, Base string
			Focused     bool
		}
		draft := ui.Local(anchor, "family-draft", func() familyDraft { return familyDraft{Value: e.GeneralFontFamily(role), Base: saved} })
		if draft.Base != saved {
			draft.Value, draft.Base = saved, saved
		}
		commit := func() {
			if e.OnSave != nil && e.CommitGeneralFontEdit(role) {
				c.Invalidate()
			}
		}
		anchor.Children(func() {
			input := ui.TextInput(c, &draft.Value).FillWidth().Label(title).Placeholder(providerCopy("默认", "Default")).Disabled(e.OnSave == nil).OnSubmit(commit)
			input.OnChange(func() {
				units := 0
				for at, r := range draft.Value {
					units++
					if r > 0xffff {
						units++
					}
					if units > 200 {
						draft.Value = draft.Value[:at]
						break
					}
				}
				e.QueueGeneralFontEdit(role, draft.Value)
			})
			if draft.Focused && !input.Focused() && !input.Composing() {
				commit()
			}
			draft.Focused = input.Focused()
		})
	})
}
func themeTypographyPane(c *ui.Context, e *theme.Editor, fg, muted ui.Color) {
	if e == nil {
		return
	}
	bodyHeading(c, providerCopy("字体与字号", "Typography"), providerCopy("字体名称可逗号分隔；留空使用系统字体。", "Use comma-separated fallback families; leave blank for the system font."), fg, muted)
	render := func(label string, f *theme.FontPreference, family func(string) string) {
		bodyPreferenceRow(c, "font-"+label, label, "", fg, muted, func() { ui.TextInput(c, &f.Family).FillWidth().Label(label + providerCopy("字体", " font")) })
		value := float64(f.Size)
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
			if ui.StepSlider(c, &value, 8, 48, 1).Grow(1).MinWidth(0).Label(label + providerCopy("字号", " size")).Changed() {
				f.Size = float32(value)
			}
			ui.Text(c, fmt.Sprintf("%.0f px", f.Size)).Width(44).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		})
		ui.Text(c, providerCopy("预览 字体 Aa 0123", "Preview Aa 0123")).FillWidth().Font(family(f.Family)).FontSize(f.Size).TextColor(fg)
	}
	render(providerCopy("界面", "Interface"), &e.Draft.Interface, theme.TextFamily)
	render(providerCopy("代码", "Code"), &e.Draft.Code, theme.CodeFamily)
	bodyDisclosure(c, "typography-advanced", providerCopy("高级字体设置", "Advanced typography"), func() {
		render(providerCopy("聊天", "Chat"), &e.Draft.Chat, theme.TextFamily)
		render(providerCopy("输入框", "Composer"), &e.Draft.Composer, theme.TextFamily)
		render(providerCopy("终端", "Terminal"), &e.Draft.Terminal, func(f string) string { return f })
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
