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
// active pills: active owns the full sidebar-accent fill plus its leading
// sidebar-primary indicator, hover uses a lighter wash without indicator, and
// leaving or switching sections clears the previous highlight through Memo.
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

			// Active cell paints the SidebarPrimary leading indicator; hovered
			// and idle cells must not.
			selInd := sampleIndicator(sel)
			hovInd := sampleIndicator(hov)
			wantInd := color.RGBA{R: tokens.SidebarPrimary.R, G: tokens.SidebarPrimary.G, B: tokens.SidebarPrimary.B, A: 255}
			if dist(selInd, wantInd) > 12 {
				t.Fatalf("active leading indicator=%v, want SidebarPrimary~%v", selInd, wantInd)
			}
			if dist(hovInd, hovFill) > 6 {
				t.Fatalf("hovered cell should not paint an active indicator: left=%v fill=%v", hovInd, hovFill)
			}

			// Moving pointer onto the options pane must clear hover through Memo.
			tt.Move(600, 300)
			tt.Frame()
			img = tt.Image()
			afterLeave := sampleFill(hov)
			if dist(afterLeave, idleFill) > 2 {
				t.Fatalf("hover highlight stuck after pointer left rail: got %v want idle %v", afterLeave, idleFill)
			}

			// Clicking Agent 预设 transfers active cleanly (no dual active).
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
			if dist(sampleIndicator(hov), wantInd) > 12 {
				t.Fatalf("new active Agent 预设 missing leading indicator: got %v want ~%v", sampleIndicator(hov), wantInd)
			}
		})
	}
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
