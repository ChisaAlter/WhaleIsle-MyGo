package coss

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// SelectOption is one choice of a Select or Combobox.
type SelectOption struct {
	Value, Label string
	Disabled     bool
	// Group starts a labelled group at this option (shown above it, after
	// a separator unless it is the first).
	Group string
}

func (o SelectOption) label() string {
	if o.Label != "" {
		return o.Label
	}
	return o.Value
}

// SelectOptions configure a Select.
type SelectOptions struct {
	Size        InputSize
	Placeholder string
	Disabled    bool
	Invalid     bool
	Label       string
	Key         any
}

// Select is coss's Select: a 32 DIP trigger (InputSM 28, InputLG 36,
// at least 144 wide) showing the choice and chevrons, opening a popup of
// options below it (4 DIPs away, at least as wide) whose chosen row shows a
// check. The keys are the toolkit's SelectBase's: arrows open and move,
// Enter/Space choose, Escape closes. It returns the trigger (Changed reports
// a new choice).
func Select(c *ui.Context, selected *string, options []SelectOption, o SelectOptions) ui.Element {
	io := InputOptions{Label: o.Label, Invalid: o.Invalid, Disabled: o.Disabled}
	io.fromField()
	t := tok(c)
	withKey(c, o.Key)
	parts := ui.SelectBase(c, selected)
	tr := parts.Trigger
	h, padX, gap := float32(32), design.Sp(3)-1, design.Sp(2)
	switch o.Size {
	case InputSM:
		h, padX, gap = 28, design.Sp(2.5)-1, design.Sp(1.5)
	case InputLG:
		h = 36
	}
	tr.MinHeight(h).PaddingX(padX).Gap(gap).Justify(ui.SpaceBetween).FocusRing(false)
	if io.Label != "" {
		tr.Label(io.Label)
	}
	s := interact(tr, isDisabled(io.Disabled))
	if io.Invalid {
		s |= StateInvalid
	}
	if parts.Open() {
		s |= StateOpen
	}
	if s.Has(StatePressed) {
		s &^= StateHover
	}
	selectLook(tr, t, s)
	joined(tr, design.RadiusLG, ui.Color{})
	disabledLook(tr, s)
	current, placeholder := "", true
	for _, op := range options {
		if op.Value == *selected {
			current, placeholder = op.label(), false
		}
	}
	if placeholder {
		// A value the option list does not know still shows itself; the
		// placeholder only stands in for no selection at all.
		current, placeholder = *selected, *selected == ""
		if placeholder {
			current = o.Placeholder
		}
	}
	tr.Children(func() {
		col := t.Foreground
		if placeholder {
			col = t.MutedForeground.Alpha(design.OpacityPlaceholder)
		}
		// Keep a 144 DIP default trigger width when unsized without stealing
		// horizontal flex space from the label when the caller sets an explicit
		// width (e.g. 88 DIP) or places the trigger in a narrow container:
		// a Column gives FlipText 100% of the available label slot while a
		// zero-height strut beneath it supplies the default intrinsic floor.
		floorW := max(0, 144-2*padX-gap-design.IconSM+design.Sp(1))
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).ClipX().Children(func() {
			FlipText(c, current, FlipTextOptions{
				Key:        "coss-select-trigger",
				Color:      col,
				FontSize:   design.TextSM.Size,
				LineHeight: design.TextSM.LineHeight,
				FontWeight: design.WeightNormal,
			}).FillWidth().MinWidth(0)
			if floorW > 0 {
				ui.Box(c).Width(floorW).Height(0)
			}
		})
		icon(c, IconChevronsUpDown, design.IconSM, t.Foreground, false).Margin(0, -design.Sp(1), 0, 0)
	})
	parts.Popup(func(panel ui.Element) {
		panel.Margin(4, 0, 0, 0)
		panel = selectPopoverReveal(c, t, panel)
		panelLook(panel, t, theme.ContextRadii(c).MD, design.ShadowLg)
		panel.Children(func() {
			ui.Column(c).Padding(design.Sp(1)).FillWidth().Children(func() {
				for i, op := range options {
					groupHeader(c, t, op.Group, i)
					item := parts.Item(op.Value)
					listRow(c, t, item, op.label(), op.Value == *selected, item.Highlighted(), isDisabled(op.Disabled), true)
				}
			})
		})
	})
	return tr
}

// selectLook is the select trigger's frame: the text control's, where the
// focus ring needs focus-visible and a press drops the shadows.
func selectLook(e ui.Element, t *design.Tokens, s State) {
	if s.Has(StatePressed) && !s.Has(StateFocusVisible) && !s.Has(StateInvalid) {
		// [data-pressed]:shadow-none and no ::before highlight.
		bg := t.Background
		if t.IsDark() {
			bg = t.Input.Alpha(0.32)
		}
		e.Background(bg).Border(design.BorderWidth, t.Input).Radius(design.RadiusLG)
		return
	}
	controlLook(e, t, s, design.RadiusLG)
}

// groupHeader draws a group label (and the separator before it) at the
// first option of a group.
func groupHeader(c *ui.Context, t *design.Tokens, group string, i int) {
	if group == "" {
		return
	}
	if i > 0 {
		ui.Box(c).Height(1).Margin(design.Sp(1), design.Sp(2)).Background(t.Border)
	}
	ui.Row(c).Padding(design.Sp(1.5), design.Sp(2)).Children(func() {
		text(c, group, design.TextXS, design.WeightMedium, t.MutedForeground)
	})
}

// listRow styles a select/combobox option: min 28 high, radius 6, a 16 DIP
// indicator column with a check when chosen (when indicator), 8 DIPs to the
// label; accent while highlighted.
func listRow(c *ui.Context, t *design.Tokens, item ui.Element, label string, chosen, highlighted, disabled, indicator bool) {
	item.MinHeight(28).Radius(design.RadiusSM).Gap(design.Sp(2)).AlignItems(ui.Center)
	if indicator {
		item.Padding(design.Sp(1), design.Sp(4), design.Sp(1), design.Sp(2))
	} else {
		item.Padding(design.Sp(1), design.Sp(2))
	}
	fg := t.Foreground
	if highlighted {
		item.Background(t.Accent)
		fg = t.AccentForeground
	}
	if disabled {
		item.Disabled(true).DisabledOpacity(design.OpacityDisabled)
	}
	item.Children(func() {
		if indicator {
			ui.Box(c).Size(16, 16).Shrink(0).Children(func() {
				if chosen {
					icon(c, IconCheck, design.IconSM, fg, true)
				}
			})
		}
		text(c, label, design.TextSM, design.WeightNormal, fg).Shrink(1).MinWidth(0)
	})
}

// ComboboxOptions configure a Combobox or Autocomplete.
type ComboboxOptions struct {
	Size        InputSize
	Placeholder string
	Disabled    bool
	Invalid     bool
	Label       string
	// StartIcon goes inside the input's start (a search icon).
	StartIcon Icon
	// Empty is shown when no option matches.
	Empty string
	Key   any
}

// Combobox is coss's Combobox: an Input with a chevron trigger whose popup
// lists the options matching what is typed; choosing one sets *selected
// (and the text to its label). It returns the text input.
func Combobox(c *ui.Context, selected *string, options []SelectOption, o ComboboxOptions) ui.Element {
	return combo(c, selected, options, o, true)
}

// Autocomplete is coss's Autocomplete: free text in *value, with
// suggestions matching it in a popup; choosing one fills the text.
func Autocomplete(c *ui.Context, value *string, suggestions []string, o ComboboxOptions) ui.Element {
	opts := make([]SelectOption, len(suggestions))
	for i, s := range suggestions {
		opts[i] = SelectOption{Value: s}
	}
	return combo(c, value, opts, o, false)
}

type comboState struct {
	text         string
	init         bool
	lastSelected string
	lastLabel    string
}

func comboLabelFor(options []SelectOption, selected string) string {
	for _, op := range options {
		if op.Value == selected {
			return op.label()
		}
	}
	return selected
}

func combo(c *ui.Context, selected *string, options []SelectOption, o ComboboxOptions, pick bool) ui.Element {
	io := InputOptions{Label: o.Label, Invalid: o.Invalid, Disabled: o.Disabled}
	io.fromField()
	t := tok(c)
	h, padX := o.Size.inner()
	withKey(c, o.Key)
	control := ui.Row(c).FillWidth().MinWidth(0).AlignItems(ui.Center)
	st := ui.Local(control, "coss.combo", func() comboState { return comboState{} })
	textp := selected
	if pick {
		want := comboLabelFor(options, *selected)
		if !st.init || *selected != st.lastSelected || want != st.lastLabel {
			st.text = want
			st.lastSelected = *selected
			st.lastLabel = want
			st.init = true
		}
		textp = &st.text
	}
	var in ui.Element
	disabled := isDisabled(io.Disabled)
	control.Children(func() {
		startPad := padX
		if o.StartIcon != IconNone {
			startPad = design.Sp(8) - 1
			if o.Size == InputSM {
				startPad = design.Sp(7) - 1
			}
			ui.Box(c).Absolute().Left(1).Top(0).Bottom(0).Padding(0, 0, 0, padX).Center().PassThrough().Children(func() {
				icon(c, o.StartIcon, design.IconSM, t.Foreground, false).MarginX(-2)
			})
		}
		endPad := padX
		if pick {
			endPad = 28
			if o.Size == InputSM {
				endPad = 26
			}
		}
		cb := ui.ComboboxBase(c, textp)
		in = cb.Input.FillWidth().MinWidth(0).Height(h).Padding(0, endPad, 0, startPad).
			FontSize(design.TextSM.Size).FixedLineHeight(h).TextColor(t.Foreground).FocusRing(false)
		if io.Label != "" {
			in.Label(io.Label)
		}
		if *textp == "" && o.Placeholder != "" {
			in.Placeholder(o.Placeholder)
			placeholder(c, t, o.Placeholder, startPad, 0, h, design.TextSM)
		}
		q := strings.ToLower(strings.TrimSpace(*textp))
		filtering := cb.Filtering()
		cb.Popup(func(panel ui.Element) {
			panel.Margin(4, 0, 0, 0)
			panel = selectPopoverReveal(c, t, panel)
			panelLook(panel, t, theme.ContextRadii(c).MD, design.ShadowLg)
			panel.Children(func() {
				ui.Column(c).Padding(design.Sp(1)).FillWidth().Children(func() {
					shown := 0
					for i, op := range options {
						if filtering && q != "" && !strings.Contains(strings.ToLower(op.label()), q) {
							continue
						}
						shown++
						groupHeader(c, t, op.Group, i)
						item := cb.Item(op.Value)
						listRow(c, t, item, op.label(), pick && op.Value == *selected, item.Highlighted(), isDisabled(op.Disabled), pick)
					}
					if shown == 0 && o.Empty != "" {
						ui.Row(c).Padding(design.Sp(2)).Justify(ui.Center).Children(func() {
							text(c, o.Empty, design.TextSM, design.WeightNormal, t.MutedForeground)
						})
					}
				})
			})
		})
		if v, ok := cb.Chosen(); ok {
			*selected = v
			if pick {
				want := comboLabelFor(options, v)
				st.text = want
				st.lastSelected = v
				st.lastLabel = want
			}
		} else if pick && !in.Focused() && !cb.Open() {
			want := comboLabelFor(options, *selected)
			if st.text != want {
				st.text = want
				st.lastSelected = *selected
				st.lastLabel = want
				c.AnimationFrame()
			}
		}
		if pick {
			trig := ui.ButtonBase(c).Absolute().Right(2).Top(0).Bottom(0).Width(28).Radius(design.RadiusMD).FocusRing(false).
				Label("Show options").Cursor(ui.CursorPointer)
			ts := interact(trig, disabled)
			a := float32(0.8)
			if ts.Has(StateHover) {
				a = 1
			}
			trig.Children(func() { icon(c, IconChevronsUpDown, design.IconSM, t.Foreground, true).Opacity(a) })
			if trig.Clicked() {
				nextOpen := !cb.Open()
				cb.SetOpen(nextOpen)
				if nextOpen {
					in.Focus()
				}
			}
		}
	})
	s := interactField(in, disabled)
	if io.Invalid {
		s |= StateInvalid
	}
	controlLook(control, t, s, design.RadiusLG)
	disabledLook(control, s)
	return in
}
