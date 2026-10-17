package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/composer"
)

// Only the main Conversation owns this target. Sidebar, resource docks and
// a subagent replacing the main view must not attach to its parent's draft.
func conversationFileDrop(c *ui.Context, m Model, area ui.Element, sessionID string) {
	if m.Store == nil || m.Store.Snapshot().SelectedID != sessionID {
		return
	}
	over := composer.FileDropTarget(area, composer.Model{
		Draft: m.Draft, State: m.ComposerState, WorkspaceRoot: m.WorkspaceRoot,
		OnAttachFiles: m.OnAttachComposerFiles,
	})
	if !over {
		return
	}
	label := paneLabel("放下文件以添加到当前对话", "Drop files into this conversation")
	if m.ComposerState != nil && (m.ComposerState.Sending || m.ComposerState.Preparing > 0) {
		label = paneLabel("请等待附件准备完成", "Wait for attachment preparation")
	}
	bounds := area.Bounds()
	ui.Overlay(c, func() {
		ui.Box(c).Absolute().Left(bounds.X+16).Top(bounds.Y+16).Padding(10).
			Background(c.Theme().Background).Border(1, c.Theme().Accent).PassThrough().Children(func() {
			ui.Text(c, label).PassThrough()
		})
	})
}
