package coss

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/design"
)

// These tests drive the components through the toolkit's headless Tester
// with real pointer and key events (no forced states).

func setTheme(c *ui.Context, dark bool) {
	th := *c.Theme()
	th.Dark, th.Accent = dark, galleryAccent[dark]
	c.SetTheme(&th)
}

func pixel(tt *ui.Tester, x, y int) [4]uint8 {
	img := tt.Image()
	o := img.PixOffset(x, y)
	return [4]uint8{img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3]}
}

func TestButtonHoverPressAndClick(t *testing.T) {
	clicks := 0
	tt := ui.NewTester(func(c *ui.Context) {
		setTheme(c, false)
		ui.Box(c).Fill().Background(design.For(c).Background).Padding(20).AlignItems(ui.Start).Children(func() {
			if Button(c, "Save", ButtonOptions{Variant: ButtonOutline}).Clicked() {
				clicks++
			}
		})
	}, 200, 80)
	r, ok := tt.Find("Save")
	if !ok {
		t.Fatal("button not found")
	}
	rest := pixel(tt, 23, 36)
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	hover := pixel(tt, 23, 36)
	if rest == hover {
		t.Fatalf("hover did not change the outline button's fill: %v", rest)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatalf("clicks = %d", clicks)
	}
}

func TestButtonKeyboardFocusRing(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		setTheme(c, false)
		ui.Box(c).Fill().Background(design.For(c).Background).Padding(20).AlignItems(ui.Start).Children(func() {
			Button(c, "Save", ButtonOptions{})
		})
	}, 200, 80)
	before := pixel(tt, 18, 36) // 2 DIPs left of the button: where the ring goes
	tt.Key(0, ui.KeyTab)
	tt.SetTime(time.Now().Add(time.Second))
	tt.Frame()
	after := pixel(tt, 18, 36)
	if before == after {
		t.Fatalf("Tab focus drew no ring: %v", after)
	}
}

func TestDisabledButtonIgnoresClicks(t *testing.T) {
	clicks := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Padding(20).AlignItems(ui.Start).Children(func() {
			if Button(c, "Save", ButtonOptions{Disabled: true}).Clicked() {
				clicks++
			}
		})
	}, 200, 80)
	tt.Click("Save")
	if clicks != 0 {
		t.Fatalf("disabled button clicked %d times", clicks)
	}
}

func TestChecksToggle(t *testing.T) {
	var on, sw bool
	sel := "a"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Gap(8).AlignItems(ui.Start).Children(func() {
			Checkbox(c, &on, "Accept", CheckOptions{})
			Switch(c, &sw, "Wifi", CheckOptions{})
			RadioGroup(c, "Density", func() {
				Radio(c, &sel, "a", "Compact", CheckOptions{})
				Radio(c, &sel, "b", "Roomy", CheckOptions{})
			})
		})
	}, 300, 200)
	for _, l := range []string{"Accept", "Wifi", "Roomy"} {
		if err := tt.Click(l); err != nil {
			t.Fatal(err)
		}
	}
	if !on || !sw || sel != "b" {
		t.Fatalf("checkbox %v switch %v radio %q", on, sw, sel)
	}
}

func TestInputTypes(t *testing.T) {
	v := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			Field(c, FieldOptions{Label: "Name"}, func() { Input(c, &v, InputOptions{Placeholder: "Your name"}) })
		})
	}, 300, 120)
	if !tt.HasText("Your name") {
		t.Fatal("placeholder not shown")
	}
	tt.Click("Your name")
	tt.Type("Ada")
	if v != "Ada" {
		t.Fatalf("value = %q", v)
	}
	if tt.HasText("Your name") {
		t.Fatal("placeholder still shown over text")
	}
}

func TestSelectChooses(t *testing.T) {
	v := "apple"
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).AlignItems(ui.Start).Children(func() {
			Select(c, &v, []SelectOption{{Value: "apple", Label: "Apple"}, {Value: "pear", Label: "Pear"}}, SelectOptions{}).Width(180)
		})
	}, 300, 200)
	if err := tt.Click("Apple"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Pear") {
		t.Fatal("popup did not open")
	}
	if err := tt.Click("Pear"); err != nil {
		t.Fatal(err)
	}
	if v != "pear" {
		t.Fatalf("value = %q", v)
	}
}

func TestMenuKeyboard(t *testing.T) {
	open := false
	chosen := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).AlignItems(ui.Start).Children(func() {
			b := Button(c, "Actions", ButtonOptions{Variant: ButtonOutline})
			if id := Menu(c, b, &open, []MenuItem{{ID: "copy", Label: "Copy"}, {ID: "paste", Label: "Paste"}}, MenuOptions{}); id != "" {
				chosen = id
			}
		})
	}, 300, 240)
	tt.Click("Actions")
	if !open || !tt.HasText("Paste") {
		t.Fatal("menu did not open")
	}
	tt.Click("Paste")
	if chosen != "paste" || open {
		t.Fatalf("chosen %q open %v", chosen, open)
	}
}

func TestDialogEscapeCloses(t *testing.T) {
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			Dialog(c, &open, DialogOptions{Title: "Edit profile", Footer: func() { Button(c, "Save", ButtonOptions{}) }}, nil)
		})
	}, 700, 400)
	if !tt.HasText("Edit profile") {
		t.Fatal("dialog not shown")
	}
	tt.Key(0, ui.KeyEscape)
	if open {
		t.Fatal("Escape did not close the dialog")
	}
}

func TestAlertDialogKeepsEscapeRejectsBackdrop(t *testing.T) {
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			AlertDialog(c, &open, DialogOptions{Title: "Delete?"}, nil)
		})
	}, 700, 400)
	tt.ClickAt(10, 10)
	if !open {
		t.Fatal("a backdrop click dismissed the alert dialog")
	}
	tt.Key(0, ui.KeyEscape)
	if open {
		t.Fatal("Escape did not close the alert dialog")
	}
}

func TestComposerSubmitsOnEnter(t *testing.T) {
	v := ""
	submitted := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			if Composer(c, &v, ComposerOptions{Placeholder: "Ask anything"}).Submit {
				submitted++
				v = ""
			}
		})
	}, 400, 200)
	tt.Click("Ask anything")
	tt.Type("hello")
	tt.Key(0, ui.KeyEnter)
	if submitted != 1 || v != "" {
		t.Fatalf("submitted %d, value %q", submitted, v)
	}
}

func TestComposerMentionPicks(t *testing.T) {
	v := ""
	var picked string
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20, 20, 20, 20).Justify(ui.End).Children(func() {
			ev := Composer(c, &v, ComposerOptions{Placeholder: "Ask", Triggers: "@/", Suggest: func(tr byte, q string) []ComposerSuggestion {
				return []ComposerSuggestion{{ID: "ada", Label: "ada"}, {ID: "alan", Label: "alan"}}
			}})
			if ev.Picked != nil {
				picked = ev.Picked.ID
			}
		})
	}, 400, 300)
	tt.Click("Ask")
	tt.Type("@a")
	if !tt.HasText("alan") {
		t.Fatal("suggestions did not open")
	}
	tt.Key(0, ui.KeyDown)
	tt.Key(0, ui.KeyEnter)
	if picked != "alan" || v != "@alan " {
		t.Fatalf("picked %q value %q", picked, v)
	}
}

func TestCommandPaletteFiltersAndRuns(t *testing.T) {
	open := true
	q := ""
	chosen := ""
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			if id := CommandPalette(c, &open, &q, []CommandItem{{ID: "new", Label: "New chat"}, {ID: "set", Label: "Settings"}}, CommandOptions{Placeholder: "Search"}); id != "" {
				chosen = id
			}
		})
	}, 800, 500)
	tt.Type("sett")
	if tt.HasText("New chat") {
		t.Fatal("filter kept a non-matching command")
	}
	tt.Key(0, ui.KeyEnter)
	if chosen != "set" || open {
		t.Fatalf("chosen %q open %v", chosen, open)
	}
}

func TestTabsArrowKeys(t *testing.T) {
	sel := 0
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).AlignItems(ui.Start).Children(func() {
			Tabs(c, &sel, []string{"One", "Two", "Three"}, TabsOptions{})
		})
	}, 400, 100)
	tt.Click("One")
	tt.Key(0, ui.KeyRight)
	if sel != 1 {
		t.Fatalf("selected %d", sel)
	}
}

func TestTokensFollowThemeAccent(t *testing.T) {
	var got *design.Tokens
	ui.NewTester(func(c *ui.Context) {
		setTheme(c, true)
		th := *c.Theme()
		th.AccentText = ui.Hex("#0f1115")
		c.SetTheme(&th)
		got = design.For(c)
	}, 10, 10)
	if got.Primary != galleryAccent[true] || got.PrimaryForeground != ui.Hex("#0f1115") {
		t.Fatalf("primary %v fg %v", got.Primary, got.PrimaryForeground)
	}
	// An empty on-accent foreground falls back to the computed contrast pick.
	if fallback := design.Resolve(design.Dark, galleryAccent[true], ui.Color{}); design.Contrast(fallback.PrimaryForeground, fallback.Primary) < 4.5 {
		t.Fatalf("fallback fg %v", fallback.PrimaryForeground)
	}
}

func TestScrollAreaScrollbarDragAndGutter(t *testing.T) {
	var st ui.ScrollState
	var rowBounds ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			ScrollArea(c, ScrollAreaOptions{State: &st}, func() {
				for i := 0; i < 20; i++ {
					row := ui.Row(c.Key(i)).FillWidth().Height(24)
					if i == 0 {
						rowBounds = row.Bounds()
					}
					row.Children(func() {
						ui.Text(c, "Item")
					})
				}
			}).Size(200, 120)
		})
	}, 220, 140)
	tt.Frame()
	tt.Frame()
	if st.MaxY <= 0 {
		t.Fatalf("expected scrollable overflow, got MaxY=%v", st.MaxY)
	}
	if rowBounds.X+rowBounds.W > 200-scrollAreaGutter+0.5 {
		t.Fatalf("expected trailing gutter of %v DIP, row right=%v", scrollAreaGutter, rowBounds.X+rowBounds.W)
	}
	// Hover then drag the native vertical scrollbar thumb near the right edge.
	tt.Move(194, 15)
	tt.Press(194, 15)
	tt.Move(194, 90)
	tt.Release(194, 90)
	if st.Y <= 0 {
		t.Fatalf("expected dragging scrollbar thumb to scroll vertically, got Y=%v", st.Y)
	}
}

func TestSelectFixedWidthDoesNotTruncateLabel(t *testing.T) {
	selected88 := "all"
	selected180 := "catalog"
	selectedAuto := "a"
	var rect88, rect180, rectAuto ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Gap(12).AlignItems(ui.Start).Children(func() {
			Select(c, &selected88, []SelectOption{
				{Value: "all", Label: "全部"},
				{Value: "on", Label: "已启用"},
			}, SelectOptions{Size: InputSM, Label: "MCP filter"}).Width(88).ObserveLayout(func(r ui.Rect) {
				rect88 = r
			})
			Select(c, &selected180, []SelectOption{
				{Value: "catalog", Label: "目录顺序"},
				{Value: "recent", Label: "最近更新"},
			}, SelectOptions{Label: "Sort"}).Width(180).ObserveLayout(func(r ui.Rect) {
				rect180 = r
			})
			Select(c, &selectedAuto, []SelectOption{
				{Value: "a", Label: "All"},
			}, SelectOptions{Label: "Auto"}).ObserveLayout(func(r ui.Rect) {
				rectAuto = r
			})
		})
	}, 400, 260)

	font := ui.Font{Family: ui.LightTheme().Font, Size: design.TextSM.Size, Weight: design.WeightNormal}
	for _, tc := range []struct {
		name  string
		label string
		rect  ui.Rect
		wantW float32
	}{
		{"88px-mcp-filter", "全部", rect88, 88},
		{"180px-market-sort", "目录顺序", rect180, 180},
	} {
		if tc.rect.W < tc.wantW-0.5 || tc.rect.W > tc.wantW+0.5 {
			t.Fatalf("%s: trigger width = %v, want %v", tc.name, tc.rect.W, tc.wantW)
		}
		textRect, ok := tt.Find(tc.label)
		if !ok {
			t.Fatalf("%s: label %q not found", tc.name, tc.label)
		}
		naturalW, _ := ui.ShapeTextLine(tc.label, font, 1e6).Size()
		if textRect.W+0.5 < naturalW {
			t.Fatalf("%s: label %q squeezed to %.2f DIP (natural=%.2f DIP), would truncate to ellipsis",
				tc.name, tc.label, textRect.W, naturalW)
		}
	}
	if rectAuto.W < 143.5 {
		t.Fatalf("unsized select width = %v, want default floor >= 144", rectAuto.W)
	}
}

func TestSelectInsideDialogKeepsFocusAcrossAllDismissPaths(t *testing.T) {
	open := true
	search := ""
	val := "tray"
	var trigRect, blankRect ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			Dialog(c, &open, DialogOptions{Title: "Settings"}, func() {
				ui.Column(c).FillWidth().Gap(12).Children(func() {
					Input(c, &search, InputOptions{Label: "Search settings", Placeholder: "Search settings"})
					Select(c, &val, []SelectOption{
						{Value: "tray", Label: "Minimize to tray"},
						{Value: "quit", Label: "Quit app"},
					}, SelectOptions{Label: "On close"}).Width(180).ObserveLayout(func(r ui.Rect) {
						trigRect = r
					})
					ui.Box(c).FillWidth().Height(60).ObserveLayout(func(r ui.Rect) {
						blankRect = r
					})
				})
			})
		})
	}, 720, 480)
	// Settle the dialog's opening position/reveal transition deterministically so
	// pointer clicks are not racing the 200ms entrance animation on fast hosts.
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	tt.Frame()

	// Open then click trigger again.
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(trigRect.X+trigRect.W/2, trigRect.Y+trigRect.H/2)
	if tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("re-click trigger hijacked focus: search=%v trigger=%v", tt.Focused("Search settings"), tt.Focused("On close"))
	}

	// Open, pick Quit app, then round-trip back to Minimize to tray.
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Quit app"); err != nil {
		t.Fatal(err)
	}
	if val != "quit" || tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("pick quit failed: val=%q search=%v trigger=%v", val, tt.Focused("Search settings"), tt.Focused("On close"))
	}
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Minimize to tray"); err != nil {
		t.Fatal(err)
	}
	if val != "tray" || tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("round-trip pick tray failed: val=%q search=%v trigger=%v", val, tt.Focused("Search settings"), tt.Focused("On close"))
	}

	// Open then click dialog whitespace backdrop.
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(blankRect.X+blankRect.W/2, blankRect.Y+blankRect.H/2)
	if tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("backdrop dismiss hijacked focus: search=%v trigger=%v", tt.Focused("Search settings"), tt.Focused("On close"))
	}

	// Open then Escape.
	if err := tt.Click("On close"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyEscape)
	if !open || tt.Focused("Search settings") || !tt.Focused("On close") {
		t.Fatalf("Escape dismiss failed: open=%v search=%v trigger=%v", open, tt.Focused("Search settings"), tt.Focused("On close"))
	}
}

func TestScrollAreaFirstFrameGutterAndZeroHoverShift(t *testing.T) {
	var st ui.ScrollState
	var rowBounds ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			ScrollArea(c, ScrollAreaOptions{State: &st}, func() {
				for i := 0; i < 20; i++ {
					row := ui.Row(c.Key(i)).FillWidth().Height(24)
					if i == 0 {
						row.ObserveLayout(func(r ui.Rect) { rowBounds = r })
					}
					row.Children(func() {
						Button(c, "Row action", ButtonOptions{Variant: ButtonGhost, Size: ButtonXS})
					})
				}
			}).Size(200, 120)
		})
	}, 220, 140)
	// Without any extra manual tt.Frame() or hover, gutter must already settle.
	if st.MaxY <= 0 {
		t.Fatalf("expected scrollable overflow on initial settle, got MaxY=%v", st.MaxY)
	}
	if rowBounds.X+rowBounds.W > 200-scrollAreaGutter+0.5 {
		t.Fatalf("first-frame gutter missing: row right=%v want <= %v", rowBounds.X+rowBounds.W, 200-scrollAreaGutter)
	}
	ui.AssertHoverFocusZeroShift(t, tt)
}

func TestSheetDefaultSideWithKeyAndMaxWidth(t *testing.T) {
	open := true
	var b ui.Rect
	ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			Sheet(c, &open, DialogOptions{
				Key:      "details-sheet",
				MaxWidth: 360,
				Title:    "Details",
			}, func() {
				ui.Text(c, "Body")
			}).ObserveLayout(func(r ui.Rect) { b = r })
		})
	}, 800, 500)
	if b.W < 359 || b.W > 361 || b.H < 499 || b.X < 400 {
		t.Fatalf("Sheet with Key+MaxWidth fell back from right edge: bounds=%v", b)
	}

	// Explicit SideBottom still docks to the bottom edge.
	openBottom := true
	var bb ui.Rect
	ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			Sheet(c, &openBottom, DialogOptions{
				Key:   "bottom-sheet",
				Side:  SideBottom,
				Title: "Bottom",
			}, func() {
				ui.Text(c, "Body")
			}).ObserveLayout(func(r ui.Rect) { bb = r })
		})
	}, 800, 500)
	if bb.W < 799 || bb.Y <= 0 || bb.H >= 499 {
		t.Fatalf("explicit SideBottom sheet bounds=%v", bb)
	}
}

func TestDialogDismissDisabledDisablesCloseButton(t *testing.T) {
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			Dialog(c, &open, DialogOptions{
				Title:           "Busy operation",
				DismissDisabled: true,
			}, func() {
				ui.Text(c, "Working...")
			})
		})
	}, 700, 400)
	for _, box := range tt.Interactives() {
		if box.Label == "Close" {
			t.Fatalf("Close button remained enabled while DismissDisabled=true: %v", box)
		}
	}
	if err := tt.Click("Close"); err != nil {
		t.Fatal(err)
	}
	if !open {
		t.Fatal("Close button dismissed dialog while DismissDisabled=true")
	}
}

func TestComboboxExternalSyncLocaleChangeAndBlurRevert(t *testing.T) {
	selected := "apple"
	opts := []SelectOption{
		{Value: "apple", Label: "Apple"},
		{Value: "pear", Label: "Pear"},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
			Button(c, "Outside", ButtonOptions{})
			Combobox(c, &selected, opts, ComboboxOptions{Label: "Fruit"})
		})
	}, 360, 240)

	readInput := func() string {
		t.Helper()
		if err := tt.Click("Fruit"); err != nil {
			t.Fatal(err)
		}
		tt.SetClipboard("")
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Key(ui.Cmd, ui.KeyC)
		if err := tt.Click("Outside"); err != nil {
			t.Fatal(err)
		}
		return tt.Clipboard()
	}

	if got := readInput(); got != "Apple" {
		t.Fatalf("initial Combobox text = %q, want Apple", got)
	}

	// External update to *selected syncs displayed text while unfocused.
	selected = "pear"
	tt.Frame()
	if got := readInput(); got != "Pear" {
		t.Fatalf("Combobox did not sync external *selected change: got %q, want Pear", got)
	}

	// Locale change updates option labels; Combobox updates displayed text.
	opts = []SelectOption{
		{Value: "apple", Label: "苹果"},
		{Value: "pear", Label: "梨"},
	}
	tt.Frame()
	if got := readInput(); got != "梨" {
		t.Fatalf("Combobox did not sync option label/locale change: got %q, want 梨", got)
	}

	// Invalid user text reverts to the active selection on blur.
	if err := tt.Click("Fruit"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("invalid-fruit")
	if err := tt.Click("Outside"); err != nil {
		t.Fatal(err)
	}
	if selected != "pear" {
		t.Fatalf("invalid typed text mutated selected to %q", selected)
	}
	if got := readInput(); got != "梨" {
		t.Fatalf("Combobox did not revert invalid text on blur: got %q, want 梨", got)
	}
}
