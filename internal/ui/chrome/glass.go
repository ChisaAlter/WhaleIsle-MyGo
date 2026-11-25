package chrome

import (
	"image"
	"math"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

// Material names one layer of the shell's glass. Each layer keeps the
// solidity the harness wallpaper tokens give it (ui-theme/src/wallpaper.ts,
// mixWallpaperSurfaces): the canvas is the most see-through and capped, the
// sidebar sits between the canvas curve and the glass slider, raised surfaces
// keep the slider, and menus keep their own fixed fill because a menu has to
// stay readable over whatever it covers.
type Material uint8

const (
	// MaterialCanvas is the main chat/work surface that the wallpaper shows
	// through.
	MaterialCanvas Material = iota
	// MaterialSidebar is the left rail and the titlebar strip above it.
	MaterialSidebar
	// MaterialPanel is a raised surface: composer card, dialog, side pane.
	MaterialPanel
	// MaterialMenu is a menu, popover or floating card.
	MaterialMenu
	// MaterialTerminal is the terminal well, which follows its own slider.
	MaterialTerminal
)

const (
	// maxCanvasSolidity caps the canvas fill so glass 100% still lets the
	// wallpaper show through the main chat (MAX_WALLPAPER_CANVAS_SOLIDITY).
	maxCanvasSolidity = 45.0
	// menuSolidityLight and menuSolidityDark are --dsw-menu-surface-fill: a
	// menu keeps its own fill instead of following the glass slider.
	menuSolidityLight = 0.58
	menuSolidityDark  = 0.45
	// maskLight and maskDark are --dsw-alias-bg-mask-1, the dim the wallpaper
	// layer paints over the image so text keeps contrast.
	maskLight = 0.24
	maskDark  = 0.5
)

// MaskAlpha is the dim the wallpaper layer paints over the image, so text over
// the bare wallpaper keeps its contrast: 0.24 in light and 0.5 in dark, as
// --dsw-alias-bg-mask-1 says. The transparent theme drops it.
func MaskAlpha(dark bool) float32 {
	if dark {
		return maskDark
	}
	return maskLight
}

// canvasSolidity is the main canvas fill as a fraction, capped so the wallpaper
// always shows through the main chat.
func canvasSolidity(glass float64) float32 {
	return min(float32(maxCanvasSolidity/100), canvasSolidityUncapped(clampPercent(glass))/100)
}

// sidebarSolidity sits halfway between the uncapped canvas curve and the glass
// slider, so glass 100% still fully opaques the rail.
func sidebarSolidity(glass float64) float32 {
	k := clampPercent(glass)
	return (float32(k) + canvasSolidityUncapped(k)) / 200
}

// canvasSolidityUncapped is canvasSolidity before the cap, as a percent: the
// sidebar mix needs the same curve without the ceiling.
func canvasSolidityUncapped(k float64) float32 {
	switch {
	case k <= 40:
		return float32(k * .375)
	case k <= 80:
		return float32(15 + (k-40)*.75)
	default:
		return float32(45 + (k-80)*2.75)
	}
}

func clampPercent(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Max(0, math.Min(100, v))
}

// MaterialAlpha is the fill opacity of material m over the backdrop. A
// transparent theme zeroes every chrome surface; the terminal keeps its own
// slider there, as the harness does.
func MaterialAlpha(m Material, p wallpapers.Preferences, dark bool) float32 {
	switch m {
	case MaterialTerminal:
		return float32(clampPercent(p.Terminal) / 100)
	case MaterialMenu:
		if dark {
			return menuSolidityDark
		}
		return menuSolidityLight
	}
	if p.Transparent {
		return 0
	}
	switch m {
	case MaterialCanvas:
		return canvasSolidity(p.Glass)
	case MaterialSidebar:
		// A hidden sidebar mask paints the canvas fill instead of an extra
		// surface; with no mask the rail is the canvas itself.
		if p.HideSidebarMask {
			return 0
		}
		return sidebarSolidity(p.Glass)
	default:
		return float32(clampPercent(p.Glass) / 100)
	}
}

// materialFrost is how strongly a layer frosts the wallpaper under it, in DIPs
// per sampled cell: 0 keeps the wallpaper as sharp as it is painted, a larger
// cell averages it more. The harness only blurs menus (backdrop-filter:
// blur(40px) saturate(150%)); the terminal well frosts the same way by user
// request, while sidebar, panel and canvas do not blur.
type materialFrost struct {
	cell     float32
	saturate float32
}

func frostOf(m Material) materialFrost {
	switch m {
	case MaterialMenu, MaterialTerminal:
		return materialFrost{cell: 18, saturate: 1.5}
	}
	return materialFrost{cell: 0, saturate: 1}
}

// BackdropActive reports whether the shell paints a wallpaper or an ambient
// gradient behind its content, so a surface should turn translucent.
func BackdropActive() bool { return backgroundState.active }

// SampleBackdrop returns the average color of the wallpapered backdrop under
// rect, after the dim mask. It reports false when no wallpaper is painted, so
// callers keep their flat surface color.
func SampleBackdrop(rect ui.Rect) (ui.Color, bool) {
	s := &backgroundState
	if !s.active || s.colors == nil || !s.hasBox || s.box.W <= 0 || s.box.H <= 0 {
		return ui.Color{}, false
	}
	keep := float32(1)
	if gradientBackdrop.frame == nil && !s.prefs.Transparent {
		keep = 1 - MaskAlpha(s.dark)
	}
	return s.colors.average(s.box, rect, keep)
}

// SurfaceFill is the glass fill of a surface at alpha over the wallpaper under
// rect: the sampled backdrop tinted with base, returned opaque. It suits small
// surfaces such as a menu row or a chip; a large panel keeps the wallpaper's
// detail by painting Glass instead.
//
// Without a wallpaper, or before the first wallpaper frame, it returns
// base.Alpha(alpha) unchanged, which is what the fixed-alpha surfaces painted
// before sampling existed.
func SurfaceFill(rect ui.Rect, base ui.Color, alpha float32) ui.Color {
	if alpha <= 0 {
		return base.Alpha(0)
	}
	backdrop, ok := SampleBackdrop(rect)
	if !ok {
		return base.Alpha(alpha)
	}
	return base.Alpha(alpha).Over(backdrop)
}

// BackdropFill returns the flat color material m produces over the wallpaper
// under rect: the same share and contrast guard DrawGlass paints, without the
// frost. It is the color form of Glass, for elements that only take a Background
// color: a menu row, a chip, a small pane.
func BackdropFill(rect ui.Rect, base ui.Color, m Material, p wallpapers.Preferences, dark bool) ui.Color {
	if !backgroundState.active {
		return base
	}
	p = effectivePrefs(p)
	alpha := MaterialAlpha(m, p, dark)
	backdrop, ok := SampleBackdrop(rect)
	if !ok {
		return MaterialFill(base, alpha)
	}
	if !dark && m != MaterialCanvas {
		alpha = MaterialAlphaFor(backdrop, base, m, p, dark, Color(theme.SecondaryInk(dark)))
	}
	if alpha <= 0 {
		return ui.Transparent
	}
	return MaterialFill(base, alpha).Over(backdrop)
}

// Glass returns a Draw callback that paints material m over base on the
// element it is drawn on. Replace a fixed-alpha Background with it so the
// surface frosts and tints the wallpaper actually under it:
//
//	ui.Column(c).Key("sidebar").Draw(chrome.Glass(fill, chrome.MaterialSidebar))
//
// radius rounds the fill and the frost to the panel's own corners, so a card
// keeps its shape: pass the element's Radius (28 for the composer card, for
// instance). Omit it for a square surface.
//
// The callback paints after the element's own background, so leave the element
// without one; its own Radius clips what it draws. Without a wallpaper or an
// ambient gradient it paints base at the material's own share, which over the
// theme's background looks like the solid surface the Background gave it.
func Glass(base ui.Color, m Material, radius ...float32) func(*ui.Painter, ui.Rect) {
	round := firstRadius(radius)
	return func(p *ui.Painter, r ui.Rect) { DrawGlass(p, r, base, m, round) }
}

// DrawGlass paints material m on rect: the frosted patch of the wallpapered
// backdrop under it, then base at the material's opacity, both rounded by
// radius so the corners do not square off or spill over the element's clip. A
// layer that neither frosts nor tints draws nothing and lets the wallpaper
// show, which is what a transparent theme asks for.
//
// Over a wallpaper the material checks its own contrast first (MaterialAlphaFor):
// where the token's share would leave secondary text under 3:1 on the surface it
// produces, the surface frosts more solid instead.
func DrawGlass(p *ui.Painter, r ui.Rect, base ui.Color, m Material, radius ...float32) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	round := firstRadius(radius)
	// The transparent theme drops every chrome fill, so the image shows
	// through unmasked; the terminal and menus keep their own fills.
	prefs := effectivePrefs(backgroundState.prefs)
	if prefs.Transparent && chromeSurface(m) {
		return
	}
	if !backgroundState.active {
		// No backdrop: dark chrome composites over the desktop exactly as the
		// native glass model does; light keeps the solid token fill.
		if alpha := ShowThroughAlpha(prefs, p.Theme().Dark); alpha < 1 && chromeSurface(m) {
			p.Fill(r, MaterialFill(base, alpha), round)
			return
		}
		// No backdrop to share: the surface keeps the theme's own fill, as the
		// Background it replaces did.
		p.Fill(r, base, round)
		return
	}
	dark := p.Theme().Dark
	// A surface token may carry the glass opacity in its own alpha
	// (--dsw-specific-input-major is an 80% rgba in the sheets); the material
	// decides that share here, so the slider drives it once instead of twice.
	alpha := MaterialAlpha(m, prefs, dark)
	if gradientBackdrop.frame != nil {
		// The gradient already paints its source-composited pixels beneath the
		// shared canvas. A raw sampled patch here would replace that canvas tint.
		// Its raised fills retain the CSS token alpha, without image-only guards.
		if alpha > 0 {
			p.Fill(r, MaterialFill(base, alpha), round)
		}
		return
	}
	if !dark && m != MaterialCanvas {
		if backdrop, ok := SampleBackdrop(r); ok {
			alpha = MaterialAlphaFor(backdrop, base, m, prefs, dark, Color(theme.SecondaryInk(dark)))
		}
	}
	if alpha <= 0 {
		return
	}
	if frost := frostOf(m); frost.cell > 0 {
		// The frost is a GPU backdrop blur of what is painted under the
		// surface; no CPU patch of the wallpaper is made.
		glass.Blur{Radius: frost.cell, Saturation: frost.saturate - 1}.PaintMaterial(p, r, [4]float32{round, round, round, round})
	}
	p.Fill(r, MaterialFill(base, alpha), round)
}

// firstRadius returns the optional radius of a glass call, 0 when square.
func firstRadius(radius []float32) float32 {
	if len(radius) == 0 || radius[0] < 0 || math.IsNaN(float64(radius[0])) {
		return 0
	}
	return radius[0]
}

// chromeSurface reports whether the material is one the transparent theme
// zeroes, rather than a well with its own opacity slider.
func chromeSurface(m Material) bool {
	switch m {
	case MaterialCanvas, MaterialSidebar, MaterialPanel:
		return true
	}
	return false
}

// GlassSurface dresses e as a surface of material m: it clears the element's
// own background and paints base through the glass instead, so the wallpaper
// shows through the surface rather than behind an opaque fill. radius is the
// element's own Radius, so the glass follows the panel's corners:
//
//	shell := chrome.GlassSurface(ui.Column(c).Fill(), fill, chrome.MaterialCanvas)
func GlassSurface(e ui.Element, base ui.Color, m Material, radius ...float32) ui.Element {
	return e.Background(ui.Transparent).Draw(Glass(base, m, radius...))
}

// MenuBaseFill returns the reference menu surface base color for colors:
// when a wallpaper or ambient gradient is active, --dsw-menu-surface-fill
// uses rgb(248, 249, 250) in light mode and rgb(67, 69, 74) (#43454a) in dark
// mode; without a backdrop, it returns the flat surface token (#ffffff in
// light, colors.Field in dark).
func MenuBaseFill(colors theme.Colors, dark bool) ui.Color {
	if backgroundState.active {
		if dark {
			return ui.RGB(67, 69, 74)
		}
		return ui.RGB(248, 249, 250)
	}
	if dark {
		return Color(colors.Field)
	}
	return ui.RGB(255, 255, 255)
}

// MenuBorderColor returns the reference menu elevation hairline color:
// in dark mode over a live backdrop, body[data-ds-dark-theme] [data-menu-material]
// uses --dsw-alias-border-l3 (rgba(255, 255, 255, 0.16)); otherwise --dsw-alias-border-l1.
func MenuBorderColor(colors theme.Colors, dark bool) ui.Color {
	if dark && backgroundState.active && colors.BorderL3 != "" {
		return Color(colors.BorderL3)
	}
	return Color(colors.BorderL1)
}

// menuBlur is the reference menu backdrop-filter: blur(40px) saturate(150%).
var menuBlur = glass.Blur{Radius: 40, Saturation: 0.5}

// MenuSurface dresses e as the shared menu panel: the menu material, which
// blurs what was rendered under the menu (backdrop-filter: blur(40px)
// saturate(150%)) and keeps its own fill instead of following the glass
// slider, the hairline border and the elevation shadow. The caller keeps its
// width and padding; the radius passed here is the panel's own corner, set on
// the element as well as given to the glass, so the border, the shadow and the
// clip follow the same curve.
//
// The hairline is painted after the glass: a Draw callback runs after the
// element's border, so a bordered glass panel would lose its edge under an
// opaque fill.
func MenuSurface(e ui.Element, base, border ui.Color, radius ...float32) ui.Element {
	round := firstRadius(radius)
	e = GlassSurface(e.Radius(round), base, MaterialMenu, round).Border(0.5, border)
	if backgroundState.active {
		e = e.Material(menuPanelGlass{})
	}
	return MenuElevation(e.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if border.A == 0 || r.W <= 0 || r.H <= 0 {
			return
		}
		p.Stroke(r, border, round, 0.5)
	}))
}

// FrostSurface dresses e as a raised surface in the menu material while a
// wallpaper or ambient gradient is painted: the menu's frosted backdrop and its
// fixed fill (menuSolidityLight/Dark), no shadow, no live scene blur. With no
// backdrop it keeps base, so no-wallpaper pixels do not change.
func FrostSurface(e ui.Element, colors theme.Colors, dark bool, base ui.Color, radius float32) ui.Element {
	if backgroundState.active {
		return GlassSurface(e.Radius(radius), MenuBaseFill(colors, dark), MaterialMenu, radius)
	}
	return e.Radius(radius).Background(base)
}

// FrostInset is the fill for a surface nested inside a FrostSurface parent —
// an input, a selected tab, a chip: over a live backdrop it is the translucent
// hover wash so the parent's frost still shows through, never a second frosted
// fill. With no backdrop it keeps base so no-wallpaper pixels do not change.
func init() { widgets.SurfaceDresser = FrostSurface }

func FrostInset(colors theme.Colors, dark bool, base ui.Color) ui.Color {
	if backgroundState.active {
		return Color(colors.Hover)
	}
	return base
}

// MenuRow is one selectable row of an app popover: no frame of its own, the
// shared 34 DIP rhythm, a rounded hover/pressed fill and room for leading and
// trailing glyphs. Callers add a label and children themselves.
func MenuRow(c *ui.Context, colors theme.Colors) ui.Element {
	return MenuRowColors(c, Color(colors.Hover), Color(colors.Pressed))
}

// MenuRowColors is MenuRow for callers that only hold ui.Theme colors.
func MenuRowColors(c *ui.Context, hover, pressed ui.Color) ui.Element {
	row := ui.ButtonBase(c).FillWidth().MinHeight(34).Padding(design.Sp(1.5), design.Sp(2)).Gap(design.Sp(1.5)).Radius(theme.ContextRadii(c).MD).BorderWidth(0).
		Background(ui.Transparent).Justify(ui.Start).AlignItems(ui.Center).Cursor(ui.CursorPointer)
	row = row.Transition(ui.ElementTransition{Duration: PopoverDuration, Colors: true})
	if row.Pressed() {
		return row.Background(pressed)
	}
	if row.Hovered() {
		return row.Background(hover)
	}
	return row
}

// MenuShadow paints the elevation of a menu or floating card on rect: two soft
// shadows and the hairline the harness draws around overlays. Paint it from a
// wrapper element's Draw so the shadow is not clipped to the panel.
func MenuShadow(p *ui.Painter, rect ui.Rect, radius float32) {
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	t := p.Theme()
	soft := float32(0.08)
	if t.Dark {
		soft = 0.28
	}
	p.Shadow(rect, radius, 0, 3, 8, 0, ui.RGBA(0, 0, 0, soft*.55))
	p.Shadow(rect, radius, 0, 12, 32, 0, ui.RGBA(0, 0, 0, soft))
	p.Stroke(rect, t.Border, radius, 0.5)
}

// MenuElevation gives a floating menu or card its drop shadow, matching the
// --dsw-elevation-* recipe: a small near shadow plus one wide soft one. A menu
// panel takes the shared material, the hairline, this elevation and the popover
// fade together:
//
//	ui.PopoverBase(c, anchor, &open, func(panel ui.Element) {
//		progress := chrome.Reveal(c, panel, chrome.PopoverDuration)
//		panel.Width(220).Radius(10).Opacity(progress)
//		chrome.MenuSurface(panel, fill, border, 10)
//	})
func MenuElevation(e ui.Element) ui.Element {
	return e.Shadow(0, 0, 20, 0, ui.RGBA(0, 0, 0, .05)).
		Shadow(0, 3, 8, 0, ui.RGBA(0, 0, 0, .04))
}

// menuPanelGlass is the ui.Material a theme's Menu asks for (Theme.Menu) and
// that MenuSurface applies: a real scene backdrop blur (blur(40px)
// saturate(150%)) over whatever was rendered behind the menu (ambient
// gradient, wallpaper, composer card, borders, and text), plus the wallpaper
// frost fallback when applicable.
type menuPanelGlass struct{}

// PaintMaterial paints the menu frost and scene backdrop blur over box,
// rounded by radii.
func (menuPanelGlass) PaintMaterial(p *ui.Painter, box ui.Rect, radii [4]float32) {
	if !backgroundState.active || box.W <= 0 || box.H <= 0 {
		return
	}
	menuBlur.PaintMaterial(p, box, radii)
}

// MenuStyle returns the ui.Theme.Menu of the resolved colors: the menu's own
// fill — the raised surface at its material share over a live backdrop,
// opaque where there is none — the hairline of a menu's elevation,
// and the menu's icon, caption and separator inks. Set it on the theme so
// toolkit-drawn menus, select popups and popovers take the same look
// MenuSurface gives the chrome's own popovers:
//
//	t.Menu = chrome.MenuStyle(colors, dark)
func MenuStyle(colors theme.Colors, dark bool) ui.MenuStyle {
	base := MenuBaseFill(colors, dark)
	fill := base
	if backgroundState.active {
		fill = MaterialFill(base, MaterialAlpha(MaterialMenu, effectivePrefs(backgroundState.prefs), dark))
	}
	st := ui.MenuStyle{
		Fill:      fill,
		Stroke:    MenuBorderColor(colors, dark),
		Hover:     Color(colors.Hover),
		Icon:      Color(theme.SkinSemanticToken("", dark, "--dsw-alias-menu-icon", colors.FG)),
		Caption:   Color(colors.Caption),
		Separator: Color(colors.Border),
	}
	if fill.A < 255 {
		st.Material = menuPanelGlass{}
	}
	return st
}

// coverRegion maps panel, in window DIPs, onto the pixels of an image that
// ui.Cover draws into box. It reports false when no pixel of the image falls
// under the panel.
func coverRegion(box, panel ui.Rect, bounds image.Rectangle) (image.Rectangle, bool) {
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 || box.W <= 0 || box.H <= 0 || panel.W <= 0 || panel.H <= 0 {
		return image.Rectangle{}, false
	}
	scale := math.Max(float64(box.W)/float64(w), float64(box.H)/float64(h))
	dw, dh := float64(w)*scale, float64(h)*scale
	originX := float64(box.X) + (float64(box.W)-dw)/2
	originY := float64(box.Y) + (float64(box.H)-dh)/2
	toX := func(v float64) int {
		fraction := (v - originX) / dw
		return bounds.Min.X + int(math.Round(math.Max(0, math.Min(1, fraction))*float64(w)))
	}
	toY := func(v float64) int {
		fraction := (v - originY) / dh
		return bounds.Min.Y + int(math.Round(math.Max(0, math.Min(1, fraction))*float64(h)))
	}
	x0, x1 := toX(float64(panel.X)), toX(float64(panel.X+panel.W))
	y0, y1 := toY(float64(panel.Y)), toY(float64(panel.Y+panel.H))
	x0, x1 = max(bounds.Min.X, min(x0, bounds.Max.X)), max(bounds.Min.X, min(x1, bounds.Max.X))
	y0, y1 = max(bounds.Min.Y, min(y0, bounds.Max.Y)), max(bounds.Min.Y, min(y1, bounds.Max.Y))
	if x1 <= x0 {
		if x0 >= bounds.Max.X {
			return image.Rectangle{}, false
		}
		x1 = x0 + 1
	}
	if y1 <= y0 {
		if y0 >= bounds.Max.Y {
			return image.Rectangle{}, false
		}
		y1 = y0 + 1
	}
	return image.Rect(x0, y0, x1, y1), true
}

// MetallicSheen paints the hover sheen of a custom-painted button, as
// metallic-paint.css does: translucent bands sweeping across the face at 115°.
// It is clipped to the face's own corners and painted over the hover fill but
// under the label. animate follows the desktop's reduce-motion setting; without
// motion the bands rest at their midpoint.
func MetallicSheen(p *ui.Painter, rect ui.Rect, radius float32, hovered, animate bool) {
	if !hovered || rect.W <= 0 || rect.H <= 0 {
		return
	}
	t := p.Theme()
	phase := float32(.5)
	now := p.Now()
	if animate && !shouldFreezeSheenAnimation(now) {
		period := float64(4500 * 1e6) // 4.5s, as the sheet's animation
		phase = float32(.5 - .5*math.Cos(float64(now.UnixNano())/period*2*math.Pi))
		p.AnimationFrame()
	}
	ink := t.Text.Alpha(.20)
	shine := t.AccentText.Alpha(.42)
	mid := t.TextMuted.Alpha(.14)
	band := max(float32(10), rect.W*.12)
	origin := rect.X - rect.W*.6 + phase*rect.W*1.2
	p.Clip(rect, radius, func() {
		for i := 0; i < 6; i++ {
			x := origin + float32(i)*band*1.35
			from, to := ink, shine
			if i%2 == 1 {
				from, to = shine, ink
			}
			if i == 0 || i == 5 {
				from, to = mid, from
			}
			p.FillGradient(ui.Rect{X: x, Y: rect.Y, W: band, H: rect.H}, ui.LinearGradient{From: from, To: to, Angle: 115}, 0)
		}
	})
}

// FlatBackdropFill is the base colour the page shows when no wallpaper and no
// gradient paints: the native sheet composites --dsw-alias-bg-base under
// #dsh-wallpaper::after (--dsw-alias-bg-mask-1, rgba(0,0,0,0.5)) on the dark
// half, so the sampled hero is ~12,12,12 while the token stays rgb(21,21,23).
// Kept as a pure helper: callers decide where a masked canvas applies.
func FlatBackdropFill(base ui.Color, dark bool) ui.Color {
	if !dark {
		return base
	}
	blended := ui.RGBA(0, 0, 0, 0.5).Over(base)
	// Force full opacity: a translucent canvas in a transparent window lets the
	// desktop tint the hero (round230 jitter), which the native mask never does.
	return ui.RGBA(blended.R, blended.G, blended.B, 255)
}

// ShowThroughAlpha is the share a chrome surface paints when no wallpaper or
// gradient sits behind it. The native glass model keeps the window transparent
// and composites the panel token at glassOpacity over the desktop (solved from
// pass303: sidebar 27,27,28 at 60% over a 5,20,46 desktop gives 18,24,35), so
// dark surfaces paint at glass/100 instead of covering the desktop.
func ShowThroughAlpha(p wallpapers.Preferences, dark bool) float32 {
	if p.Transparent || !dark {
		return 1
	}
	if p.Glass <= 0 || p.Glass >= 100 {
		return 1
	}
	return float32(p.Glass / 100)
}

// GlassComposite is the solved glass model: the panel token at alpha over the
// desktop pixel behind it.
func GlassComposite(panel, desktop ui.Color, alpha float32) ui.Color {
	return MaterialFill(panel, alpha).Over(desktop)
}
