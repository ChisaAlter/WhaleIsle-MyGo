package settings

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func themeFontPane(c *ui.Context, e *theme.Editor) {
	if e == nil {
		return
	}
	themeLibraryPane(c, e, theme.Resolve(theme.IDDeepseekLight).Light, c.Theme().Text, c.Theme().TextMuted)
	themeTypographyPane(c, e, c.Theme().Text, c.Theme().TextMuted)
}
func themeLibraryPane(c *ui.Context, e *theme.Editor, colors theme.Colors, fg, muted ui.Color) {
	if e == nil {
		return
	}
	border := theme.Color(colors.Border)
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, providerCopy("主题", "Themes")).FillWidth().FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
			ui.Text(c, providerCopy("浅色与深色各选一个主题，卡片上半为浅色预览，下半为深色预览。", "Pick one theme for each color scheme; each card previews its light half above and dark half below.")).
				FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		})
		if widgets.Button(c, colors, providerCopy("创建主题", "Create theme"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall, Disabled: e.OnSave == nil}).Clicked() {
			e.StartFamily(e.Saved.DuplicateFamily(e.Saved.Family(e.Saved.Dark)))
		}
		importDisabled := e.OnSave == nil || e.OnImport == nil
		importButton := widgets.Button(c, colors, providerCopy("导入主题", "Import theme"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall, Disabled: importDisabled})
		if importDisabled {
			importButton.Tooltip(providerCopy("此构建未接入主题文件导入", "Theme file import is not wired in this build"))
		}
		if !importDisabled && importButton.Clicked() {
			raw, err := e.OnImport()
			if err != nil {
				e.Status = err.Error()
			} else if raw != "" {
				if family, err := e.Saved.ImportFamily(raw); err != nil {
					e.Status = err.Error()
				} else {
					e.Draft = e.Saved.Clone()
					e.Draft.Custom = append(e.Draft.Custom, family)
					e.Save()
				}
			}
		}
	})
	grid := ui.Grid(c.Key("theme-grid")).FillWidth().MinWidth(0).Gap(12)
	grid.Columns(2).Children(func() {
		for _, f := range e.Draft.Families() {
			family := f
			currentLight := e.Draft.Light == family.ID
			currentDark := e.Draft.Dark == family.ID
			card := ui.Column(c.Key("theme-card-"+family.ID)).MinWidth(0).Gap(0).Radius(10).Clip().Border(.5, border)
			if currentLight || currentDark {
				card.Border(1.5, fg)
			}
			card.Children(func() {
				ui.Row(c).FillWidth().Height(92).Children(func() {
					themePreviewHalf(c, family, family.Light, "light", currentLight, e.OnSave == nil, muted, func() {
						e.SelectThemeHalf(false, family.ID)
					})
					themePreviewHalf(c, family, family.Dark, "dark", currentDark, e.OnSave == nil, muted, func() {
						e.SelectThemeHalf(true, family.ID)
					})
				})
				ui.Row(c).FillWidth().MinWidth(0).Gap(6).Padding(10, 12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, themeFamilyTitle(family.Name)).Grow(1).MinWidth(0).SingleLine().Tooltip(family.Name).FontSize(theme.InterfaceSize(14)).TextColor(fg)
					if currentLight || currentDark {
						status := providerCopy("当前主题", "Current theme")
						switch {
						case currentLight && !currentDark:
							status = providerCopy("当前浅色", "Current light")
						case currentDark && !currentLight:
							status = providerCopy("当前深色", "Current dark")
						}
						// One family can supply both schemes; state it once
						// instead of drawing a check on each half.
						ui.Text(c, status).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					}
				})
			})
		}
	})
	bodyDisclosure(c, "theme-names", providerCopy("按名称选择主题", "Pick themes by name"), func() {
		ids := []string{}
		for _, f := range e.Draft.Families() {
			ids = append(ids, f.ID)
		}
		for _, half := range []struct {
			label string
			value *string
		}{{providerCopy("浅色主题", "Light theme"), &e.Draft.Light}, {providerCopy("深色主题", "Dark theme"), &e.Draft.Dark}} {
			bodyPreferenceRow(c, "theme-half-"+half.label, half.label, "", fg, muted, func() {
				if ui.Select(c, half.value, ids).FillWidth().Label(half.label).Disabled(e.OnSave == nil).Changed() {
					e.SelectThemeHalf(half.value == &e.Draft.Dark, *half.value)
				}
			})
		}
	})
	themeCustomEditor(c, e, fg, muted)
}

func themeFamilyTitle(name string) string {
	parts := strings.SplitN(name, " / ", 2)
	if len(parts) != 2 {
		return name
	}
	if i18n.Locale() == "en" {
		return parts[1]
	}
	return parts[0]
}

// themePreviewHalf is one half of a theme card: the seed's own background, its
// accent dot, two skeleton lines and the scheme chip. A click selects that half.
func themePreviewHalf(c *ui.Context, family theme.Family, seed theme.Seed, scheme string, current, disabled bool, muted ui.Color, onPick func()) {
	root := ui.Button(c.Key("theme-half-"+family.ID+"-"+scheme), "").
		Label(family.Name + " " + scheme).
		Basis(0).Grow(1).MinWidth(0).Height(92).Padding(12).
		Background(chrome.Color(seed.Background)).BorderWidth(0).Disabled(disabled)
	if root.Hovered() && !disabled {
		root = root.Opacity(0.92)
	}
	root.Children(func() {
		// Button.Children lays children out horizontally; the preview needs an
		// explicit column: dot/skeleton row, hairline, then the scheme chip.
		ui.Column(c).FillWidth().Grow(1).MinHeight(0).Justify(ui.SpaceBetween).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Box(c).Size(10, 10).Shrink(0).Radius(5).Background(chrome.Color(seed.Accent))
				ui.Box(c).Width(38).Height(6).Shrink(0).Radius(3).Background(chrome.Color(seed.Foreground).Alpha(.35))
				ui.Box(c).Grow(1)
			})
			ui.Box(c).Height(4)
			ui.Box(c).FillWidth().Height(6).Radius(3).Background(chrome.Color(seed.Foreground).Alpha(.18))
			switchText := providerCopy(map[string]string{"light": "浅色", "dark": "深色"}[scheme], map[string]string{"light": "Light", "dark": "Dark"}[scheme])
			// The check above already marks the family selected for this
			// scheme; repeating "(current mode)" on both halves reads as if both
			// modes were active at once.
			ui.Row(c).Gap(0).Children(func() {
				ui.Text(c, switchText).Padding(1, 6).Radius(4).FontSize(theme.InterfaceSize(10)).
					Border(.5, chrome.Color(seed.Foreground).Alpha(.25)).TextColor(chrome.Color(seed.Foreground).Alpha(.85))
			})
		})
	})
	if root.Clicked() && !disabled {
		onPick()
	}
}

// themeCustomEditor is the advanced editor for custom theme families, kept out of
// the visual theme grid.
func themeCustomEditor(c *ui.Context, e *theme.Editor, fg, muted ui.Color) {
	ui.Column(c.Key("custom-theme-editor")).FillWidth().Gap(8).Children(func() {
		customIDs := []string{""}
		labels := []string{providerCopy("选择自定义主题", "Select custom theme")}
		for _, f := range e.Draft.Custom {
			customIDs = append(customIDs, f.ID)
			name := strings.TrimSpace(f.Name)
			if name == "" {
				name = f.ID
			}
			labels = append(labels, name)
		}
		selected := 0
		for index, id := range customIDs {
			if id == e.Selected {
				selected = index
			}
		}
		selectedLabel := labels[selected]
		if ui.Select(c, &selectedLabel, labels).FillWidth().Label(providerCopy("选择自定义主题", "Select custom theme")).Changed() {
			for index, label := range labels {
				if label == selectedLabel {
					e.Selected = customIDs[index]
					break
				}
			}
			if e.Selected != "" {
				e.StartFamily(e.Draft.Family(e.Selected))
			}
		}
		ui.Row(c).Wrap().Gap(8).Children(func() {
			if ui.Button(c, providerCopy("复制浅色主题", "Duplicate light theme")).Clicked() {
				f := e.Draft.DuplicateFamily(e.Draft.Family(e.Draft.Light))
				e.StartFamily(f)
			}
			if ui.Button(c, providerCopy("删除自定义主题", "Delete custom theme")).Disabled(e.Selected == "").Clicked() {
				e.DeleteSelected()
				e.Save()
			}
		})
		for i := range e.Draft.Custom {
			f := &e.Draft.Custom[i]
			if f.ID != e.Selected {
				continue
			}
			ui.TextInput(c, &f.Name).FillWidth().Label(providerCopy("主题名称", "Theme name"))
			if ui.Button(c, providerCopy("导出主题 JSON", "Export theme JSON")).Clicked() {
				c.WriteClipboard(theme.SerializeFamily(*f))
			}
			for _, variant := range []struct {
				label string
				seed  *theme.Seed
			}{{providerCopy("浅色", "Light"), &f.Light}, {providerCopy("深色", "Dark"), &f.Dark}} {
				bodyHeading(c, variant.label, "", fg, muted)
				for _, color := range []struct {
					label string
					value *string
				}{{providerCopy("背景", "Background"), &variant.seed.Background}, {providerCopy("前景", "Foreground"), &variant.seed.Foreground}, {providerCopy("强调色", "Accent"), &variant.seed.Accent}} {
					bodyPreferenceRow(c, "color-"+variant.label+color.label, color.label, "#RRGGBB", fg, muted, func() { ui.TextInput(c, color.value).FillWidth().Label(variant.label + " " + color.label) })
				}
				contrast := variant.seed.ContrastValue()
				bodyPreferenceRow(c, "contrast-"+variant.label, providerCopy("对比度", "Contrast"), "", fg, muted, func() {
					if ui.StepSlider(c, &contrast, 0, 100, 1).FillWidth().Changed() {
						variant.seed.Contrast = &contrast
					}
				})
				bodyDisclosure(c, "overrides-"+variant.label, providerCopy("高级颜色覆盖", "Advanced token overrides"), func() {
					for _, key := range []string{"--dsw-alias-bg-base", "--dsw-alias-bg-layer-1", "--dsw-alias-bg-layer-2", "--dsw-alias-bg-overlay", "--dsw-alias-label-primary", "--dsw-alias-label-secondary", "--dsw-alias-brand-primary", "--dsw-alias-state-business-primary", "--dsw-alias-button-info-fill", "--dsw-alias-border-l1", "--dsw-alias-border-l2", "--dsw-specific-sidebar-fill", "--dsw-specific-bubble"} {
						value := variant.seed.Overrides[key]
						if ui.TextInput(c, &value).FillWidth().Label(key).Placeholder("Auto").Changed() {
							if variant.seed.Overrides == nil {
								variant.seed.Overrides = map[string]string{}
							}
							if value == "" {
								delete(variant.seed.Overrides, key)
							} else {
								variant.seed.Overrides[key] = value
							}
						}
					}
				})
			}
		}
		if e.Selected != "" {
			e.PreviewSelectedFamily()
			ui.Row(c).Gap(8).Children(func() {
				if ui.Button(c, providerCopy("取消", "Cancel")).Clicked() {
					e.Cancel()
					e.Selected = ""
				}
				if ui.PrimaryButton(c, providerCopy("保存主题", "Save theme")).Disabled(e.OnSave == nil).Clicked() {
					e.SaveSelectedFamily()
				}
			})
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
		generalFontSizeRow(c, e, "text", providerCopy("字号大小", "Font size"), providerCopy("仅影响会话内容的字号", "Only affects conversation content"), e.Draft.Chat.Size, 10, 22, open, fg, muted)
		if *open {
			generalFontFamilyRow(c, e, "text", providerCopy("文本字体", "Text font"), providerCopy("用于界面与会话正文。多个字体用逗号分隔，留空使用默认", "Used for the interface and conversation text. Separate fonts with commas; leave empty for the default"), e.Saved.Interface.Family, fg, muted)
			generalFontFamilyRow(c, e, "code", providerCopy("代码字体", "Code font"), providerCopy("用于代码块、行内代码与工具输出。多个字体用逗号分隔，留空使用默认", "Used for code blocks, inline code, and tool output. Separate fonts with commas; leave empty for the default"), e.Saved.Code.Family, fg, muted)
			generalFontSizeRow(c, e, "code", providerCopy("代码字号", "Code size"), providerCopy("用于代码块、行内代码与工具输出", "Used for code blocks, inline code, and tool output"), e.Draft.Code.Size, 10, 20, nil, fg, muted)
			generalFontFamilyRow(c, e, "terminal", providerCopy("终端字体", "Terminal font"), providerCopy("用于侧栏终端。多个字体用逗号分隔，留空使用默认", "Used for the sidebar terminal. Separate fonts with commas; leave empty for the default"), e.Saved.Terminal.Family, fg, muted)
			generalFontSizeRow(c, e, "terminal", providerCopy("终端字号", "Terminal font size"), providerCopy("用于侧栏终端", "Used for the sidebar terminal"), e.Draft.Terminal.Size, 10, 20, nil, fg, muted)
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
			pill := ui.Row(c).Width(88).Height(36).Radius(12).Background(c.Theme().Surface).Center()
			pill.Children(func() {
				ui.Text(c, fmt.Sprintf("%.0f", value)).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).FontFeatures("tnum").TextColor(fg)
				ui.Text(c, " px").FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(muted)
				arrows := ui.Column(c).Absolute().Right(6).Top(5).Gap(2)
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
	bodyHeading(c, providerCopy("字体", "Typography"), providerCopy("字体名称可逗号分隔；留空使用系统字体。", "Use comma-separated fallback families; leave blank for the system font."), fg, muted)
	render := func(role, label string, f *theme.FontPreference, family func(string) string, lo, hi float64, size bool) {
		bodyPreferenceRow(c, "font-"+label, label, "", fg, muted, func() {
			if ui.TextInput(c, &f.Family).FillWidth().Label(label + providerCopy("字体", " font")).Placeholder(providerCopy("系统字体", "System font")).Disabled(e.OnSave == nil).Changed() {
				e.SaveAppearanceFont(role, *f)
			}
		})
		value := float64(f.Size)
		if size {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
				if ui.StepSlider(c, &value, lo, hi, 1).Grow(1).MinWidth(0).Label(label + providerCopy("字号", " size")).Disabled(e.OnSave == nil).Changed() {
					f.Size = float32(value)
					e.SaveAppearanceFont(role, *f)
				}
				ui.Text(c, fmt.Sprintf("%.0f px", f.Size)).Width(44).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			})
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, providerCopy("预览", "Preview")).FontSize(11).TextColor(muted)
			ui.Text(c, providerCopy("字体 Aa 0123", "Aa 0123 — the quick brown fox")).Font(family(f.Family)).FontSize(f.Size).TextColor(fg)
		})
	}
	render("interface", providerCopy("界面", "Interface"), &e.Draft.Interface, theme.TextFamily, 12, 22, true)
	render("code", providerCopy("代码", "Code"), &e.Draft.Code, theme.CodeFamily, 10, 20, true)
	bodyDisclosure(c, "typography-advanced", providerCopy("高级字体设置", "Advanced typography"), func() {
		render("composer", providerCopy("输入框", "Composer"), &e.Draft.Composer, theme.TextFamily, 10, 22, true)
		render("terminal", providerCopy("终端", "Terminal"), &e.Draft.Terminal, func(f string) string { return f }, 10, 20, true)
	})
	ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
		if ui.Button(c, providerCopy("恢复默认字体", "Reset fonts")).Disabled(e.OnSave == nil).Clicked() {
			e.ResetAppearanceFonts()
		}
		if ui.Button(c, providerCopy("预览", "Preview")).Clicked() {
			e.ApplyPreview()
			c.Invalidate()
		}
		if ui.Button(c, providerCopy("取消预览", "Cancel preview")).Disabled(!e.Preview).Clicked() {
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