package coss

import (
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// BadgeVariant is coss's badgeVariants.variant.
type BadgeVariant uint8

const (
	BadgeDefault BadgeVariant = iota
	BadgeSecondary
	BadgeOutline
	BadgeDestructive
	BadgeError
	BadgeInfo
	BadgeSuccess
	BadgeWarning
)

// BadgeSize is coss's badgeVariants.size.
type BadgeSize uint8

const (
	BadgeMD BadgeSize = iota // 18 high
	BadgeSM                  // 16
	BadgeLG                  // 22
)

// BadgeOptions configure a Badge.
type BadgeOptions struct {
	Variant BadgeVariant
	Size    BadgeSize
	Icon    Icon
	// Interactive renders the badge as a button (coss [button&]): hover
	// fills and Clicked.
	Interactive bool
	Disabled    bool
	Key         any
}

// Badge is coss's Badge.
func Badge(c *ui.Context, label string, o BadgeOptions) ui.Element {
	withKey(c, o.Key)
	t := tok(c)
	h, padX, radius, iconSize := float32(18), design.Sp(1)-1, design.RadiusSM, float32(12)
	ts := design.TextXS
	switch o.Size {
	case BadgeSM:
		h, radius = 16, 4
		ts = design.TextStyle{Size: 10, LineHeight: 13.33}
	case BadgeLG:
		h, padX = 22, design.Sp(1.5)-1
		ts = design.TextSM
	}
	var e ui.Element
	if o.Interactive {
		e = ui.ButtonBase(c).FocusRing(false).Cursor(ui.CursorPointer)
	} else {
		e = ui.Row(c).Center().Shrink(0)
	}
	e.Height(h).MinWidth(h).PaddingX(padX).Radius(radius).Gap(design.Sp(1))
	s := State(0)
	if o.Interactive {
		s = interact(e, isDisabled(o.Disabled))
	} else if isDisabled(o.Disabled) {
		s = StateDisabled
	}
	hover := o.Interactive && (s.Has(StateHover) || s.Has(StatePressed))
	dark := t.IsDark()
	tint := func(c ui.Color) ui.Color {
		if dark {
			return c.Alpha(0.16)
		}
		return c.Alpha(0.08)
	}
	bg, fg, bc := t.Primary, t.PrimaryForeground, ui.Transparent
	switch o.Variant {
	case BadgeDefault:
		if hover {
			bg = t.Primary.Alpha(0.9)
		}
	case BadgeSecondary:
		bg, fg = t.Secondary, t.SecondaryForeground
		if hover {
			bg = t.Secondary.Alpha(0.9)
		}
	case BadgeOutline:
		bg, fg, bc = t.Background, t.Foreground, t.Input
		if dark {
			bg = t.Input.Alpha(0.32)
		}
		if hover {
			bg = t.Accent.Alpha(0.5)
			if dark {
				bg = t.Input.Alpha(0.48)
			}
		}
	case BadgeDestructive:
		bg, fg = t.Destructive, white
		if hover {
			bg = t.Destructive.Alpha(0.9)
		}
	case BadgeError:
		bg, fg = tint(t.Destructive), t.DestructiveForeground
	case BadgeInfo:
		bg, fg = tint(t.Info), t.InfoForeground
	case BadgeSuccess:
		bg, fg = tint(t.Success), t.SuccessForeground
	case BadgeWarning:
		bg, fg = tint(t.Warning), t.WarningForeground
	}
	e.Background(bg).Border(design.BorderWidth, bc)
	var sh shadowList
	var d deco
	d.border, d.radius = design.BorderWidth, radius
	if o.Interactive {
		d.setRing(c, e, t, s.Has(StateFocusVisible), t.Ring)
	}
	finish(e, &sh, &d)
	disabledLook(e, s)
	e.Children(func() {
		if o.Icon != IconNone {
			icon(c, o.Icon, iconSize, fg, false)
		}
		if label != "" {
			text(c, label, ts, design.WeightMedium, fg)
		}
	})
	return e
}

// Kbd is coss's Kbd: a 20 DIP muted key cap.
func Kbd(c *ui.Context, key string) ui.Element {
	t := tok(c)
	return ui.Row(c).Center().Shrink(0).Height(20).MinWidth(20).PaddingX(design.Sp(1)).Radius(4).Gap(design.Sp(1)).
		Background(t.Muted).Children(func() {
		text(c, key, design.TextXS, design.WeightMedium, t.MutedForeground)
	})
}

// KbdGroup lays key caps out 4 DIPs apart.
func KbdGroup(c *ui.Context, keys ...string) ui.Element {
	return ui.Row(c).Gap(design.Sp(1)).AlignItems(ui.Center).Shrink(0).Children(func() {
		for _, k := range keys {
			Kbd(c, k)
		}
	})
}

// Orientation of separators, groups, toolbars and sliders.
type Orientation uint8

const (
	Horizontal Orientation = iota
	Vertical
)

// Separator is coss's Separator: a 1 DIP border-colored line.
func Separator(c *ui.Context, o Orientation) ui.Element {
	t := tok(c)
	e := ui.Box(c).Shrink(0).Background(t.Border)
	if o == Vertical {
		return e.Width(1).AlignSelf(ui.Stretch)
	}
	return e.Height(1).FillWidth()
}

// Skeleton is coss's Skeleton: a muted placeholder with a light sweep
// (animate-skeleton, 2.6 s linear, the gradient fixed to the window). Size it
// with Width/Height.
func Skeleton(c *ui.Context) ui.Element {
	t := tok(c)
	hl := white.Alpha(0.64)
	if t.IsDark() {
		hl = white.Alpha(0.04)
	}
	e := ui.Box(c).Radius(design.RadiusSM).Background(t.Muted)
	phase := float32(0.5)
	if forced == 0 && !c.Preferences().ReduceMotion {
		// -0.5 cycle delay: the sweep starts halfway.
		phase = e.Loop("coss.skeleton", design.DurationSkeleton, design.EaseLinear) + 0.5
	}
	ww, _ := c.Size()
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		// background: linear-gradient(120deg, transparent 40%, hl,
		// transparent 60%) 0 0 / 200% 100% fixed; position animates to
		// -200%. The band's centre sits mid-gradient; with the tile 2W wide
		// its centre is at W − 2W·phase, repeating every 2W.
		tile := 2 * ww
		center := ww - tile*phase
		for center < r.X-tile/2 {
			center += tile
		}
		half := 0.1 * tile * 0.866
		for _, cx := range [2]float32{center, center - tile} {
			if cx+half < r.X || cx-half > r.X+r.W {
				continue
			}
			p.Clip(r, design.RadiusSM, func() {
				p.FillGradient(ui.Rect{X: cx - half, Y: r.Y, W: half, H: r.H}, ui.LinearGradient{From: ui.Transparent, To: hl, Angle: 90}, 0)
				p.FillGradient(ui.Rect{X: cx, Y: r.Y, W: half, H: r.H}, ui.LinearGradient{From: hl, To: ui.Transparent, Angle: 90}, 0)
			})
		}
	})
	return e
}
