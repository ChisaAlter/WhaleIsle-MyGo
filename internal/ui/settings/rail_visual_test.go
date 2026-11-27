package settings

import (
	"fmt"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

// TestSettingsRailScrollbarDoesNotOverlapNavPills verifies that when the
// settings navigation rail overflows and paints its vertical overlay scrollbar,
// every navigation pill stops strictly left of the scrollbar thumb and drag
// track, and keeps the exact same horizontal geometry when no overflow occurs.
func TestSettingsRailScrollbarDoesNotOverlapNavPills(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	for _, sbWidth := range []float32{5, 6} {
		var sidebar ui.Color
		m := fullRailModel()
		m.ActiveSection = SectionModels
		m.SectionContent = func(c *ui.Context) { ui.Text(c, "body") }

		tt := ui.NewTester(func(c *ui.Context) {
			c.Theme().ScrollbarWidth = sbWidth
			c.Theme().ScrollbarAlways = false
			sidebar = tk(c).Sidebar
			View(c, m)
		}, 900, 480)
		tt.SetTime(time.Unix(1000, 0))
		tt.SetTime(time.Unix(1001, 0))

		railRect, ok := tt.Find("设置导航")
		if !ok || railRect.W == 0 {
			t.Fatalf("sbWidth=%g: rail not found: %v", sbWidth, tt.Texts())
		}
		railInnerRight := railRect.X + railRect.W - design.BorderWidth
		// Matches third_party/mygo/ui/paint.go scrollBars():
		//   vTrack.X = box.X + box.W - width - 6
		//   v.X      = box.X + box.W - width - 3 (widens left by 2 while dragging)
		trackLeft := railInnerRight - sbWidth - 6
		thumbLeft := railInnerRight - sbWidth - 3

		findRailCell := func(label string) ui.Rect {
			t.Helper()
			var best ui.Rect
			for _, r := range tt.FindAll(label) {
				if r.X < railInnerRight && r.W*r.H > best.W*best.H {
					best = r
				}
			}
			if best.W == 0 {
				t.Fatalf("sbWidth=%g: rail cell %q not found", sbWidth, label)
			}
			return best
		}

		modelsBefore := findRailCell("模型")
		presets := findRailCell("Agent 预设")

		// Hover over Agent 预设 so the overlay scrollbar becomes visible.
		tt.Move(presets.X+presets.W/2, presets.Y+presets.H/2)
		tt.Frame()

		// Confirm the rail actually overflows and can scroll vertically.
		tt.Scroll(presets.X+presets.W/2, presets.Y+presets.H/2, 0, 40)
		tt.Frame()
		modelsAfterScroll := findRailCell("模型")
		if modelsAfterScroll.Y == modelsBefore.Y {
			t.Fatalf("sbWidth=%g: rail did not scroll at height 480; overflow case unproven", sbWidth)
		}
		// Scroll back to top and re-hover Agent 预设 for geometry + pixel checks.
		tt.Scroll(presets.X+presets.W/2, presets.Y+presets.H/2, 0, -40)
		tt.Frame()
		presets = findRailCell("Agent 预设")
		tt.Move(presets.X+presets.W/2, presets.Y+presets.H/2)
		tt.Frame()

		for _, label := range []string{"通用设置", "模型", "Agent 预设", "技能"} {
			cell := findRailCell(label)
			pillRight := cell.X + cell.W
			if pillRight > trackLeft {
				t.Fatalf("sbWidth=%g: %q right edge %.1f overlaps scrollbar track left %.1f (thumb left %.1f)",
					sbWidth, label, pillRight, trackLeft, thumbLeft)
			}
			if pillRight > thumbLeft-2 {
				t.Fatalf("sbWidth=%g: %q right edge %.1f intrudes into dragging scrollbar thumb %.1f",
					sbWidth, label, pillRight, thumbLeft-2)
			}
		}

		// Verify the scrollbar thumb is actually painted inside the reserved
		// gutter strictly to the right of the pills.
		saveBodySnapshot(t, tt, fmt.Sprintf("settings-rail-scroll-w%.0f.png", sbWidth))
		topCell := findRailCell("用量与费用")
		models := findRailCell("模型")
		img := tt.Image()
		thumbSampleX := int(thumbLeft + sbWidth/2)
		if float32(thumbSampleX) <= models.X+models.W {
			t.Fatalf("sbWidth=%g: thumb sample x=%d not right of pill edge %.1f", sbWidth, thumbSampleX, models.X+models.W)
		}
		thumbVisible := false
		for y := int(topCell.Y); y < int(railRect.Y+railRect.H-4); y++ {
			px := img.RGBAAt(thumbSampleX, y)
			if absDiff(px.R, sidebar.R) > 12 || absDiff(px.G, sidebar.G) > 12 || absDiff(px.B, sidebar.B) > 12 {
				thumbVisible = true
				break
			}
		}
		if !thumbVisible {
			t.Fatalf("sbWidth=%g: expected visible scrollbar thumb at x=%d inside gutter (y=%d..%d)",
				sbWidth, thumbSampleX, int(topCell.Y), int(railRect.Y+railRect.H-4))
		}

		// Expanding the viewport so all rail entries fit must keep the exact
		// same pill X and W (no horizontal layout shift).
		tt.SetSize(900, 1200)
		tt.Frame()
		modelsTall := findRailCell("模型")
		if math.Abs(float64(modelsTall.X-models.X)) > 0.01 || math.Abs(float64(modelsTall.W-models.W)) > 0.01 {
			t.Fatalf("sbWidth=%g: rail pill geometry shifted without overflow: overflow=%+v tall=%+v", sbWidth, models, modelsTall)
		}
	}
}

// TestSettingsRailActiveAndHoverAreVisuallyDistinct verifies that hovering an
// inactive rail entry next to the active entry never renders two identical
// active pills: active owns the full sidebar-accent fill with neutral ink and
// NO leading blue/primary vertical bar, hover uses a lighter wash, and leaving
// or switching sections clears the previous highlight cleanly.
func TestSettingsRailActiveAndHoverAreVisuallyDistinct(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	for _, tc := range []struct {
		name   string
		dark   bool
		accent ui.Color
	}{
		{"light-default", false, ui.Color{}},
		{"dark-default", true, ui.Color{}},
		{"light-accent", false, ui.RGB(77, 107, 254)},
		{"dark-accent", true, ui.RGB(77, 107, 254)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			active := SectionModels
			var tokens design.Tokens
			m := fullRailModel()
			m.OnSelectModels = func() { active = SectionModels }
			m.OnSelectAgentPresets = func() { active = SectionAgentPresets }
			m.SectionContent = func(c *ui.Context) { ui.Text(c, "options body") }

			tt := ui.NewTester(func(c *ui.Context) {
				scheme := design.Light
				if tc.dark {
					scheme = design.Dark
				}
				design.Bind(c, scheme, tc.accent, ui.Color{})
				tokens = *tk(c)
				cur := m
				cur.ActiveSection = active
				View(c, cur)
			}, 900, 800)
			tt.SetDark(tc.dark)
			tt.SetTime(time.Unix(1000, 0))
			tt.SetTime(time.Unix(1001, 0))

			findCell := func(label string) ui.Rect {
				t.Helper()
				var best ui.Rect
				for _, r := range tt.FindAll(label) {
					if r.X < 208 && r.W*r.H > best.W*best.H {
						best = r
					}
				}
				if best.W == 0 {
					t.Fatalf("rail cell %q not found", label)
				}
				return best
			}

			sel := findCell("模型")
			hov := findCell("Agent 预设")
			idle := findCell("技能")

			// Hover Agent 预设 while 模型 is active.
			tt.Move(hov.X+hov.W/2, hov.Y+hov.H/2)
			tt.Frame()
			saveBodySnapshot(t, tt, "settings-rail-"+tc.name+".png")

			img := tt.Image()
			sampleFill := func(r ui.Rect) color.RGBA {
				return img.RGBAAt(int(r.X+r.W)-6, int(r.Y+r.H/2))
			}
			sampleIndicator := func(r ui.Rect) color.RGBA {
				return img.RGBAAt(int(r.X)+3, int(r.Y+r.H/2))
			}

			selFill := sampleFill(sel)
			hovFill := sampleFill(hov)
			idleFill := sampleFill(idle)

			// Hover fill must be clearly distinct from BOTH active and idle.
			if selFill == hovFill {
				t.Fatalf("active and hovered pills paint identical fill %v", selFill)
			}
			if hovFill == idleFill {
				t.Fatalf("hovered pill painted bare idle fill %v (no hover feedback)", hovFill)
			}
			// Distance from bare sidebar to hover must be strictly less than
			// distance from bare sidebar to active (hover is a lighter wash).
			dist := func(a, b color.RGBA) int {
				return absDiff(a.R, b.R) + absDiff(a.G, b.G) + absDiff(a.B, b.B)
			}
			if dist(hovFill, idleFill) >= dist(selFill, idleFill) {
				t.Fatalf("hover wash %v should be subtler than active wash %v over idle %v", hovFill, selFill, idleFill)
			}

			// Active, hovered, and idle cells must NEVER paint a SidebarPrimary
			// leading vertical bar: the left edge (X+2..X+4) must stay uniform
			// with the pill's own fill.
			primaryRGB := color.RGBA{R: tokens.SidebarPrimary.R, G: tokens.SidebarPrimary.G, B: tokens.SidebarPrimary.B, A: 255}
			for dx := 2; dx <= 4; dx++ {
				selLeft := img.RGBAAt(int(sel.X)+dx, int(sel.Y+sel.H/2))
				if dist(selLeft, selFill) > 4 {
					t.Fatalf("active cell left edge dx=%d painted unexpected stripe %v (want selFill %v; primary=%v)", dx, selLeft, selFill, primaryRGB)
				}
				if dist(primaryRGB, selFill) > 20 && dist(selLeft, primaryRGB) <= 12 {
					t.Fatalf("active cell left edge dx=%d still paints SidebarPrimary bar %v (primary=%v)", dx, selLeft, primaryRGB)
				}
			}
			hovInd := sampleIndicator(hov)
			if dist(hovInd, hovFill) > 6 {
				t.Fatalf("hovered cell should not paint a leading indicator: left=%v fill=%v", hovInd, hovFill)
			}

			// Moving pointer onto the options pane must clear hover cleanly.
			tt.Move(600, 300)
			tt.Frame()
			img = tt.Image()
			afterLeave := sampleFill(hov)
			if dist(afterLeave, idleFill) > 2 {
				t.Fatalf("hover highlight stuck after pointer left rail: got %v want idle %v", afterLeave, idleFill)
			}

			// Clicking Agent 预设 transfers active cleanly (no dual active, no blue bar).
			tt.ClickAt(hov.X+hov.W/2, hov.Y+hov.H/2)
			tt.Move(600, 300)
			tt.Frame()
			img = tt.Image()
			if active != SectionAgentPresets {
				t.Fatalf("click did not activate Agent 预设: active=%s", active)
			}
			if dist(sampleFill(sel), idleFill) > 2 {
				t.Fatalf("previous active 模型 kept highlight after switch: got %v want idle %v", sampleFill(sel), idleFill)
			}
			newSelFill := sampleFill(hov)
			if dist(newSelFill, selFill) > 4 {
				t.Fatalf("new active Agent 预设 fill=%v, want selFill~%v", newSelFill, selFill)
			}
			if dist(sampleIndicator(hov), newSelFill) > 4 {
				t.Fatalf("new active Agent 预设 left edge painted stripe %v, want uniform fill %v", sampleIndicator(hov), newSelFill)
			}
		})
	}
}

// TestSettingsRailClickAfterScrollHitsVisibleTarget verifies that scrolling
// the settings navigation rail and clicking a scrolled-in entry activates the
// clicked entry without Memo hit-test offset drift.
func TestSettingsRailClickAfterScrollHitsVisibleTarget(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	active := SectionGeneral
	m := fullRailModel()
	m.OnSelectAbout = func() { active = SectionAbout }
	m.SectionContent = func(c *ui.Context) { ui.Text(c, "body") }

	tt := ui.NewTester(func(c *ui.Context) {
		cur := m
		cur.ActiveSection = active
		View(c, cur)
	}, 900, 360)
	tt.Frame()

	railRect, ok := tt.Find("设置导航")
	if !ok {
		t.Fatalf("rail missing: %v", tt.Texts())
	}
	// Scroll the rail down to bring 关于 into view, then click it directly.
	tt.Scroll(railRect.X+railRect.W/2, railRect.Y+railRect.H/2, 0, 600)
	tt.Frame()

	var about ui.Rect
	for _, r := range tt.FindAll("关于") {
		if r.X < railRect.X+railRect.W && r.W*r.H > about.W*about.H {
			about = r
		}
	}
	if about.W == 0 {
		t.Fatalf("关于 rail button missing after scroll")
	}
	tt.ClickAt(about.X+about.W/2, about.Y+about.H/2)
	tt.Frame()
	if active != SectionAbout {
		t.Fatalf("clicking 关于 after scrolling rail selected %q, want %q (about rect=%+v)", active, SectionAbout, about)
	}
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
