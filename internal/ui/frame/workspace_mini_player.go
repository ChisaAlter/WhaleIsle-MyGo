package frame

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Only the active conversation layout presents mini-players. Hidden session
// layouts still own their guests, but EndContentFrame/PauseContent hide them.
func workspaceMiniPlayers(c *ui.Context, m Model) {
	w := m.Sizes.Workspace
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.browser != nil {
			n.browser.ViewMiniPlayer(c, w.conversationBounds)
		}
	})
}

func (w *WorkspacePanes) setConversationBounds(r ui.Rect) {
	w.conversationBounds = mygo.Rectangle{X: int(r.X), Y: int(r.Y), Width: int(r.W), Height: int(r.H)}
}
