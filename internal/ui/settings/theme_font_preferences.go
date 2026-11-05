package settings

import (
	"fmt"
	"strconv"
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
	// Installed ThemeLibrary / AppearanceSection.module.css: the library is
	// one 12px-spaced group. A pane's row spacing must not also enter the grid.
	library := ui.Column(c.Key("theme-library")).FillWidth().MinWidth(0).Gap(12)
	settingsSearchTarget(c, "theme-library", providerCopy("主题", "Theme"), library)
	library.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(12).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
			ui.Text(c, providerCopy("主题", "Themes")).Shrink(0).SingleLine().FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
			ui.Row(c).Shrink(0).Wrap().Gap(8).Children(func() {
				if widgets.Button(c, colors, providerCopy("创建主题", "Create theme"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonMedium, Disabled: e.OnSave == nil}).Clicked() {
					e.StartFamily(e.Saved.DuplicateFamily(e.Saved.Family(e.Saved.Dark)))
				}
				importDisabled := e.OnSave == nil || e.OnImport == nil
				importButton := widgets.Button(c, colors, providerCopy("导入主题", "Import theme"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonMedium, Disabled: importDisabled})
				if importDisabled {
					importButton.Tooltip(providerCopy("此构建未接入主题文件导入", "Theme file import is not wired in this build"))
				}
				if !importDisabled && importButton.Clicked() {
					raw, err := e.OnImport()
					if err != nil {
						e.Status = err.Error()
					} else if raw != "" {
						e.ImportLibraryFamily(raw)
					}
				}
			})
		})
		grid := ui.Grid(c.Key("theme-grid")).FillWidth().MinWidth(0).Gap(14)
		width := grid.Bounds().W
		grid.Draw(func(_ *ui.Painter, bounds ui.Rect) {
			if bounds.W != width {
				c.Invalidate()
			}
		})
		columns := max(1, int((width+14)/214)) // repeat(auto-fill, minmax(200px, 1fr))
		grid.Columns(columns).Children(func() {
			for _, f := range e.Saved.Families() {
				family := f
				currentLight := e.Draft.Light == family.ID
				currentDark := e.Draft.Dark == family.ID
				radius := themeLibraryRadius(e.Draft, c.Theme().Dark)
				border := chrome.Color(colors.BorderL1)
				card := ui.Column(c.Key("theme-card-"+family.ID)).Label("Theme card "+family.Name).MinWidth(0).Gap(0).Radius(radius).Clip().Border(.5, border).Background(themeLibrarySurface(colors, c.Theme().Dark))
				hovered := card.Hovered()
				if hovered {
					card.Border(.5, chrome.Color(colors.Border)).Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, .08))
				}
				actionsVisible := hovered || card.FocusWithin()
				card.Children(func() {
					ui.Row(c).FillWidth().Height(88).Children(func() {
						themePreviewHalf(c, family, family.Light, "light", currentLight, e.OnSave == nil, colors, radius, func() {
							e.SelectThemeHalf(false, family.ID)
						})
						themePreviewHalf(c, family, family.Dark, "dark", currentDark, e.OnSave == nil, colors, radius, func() {
							e.SelectThemeHalf(true, family.ID)
						})
					})
					ui.Row(c.Key("theme-foot-"+family.ID)).Label("Theme footer "+family.Name).FillWidth().MinWidth(0).MinHeight(54.5).Wrap().Gap(8).Padding(19, 12, 8, 12).AlignItems(ui.Center).
						BorderWidth(.5, 0, 0, 0).BorderColor(border).Children(func() {
						ui.Text(c, themeFamilyTitle(family.Name)).Grow(1).MinWidth(0).SingleLine().Tooltip(family.Name).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(fg)
						actions := ui.Row(c).Gap(2)
						if !actionsVisible {
							actions.Opacity(0)
						}
						actions.Children(func() {
							if themeFamilyAction(c, family, "duplicate", providerCopy("复制主题", "Duplicate theme"), "+", e.OnSave == nil) {
								e.StartFamily(e.Draft.DuplicateFamily(family))
							}
							for _, custom := range e.Draft.Custom {
								if custom.ID != family.ID {
									continue
								}
								if themeFamilyAction(c, family, "edit", providerCopy("编辑主题", "Edit theme"), "✎", e.OnSave == nil) {
									e.StartFamily(family)
								}
								if themeFamilyAction(c, family, "export", providerCopy("导出主题", "Export theme"), "↗", false) {
									c.WriteClipboard(theme.SerializeFamily(family))
								}
								if themeFamilyAction(c, family, "delete", providerCopy("删除主题", "Delete theme"), "×", e.OnSave == nil) {
									e.DeleteFamily(family.ID)
								}
								break
							}
						})
					})
				})
			}
		})
	})

	themeCustomEditor(c, e, fg, muted)
	themeEditorFeedback(c, e)
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

func themeFamilyAction(c *ui.Context, family theme.Family, action, label, glyph string, disabled bool) bool {
	button := ui.Button(c.Key("theme-"+action+"-"+family.ID), glyph).
		Label(label+" "+family.Name).Tooltip(label).Size(24, 24).Padding(0).
		Background(ui.Transparent).BorderWidth(0).Radius(6).TextColor(c.Theme().TextMuted).Disabled(disabled)
	if button.Hovered() && !disabled {
		button.Background(c.Theme().SurfaceHover).TextColor(c.Theme().Text)
	}
	return button.Clicked() && !disabled
}

// The selected half determines library chrome, while each miniature paints its
// own seed. Radius overrides remain attached to custom copies of a skin.
func themeLibraryRadius(p theme.Preferences, dark bool) float32 {
	id := p.Light
	if dark {
		id = p.Dark
	}
	seed := p.Family(id).Light
	if dark {
		seed = p.Family(id).Dark
	}
	if raw := seed.Overrides["--dsw-radius-button-sm"]; raw != "" {
		if v, err := strconv.ParseFloat(strings.TrimSuffix(raw, "px"), 32); err == nil && v >= 0 && v <= 64 {
			return float32(v)
		}
	}
	return 14
}

func themeLibraryBrand(colors theme.Colors, dark bool) ui.Color {
	p := theme.RenderPreferences()
	id := p.Light
	if dark {
		id = p.Dark
	}
	family := p.Family(id)
	seed := family.Light
	if dark {
		seed = family.Dark
	}
	if value := seed.Overrides["--dsw-alias-brand-primary"]; value != "" {
		return chrome.Color(value)
	}
	if family.ID != "deepseek" {
		return chrome.Color(seed.Accent)
	}
	return chrome.Color(colors.BrandPrimary)
}

func themeLibrarySurface(colors theme.Colors, dark bool) ui.Color {
	// The default native Field has historically represented module-platform;
	// ThemeLibrary explicitly asks for bg-layer-1, which is white in light mode.
	if !dark && colors.Field == "#f5f6f7" {
		return ui.RGB(255, 255, 255)
	}
	return chrome.Color(colors.Field)
}

// themePreviewHalf keeps its pressed semantics without inheriting MyGo's button
// layout or face. The installed preview is 88px high with a per-half inset ring.
func themePreviewHalf(c *ui.Context, family theme.Family, seed theme.Seed, scheme string, current, disabled bool, colors theme.Colors, radius float32, onPick func()) {
	ink := chrome.Color(seed.Foreground)
	brand := themeLibraryBrand(colors, c.Theme().Dark)
	root := ui.ButtonBase(c.Key("theme-half-"+family.ID+"-"+scheme)).Label(family.Name+" "+scheme).
		Basis(0).Grow(1).MinWidth(0).Height(88).Padding(10, 12).BorderWidth(0).
		Selected(current).Background(chrome.Color(seed.Background)).TextColor(ink).
		AlignItems(ui.Stretch).FocusRing(false).Disabled(disabled)
	if scheme == "light" {
		root.Radius(radius, 0, 0, 0)
	} else {
		root.Radius(0, radius, 0, 0)
	}
	if disabled {
		root.Opacity(.4)
	}
	root.Children(func() {
		ui.Column(c).FillWidth().Grow(1).MinWidth(0).MinHeight(0).Justify(ui.SpaceBetween).Gap(8).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Box(c).Label("Theme accent "+family.Name+" "+scheme).Size(16, 16).Shrink(0).Radius(8).
					Background(chrome.Color(seed.Accent)).Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, .18))
				ui.Column(c).Gap(4).Children(func() {
					ui.Box(c).Size(44, 5).Shrink(0).Radius(3).Background(ink.Alpha(.45))
					ui.Box(c).Size(28, 5).Shrink(0).Radius(3).Background(ink.Alpha(.25))
				})
			})
			switchText := providerCopy(map[string]string{"light": "浅色", "dark": "深色"}[scheme], map[string]string{"light": "Light", "dark": "Dark"}[scheme])
			ui.Row(c).MinWidth(0).Gap(6).AlignItems(ui.Center).Opacity(.75).Children(func() {
				ui.Text(c, switchText).SingleLine().FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(11)).TextColor(ink)
				if (scheme == "dark") == c.Theme().Dark {
					ui.Text(c, providerCopy("当前模式", "Current mode")).Label("Theme current mode "+family.Name+" "+scheme).
						SingleLine().Padding(1, 5).Border(1, ink).Radius(999).FontSize(theme.InterfaceSize(10)).
						FixedLineHeight(theme.InterfaceSize(14)).TextColor(ink).Opacity(.85)
				}
			})
		})
		if current {
			ui.Box(c).Absolute().Top(6).Right(6).Size(16, 16).Radius(8).Background(brand).Center().PassThrough().Children(func() {
				ui.Text(c, "✓").FontSize(theme.InterfaceSize(10)).FixedLineHeight(theme.InterfaceSize(10)).TextColor(ui.RGB(255, 255, 255))
			})
			ring := ui.Box(c).Absolute().Top(0).Right(0).Bottom(0).Left(0).Border(2, brand).PassThrough()
			if scheme == "light" {
				ring.Radius(radius, 0, 0, 0)
			} else {
				ring.Radius(0, radius, 0, 0)
			}
		}
		if root.FocusVisible() {
			ring := ui.Box(c).Absolute().Top(3).Right(3).Bottom(3).Left(3).Border(2, brand).PassThrough()
			if scheme == "light" {
				ring.Radius(max(0, radius-3), 0, 0, 0)
			} else {
				ring.Radius(0, max(0, radius-3), 0, 0)
			}
		}
	})
	if root.Clicked() && !disabled {
		onPick()
	}
}

// themeCustomEditor is the advanced editor for custom theme families, kept out of
// the visual theme grid.
func themeCustomEditor(c *ui.Context, e *theme.Editor, fg, muted ui.Color) {
	if e.Selected == "" {
		return
	}
	ui.Column(c.Key("custom-theme-editor")).FillWidth().Gap(8).Children(func() {
		for i := range e.Draft.Custom {
			f := &e.Draft.Custom[i]
			if f.ID != e.Selected {
				continue
			}
			ui.TextInput(c, &f.Name).FillWidth().Label(providerCopy("主题名称", "Theme name")).Disabled(e.FamilySavePending())
			if ui.Button(c, providerCopy("导出主题 JSON", "Export theme JSON")).Clicked() {
				c.WriteClipboard(theme.SerializeFamily(*f))
			}
			for _, variant := range []struct {
				label string
				seed  *theme.Seed
			}{{providerCopy("浅色", "Light"), &f.Light}, {providerCopy("深色", "Dark"), &f.Dark}} {
				hint := providerCopy("当前显示此模式的预览。", "This half is currently visible in the preview.")
				if (variant.seed == &f.Dark) != c.Theme().Dark {
					hint = providerCopy("切换颜色模式可查看此模式的预览。", "Switch color mode to preview this half.")
				}
				bodyHeading(c, variant.label, hint, fg, muted)
				for _, color := range []struct {
					label string
					value *string
				}{{providerCopy("背景", "Background"), &variant.seed.Background}, {providerCopy("前景", "Foreground"), &variant.seed.Foreground}, {providerCopy("强调色", "Accent"), &variant.seed.Accent}} {
					bodyPreferenceRow(c, "color-"+variant.label+color.label, color.label, "#RRGGBB", fg, muted, func() {
						appearanceColorWell(c, color.value, variant.label+" "+color.label, e.OnSave == nil || e.FamilySavePending(), false)
					})
				}
				contrast := variant.seed.ContrastValue()
				bodyPreferenceRow(c, "contrast-"+variant.label, providerCopy("对比度", "Contrast"), "", fg, muted, func() {
					if ui.StepSlider(c, &contrast, 0, 100, 1).FillWidth().Disabled(e.FamilySavePending()).Changed() {
						variant.seed.Contrast = &contrast
					}
				})
				bodyDisclosure(c, "overrides-"+variant.label, providerCopy("高级颜色覆盖", "Advanced token overrides"), func() {
					for _, key := range []string{"--dsw-alias-bg-base", "--dsw-alias-bg-layer-1", "--dsw-alias-bg-layer-2", "--dsw-alias-bg-overlay", "--dsw-alias-label-primary", "--dsw-alias-label-secondary", "--dsw-alias-brand-primary", "--dsw-alias-state-business-primary", "--dsw-alias-button-info-fill", "--dsw-alias-border-l1", "--dsw-alias-border-l2", "--dsw-specific-sidebar-fill", "--dsw-specific-bubble"} {
						value := variant.seed.Overrides[key]
						if ui.TextInput(c, &value).FillWidth().Label(key).Placeholder("Auto").Disabled(e.FamilySavePending()).Changed() {
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
				if ui.Button(c, providerCopy("取消", "Cancel")).Disabled(e.FamilySavePending()).Clicked() {
					e.Cancel()
					e.Selected = ""
				}
				saveLabel := providerCopy("保存主题", "Save theme")
				if e.FamilySavePending() {
					saveLabel = providerCopy("正在保存…", "Saving…")
				}
				if ui.PrimaryButton(c, saveLabel).Disabled(e.OnSave == nil || e.FamilySavePending()).Clicked() {
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
	if !settingsItemBefore(c, "general-font-settings") {
		return
	}
	group := ui.Column(c.Key("general-font-settings")).FillWidth().MinWidth(0).Gap(0)
	open := ui.Local(group, "expanded", func() bool { return false })
	if settingsSearchReveal(c, "general-font-settings") {
		*open = true
	}
	group.Children(func() {
		generalFontSizeRow(c, e, "text", providerCopy("字号大小", "Font size"), providerCopy("仅影响会话内容的字号", "Only affects conversation content"), e.Draft.Chat.Size, 10, 22, open, fg, muted)
		generalFontSizeRow(c, e, "code", providerCopy("代码字号", "Code size"), providerCopy("用于代码块、行内代码与工具输出", "Used for code blocks, inline code, and tool output"), e.Draft.Code.Size, 10, 20, nil, fg, muted)
		generalFontSizeRow(c, e, "terminal", providerCopy("终端字号", "Terminal font size"), providerCopy("用于侧栏终端", "Used for the sidebar terminal"), e.Draft.Terminal.Size, 10, 20, nil, fg, muted)
		if *open {
			generalFontFamilyRow(c, e, "text", providerCopy("文本字体", "Text font"), providerCopy("用于界面与会话正文。多个字体用逗号分隔，留空使用默认", "Used for the interface and conversation text. Separate fonts with commas; leave empty for the default"), e.Saved.Interface.Family, fg, muted)
			generalFontFamilyRow(c, e, "code", providerCopy("代码字体", "Code font"), providerCopy("用于代码块、行内代码与工具输出。多个字体用逗号分隔，留空使用默认", "Used for code blocks, inline code, and tool output. Separate fonts with commas; leave empty for the default"), e.Saved.Code.Family, fg, muted)
			generalFontFamilyRow(c, e, "terminal", providerCopy("终端字体", "Terminal font"), providerCopy("用于侧栏终端。多个字体用逗号分隔，留空使用默认", "Used for the sidebar terminal. Separate fonts with commas; leave empty for the default"), e.Saved.Terminal.Family, fg, muted)
		}

	})
	themeEditorFeedback(c, e)
}

func themeEditorFeedback(c *ui.Context, e *theme.Editor) {
	previous := ui.Local(c.Root(), "theme-preference-notice", func() string { return "" })
	if *previous == e.Status {
		return
	}
	*previous = e.Status
	switch e.Status {
	case "", "已保存 / Saved", "正在保存… / Saving…", i18n.T(i18n.KeyThemeEditorSaved), i18n.T(i18n.KeyThemeEditorSaving), i18n.T(i18n.KeyThemeEditorPreview), i18n.T(i18n.KeyThemeEditorRestored):
		return
	}
	c.AddToast(ui.Toast{ID: "theme-preference-error", Title: e.Status, Type: "error"})
}

func generalFontRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, control func()) {
	if !settingsItemBefore(c, key) {
		return
	}
	row := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(0)
	settingsSearchTarget(c, key, title, row)
	row.Children(func() {
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

// typographyField is one AppearanceSection .field: a muted 14 DIP label, the
// control, then an optional hint, stacked across the block's width.
func typographyField(c *ui.Context, key, label, hint string, muted ui.Color, control func()) {
	field := ui.Column(c.Key(key)).FillWidth().MinWidth(0).Gap(6)
	settingsSearchTarget(c, key, label, field)
	field.Children(func() {
		ui.Text(c, label).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).TextColor(muted)
		control()
		if hint != "" {
			ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
		}
	})
}

func typographyFontField(c *ui.Context, e *theme.Editor, key, label, hint, role string, f *theme.FontPreference, muted ui.Color) {
	typographyField(c, key, label, hint, muted, func() {
		if ui.TextInput(c, &f.Family).FillWidth().Label(label).Placeholder(providerCopy("系统默认", "System default")).Disabled(e.OnSave == nil).Changed() {
			e.SaveAppearanceFont(role, *f)
		}
	})
}

func typographySizeField(c *ui.Context, e *theme.Editor, key, label, role string, f *theme.FontPreference, lo, hi float64, muted ui.Color) {
	typographyField(c, key, label, "", muted, func() {
		value := float64(f.Size)
		if ui.StepSlider(c, &value, lo, hi, 1).FillWidth().Label(label).Disabled(e.OnSave == nil).Changed() {
			f.Size = float32(value)
			e.SaveAppearanceFont(role, *f)
		}
		ui.Text(c, fmt.Sprintf("%.0fpx", f.Size)).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).FontFeatures("tnum").TextColor(muted)
	})
}

// themeTypographyPane is the AppearanceSection §type block: heading + ghost
// reset, interface/code family fields with hints, the two px size sliders,
// then a ghost 高级 toggle for the composer and terminal families.
func themeTypographyPane(c *ui.Context, e *theme.Editor, fg, muted ui.Color) {
	if e == nil {
		return
	}
	block := ui.Column(c.Key("appearance-typography")).FillWidth().MinWidth(0).Gap(12)
	settingsSearchTarget(c, "appearance-typography", providerCopy("字体", "Typography"), block)
	advanced := ui.Local(block, "advanced", func() bool { return false })
	if settingsSearchReveal(c, "typography-advanced") {
		*advanced = true
	}
	block.Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, providerCopy("字体", "Typography")).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(16)).FixedLineHeight(theme.InterfaceSize(24)).FontWeight(500).TextColor(fg)
			if appearanceGhostButton(c, providerCopy("重置", "Reset"), e.OnSave == nil).Clicked() {
				e.ResetAppearanceFonts()
			}
		})
		typographyFontField(c, e, "font-interface", providerCopy("界面字体", "Interface font"), providerCopy("填写本机已安装的字体名（例如 Microsoft YaHei）。留空则用系统默认。用于代码块和终端以外的文字。", "Type a font name installed on this computer (for example Segoe UI). Leave empty for the system default. Used for text outside code blocks and the terminal."), "interface", &e.Draft.Interface, muted)
		typographySizeField(c, e, "font-size-interface", providerCopy("字号", "Size"), "interface", &e.Draft.Interface, 12, 22, muted)
		typographyFontField(c, e, "font-code", providerCopy("等宽字体", "Monospace font"), providerCopy("填写本机已安装的等宽字体名。留空则用系统默认。用于代码块、diff 和文件预览。", "Type a monospace font name installed on this computer. Leave empty for the system default. Used for code blocks, diffs, and file previews."), "code", &e.Draft.Code, muted)
		typographySizeField(c, e, "font-size-code", providerCopy("代码字号", "Code size"), "code", &e.Draft.Code, 10, 20, muted)
		if appearanceGhostButton(c, providerCopy("高级", "Advanced"), false).Clicked() {
			*advanced = !*advanced
		}
		if *advanced {
			ui.Column(c).FillWidth().MinWidth(0).Gap(12).Padding(4, 0, 0, 0).Children(func() {
				typographyFontField(c, e, "font-composer", providerCopy("输入框字体", "Composer font"), providerCopy("填写本机已安装的字体名。留空则跟随界面字体。", "Type a font name installed on this computer. Leave empty to follow the interface font."), "composer", &e.Draft.Composer, muted)
				typographyFontField(c, e, "font-terminal", providerCopy("终端字体", "Terminal font"), providerCopy("填写本机已安装的等宽字体名。留空则跟随等宽字体。", "Type a monospace font name installed on this computer. Leave empty to follow the monospace font."), "terminal", &e.Draft.Terminal, muted)
			})
		}
	})
}
