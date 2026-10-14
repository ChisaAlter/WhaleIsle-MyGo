package chrome

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestPRPanelHiddenWhenClosed(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	tt := ui.NewTester(func(c *ui.Context) {
		DrawPRPanel(c, DefaultTitleBarColors(), false, PRPanel{
			CurrentLabel: "#1 Hi [OPEN]",
			ViewLabel:    i18n.T(i18n.KeyGitPRView),
		})
	}, 600, 120)

	if tt.HasText(i18n.T(i18n.KeyGitPRView)) {
		t.Fatalf("panel should be hidden; texts = %q", tt.Texts())
	}
}

func TestPRPanelOpenShowsCurrentAndActions(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	tt := ui.NewTester(func(c *ui.Context) {
		DrawPRPanel(c, DefaultTitleBarColors(), true, PRPanel{
			CurrentLabel: "#12 Fix login [OPEN]",
			ViewLabel:    i18n.T(i18n.KeyGitPRView),
			CreateLabel:  i18n.T(i18n.KeyGitPRCreate),
			RefreshLabel: i18n.T(i18n.KeyGitPRRefresh),
			CloseLabel:   i18n.T(i18n.KeyGitPRClose),
			Recent: []PRListItem{{
				Label:         "#11 Older [OPEN]",
				CheckoutLabel: i18n.T(i18n.KeyGitPRCheckout),
				OnCheckout:    func() {},
			}},
			OnView:    func() {},
			OnCreate:  func() {},
			OnRefresh: func() {},
			OnClose:   func() {},
		})
	}, 700, 160)

	for _, want := range []string{
		"#12 Fix login [OPEN]",
		i18n.T(i18n.KeyGitPRView),
		i18n.T(i18n.KeyGitPRCreate),
		i18n.T(i18n.KeyGitPRRefresh),
		"#11 Older [OPEN]",
		i18n.T(i18n.KeyGitPRCheckout),
	} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q; texts = %q", want, tt.Texts())
		}
	}
}

func TestPRPanelMissingGH(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	msg := i18n.T(i18n.KeyGitPRNoGH)
	tt := ui.NewTester(func(c *ui.Context) {
		DrawPRPanel(c, DefaultTitleBarColors(), true, PRPanel{
			MissingGH:     true,
			MissingGHText: msg,
			RefreshLabel:  i18n.T(i18n.KeyGitPRRefresh),
			OnRefresh:     func() {},
		})
	}, 600, 100)

	if !tt.HasText(msg) {
		t.Fatalf("missing no-gh message; texts = %q", tt.Texts())
	}
	if tt.HasText(i18n.T(i18n.KeyGitPRView)) {
		t.Fatalf("view should be hidden without gh; texts = %q", tt.Texts())
	}
}

func TestTitleBarPRSummaryAndPanelToggle(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)

	appName := i18n.T(i18n.KeyAppName)
	tt := ui.NewTester(func(c *ui.Context) {
		TitleBarWithActions(c, appName, DefaultTitleBarColors(), &GitBadge{
			Branch:       "feature",
			PRSummary:    "#12 OPEN",
			PRPanelLabel: i18n.T(i18n.KeyGitPRPanel),
			PRViewLabel:  i18n.T(i18n.KeyGitPRView),
			OnPRPanel:    func() {},
			OnPRView:     func() {},
		}, nil)
	}, 900, TitleBarHeight)

	if !tt.HasText("#12 OPEN") {
		t.Fatalf("missing PR summary; texts = %q", tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeyGitPRPanel)) {
		t.Fatalf("missing PR panel toggle; texts = %q", tt.Texts())
	}
}
