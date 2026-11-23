package coss

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// MenuItemKind is what a menu row is.
type MenuItemKind uint8

const (
	MenuAction MenuItemKind = iota
	MenuCheckbox
	MenuRadio
	MenuSeparator
	MenuLabel
	MenuSub
)

// MenuItem is one row of a Menu, ContextMenu or MenuPanel.
type MenuItem struct {
	Kind MenuItemKind
	// ID is what the menu returns when the row is chosen; Label is shown.
	ID       string
	Label    string
	Icon     Icon
	Shortcut string
	// Destructive colors an action destructive-foreground.
	Destructive bool
	Disabled    bool
	// Inset indents the row as if it had an icon (ps-8).
	Inset bool
	// Checked is a checkbox row's value; Switch draws it as a switch.
	Checked *bool
	Switch  bool
	// Group and Value make a radio row: it sets *Group to Value.
	Group *string
	Value string
	// Items are a submenu's rows.
	Items []MenuItem
}

// MenuOptions configure a Menu.
type MenuOptions struct {
	Side  Side
	Align Align
	// MinWidth overrides the 128 DIP minimum (min-w-32).
	MinWidth float32
	Label    string
}

// menuState is what a menu panel keeps between frames: the open submenu.
type menuState struct {
	sub int
}

// Menu is coss's Menu, opened from trigger while *open: rows of actions,
// checkbox and radio items, labels, separators, shortcuts and submenus.
// Up/Down/Home/End move among rows, Enter and Space choose, Right opens a
// submenu and Left closes it, Escape or a press outside closes the menu. It
// returns the ID of the row chosen in this frame, "" otherwise; actions
// close the menu, checkbox and radio rows keep it open.
func Menu(c *ui.Context, trigger ui.Element, open *bool, items []MenuItem, o MenuOptions) string {
	t := tok(c)
	chosen := ""
	trigger.Expanded(*open)
	if trigger.Clicked() {
		*open = !*open
	}
	align := o.Align
	ui.PopoverBase(c, trigger, open, func(panel ui.Element) {
		place(panel, trigger, o.Side, align, 4)
		chosen = menuPanel(c, t, panel, items, o, open, 0)
	})
	return chosen
}

// MenuPanel draws a menu's panel in place (the command palette's and the
// gallery's); it returns the chosen row's ID.
func MenuPanel(c *ui.Context, items []MenuItem, o MenuOptions) string {
	t := tok(c)
	p := ui.Column(c)
	return menuPanel(c, t, p, items, o, nil, 0)
}

// ContextMenu opens items at the pointer when target is right-clicked (or
// the menu key / Shift+F10 pressed while it has the focus), with the menu's
// look and keys. It returns the chosen row's ID.
func ContextMenu(c *ui.Context, target ui.Element, items []MenuItem, o MenuOptions) string {
	t := tok(c)
	st := ui.Local(target, "coss.context", func() contextState { return contextState{} })
	if target.RightClicked() {
		x, y, _ := target.PointerPosition()
		b := target.Bounds()
		st.open, st.x, st.y = true, b.X+x, b.Y+y
	}
	if target.Shortcut(ui.Shift, ui.KeyF10) {
		b := target.Bounds()
		st.open, st.x, st.y = true, b.X, b.Y+b.H
	}
	if !st.open {
		return ""
	}
	chosen := ""
	ui.Overlay(c, func() {
		panel := ui.Column(c).Absolute().Left(st.x).Top(st.y).Role(ui.RoleMenu)
		chosen = menuPanel(c, t, panel, items, o, &st.open, 0)
		if panel.OverlayShortcut(0, ui.KeyEscape) || panel.PressedOutside() {
			st.open = false
		}
	})
	return chosen
}

type contextState struct {
	open bool
	x, y float32
}

func menuPanel(c *ui.Context, t *design.Tokens, panel ui.Element, items []MenuItem, o MenuOptions, open *bool, depth int) string {
	panelLook(panel, t, design.RadiusLG, design.ShadowLg)
	minW := o.MinWidth
	if minW == 0 {
		minW = 128
	}
	panel.MinWidth(minW).Role(ui.RoleMenu)
	if o.Label != "" {
		panel.Label(o.Label)
	}
	st := ui.Local(panel, "coss.menu", func() menuState { return menuState{sub: -1} })
	chosen := ""
	panel.Children(func() {
		list := ui.Column(c).FillWidth().Padding(design.Sp(1)).FocusGroup(ui.Vertical)
		list.Children(func() {
			first := true
			for i := range items {
				it := &items[i]
				switch it.Kind {
				case MenuSeparator:
					ui.Box(c).Key(i).Height(1).Margin(design.Sp(1), design.Sp(2)).Background(t.Border)
					continue
				case MenuLabel:
					pad := design.Sp(2)
					if it.Inset {
						pad = design.Sp(8)
					}
					ui.Row(c).Key(i).Padding(design.Sp(1.5), design.Sp(2), design.Sp(1.5), pad).Children(func() {
						text(c, it.Label, design.TextXS, design.WeightMedium, t.MutedForeground)
					})
					continue
				}
				if id := menuRow(c, t, it, i, first && depth > 0, st, open, o, depth); id != "" {
					chosen = id
				}
				first = false
			}
		})
	})
	return chosen
}

// menuRow builds one interactive row and returns its ID when chosen.
func menuRow(c *ui.Context, t *design.Tokens, it *MenuItem, i int, autofocus bool, st *menuState, open *bool, o MenuOptions, depth int) string {
	c.Key(i)
	row := ui.ButtonBase(c).FillWidth().MinHeight(28).Radius(design.RadiusSM).Gap(design.Sp(2)).FocusRing(false).Justify(ui.Start)
	if it.Label != "" {
		row.Label(it.Label)
	}
	if autofocus {
		row.AutoFocus()
	}
	s := interact(row, isDisabled(it.Disabled))
	highlighted := s.Has(StateHover) || row.Focused() || s.Has(StateHighlighted)
	subOpen := it.Kind == MenuSub && st.sub == i
	fg := t.Foreground
	if it.Destructive {
		fg = t.DestructiveForeground
	}
	if highlighted || subOpen {
		row.Background(t.Accent)
		if !it.Destructive {
			fg = t.AccentForeground
		}
	}
	switch it.Kind {
	case MenuCheckbox, MenuRadio:
		if it.Switch {
			row.Padding(design.Sp(1), design.Sp(1.5), design.Sp(1), design.Sp(2)).Gap(design.Sp(4))
		} else {
			row.Padding(design.Sp(1), design.Sp(4), design.Sp(1), design.Sp(2))
		}
	default:
		left := design.Sp(2)
		if it.Inset {
			left = design.Sp(8)
		}
		row.Padding(design.Sp(1), design.Sp(2), design.Sp(1), left)
	}
	switch it.Kind {
	case MenuCheckbox:
		row.Role(ui.RoleMenuItemCheckBox)
		if it.Checked != nil {
			row.Checked(*it.Checked)
		}
	case MenuRadio:
		row.Role(ui.RoleMenuItemRadio)
		if it.Group != nil {
			row.Checked(*it.Group == it.Value)
		}
	case MenuSub:
		row.Role(ui.RoleMenuItem).Expanded(subOpen)
	default:
		row.Role(ui.RoleMenuItem)
	}
	disabledLook(row, s)
	row.Children(func() {
		switch it.Kind {
		case MenuCheckbox, MenuRadio:
			on := false
			if it.Checked != nil {
				on = *it.Checked
			}
			if it.Group != nil {
				on = *it.Group == it.Value
			}
			if it.Switch {
				text(c, it.Label, design.TextSM, design.WeightNormal, fg).Grow(1)
				menuSwitch(c, t, on)
				return
			}
			// grid-cols-[.75rem_1fr]: the indicator column, -ms-0.5.
			ui.Box(c).Width(12).Shrink(0).Margin(0, 0, 0, -2).Children(func() {
				if on {
					if it.Kind == MenuRadio {
						ui.Box(c).Size(16, 16).Center().Children(func() {
							ui.Box(c).Size(6, 6).Radius(design.RadiusFull).Background(fg)
						})
					} else {
						icon(c, IconCheck, design.IconSM, fg, true)
					}
				}
			})
			text(c, it.Label, design.TextSM, design.WeightNormal, fg)
			return
		}
		if it.Icon != IconNone {
			icon(c, it.Icon, design.IconSM, fg, false).MarginX(-2)
		}
		text(c, it.Label, design.TextSM, design.WeightNormal, fg)
		if it.Shortcut != "" || it.Kind == MenuSub {
			ui.Spacer(c)
		}
		if it.Shortcut != "" {
			text(c, it.Shortcut, design.TextXS, design.WeightMedium, t.MutedForeground.Alpha(design.OpacityPlaceholder)).
				LetterSpacing(design.TrackingWidest * design.TextXS.Size)
		}
		if it.Kind == MenuSub {
			icon(c, IconChevronRight, design.IconSM, fg, false).Margin(0, -2, 0, 0)
		}
	})
	chosen := ""
	if s.Has(StateDisabled) {
		return ""
	}
	activate := row.Clicked()
	switch it.Kind {
	case MenuSub:
		if activate || row.Shortcut(0, ui.KeyRight) || s.Has(StateHover) {
			st.sub = i
		}
		if st.sub == i {
			ui.PopoverBase(c, row, &subOpenFlag, func(panel ui.Element) {
				// side inline-end, sideOffset 0, align start, offset by the
				// popup's padding and border.
				panel.AttachTo(row, ui.AnchorTopRight, ui.AnchorTopLeft).Margin(-design.Sp(1)-1, 0, 0, design.Sp(1))
				sub := subOpenFlag
				id := menuPanel(c, t, panel, it.Items, MenuOptions{}, open, depth+1)
				if id != "" {
					chosen = id
				}
				if panel.Shortcut(0, ui.KeyLeft) || panel.OverlayShortcut(0, ui.KeyEscape) || !sub {
					st.sub = -1
				}
			})
			subOpenFlag = true
		}
		return chosen
	case MenuCheckbox:
		if activate && it.Checked != nil {
			*it.Checked = !*it.Checked
			chosen = it.ID
		}
	case MenuRadio:
		if activate && it.Group != nil {
			*it.Group = it.Value
			chosen = it.ID
		}
	default:
		if s.Has(StateHover) && st.sub >= 0 {
			st.sub = -1
		}
		if activate {
			chosen = it.ID
			if open != nil {
				*open = false
			}
		}
	}
	return chosen
}

// subOpenFlag is the open flag a submenu's PopoverBase takes; the parent
// panel's state decides which submenu shows.
var subOpenFlag = true

// menuSwitch is the switch of a switch-variant checkbox row: 14×22 track,
// 12 DIP thumb, inset black/4 shadow.
func menuSwitch(c *ui.Context, t *design.Tokens, on bool) {
	const thumb = 12
	track := ui.Box(c).Size(2*thumb-2, thumb+2).Padding(1).Radius(design.RadiusFull).Shrink(0)
	if on {
		track.Background(t.Primary)
	} else {
		track.Background(t.Input)
	}
	var sh shadowList
	sh.inset(0, 1, 0, 0, black.Alpha(0.04))
	d := deco{radius: design.RadiusFull}
	finish(track, &sh, &d)
	track.Children(func() {
		th := ui.Box(c).Size(thumb, thumb).Radius(design.RadiusFull).Background(t.Background).Margin(0, 0, 0, b2f(on)*(thumb-4))
		var ts shadowList
		ts.scale(design.ShadowSm, black, 0.05)
		finish(th, &ts, nil)
	})
}
