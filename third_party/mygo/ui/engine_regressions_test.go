package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

// TestModalSelectDismissKeepsFocusOnTrigger verifies all four dismissal paths
// of a SelectBase inside a modal dialog that has a leading text input:
// clicking the trigger again, choosing an option, clicking the backdrop, and
// pressing Escape must keep focus on the trigger and never hijack it back to
// focusOrder[0].
func TestModalSelectDismissKeepsFocusOnTrigger(t *testing.T) {
	open := true
	search := ""
	choice := "tray"
	var triggerRect, dialogBlank Rect

	tt := coreNewTester(func(c *context) {
		coreModal(c, &open, func() {
			coreColumn(c).Width(360).Height(260).Gap(12).Children(func() {
				coreTextInput(c, &search).Label("Search settings").FillWidth()
				sel := coreSelectBase(c, &choice)
				sel.Trigger.Label("On close").Padding(6, 10).Children(func() {
					coreText(c, choice)
				})
				triggerRect = sel.Trigger.Bounds()
				sel.Popup(func(panel *node) {
					for _, opt := range []string{"tray", "quit"} {
						it := sel.Item(opt).Padding(6, 10)
						it.Children(func() { coreText(c, opt) })
					}
				})
				blank := coreBox(c).FillWidth().Height(80)
				dialogBlank = blank.Bounds()
				coreButton(c, "Done")
			})
		})
	}, 600, 480)

	// Initial modal open focuses focusOrder[0] (Search settings).
	if !tt.Focused("Search settings") {
		t.Fatal("initial modal open did not focus first input")
	}

	// 1. Open select, then click trigger again to close: focus stays on trigger.
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("quit") {
		t.Fatal("select popup did not open")
	}
	tt.ClickAt(triggerRect.X+triggerRect.W/2, triggerRect.Y+triggerRect.H/2)
	if tt.HasText("quit") {
		t.Fatal("clicking trigger did not close select popup")
	}
	if tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("trigger toggle hijacked focus: search=%v trigger=%v", tt.Focused("Search settings"), tt.Focused("On close"))
	}

	// 2. Open select, choose option: value changes and focus stays on trigger.
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("quit"); err != nil {
		t.Fatal(err)
	}
	if choice != "quit" {
		t.Fatalf("choice = %q, want quit", choice)
	}
	if tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("choosing option hijacked focus: search=%v trigger=%v", tt.Focused("Search settings"), tt.Focused("On close"))
	}

	// 3. Open select, click backdrop whitespace to close: focus stays on trigger.
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(dialogBlank.X+dialogBlank.W/2, dialogBlank.Y+dialogBlank.H/2)
	if tt.HasText("tray") {
		t.Fatal("clicking backdrop did not close select popup")
	}
	if tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("backdrop close hijacked focus: search=%v trigger=%v", tt.Focused("Search settings"), tt.Focused("On close"))
	}

	// 4. Open select, press Escape to close: focus stays on trigger.
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, KeyEscape)
	if tt.HasText("tray") || !open {
		t.Fatalf("Escape did not close only the select popup (open=%v)", open)
	}
	if tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("Escape close hijacked focus: search=%v trigger=%v", tt.Focused("Search settings"), tt.Focused("On close"))
	}

	// 5. Clicking modal whitespace when no select is open clears focus without
	// hijacking back to Search settings; Tab and Shift+Tab still enter first/last.
	tt.ClickAt(dialogBlank.X+dialogBlank.W/2, dialogBlank.Y+dialogBlank.H/2)
	if tt.rt.focused != 0 || tt.Focused("Search settings") {
		t.Fatalf("clicking modal whitespace hijacked focus: focused=%d search=%v", tt.rt.focused, tt.Focused("Search settings"))
	}
	tt.Key(0, KeyTab)
	if !tt.Focused("Search settings") {
		t.Fatal("Tab after whitespace blur did not enter modal first item")
	}
	tt.ClickAt(dialogBlank.X+dialogBlank.W/2, dialogBlank.Y+dialogBlank.H/2)
	if tt.rt.focused != 0 {
		t.Fatalf("second whitespace click did not stay unfocused: focused=%d", tt.rt.focused)
	}
	tt.Key(Shift, KeyTab)
	if !tt.Focused("Done") {
		t.Fatal("Shift+Tab after whitespace blur did not enter modal last item")
	}
}

// TestModalPopoverRestoreFocusBeatsArrangeFocus verifies that closing a
// non-modal PopoverBase inside a modal dialog restores focus to its opener
// rather than letting arrangeFocus hijack focus to focusOrder[0].
func TestModalPopoverRestoreFocusBeatsArrangeFocus(t *testing.T) {
	dialogOpen := true
	popOpen := false
	search := ""
	var blankRect Rect

	tt := coreNewTester(func(c *context) {
		coreModal(c, &dialogOpen, func() {
			coreColumn(c).Width(360).Height(260).Gap(12).Children(func() {
				coreTextInput(c, &search).Label("Search").FillWidth()
				trig := coreButton(c, "More")
				if trig.Clicked() {
					popOpen = !popOpen
				}
				corePopoverBase(c, trig, &popOpen, func(panel *node) {
					panel.Padding(8).Children(func() {
						if coreButton(c, "Inner action").Clicked() {
							popOpen = false
						}
					})
				})
				blank := coreBox(c).FillWidth().Height(80)
				blankRect = blank.Bounds()
			})
		})
	}, 600, 480)

	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	if !popOpen {
		t.Fatal("popover did not open")
	}
	// Clicking an inner button closes the popover and restores focus to More.
	if err := tt.Click("Inner action"); err != nil {
		t.Fatal(err)
	}
	if popOpen {
		t.Fatal("inner action did not close popover")
	}
	if tt.Focused("Search") || !tt.Focused("More") {
		t.Fatalf("popover close failed to restore focus to opener: search=%v more=%v", tt.Focused("Search"), tt.Focused("More"))
	}

	// Clicking modal whitespace outside popover closes it and restores opener focus.
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(blankRect.X+blankRect.W/2, blankRect.Y+blankRect.H/2)
	if popOpen {
		t.Fatal("outside click did not close popover")
	}
	if tt.Focused("Search") || !tt.Focused("More") {
		t.Fatalf("popover outside press hijacked focus: search=%v more=%v", tt.Focused("Search"), tt.Focused("More"))
	}
}

// TestSelectBaseKeyboardSkipsDisabledOptions verifies that Down, Up, Home, End,
// and Enter/Space never highlight or choose Disabled options in SelectBase.
func TestSelectBaseKeyboardSkipsDisabledOptions(t *testing.T) {
	selected := "enabled-mid"
	opts := []struct {
		val      string
		disabled bool
	}{
		{"disabled-first", true},
		{"enabled-mid", false},
		{"disabled-mid", true},
		{"enabled-last", false},
		{"disabled-last", true},
	}

	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(20).Children(func() {
			sel := coreSelectBase(c, &selected)
			sel.Trigger.Label("Picker").Padding(6, 10).Children(func() {
				coreText(c, selected)
			})
			sel.Popup(func(panel *node) {
				for _, o := range opts {
					it := sel.Item(o.val).Padding(4, 8)
					if o.disabled {
						it.Disabled(true)
					}
					it.Children(func() { coreText(c, o.val) })
				}
			})
		})
	}, 360, 320)

	tt.Key(0, KeyTab)
	if !tt.Focused("Picker") {
		t.Fatal("trigger not focused")
	}
	// Open via Down arrow; initial highlight is on enabled-mid (index 1).
	tt.Key(0, KeyDown)
	if !tt.HasText("enabled-last") {
		t.Fatal("popup did not open via Down")
	}
	// Down must skip disabled-mid (index 2) and land on enabled-last (index 3).
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if selected != "enabled-last" {
		t.Fatalf("Down+Enter chose %q, want enabled-last", selected)
	}

	// Re-open, press Down at enabled-last: must NOT move onto disabled-last.
	tt.Key(0, KeyDown)
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnter)
	if selected != "enabled-last" {
		t.Fatalf("Down past last enabled chose %q, want enabled-last", selected)
	}

	// Re-open, press Home: must land on first enabled (enabled-mid), not disabled-first.
	tt.Key(0, KeyDown)
	tt.Key(0, KeyHome)
	tt.Key(0, KeySpace)
	if selected != "enabled-mid" {
		t.Fatalf("Home+Space chose %q, want enabled-mid", selected)
	}

	// Re-open, press End: must land on last enabled (enabled-last), not disabled-last.
	tt.Key(0, KeyDown)
	tt.Key(0, KeyEnd)
	tt.Key(0, KeyEnter)
	if selected != "enabled-last" {
		t.Fatalf("End+Enter chose %q, want enabled-last", selected)
	}

	// Re-open, press Up: must skip disabled-mid and return to enabled-mid.
	tt.Key(0, KeyDown)
	tt.Key(0, KeyUp)
	tt.Key(0, KeyEnter)
	if selected != "enabled-mid" {
		t.Fatalf("Up+Enter chose %q, want enabled-mid", selected)
	}
}

// TestScrollMemoSmoothWheelHitTargetsStayAccurate verifies that scrolling a
// Scroll container carrying .Memo(...) via discrete smooth wheel frames updates
// child hit rects and labels so clicking scrolled-in rows hits the right item.
func TestScrollMemoSmoothWheelHitTargetsStayAccurate(t *testing.T) {
	clicked := -1
	var scrollSt ScrollState

	tt := newRepaintTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			if scrollSt.Y > 0 {
				coreText(c, "Scrolled")
			}
			coreScroll(c).Key("nav-scroll").Size(200, 120).TrackScroll(&scrollSt).Memo(1).Children(func() {
				for i := range 30 {
					idx := i
					b := coreButton(c, fmt.Sprintf("Nav %d", idx)).FillWidth().Height(24)
					if b.Clicked() {
						clicked = idx
					}
				}
			})
		})
	})
	tt.rt.smoothWheelInTest = true
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerMove, X: 50, Y: 50})
	tt.frame(16 * time.Millisecond)

	// Scroll down ~240 DIPs with discrete mouse wheel ticks.
	tt.rt.event(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 50, Y: 50, DY: 240, Precise: false})
	for range 60 {
		_, more := tt.frame(16 * time.Millisecond)
		if !more {
			break
		}
	}

	if scrollSt.Y < 200 {
		t.Fatalf("TrackScroll did not sync after smooth wheel scroll: %+v", scrollSt)
	}
	if !tt.HasText("Scrolled") {
		t.Fatalf("view did not observe TrackScroll update after smooth wheel scroll: %+v", scrollSt)
	}

	row12, ok := tt.Find("Nav 12")
	if !ok {
		t.Fatal("Nav 12 label not found")
	}
	if row12.Y < 0 || row12.Y+row12.H > 140 {
		t.Fatalf("Nav 12 not scrolled into viewport: %v", row12)
	}
	tt.ClickAt(row12.X+row12.W/2, row12.Y+row12.H/2)
	if clicked != 12 {
		t.Fatalf("clicking scrolled-in Nav 12 hit item %d instead of 12 (rect=%v)", clicked, row12)
	}
}

type captureTestingT struct {
	errors []string
}

func (c *captureTestingT) Helper() {}
func (c *captureTestingT) Errorf(format string, args ...any) {
	c.errors = append(c.errors, fmt.Sprintf(format, args...))
}

// TestAssertHoverFocusZeroShiftGuardsRegressions verifies both positive
// (stable views pass, including tooltips and open modals) and negative
// (hover text injection, animated hover width growth, focus border shift are
// caught) behavior of FlowSnapshot and AssertHoverFocusZeroShift.
func TestAssertHoverFocusZeroShiftGuardsRegressions(t *testing.T) {
	t.Run("stable-controls-pass", func(t *testing.T) {
		tab := 0
		day := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
		tt := coreNewTester(func(c *context) {
			coreColumn(c).Padding(16).Gap(8).Children(func() {
				b := coreButton(c, "New chat").Tooltip("New chat (Ctrl+N)")
				_ = b
				coreTabs(c, &tab, "General", "Appearance", "Advanced")
				coreCalendar(c, &day)
			})
		}, 480, 420)
		snap := tt.FlowSnapshot()
		if len(snap) == 0 {
			t.Fatal("FlowSnapshot returned no nodes")
		}
		AssertHoverFocusZeroShift(t, tt)
	})

	t.Run("catches-hover-injected-text", func(t *testing.T) {
		tt := coreNewTester(func(c *context) {
			coreRow(c).Padding(16).Children(func() {
				b := coreButtonBase(c).Width(160).Height(32).Gap(8)
				b.Children(func() {
					coreText(c, "New chat")
					if b.Hovered() {
						coreText(c, "Ctrl+N")
					}
				})
			})
		}, 300, 120)
		var capT captureTestingT
		AssertHoverFocusZeroShift(&capT, tt)
		if len(capT.errors) == 0 {
			t.Fatal("expected AssertHoverFocusZeroShift to catch hover-injected in-flow text")
		}
		joined := strings.Join(capT.errors, "\n")
		if !strings.Contains(joined, "in-flow texts changed") && !strings.Contains(joined, "node count changed") {
			t.Fatalf("unexpected error output: %s", joined)
		}
	})

	t.Run("catches-animated-hover-width-growth", func(t *testing.T) {
		tt := coreNewTester(func(c *context) {
			coreRow(c).Padding(16).Gap(8).Children(func() {
				b := coreButtonBase(c).Height(32).Padding(0, 8)
				w := b.Animate("expand", b2f(b.Hovered()), 200*time.Millisecond) * 80
				b.Children(func() {
					coreText(c, "Git")
					coreBox(c).Width(w).Height(16)
				})
				coreButton(c, "Sibling")
			})
		}, 400, 120)
		var capT captureTestingT
		AssertHoverFocusZeroShift(&capT, tt)
		if len(capT.errors) == 0 {
			t.Fatal("expected AssertHoverFocusZeroShift to catch animated hover width shift")
		}
	})

	t.Run("catches-focus-border-shift", func(t *testing.T) {
		tt := coreNewTester(func(c *context) {
			coreRow(c).Padding(16).Children(func() {
				b := coreButtonBase(c).Size(120, 32).Justify(Start)
				if b.FocusVisible() {
					b.Border(2, RGB(0, 100, 255))
				}
				b.Children(func() { coreText(c, "Focus me") })
			})
		}, 300, 120)
		var capT captureTestingT
		AssertHoverFocusZeroShift(&capT, tt)
		if len(capT.errors) == 0 {
			t.Fatal("expected AssertHoverFocusZeroShift to catch focus border geometry shift")
		}
	})

	t.Run("audits-open-modal-dialog", func(t *testing.T) {
		open := true
		tt := coreNewTester(func(c *context) {
			coreButton(c, "Behind")
			coreModal(c, &open, func() {
				coreColumn(c).Gap(8).Children(func() {
					coreButton(c, "Inside A").Tooltip("Tip A")
					coreButton(c, "Inside B")
				})
			})
		}, 480, 360)
		foundInside := false
		for _, n := range tt.FlowSnapshot() {
			if n.Text == "Inside A" {
				foundInside = true
				break
			}
		}
		if !foundInside {
			t.Fatal("FlowSnapshot did not include open modal dialog content")
		}
		AssertHoverFocusZeroShift(t, tt)
	})

	t.Run("allows-fixed-box-invisible-toggle", func(t *testing.T) {
		tt := coreNewTester(func(c *context) {
			coreRow(c).Width(240).Padding(8).Children(func() {
				b := coreButtonBase(c).FillWidth().Height(32).AlignItems(Center)
				b.Children(func() {
					coreText(c, "Session title").Grow(1)
					stamp := coreRow(c).Shrink(0)
					if b.Hovered() {
						stamp.Invisible()
					}
					stamp.Children(func() { coreText(c, "2m") })
				})
			})
		}, 280, 80)
		AssertHoverFocusZeroShift(t, tt)
	})
}
