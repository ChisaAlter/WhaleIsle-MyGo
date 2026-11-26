package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/design"
)

// Only the main Conversation owns this target. Sidebar, resource docks and
// a subagent replacing the main view must not attach to its parent's draft.
func conversationFileDrop(c *ui.Context, m Model, area ui.Element, sessionID string) {
	conversationFileDropPtr(c, &m, area, sessionID)
}

func conversationFileDropPtr(c *ui.Context, m *Model, area ui.Element, sessionID string) {
	if m.Store == nil || m.Store.SelectedSessionID() != sessionID {
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
		t := tk(c)
		mask := t.Primary.Alpha(0.12)
		border := t.Primary
		if disabled {
			mask = t.MutedForeground.Alpha(0.12)
			border = t.MutedForeground
		}
		ui.Column(c).Absolute().Left(bounds.X).Top(bounds.Y).Width(bounds.W).Height(bounds.H).
			Padding(design.Sp(6)).AlignItems(ui.Center).Justify(ui.Center).
			Background(mask).Border(design.Sp(0.5), border).PassThrough().Children(func() {
			frameText(c, label, design.TextBase, design.WeightMedium, t.Foreground).PassThrough()
			if !disabled {
				frameText(c, paneLabel("文件将作为当前对话附件添加", "Files will be attached to this conversation"), design.TextXS, design.WeightNormal, t.MutedForeground).PassThrough()
			}
		})
	})
}
