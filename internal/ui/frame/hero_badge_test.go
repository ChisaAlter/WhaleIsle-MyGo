package frame

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// TestHeroPreviewBadgeStaysInsideTheColumn covers the after-capture defect: a
// resolved Center wider than the real conversation box centred the hero stack
// past the column's right edge, slicing the 预览版 badge. The seat now measures
// the laid-out column, so the headline stays inside on every width and even when
// a stale Center is handed in.
func TestHeroPreviewBadgeStaysInsideTheColumn(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, tc := range []struct {
		name        string
		width       int
		sidebar     float32
		staleCenter float32
	}{
		{"1440-wide", 1440, 420, 0},
		{"1440-stale-center", 1440, 420, 1600},
		{"1024-boundary", 1024, 280, 1200},
		{"800-drawer", 800, 280, 900},
		{"600-narrow", 600, 280, 700},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom, sizes.Right = 0, 0
			sizes.Sidebar, sizes.SidebarExpanded = tc.sidebar, tc.sidebar
			if tc.staleCenter > 0 {
				sizes.Center = tc.staleCenter
			}
			right := float32(0)
			colors := theme.Resolve("deepseek.light").Light
			tt := ui.NewTester(func(c *ui.Context) {
				m := Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right}
				ui.Column(c).Fill().Children(func() {
					mainRow(c, m, session.ViewState{}, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
				})
			}, tc.width, 800)
			// The seat measures the column on its first frame; one settle frame is
			// the contract for a stale Center.
			tt.Frame()

			badge, ok := tt.Find(i18n.T(i18n.KeyChatHeroPreview))
			if !ok {
				t.Fatalf("missing hero preview badge; texts=%q", tt.Texts())
			}
			if badge.X < 0 || badge.X+badge.W > float32(tc.width)+0.5 {
				t.Fatalf("badge=%+v outside the %d DIP window", badge, tc.width)
			}
			slogan, ok := tt.Find(i18n.T(i18n.KeyChatHeroSlogan))
			if !ok || slogan.X < 0 || slogan.X+slogan.W > float32(tc.width)+0.5 {
				t.Fatalf("slogan=%+v outside the window", slogan)
			}
			card, ok := tt.Find("Composer card")
			if !ok || card.X < 0 || card.X+card.W > float32(tc.width)+0.5 {
				t.Fatalf("composer card=%+v outside the window", card)
			}
			if badge.X < slogan.X+slogan.W {
				t.Fatalf("badge=%+v overlaps the slogan=%+v", badge, slogan)
			}
		})
	}
}

// TestHeroStackCentresInTheConversationColumn pins the stack's horizontal seat:
// the headline and the input card share one centred column at desktop widths.
func TestHeroStackCentresInTheConversationColumn(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	sizes := DefaultLayout()
	sizes.Bottom, sizes.Right = 0, 0
	right := float32(0)
	colors := theme.Resolve("deepseek.light").Light
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right}
		ui.Column(c).Fill().Children(func() {
			mainRow(c, m, session.ViewState{}, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
		})
	}, 1440, 800)
	tt.Frame()
	divider, _ := tt.Find("Sidebar divider")
	columnLeft := divider.X + divider.W
	columnRight := float32(1440)
	card, ok := tt.Find("Composer card")
	if !ok {
		t.Fatal("composer card missing")
	}
	cardCenter := card.X + card.W/2
	// ConversationRoot reserves the stable scrollbar gutter on the right, so the
	// seat's centre sits half a gutter left of the column's.
	columnCenter := columnLeft + (columnRight-columnLeft)/2 - conversationScrollGutter/2
	if diff := cardCenter - columnCenter; diff > 1 || diff < -1 {
		t.Fatalf("card centre %.1f vs seat centre %.1f (column %s..%.0f)", cardCenter, columnCenter, fmt.Sprint(columnLeft), columnRight)
	}
}
