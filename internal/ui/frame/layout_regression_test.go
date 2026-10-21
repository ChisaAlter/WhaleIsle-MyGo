package frame

import (
	"fmt"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestCurrentColumnsContract(t *testing.T) {
	for _, tc := range []struct {
		name            string
		viewport, right float32
		want            Columns
	}{
		{"1200 concedes right", 1200, 540, Columns{280, 400, 520}},
		{"1600 first-open ratio", 1600, 720, Columns{280, 600, 720}},
		{"900 closes right", 900, 540, Columns{280, 620, 0}},
		{"3000 viewport-relative maximum", 3000, 9999, Columns{280, 620, 2100}},
		{"minimum right", 1920, 1, Columns{280, 1340, 300}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ComputeColumns(tc.viewport, 280, tc.right); got != tc.want {
				t.Fatalf("columns=%+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestRenderedColumnsResolveCenterAndSplitterSpace(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	sizes.Right = 540
	right := float32(540)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right, RightPaneOpen: true})
	}, 1200, 800)
	if sizes.Center != 400 || right != 518 {
		t.Fatalf("rendered center=%v right=%v, want center400 right518 after 2 divider DIPs", sizes.Center, right)
	}
	r, ok := tt.Find(i18n.T(i18n.KeyRightPaneFiles))
	if !ok {
		t.Fatal("missing files pane")
	}
	if r.X < 682 {
		t.Fatalf("files tab bounds=%+v, right track must start at682", r)
	}
}

func TestTerminalStaysInsideCenterAndSidebarKeepsFullHeight(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	sizes := DefaultLayout()
	sizes.Center = 600
	sizes.Main = 560
	sizes.Right = 720
	right := float32(720)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right, RightPaneOpen: true, OnOpenSettings: func() {}})
	}, 1600, 800)
	term, ok := tt.Find(i18n.T(i18n.KeyTerminalUnavailable))
	if !ok {
		t.Fatal("unconnected terminal drawer missing")
	}
	if term.X < 281 || term.X+term.W > 880 {
		t.Fatalf("terminal bounds=%+v, want inside center x281..880", term)
	}
	settings, ok := tt.Find(i18n.T(i18n.KeySettingsAccountSeat))
	if !ok {
		t.Fatal("sidebar settings missing")
	}
	// The merged settings/account seat must still reach the viewport's foot
	// with the terminal drawer open, which proves the sidebar keeps its height.
	if settings.Y+settings.H < 788 {
		t.Fatalf("sidebar bottom seat=%+v, terminal must not shorten sidebar", settings)
	}
}

func TestFrameBackgroundUsesThemeBG(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: session.New(), Sizes: &sizes, ThemeID: "deepseek.light"}) }, 1200, 800)
	want := chrome.Color(theme.Resolve("deepseek.light").Light.BG)
	got := tt.Image().RGBAAt(700, 5)
	if got.R != want.R || got.G != want.G || got.B != want.B {
		t.Fatalf("center background=%+v, want theme BG=%+v", got, want)
	}
}

func TestViewRemembersRightPreferenceAcrossResizeAndExplicitClose(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	sizes.Right = 540
	right := float32(0)
	open := true
	st := session.New()
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: st, Sizes: &sizes, RightPaneOpen: open, FilePaneWidth: &right})
	}, 1200, 800)
	check := func(center, details float32) {
		t.Helper()
		if sizes.Center != center || right != details || sizes.Right != 540 {
			t.Fatalf("center=%v details=%v preference=%v, want %v %v 540", sizes.Center, right, sizes.Right, center, details)
		}
	}
	check(400, 518)
	tt.SetSize(900, 800)
	// The sidebar keeps its 280-DIP track at 900 DIP (the auto-collapse band is the
	// phone drawer), so the right pane concedes entirely; the preference survives.
	check(619, 0)
	tt.SetSize(1600, 800)
	check(778, 540)
	open = false
	tt.Frame()
	check(1319, 0)
	open = true
	tt.Frame()
	check(778, 540)
}

func TestViewUsesFirstOpenViewportRatioWithoutRightWidthOutput(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: true}) }, 1600, 800)
	if sizes.Right != 720 || sizes.Center != 598 {
		t.Fatalf("sizes=%+v, want first-open720 center598 after dividers", sizes)
	}
	if !tt.HasText(i18n.T(i18n.KeyRightPaneFiles)) {
		t.Fatal("right-open flag must render track even without output pointer")
	}
}

func TestTerminalRemainsVisibleAfterHeightShrink(t *testing.T) {
	sizes := DefaultLayout()
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: session.New(), Sizes: &sizes, OnOpenSettings: func() {}}) }, 1200, 800)
	tt.SetSize(1200, 400)
	r, ok := tt.Find(i18n.T(i18n.KeyTerminalUnavailable))
	if !ok {
		t.Fatal("terminal missing after height shrink")
	}
	if r.Y+r.H > 400 {
		t.Fatalf("terminal header %+v is below resized center", r)
	}
}

func TestRightDragChangesPreferenceAndSurvivesReopen(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	sizes.Right = 540
	right := float32(0)
	open := true
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: open, FilePaneWidth: &right})
	}, 1200, 800)
	// Grab the concession-clamped 518-DIP track at the center divider.
	tt.Press(681, 400)
	tt.Move(731, 400)
	tt.Release(731, 400)
	if sizes.Right != 468 || right != 468 || sizes.Center != 450 {
		t.Fatalf("drag sizes=%+v right=%v, want right468 center450", sizes, right)
	}
	tt.SetSize(900, 800)
	// 900 DIP keeps the 280-DIP sidebar track, so the right pane is squeezed out
	// while the preference it will reopen at stays 468.
	if sizes.Right != 468 || right != 0 || sizes.Center != 619 {
		t.Fatalf("shrink sizes=%+v right=%v", sizes, right)
	}
	tt.SetSize(1600, 800)
	if sizes.Right != 468 || right != 468 || sizes.Center != 850 {
		t.Fatalf("reexpand sizes=%+v right=%v", sizes, right)
	}
	open = false
	tt.Frame()
	open = true
	tt.Frame()
	if right != 468 || sizes.Right != 468 {
		t.Fatalf("reopen sizes=%+v right=%v", sizes, right)
	}
}

func TestTerminalHeightConcessionDoesNotRewritePreference(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Main = 560
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { ui.Box(c).Height(36).FillWidth(); View(c, Model{Store: session.New(), Sizes: &sizes}) })
	}, 1200, 800)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	tt.Frame()
	tt.SetSize(1200, 400)
	if sizes.Main != 560 {
		t.Fatalf("height concession rewrote top-height preference: %v", sizes.Main)
	}
	r, _ := tt.Find(i18n.T(i18n.KeyTerminalUnavailable))
	if r.Y+r.H > 400 || r.H == 0 {
		t.Fatalf("shrunk terminal header=%+v", r)
	}
	tt.SetSize(1200, 800)
	r, _ = tt.Find(i18n.T(i18n.KeyTerminalUnavailable))
	// Label bounds include their padding: outer36 + caption48 + chat560 + divider1.
	if r.Y != 645 {
		t.Fatalf("restored terminal status=%+v, want y645 below the caption row", r)
	}
}

func TestConversationContentAxisStaysRawOnNarrowColumns(t *testing.T) {
	for _, tc := range []struct{ column, want float32 }{{400, 680}, {500, 680}, {1200, 768}, {2000, 920}} {
		if got := conversationContentWidth(tc.column); got != tc.want {
			t.Fatalf("raw axis(%v)=%v, want %v; actual envelope must clamp separately", tc.column, got, tc.want)
		}
	}
}

func TestHeroComposerCardFitsCenterWithOriginalClearance(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	sizes.Right = 540
	draft := "hello"
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: true, Draft: &draft})
	}, 1200, 800)
	card, ok := tt.Find("Composer card")
	if !ok {
		t.Fatal("composer card missing")
	}
	// Both seats clear16 per side inside the stable2+5 DIP scroll viewport.
	if card.X != 297 || card.W != 361 {
		t.Fatalf("hero card=%+v, want x297 width361", card)
	}
}

func TestPopulatedComposerCardKeepsSharedAxisAndFitsCenter(t *testing.T) {
	for _, viewport := range []int{1200, 1600, 2200} {
		t.Run(fmt.Sprint(viewport), func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom = 0
			sizes.Right = 540
			right := float32(0)
			draft := "hello"
			snap := session.ViewState{Messages: []session.Message{{ID: "layout-user", Seq: 1, Kind: session.MessageKindText, Role: "user", Body: "short user fixture"}, {ID: "layout-assistant", Seq: 2, Kind: session.MessageKindText, Role: "assistant", Body: "assistant fixture"}}}
			tt := ui.NewTester(func(c *ui.Context) {
				cols := sizes.ResolveColumns(float32(viewport), true)
				right = cols.Details
				colors := theme.Resolve("deepseek.light").Light
				m := Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right, RightPaneOpen: true, Draft: &draft}
				ui.Column(c).Fill().Children(func() { mainRow(c, m, snap, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted)) })
			}, viewport, 800)
			card, ok := tt.Find("Composer card")
			if !ok {
				t.Fatal("composer missing")
			}
			want := min(conversationContentWidth(sizes.Center)+32, sizes.Center-conversationScrollGutter-32)
			if math.Abs(float64(card.W-want)) > 0.01 {
				t.Fatalf("card=%+v want width%v for center%v", card, want, sizes.Center)
			}
			if card.X < 281+16 || card.X+card.W > 281+sizes.Center-conversationScrollGutter-16 {
				t.Fatalf("card=%+v exceeds center%v clearance", card, sizes.Center)
			}
		})
	}
}

func TestRightDragExcursionReturnsToFrozenRenderedBase(t *testing.T) {
	sizes := DefaultLayout()
	sizes.Bottom = 0
	sizes.Right = 540
	right := float32(0)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: session.New(), Sizes: &sizes, RightPaneOpen: true, FilePaneWidth: &right})
	}, 1200, 800)
	tt.Press(681, 400)
	for _, x := range []float32{661, 641, 621, 641, 661, 681} {
		tt.Move(x, 400)
	}
	tt.Release(681, 400)
	if sizes.Right != 518 || sizes.Center != 400 || right != 518 {
		t.Fatalf("returned gesture sizes=%+v right=%v, want frozen rendered right518 center400", sizes, right)
	}
}

func TestTerminalTogglePreservesComposerFocusAndSelection(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprint(native), func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom = 0
			draft := "ABCDE"
			client := composer.NewDraftClient(&draft)
			// An unselected Hero is intentionally inert until a workspace/session
			// is chosen. Exercise the editable active conversation, retaining its
			// Store owner across every terminal/viewport frame.
			store := session.New()
			store.Select("terminal-focus-session")
			store.AppendStreamDelta("user", "Owned active conversation fixture")
			rightOpen := true
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Store: store, Sizes: &sizes, Draft: &draft, UseComposerTextInput: native, ComposerTextInput: client, RightPaneOpen: rightOpen})
			}, 1200, 800)
			if err := tt.Click("Composer draft"); err != nil {
				t.Fatal(err)
			}
			if !tt.Focused("Composer draft") {
				t.Fatal("editable composer never acquired initial focus")
			}
			if native {
				client.SetSelectionUTF16(1, 4)
				tt.Frame()
			} else {
				tt.Key(ui.Cmd, ui.KeyA)
			}
			if !tt.Focused("Composer draft") {
				t.Fatal("selection setup lost composer focus before terminal toggle")
			}
			for _, bottom := range []float32{180, 0, 180, 0} {
				sizes.Bottom = bottom
				tt.Frame()
				if !tt.Focused("Composer draft") {
					t.Fatalf("terminal%v lost composer focus", bottom)
				}
				if native && client.Selection().Range != (ui.TextInputRange{Start: 1, End: 4}) {
					t.Fatalf("terminal%v changed selection %+v", bottom, client.Selection())
				}
			}
			for _, width := range []int{900, 1600} {
				tt.SetSize(width, 800)
				if !tt.Focused("Composer draft") {
					t.Fatalf("width%d lost composer focus", width)
				}
			}
			for _, open := range []bool{false, true} {
				rightOpen = open
				tt.Frame()
				if !tt.Focused("Composer draft") {
					t.Fatalf("right open%v lost composer focus", open)
				}
			}
			if native && client.Selection().Range != (ui.TextInputRange{Start: 1, End: 4}) {
				t.Fatalf("right changes changed selection%+v", client.Selection())
			}
			tt.Type("Z")
			want := "Z"
			if native {
				want = "AZE"
			}
			if draft != want {
				t.Fatalf("draft=%q, want preserved-selection replacement %q", draft, want)
			}
		})
	}
}

func TestTerminalToggleDoesNotStealFocusFromSidebar(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(fmt.Sprint(native), func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom = 0
			draft := "ABCDE"
			client := composer.NewDraftClient(&draft)
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Store: session.New(), Sizes: &sizes, Draft: &draft, UseComposerTextInput: native, ComposerTextInput: client, OnOpenSettings: func() {}})
			}, 1200, 800)
			label := i18n.T(i18n.KeySettingsAccountSeat)
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if !tt.Focused(label) {
				t.Fatal("sidebar button never retained focus")
			}
			for _, bottom := range []float32{180, 0, 180, 0} {
				sizes.Bottom = bottom
				tt.Frame()
				if !tt.Focused(label) || tt.Focused("Composer draft") {
					t.Fatalf("terminal%v stole sidebar focus", bottom)
				}
			}
			for _, width := range []int{900, 1600} {
				tt.SetSize(width, 800)
				if width < 768 && SidebarCollapsedWidth() == 0 {
					continue
				}
				if !tt.Focused(label) || tt.Focused("Composer draft") {
					t.Fatalf("width%d stole sidebar focus", width)
				}
			}
			if draft != "ABCDE" {
				t.Fatalf("terminal changed draft%q", draft)
			}
		})
	}
}
