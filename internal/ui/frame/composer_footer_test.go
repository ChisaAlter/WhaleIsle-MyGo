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
	"whaleisle/internal/ui/i18n"
	costui "whaleisle/internal/ui/sessioncost"
	"whaleisle/internal/ui/theme"
)

func TestConversationFooterStaysInsideNativeViewport(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
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
					// The composer owns this copy; resolve it through its locale keys so
					// the frame test never pins one language's wording.
					firstLabel := i18n.T(i18n.KeyComposerCostUnavailable)
					lastLabel := firstLabel
					if showCost {
						lastLabel = i18n.T(i18n.KeyComposerCostPartial)
						if !textsContain(tt.Texts(), i18n.T(i18n.KeyComposerCostEstimate)) {
							t.Fatalf("expanded cost block missing the estimate row; texts=%q", tt.Texts())
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

// The Electron dock keeps its min-height seat even when both statistics
// switches are off (.dock[data-composer-footer] + invisible pill
// placeholders): the card must never sit flush on the window edge.
func TestConversationFooterKeepsSeatWhenStatsHidden(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, prefs := range []json.RawMessage{
		json.RawMessage(`{"statsLine":false,"sessionCost":false}`),
		json.RawMessage(`{"statsLine":true,"sessionCost":false}`),
	} {
		sizes := DefaultLayout()
		sizes.Bottom = 0
		draft := ""
		state := composer.NewDraftState("")
		costState := costui.State{}
		preferences := settings.ViewState{Namespaces: []settings.Namespace{{NS: "ui-conversation", Value: prefs}}}
		store := session.New()
		store.Select("current")
		store.AppendStreamDelta("user", "one persisted message")
		const height = 800
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
		dock, ok := tt.Find("Composer dock")
		if !ok {
			t.Fatalf("dock seat missing with prefs %s", prefs)
		}
		if dock.H < 24 {
			t.Fatalf("dock seat shorter than the 24 DIP floor: %+v", dock)
		}
		card, ok := tt.Find("Composer card")
		if !ok || card.Y+card.H > dock.Y {
			t.Fatalf("card=%+v overlaps dock=%+v", card, dock)
		}
		if dock.Y-(card.Y+card.H) < 4 {
			t.Fatalf("card touches the dock: card=%+v dock=%+v", card, dock)
		}
	}
}

// textsContain reports whether any rendered text contains want, for copy the
// composer composes from several locale keys.
func textsContain(texts []string, want string) bool {
	for _, text := range texts {
		if strings.Contains(text, want) {
			return true
		}
	}
	return false
}
