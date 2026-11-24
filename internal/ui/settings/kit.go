package settings

import (
	"hash/fnv"
	"math"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

// The settings kit is the coss vocabulary every settings page is built from:
// type pairs scaled by the interface-size preference, the page and row text
// styles, and thin adapters over internal/ui/coss controls. Colors, radii and
// spacing come from internal/ui/design; nothing here holds a literal of its
// own beyond coss's documented metrics.

// tk returns the window's coss tokens (scheme, Host accent as primary, the
// optional glass filter).
func tk(c *ui.Context) *design.Tokens { return design.For(c) }

// fs and lh are a coss type pair's size and line height, scaled by the
// interface-size preference the Interface page edits.
func fs(s design.TextStyle) float32 { return theme.InterfaceSize(s.Size) }
func lh(s design.TextStyle) float32 { return theme.InterfaceSize(s.LineHeight) }

// kText is wrapping text in a coss type pair.
func kText(c *ui.Context, s string, style design.TextStyle, weight int, col ui.Color) ui.Element {
	return ui.Text(c, s).FontSize(fs(style)).FixedLineHeight(lh(style)).FontWeight(weight).TextColor(col)
}

// kPageTitle is a page's heading: text-lg semibold foreground.
func kPageTitle(c *ui.Context, s string) ui.Element {
	return kText(c, s, design.TextLG, design.WeightSemibold, tk(c).Foreground).FillWidth().Role(ui.RoleHeading)
}

// kHeading is a group heading inside a page: text-sm semibold foreground.
func kHeading(c *ui.Context, s string) ui.Element {
	return kText(c, s, design.TextSM, design.WeightSemibold, tk(c).Foreground).FillWidth().Role(ui.RoleHeading)
}

// kLabel is a row's title: coss Label, text-sm medium foreground.
func kLabel(c *ui.Context, s string, col ui.Color) ui.Element {
	return kText(c, s, design.TextSM, design.WeightMedium, col)
}

// kBody is ordinary copy: text-sm foreground.
func kBody(c *ui.Context, s string) ui.Element {
	return kText(c, s, design.TextSM, design.WeightNormal, tk(c).Foreground).FillWidth()
}

// kHint is a row or field description: coss FieldDescription, text-xs muted.
func kHint(c *ui.Context, s string) ui.Element {
	return kText(c, s, design.TextXS, design.WeightNormal, tk(c).MutedForeground).FillWidth()
}

// kError is an inline failure sentence (from internal/errcopy): text-xs
// destructive-foreground, announced as a status.
func kError(c *ui.Context, s string) ui.Element {
	return kText(c, s, design.TextXS, design.WeightNormal, tk(c).DestructiveForeground).FillWidth().Role(ui.RoleStatus)
}

// kButton is coss's outline small button, the settings pages' default
// secondary action.
func kButton(c *ui.Context, label string) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM})
}

// kButtonDisabled is kButton, disabled when off is set.
func kButtonDisabled(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: off})
}

// kPrimary is coss's default (primary) small button.
func kPrimary(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Size: coss.ButtonSM, Disabled: off})
}

// kGhost is coss's ghost small button.
func kGhost(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Disabled: off})
}

// kDanger is coss's destructive-outline small button.
func kDanger(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonDestructiveOutline, Size: coss.ButtonSM, Disabled: off})
}

// kIconButton is a ghost icon button drawing one of the settings icons,
// named label for assistive technology and its tooltip.
func kIconButton(c *ui.Context, icon *ui.SVG, label string, off bool) ui.Element {
	size := coss.ButtonIconSM
	m := size.Metrics()
	b := coss.Button(c, "", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: size, Label: label, Disabled: off, Content: func() {
		if icon != nil {
			ui.Icon(c, icon).Size(m.IconSize, m.IconSize).Shrink(0).TextColor(tk(c).Foreground).Opacity(design.OpacityIcon).PassThrough()
		}
	}})
	if label != "" {
		b.Tooltip(label)
	}
	return b
}

// kOptions turns plain choice labels into coss select options.
func kOptions(labels []string) []coss.SelectOption {
	out := make([]coss.SelectOption, len(labels))
	for i, l := range labels {
		out[i] = coss.SelectOption{Value: l}
	}
	return out
}

// kSelect is coss's Select over plain labels, the value being the label.
func kSelect(c *ui.Context, value *string, labels []string, label string, off bool) ui.Element {
	return coss.Select(c, value, kOptions(labels), coss.SelectOptions{Label: label, Disabled: off})
}

// kInput is coss's Input.
func kInput(c *ui.Context, value *string, o coss.InputOptions) ui.Element {
	return coss.Input(c, value, o)
}

// kRule is a 1 DIP border-colored separator.
func kRule(c *ui.Context) ui.Element { return coss.Separator(c, coss.Horizontal) }

// kSurface paints e as a coss card surface: card background, border, the
// xl radius. The glass appearance, when on, has already made the card
// token translucent.
func kSurface(c *ui.Context, e ui.Element) ui.Element {
	t := tk(c)
	return e.Background(t.Card).Border(design.BorderWidth, t.Border).Radius(design.RadiusXL).TextColor(t.CardForeground)
}

// memoKey folds the parts a memo region shows into one version. Use it for
// regions whose content is a handful of values outside any store revision:
// hashing them is cheaper than building the region.
type memoKey struct{ h uint64 }

func newMemoKey(seed uint64) *memoKey {
	k := &memoKey{h: 1469598103934665603}
	return k.u(seed)
}

func (k *memoKey) u(v uint64) *memoKey {
	for i := 0; i < 8; i++ {
		k.h ^= v & 0xff
		k.h *= 1099511628211
		v >>= 8
	}
	return k
}

func (k *memoKey) s(v ...string) *memoKey {
	for _, s := range v {
		h := fnv.New64a()
		_, _ = h.Write([]byte(s))
		k.u(h.Sum64())
	}
	return k
}

func (k *memoKey) b(v ...bool) *memoKey {
	var bits uint64
	for i, on := range v {
		if on {
			bits |= 1 << uint(i%64)
		}
	}
	return k.u(bits).u(uint64(len(v)))
}

func (k *memoKey) f(v float64) *memoKey { return k.u(math.Float64bits(v)) }

func (k *memoKey) sum() uint64 { return k.h }
