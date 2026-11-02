package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// Keep the initial red test buildable before the generated facade or setter
// exists. The missing method is a runtime contract failure, not a compile error.
func setInertForTest(t *testing.T, element *node, on bool) *node {
	t.Helper()
	setter, ok := any(element).(interface{ Inert(bool) *node })
	if !ok {
		t.Fatal("node is missing the generic Inert(bool) setter")
	}
	if result := setter.Inert(on); result != element {
		t.Fatal("Inert did not preserve the fluent element identity")
	}
	return element
}

func TestInertAccessibilityHidesSubtreeAndRestores(t *testing.T) {
	hidden := true
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			run := coreColumn(c).Key("retained-run").Size(160, 80).Label("Retained run")
			// Last write wins on one node; a descendant's false cannot opt out
			// of an inert ancestor.
			setInertForTest(t, run, true)
			setInertForTest(t, run, hidden).Children(func() {
				child := coreColumn(c).Fill().Label("Retained child")
				setInertForTest(t, child, false).Children(func() {
					coreText(c, "Outgoing words")
					coreButtonBase(c).Size(100, 28).Label("Outgoing action")
				})
			})
			coreBox(c).Role(RoleNone).Children(func() { coreText(c, "Presentational child stays") })
			coreButtonBase(c).Size(120, 28).Label("Live sibling")
		})
	}, 300, 220)
	t.Cleanup(tt.rt.close)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	for _, n := range tt.h.access.Nodes {
		switch n.Label {
		case "Retained run", "Retained child", "Outgoing words", "Outgoing action":
			t.Fatalf("inert subtree leaked into the accessibility tree: %+v", n)
		}
	}
	accessNode(t, tt.h.access, platform.RoleText, "Presentational child stays")
	accessNode(t, tt.h.access, platform.RoleButton, "Live sibling")
	if tt.HasText("Outgoing words") || tt.HasText("Outgoing action") {
		t.Fatal("inert descendants remained in semantic test labels")
	}
	hidden = false
	tt.Frame()
	accessNode(t, tt.h.access, platform.RoleGroup, "Retained run")
	accessNode(t, tt.h.access, platform.RoleGroup, "Retained child")
	accessNode(t, tt.h.access, platform.RoleText, "Outgoing words")
	accessNode(t, tt.h.access, platform.RoleButton, "Outgoing action")
}

func TestInertRetainsPaintAndLayoutWithoutPointerOrTabTarget(t *testing.T) {
	inert, clicks := false, 0
	var childID uint64
	tt := coreNewTester(func(c *context) {
		coreRow(c).Fill().Children(func() {
			run := coreBox(c).Key("paint-run").Size(80, 60).Background(RGB(220, 40, 20))
			setInertForTest(t, run, inert).Children(func() {
				button := coreButtonBase(c).Fill().Label("Retained paint action")
				childID = button.id
				if button.Clicked() {
					clicks++
				}
			})
			coreBox(c).Size(20, 60).Label("Following box")
			coreButtonBase(c).Size(80, 60).Label("Available action")
		})
	}, 220, 100)
	t.Cleanup(tt.rt.close)
	before, ok := tt.Find("Following box")
	if !ok || before.X != 80 {
		t.Fatalf("unexpected following geometry: %+v", before)
	}
	paint := tt.Image().RGBAAt(10, 50)
	if paint.R != 220 || paint.G != 40 || paint.B != 20 {
		t.Fatalf("fixture did not paint retained content: %+v", paint)
	}
	inert = true
	tt.Frame()
	after, ok := tt.Find("Following box")
	if !ok || before != after || tt.Image().RGBAAt(10, 50) != paint {
		t.Fatal("inert content changed its layout or disappeared from paint")
	}
	for _, id := range tt.rt.focusOrder {
		if id == childID {
			t.Fatal("inert descendant remained a Tab target")
		}
	}
	tt.ClickAt(30, 30)
	if clicks != 0 {
		t.Fatal("pointer activated an inert descendant")
	}
	tt.Key(0, KeyTab)
	live := accessNodeAfterEnable(t, tt, platform.RoleButton, "Available action")
	if tt.rt.focused != live.ID {
		t.Fatalf("Tab did not skip retained content: got%d want%d", tt.rt.focused, live.ID)
	}
	inert = false
	tt.Frame()
	if err := tt.Click("Retained paint action"); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatalf("clearing inert did not restore pointer input: %d", clicks)
	}
}

func accessNodeAfterEnable(t *testing.T, tt *Tester, role platform.AccessRole, label string) platform.AccessNode {
	t.Helper()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	return accessNode(t, tt.h.access, role, label)
}

func TestInertRetiresPreviouslyFocusedDescendant(t *testing.T) {
	inert, clicks := false, 0
	tt := coreNewTester(func(c *context) {
		run := coreColumn(c).Fill()
		setInertForTest(t, run, inert).Children(func() {
			if coreButtonBase(c).Size(120, 32).Label("Focus before exit").Clicked() {
				clicks++
			}
		})
	}, 180, 90)
	t.Cleanup(tt.rt.close)
	if err := tt.Click("Focus before exit"); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 || tt.rt.focused == 0 {
		t.Fatal("fixture did not focus and activate the control")
	}
	inert = true
	tt.Frame()
	if tt.rt.focused != 0 {
		t.Fatal("inert subtree retained an existing keyboard focus")
	}
	tt.Key(0, KeyEnter)
	tt.Key(0, KeySpace)
	if clicks != 1 {
		t.Fatal("keyboard reactivated an outgoing control")
	}
}

func TestDisabledInertCompositionRejectsCachedActionsAndPendingInput(t *testing.T) {
	for _, pending := range []string{"none", "button-press", "field-focus"} {
		t.Run(pending, func(t *testing.T) {
			outgoing, clicks := false, 0
			value := "kept"
			tt := coreNewTester(func(c *context) {
				run := coreColumn(c).Size(180, 90).Disabled(outgoing)
				setInertForTest(t, run, outgoing).Children(func() {
					if coreButtonBase(c).Size(150, 30).Label("Exit action").Clicked() {
						clicks++
					}
					coreTextInput(c, &value).Size(150, 30).Label("Exit editor")
				})
			}, 240, 140)
			t.Cleanup(tt.rt.close)
			tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
			button := accessNode(t, tt.h.access, platform.RoleButton, "Exit action")
			field := accessNode(t, tt.h.access, platform.RoleTextField, "Exit editor")
			if pending == "button-press" {
				tt.Press(30, 15)
			}
			if pending == "field-focus" {
				if err := tt.Click("Exit editor"); err != nil {
					t.Fatal(err)
				}
			}
			outgoing = true
			tt.Frame()
			if tt.rt.focused != 0 {
				t.Fatal("outgoing input retained focus")
			}
			if len(tt.h.access.Nodes) != 0 {
				t.Fatalf("outgoing accessibility subtree survived: %+v", tt.h.access.Nodes)
			}
			// These are cached in-memory IDs, not OS accessibility requests.
			for _, action := range []platform.AccessActionKind{platform.AccessPress, platform.AccessFocus, platform.AccessIncrement, platform.AccessExpand} {
				tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: button.ID, Action: action})
			}
			tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "must not apply"})
			if pending == "button-press" {
				tt.Release(30, 15)
			}
			tt.ClickAt(30, 15)
			tt.Key(0, KeyEnter)
			tt.Key(0, KeySpace)
			tt.Type("must not type")
			if clicks != 0 || value != "kept" || tt.rt.focused != 0 {
				t.Fatalf("outgoing inputs applied: clicks%d value%q focus%d", clicks, value, tt.rt.focused)
			}
			outgoing = false
			tt.Frame()
			accessNode(t, tt.h.access, platform.RoleButton, "Exit action")
			if err := tt.Click("Exit action"); err != nil {
				t.Fatal(err)
			}
			if clicks != 1 {
				t.Fatal("reopening did not restore the original action")
			}
		})
	}
}
