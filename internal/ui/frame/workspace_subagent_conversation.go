package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/agents"
)

// A transient addressed route replaces only the conversation surface. The root
// Store stays selected on the parent; no child is promoted to a top-level Session.
// Never export/restore a live route, prompt draft or child interaction authority.
type mainChildConversation struct {
	Pane    *WorkspacePane
	Owner   *session.Store
	Binding string
}

func (w *WorkspacePanes) closeMainChild() {
	if w == nil || w.conversation == nil {
		return
	}
	w.conversation.Owner.EndMainConversationChildView()
	w.conversation.Pane.releaseAgentPane()
	w.conversation = nil
}
func (w *WorkspacePanes) currentMainChild(m Model) *WorkspacePane {
	if w == nil || w.conversation == nil {
		return nil
	}
	route := w.conversation
	if m.Store == nil || route.Owner != m.Store || route.Pane.SubagentChat == nil {
		w.closeMainChild()
		return nil
	}
	snap := m.Store.Snapshot()
	if snap.SelectedID != route.Pane.SubagentChat.Root || snap.PlanHistoryBinding != route.Binding {
		w.closeMainChild()
		return nil
	}
	return route.Pane
}
func subagentConversationOpener(m Model) func(uint64, session.AgentSessionAddress) {
	return func(token uint64, address session.AgentSessionAddress) {
		if m.Store == nil || m.Sizes == nil || m.Sizes.Workspace == nil {
			return
		}
		source := m.AgentNavigationStore
		if source == nil {
			source = m.Store
		}
		root, title, ok := source.AgentChatTarget(token, address)
		if !ok {
			return
		}
		w := m.Sizes.Workspace
		n := w.leaf("subagentchat")
		n.SubagentChat = &SubagentChatPaneState{Root: root, Address: address, Title: title, Lineage: source.AgentChatRestoreLineage(token, address)}
		local := w.agentPaneModel(m, n)
		// Revalidate the exact source occurrence/token after creating the destination.
		if _, _, valid := source.AgentChatTarget(token, address); !valid || !local.AgentNavigationStore.OpenAgentChatFromCatalog(source, token, root, address, title) {
			n.releaseAgentPane()
			return
		}
		w.closeMainChild()
		w.conversation = &mainChildConversation{Pane: n, Owner: m.Store, Binding: m.Store.Snapshot().PlanHistoryBinding}
		m.Store.MarkMainConversationViewed(address.ChildSessionID)
	}
}

func mainSubagentConversation(c *ui.Context, m Model) bool {
	if m.Sizes == nil || m.Sizes.Workspace == nil {
		return false
	}
	w := m.Sizes.Workspace
	n := w.currentMainChild(m)
	if n == nil {
		return false
	}
	local := w.agentPaneModel(m, n)
	// The main occurrence is outside the resource tree; opening its documents
	// must not make an unknown ID the active resource-dock tab.
	if open := m.OnOpenAgentPaneDocument; open != nil {
		local.OnOpenAgentDocument = func(token uint64, address session.AgentSessionAddress, path string) {
			open(n.ID, local.AgentNavigationStore, token, address, path)
		}
	}
	target := n.SubagentChat
	ui.Column(c.Key(n.ID)).Fill().MinHeight(0).Children(func() {
		ui.Row(c).FillWidth().Padding(10).Gap(8).Children(func() {
			if ui.Button(c, paneLabel("返回父会话", "Back to parent conversation")).Clicked() {
				w.closeMainChild()
			}
			ui.Text(c, target.Root+" → "+target.Address.ChildSessionID).Grow(1).Selectable()
		})
		if w.conversation == nil {
			return
		}
		navigation := local.AgentNavigationStore
		v := navigation.AgentNavigationSnapshot()
		if v.Address != target.Address {
			ui.Text(c, paneLabel("子会话已暂停，重新打开后继续。", "Child conversation paused. Reopen to continue."))
			directRoot := target.Address.ParentSessionID == target.Root
			if !directRoot {
				ui.Text(c, paneLabel("请从来源会话当前展开的子代理目录打开此聊天。", "Open this chat from the currently expanded child catalog in the source session."))
			}
			if directRoot && ui.Button(c, paneLabel("重新打开聊天", "Reopen chat")).Clicked() {
				navigation.OpenAgentChat(target.Root, target.Address, target.Title)
			}
			return
		}
		agents.View(c, agents.Model{Store: navigation, Snapshot: m.Store.Snapshot(), ChatOnly: true, MainConversation: true,
			Interactions: m.Interactions, OnOpenPlanResource: m.OnOpenPlanResource,
			ThemeID: m.ThemeID, SystemDark: m.SystemDark, PromptDraft: local.AgentsPromptDraft,
			OnOpenURL: m.OnOpenCitationURL, OnOpenDocument: local.OnOpenAgentDocument,
			OnClose: func() { w.closeMainChild() },
		})
	})
	return true
}

// An addressed child has no top-level rename/fork/archive semantics. Do not let
// those page shortcuts unexpectedly act on the hidden parent conversation.
func guardMainChildCommands(m Model, actions map[string]func()) {
	if m.Sizes == nil || m.Sizes.Workspace == nil || m.Sizes.Workspace.currentMainChild(m) == nil {
		return
	}
	for _, id := range []string{"session.rename", "session.fork", "session.archive"} {
		delete(actions, id)
	}
}
