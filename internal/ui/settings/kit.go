package settings

import (
	"hash/fnv"
	"math"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
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

// kSecondary is coss's secondary small button.
func kSecondary(c *ui.Context, label string, off bool) ui.Element {
	return coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonSecondary, Size: coss.ButtonSM, Disabled: off})
}

// kChoiceButton is a selectable pill/tab button: secondary when selected,
// outline otherwise, with aria-selected preserved for assistive technology.
func kChoiceButton(c *ui.Context, label string, selected, off bool) ui.Element {
	variant := coss.ButtonOutline
	if selected {
		variant = coss.ButtonSecondary
	}
	return coss.Button(c, label, coss.ButtonOptions{Variant: variant, Size: coss.ButtonSM, Pressed: selected, Disabled: off}).Selected(selected)
}

// kCheckbox is coss's Checkbox.
func kCheckbox(c *ui.Context, checked *bool, label string, off bool) ui.Element {
	return coss.Checkbox(c, checked, label, coss.CheckOptions{Disabled: off})
}

// kSwitch is coss's Switch with an empty inline label and accessible name so
// every settings toggle shares the exact 30×18 geometry.
func kSwitch(c *ui.Context, checked *bool, label string, off bool) ui.Element {
	return coss.Switch(c, checked, "", coss.CheckOptions{Label: label, Disabled: off})
}

// kTextarea is coss's Textarea.
func kTextarea(c *ui.Context, value *string, o coss.TextareaOptions) ui.Element {
	return coss.Textarea(c, value, o)
}

// kInputRow wraps coss.Input in a flexible box so it expands inside a ui.Row.
func kInputRow(c *ui.Context, value *string, o coss.InputOptions) ui.Element {
	var in ui.Element
	ui.Box(c).Grow(1).MinWidth(0).Children(func() {
		in = coss.Input(c, value, o)
	})
	return in
}

// settingsTitleBarTopInset returns the minimum top clearance needed to keep
// interactive controls clear of hidden-titlebar window buttons (- □ × / traffic
// lights). Returns 0 when the window has no overlaid caption controls.
func settingsTitleBarTopInset(bar ui.TitleBar) float32 {
	if bar.Height <= 0 && bar.Left <= 0 && bar.Right <= 0 {
		return 0
	}
	safe := bar.Height
	if bar.Right > 0 {
		// Windows caption buttons sit at Top=12, Height=32 (bottom=44) even
		// when TitleBarHeight is the 12 DIP drag strip.
		winBottom := chrome.TitleBarHeight + design.Sp(8)
		if winBottom > safe {
			safe = winBottom
		}
	}
	return safe + design.Sp(2)
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

// settingsGlassActive reports whether the settings surface should render with
// the translucent frosted glass treatment.
func settingsGlassActive(t *design.Tokens) bool {
	return t.Glass || chrome.BackdropActive()
}

// settingsRailGlassFill returns the left navigation rail's translucent glass
// tint (macOS System Settings style: more translucent than the right options
// pane). Light: white @ 0.36; Dark: rgb(24, 25, 30) @ 0.46.
func settingsRailGlassFill(dark bool) ui.Color {
	if dark {
		return design.TW.Zinc900.Mix(design.TW.Slate800, 0.08).Alpha(0.46)
	}
	return design.TW.White.Alpha(0.36)
}

// settingsPaneGlassFill returns the right options pane's translucent glass
// tint (slightly more solid than the rail so long forms remain comfortable to
// read). Light: white @ 0.54; Dark: rgb(30, 32, 38) @ 0.60.
func settingsPaneGlassFill(dark bool) ui.Color {
	if dark {
		return design.TW.Zinc800.Mix(design.TW.Zinc900, 0.60).Mix(design.TW.Slate800, 0.18).Alpha(0.60)
	}
	return design.TW.White.Alpha(0.54)
}

// settingsDialogGlassFill returns the translucent glass fill for secondary
// confirmation and editor dialogs inside settings.
// Light: white @ 0.48; Dark: rgb(30, 32, 38) @ 0.54.
func settingsDialogGlassFill(dark bool) ui.Color {
	if dark {
		return design.TW.Zinc800.Mix(design.TW.Zinc900, 0.60).Mix(design.TW.Slate800, 0.18).Alpha(0.54)
	}
	return design.TW.White.Alpha(0.48)
}

// settingsGlassDivider returns the soft semi-transparent divider between
// layered glass regions (rail / options pane / header).
func settingsGlassDivider(t *design.Tokens) ui.Color {
	if settingsGlassActive(t) {
		if t.IsDark() {
			return design.TW.White.Alpha(0.08)
		}
		return design.TW.Black.Alpha(0.08)
	}
	return t.SidebarBorder
}

// settingsGlassInnerCard returns the light glass surface fill and fine hairline
// border for cards nested inside the settings modal so they preserve backdrop
// translucency instead of stacking opaque t.Card fills.
func settingsGlassInnerCard(dark bool) (fill, border ui.Color) {
	if dark {
		return design.TW.White.Alpha(0.045), design.TW.White.Alpha(0.07)
	}
	return design.TW.White.Alpha(0.26), design.TW.Black.Alpha(0.06)
}

// settingsGlassListWash returns the light translucent hover, pressed, and
// selected pill fills for list/rail items in glass mode.
func settingsGlassListWash(dark bool) (hover, pressed, selected ui.Color) {
	if dark {
		return design.TW.White.Alpha(0.07), design.TW.White.Alpha(0.095), design.TW.White.Alpha(0.12)
	}
	return design.TW.Black.Alpha(0.065), design.TW.Black.Alpha(0.08), design.TW.Black.Alpha(0.095)
}

// settingsPaneWidthHintState keeps the known inner width of the right options
// pane for the active settings.View pass so responsive grids and rows choose
// their final column count on frame 1 without a width=0 double reflow.
func settingsPaneWidthHintState(c *ui.Context) *float32 {
	return ui.Local(c.Root(), "settings-pane-width-hint", func() float32 { return 0 })
}

func settingsModalWidthHintState(c *ui.Context) *float32 {
	return ui.Local(c.Root(), "settings-modal-width-hint", func() float32 { return 0 })
}

func settingsContentWidth(c *ui.Context, e ui.Element, maxW float32) float32 {
	if w := e.Bounds().W; w > 0 {
		return w
	}
	if w := *settingsPaneWidthHintState(c); w > 0 {
		if maxW > 0 && w > maxW {
			return maxW
		}
		return w
	}
	vw, _ := c.Size()
	if maxW > 0 && vw > maxW {
		return maxW
	}
	return vw
}

// kSurface paints e as a coss card surface: card background, border, the
// xl radius. In glass mode, it uses a light translucent inner glass card so
// cards inside settings stay airy over the blurred backdrop.
func kSurface(c *ui.Context, e ui.Element) ui.Element {
	t := tk(c)
	if settingsGlassActive(t) {
		fill, stroke := settingsGlassInnerCard(t.IsDark())
		return e.Background(fill).Border(design.BorderWidth/2, stroke).Radius(design.RadiusXL).TextColor(t.CardForeground)
	}
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
