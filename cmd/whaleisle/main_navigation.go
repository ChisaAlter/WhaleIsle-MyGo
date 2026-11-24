package main

import (
	"context"
	"strings"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
)

func connectionUnavailableMessage(bootstrapError string, offline, lost bool) string {
	if bootstrapError != "" {
		return bootstrapError
	}
	if offline && lost {
		return i18n.T(i18n.KeyCmdOfflineWaitingForNetworkAvailability)
	}
	return ""
}

// Main panel selection preserves marketplace, scheduler, assistant and
// per-session composer drafts while retiring the previous page's surface.
func (s *mainState) selectMainPanel(panel string) {
	s.navigateMainRoute(panelMainRoute(panel))
	s.mu.Lock()
	set := s.settings
	s.mu.Unlock()
	if panel == "plugins" && set != nil {
		set.RefreshDescribe(context.Background())
	}
}

// Package management is a distinct destination from the featured marketplace.
// Reuse the normal exclusive-page transition while retaining page drafts.
func (s *mainState) openNativePluginManager() {
	s.selectMainPanel("plugins")
}

// Global session actions reveal the conversation even from exclusive pages.
// Drafts and each page's stores remain resident; presentation resources do not.
func (s *mainState) revealMainConversation() {
	s.navigateMainRoute(routeConversation)
}

// withMainNavigation is the stable parent of both chat and the three native
// global panels. Settings overlay this stable shell; guest/PDF views bypass it.
func (s *mainState) withMainNavigation(c *ui.Context, enabled bool, m frame.Model, content func()) {
	m.PageShortcutBindings = s.pageShortcutBindings()
	m.SessionSearchFocus = &s.commandSearchFocus
	m.OnOpenWorkspaceFiles = func() {
		s.selectMainPanel("")
		s.mu.Lock()
		s.filesPaneOpen = true
		s.rightPaneTab = "files"
		s.mu.Unlock()
	}
	if !enabled {
		content()
		return
	}
	uFrame := s.updaterCenter().FrameSnapshot()
	assistantOpen := s.assistantService().Open()
	s.mu.Lock()
	win := s.win
	m.ConnectionReady = s.bootStatus.State == harness.Ready && s.dshMuxReady
	m.ConnectionError = connectionUnavailableMessage(s.dshBootstrapError, s.dshNetworkOffline, s.dshMuxLost)
	m.ConnectionConnecting = s.bootStatus.State == harness.Starting || s.bootStatus.State == harness.Ready && !s.dshMuxReady && s.dshBootstrapError == "" && (!s.dshNetworkOffline || !s.dshMuxLost)
	m.ConnectionEpoch = uint64(s.bootStatus.Generation)
	m.DesktopUpdateInstalling = uFrame.Installing
	m.DesktopUpdatePhase, m.DesktopUpdateDetail = uFrame.Phase, uFrame.Detail
	if m.OnReconnect == nil {
		m.OnReconnect = s.requestConnectionReconnect
	}
	m.RightPaneOpen = s.filesPaneOpen && !s.marketOpen && !s.scheduleOpen && !assistantOpen
	m.ActivePanel = ""
	switch {
	case s.marketOpen:
		m.ActivePanel = "plugins"
	case s.scheduleOpen:
		m.ActivePanel = "schedule"
	case assistantOpen:
		m.ActivePanel = "assistant"
	}
	if m.Store == nil {
		if s.sessions == nil {
			s.replaceSessionsLocked(session.New())
		}
		m.Store = s.sessions
	}
	s.mu.Unlock()
	s.bindShellParity(c, &m, win)
	sessions := m.Store
	if m.ActivePanel == "" && sessions != nil {
		snap := sessions.Published().Value
		if strings.HasPrefix(snap.SelectedID, "session-whale-") {
			m.ActivePanel = "assistant"
		} else if snap.SelectedID != "" {
			for _, it := range snap.Items {
				if it.SessionID == snap.SelectedID && it.Presentation != nil && it.Presentation.Owner == "dsh-whale:assistant" {
					m.ActivePanel = "assistant"
					break
				}
			}
		}
	}
	m.ScheduleStore = s.schedule
	m.OnOpenScheduleTask = s.openScheduleTask
	m.OnOpenScheduleSession = s.openScheduleSession
	m.OnScheduleInvalidate = func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	s.bindProjectModel(win, &m)
	s.bindProjectNativeConsumers(win, &m.Projects)
	projectOpen := m.Projects.OnOpen
	m.Projects.OnOpen = func(id string) {
		s.selectMainPanel("")
		if projectOpen != nil {
			projectOpen(id)
		}
	}
	m.OnOpenDeliverable = func(path string) { s.queueNavigationFile(sessions, path) }
	m.OnSelectSession = func(id string) {
		s.mu.Lock()
		s.resumePendingSession = ""
		s.mu.Unlock()
		s.selectMainPanel("")
		sessions.Select(id)
		retireStaleInlineVisualPages()
	}
	m.OnCreateSession = func() { s.openGlobalWorkspaceSession(win, sessions) }
	m.DirectoryActions = s.projectDirectoryActions(win)
	m.OnCreateWorkspaceSession = func(generation uint64, id string) {
		s.openWorkspaceSession(win, sessions, generation, id)
	}
	m.OnOpenMarket = func() { s.selectMainPanel("plugins") }
	m.SidebarPlugins = s.sidebarPluginEntries(i18n.Locale())
	m.OnOpenSidebarPlugin = func(id string) { s.openInstalledPluginViewWithLabel(id, s.sidebarPluginLabel(id)) }
	if m.OnCreatePluginDraft == nil && sessions != nil {
		m.OnCreatePluginDraft = func(region string) { s.startCreatorPluginDraft(win, sessions, region) }
	}
	m.OnOpenSchedule = func() { s.selectMainPanel("schedule") }
	m.OnOpenAssistant = func() { s.openWhaleFromSidebar(win) }
	s.gateAssistantNavigation(&m)
	m.OnOpenDesktopSettings = s.openWorkbenchSettings
	m.OnOpenDesktopUpdate = func() { s.openNativeUpdateCenter(win) }
	m.Account.OnOpen = s.openAccountSettings
	m.OnOpenSettings = func() {
		s.mu.Lock()
		s.setMainRouteLocked(routeSettings)
		s.settingsEmbeddedSection = ""
		s.settingsDesktopActive = true
		s.seedDesktopDraft()
		settings := s.settings
		s.mu.Unlock()
		s.syncMainRouteResources()
		if settings != nil {
			settings.RefreshDescribe(context.Background())
		}
	}
	m.OnNoDirectory = func() {
		s.selectMainPanel("")
		snap := sessions.SelectionSummary()
		s.mu.Lock()
		draft := s.composerDraftLocked(snap.SelectedID)
		s.mu.Unlock()
		s.openNoDirectory(win, sessions, draft, snap.SelectedID)
	}
	m.OnBrowseWorkspace = func() { s.selectMainPanel(""); s.openDirectoryPicker(sessions, win, "workspace") }
	m.OnRevealWorkspace = func() {
		root := strings.TrimSpace(m.WorkspaceRoot)
		if root != "" {
			s.revealInFileManager(win, root)
		}
	}
	m.WorkspaceRenameDraft = &s.workspaceRenameDraft
	m.WorkspaceMenuOpen = &s.workspaceMenuOpen
	m.OnRenameWorkspace = func(title string) {
		id := sessions.WorkspaceID()
		if id == "" {
			sessions.SetLastAction(i18n.T(i18n.KeyCmdWorkspaceRenameIDMissing))
			return
		}
		sessions.RenameWorkspace(context.Background(), id, title)
	}
	m.OnDeleteWorkspace = func() {
		id := sessions.WorkspaceID()
		if id == "" {
			sessions.SetLastAction(i18n.T(i18n.KeyCmdWorkspaceDeleteIDMissing))
			return
		}
		sessions.DeleteWorkspace(context.Background(), id)
	}
	m.OnMoveSessionBefore = func(id, before string) {
		sessions.InsertSessionBefore(context.Background(), sessions.WorkspaceID(), id, before)
	}
	// The old chat-only writeback is skipped by panel returns. Persist the same
	// shared rail preferences/inputs there too, without overwriting pane heights.
	defer func() {
		s.mu.Lock()
		s.layout.Sidebar, s.layout.SidebarExpanded = m.Sizes.Sidebar, m.Sizes.SidebarExpanded
		s.layout.Responsive = m.Sizes.Responsive
		if m.WorkspacePage {
			s.layout.Main = m.Sizes.Main
		}
		s.sessionSearch = *m.SessionSearch
		s.renameSessionID, s.renameDraft = *m.RenameSessionID, *m.RenameDraft
		s.mu.Unlock()
		sessions.UpdateSessionSearch(context.Background(), *m.SessionSearch)
	}()
	frame.NavigationShell(c, s.navigationFileDropModel(m), content)
}

// A Project reference selected from the persistent rail is delivered to the
// existing chat file callback. That callback retains confinement, dirty tabs,
// Office/PDF handling and Host fallback; no second file-opening path is added.
type navigationFileRequest struct {
	path, sessionID, origin, cookie string
	generation                      int
	store                           *session.Store
}

func (s *mainState) queueNavigationFile(sessions *session.Store, path string) {
	if sessions == nil {
		return
	}
	s.mu.Lock()
	host := s.bootStatus
	s.navigationFile = &navigationFileRequest{path: path, sessionID: sessions.SelectedSessionID(), origin: host.Endpoint.Origin, cookie: host.Endpoint.Cookie, generation: host.Generation, store: sessions}
	s.mu.Unlock()
	s.selectMainPanel("")
}
func (s *mainState) consumeNavigationFile(m frame.Model) {
	s.mu.Lock()
	request := s.navigationFile
	s.navigationFile = nil
	host := s.bootStatus
	same := request != nil && request.store == s.sessions && request.store == m.Store && request.generation == host.Generation && request.origin == host.Endpoint.Origin && request.cookie == host.Endpoint.Cookie
	s.mu.Unlock()
	if same && m.OnOpenDeliverable != nil && m.Store.SelectedSessionID() == request.sessionID {
		m.OnOpenDeliverable(request.path)
	}
}
