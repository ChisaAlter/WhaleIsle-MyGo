package frame

import (
	"github.com/egoist/mygo/ui"
	"math"
	"testing"
)

func TestTranscriptWidthPreferenceAndNarrowConcession(t *testing.T) {
	s := DefaultLayout()
	s.TranscriptWidth = 1400
	if got := s.conversationWidth(1800); got != 1400 {
		t.Fatal(got)
	}
	if got := s.conversationWidth(900); got != 724 {
		t.Fatal(got)
	}
	if got := s.conversationWidth(400); got != 640 {
		t.Fatal(got)
	}
	if s.TranscriptWidth != 1400 || s.conversationWidth(1800) != 1400 {
		t.Fatal("viewport overwrote preference")
	}
	_, _, width := conversationWidthRails(400, 336)
	if width != 0 {
		t.Fatal("narrow window exposes unsafe handle")
	}
	if resolvedConversationWidth(1200, float32(math.NaN())) != 768 {
		t.Fatal("corrupt preference")
	}
}
func TestTranscriptWidthSymmetryReleaseAndCancel(t *testing.T) {
	s := DefaultLayout()
	s.TranscriptWidth = 1400
	s.startConversationWidth("right", 100, 1200)
	s.endConversationWidth(100, true)
	if s.TranscriptWidth != 1400 {
		t.Fatal("press-only overwrote wider preference")
	}
	s.startConversationWidth("right", 100, 1200)
	s.dragConversationWidth(120)
	if s.transcriptGesture.preview != 1024 || s.TranscriptWidth != 1400 {
		t.Fatal("drag prematurely persisted")
	}
	s.endConversationWidth(120, false)
	if s.TranscriptWidth != 1400 {
		t.Fatal("cancel persisted")
	}
	s.TranscriptWidth = 800
	s.startConversationWidth("left", 200, 1200)
	s.endConversationWidth(180, true)
	if s.TranscriptWidth != 840 {
		t.Fatal("left handle must grow symmetrically")
	}
	if s.Sidebar != 280 || s.Right != 0 {
		t.Fatal("transcript changed global pane widths")
	}
	s.startConversationWidth("right", 200, 1200)
	s.dragConversationWidth(1000)
	s.endConversationWidth(200, true)
	if s.TranscriptWidth != 840 {
		t.Fatal("overshoot return changed preference")
	}
}
func TestTranscriptWidthHandleDragAndKeyboard(t *testing.T) {
	s := DefaultLayout()
	s.Center = 1200
	tt := ui.NewTester(func(c *ui.Context) {
		body := ui.Box(c).Fill()
		body.Children(func() {
			paintConversationWidthControls(c, Model{Sizes: &s, FileBindingID: "session"}, body, s.conversationWidth(1200), true)
		})
	}, 1200, 400)
	rail, ok := tt.Find("Resize transcript right")
	if !ok {
		t.Fatal("missing right rail")
	}
	x, y := rail.X+rail.W/2, rail.Y+100
	tt.Press(x, y)
	tt.Move(x+20, y)
	tt.Release(x+20, y)
	if s.TranscriptWidth != 808 {
		t.Fatalf("drag width %v", s.TranscriptWidth)
	}
	tt.Key(0, ui.KeyRight)
	if s.TranscriptWidth != 828 {
		t.Fatal("keyboard resize")
	}
	rail, _ = tt.Find("Resize transcript right")
	x = rail.X + rail.W/2
	tt.Press(x, y)
	tt.Move(x+20, y)
	tt.Key(0, ui.KeyEscape)
	tt.Release(x+20, y)
	if s.TranscriptWidth != 828 {
		t.Fatal("Escape did not cancel drag")
	}
}
