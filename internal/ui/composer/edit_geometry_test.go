package composer

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestComposerEditStatusFloatsWithoutMovingDraft(t *testing.T) {
	for _, width := range []int{320, 800} {
		for _, native := range []bool{false, true} {
			t.Run(fmt.Sprintf("width%d/native%v", width, native), func(t *testing.T) {
				d := NewDraftState("An existing draft")
				editing, cancelled := false, 0
				tt := ui.NewTester(func(c *ui.Context) {
					ui.Column(c).Fill().Padding(70, 0, 0, 0).Children(func() {
						View(c, Model{State: d, Draft: &d.Text, TextInput: d.Client, UseTextInputClient: native, Editing: editing, OnCancelEdit: func() { cancelled++ }})
					})
				}, width, 300)
				card, _ := tt.Find("Composer card")
				draft, _ := tt.Find("Composer draft")
				if err := tt.Click("Composer draft"); err != nil {
					t.Fatal(err)
				}
				editing = true
				tt.Frame()
				editCard, _ := tt.Find("Composer card")
				editDraft, _ := tt.Find("Composer draft")
				status, found := tt.Find("Composer edit status")
				cancel, button := tt.Find(i18n.T(i18n.KeyQuitConfirmCancel))
				if card != editCard || draft != editDraft || !tt.Focused("Composer draft") {
					t.Fatalf("edit changed card/input/focus: %v -> %v; %v -> %v", card, editCard, draft, editDraft)
				}
				if !found || !button || status.Y+status.H != card.Y-8 || status.X+status.W != card.X+card.W-8 || cancel.W != 22 || cancel.H != 22 {
					t.Fatalf("floating edit geometry card=%+v status=%+v cancel=%+v", card, status, cancel)
				}
				if dir := os.Getenv("PARITY_GAP_SNAPSHOT_DIR"); dir != "" && native {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatal(err)
					}
					f, err := os.Create(filepath.Join(dir, fmt.Sprintf("composer-edit-status-%d.png", width)))
					if err != nil {
						t.Fatal(err)
					}
					err = png.Encode(f, tt.Image())
					f.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := tt.Click(i18n.T(i18n.KeyQuitConfirmCancel)); err != nil || cancelled != 1 {
					t.Fatalf("cancel callback=%d error=%v", cancelled, err)
				}
				d.Sending = true
				tt.Frame()
				tt.Click(i18n.T(i18n.KeyQuitConfirmCancel))
				if cancelled != 1 {
					t.Fatal("pending edit cancellation was enabled")
				}
			})
		}
	}
}
