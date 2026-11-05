package composer

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

func TestChildCompactToolbarKeepsCapturedCallbacksAndBusyGates(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	draft := NewDraftState("")
	picked, pasted, sent := 0, 0, 0
	var submitted string
	v := ChildEditorModel{Draft: draft, OnPickImages: func() { picked++ }, OnPasteImage: func(explicit bool) {
		if !explicit {
			t.Error("explicit menu paste lost its captured intent")
		}
		pasted++
	}, OnSend: func(text string) { sent++; submitted = text }}
	tt := ui.NewTester(func(c *ui.Context) { ChildEditor(c, v) }, 180, 200)
	for _, control := range []struct {
		label string
		size  float32
	}{{"Tools", 28}, {i18n.T(i18n.KeyComposerSend), 34}} {
		bounds, ok := tt.Find(control.label)
		if !ok || bounds.W != control.size || bounds.H != control.size || bounds.X < 0 || bounds.X+bounds.W > 180 {
			t.Fatalf("compact child control %q clipped at narrow width: %+v", control.label, bounds)
		}
	}
	if tt.HasText("Add images") || tt.HasText("Paste image") {
		t.Fatal("occasional intake retained permanent toolbar buttons")
	}
	for _, item := range []string{"Add images", "Paste image"} {
		if err := tt.Click("Tools"); err != nil {
			t.Fatal(err)
		}
		if got := tt.Menu(); !reflect.DeepEqual(got, []string{"Add images", "Paste image"}) {
			t.Fatalf("child tools borrowed parent controls or lost intake: %v", got)
		}
		if err := tt.ChooseMenuItem(item); err != nil {
			t.Fatal(err)
		}
	}
	if picked != 1 || pasted != 1 || sent != 0 || draft.Text != "" {
		t.Fatalf("intake callbacks affected sending or another draft: pick=%d paste=%d send=%d draft=%q", picked, pasted, sent, draft.Text)
	}
	_ = tt.Click(i18n.T(i18n.KeyComposerSend))
	if sent != 0 {
		t.Fatal("empty child draft sent")
	}
	draft.Text = "/model literal child text"
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeyComposerSend)); err != nil || sent != 1 || submitted != draft.Text || draft.ModelCommand.Open {
		t.Fatalf("child send was rerouted into a root command: text=%q send=%d err=%v", submitted, sent, err)
	}
	for _, lock := range []string{"disabled", "sending"} {
		v.Disabled, draft.Sending = lock == "disabled", lock == "sending"
		tt.Frame()
		_ = tt.Click("Tools")
		if len(tt.Menu()) != 0 {
			t.Fatalf("%s child opened intake", lock)
		}
		_ = tt.Click(i18n.T(i18n.KeyComposerSend))
		if picked != 1 || pasted != 1 || sent != 1 {
			t.Fatalf("%s child invoked a locked callback", lock)
		}
	}
}

func TestChildAttachmentRailRetainsNativeDraftOwnerAcrossMountAndRemoval(t *testing.T) {
	draft := NewDraftState("A")
	v := ChildEditorModel{Draft: draft}
	tt := ui.NewTester(func(c *ui.Context) { ChildEditor(c, v) }, 320, 300)
	before, _ := tt.Find("Composer draft")
	if err := tt.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	client := draft.Client
	input := &client.input
	input.SetMarkedText(nil, "中文", ui.TextInputRange{Start: 2, End: 2})
	path := filepath.Join(t.TempDir(), "owned.png")
	v.Images = []string{path}
	tt.Frame()
	card, ok := tt.Find("Attachment owned.png")
	if !ok || card.W != 64 || card.H != 64 {
		t.Fatalf("child image does not use shared 64x64 thumb: %+v", card)
	}
	after, _ := tt.Find("Composer draft")
	// The shared InputBar keeps its 12-DIP content gap after the rail's
	// 66-DIP viewport and -6-DIP trailing margin.
	if after.Y-before.Y != 72 {
		t.Fatalf("child rail retained excess height: before=%+v after=%+v", before, after)
	}
	v.Images = nil
	for range 3 {
		tt.Frame()
		marked, active := client.MarkedRange()
		if draft.Client != client || !active || marked != (ui.TextInputRange{Start: 1, End: 3}) || !tt.Focused("Composer draft") || draft.Text != "A中文" {
			t.Fatalf("child rail remount released captured native IME owner: active=%v range=%v text=%q", active, marked, draft.Text)
		}
	}
	if restored, _ := tt.Find("Composer draft"); restored.Y != before.Y {
		t.Fatalf("empty child retained an image rail: %+v", restored)
	}
}

func TestChildCompactImagePreviewAndRemovalRemainChildOwned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image.NewRGBA(image.Rect(0, 0, 5, 3))); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	draft := NewDraftState("preserved child draft")
	v := ChildEditorModel{Draft: draft, Images: []string{path}}
	removed := ""
	v.OnRemoveImage = func(value string) { removed = value; v.Images = nil }
	tt := ui.NewTester(func(c *ui.Context) { ChildEditor(c, v) }, 420, 500)
	deadline := time.Now().Add(2 * time.Second)
	for {
		_ = tt.Click("Preview attachment owned.png")
		if _, open := tt.Find("1:1 / Fit"); open {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("compact child thumbnail never became a real preview")
		}
		time.Sleep(5 * time.Millisecond)
		tt.Frame()
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if err := tt.Click("Remove attachment owned.png"); err != nil {
		t.Fatal(err)
	}
	if removed != path || len(v.Images) != 0 || len(draft.Attachments) != 0 || draft.Text != "preserved child draft" {
		t.Fatalf("removal escaped the captured child callback: removed=%q child=%v root=%v draft=%q", removed, v.Images, draft.Attachments, draft.Text)
	}
}
