package composer

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// Empty Hero's114-DIP card is a CSS component contract, not a viewport fit.
// Both shipped text clients must agree on its52-DIP well and shared text inset.
func TestHeroSourceGeometryBothInputOwners(t *testing.T) {
	for _, native := range []bool{false, true} {
		draft := NewDraftState("")
		m := Model{State: draft, Draft: &draft.Text, TextInput: draft.Client, UseTextInputClient: native, Hero: true}
		tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 840, 500)
		card, ok := tt.Find("Composer card")
		if !ok {
			t.Fatal("missingcard")
		}
		input, ok := tt.Find("Composer draft")
		if !ok {
			t.Fatal("missinginput")
		}
		send, ok := tt.Find(i18n.T(i18n.KeyComposerSend))
		if !ok {
			t.Fatal("missingsend")
		}
		closeMetric(t, card.H, 114, "sourcehero cardheight")
		closeMetric(t, input.H, 52, "sourcehero inputminimum")
		closeMetric(t, input.Y-card.Y, 8, "sourcecard top")
		closeMetric(t, input.X-card.X, 0, "sourceinput fullwidth")
		closeMetric(t, input.W, card.W, "sourceinput scrollport width")
		closeMetric(t, send.W, 34, "sourceprimarywidth")
		closeMetric(t, send.H, 34, "sourceprimaryheight")
		closeMetric(t, card.X+card.W-send.X-send.W, 8, "sourcetoolbar rightpadding")
		left, right, top := draftInsets(true)
		if left != 14 || right != 8 || top != 4 {
			t.Fatalf("hero sourceinsets=%.2f/%.2f/%.2f want14/8/4", left, right, top)
		}
	}
}
