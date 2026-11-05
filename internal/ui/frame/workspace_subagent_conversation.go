package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/agents"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/projects"
	"whaleisle/internal/ui/theme"
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
	if m.Store.SelectedSessionID() != route.Pane.SubagentChat.Root || m.Store.PlanHistoryBinding() != route.Binding {
		w.closeMainChild()
		return nil
	}
	return route.Pane
}

// MainConversationChildActive follows the same visual route as the main header.
// Stale child routes are retired by currentMainChild before a parent feature may
// attach to the newly visible conversation.
func MainConversationChildActive(m Model) bool {
	return m.Sizes != nil && m.Sizes.Workspace != nil && m.Sizes.Workspace.currentMainChild(m) != nil
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
		w.conversation = &mainChildConversation{Pane: n, Owner: m.Store, Binding: m.Store.PlanHistoryBinding()}
		m.Store.MarkMainConversationViewed(address.ChildSessionID)
	}
}

func mainSubagentConversation(c *ui.Context, m Model) bool {
	if m.Sizes == nil || m.Sizes.Workspace == nil || m.Sizes.Workspace.currentMainChild(m) == nil {
		return false
	}
	ui.Column(c).Key("main-child-conversation-whole").Fill().MinHeight(0).Children(func() {
		mainSubagentConversationHeader(c, m)
		mainSubagentConversationBody(c, m)
	})
	return true
}

// mainSubagentConversationHeader owns only the addressed child's row-1 chrome.
// Its presence suppresses the hidden parent's title and view tabs.
func mainSubagentConversationHeader(c *ui.Context, m Model, rightInset ...float32) bool {
	if m.Sizes == nil || m.Sizes.Workspace == nil {
		return false
	}
	w := m.Sizes.Workspace
	n := w.currentMainChild(m)
	if n == nil {
		return false
	}
	target := n.SubagentChat
	paddingRight := float32(20)
	if len(rightInset) > 0 {
		paddingRight = rightInset[0]
	}
	ui.Row(c).Key(n.ID+":header").FillWidth().MinHeight(workspaceCaptionContent).Padding(0, paddingRight, 0, 20).Gap(8).DragWindow().Children(func() {
		if ui.Button(c, paneLabel("返回父会话", "Back to parent conversation")).Clicked() {
			w.closeMainChild()
		}
		source := m.AgentNavigationStore
		if source == nil {
			source = m.Store
		}
		if source != nil {
			view := source.AgentNavigationSnapshot()
			ancestors, siblings := source.AgentConversationRoutes(view.Token, target.Address)
			if len(ancestors) > 0 {
				ui.Button(c, paneLabel("上级会话", "Ancestors")).Menu(func(menu *ui.Menu) {
					for _, ancestor := range ancestors {
						ancestor := ancestor
						label := ancestor.Label
						if label == "" {
							label = ancestor.Address.ChildSessionID
						}
						if menu.Item(label).Chosen() {
							subagentConversationOpener(m)(view.Token, ancestor.Address)
						}
					}
				})
			}
			if len(siblings) > 1 {
				ui.Button(c, paneLabel("同级会话", "Siblings")).Menu(func(menu *ui.Menu) {
					for _, sibling := range siblings {
						sibling := sibling
						if menu.Item(sibling.Label).Disabled(sibling.Address == target.Address).Chosen() {
							subagentConversationOpener(m)(view.Token, sibling.Address)
						}
					}
				})
			}
		}
		ui.Text(c, target.Root+" → "+target.Address.ChildSessionID).Grow(1).MinWidth(0).NoWrap().Ellipsis("…")
		colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
		conversationTeamAction(c, m, renderSnapshot(m), m.Sizes.Center-20-paddingRight <= 560, chrome.Color(colors.Muted))
	})
	return true
}

// mainSubagentConversationBody builds the addressed child exactly once in the
// shared content row, without rebuilding either the child or parent header.
func mainSubagentConversationBody(c *ui.Context, m Model) bool {
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
		agents.View(c, agents.Model{
			ChatPreferences: m.ChatPreferences, Store: navigation, Snapshot: renderSnapshot(m), ChatOnly: true, MainConversation: true,
			InteractionRenderSlot: m.InteractionRenderSlot,
			ChildDraft:            m.ChildDraft,
			BusyEnter:             m.ChildBusyEnter,
			ComposerOverride: func(ctx *ui.Context, address session.AgentSessionAddress) bool {
				return projects.ConversationComposer(ctx, m.Projects, address.ChildSessionID, address.ParentSessionID)
			},
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
