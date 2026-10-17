package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/agents"
)

// Persist routing facts only. Restoring/splitting never restores a live reader,
// draft, prompt token or a child's pending approval/question capabilities.
type SubagentChatPaneState struct {
	Root    string                      `json:"root"`
	Address session.AgentSessionAddress `json:"address"`
	Title   string                      `json:"title"`
}

func copySubagentChatPane(target, source *WorkspacePane) {
	if source.Kind == "subagentchat" && source.SubagentChat != nil {
		copy := *source.SubagentChat
		target.SubagentChat = &copy
	}
}
func subagentChatOpener(m Model) func(uint64, session.AgentSessionAddress) {
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
		w.pending = func() {
			// Recheck the source after deferred layout mutation; stale clicks cannot
			// outlive a root/Host switch or closed catalog occurrence.
			if _, _, current := source.AgentChatTarget(token, address); !current {
				return
			}
			n := w.leaf("subagentchat")
			n.SubagentChat = &SubagentChatPaneState{Root: root, Address: address, Title: title}
			if w.Root == nil {
				w.Root = n
			} else {
				current := w.Root.find(w.Active)
				if current == nil {
					current = w.Root.firstLeaf()
				}
				old := *current
				*current = WorkspacePane{ID: w.leaf("").ID, Axis: "horizontal", Ratio: 0.5, First: &old, Second: n}
			}
			w.Active, w.route, w.Fullscreen = n.ID, n.Kind, ""
			local := w.agentPaneModel(m, n)
			local.AgentNavigationStore.OpenAgentChat(root, address, title)
		}
	}
}
func subagentChatPaneTitle(m Model) string {
	if m.workspacePane != nil && m.workspacePane.SubagentChat != nil && m.workspacePane.SubagentChat.Title != "" {
		return m.workspacePane.SubagentChat.Title
	}
	return paneLabel("子代理聊天", "Child chat")
}
func subagentChatPane(c *ui.Context, m Model) {
	n := m.workspacePane
	if n == nil || n.SubagentChat == nil || m.Store == nil || m.AgentNavigationStore == nil {
		ui.Text(c, paneLabel("请从子代理目录打开聊天。", "Open a chat from the child catalog."))
		return
	}
	target := n.SubagentChat
	navigation := m.AgentNavigationStore
	v := navigation.AgentNavigationSnapshot()
	if v.Address != target.Address {
		ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
			ui.Text(c, subagentChatPaneTitle(m))
			ui.Text(c, target.Address.ParentSessionID+" → "+target.Address.ChildSessionID+" · "+target.Address.Mode).Selectable()
			ui.Text(c, paneLabel("聊天已暂停。重新打开只会读取此子会话。", "Chat is paused. Reopen to read this child session."))
			sameRoot := m.Store.Snapshot().SelectedID == target.Root
			if !sameRoot {
				ui.Text(c, paneLabel("请先切回来源主会话：", "Return to the source session first: ")+target.Root)
			}
			if ui.Button(c, paneLabel("重新打开聊天", "Reopen chat")).Disabled(!sameRoot).Clicked() {
				navigation.OpenAgentChat(target.Root, target.Address, target.Title)
			}
		})
		return
	}
	agents.View(c, agents.Model{Store: navigation, Snapshot: m.Store.Snapshot(), ChatOnly: true,
		Interactions: m.Interactions, OnOpenPlanResource: m.OnOpenPlanResource,
		ThemeID: m.ThemeID, SystemDark: m.SystemDark, PromptDraft: m.AgentsPromptDraft,
		OnOpenURL: m.OnOpenCitationURL, OnOpenDocument: m.OnOpenAgentDocument,
		OnClose: func() { m.Sizes.Workspace.pending = func() { m.Sizes.Workspace.close(n.ID) } },
	})
}
