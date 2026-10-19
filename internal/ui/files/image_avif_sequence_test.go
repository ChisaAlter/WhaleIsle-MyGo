package files

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestDesktopFilesAVIFSequenceUsesNativeTimeline(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/test.avifs")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	path := "sequence.avif"
	if !isPreviewImage(path) {
		t.Fatal("desktop .avif image route missing")
	}
	if err := os.WriteFile(filepath.Join(root, path), data, 0600); err != nil {
		t.Fatal(err)
	}
	preview := loadImagePreview(root, path)
	if preview.err != nil || preview.bitmap == nil || preview.animation == nil || len(preview.frames) != 17 || len(preview.animation.Frames) != 0 {
		t.Fatalf("desktop AVIF sequence/staging: %+v", preview)
	}
	if w, h := preview.bitmap.Size(); w != 500 || h != 360 {
		t.Fatalf("AVIF canvas=%dx%d", w, h)
	}
	start := time.Unix(100, 0)
	preview.advance(start)
	if wait := preview.advance(start.Add(90 * time.Millisecond)); wait != 70*time.Millisecond || preview.bitmap != preview.frames[1] {
		t.Fatalf("AVIF timing=%v", wait)
	}
	preview.started = time.Time{}
	// Raster workers are process-global: all native seat drawing uses the
	// ordinary test lifetime. Exact frame timing is checked with advance above.
	{
		selected := path
		tab := &FileTab{Path: path, image: preview}
		m := Model{WorkspaceRoot: root, BindingID: "fixture-host", SessionID: "fixture-session", SelectedFile: &selected, Tabs: &Tabs{}}
		w := tabsFor(m)
		w.Open, w.Active = []*FileTab{tab}, path
		preview.owner = imageOwner(m, tab)
		visible := true
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				if visible {
					paintImagePreview(c, m, path, ui.RGB(0, 0, 0))
				}
			})
		}, 700, 500)
		before, first := append([]byte(nil), tt.Image().Pix...), preview.bitmap
		changed := false
		for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
			time.Sleep(5 * time.Millisecond)
			tt.Frame()
			if preview.bitmap != first && !bytes.Equal(before, tt.Image().Pix) {
				changed = true
				break
			}
		}
		if !changed {
			t.Fatal("actual Files image seat did not paint the next AVIF frame")
		}
		visible = false
		tt.Frame()
		current := preview.bitmap
		time.Sleep(200 * time.Millisecond)
		tt.Frame()
		if preview.bitmap != current {
			t.Fatal("hidden Files seat continued advancing AVIF")
		}
		other := &FileTab{Path: "other.png"}
		w.Open = append(w.Open, other)
		retainImagePreview(m, other, &imagePreview{})
		if tab.image != nil {
			t.Fatal("inactive Files tab retained AVIF frames")
		}
	}
	// The shared decoder must still fail the entire sequence if a later frame
	// cannot finish, and the Files entry must honor cancellation before decode.
	bad := append([]byte(nil), data...)
	for i := len(bad) - 5; i < len(bad); i++ {
		bad[i] = 0
	}
	if p := decodeImagePreviewContext(context.Background(), path, bad); p.err == nil || p.bitmap != nil || len(p.frames) != 0 {
		t.Fatal("corrupt later AVIF frame exposed incomplete Files pixels")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p := decodeImagePreviewContext(ctx, path, data); p.err == nil || p.bitmap != nil {
		t.Fatal("canceled Files AVIF entry decoded pixels")
	}
}

func TestAVIFSequenceKeepsUnregisteredAndMislabeledFilesBoundaries(t *testing.T) {
	data, err := os.ReadFile("../../desktop/imagepreview/testdata/test.avifs")
	if err != nil {
		t.Fatal(err)
	}
	if p := decodeImagePreviewContext(context.Background(), "sequence.gif", data); p.err == nil || p.bitmap != nil || p.animation != nil {
		t.Fatal("mislabeled GIF gained an AVIF entry")
	}
	if isPreviewImage("sequence.avifs") {
		t.Fatal("new AVIS Files suffix was registered")
	}
}
