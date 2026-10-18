package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
)

func TestMiniPlayerUsesTranscriptAndAllFloatingOcclusions(t *testing.T) {
	w := &WorkspacePanes{}
	w.setConversationBounds(ui.Rect{X: 280, Y: 90, W: 720, H: 410})
	if w.conversationBounds.X != 280 || w.conversationBounds.Y != 90 || w.conversationBounds.Width != 720 || w.conversationBounds.Height != 410 {
		t.Fatal("transcript bounds changed to outer column", w.conversationBounds)
	}
	source := &WorkspacePane{ID: "source-browser", Float: &WorkspaceFloat{X: 300, Y: 100, Width: 380, Height: 300}}
	above := &WorkspacePane{ID: "other-resource", Float: &WorkspaceFloat{X: 500, Y: 180, Width: 280, Height: 200}}
	holes := miniPlayerFloatOcclusions([]*WorkspacePane{source, above}, 1000, 800)
	if len(holes) != 2 || holes[0].X != 300 || holes[0].Width != 380 || holes[1].X != 500 {
		t.Fatal("source float incorrectly exempted from mini-player clipping", holes)
	}
	source.Float.Minimized = true
	holes = miniPlayerFloatOcclusions([]*WorkspacePane{source}, 1000, 800)
	if len(holes) != 1 || holes[0].Height != 38 {
		t.Fatal("minimized resource chrome not clipped", holes)
	}
	if len(miniPlayerFloatOcclusions(nil, 1000, 800)) != 0 {
		t.Fatal("closed floats left stale occlusions")
	}
}
