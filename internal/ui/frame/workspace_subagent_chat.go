package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/agents"
)

// Persist routing facts only. Restoring/splitting never restores a live reader,
// draft, prompt token or a child's pending approval/question capabilities.
type SubagentChatPaneState struct {
	Lineage []session.AgentSessionAddress `json:"lineage,omitempty"`
	Root    string                        `json:"root"`
	Address session.AgentSessionAddress   `json:"address"`
	Title   string                        `json:"title"`
}

func copySubagentChatPane(target, source *WorkspacePane) {
	if source.Kind == "subagentchat" && source.SubagentChat != nil {
		copy := *source.SubagentChat
		copy.Lineage = append([]session.AgentSessionAddress(nil), source.SubagentChat.Lineage...)
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
			n.SubagentChat = &SubagentChatPaneState{Root: root, Address: address, Title: title, Lineage: source.AgentChatRestoreLineage(token, address)}
			w.placeDockResource(n, w.Root.find(w.Active), true)
			w.Active, w.route, w.Fullscreen = n.ID, n.Kind, ""
			local := w.agentPaneModel(m, n)
			local.AgentNavigationStore.OpenAgentChatFromCatalog(source, token, root, address, title)
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
	// A restored routing record acquires a fresh addressed reader, never an old
	// token or prompt authority. Direct-root records may be reopened from this
	// pane. Nested legacy records have no saved ancestry proof and must be
	// opened from a currently expanded source catalog instead.
	directRoot := target.Address.ParentSessionID == target.Root
	if v.Address != target.Address && directRoot && !navigation.HasRetainedAgentRead() && m.Store.SelectedSessionID() == target.Root {
		if navigation.OpenAgentChat(target.Root, target.Address, target.Title) {
			v = navigation.AgentNavigationSnapshot()
		}
	}
	if v.Address != target.Address {
		ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
			ui.Text(c, subagentChatPaneTitle(m))
			ui.Text(c, target.Address.ParentSessionID+" → "+target.Address.ChildSessionID+" · "+target.Address.Mode).Selectable()
			ui.Text(c, paneLabel("聊天已暂停。重新打开只会读取此子会话。", "Chat is paused. Reopen to read this child session."))
			sameRoot := m.Store.SelectedSessionID() == target.Root
			if !sameRoot {
				ui.Text(c, paneLabel("请先切回来源主会话：", "Return to the source session first: ")+target.Root)
			}
			if !directRoot {
				ui.Text(c, paneLabel("请从来源会话当前展开的子代理目录打开此聊天。", "Open this chat from the currently expanded child catalog in the source session."))
			}
			if directRoot && ui.Button(c, paneLabel("重新打开聊天", "Reopen chat")).Disabled(!sameRoot).Clicked() {
				navigation.OpenAgentChat(target.Root, target.Address, target.Title)
			}
		})
		return
	}
	if v.RetainedReadOnly {
		ui.Text(c, paneLabel("已保留来源会话；明确发送、停止或回复时会重新验证原目标。", "Source session retained; sending, stopping, or replying revalidates the original target."))
		if !directRoot {
			ui.Text(c, paneLabel("嵌套聊天的重新授权必须从来源会话当前展开的子代理目录发起。", "Reauthorize a nested chat from the currently expanded child catalog in the source session."))
		}
		if directRoot && m.Store.SelectedSessionID() == target.Root && ui.Button(c, paneLabel("重新授权聊天操作", "Reauthorize chat actions")).Clicked() {
			navigation.OpenAgentChat(target.Root, target.Address, target.Title)
		}
	}
	agents.View(c, agents.Model{Store: navigation, Snapshot: navigation.RenderSnapshot(), ChatOnly: true,
		Interactions: m.Interactions, OnOpenPlanResource: m.OnOpenPlanResource,
		ThemeID: m.ThemeID, SystemDark: m.SystemDark, PromptDraft: m.AgentsPromptDraft,
		OnOpenURL: m.OnOpenCitationURL, OnOpenDocument: m.OnOpenAgentDocument,
		OnClose: func() { m.Sizes.Workspace.pending = func() { m.Sizes.Workspace.close(n.ID) } },
	})
}
