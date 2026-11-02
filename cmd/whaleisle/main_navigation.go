package main

import (
	"context"
	"strings"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/launcher"
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

// Main panel selection changes visibility only. The marketplace, scheduler,
// assistant and per-session composer stores remain alive with their drafts.
func (s *mainState) selectMainPanel(panel string) {
	s.mu.Lock()
	s.marketOpen = panel == "market" || panel == "plugins"
	s.marketManager = panel == "plugins"
	s.scheduleOpen = panel == "schedule"
	s.assistantConversationGeneration++
	s.assistantOpen = panel == "assistant"
	s.toolsMenuOpen = false
	win := s.win
	set := s.settings
	s.mu.Unlock()
	if panel == "plugins" && set != nil {
		set.RefreshDescribe(context.Background())
	}
	if win != nil {
		win.Update(func() {})
	}
}

// Global session actions reveal the conversation even from exclusive pages.
// Only route flags change: drafts and each page's live stores remain resident.
func (s *mainState) revealMainConversation() {
	s.mu.Lock()
	s.launcherOpen = false
	s.settingsOpen = false
	s.settingsEmbeddedSection = ""
	s.usageOpen = false
	s.imOpen = false
	s.remoteOpen = false
	s.webviewOpen = false
	s.browserPreviewOpen = false
	s.marketOpen = false
	s.scheduleOpen = false
	s.assistantOpen = false
	s.assistantConversationGeneration++
	s.toolsMenuOpen = false
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

// gateAssistantNavigation follows the desktop Whale overlay preference, not Host
// readiness. overlay.optionalPlugins mounts Whale even during SkipUserPlugins;
// OfflinePlugins.ApplyDisabled also preserves this immutable built-in. Neither
// recovery nor a stale disabled-list entry removes its navigation capability.
func (s *mainState) gateAssistantNavigation(m *frame.Model) {
	s.mu.Lock()
	store := s.cfgStore
	s.mu.Unlock()
	if store == nil || !store.Get().WhaleAssistantEnabled {
		m.OnOpenAssistant = nil
	}
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
	s.mu.Lock()
	win := s.win
	m.ConnectionReady = s.bootStatus.State == harness.Ready && s.dshMuxReady
	m.ConnectionError = connectionUnavailableMessage(s.dshBootstrapError, s.dshNetworkOffline, s.dshMuxLost)
	m.ConnectionConnecting = s.bootStatus.State == harness.Starting || s.bootStatus.State == harness.Ready && !s.dshMuxReady && s.dshBootstrapError == "" && (!s.dshNetworkOffline || !s.dshMuxLost)
	m.ConnectionEpoch = uint64(s.bootStatus.Generation)
	m.DesktopUpdateInstalling = s.nativeReleases.installing
	m.DesktopUpdatePhase, m.DesktopUpdateDetail = s.nativeReleases.phase, strings.TrimSpace(s.nativeReleases.status+" "+s.nativeReleases.err)
	if m.OnReconnect == nil {
		m.OnReconnect = s.requestConnectionReconnect
	}
	m.RightPaneOpen = s.filesPaneOpen && !s.marketOpen && !s.scheduleOpen && !s.assistantOpen
	m.ActivePanel = ""
	switch {
	case s.marketOpen:
		m.ActivePanel = "plugins"
	case s.scheduleOpen:
		m.ActivePanel = "schedule"
	case s.assistantOpen:
		m.ActivePanel = "assistant"
	}
	if m.Store == nil {
		if s.sessions == nil {
			s.sessions = session.New()
		}
		m.Store = s.sessions
	}
	s.mu.Unlock()
	sessions := m.Store
	m.ScheduleStore = s.schedule
	m.OnOpenScheduleTask = s.openScheduleTask
	m.OnOpenScheduleSession = s.openScheduleSession
	m.OnScheduleInvalidate = func() {
		if win != nil {
			win.Update(func() {})
		}
	}
	s.bindProjectModel(win, &m)
	projectOpen := m.Projects.OnOpen
	m.Projects.OnOpen = func(id string) {
		s.selectMainPanel("")
		if projectOpen != nil {
			projectOpen(id)
		}
	}
	m.OnOpenDeliverable = func(path string) { s.queueNavigationFile(sessions, path) }
	m.OnSelectSession = func(id string) { s.selectMainPanel(""); sessions.Select(id) }
	m.OnCreateSession = func() { s.selectMainPanel(""); sessions.CreateSession(context.Background()) }
	m.OnCreateWorkspaceSession = func(generation uint64, id string) {
		s.openWorkspaceSession(win, sessions, generation, id)
	}
	m.OnOpenMarket = func() { s.selectMainPanel("plugins") }
	m.OnOpenSchedule = func() { s.selectMainPanel("schedule") }
	m.OnOpenAssistant = func() { s.selectMainPanel("assistant") }
	s.gateAssistantNavigation(&m)
	m.OnOpenLauncher = func() {
		s.selectMainPanel("")
		s.mu.Lock()
		s.launcherOpen = true
		s.launcherTab = launcher.TabHome
		s.mu.Unlock()
	}
	openLauncher := m.OnOpenLauncher
	m.OnOpenDesktopUpdate = func() { openLauncher(); s.mu.Lock(); s.launcherTab = launcher.TabVersions; s.mu.Unlock() }
	m.Account.OnOpen = s.openAccountSettings
	m.OnOpenSettings = func() {
		s.mu.Lock()
		s.settingsEmbeddedSection = ""
		s.settingsOpen = true
		s.settingsDesktopActive = true
		s.seedDesktopDraft()
		settings := s.settings
		s.mu.Unlock()
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
			_ = desktopcomponents.OpenInFileManager(root)
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
	s.bindPluginSessionNavigation(&m)
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
