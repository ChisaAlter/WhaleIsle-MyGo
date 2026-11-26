package coss

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// CommandItem is one command of a CommandPalette.
type CommandItem struct {
	ID, Label string
	// Group heads a labelled group starting at this item.
	Group    string
	Icon     Icon
	Shortcut string
	// Keywords also match the query.
	Keywords string
	Disabled bool
}

// CommandOptions configure a CommandPalette.
type CommandOptions struct {
	Placeholder string
	// Empty is shown when nothing matches.
	Empty string
	// Footer builds the muted 12 px strip under the list (key hints).
	Footer func()
	Key    any
}

type commandState struct {
	highlight int
	query     string
}

// CommandPalette is coss's Command dialog: a 576 DIP-wide, 16 DIP-radius
// popup 10% down the window with a borderless large search input, the
// matching commands in an inner card (groups, separators, shortcuts) and an
// optional footer. Up/Down move the highlight, Enter runs it, Escape
// closes. It returns the chosen command's ID and closes.
func CommandPalette(c *ui.Context, open *bool, query *string, items []CommandItem, o CommandOptions) string {
	t := tok(c)
	chosen := ""
	withKey(c, o.Key)
	ui.DialogBaseWithOptions(c, open, ui.DialogOptions{}, func(back, panel ui.Element) {
		backdropLook(back, t, "command/command-dialog-backdrop")
		_, wh := c.Size()
		back.Column().Justify(ui.Start).AlignItems(ui.Center).Padding(max(design.Sp(4), wh*0.1), design.Sp(4))
		chosen = commandPanel(c, t, panel, query, items, o, true)
		if chosen != "" {
			*open = false
		}
	})
	return chosen
}

// CommandPanel draws the palette's popup in place (the gallery's, and
// pages that embed it).
func CommandPanel(c *ui.Context, query *string, items []CommandItem, o CommandOptions) string {
	t := tok(c)
	p := ui.Column(c)
	return commandPanel(c, t, p, query, items, o, false)
}

func commandMatches(it CommandItem, q string) bool {
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(it.Label+" "+it.Keywords+" "+it.Group), q)
}

func commandPanel(c *ui.Context, t *design.Tokens, panel ui.Element, query *string, items []CommandItem, o CommandOptions, focus bool) string {
	ww, _ := c.Size()
	dlgRadius := theme.ContextRadii(c).Dialog
	panel.Width(min(576, ww-2*design.Sp(4))).MaxHeight(420).Radius(dlgRadius).Background(t.Popover).Border(design.BorderWidth, t.Border).TextColor(t.PopoverForeground)
	var sh shadowList
	sh.scale(design.ShadowLg, black, 0.05)
	d := deco{border: design.BorderWidth, radius: dlgRadius}
	d.highlight(t, 0.04, 0.06)
	applyGlassMaterial(panel, t, t.Popover, 40, 0.50)
	applyGlassElevation(t, &sh, &d, glassTierModal)
	finish(panel, &sh, &d)
	motion(panel, "command/command-dialog-popup", 0, 0)
	st := ui.Local(panel, "coss.command", func() commandState { return commandState{} })
	q := strings.ToLower(strings.TrimSpace(*query))
	if q != st.query {
		st.query, st.highlight = q, 0
	}
	var shown []int
	for i, it := range items {
		if !it.Disabled && commandMatches(it, q) {
			shown = append(shown, i)
		}
	}
	if st.highlight >= len(shown) {
		st.highlight = max(0, len(shown)-1)
	}
	chosen := ""
	panel.Children(func() {
		// The search: a large (36) borderless autocomplete input with a
		// search icon, in 6×10 padding.
		ui.Row(c).FillWidth().Padding(design.Sp(1.5), design.Sp(2.5)).Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Box(c).Absolute().Left(1).Top(0).Bottom(0).Padding(0, 0, 0, design.Sp(3)-1).Center().PassThrough().Children(func() {
					icon(c, IconSearch, design.IconSM, t.Foreground, false).MarginX(-2)
				})
				in := ui.TextInputBase(c, query).FillWidth().Height(34).Padding(0, design.Sp(3)-1, 0, design.Sp(8)-1).
					FontSize(design.TextSM.Size).FixedLineHeight(34).TextColor(t.Foreground).FocusRing(false).Label(firstNonEmpty(o.Placeholder, "Search"))
				if focus {
					in.AutoFocus()
				}
				if *query == "" && o.Placeholder != "" {
					placeholder(c, t, o.Placeholder, design.Sp(8)-1, 0, 34, design.TextSM)
				}
				switch {
				case in.Shortcut(0, ui.KeyDown):
					if len(shown) > 0 {
						st.highlight = (st.highlight + 1) % len(shown)
					}
				case in.Shortcut(0, ui.KeyUp):
					if len(shown) > 0 {
						st.highlight = (st.highlight + len(shown) - 1) % len(shown)
					}
				case in.Submitted():
					if st.highlight < len(shown) {
						chosen = items[shown[st.highlight]].ID
					}
				}
			})
		})
		// The inner card (-mx-px, rounded top, border without bottom).
		innerBg := t.Popover
		if t.Glass {
			if t.IsDark() {
				innerBg = white.Alpha(0.03)
			} else {
				innerBg = white.Alpha(0.24)
			}
		}
		inner := ui.Column(c).FillWidth().Grow(1).MinHeight(0).Margin(0, -1, 0, -1).Background(innerBg).
			BorderWidth(design.BorderWidth, design.BorderWidth, 0, design.BorderWidth).BorderColor(t.Border)
		if o.Footer != nil {
			inner.Radius(design.RadiusXL, design.RadiusXL, 0, 0)
		} else {
			inner.Radius(design.RadiusXL, design.RadiusXL, dlgRadius-1, dlgRadius-1)
		}
		var ish shadowList
		ish.scale(design.ShadowXs, black, 0.05)
		finish(inner, &ish, nil)
		inner.Children(func() {
			ScrollArea(c, ScrollAreaOptions{}, func() {
				ui.Column(c).FillWidth().Padding(design.Sp(2)).Role(ui.RoleList).Children(func() {
					if len(shown) == 0 {
						if o.Empty != "" {
							ui.Row(c).FillWidth().Justify(ui.Center).PaddingY(design.Sp(6)).Children(func() {
								text(c, o.Empty, design.TextSM, design.WeightNormal, t.MutedForeground)
							})
						}
						return
					}
					group := ""
					for k, idx := range shown {
						it := items[idx]
						if it.Group != "" && it.Group != group {
							if group != "" {
								ui.Box(c).Height(1).Margin(design.Sp(2), design.Sp(2)).Background(t.Border)
							}
							group = it.Group
							ui.Row(c).Padding(design.Sp(1.5), design.Sp(2)).Children(func() {
								text(c, it.Group, design.TextXS, design.WeightMedium, t.MutedForeground)
							})
						}
						c.Key(it.ID)
						row := ui.ButtonBase(c).FillWidth().MinHeight(28).Radius(design.RadiusSM).Gap(design.Sp(2)).Padding(design.Sp(1.5), design.Sp(2)).
							Justify(ui.Start).FocusRing(false).Role(ui.RoleListItem)
						if row.Hovered() && forced == 0 {
							st.highlight = k
						}
						fg := t.Foreground
						if k == st.highlight {
							row.Background(t.Accent)
							fg = t.AccentForeground
						}
						row.Children(func() {
							if it.Icon != IconNone {
								icon(c, it.Icon, design.IconSM, fg, false)
							}
							text(c, it.Label, design.TextSM, design.WeightNormal, fg)
							if it.Shortcut != "" {
								ui.Spacer(c)
								text(c, it.Shortcut, design.TextXS, design.WeightMedium, t.MutedForeground.Alpha(design.OpacityPlaceholder)).
									LetterSpacing(design.TrackingWidest * design.TextXS.Size)
							}
						})
						if row.Clicked() {
							chosen = it.ID
						}
					}
				})
			}).FillWidth().Grow(1).MinHeight(0)
		})
		if o.Footer != nil {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(design.Sp(2)).Padding(design.Sp(3), design.Sp(5)).
				BorderWidth(design.BorderWidth, 0, 0, 0).BorderColor(t.Border).Radius(0, 0, dlgRadius-1, dlgRadius-1).
				TextColor(t.MutedForeground).FontSize(design.TextXS.Size).Children(o.Footer)
		}
	})
	return chosen
}
