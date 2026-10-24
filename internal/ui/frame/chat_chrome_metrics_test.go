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

// chatChromeTester renders the conversation shell at the reference window size.
// The sidebar carries the WhaleIsle 鲸鱼娘 row, which the browser reference does
// not, so only positions above it and same-row relationships are asserted
// absolutely.
func chatChromeTester(t *testing.T) *ui.Tester {
	t.Helper()
	i18n.SetLocale("zh")
	sizes := DefaultLayout()
	sizes.Bottom, sizes.Right = 0, 0
	right := float32(0)
	colors := theme.Resolve("deepseek.light").Light
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Store: session.New(), Sizes: &sizes, FilePaneWidth: &right, OnOpenSettings: func() {}}
		ui.Column(c).Fill().Children(func() {
			mainRow(c, m, session.ViewState{}, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
		})
	}, 1440, 920)
	tt.Frame()
	return tt
}

// TestSidebarChromeMatchesTheNativePage pins the sidebar controls measured from
// the installed harness page at 1440x920: the new-session button is 252x38 at
// (14,70), the collapse control is the 28x28 seat at (240,22), and the page rows
// sit 40 apart inside the 256-DIP list with their label inset at x=46.
func TestSidebarChromeMatchesTheNativePage(t *testing.T) {
	tt := chatChromeTester(t)
	defer i18n.SetLocale(i18n.DefaultLocale)

	fresh, ok := tt.Find(i18n.T(i18n.KeySidebarNewSessionFallback))
	if !ok {
		t.Fatalf("missing new-session button; texts=%q", tt.Texts())
	}
	if fresh.X != 14 || fresh.Y != 70 || fresh.W != 252 || fresh.H != 38 {
		t.Fatalf("new-session button=%+v, want 14,70 252x38", fresh)
	}
	collapse, ok := tt.Find(i18n.T(i18n.KeySidebarCollapse))
	if !ok {
		t.Fatalf("missing collapse control; texts=%q", tt.Texts())
	}
	if collapse.X != 240 || collapse.Y != 22 || collapse.W != 28 || collapse.H != 28 {
		t.Fatalf("collapse control=%+v, want 240,22 28x28", collapse)
	}
	market, ok := tt.Find(i18n.T(i18n.KeySidebarMarket))
	if !ok {
		t.Fatalf("missing 插件 row; texts=%q", tt.Texts())
	}
	schedule, ok := tt.Find(i18n.T(i18n.KeySidebarSchedule))
	if !ok {
		t.Fatalf("missing 自动化任务 row; texts=%q", tt.Texts())
	}
	// Native labels: 插件 at y=127 inside the row at y=120, 自动化任务 at y=167.
	if market.X != 46 || market.Y < 125.5 || market.Y > 129.5 {
		t.Fatalf("插件 label=%+v, want x=46 y≈127", market)
	}
	// The native row is a 22-DIP line box under 7-DIP padding, so the label box is
	// 22 tall; an 18.7 box centres the ink ~1.7 DIP lower.
	if market.H != 22 {
		t.Fatalf("插件 label height=%.1f, want the native 22-DIP line box", market.H)
	}
	if pitch := schedule.Y - market.Y; pitch < 39.5 || pitch > 40.5 {
		t.Fatalf("page row pitch=%.1f, want 40", pitch)
	}
	// The workspace toolbar's trailing icon sits at the native x=240, 28x28.
	options, ok := tt.Find("视图选项")
	if !ok {
		t.Fatalf("missing view-options control; texts=%q", tt.Texts())
	}
	if options.X != 240 || math.Abs(float64(options.W-28)) > 0.01 || math.Abs(float64(options.H-28)) > 0.01 {
		t.Fatalf("view options=%+v, want x=240 28x28", options)
	}
	// round182: the whole heading row (工作区/多选会话/视图选项/添加工作区/搜索会话)
	// sat 6.67 DIP low; it must now sit within 1 DIP of the native y=250.
	if math.Abs(float64(options.Y-250)) > 1 {
		t.Fatalf("view options y=%.2f, want 250±1 (native heading row)", options.Y)
	}
}

// TestSidebarFooterSeatIsTheNativeWidth keeps the merged settings/account seat on
// the page's 14-DIP inset with the native 252-DIP width.
func TestSidebarFooterSeatIsTheNativeWidth(t *testing.T) {
	tt := chatChromeTester(t)
	defer i18n.SetLocale(i18n.DefaultLocale)
	seat, ok := tt.Find(i18n.T(i18n.KeySettingsAccountMenu))
	if !ok {
		t.Fatalf("missing footer seat; texts=%q", tt.Texts())
	}
	if seat.X < 13.5 || seat.X > 14.5 || seat.W < 251.5 || seat.W > 252.5 {
		t.Fatalf("footer seat=%+v, want x=14 w=252", seat)
	}
	if seat.H != 36 {
		t.Fatalf("footer seat height=%.1f, want 36", seat.H)
	}
}
