package main

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
)

func (s *mainState) currentNativePluginSessionOwner() *pluginui.Owner {
	s.mu.Lock()
	p := s.nativePlugins.session
	s.mu.Unlock()
	if p == nil || !s.pluginSessionCurrent(p) {
		return nil
	}
	s.mu.Lock()
	owner := p.owner
	s.mu.Unlock()
	return owner
}

// Native shell call sites supply an exact durable message seq or tool call ID.
// Unmatched/absent contributions retain the existing native drawing path.
func (s *mainState) renderPluginNativeSlot(c *ui.Context, slot, callID string, seq float64, fallback func()) {
	slot = pluginui.CanonicalNativeSlot(slot)
	owner := s.currentNativePluginSessionOwner()
	if owner != nil {
		if slot == "conversation.composer" {
			if message := owner.ComposerError(); message != "" {
				ui.Text(c, message).FillWidth().TextColor(c.Theme().Warning)
			}
		}
		if pluginui.AdditiveNativeSlot(slot) {
			// The original UI fills these seats with every entry: the native controls stay and
			// the plugins' entries join them.
			if fallback != nil {
				fallback()
			}
			owner.RenderNativeSlot(c, slot, callID, -1)
			return
		}
		if seq >= 0 {
			if owner.RenderNativeMessage(c, slot, callID, seq) {
				return
			}
		} else if owner.RenderNativeSlot(c, slot, callID, -1) {
			return
		}
	}
	if fallback != nil {
		fallback()
	}
}

func (s *mainState) pluginConversationViews() []pluginui.ConversationView {
	if owner := s.currentNativePluginSessionOwner(); owner != nil {
		return owner.Snapshot().ConversationViews
	}
	return nil
}
func (s *mainState) selectPluginConversationView(handle string) {
	owner := s.currentNativePluginSessionOwner()
	if owner == nil {
		return
	}
	snapshot := owner.Snapshot()
	if owner.SelectConversationView(snapshot, handle) != nil {
		return
	}
	s.mu.Lock()
	if p := s.nativePlugins.session; p != nil && p.owner == owner {
		p.selectedView = handle
	}
	s.mu.Unlock()
}
func (s *mainState) clearPluginConversationView() {
	s.selectBuiltinPluginConversationView("chat")
}

func (s *mainState) selectBuiltinPluginConversationView(id string) {
	if owner := s.currentNativePluginSessionOwner(); owner != nil {
		_ = owner.SelectConversationView(owner.Snapshot(), id)
	}
	s.mu.Lock()
	if p := s.nativePlugins.session; p != nil {
		p.selectedView = ""
	}
	s.mu.Unlock()
}

// Render only the active conversation body; native header and composer remain.
func (s *mainState) renderPluginConversationView(c *ui.Context) bool {
	owner := s.currentNativePluginSessionOwner()
	if owner == nil {
		return false
	}
	s.mu.Lock()
	selected := s.nativePlugins.session.selectedView
	s.mu.Unlock()
	if selected == "" || selected == "chat" {
		return false
	}
	snapshot := owner.Snapshot()
	found := false
	for _, view := range snapshot.ConversationViews {
		if view.Handle == selected {
			found = true
			break
		}
	}
	if !found {
		s.clearPluginConversationView()
		return false
	}
	if !owner.RenderNativeSlot(c, "conversation.view", "", -1) {
		ui.Text(c, resumeText("正在加载会话视图…", "Loading conversation view…")).FillWidth()
	}
	return true
}

func (s *mainState) pluginRightPaneTabs() []pluginui.RightPaneTab {
	if owner := s.currentNativePluginSessionOwner(); owner != nil {
		return owner.Snapshot().RightPaneTabs
	}
	return nil
}
func (s *mainState) selectPluginRightPaneTab(handle string) {
	if owner := s.currentNativePluginSessionOwner(); owner != nil {
		_ = owner.SelectRightPaneTab(owner.Snapshot(), handle)
	}
}
func (s *mainState) renderPluginRightPaneTab(c *ui.Context, handle string) bool {
	if owner := s.currentNativePluginSessionOwner(); owner != nil {
		return owner.RenderNativeOutlet(c, "sidebar.right.pane.tab", handle)
	}
	return false
}

func (s *mainState) closePluginRightPaneTab(handle string) {
	if owner := s.currentNativePluginSessionOwner(); owner != nil {
		_ = owner.CloseRightPaneTab(owner.Snapshot(), handle)
	}
}
