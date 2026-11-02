package pet

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/storelocale"
)

func TestQuickChatComposerFitsActualCard(t *testing.T) {
	for _, height := range []int{quickChatHeight, quickChatMinHeight} {
		q := &QuickChat{stale: true, status: "Connection changed; reopen this card to keep using the shared conversation.", state: quickChatSnapshot{SessionID: "12345678901234567890123456789012345678901234567890", History: []quickChatRow{{Role: "her", Text: "A message that should use the available scrollable history area."}}}}
		tt := ui.NewTester(q.view, quickChatWidth, height)
		for _, label := range []string{storelocale.T(storelocale.KeyDesktopPetQuickChatDraft), storelocale.T(storelocale.KeyDesktopPetSend), storelocale.T(storelocale.KeyDesktopPetCloseQuickChat)} {
			box, ok := tt.Find(label)
			if !ok || box.X < 0 || box.Y < 0 || box.X+box.W > float32(quickChatWidth)+.5 || box.Y+box.H > float32(height)+.5 || box.W <= 0 || box.H <= 0 {
				t.Errorf("%dx%d %q outside compact card: %+v found=%v", quickChatWidth, height, label, box, ok)
			}
		}
	}
}
