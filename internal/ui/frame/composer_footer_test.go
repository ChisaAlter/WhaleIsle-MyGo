package frame

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	costui "whaleisle/internal/ui/sessioncost"
	"whaleisle/internal/ui/theme"
)

func TestConversationFooterStaysInsideNativeViewport(t *testing.T) {
	for _, height := range []int{800, 792, 600} {
		for _, transcript := range []string{"one persisted message", strings.Repeat("history line\n", 60)} {
			for _, showCost := range []bool{false, true} {
				t.Run(fmt.Sprintf("height%d/lines%d/cost%v", height, strings.Count(transcript, "\n"), showCost), func(t *testing.T) {
					sizes := DefaultLayout()
					sizes.Bottom = 0
					draft := ""
					state := composer.NewDraftState("")
					costState := costui.State{}
					preferences := settings.ViewState{}
					if showCost {
						preferences.Namespaces = []settings.Namespace{{NS: "ui-conversation", Value: json.RawMessage(`{"statsLine":true,"sessionCost":true,"officialPeakValley":true}`)}}
					}
					store := session.New()
					store.Select("current")
					store.AppendStreamDelta("user", transcript)
					tt := ui.NewTester(func(c *ui.Context) {
						originalScrollbarTheme(c)
						ui.Column(c).Fill().Children(func() {
							MainCaptionFileDrop(c, Model{Store: store, Sizes: &sizes}, func() { ui.Row(c).Fill().Height(chrome.EffectiveTitleBarHeight()) })
							ui.Column(c).Grow(1).MinHeight(0).Children(func() {
								View(c, Model{Store: store, Sizes: &sizes, Draft: &draft, ComposerState: state,
									UseComposerTextInput: true, ComposerTextInput: state.Client, ThemeID: theme.IDDeepseekLight, SessionCost: costui.Model{State: &costState, Preferences: preferences}})
							})
						})
					}, 1200, height)
					firstLabel := "Host 未提供计费 Token / Billed tokens unavailable"
					lastLabel := firstLabel
					if showCost {
						lastLabel = "仅包含已上报用量的会话及子代理；缺失部分未计入 / Reported sessions and subagents only; missing usage excluded"
						if !tt.HasText("Switch in") || !tt.HasText("Session estimate") {
							t.Fatal("expanded official cost rows were not mounted")
						}
					}
					footer, ok := tt.Find(lastLabel)
					if !ok || footer.H < 11 || footer.Y+footer.H > float32(height)-3 {
						t.Fatalf("below-composer status clipped: footer=%+v viewportHeight=%d", footer, height)
					}
					card, ok := tt.Find("Composer card")
					first, _ := tt.Find(firstLabel)
					if !ok || card.Y+card.H > first.Y {
						t.Fatalf("card=%+v overlaps footer=%+v", card, footer)
					}
					saveConversationEvidence(t, tt, fmt.Sprintf("composer-footer-%d-lines%d-cost%v", height, strings.Count(transcript, "\n"), showCost))
				})
			}
		}
	}
}
