package sidebar

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/product"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func navigationEvidence(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	if dir := os.Getenv("SIDEBAR_NAV_SNAPSHOT_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, tt.Image()); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func navigationRect(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok {
		t.Fatalf("missing %q: %v", label, tt.Texts())
	}
	return r
}

func navigationPanelRect(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	r := navigationRect(t, tt, label)
	if r.H == 22 {
		return ui.Rect{X: 14, Y: r.Y - 7, W: 252, H: 36}
	}
	return r
}

func TestSidebarNavigationSelectionTracksActionsAndKeepsGeometry(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		for _, dark := range []bool{false, true} {
			for _, collapsed := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/dark%v/collapsed%v", locale, dark, collapsed), func(t *testing.T) {
					i18n.SetLocale(locale)
					m := Model{Collapsed: collapsed, SystemDark: dark, ThemeID: "deepseek.light", Snapshot: session.ViewState{WorkspaceReady: true}}
					if dark {
						m.ThemeID = "deepseek.dark"
					}
					calls := map[string]int{}
					m.OnOpenMarket = func() { m.ActivePanel = "plugins"; calls["plugins"]++ }
					m.OnOpenSchedule = func() { m.ActivePanel = "schedule"; calls["schedule"]++ }
					m.OnOpenAssistant = func() { m.ActivePanel = "assistant"; calls["assistant"]++ }
					m.OnCreate = func() { m.ActivePanel = ""; calls["new"]++ }
					m.OnToggle = func() { m.Collapsed = !m.Collapsed }
					width := 280
					if collapsed {
						width = 56
					}
					tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, width, 800)
					tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
					labels := map[string]string{"plugins": i18n.T(i18n.KeySidebarMarket), "schedule": i18n.T(i18n.KeySidebarSchedule), "assistant": i18n.T(i18n.KeySidebarAssistant)}
					original := map[string]ui.Rect{}
					idle, hovered := map[string]color.RGBA{}, map[string]color.RGBA{}
					for id, label := range labels {
						r := navigationPanelRect(t, tt, label)
						original[id] = r
						tt.Move(float32(width-1), 600)
						idle[id] = tt.Image().RGBAAt(int(r.X+r.W-8), int(r.Y+r.H/2))
						tt.Move(r.X+r.W/2, r.Y+r.H/2)
						hovered[id] = tt.Image().RGBAAt(int(r.X+r.W-8), int(r.Y+r.H/2))
						if idle[id] == hovered[id] {
							t.Fatal("hover did not distinguish panel from the sidebar", id)
						}
					}
					for _, chosen := range []string{"plugins", "schedule", "assistant"} {
						if err := tt.Click(labels[chosen]); err != nil {
							t.Fatal(err)
						}
						tt.Move(float32(width-1), 600)
						tt.Frame()
						if calls[chosen] != 1 || m.ActivePanel != chosen {
							t.Fatalf("route=%q calls=%v", m.ActivePanel, calls)
						}
						for id, label := range labels {
							r := navigationPanelRect(t, tt, label)
							if r != original[id] || r.H != 36 || (collapsed && r.W != 36) {
								t.Fatalf("navigation moved %s: %+v -> %+v", id, original[id], r)
							}
							want := idle[id]
							got := tt.Image().RGBAAt(int(r.X+r.W-8), int(r.Y+r.H/2))
							if id == chosen {
								if got == idle[id] {
									t.Fatalf("%s selected paint matched idle %v", id, got)
								}
								if !collapsed && got == hovered[id] {
									t.Fatalf("%s selected paint %v must differ from hover %v", id, got, hovered[id])
								}
							} else if got != want {
								t.Fatalf("%s selected=%v paint=%v want=%v", id, id == chosen, got, want)
							}
						}
					}
					// Keyboard focus is visible even while the chosen route stays selected.
					for n := 0; n < 12 && !tt.Focused(labels["plugins"]); n++ {
						tt.Key(0, ui.KeyTab)
					}
					if !tt.Focused(labels["plugins"]) {
						t.Fatal("panel row absent from keyboard navigation")
					}
					navigationEvidence(t, tt, fmt.Sprintf("navigation-%s-dark%v-collapsed%v", locale, dark, collapsed))
					tt.Key(0, ui.KeyEnter)
					if calls["plugins"] != 2 || m.ActivePanel != "plugins" {
						t.Fatalf("keyboard did not activate real route callback: %v", calls)
					}
					if err := tt.Click(i18n.T(i18n.KeySidebarNewSessionFallback)); err != nil {
						t.Fatal(err)
					}
					tt.Move(float32(width-1), 600)
					tt.Frame()
					if m.ActivePanel != "" || calls["new"] != 1 {
						t.Fatal("new session did not return to the conversation")
					}
					for id, label := range labels {
						r := navigationPanelRect(t, tt, label)
						want := idle[id]
						got := tt.Image().RGBAAt(int(r.X+r.W-8), int(r.Y+r.H/2))
						if got != want {
							t.Fatalf("conversation left a panel selected: %s", label)
						}
					}
				})
			}
		}
	}
}

// buttonInkBox returns the tight bounding box of interior foreground (icon +
// text) pixels inside rect, excluding the 3px outer border/corner band. Any
// horizontal or vertical layout shift of the icon or label changes this box.
func buttonInkBox(img *image.RGBA, r ui.Rect) image.Rectangle {
	x0, y0 := int(r.X)+4, int(r.Y)+4
	x1, y1 := int(r.X+r.W)-4, int(r.Y+r.H)-4
	bg := img.RGBAAt(x0, y0)
	diff := func(a, b color.RGBA) int {
		abs := func(v int) int {
			if v < 0 {
				return -v
			}
			return v
		}
		return abs(int(a.R)-int(b.R)) + abs(int(a.G)-int(b.G)) + abs(int(a.B)-int(b.B))
	}
	minX, minY, maxX, maxY := x1, y1, x0, y0
	found := false
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if diff(img.RGBAAt(x, y), bg) > 24 {
				found = true
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if !found {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

func TestSidebarNavigationNewSessionShortcutHoverFocusAndUnbound(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, locale := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dark%v", locale, dark), func(t *testing.T) {
				i18n.SetLocale(locale)
				calls := 0
				m := Model{ThemeID: "deepseek.light", SystemDark: dark, OnCreate: func() { calls++ }, OnToggle: func() {}, NewSessionShortcut: "Ctrl+Alt+F8", Snapshot: session.ViewState{WorkspaceReady: true}}
				if dark {
					m.ThemeID = "deepseek.dark"
				}
				tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 800)
				tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
				now := time.Unix(3000, 0)
				tt.SetTime(now)
				label := i18n.T(i18n.KeySidebarNewSessionFallback)
				before := navigationRect(t, tt, label)
				if before.H != 38 {
					t.Fatalf("new session button height=%v want 38", before.H)
				}
				if tt.HasText(m.NewSessionShortcut) {
					t.Fatal("shortcut visible inline at rest")
				}
				restInk := buttonInkBox(tt.Image(), before)
				if restInk.Empty() {
					t.Fatal("missing rendered icon/label pixels inside new session button")
				}
				navigationEvidence(t, tt, fmt.Sprintf("new-session-%s-dark%v-rest", locale, dark))

				// Hover immediately: NO inline shortcut injection and ZERO displacement of button or inner icon/text.
				tt.Move(before.X+before.W/2, before.Y+before.H/2)
				tt.Frame()
				if tt.HasText(m.NewSessionShortcut) {
					t.Fatalf("hover injected inline shortcut into button flow: %v", tt.Texts())
				}
				if after := navigationRect(t, tt, label); after != before {
					t.Fatalf("hover changed button seat %+v -> %+v", before, after)
				}
				hoverInk := buttonInkBox(tt.Image(), before)
				if hoverInk != restInk {
					t.Fatalf("hover shifted inner icon/label pixels: rest=%v hover=%v", restInk, hoverInk)
				}
				navigationEvidence(t, tt, fmt.Sprintf("new-session-%s-dark%v-hover", locale, dark))

				// After tooltip delay, the shortcut appears in the Tooltip overlay (not inside the button box).
				tt.SetTime(now.Add(time.Second))
				wantTip := label + " (Ctrl+Alt+F8)"
				if !tt.HasText(wantTip) {
					t.Fatalf("tooltip missing %q: %v", wantTip, tt.Texts())
				}
				m.NewSessionShortcut = "Ctrl+Shift+F9"
				tt.SetTime(now.Add(1500 * time.Millisecond))
				wantTipUpdated := label + " (Ctrl+Shift+F9)"
				if !tt.HasText(wantTipUpdated) || tt.HasText(wantTip) {
					t.Fatalf("tooltip did not track updated binding: %v", tt.Texts())
				}
				m.NewSessionShortcut = ""
				tt.SetTime(now.Add(2000 * time.Millisecond))
				if tt.HasText(wantTipUpdated) {
					t.Fatalf("unbound shortcut leaked stale tooltip: %v", tt.Texts())
				}

				tt.Move(279, 600)
				tt.Frame()
				m.NewSessionShortcut = "Ctrl+Alt+F8"
				for n := 0; n < 8 && !tt.Focused(label); n++ {
					tt.Key(0, ui.KeyTab)
				}
				if !tt.Focused(label) {
					t.Fatalf("new session button unreachable via keyboard focus: %v", tt.Texts())
				}
				for _, txt := range tt.Texts() {
					if txt == m.NewSessionShortcut {
						t.Fatalf("keyboard focus injected standalone inline shortcut into button flow: %v", tt.Texts())
					}
				}
				if tipRect, ok := tt.Find(wantTip); ok {
					// Tooltip must render as an external overlay, never inside the button box.
					if tipRect.X >= before.X && tipRect.X+tipRect.W <= before.X+before.W &&
						tipRect.Y >= before.Y && tipRect.Y+tipRect.H <= before.Y+before.H {
						t.Fatalf("shortcut tooltip rendered inside button box: tip=%+v btn=%+v", tipRect, before)
					}
				}
				focusInk := buttonInkBox(tt.Image(), before)
				if focusInk != restInk {
					t.Fatalf("keyboard focus shifted inner icon/label pixels: rest=%v focus=%v", restInk, focusInk)
				}
				navigationEvidence(t, tt, fmt.Sprintf("new-session-%s-dark%v-focus", locale, dark))
				tt.Key(0, ui.KeyEnter)
				if calls != 1 {
					t.Fatalf("Enter calls=%d", calls)
				}
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				if calls != 2 {
					t.Fatalf("click calls=%d", calls)
				}
			})
		}
	}
}

func TestSidebarNavigationBrandActionAndDarwinDrag(t *testing.T) {
	for _, platform := range []string{"windows", "linux", "darwin"} {
		t.Run(platform, func(t *testing.T) {
			calls := 0
			m := Model{OnCreate: func() { calls++ }}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Row(c).FillWidth().Height(60).DragWindow().Children(func() {
					sidebarBrand(c, m, platform)
				})
			}, 280, 100)
			if err := tt.Click(product.Name); err != nil {
				t.Fatal(err)
			}
			want := 1
			if platform == "darwin" {
				want = 0
			}
			if calls != want {
				t.Fatalf("platform=%s brand calls=%d want=%d", platform, calls, want)
			}
		})
	}
	if runtime.GOOS == "darwin" {
		return
	}
	created, collapsed := 0, 0
	m := Model{OnCreate: func() { created++ }, OnToggle: func() { collapsed++ }}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 280, 800)
	if err := tt.Click(product.Name); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeySidebarCollapse)); err != nil {
		t.Fatal(err)
	}
	if created != 1 || collapsed != 1 {
		t.Fatalf("brand/collapse bubbled: created=%d collapsed=%d", created, collapsed)
	}
}

func TestSidebarNavigationNarrowHintStaysInsideButton(t *testing.T) {
	m := Model{NewSessionShortcut: "Control+Alt+Shift+Meta+F10 & F11", OnCreate: func() {}}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { newSessionButton(c, m) })
	}, 180, 60)
	label := i18n.T(i18n.KeySidebarNewSessionFallback)
	r := navigationRect(t, tt, label)
	restInk := buttonInkBox(tt.Image(), r)
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	if tt.HasText(m.NewSessionShortcut) {
		t.Fatalf("narrow button injected long shortcut inline: %v", tt.Texts())
	}
	after := navigationRect(t, tt, label)
	if after != r || after.H != 38 {
		t.Fatalf("narrow button changed seat on hover: %+v -> %+v", r, after)
	}
	if hoverInk := buttonInkBox(tt.Image(), after); hoverInk != restInk {
		t.Fatalf("narrow button shifted icon/label on hover: %v -> %v", restInk, hoverInk)
	}
	navigationEvidence(t, tt, "new-session-narrow-long-shortcut")
}
