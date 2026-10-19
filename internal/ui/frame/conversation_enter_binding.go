package frame

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func conversationEnterOwner(snap session.ViewState) string {
	return fmt.Sprintf("%d:%s", snap.ComposerHostGeneration, snap.SelectedID)
}

// The original observes mounted structural/attribute changes, not streaming
// character data. Keep this native signal to row identity/type and chrome;
// actual card/viewport resizing is measured separately after layout.
func conversationEnterStructure(m Model, snap session.ViewState, phase conversationLayoutPhase, takeover bool) string {
	var signature strings.Builder
	fmt.Fprintf(&signature, "%d:%t:%t:%t:%d:%d:%d:%t:%s", phase, takeover, snap.PromptBusy, snap.HistoryLoading, len(snap.BandActions), len(snap.WorkflowRuns), len(snap.Messages), !snap.PlanBoard.Empty(), snap.QueueEditor.ItemID)
	if m.ChatPaneTab != nil {
		fmt.Fprintf(&signature, ":%s", *m.ChatPaneTab)
	}
	for _, msg := range snap.Messages {
		fmt.Fprintf(&signature, "|%s:%s:%s:%s:%t:%d", msg.ID, msg.Kind, msg.Role, msg.EventType, msg.Streaming, len(msg.Files))
	}
	if m.ComposerState != nil {
		fmt.Fprintf(&signature, "|draft:%d:%t:%t", len(m.ComposerState.Attachments), m.ComposerState.Error != "", m.ComposerState.EditSeq != 0)
	}
	return signature.String()
}

type conversationEnterScroll struct {
	scroll       ui.ScrollState
	lastX, lastY float32
}

func (s *conversationEnterScroll) signal(enter *conversationEnterState, now time.Time) {
	if s.scroll.X != s.lastX || s.scroll.Y != s.lastY {
		enter.signal(now)
		s.lastX, s.lastY = s.scroll.X, s.scroll.Y
	}
}
