package settings

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/whale"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// petWhaleState keeps the page's local editing: the two identity drafts, the
// inline persona editor, the memory dialog and the skills disclosure.
type petWhaleState struct {
	Name, Title           string
	NameDirty, TitleDirty bool
	NameFocus, TitleFocus bool

	PersonaOpen bool
	PersonaText string

	MemoryOpen  bool
	MemoryText  string
	MemoryClear bool

	SkillsOpen bool
}

// petWhaleSection is the Host-driven half of the Whale page: every row edits
// the dsh-whale catalog through the access callbacks. A nil catalog means the
// Host has not answered yet, so the page shows one loading line instead of
// controls that would write nowhere.
func petWhaleSection(c *ui.Context, colors theme.Colors, a *PetSettingsAccess, fg, muted ui.Color) {
	anchor := ui.Column(c.Key("pet-whale-section")).FillWidth().MinWidth(0).Gap(4)
	st := ui.Local(anchor, "state", func() petWhaleState { return petWhaleState{} })
	anchor.Children(func() {
		cat := a.Whale
		if cat == nil {
			ui.Text(c, i18n.T(i18n.KeySettingsWhaleUnavailable)).FillWidth().
				FontSize(theme.InterfaceSize(12)).TextColor(muted)
			return
		}
		update := func(patch whale.Patch) {
			if a.OnUpdateWhale != nil {
				a.OnUpdateWhale(patch)
			}
		}

		// Seed the drafts before rendering so the first frame already shows the
		// stored name instead of an empty field.
		if !st.NameFocus && !st.NameDirty {
			st.Name = cat.Name
		}
		if !st.TitleFocus && !st.TitleDirty {
			st.Title = cat.UserTitle
		}
		// 名字与称呼: two inputs on one row, committed on blur or Enter. They need
		// more than the shared 180 DIP control column, so this row lays itself out.
		whaleWideRow(c, "pet-whale-identity", i18n.T(i18n.KeySettingsWhaleIdentity),
			i18n.T(i18n.KeySettingsWhaleIdentityHint), fg, muted, func() {
				ui.Row(c).FillWidth().Gap(8).Children(func() {
					name := ui.TextInput(c, &st.Name).Width(180).Label(i18n.T(i18n.KeySettingsWhaleName)).
						Placeholder(i18n.T(i18n.KeySettingsWhaleNamePlaceholder))
					title := ui.TextInput(c, &st.Title).Width(180).Label(i18n.T(i18n.KeySettingsWhaleTitle)).
						Placeholder(i18n.T(i18n.KeySettingsWhaleTitlePlaceholder))
					if name.Changed() {
						st.NameDirty = true
					}
					if title.Changed() {
						st.TitleDirty = true
					}
					commitName := func() {
						if !st.NameDirty && strings.TrimSpace(st.Name) == strings.TrimSpace(cat.Name) {
							return
						}
						value := st.Name
						st.NameDirty = false
						update(whale.Patch{Name: &value})
					}
					commitTitle := func() {
						if !st.TitleDirty && strings.TrimSpace(st.Title) == strings.TrimSpace(cat.UserTitle) {
							return
						}
						value := st.Title
						st.TitleDirty = false
						update(whale.Patch{UserTitle: &value})
					}
					if st.NameFocus && !name.Focused() {
						commitName()
					}
					if st.TitleFocus && !title.Focused() {
						commitTitle()
					}
					if name.Submitted() {
						commitName()
					}
					if title.Submitted() {
						commitTitle()
					}
					st.NameFocus, st.TitleFocus = name.Focused(), title.Focused()
				})
			})

		// 默认模型: the catalog only carries the current route, so an empty
		// value follows the session and a set one is read-only here.
		modelLabel := strings.TrimSpace(strings.TrimSpace(cat.ModelProvider) + " / " + strings.TrimSpace(cat.ModelModel))
		modelLabel = strings.Trim(strings.TrimSpace(modelLabel), "/")
		bodyPreferenceRow(c, "pet-whale-model", i18n.T(i18n.KeySettingsWhaleModel),
			i18n.T(i18n.KeySettingsWhaleModelHint), fg, muted, func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
					text := i18n.T(i18n.KeySettingsWhaleFollowDefault)
					if modelLabel != "" {
						text = modelLabel
					}
					ui.Text(c, text).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(fg)
					if a.OnUpdateWhale != nil {
						ui.Text(c, i18n.T(i18n.KeySettingsWhaleModelElsewhere)).FillWidth().
							FontSize(theme.InterfaceSize(12)).TextColor(muted)
					}
				})
			})

		// IM 接入同一对话
		im := cat.IMDefault
		bodyPreferenceRow(c, "pet-whale-im", i18n.T(i18n.KeySettingsWhaleIM),
			i18n.T(i18n.KeySettingsWhaleIMHint), fg, muted, func() {
				if ui.Switch(c, &im).Label("Whale IM toggle").Size(44, 24).
					Disabled(a.OnUpdateWhale == nil).Changed() && a.OnUpdateWhale != nil {
					update(whale.Patch{IMDefault: &im})
				}
			})

		// 额外人设: inline editor behind a text action.
		bodyPreferenceRow(c, "pet-whale-persona", i18n.T(i18n.KeySettingsWhalePersona),
			i18n.T(i18n.KeySettingsWhalePersonaHint), fg, muted, func() {
				if !st.PersonaOpen {
					ui.Row(c).FillWidth().Children(func() {
						ui.Text(c, personaSummary(cat.PersonaText)).Grow(1).MinWidth(0).SingleLine().
							MaxLines(1).FontSize(theme.InterfaceSize(13)).TextColor(muted)
						linkAction(c, i18n.T(i18n.KeySettingsWhaleEdit), muted, a.OnUpdateWhale != nil, func() {
							st.PersonaOpen, st.PersonaText = true, cat.PersonaText
						})
					})
					return
				}
				ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
					ui.TextArea(c, &st.PersonaText).FillWidth().Lines(3, 10).
						Label(i18n.T(i18n.KeySettingsWhalePersona))
					ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.End).Children(func() {
						if widgets.Button(c, colors, i18n.T(i18n.KeySettingsWhaleCancel),
							widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
							st.PersonaOpen = false
						}
						if widgets.Button(c, colors, i18n.T(i18n.KeySettingsSave),
							widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: a.OnUpdateWhale == nil}).Clicked() {
							value := st.PersonaText
							st.PersonaOpen = false
							update(whale.Patch{PersonaText: &value})
						}
					})
				})
			})

		// 长期记忆: a dialog with save and a second-confirmation clear.
		bodyPreferenceRow(c, "pet-whale-memory", i18n.T(i18n.KeySettingsWhaleMemory),
			i18n.T(i18n.KeySettingsWhaleMemoryHint), fg, muted, func() {
				ui.Row(c).FillWidth().Children(func() {
					ui.Text(c, memorySummary(cat.Memory)).Grow(1).MinWidth(0).SingleLine().
						MaxLines(1).FontSize(theme.InterfaceSize(13)).TextColor(muted)
					linkAction(c, i18n.T(i18n.KeySettingsWhaleMemoryOpen), muted, a.OnUpdateWhale != nil, func() {
						st.MemoryOpen, st.MemoryText, st.MemoryClear = true, cat.Memory, false
					})
				})
			})
		petWhaleMemoryDialog(c, colors, st, a, fg, muted)

		// 技能: inline list with a switch per skill.
		bodyPreferenceRow(c, "pet-whale-skills", i18n.T(i18n.KeySettingsWhaleSkills),
			i18n.T(i18n.KeySettingsWhaleSkillsHint), fg, muted, func() {
				if !st.SkillsOpen {
					ui.Row(c).FillWidth().Children(func() {
						ui.Text(c, skillsSummary(cat.Skills)).Grow(1).MinWidth(0).SingleLine().
							MaxLines(1).FontSize(theme.InterfaceSize(13)).TextColor(muted)
						linkAction(c, i18n.T(i18n.KeySettingsWhaleManage), muted, true, func() {
							st.SkillsOpen = true
						})
					})
					return
				}
				ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
					if len(cat.Skills) == 0 {
						ui.Text(c, i18n.T(i18n.KeySettingsWhaleSkillsEmpty)).FillWidth().
							FontSize(theme.InterfaceSize(12)).TextColor(muted)
						return
					}
					for _, skill := range cat.Skills {
						enabled := skill.Enabled
						row := ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Padding(4, 0)
						row.Children(func() {
							ui.Text(c, skill.Name).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).TextColor(fg)
							toggle := ui.Switch(c, &enabled).Size(44, 24).Label("Whale skill " + skill.Name).Disabled(a.OnToggleSkill == nil)
							if toggle.Changed() && a.OnToggleSkill != nil {
								a.OnToggleSkill(skill.Name, enabled)
							}
						})
					}
				})
			})
	})
}

// petWhaleMemoryDialog edits MEMORY.md. Clearing asks once more because the
// plugin overwrites the file with its empty note.
func petWhaleMemoryDialog(c *ui.Context, colors theme.Colors, st *petWhaleState, a *PetSettingsAccess, fg, muted ui.Color) {
	if !st.MemoryOpen {
		return
	}
	ui.DialogBase(c, &st.MemoryOpen, func(backdrop, panel ui.Element) {
		chrome.DialogSurface(c, backdrop, panel, colors, 560, 20).Gap(12).
			Label(i18n.T(i18n.KeySettingsWhaleMemoryDialog)).
			Children(func() {
				ui.Text(c, i18n.T(i18n.KeySettingsWhaleMemoryDialog)).FontSize(theme.InterfaceSize(16)).Bold().TextColor(fg)
				ui.Text(c, i18n.T(i18n.KeySettingsWhaleMemoryHint)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				ui.TextArea(c, &st.MemoryText).FillWidth().Lines(8, 18).
					Label(i18n.T(i18n.KeySettingsWhaleMemory))
				if st.MemoryClear {
					ui.Text(c, i18n.T(i18n.KeySettingsWhaleMemoryClearConfirm)).FontSize(theme.InterfaceSize(12)).
						TextColor(chrome.Color(colors.Danger))
				}
				ui.Row(c).FillWidth().Wrap().Gap(8).Justify(ui.End).Children(func() {
					clearLabel := i18n.T(i18n.KeySettingsWhaleMemoryClear)
					clearVariant := widgets.ButtonOutline
					if st.MemoryClear {
						clearLabel, clearVariant = i18n.T(i18n.KeySettingsWhaleMemoryClearConfirmAct), widgets.ButtonDanger
					}
					if widgets.Button(c, colors, clearLabel, widgets.ButtonSpec{
						Variant: clearVariant, Size: widgets.ButtonSmall, Disabled: a.OnClearMemory == nil,
					}).Clicked() {
						if st.MemoryClear {
							st.MemoryClear, st.MemoryOpen = false, false
							a.OnClearMemory()
						} else {
							st.MemoryClear = true
						}
					}
					ui.Box(c).Grow(1).MinWidth(0)
					if widgets.Button(c, colors, i18n.T(i18n.KeySettingsWhaleCancel),
						widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
						st.MemoryOpen, st.MemoryClear = false, false
					}
					if widgets.Button(c, colors, i18n.T(i18n.KeySettingsSave),
						widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonSmall, Disabled: a.OnReplaceMemory == nil}).Clicked() {
						text := st.MemoryText
						st.MemoryOpen = false
						a.OnReplaceMemory(text)
					}
				})
			})
	})
}

// linkAction is the reference's blue text action on a settings row.
func linkAction(c *ui.Context, label string, ink ui.Color, enabled bool, action func()) {
	button := ui.Button(c, "").Label(label).Height(24).Padding(0, 4).Shrink(0).BorderWidth(0).
		Background(ui.Transparent).TextColor(ink).Disabled(!enabled)
	button.Children(func() {
		ui.Text(c, label).SingleLine().FontSize(theme.InterfaceSize(13)).TextColor(ink)
	})
	if button.Clicked() && enabled {
		action()
	}
}

func personaSummary(text string) string {
	return summarise(text, 48)
}

func memorySummary(text string) string {
	return summarise(text, 60)
}

func summarise(text string, limit int) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit]) + "…"
}

func skillsSummary(skills []whale.Skill) string {
	if len(skills) == 0 {
		return ""
	}
	enabled := 0
	for _, skill := range skills {
		if skill.Enabled {
			enabled++
		}
	}
	return strconv.Itoa(enabled) + "/" + strconv.Itoa(len(skills))
}

// whaleWideRow is a preference row whose controls keep their natural width, for
// the identity pair that does not fit the shared 180 DIP column.
func whaleWideRow(c *ui.Context, key, title, hint string, fg, muted ui.Color, controls func()) {
	row := ui.Row(c.Key(key)).FillWidth().MinWidth(0).MinHeight(64).Gap(24).Padding(16, 0).AlignItems(ui.Center)
	row.Draw(func(p *ui.Painter, bounds ui.Rect) {
		p.Fill(ui.Rect{X: bounds.X, Y: bounds.Y, W: bounds.W, H: .5}, ui.RGBA(muted.R, muted.G, muted.B, .25), 0)
	})
	row.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, title).FillWidth().FontSize(theme.InterfaceSize(14)).TextColor(fg)
			if hint != "" {
				ui.Text(c, hint).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		})
		ui.Row(c).Shrink(0).Gap(8).AlignItems(ui.Center).Children(controls)
	})
}
