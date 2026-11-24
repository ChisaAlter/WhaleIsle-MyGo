package coss

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// CheckOptions configure a Checkbox, Radio or Switch.
type CheckOptions struct {
	Disabled bool
	Invalid  bool
	// Indeterminate shows a checkbox as partly on (a minus, no fill).
	Indeterminate bool
	// Label names the control when label is empty.
	Label string
	Key   any
}

// boxLook is the frame of checkboxes and radios: 16 DIPs, border input,
// background (input/32 in dark unless checked), shadow-xs/5 and the
// ::before highlight while unchecked, valid and enabled.
func boxLook(e ui.Element, t *design.Tokens, s State, radius float32) {
	checked := s.Has(StateChecked)
	bg := t.Background
	if t.IsDark() && !checked {
		bg = t.Input.Alpha(0.32)
	}
	bc := t.Input
	invalid, focus, disabled := s.Has(StateInvalid), s.Has(StateFocusVisible), s.Has(StateDisabled)
	if invalid {
		bc = t.Destructive.Alpha(0.36)
		if focus {
			bc = t.Destructive.Alpha(design.InvalidBorderAlpha)
		}
	}
	var sh shadowList
	var d deco
	amt := ringAmount(e, focus)
	if amt > 0 {
		ring := t.Ring
		if invalid {
			ring = t.Destructive.Alpha(0.48)
			if t.IsDark() {
				ring = t.Destructive.Alpha(0.24)
			}
		}
		sh.add(0, 0, 0, design.RingOffset, t.Background.Alpha(amt))
		sh.add(0, 0, 0, design.RingOffset+design.RingWidth, ring.Alpha(amt))
	}
	if !disabled && !checked && !invalid {
		sh.scale(design.ShadowXs, black, 0.05)
		d.highlight(t, 0.04, 0.06)
	}
	e.Size(16, 16).Radius(radius).Background(bg).Border(design.BorderWidth, bc)
	d.border, d.radius = design.BorderWidth, radius
	finish(e, &sh, &d)
}

// checkLabel lays a control and its label out as coss's Label does
// (inline-flex, gap 8, 14/16 medium).
func checkLabel(c *ui.Context, t *design.Tokens, label string) {
	if label != "" {
		text(c, label, design.TextStyle{Size: design.TextSM.Size, LineHeight: 16}, design.WeightMedium, t.Foreground)
	}
}

// Checkbox is coss's Checkbox toggling *checked, with label beside it. It
// returns the checkbox row (Changed reports a toggle).
func Checkbox(c *ui.Context, checked *bool, label string, o CheckOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	row := ui.CheckboxBase(c, checked).Gap(design.Sp(2)).AlignItems(ui.Center).FocusRing(false).Cursor(ui.CursorPointer)
	if label == "" && o.Label != "" {
		row.Label(o.Label)
	}
	if o.Indeterminate {
		row.Mixed()
	}
	s := interact(row, isDisabled(o.Disabled))
	if *checked && !o.Indeterminate {
		s |= StateChecked
	}
	if o.Invalid {
		s |= StateInvalid
	}
	row.Children(func() {
		box := ui.Box(c).Center().Shrink(0)
		boxLook(box, t, s, 4)
		if s.Has(StateDisabled) {
			box.Opacity(design.OpacityDisabled)
		}
		box.Children(func() {
			switch {
			case o.Indeterminate:
				icon(c, IconCheckboxMinus, 12, t.Foreground, true)
			case *checked:
				// The indicator covers the border (-inset-px) in primary.
				ui.Box(c).Absolute().Left(-1).Top(-1).Size(16, 16).Radius(4).Background(t.Primary).Center().Children(func() {
					icon(c, IconCheckboxCheck, 12, t.PrimaryForeground, true)
				})
			}
		})
		checkLabel(c, t, label)
	})
	disabledRow(row, s)
	return row
}

// CheckboxGroup lays checkboxes out in a column 12 DIPs apart, named label
// for assistive technology.
func CheckboxGroup(c *ui.Context, label string, fn func()) ui.Element {
	return ui.Column(c).AlignItems(ui.Start).Gap(design.Sp(3)).Role(ui.RoleGroup).Label(label).Children(fn)
}

// RadioGroup lays radios out in a column 12 DIPs apart; arrows move among
// them as one Tab stop.
func RadioGroup(c *ui.Context, label string, fn func()) ui.Element {
	return ui.Column(c).Gap(design.Sp(3)).Role(ui.RoleRadioGroup).Label(label).FocusGroup(ui.Vertical).Children(fn)
}

// Radio is coss's Radio choosing value into *selected.
func Radio[T comparable](c *ui.Context, selected *T, value T, label string, o CheckOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	row := ui.RadioBase(c, selected, value).Gap(design.Sp(2)).AlignItems(ui.Center).FocusRing(false).Cursor(ui.CursorPointer)
	if label == "" && o.Label != "" {
		row.Label(o.Label)
	}
	s := interact(row, isDisabled(o.Disabled))
	on := *selected == value
	if on {
		s |= StateChecked
	}
	if o.Invalid {
		s |= StateInvalid
	}
	row.Children(func() {
		box := ui.Box(c).Center().Shrink(0)
		boxLook(box, t, s, design.RadiusFull)
		if s.Has(StateDisabled) {
			box.Opacity(design.OpacityDisabled)
		}
		box.Children(func() {
			if on {
				ui.Box(c).Absolute().Left(-1).Top(-1).Size(16, 16).Radius(design.RadiusFull).Background(t.Primary).Center().Children(func() {
					ui.Box(c).Size(6, 6).Radius(design.RadiusFull).Background(t.PrimaryForeground)
				})
			}
		})
		checkLabel(c, t, label)
	})
	disabledRow(row, s)
	return row
}

// Switch is coss's Switch: an 18×30 track (input, primary when on) with a
// 16 DIP thumb sliding 12 DIPs over 150 ms.
func Switch(c *ui.Context, on *bool, label string, o CheckOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	row := ui.SwitchBase(c, on).Gap(design.Sp(2)).AlignItems(ui.Center).FocusRing(false).Cursor(ui.CursorPointer)
	if label == "" && o.Label != "" {
		row.Label(o.Label)
	}
	s := interact(row, isDisabled(o.Disabled))
	const thumb = 16
	row.Children(func() {
		track := ui.Box(c).Size(2*thumb-2, thumb+2).Padding(1).Radius(design.RadiusFull).Shrink(0).
			Transition(ui.ElementTransition{Colors: true, Duration: design.DurationPanel, Ease: design.EaseInOut})
		if *on {
			track.Background(t.Primary)
		} else {
			track.Background(t.Input)
		}
		var sh shadowList
		sh.ring(t, ringAmount(track, s.Has(StateFocusVisible)))
		finish(track, &sh, nil)
		if s.Has(StateDisabled) {
			track.Opacity(design.OpacityDisabled)
		}
		x := b2f(*on) * (thumb - 4)
		if forced == 0 {
			x = track.AnimateWith("coss.thumb", x, 150*1e6, design.EaseInOut)
		}
		w := float32(thumb)
		if s.Has(StatePressed) {
			w = thumb * 1.1
			if *on {
				x -= thumb * 0.1
			}
		}
		track.Children(func() {
			th := ui.Box(c).Size(w, thumb).Radius(design.RadiusFull).Background(t.Background).Margin(0, 0, 0, x)
			var ts shadowList
			ts.scale(design.ShadowSm, black, 0.05)
			finish(th, &ts, nil)
		})
		checkLabel(c, t, label)
	})
	disabledRow(row, s)
	return row
}

// ToggleVariant is coss's toggleVariants.variant.
type ToggleVariant uint8

const (
	ToggleDefault ToggleVariant = iota
	ToggleOutline
)

// ToggleSize is coss's toggleVariants.size.
type ToggleSize uint8

const (
	ToggleMD ToggleSize = iota // 32
	ToggleSM                   // 28
	ToggleLG                   // 36
)

// ToggleOptions configure a Toggle.
type ToggleOptions struct {
	Variant  ToggleVariant
	Size     ToggleSize
	Icon     Icon
	Disabled bool
	Label    string
	Key      any
	// joined is set by ToggleGroup's outline variant: which outer corners
	// stay round, and whether the left border is dropped.
	joined     bool
	first      bool
	last       bool
	vertical   bool
	groupColor bool
}

func (s ToggleSize) metrics() (h, padX float32) {
	switch s {
	case ToggleSM:
		return 28, design.Sp(1.5) - 1
	case ToggleLG:
		return 36, design.Sp(2.5) - 1
	}
	return 32, design.Sp(2) - 1
}

// Toggle is coss's Toggle: a button that stays pressed while *on.
func Toggle(c *ui.Context, on *bool, label string, o ToggleOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	h, padX := o.Size.metrics()
	e := ui.ToggleBase(c, on).Height(h).MinWidth(h).PaddingX(padX).Gap(design.Sp(2)).FocusRing(false).Cursor(ui.CursorPointer)
	if label == "" && o.Label != "" {
		e.Label(o.Label)
	}
	s := interact(e, isDisabled(o.Disabled))
	pressed := *on || s.Has(StatePressed)
	dark := t.IsDark()
	r := [4]float32{design.RadiusLG, design.RadiusLG, design.RadiusLG, design.RadiusLG}
	if o.joined {
		if !o.first {
			if o.vertical {
				r[0], r[1] = 0, 0
			} else {
				r[0], r[3] = 0, 0
			}
		}
		if !o.last {
			if o.vertical {
				r[2], r[3] = 0, 0
			} else {
				r[1], r[2] = 0, 0
			}
		}
	}
	e.Radius(r[0], r[1], r[2], r[3])
	var bg, bc ui.Color
	fg := t.Foreground
	var sh shadowList
	var d deco
	sh.ring(t, ringAmount(e, s.Has(StateFocusVisible)))
	if s.Has(StateHover) {
		bg = t.Accent
	}
	switch o.Variant {
	case ToggleDefault:
		if pressed {
			bg, fg = t.Input.Alpha(0.64), t.AccentForeground
		}
	case ToggleOutline:
		bc = t.Input
		bg = t.Background
		if dark {
			bg = t.Input.Alpha(0.32)
			if s.Has(StateHover) {
				bg = t.Input.Alpha(0.64)
			}
		} else if s.Has(StateHover) {
			bg = t.Accent.Over(t.Background)
		}
		if pressed {
			bg, fg = t.Input.Alpha(0.64), t.AccentForeground
			if dark {
				bg = t.Input
			}
		}
		if !pressed && !s.Has(StateDisabled) {
			sh.scale(design.ShadowXs, black, 0.05)
			if !o.joined || !dark || o.first {
				d.highlight(t, 0.04, 0.06)
			}
		}
	}
	e.Background(bg)
	if bc.A > 0 {
		if o.joined {
			// Inner borders collapse: the first item keeps its leading
			// border, the last its trailing one, the rest neither
			// (border-s-0 / border-e-0).
			w := [4]float32{design.BorderWidth, design.BorderWidth, design.BorderWidth, design.BorderWidth}
			lead, trail := 3, 1
			if o.vertical {
				lead, trail = 0, 2
			}
			if !o.first {
				w[lead] = 0
			}
			if !o.last {
				w[trail] = 0
			}
			e.BorderWidth(w[0], w[1], w[2], w[3]).BorderColor(bc)
		} else {
			e.Border(design.BorderWidth, bc)
		}
	} else {
		e.Border(design.BorderWidth, ui.Transparent)
	}
	d.border, d.radius = design.BorderWidth, design.RadiusLG
	finish(e, &sh, &d)
	disabledLook(e, s)
	e.Children(func() {
		if o.Icon != IconNone {
			icon(c, o.Icon, design.IconSM, fg, false).MarginX(-2)
		}
		if label != "" {
			text(c, label, design.TextSM, design.WeightMedium, fg)
		}
	})
	return e
}

// ToggleGroupItem is one toggle of a ToggleGroup.
type ToggleGroupItem struct {
	On    *bool
	Label string
	Icon  Icon
	// Name labels icon-only items for assistive technology.
	Name     string
	Disabled bool
}

// ToggleGroupOptions configure a ToggleGroup.
type ToggleGroupOptions struct {
	Variant     ToggleVariant
	Size        ToggleSize
	Orientation Orientation
	// Separators puts a 1 DIP input-colored line between outline items.
	Separators bool
	Label      string
	Key        any
}

// ToggleGroup is coss's ToggleGroup: toggles 2 DIPs apart, or joined into
// one outlined strip, one Tab stop with arrow keys between them.
func ToggleGroup(c *ui.Context, items []ToggleGroupItem, o ToggleGroupOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	g := ui.Row(c).Shrink(0).Role(ui.RoleToolbar).Label(o.Label)
	orient := ui.Horizontal
	if o.Orientation == Vertical {
		g.Column()
		orient = ui.Vertical
	}
	g.FocusGroup(orient)
	if o.Variant == ToggleDefault {
		g.Gap(design.Sp(0.5))
	}
	g.Children(func() {
		for i, it := range items {
			if i > 0 && o.Separators && o.Variant == ToggleOutline {
				sep := ui.Box(c).Shrink(0).Background(t.Input)
				if o.Orientation == Vertical {
					sep.Height(1).AlignSelf(ui.Stretch)
				} else {
					sep.Width(1).AlignSelf(ui.Stretch)
				}
			}
			Toggle(c, it.On, it.Label, ToggleOptions{Variant: o.Variant, Size: o.Size, Icon: it.Icon, Label: it.Name, Disabled: it.Disabled, Key: i,
				joined: o.Variant == ToggleOutline, first: i == 0, last: i == len(items)-1, vertical: o.Orientation == Vertical})
		}
	})
	return g
}

// SegmentedSize is the segmented control's size.
type SegmentedSize uint8

const (
	SegmentedMD SegmentedSize = iota // items 30 high
	SegmentedSM                      // 26
	SegmentedLG                      // 34
)

// SegmentedOptions configure a SegmentedControl.
type SegmentedOptions struct {
	Size     SegmentedSize
	Icons    []Icon
	Disabled bool
	Label    string
	Key      any
}

// SegmentedControl is coss's SegmentedControl choosing *selected among
// labels: a muted strip whose chosen item sits on a raised pill that
// slides over 200 ms.
func SegmentedControl(c *ui.Context, selected *int, labels []string, o SegmentedOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	h, padX := float32(30), design.Sp(2.5)-1
	switch o.Size {
	case SegmentedSM:
		h, padX = 26, design.Sp(2)-1
	case SegmentedLG:
		h, padX = 34, design.Sp(3)-1
	}
	parts := ui.SegmentedBase(c, selected, len(labels))
	track := parts.Track.Padding(design.Sp(0.5)).Gap(design.Sp(0.5)).Radius(design.RadiusLG).Background(t.Muted).Shrink(0)
	if o.Label != "" {
		track.Label(o.Label)
	}
	pill := t.Background
	if t.IsDark() {
		pill = t.Input
	}
	track.Children(func() {
		for i, l := range labels {
			seg := parts.Segment(i).Height(h).PaddingX(padX).Gap(design.Sp(1.5)).Radius(design.RadiusMD).
				Border(design.BorderWidth, ui.Transparent).FocusRing(false).Cursor(ui.CursorPointer)
			s := interact(seg, isDisabled(o.Disabled))
			on := i == *selected
			fg := t.MutedForeground.Alpha(design.OpacityPlaceholder)
			if s.Has(StateHover) {
				fg = t.MutedForeground
			}
			var sh shadowList
			if on {
				fg = t.Foreground
				seg.Background(pill)
				sh.scale(design.ShadowSm, black, 0.05)
				seg.Transition(ui.ElementTransition{Position: true, Size: true, Duration: design.DurationPanel, Ease: design.EaseInOut})
			}
			if s.Has(StateFocusVisible) {
				// outline-2 outline-ring (no offset).
				sh.add(0, 0, 0, 2, t.Ring)
			}
			finish(seg, &sh, nil)
			disabledLook(seg, s)
			seg.Children(func() {
				if i < len(o.Icons) && o.Icons[i] != IconNone {
					icon(c, o.Icons[i], design.IconSM, fg, false).MarginX(-2)
				}
				text(c, l, design.TextSM, design.WeightMedium, fg)
			})
		}
	})
	return track
}

// SliderOptions configure a Slider.
type SliderOptions struct {
	Step     float64
	Disabled bool
	Label    string
	Key      any
}

// Slider is coss's Slider over [lo, hi]: a 4 DIP input-colored track, the
// primary range from lo, and a 16 DIP white thumb kept inside the track
// (thumbAlignment edge). The element is 16 high: coss's control box is the
// 4 DIP track with the thumb overflowing it.
func Slider(c *ui.Context, value *float64, lo, hi float64, o SliderOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	const thumb = 16
	e := ui.SliderBase(c, value, lo, hi).Height(thumb).FillWidth().MinWidth(176).PaddingX(thumb / 2).FocusRing(false)
	if o.Step > 0 {
		e.Step(o.Step)
	}
	if o.Label != "" {
		e.Label(o.Label)
	}
	s := interact(e, isDisabled(o.Disabled))
	dragging := e.Dragging() || s.Has(StatePressed)
	disabledLook(e, s)
	frac := float32(0)
	if hi > lo {
		frac = float32((*value - lo) / (hi - lo))
	}
	frac = max(0, min(frac, 1))
	dark := t.IsDark()
	focus := s.Has(StateFocusVisible)
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		cy := r.Y + r.H/2
		// Track: before:inset-x-0.5, 4 high, rounded.
		p.Fill(ui.Rect{X: r.X + 2, Y: cy - 2, W: r.W - 4, H: 4}, t.Input, 2)
		x := r.X + thumb/2 + (r.W-thumb)*frac
		if x > r.X+2 {
			p.Fill(ui.Rect{X: r.X + 2, Y: cy - 2, W: x - r.X - 2, H: 4}, t.Primary, 2)
		}
		size := float32(thumb)
		if dragging {
			size *= 1.2
		}
		k := ui.Rect{X: x - size/2, Y: cy - size/2, W: size, H: size}
		bc := t.Input
		if dark {
			bc = t.Background
		}
		if focus {
			a := design.InputRingAlpha
			if dark {
				a = 0.48
			}
			p.Shadow(k, size/2, 0, 0, 0, design.InputRingWidth, t.Ring.Alpha(a))
		} else if !dragging {
			p.Shadow(k, size/2, 0, 1, 2, 0, black.Alpha(0.05))
		}
		p.Fill(k, white, size/2)
		p.Stroke(k, bc, size/2, 1)
		inner := ui.Rect{X: k.X + 1, Y: k.Y + 1, W: k.W - 2, H: k.H - 2}
		p.Shadow(inner, inner.W/2, 0, 1, 0, 0, black.Alpha(0.04))
	})
	return e
}

// NumberFieldOptions configure a NumberField.
type NumberFieldOptions struct {
	Size     InputSize
	Step     float64
	Min, Max float64
	Disabled bool
	Invalid  bool
	Label    string
	Key      any
}

type numberState struct {
	text string
	val  float64
	ok   bool
}

// NumberField is coss's NumberField: − and + buttons around a centered
// tabular input, in the text control frame. Up and Down step *value.
func NumberField(c *ui.Context, value *float64, o NumberFieldOptions) ui.Element {
	withKey(c, o.Key)
	io := InputOptions{Label: o.Label, Invalid: o.Invalid, Disabled: o.Disabled}
	io.fromField()
	t := tok(c)
	step := o.Step
	if step <= 0 {
		step = 1
	}
	lo, hi := o.Min, o.Max
	if lo == 0 && hi == 0 {
		lo, hi = math.Inf(-1), math.Inf(1)
	}
	h, padX := o.Size.inner()
	group := ui.Row(c).FillWidth().AlignItems(ui.Stretch).Justify(ui.SpaceBetween).Role(ui.RoleStepper)
	st := ui.Local(group, "coss.number", func() numberState { return numberState{} })
	format := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	if !st.ok || st.val != *value {
		st.text, st.val, st.ok = format(*value), *value, true
	}
	set := func(v float64) {
		v = max(lo, min(v, hi))
		*value, st.val, st.text = v, v, format(v)
	}
	disabled := isDisabled(io.Disabled)
	var in ui.Element
	stepper := func(dec bool) {
		b := ui.ButtonBase(c).PaddingX(padX).FocusRing(false).Cursor(ui.CursorPointer)
		if dec {
			b.Radius(design.RadiusLG-1, 0, 0, design.RadiusLG-1).Label("Decrease")
		} else {
			b.Radius(0, design.RadiusLG-1, design.RadiusLG-1, 0).Label("Increase")
		}
		bs := interact(b, disabled)
		if bs.Has(StateHover) {
			b.Background(t.Accent)
		}
		b.Children(func() {
			ic := IconPlus
			if dec {
				ic = IconMinus
			}
			icon(c, ic, design.IconSM, t.Foreground, true)
		})
		if b.Clicked() && !disabled {
			if dec {
				set(*value - step)
			} else {
				set(*value + step)
			}
		}
	}
	group.Children(func() {
		stepper(true)
		in = ui.TextInputBase(c, &st.text).Grow(1).MinWidth(0).Height(h).PaddingX(padX).TextAlign(ui.Center).
			FontSize(design.TextSM.Size).FixedLineHeight(h).TextColor(t.Foreground).FontFeatures("tnum").FocusRing(false)
		if io.Label != "" {
			in.Label(io.Label)
		}
		if in.Changed() {
			if v, err := strconv.ParseFloat(st.text, 64); err == nil && v >= lo && v <= hi {
				*value, st.val = v, v
			}
		}
		if in.Shortcut(0, ui.KeyUp) {
			set(*value + step)
		}
		if in.Shortcut(0, ui.KeyDown) {
			set(*value - step)
		}
		stepper(false)
	})
	s := interactField(in, disabled)
	if io.Invalid {
		s |= StateInvalid
	}
	controlLook(group, t, s, design.RadiusLG)
	disabledLook(group, s)
	return group
}

// ProgressOptions configure a Progress or Meter.
type ProgressOptions struct {
	Label string
	// Value shows the formatted value beside the label (tabular).
	Value string
	Key   any
}

// Progress is coss's Progress: a 6 DIP rounded input-colored track with a
// primary indicator at fraction (0–1) that eases over 500 ms.
func Progress(c *ui.Context, fraction float64, o ProgressOptions) ui.Element {
	withKey(c, o.Key)
	return bar(c, fraction, o, 6, design.RadiusFull, ui.RoleProgress)
}

// Meter is coss's Meter: an 8 DIP square-ended track.
func Meter(c *ui.Context, fraction float64, o ProgressOptions) ui.Element {
	withKey(c, o.Key)
	return bar(c, fraction, o, 8, 0, ui.RoleMeter)
}

func bar(c *ui.Context, fraction float64, o ProgressOptions, h, radius float32, role ui.Role) ui.Element {
	t := tok(c)
	root := ui.Column(c).FillWidth().Gap(design.Sp(2)).Role(role)
	if o.Label != "" {
		root.Label(o.Label)
	}
	f := float32(max(0, min(fraction, 1)))
	root.Children(func() {
		if o.Label != "" || o.Value != "" {
			ui.Row(c).FillWidth().Justify(ui.SpaceBetween).Children(func() {
				text(c, o.Label, design.TextSM, design.WeightMedium, t.Foreground)
				text(c, o.Value, design.TextSM, design.WeightNormal, t.Foreground).FontFeatures("tnum")
			})
		}
		track := ui.Box(c).FillWidth().Height(h).Radius(radius).Background(t.Input).Clip()
		if forced == 0 {
			f = track.AnimateWith("coss.bar", f, design.DurationProgress, design.EaseInOut)
		}
		track.Children(func() {
			ui.Box(c).FillHeight().WidthPercent(100 * f).Background(t.Primary)
		})
	})
	return root
}

// disabledRow disables a labelled check control: coss dims the box, not
// the label beside it.
func disabledRow(row ui.Element, s State) {
	if s.Has(StateDisabled) {
		row.Disabled(true).DisabledOpacity(1)
	}
}
