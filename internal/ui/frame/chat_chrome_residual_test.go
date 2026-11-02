package frame

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// chatChromeSeatTester renders the reference chat shell at the native 1440-wide
// column (280 sidebar, divider-shrunk 1158 centre) so the seats below are the
// installed page's numbers, not a synthetic narrow layout.
func chatChromeSeatTester(t *testing.T, height int) *ui.Tester {
	t.Helper()
	i18n.SetLocale("zh")
	sizes := LayoutSizes{Center: 1158, Sidebar: 280, Main: 560, Bottom: 0, Right: 0}
	right := float32(0)
	colors := theme.Resolve("deepseek.light").Light
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right, OnOpenSettings: func() {}}
		ui.Column(c).Fill().Children(func() {
			mainRow(c, m, session.ViewState{}, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
		})
	}, 1440, height)
	tt.Frame()
	return tt
}

func chatChromeSeatFind(t *testing.T, tt *ui.Tester, name string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(name)
	if !ok {
		t.Fatalf("missing %q; texts=%q", name, tt.Texts())
	}
	return r
}

// TestComposerCardMatchesTheConversationBodyWidth is the S28 width lock: the
// rendered composer card and the native conversation body are the same seat, so
// the difference must be 0.00 DIP (the old residual was 772 vs 774.4 = 2.4).
func TestComposerCardMatchesTheConversationBodyWidth(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := chatChromeSeatTester(t, 920)
	card := chatChromeSeatFind(t, tt, "Composer card")
	// Native column: 1440 - 280 sidebar = 1160, axis = 0.64 x 1160 = 742.4,
	// body = axis + two 16-DIP InputBar insets = 774.4.
	axis := conversationContentWidth(1160)
	body := conversationComposerBodyWidth(axis)
	if diff := math.Abs(float64(card.W - body)); diff > 0.001 {
		t.Fatalf("composer width %.4f vs body %.4f = %.4f DIP, want 0", card.W, body, card.W-body)
	}
	if diff := math.Abs(float64(body - 774.4)); diff > 0.01 {
		t.Fatalf("body seat %.4f, want native 774.4 (diff %.4f)", body, diff)
	}
	// Native card: 774.39 at x=469.3 (left inset 189.3 inside the 280..1440
	// column; the 5-DIP scrollbar gutter sits on the trailing edge). Our seat is
	// +0.5 from the divider-rounded centre, so assert the native box, not a
	// synthetic column centre.
	if diff := math.Abs(float64(card.X - 469.3)); diff > 0.6 {
		t.Fatalf("composer x %.2f, want the native 469.3 (diff %.2f)", card.X, diff)
	}
	if centre, native := card.X+card.W/2, float32(469.3)+774.39/2; math.Abs(float64(centre-native)) > 0.6 {
		t.Fatalf("composer centre %.2f, want the native %.2f", centre, native)
	}
}

// TestNavigationLabelKeepsTheNativeBaseline is the S28 baseline lock: the native
// panel row is a 22-DIP label box at y=127 (7 DIP row padding + 11), so the box
// centre must equal 138 with zero deviation. The pre-round177 18.7-DIP text box
// put the ink 1.7 DIP lower.
func TestNavigationLabelKeepsTheNativeBaseline(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := chatChromeSeatTester(t, 920)
	market := chatChromeSeatFind(t, tt, i18n.T(i18n.KeySidebarMarket))
	if market.H != conversationNavRowLineBox {
		t.Fatalf("插件 label height %.2f, want the native 22-DIP line box", market.H)
	}
	if market.Y != 127 {
		t.Fatalf("插件 label y %.2f, want the native 127", market.Y)
	}
	if deviation := market.Y + market.H/2 - conversationNavLabelCentre; deviation != 0 {
		t.Fatalf("navigation baseline deviation %.2f DIP, want 0", deviation)
	}
	schedule := chatChromeSeatFind(t, tt, i18n.T(i18n.KeySidebarSchedule))
	if pitch := schedule.Y - market.Y; pitch != 40 {
		t.Fatalf("page row pitch %.2f, want the native 40", pitch)
	}
}

// The installed source defines a 208-DIP visual stack and32-DIP internal
// bottom padding. Earlier250/155.34 absolute seats are obsolete and must not
// compensate a correctly sized card with unrelated top blocks.
func TestHeroStackHasADefinedNativeSeat(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	if conversationHeroStackHeight != 208 || conversationHeroStackFoot != 32 {
		t.Fatalf("hero source stack=%.2f footer=%.2f want208/32", conversationHeroStackHeight, conversationHeroStackFoot)
	}
	for _, height := range []int{864, 920} {
		tt := chatChromeSeatTester(t, height)
		headline := chatChromeSeatFind(t, tt, "Conversation hero headline")
		preset := chatChromeSeatFind(t, tt, "Composer hero preset row")
		card := chatChromeSeatFind(t, tt, "Composer card")
		body := chatChromeSeatFind(t, tt, "Conversation body")
		if got := card.Y + card.H - headline.Y; math.Abs(float64(got-208)) > .1 {
			t.Fatalf("visual stack=%.2f want208", got)
		}
		if card.H != 114 || preset.H != 28 || card.Y-(preset.Y+preset.H) != 8 {
			t.Fatalf("sourcecard/presetgeometry card=%+v preset=%+v", card, preset)
		}
		if got := headline.Y + (208+32)/2 - (body.Y + body.H/2); math.Abs(float64(got)) > .1 {
			t.Fatalf("centered flex item offset=%.2f want0", got)
		}
	}
}
