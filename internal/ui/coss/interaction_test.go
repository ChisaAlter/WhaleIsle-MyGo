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

func TestAlertDialogIgnoresEscape(t *testing.T) {
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Children(func() {
			AlertDialog(c, &open, DialogOptions{Title: "Delete?"}, nil)
		})
	}, 700, 400)
	tt.Key(0, ui.KeyEscape)
	if !open {
		t.Fatal("alert dialog dismissed by Escape")
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
		got = design.For(c)
	}, 10, 10)
	if got.Primary != galleryAccent[true] || design.Contrast(got.PrimaryForeground, got.Primary) < 4.5 {
		t.Fatalf("primary %v fg %v", got.Primary, got.PrimaryForeground)
	}
}
