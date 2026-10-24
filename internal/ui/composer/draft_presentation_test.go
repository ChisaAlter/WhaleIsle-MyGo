package composer

import (
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func shippedDraftTester(draft *string, client *DraftClient, width int, hero, dark bool) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		if dark {
			c.SetTheme(ui.DarkTheme())
		}
		View(c, Model{Draft: draft, TextInput: client, UseTextInputClient: true, Hero: hero, SystemDark: dark, WorkspaceRoot: "fixture"})
	}, width, 1000)
}

// heroCardTopInset and heroSidePadding describe the card's own hero padding,
// which replaces the InputBar top pad and the input's outer insets. HeroShell.card
// is `padding: 8px 0 0`, so the top inset is the same 8 as the docked card and the
// sides are flush (the writing well carries the 12-DIP text inset).
func heroCardTopInset(hero bool) float32 {
	return 8
}

func heroSidePadding(hero bool) float32 {
	// round242/249: the hero card carries the native writing-well insets
	// (12.67 left + 16.39 right = 29.06), so the draft box is that much narrower.
	if hero {
		return 29.06
	}
	return 0
}

func TestShippedDraftClientInsetsAnd24pxLineAdvance(t *testing.T) {
	for _, hero := range []bool{false, true} {
		draft := "MMMM\nMMMM"
		client := NewDraftClient(&draft)
		tt := shippedDraftTester(&draft, client, 320, hero, false)
		card, _ := tt.Find("Composer card")
		input, _ := tt.Find("Composer draft")
		left, right, top := draftInsets(hero)
		closeMetric(t, input.Y-card.Y, heroCardTopInset(hero), "shipped card top inset")
		closeMetric(t, input.W, card.W-heroSidePadding(hero), "shipped input width")
		closeMetric(t, client.font.Size, 14, "shipped glyph font size")
		closeMetric(t, client.wrapWidth, input.W-left-right, "shipped wrap width")
		first, _, _ := client.BoundsForRange(ui.TextInputRange{Start: 0, End: 0})
		second, _, _ := client.BoundsForRange(ui.TextInputRange{Start: 5, End: 5})
		closeMetric(t, first.X, left, "native caret left inset")
		closeMetric(t, first.Y, top, "native caret top inset")
		closeMetric(t, second.Y-first.Y, 24, "native line advance")
		closeMetric(t, first.H, 24, "native caret line height")
		// Actual ink, rather than only the native client's returned geometry.
		image := tt.Image()
		lineInk := map[int]bool{}
		for y := int(input.Y); y < int(input.Y+input.H); y++ {
			ink := false
			for x := int(input.X + left); x < int(input.X+100); x++ {
				p := image.RGBAAt(x, y)
				if p.R < 60 && p.G < 60 && p.B < 60 {
					ink = true
					break
				}
			}
			if ink {
				line := int((float32(y) - input.Y - top) / 24)
				lineInk[line] = true
			}
		}
		if len(lineInk) != 2 || !lineInk[0] || !lineInk[1] {
			t.Errorf("visible native draft ink lines=%v, want two 24px lines", lineInk)
		}
	}
}

func TestShippedDraftClientGrowsAndClipsAtOriginal336pxCap(t *testing.T) {
	for _, hero := range []bool{false, true} {
		for _, lines := range []int{1, 2, 20} {
			draft := strings.TrimSuffix(strings.Repeat("MMMM\n", lines), "\n")
			client := NewDraftClient(&draft)
			tt := shippedDraftTester(&draft, client, 400, hero, false)
			input, _ := tt.Find("Composer draft")
			minH := float32(36)
			if hero {
				minH = 52
			}
			_, _, top := draftInsets(hero)
			want := max(minH, float32(lines*24)+top)
			want = min(336, want)
			closeMetric(t, input.H, want, "shipped growing draft height")
			image := tt.Image()
			for y := int(input.Y + input.H + 1); y < int(input.Y+input.H+10); y++ {
				for x := int(input.X + 0); x < int(input.X+90); x++ {
					p := image.RGBAAt(x, y)
					if p.R < 60 && p.G < 60 && p.B < 60 {
						t.Fatalf("draft text escaped scrollport at(%d,%d)", x, y)
					}
				}
			}
		}
	}
}

func TestShippedDraftClientUTF16SelectionAndIMEGeometryAfterResize(t *testing.T) {
	draft := "A😀B\n中文 ending"
	client := NewDraftClient(&draft)
	tt := shippedDraftTester(&draft, client, 800, false, false)
	stable := client.Draft
	client.SetSelectionUTF16(0, 4)
	tt.Frame()
	b, actual, ok := client.BoundsForRange(ui.TextInputRange{Start: 0, End: 4})
	if !ok || actual != (ui.TextInputRange{Start: 0, End: 4}) {
		t.Fatalf("selection native range=%+v ok=%v", actual, ok)
	}
	closeMetric(t, b.H, 24, "selected line height")
	input, _ := tt.Find("Composer draft")
	pixel := tt.Image().RGBAAt(int(input.X+b.X+2), int(input.Y+b.Y+1))
	if int(pixel.B) <= int(pixel.R)+20 {
		t.Errorf("selected text highlight absent: %v", pixel)
	}
	for _, width := range []int{320, 800, 320} {
		tt.SetSize(width, 1000)
		input, _ = tt.Find("Composer draft")
		closeMetric(t, client.wrapWidth, input.W-22, "resized native wrap width")
		for _, offset := range []int{0, 1, 3, 5, 6, 7, ui.UTF16Len(draft)} {
			b, _, ok := client.BoundsForRange(ui.TextInputRange{Start: offset, End: offset})
			if !ok {
				t.Fatal("native bounds missing")
			}
			got, ok := client.IndexForPoint(ui.Point{X: b.X + .01, Y: b.Y + b.H/2})
			if !ok || got != offset {
				t.Errorf("width%d UTF16offset%d maps to%d via%v", width, offset, got, b)
			}
		}
		if client.Draft != stable || stable != &draft {
			t.Fatal("presentation replaced stable draft owner")
		}
	}
}

func TestShippedDraftPlaceholderUsesSourceInsetsAndCaption(t *testing.T) {
	for _, hero := range []bool{false, true} {
		for _, dark := range []bool{false, true} {
			draft := ""
			client := NewDraftClient(&draft)
			tt := shippedDraftTester(&draft, client, 320, hero, dark)
			input, _ := tt.Find("Composer draft")
			placeholder, ok := tt.Find("Composer placeholder")
			if !ok {
				t.Fatalf("shipped placeholder missing hero=%v dark=%v", hero, dark)
			}
			left, right, top := draftInsets(hero)
			closeMetric(t, placeholder.X-input.X, left, "placeholder left inset")
			closeMetric(t, placeholder.Y-input.Y, top, "placeholder top inset")
			if placeholder.X+placeholder.W > input.X+input.W-right+.1 {
				t.Errorf("placeholder escaped narrow draft: %v", placeholder)
			}
			label := i18n.T(i18n.KeyComposerPlaceholder)
			if hero {
				label = i18n.T(i18n.KeyComposerPlaceholderHero)
			}
			if !tt.HasText(label) {
				t.Errorf("placeholder source text absent: %q", tt.Texts())
			}
		}
	}
}

func TestShippedDraftNativeCompositionKeepsOwnerAndGeometry(t *testing.T) {
	draft := "A😀B\n"
	client := NewDraftClient(&draft)
	tt := shippedDraftTester(&draft, client, 320, false, false)
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	client.SetSelectionUTF16(ui.UTF16Len(draft), ui.UTF16Len(draft))
	tt.Frame()
	tt.Compose("中文", 1)
	marked, ok := client.MarkedRange()
	if !ok || marked != (ui.TextInputRange{Start: 5, End: 7}) || draft != "A😀B\n中文" {
		t.Fatalf("native marked range=%v ok=%v draft=%q", marked, ok, draft)
	}
	input, _ := tt.Find("Composer draft")
	caret, active := tt.TextCaret()
	if !active {
		t.Fatal("native input method inactive")
	}
	closeMetric(t, caret.H, 24, "native composition caret height")
	closeMetric(t, caret.Y-input.Y, 28, "native composition second line inset")
	tt.Type("中文")
	if _, ok := client.MarkedRange(); ok || draft != "A😀B\n中文" {
		t.Fatalf("native composition commit left marked text: %q", draft)
	}
	if client.Draft != &draft {
		t.Fatal("native composition changed draft owner")
	}
}

func TestShippedDraftScrollClipAndCaretReveal(t *testing.T) {
	draft := strings.TrimSuffix(strings.Repeat("MMMM\n", 20), "\n")
	client := NewDraftClient(&draft)
	tt := shippedDraftTester(&draft, client, 320, false, false)
	client.SetSelectionUTF16(0, 0)
	tt.Frame()
	input, _ := tt.Find("Composer draft")
	closeMetric(t, input.H, 336, "original scroll cap")
	closeMetric(t, client.scrollY, 0, "first line caret reveal")
	tt.Scroll(input.X+30, input.Y+30, 0, 48)
	closeMetric(t, client.scrollY, 48, "native draft wheel scroll")
	third, _, _ := client.BoundsForRange(ui.TextInputRange{Start: 10, End: 10})
	closeMetric(t, third.Y, 4, "scrolled third line native caret")
	index, ok := client.IndexForPoint(ui.Point{X: third.X, Y: third.Y + third.H/2})
	if !ok || index != 10 {
		t.Fatalf("scrolled native hit test=%d ok=%v", index, ok)
	}
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl, ui.KeyEnd)
	end, _, _ := client.BoundsForRange(ui.TextInputRange{Start: ui.UTF16Len(draft), End: ui.UTF16Len(draft)})
	if end.Y < 0 || end.Y+end.H > input.H+.1 {
		t.Errorf("last line caret not revealed: %v viewport=%v", end, input)
	}
	if client.scrollBy(1000) {
		t.Fatal("wheel should bubble at lower scroll boundary")
	}
	tt.Key(ui.Ctrl, ui.KeyHome)
	closeMetric(t, client.scrollY, 0, "Home reveals beginning")
	if client.scrollBy(-1000) {
		t.Fatal("wheel should bubble at upper scroll boundary")
	}
}

func TestComposerRetainsOtherControlFocusForBothInputs(t *testing.T) {
	for _, native := range []bool{false, true} {
		draft, other := "A😀BCDE", "other"
		client := NewDraftClient(&draft)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Children(func() {
				ui.TextInput(c, &other).Label("Outside composer input")
				View(c, Model{Draft: &draft, UseTextInputClient: native, TextInput: client})
			})
		}, 400, 600)
		if err := tt.Click("Outside composer input"); err != nil {
			t.Fatal(err)
		}
		for _, width := range []int{400, 320, 800} {
			tt.SetSize(width, 600)
			if !tt.Focused("Outside composer input") || tt.Focused("Composer draft") {
				t.Errorf("native=%v width=%d composer stole other control focus", native, width)
			}
		}
		tt.Type("Z")
		if draft != "A😀BCDE" || !strings.Contains(other, "Z") {
			t.Errorf("native=%v input went to wrong owner: draft=%q other=%q", native, draft, other)
		}
	}
}

func TestShippedDraftResolvedAndCustomInk(t *testing.T) {
	for _, colors := range []theme.Colors{
		theme.Default().Light, theme.Default().Dark,
		{FG: "#a03050", Caption: "#336699", InputMajor: "#f0f0f0", Border: "#888888", Muted: "#606060", Accent: "#2266aa", ButtonInfoFill: "#4176e6", ButtonFG: "#ffffff", SelectorFill: "#eeeeee"},
	} {
		for _, text := range []string{"", "MMMM\nMMMM"} {
			draft := text
			client := NewDraftClient(&draft)
			tt := ui.NewTester(func(c *ui.Context) {
				viewWithColors(c, Model{Draft: &draft, UseTextInputClient: true, TextInput: client}, colors)
			}, 320, 500)
			input, _ := tt.Find("Composer draft")
			ink := chrome.Color(colors.FG)
			if text == "" {
				ink = chrome.Color(colors.Caption)
			}
			matches := 0
			image := tt.Image()
			for y := int(input.Y + 4); y < int(input.Y+input.H); y++ {
				for x := int(input.X + 14); x < int(input.X+100); x++ {
					p := image.RGBAAt(x, y)
					if absByte(p.R, ink.R) < 20 && absByte(p.G, ink.G) < 20 && absByte(p.B, ink.B) < 20 {
						matches++
					}
				}
			}
			if matches < 3 {
				t.Errorf("native draft text=%q ignores semantic ink %v (%d pixels)", text, ink, matches)
			}
		}
	}
}

func TestComposerFirstMountAutoFocusIsOnceAndDoesNotSteal(t *testing.T) {
	for _, native := range []bool{false, true} {
		draft, other := "hello", "other"
		client := NewDraftClient(&draft)
		show := true
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Children(func() {
				ui.TextInput(c, &other).Label("Outside composer input")
				if show {
					View(c, Model{Draft: &draft, UseTextInputClient: native, TextInput: client})
				}
			})
		}, 400, 600)
		if !tt.Focused("Composer draft") {
			t.Errorf("native=%v first mount did not focus draft", native)
		}
		if err := tt.Click("Outside composer input"); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
		tt.SetSize(320, 600)
		if !tt.Focused("Outside composer input") {
			t.Errorf("native=%v rebuild stole focus", native)
		}
		show = false
		tt.Frame()
		show = true
		tt.Frame()
		if !tt.Focused("Outside composer input") {
			t.Errorf("native=%v remount stole existing focus", native)
		}
	}
}

func TestShippedDraftBlurHidesCaretWithoutChangingSelection(t *testing.T) {
	draft, other := "MMMM", "other"
	client := NewDraftClient(&draft)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Children(func() {
			ui.TextInput(c, &other).Label("Outside composer input")
			View(c, Model{Draft: &draft, UseTextInputClient: true, TextInput: client})
		})
	}, 400, 600)
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	client.SetSelectionUTF16(ui.UTF16Len(draft), ui.UTF16Len(draft))
	tt.Frame()
	selection := client.Selection()
	input, _ := tt.Find("Composer draft")
	b, _, _ := client.BoundsForRange(selection.Range)
	x, y := int(input.X+b.X), int(input.Y+b.Y+1)
	focusedPixel := tt.Image().RGBAAt(x, y)
	if err := tt.Click("Outside composer input"); err != nil {
		t.Fatal(err)
	}
	blurredPixel := tt.Image().RGBAAt(x, y)
	if focusedPixel == blurredPixel {
		t.Errorf("native caret pixel remained after blur: %v", blurredPixel)
	}
	if client.Selection() != selection {
		t.Fatal("blur changed native selection")
	}
	nativeCaret, active := tt.TextCaret()
	if !active || nativeCaret.Y >= input.Y {
		t.Errorf("native input method still targets composer after blur: %v active=%v", nativeCaret, active)
	}
}

func TestShippedEmptyDraftPaintsFocusedCaretBesidePlaceholder(t *testing.T) {
	for _, hero := range []bool{false, true} {
		draft := ""
		client := NewDraftClient(&draft)
		tt := shippedDraftTester(&draft, client, 320, hero, false)
		if !tt.Focused("Composer draft") {
			t.Fatal("empty draft did not receive initial focus")
		}
		input, _ := tt.Find("Composer draft")
		caret, active := tt.TextCaret()
		left, _, top := draftInsets(hero)
		closeMetric(t, caret.X-input.X, left, "empty native caret left inset")
		closeMetric(t, caret.Y-input.Y, top, "empty native caret top inset")
		closeMetric(t, caret.H, 24, "empty native caret height")
		if !active {
			t.Fatal("empty draft native input method inactive")
		}
		// The caret sits at a fractional x now that the hero draft is inset; sample
		// the nearest covered pixel instead of truncating onto its left edge.
		p := tt.Image().RGBAAt(int(math.Ceil(float64(caret.X))), int(caret.Y+1))
		t.Logf("hero=%v input=%v caret=%v pixel=%v", hero, input, caret, p)
		if int(p.B) <= int(p.R)+50 || int(p.B) <= int(p.G)+50 {
			t.Errorf("empty focused caret ink absent: %v", p)
		}
		if _, ok := tt.Find("Composer placeholder"); !ok {
			t.Fatal("caret suppressed placeholder")
		}
	}
}
