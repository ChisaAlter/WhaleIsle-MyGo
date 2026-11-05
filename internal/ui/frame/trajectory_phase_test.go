package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestTrajectoryRemainsVisibleAcrossChatHistoryPhases(t *testing.T) {
	for _, tc := range []struct {
		name string
		snap session.ViewState
	}{
		{"ready-empty", session.ViewState{SelectedID: "current", HistoryReady: true}},
		{"settling", session.ViewState{SelectedID: "current", HistoryLoading: true}},
		{"failed-empty", session.ViewState{SelectedID: "current", HistoryError: "history unavailable"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sizes := DefaultLayout()
			sizes.Bottom = 0
			tab := "trajectory"
			store := session.New()
			colors := theme.Resolve(theme.IDDeepseekLight).Light
			tt := ui.NewTester(func(c *ui.Context) {
				sizes.ResolveColumns(1200, false)
				ui.Column(c).Fill().Children(func() {
					mainRow(c, Model{Store: store, Sizes: &sizes, ChatPaneTab: &tab}, tc.snap,
						colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
				})
			}, 1200, 800)
			ledger, ok := tt.Find("Trajectory ledger")
			if !ok || ledger.W <= 0 || ledger.H <= 0 || !tt.HasText(i18n.T(i18n.KeyTrajEmpty)) {
				t.Fatalf("selected trajectory was hidden by %s: ledger=%+v found=%v texts=%q", tc.name, ledger, ok, tt.Texts())
			}
		})
	}
}
