package ui

import (
	"image/color"
	"slices"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

var menuFixtureIcon = MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16"><rect x="2" y="2" width="12" height="12" fill="currentColor"/></svg>`))

func TestMenuPresentationPreservesChoiceAndDisabledItems(t *testing.T) {
	var chosen []string
	tt := coreNewTester(func(c *context) {
		coreMenuButton(c, "Actions", func(m *Menu) {
			if m.Item("Open").Icon(menuFixtureIcon).Shortcut(Ctrl, KeyO).Chosen() {
				chosen = append(chosen, "Open")
			}
			m.Item("Locked delete").Icon(menuFixtureIcon).Danger(true).Disabled(true)
			m.Submenu("More", func(m *Menu) {
				if m.Item("Delete child").Icon(menuFixtureIcon).Danger(true).Chosen() {
					chosen = append(chosen, "Delete child")
				}
			}).Icon(menuFixtureIcon)
		})
	}, 400, 300)
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	p := tt.rt.menu.overlay
	if p == nil || len(p.presentation) != 4 || !p.presentation[2].danger || p.presentation[4].icon != menuFixtureIcon {
		t.Fatalf("UI snapshot lost main/submenu presentation: %+v", p)
	}
	if got := tt.Menu(); !slices.Equal(got, []string{"Open", "Locked delete", "More"}) {
		t.Fatalf("presentation changed label/order: %v", got)
	}
	if tt.h.menu.Items[0].ID != 1 || tt.h.menu.Items[2].Submenu.Items[0].ID != 4 || tt.h.menu.Items[0].Accelerator != accelerator(Ctrl, KeyO) {
		t.Fatal("presentation changed choice indices or shortcut")
	}
	if err := tt.ChooseMenuItem("Locked delete"); err == nil {
		t.Fatal("danger styling enabled a disabled item")
	}
	if err := tt.ChooseMenuItem("More", "Delete child"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(chosen, []string{"Delete child"}) || tt.rt.menu.overlay != nil {
		t.Fatalf("wrong choice or retained menu: %v", chosen)
	}
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Open"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(chosen, []string{"Delete child", "Open"}) {
		t.Fatalf("normal choice changed: %v", chosen)
	}
}

func TestMenuOpenTracksOwnAndDescendantMenusWithoutStaleOwner(t *testing.T) {
	show := true
	var parentOpen, buttonOpen, siblingOpen bool
	tt := coreNewTester(func(c *context) {
		parent := coreColumn(c).Key("row").Width(200)
		parent.Children(func() {
			if show {
				button := coreMenuButton(c, "More", func(m *Menu) { m.Item("Rename") })
				buttonOpen = button.MenuOpen()
			}
		})
		parent.ContextMenu(func(m *Menu) { m.Item("Row item") })
		parentOpen = parent.MenuOpen()
		siblingOpen = coreBox(c).Key("sibling").Size(100, 30).MenuOpen()
	}, 400, 300)
	if parentOpen || buttonOpen || siblingOpen {
		t.Fatal("menu reports open before use")
	}
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	if !parentOpen || !buttonOpen || siblingOpen {
		t.Fatal("descendant menu did not retain only its row")
	}
	tt.CloseMenu()
	if parentOpen || buttonOpen || siblingOpen {
		t.Fatal("dismissal left stale open state")
	}
	if err := tt.RightClick("More"); err != nil {
		t.Fatal(err)
	}
	if !parentOpen || buttonOpen {
		t.Fatal("ancestor context menu was confused with the child button menu")
	}
	if err := tt.ChooseMenuItem("Row item"); err != nil {
		t.Fatal(err)
	}
	if parentOpen || buttonOpen || tt.rt.menu.overlay != nil {
		t.Fatal("choice retained the previous menu owner")
	}
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	show = false
	tt.Frame()
	if parentOpen || tt.Menu() != nil || tt.rt.menu.overlay != nil {
		t.Fatal("unmounted menu owner retained its menu")
	}
	show = true
	tt.Frame()
	if parentOpen || buttonOpen {
		t.Fatal("reappearing key inherited the retired open state")
	}
}

// Wrap the same in-memory host so the engine draws its production menu overlay,
// rather than Tester's native-menu model. No operating-system window is opened.
type menuOverlayFixtureHost struct{ *headless }

// The tests freeze the clock, which would hold a menu below its place,
// rising, and ask for frames all along: menus show at once with less
// motion. TestMenuRisesIntoPlace has the motion.
func menuOverlayTester(view func(*context), width, height int) *Tester {
	tt := coreNewTester(view, width, height)
	tt.rt.host = menuOverlayFixtureHost{tt.h}
	tt.h.prefs = platform.Preferences{ReduceMotion: true}
	tt.rt.themeChanged()
	tt.Frame()
	return tt
}

// menuRowChecked reports whether the row of the item labeled label, under
// root, shows the check mark.
func menuRowChecked(root *node, label string) bool {
	var checked, found bool
	var walk func(*node)
	walk = func(n *node) {
		if n == nil || found {
			return
		}
		if n.kind == kindText && n.text == label && n.parent != nil {
			found = true
			for s := n.parent.first; s != nil; s = s.next {
				checked = checked || s.kind == kindIcon && s.svg == menuCheck
			}
			return
		}
		for k := n.first; k != nil; k = k.next {
			walk(k)
		}
	}
	walk(root)
	return checked
}

func TestMenuOverlayPaintsIconsDangerChecksAndSubmenu(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(map[bool]string{false: "light", true: "dark"}[dark], func(t *testing.T) {
			var parentOpen bool
			tt := menuOverlayTester(func(c *context) {
				parent := coreColumn(c).Key("owner").Width(160)
				parent.Children(func() {
					coreMenuButton(c, "Actions", func(m *Menu) {
						m.Item("Pinned").Checked(true).Icon(menuFixtureIcon)
						m.Item("Delete").Icon(menuFixtureIcon).Danger(true).Shortcut(Ctrl, KeyD)
						m.Item("Locked").Icon(menuFixtureIcon).Danger(true).Disabled(true)
						m.Submenu("More", func(m *Menu) { m.Item("Delete child").Icon(menuFixtureIcon).Danger(true).Checked(true) }).Icon(menuFixtureIcon)
					})
				})
				parentOpen = parent.MenuOpen()
			}, 600, 380)
			tt.SetDark(dark)
			if err := tt.Click("Actions"); err != nil {
				t.Fatal(err)
			}
			if !parentOpen || tt.rt.menu.overlay == nil {
				t.Fatal("production overlay did not report its owner")
			}
			deleteBox, ok := tt.Find("Delete")
			if !ok {
				t.Fatal("no delete item in overlay")
			}
			var icon *node
			walk := func(root *node) {}
			walk = func(root *node) {
				if root == nil {
					return
				}
				if root.kind == kindText && root.text == "Delete" {
					for n := root.parent.first; n != nil; n = n.next {
						if n.kind == kindIcon {
							icon = n
						}
					}
				}
				for n := root.first; n != nil; n = n.next {
					walk(n)
				}
			}
			walk(tt.rt.c.root)
			if icon == nil || icon.Bounds().W != 14 || icon.ts.color != tt.rt.c.theme.Danger {
				t.Fatal("danger icon is not a 14px vector in the danger color")
			}
			box := icon.Bounds()
			pixel := tt.Image().RGBAAt(int(box.X+box.W/2), int(box.Y+box.H/2))
			want := tt.rt.c.theme.Danger
			if pixel != (color.RGBA{want.R, want.G, want.B, want.A}) {
				t.Fatalf("danger icon was not painted: %v want %v", pixel, want)
			}
			if deleteBox.X <= box.X+box.W || !tt.HasText(accelerator(Ctrl, KeyD)) {
				t.Fatal("icon/label/shortcut columns overlap or disappeared")
			}
			if !menuRowChecked(tt.rt.c.root, "Pinned") || menuRowChecked(tt.rt.c.root, "Delete") {
				t.Fatal("the check mark is not on the checked item alone")
			}
			if err := tt.Click("Locked"); err != nil {
				t.Fatal(err)
			}
			if tt.rt.menu.overlay == nil {
				t.Fatal("disabled item closed the menu")
			}
			more, ok := tt.Find("More")
			if !ok {
				t.Fatal("submenu item disappeared")
			}
			tt.Move(more.X+more.W/2, more.Y+more.H/2)
			if !tt.HasText("Delete child") {
				t.Fatal("submenu item did not render")
			}
			if err := tt.Click("Delete child"); err != nil {
				t.Fatal(err)
			}
			if parentOpen || tt.rt.menu.overlay != nil {
				t.Fatal("submenu choice left parent menu open")
			}
			if err := tt.Click("Actions"); err != nil {
				t.Fatal(err)
			}
			tt.Key(0, KeyEscape)
			if parentOpen || tt.rt.menu.overlay != nil {
				t.Fatal("Escape left menu open")
			}
		})
	}
}

func TestMenuOverlayPointerLeaveDelayReentryAndKeyboard(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "explicit"}[enabled], func(t *testing.T) {
			var ownerOpen bool
			tt := menuOverlayTester(func(c *context) {
				button := coreMenuButton(c, "Actions", func(m *Menu) {
					m.CloseOnPointerLeave(enabled)
					m.Item("Rename")
					m.Submenu("More", func(m *Menu) { m.Item("Child") })
				})
				ownerOpen = button.MenuOpen()
			}, 600, 380)
			base := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
			tt.SetTime(base)
			tt.Move(550, 350)
			tt.Key(0, KeyTab)
			tt.Key(0, KeyDown)
			if !ownerOpen {
				t.Fatal("keyboard did not open menu")
			}
			tt.SetTime(base.Add(time.Second))
			if !ownerOpen {
				t.Fatal("menu opened outside pointer armed a leave timer")
			}
			anchor, _ := tt.Find("Actions")
			tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
			tt.SetTime(base.Add(2 * time.Second))
			if !ownerOpen {
				t.Fatal("hovering the anchor closed its menu")
			}
			tt.Move(550, 350)
			tt.SetTime(base.Add(2*time.Second + 199*time.Millisecond))
			if !ownerOpen {
				t.Fatal("menu closed before200ms")
			}
			more, _ := tt.Find("More")
			tt.Move(more.X+more.W/2, more.Y+more.H/2)
			tt.SetTime(base.Add(3 * time.Second))
			if !ownerOpen || !tt.HasText("Child") {
				t.Fatal("returning into the menu failed to cancel close or open submenu")
			}
			child, _ := tt.Find("Child")
			tt.Move(550, 350)
			tt.SetTime(base.Add(3*time.Second + 199*time.Millisecond))
			tt.Move(child.X+child.W/2, child.Y+child.H/2)
			tt.SetTime(base.Add(4 * time.Second))
			if !ownerOpen {
				t.Fatal("submenu hover failed to cancel close")
			}
			tt.Move(550, 350)
			tt.SetTime(base.Add(4*time.Second + 199*time.Millisecond))
			if !ownerOpen {
				t.Fatal("second leave closed early")
			}
			tt.SetTime(base.Add(4*time.Second + 200*time.Millisecond))
			if ownerOpen == enabled {
				t.Fatalf("200ms menu state wrong: open=%v explicit=%v", ownerOpen, enabled)
			}
			if enabled && tt.rt.menu.overlay != nil {
				t.Fatal("deadline retained stale owner")
			}
		})
	}
}

func TestMenuPlainContextAnchorRetainsPointerLeaveSubscription(t *testing.T) {
	var open bool
	var id uint64
	tt := menuOverlayTester(func(c *context) {
		anchor := coreBox(c).Size(120, 40).Label("Plain anchor").ContextMenu(func(m *Menu) { m.CloseOnPointerLeave(true); m.Item("Action") })
		id, open = anchor.id, anchor.MenuOpen()
	}, 400, 250)
	base := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	tt.SetTime(base)
	if err := tt.RightClick("Plain anchor"); err != nil {
		t.Fatal(err)
	}
	if !open {
		t.Fatal("plain context anchor did not open")
	}
	tt.Frame()
	if tt.rt.states[id].flags&flagHover == 0 {
		t.Fatal("later frame dropped the plain anchor hover subscription")
	}
	// Feed only a pointer move, without Tester's automatic frame, so the
	// assertion proves the live subscription itself asks for a frame.
	tt.h.requested.Store(false)
	tt.rt.pointerMove(370, 220)
	if !tt.h.requested.Load() {
		t.Fatal("leaving a plain context anchor did not request a frame")
	}
	tt.settle()
	tt.SetTime(base.Add(199 * time.Millisecond))
	if !open {
		t.Fatal("plain anchor leave closed before200ms")
	}
	tt.SetTime(base.Add(200 * time.Millisecond))
	if open || tt.rt.menu.overlay != nil {
		t.Fatal("plain anchor leave retained the menu owner")
	}
}

func TestMenuShortcutLabelIsDisplayOnly(t *testing.T) {
	chosen := 0
	tt := menuOverlayTester(func(c *context) {
		coreMenuButton(c, "Actions", func(m *Menu) {
			if m.Item("Rename").ShortcutLabel("Ctrl+K Ctrl+R").Chosen() {
				chosen++
			}
			m.Item("Hidden shortcut").Shortcut(Ctrl, KeyD).ShortcutLabel("")
		})
	}, 400, 240)
	if err := tt.Click("Actions"); err != nil {
		t.Fatal(err)
	}
	p := tt.rt.menu.overlay
	if !tt.HasText("Ctrl+K Ctrl+R") || tt.HasText(accelerator(Ctrl, KeyD)) {
		t.Fatal("display label override was not shown/hidden")
	}
	if p.menu.Items[0].Accelerator != "" || p.menu.Items[1].Accelerator != accelerator(Ctrl, KeyD) {
		t.Fatal("display label changed platform key metadata")
	}
	tt.Key(Ctrl, KeyK)
	tt.Key(Ctrl, KeyR)
	if chosen != 0 {
		t.Fatal("display-only shortcut dispatched a menu action")
	}
	if err := tt.Click("Rename"); err != nil {
		t.Fatal(err)
	}
	if chosen != 1 {
		t.Fatal("display label changed action identity")
	}
}

func TestMenuPresentationNilAndFalseRestoreDefault(t *testing.T) {
	run := &menuRun{collect: true}
	menu := &Menu{run: run, menu: &platform.Menu{}}
	item := menu.Item("Normal").Icon(menuFixtureIcon).Danger(true).Icon(nil).Danger(false)
	if run.presentation[item.seq].icon != nil || run.presentation[item.seq].danger {
		t.Fatal("nil icon or Danger(false) did not reset presentation")
	}
	if item.seq != 1 || item.p.ID != 1 || item.p.Label != "Normal" {
		t.Fatal("style reset changed the action identity")
	}
}
