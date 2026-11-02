package sidebar

import (
	"fmt"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/product"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
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
							if id == chosen {
								want = hovered[id]
							}
							got := tt.Image().RGBAAt(int(r.X+r.W-8), int(r.Y+r.H/2))
							if got != want {
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
				label := i18n.T(i18n.KeySidebarNewSessionFallback)
				before := navigationRect(t, tt, label)
				if tt.HasText(m.NewSessionShortcut) {
					t.Fatal("shortcut visible at rest")
				}
				navigationEvidence(t, tt, fmt.Sprintf("new-session-%s-dark%v-rest", locale, dark))
				tt.Move(before.X+before.W/2, before.Y+before.H/2)
				shortcut := navigationRect(t, tt, m.NewSessionShortcut)
				if shortcut.X < before.X || shortcut.X+shortcut.W > before.X+before.W {
					t.Fatalf("shortcut escaped button: %+v %+v", shortcut, before)
				}
				if after := navigationPanelRect(t, tt, label); after != before {
					t.Fatalf("hover changed button seat %+v -> %+v", before, after)
				}
				navigationEvidence(t, tt, fmt.Sprintf("new-session-%s-dark%v-hover", locale, dark))
				tt.Move(279, 600)
				if tt.HasText(m.NewSessionShortcut) {
					t.Fatal("shortcut remained after hover")
				}
				for n := 0; n < 8 && !tt.Focused(label); n++ {
					tt.Key(0, ui.KeyTab)
				}
				if !tt.Focused(label) || !tt.HasText(m.NewSessionShortcut) {
					t.Fatalf("keyboard focus missing hint: %v", tt.Texts())
				}
				navigationEvidence(t, tt, fmt.Sprintf("new-session-%s-dark%v-focus", locale, dark))
				tt.Key(0, ui.KeyEnter)
				if calls != 1 {
					t.Fatalf("Enter calls=%d", calls)
				}
				m.NewSessionShortcut = "Ctrl+Shift+F9"
				tt.Frame()
				if !tt.HasText(m.NewSessionShortcut) || tt.HasText("Ctrl+Alt+F8") {
					t.Fatal("hint did not follow current binding")
				}
				m.NewSessionShortcut = ""
				tt.Frame()
				if tt.HasText("Ctrl+Shift+F9") {
					t.Fatal("unbound command retained stale hint")
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
					colors := theme.Resolve("deepseek.light").Light
					sidebarBrand(c, m, colors, chrome.Color(colors.FG), chrome.Color(colors.Muted), platform)
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
	colors := theme.Resolve("deepseek.light").Light
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { newSessionButton(c, m, colors, chrome.Color(colors.FG), chrome.Color(colors.Border)) })
	}, 180, 60)
	r := navigationRect(t, tt, i18n.T(i18n.KeySidebarNewSessionFallback))
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	s := navigationRect(t, tt, m.NewSessionShortcut)
	if s.X < r.X || s.X+s.W > r.X+r.W || r.H != 38 {
		t.Fatalf("narrow shortcut escaped: button=%+v hint=%+v", r, s)
	}
	navigationEvidence(t, tt, "new-session-narrow-long-shortcut")
}
