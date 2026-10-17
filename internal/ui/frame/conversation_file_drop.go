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
	disabled := m.ComposerState != nil && (m.ComposerState.Sending || m.ComposerState.Preparing > 0)
	label := paneLabel("放下文件以添加到当前对话", "Drop files into this conversation")
	if disabled {
		label = paneLabel("请等待附件准备完成", "Wait for attachment preparation")
	}
	bounds := area.Bounds()
	ui.Overlay(c, func() {
		// The browser overlay is full-viewport. Keep the equivalent decoration
		// bounded to this owner target so resource panes, child views and WebViews
		// never become a parent-conversation drop surface.
		mask := ui.RGBA(37, 99, 235, 0.12)
		border := c.Theme().Accent
		if disabled {
			mask = ui.RGBA(113, 113, 122, 0.12)
			border = c.Theme().TextMuted
		}
		ui.Column(c).Absolute().Left(bounds.X).Top(bounds.Y).Width(bounds.W).Height(bounds.H).
			Padding(24).AlignItems(ui.Center).Justify(ui.Center).
			Background(mask).Border(2, border).PassThrough().Children(func() {
			ui.Text(c, label).FontSize(16).TextColor(c.Theme().Text).PassThrough()
			if !disabled {
				ui.Text(c, paneLabel("文件将作为当前对话附件添加", "Files will be attached to this conversation")).
					FontSize(12).TextColor(c.Theme().TextMuted).PassThrough()
			}
		})
	})
}
