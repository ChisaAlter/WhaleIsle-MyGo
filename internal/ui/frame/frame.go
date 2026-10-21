// Package frame is the M4 main shell: resizable sidebar, chat, optional
// right and bottom regions.
package frame

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/store/interaction"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/agents"
	"whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/i18n"
	interui "whaleisle/internal/ui/interaction"
	"whaleisle/internal/ui/planpreview"
	"whaleisle/internal/ui/projects"
	remoteui "whaleisle/internal/ui/remote"
	costui "whaleisle/internal/ui/sessioncost"
	"whaleisle/internal/ui/sidebar"
	"whaleisle/internal/ui/svgimage"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

// LayoutSizes persist split pane dimensions (DIPs) across frames.
type LayoutSizes struct {
	TranscriptWidth   float32 // local message-body preference; independent of pane and composer widths
	transcriptGesture conversationWidthGesture
	dragging          bool               // transient parent-divider gesture; child tracks must not ease behind it
	Responsive        *SidebarResponsive // transient; never persisted as a width preference
	Workspace         *WorkspacePanes
	Sidebar           float32
	SidebarExpanded   float32 // last expanded preference; Sidebar=0 is the compact rail
	Center            float32 // resolved chat width; not a saved right-pane preference
	viewportW         float32 // viewport the columns were resolved against
	Right             float32 // remembered right width, 0 = first-open viewport default
	Main              float32 // height of the dialogue column when the terminal is open
	Bottom            float32 // >40 shows the terminal (saved height preference when hidden)
	RightTree         float32 // height of file tree above preview in the right pane
}

// Desktop ui-layout columns.ts contract (anywhere-labs harness-desktop).
const (
	SidebarMin          = float32(264)
	SidebarMax          = float32(420)
	SidebarDefault      = float32(280)
	SidebarCollapsed    = float32(56)
	CenterMin           = float32(400) // protected while details is open
	DetailsMin          = float32(300)
	DetailsDefaultRatio = float32(0.45)
	DetailsMaxRatio     = float32(0.70)
	SplitterWidth       = float32(1) // MyGo Split reserves one DIP per divider
)

// Columns is the resolved three-column geometry for one frame.
type Columns struct {
	Sidebar float32
	Center  float32
	Details float32 // 0 = visually closed (preference may still be open)
}

// ComputeColumns mirrors the current ui-layout columns.ts three-track contract.
// It is pure: concessions never change the saved right-pane preference.
func ComputeColumns(viewport, sidebarPref, detailsPref float32) Columns {
	return computeColumns(viewport, sidebarPref, detailsPref, viewport)
}

func computeColumns(available, sidebarPref, detailsPref, viewport float32) Columns {
	s := clampSidebarWidth(sidebarPref)
	if sidebarPref <= 0 {
		s = SidebarCollapsedWidth()
	}
	if detailsPref <= 0 {
		return Columns{Sidebar: s, Center: maxf(0, available-s)}
	}
	d0 := clampDetailsWidth(detailsPref, viewport)
	if s+d0+CenterMin <= available {
		return Columns{Sidebar: s, Center: available - s - d0, Details: d0}
	}
	d1 := maxf(DetailsMin, available-s-CenterMin)
	if s+d1+CenterMin <= available {
		return Columns{Sidebar: s, Center: CenterMin, Details: d1}
	}
	return Columns{Sidebar: s, Center: maxf(0, available-s)}
}

func clampDetailsWidth(width, viewport float32) float32 {
	return min(maxf(DetailsMin, float32(math.Round(float64(width)))), maxf(DetailsMin, viewport*DetailsMaxRatio))
}

// DefaultDetailsWidth is the first-open preference, independent of concessions.
func DefaultDetailsWidth(viewport float32) float32 {
	return clampDetailsWidth(viewport*DetailsDefaultRatio, viewport)
}

// ResolveColumns includes the real MyGo dividers in the rendered geometry.
// Right is initialized only on first opening and survives shrink and close/open.
func (sizes *LayoutSizes) ResolveColumns(viewport float32, rightOpen bool) Columns {
	sizes.observeSidebarViewport(viewport)
	sizes.viewportW = viewport
	if rightOpen && !sizes.Responsive.rightOpen && sizes.Responsive.narrow {
		sizes.Responsive.expanded = false
	}
	sizes.Responsive.rightOpen = rightOpen
	sidebar := sizes.sidebarPreference()
	pref := float32(0)
	if rightOpen {
		if sizes.Right <= 0 {
			sizes.Right = DefaultDetailsWidth(viewport)
		}
		pref = sizes.Right
	}
	sidebarDivider := SplitterWidth
	if sidebar <= 0 && SidebarCollapsedWidth() == 0 {
		sidebarDivider = 0
	}
	cols := computeColumns(maxf(0, viewport-sidebarDivider-SplitterWidth), sidebar, pref, viewport)
	if cols.Details == 0 {
		// A closed right track has no divider; only the sidebar divider remains.
		cols = computeColumns(maxf(0, viewport-sidebarDivider), sidebar, 0, viewport)
	}
	sizes.Center = cols.Center
	return cols
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// DefaultLayout returns initial split sizes for a 1200脳800 window.
// SplitVertical *size is the FIRST (top) pane height 鈥?that must be Main, not Bottom.
func DefaultLayout() LayoutSizes {
	// ui-layout columns.ts: SIDEBAR_DEFAULT = 280 (min 264, max 420).
	return LayoutSizes{Sidebar: SidebarDefault, Center: 680, Main: 560, Bottom: 180, RightTree: 280}
}

// Model wires the frame to session state and layout handles.
type Model struct {
	SVGImages               *svgimage.Owner
	OpenUIParent            uintptr
	OpenUIWindow            *mygo.Window
	CustomPanelContent      func(*ui.Context)
	OnOpenDocumentViewer    func(string) (files.DocumentSurface, error)
	GroupRoom               bool
	GroupMembers            []session.GroupMember
	OnReplyGroupThread      func(string)
	DisableDeveloperTools   bool
	TitlebarPreferences     settings.TitlebarPreferences
	WorkspaceTrailing       func(*ui.Context, string)
	WorkspacePage           bool
	WorkspaceOpenInApp      *WorkspaceOpenInAppState
	HideArchivedList        bool
	ChatPreferences         settings.ChatPreferences
	ConversationPreferences settings.ConversationPreferences
	ComposerResize          *composer.ResizeState
	OnComposerResizeEnd     func(axis string, value float32)
	ScheduleStore           *schedstore.Store
	OnOpenScheduleTask      func(schedstore.Entry)
	OnOpenScheduleSession   func(schedstore.Entry)
	OnScheduleInvalidate    func()
	// PageShortcutBindings contains committed native overrides; nil uses defaults.
	PageShortcutBindings map[string]string
	SessionSearchFocus   *bool
	OnOpenWorkspaceFiles func()
	BrowserContent       func(*ui.Context)
	RemoteFilesContent   func(*ui.Context, *session.Store, *remoteui.FileWorkspaceState) bool
	NewWorkspaceBrowser  func() *browserpreview.Controller
	OnHideBrowser        func()
	// ExternalNavigation reuses NavigationShell instead of mounting a second rail.
	ExternalNavigation bool
	// ViewportW/H are the window's client area in DIPs (viewportDIP). Children
	// that clamp tooltips or popovers to the screen read them instead of
	// Context.Size, which reports device pixels on a scaled display.
	ViewportW float32
	ViewportH float32
	OnSelectSession    func(string)
	OnCreateSession    func()

	Projects                projects.Model
	// Account is the sidebar's bottom account seat (avatar, name, menu).
	Account                 sidebar.Account
	Store                   *session.Store
	AgentPresetManager      *session.AgentPresetManager
	Interactions            *interaction.Store
	OnAutoAllowChanged      func(on bool)
	Sizes                   *LayoutSizes
	ChatList                *ui.ListState
	ThemeID                 string
	SystemDark              bool
	Draft                   *string
	ComposerState           *composer.DraftState
	OnSubmitComposer        func()
	OnSubmitComposerGesture func(accelerated bool)
	OnNativeCommandAction   func(composer.NativeCommandOwner, string) bool
	OnNativeCommandChanged  func()
	OnAttachComposerFiles   func([]string)
	OnRetryComposerFile     func(int)
	OnPasteComposerImage    func(bool)
	ModelPick               *string
	PermissionPick          *string
	AgentPresetPick         *string
	ThinkingExpanded        *map[string]bool
	RenameSessionID         *string
	RenameDraft             *string
	SessionSearch           *string

	WorkspaceRoot    string
	FileBindingID    string
	FileTabs         *files.Tabs
	GitWorkbench     *diffsurface.Workbench
	FileExpanded     *map[string]bool
	SelectedFile     *string
	FileFilter       *string
	FilePreview      *string
	FilePreviewErr   *string
	FilePreviewList  *ui.ListState
	FileTreeSplit    *float32
	FileShortcuts    func() bool
	// OnOpenFilePicker backs the file strip's trailing control. Nil hides it.
	OnOpenFilePicker func()
	FileEditing      *bool
	FileEditDraft    *string
	FileEditStatus   *string
	FileEditViewMode *files.EditViewMode
	FileEditSplit    *float32
	FilePaneWidth    *float32 // resolved width for responsive file-preview layout
	RightPaneOpen    bool     // explicit visibility; Sizes.Right remains remembered when closed

	RightPaneTab            *string // "files" | "diff" | "agents"
	OnOpenAgents            func()
	AgentsPromptDraft       *string
	OnOpenAgentDocument     func(uint64, session.AgentSessionAddress, string)
	OnOpenAgentPaneDocument func(string, *session.Store, uint64, session.AgentSessionAddress, string)
	AgentNavigationStore    *session.Store
	DiffPath                *string
	DiffText                *string
	DiffErr                 *string

	ComposerSuggest         *ui.ListState
	ComposerSlash           []composer.SlashCommand
	ComposerPaths           []string
	ComposerSkillsOpen      bool
	ComposerSkillsList      *ui.ListState
	ComposerSkillsFilter    *string
	ComposerChipEdit        *composer.ChipEdit
	ComposerTextInput       *composer.DraftClient
	UseComposerTextInput    bool
	OnIndexWorkspace        func(root string) []string
	OnSearchFileRefs        func(query string) []string
	OnSearchReferences      func(query string, quoted bool) []session.ReferenceCandidate
	OnToggleComposerSkills  func(open bool)
	OnSearchSkillHub        func(query string)
	OnInstallSkillHub       func(hit session.SkillHubHit)
	OnDeleteSkill           func(name string)
	SkillCreateName         *string
	OnCreateSkill           func(name string)
	OnToggleSkillModel      func(name string, modelInvocable bool)
	OnPickComposerDirectory func()
	OnPickComposerImage     func()
	OnPickComposerSpeech    func()
	OnCycleComposerSpeech   func()

	ComposerVoice         voiceinput.Status
	OnStartComposerVoice  func()
	OnStopComposerVoice   func()
	OnCancelComposerVoice func()
	OnInsertComposerVoice func()

	// OnListWorkspaceFiles is optional workspaceFiles/list fallback for the files pane.
	OnListWorkspaceFiles             func(rel string) ([]workspacefs.Entry, error)
	OnListWorkspaceFilesContext      workspacefs.RemoteDirectoryLister
	OnWatchWorkspaceDirectoryContext workspacefs.RemoteDirectoryWatcher
	// OnReadWorkspaceFile is optional workspaceFiles/read fallback for preview.
	OnReadWorkspaceFile            func(rel string) (string, error)
	OnReadWorkspaceTextPageContext func(context.Context, string, int64) (workspacefs.TextPage, error)
	OnReadWorkspaceFileContext     func(context.Context, string) (string, error)
	OnReadWorkspaceTableContext    func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error)
	FileImageGeneration            uint64
	OnStatWorkspaceFileContext     func(context.Context, string) string
	// OnStatWorkspaceFile returns a short workspaceFiles/stat summary.
	OnStatWorkspaceFile   func(rel string) string
	FileStatNote          *string
	OnPickHeroWorkspace   func()
	WorkspacePickerAnchor *ui.Handle
	OnNoDirectory         func()
	OnSelectWorkspace     func(string)
	OnBrowseWorkspace     func() // SidebarRoot 鈽?/ folder row
	OnRevealWorkspace     func()
	OnOpenWorkspaceLocal  func()
	OnRenameWorkspace     func(title string)
	OnDeleteWorkspace     func()
	OnMoveSessionBefore   func(sessionID, beforeSessionID string)
	WorkspaceRenameDraft  *string
	WorkspaceMenuOpen     *bool
	OnOpenFloatingDoc     func(string)
	OnOpenDoc             func(absPath string, kind workspacefs.DocKind, inWebView bool)
	OnOpenMedia           func(path string, floating bool)
	OnRevealFile          func(absPath string)
	OnOpenWithApp         func(absPath, appID string)
	FileOpenWithApps      []files.OpenWithApp
	OnRenderOfficePDF     func(rel string)
	OfficePreviewStatus   func(path string) string
	OnOpenBrowserPreview  func()
	OnOpenMarket          func()
	OnOpenSchedule        func()
	OnOpenAssistant       func()
	OnOpenLauncher        func()
	OnContactAccount      func()
	OnOpenSettings        func()
	// OnOpenModelSettings opens Settings on the Models page; a failed turn offers
	// it as the fix for a missing or rejected API key.
	OnOpenModelSettings func()
	OnVoiceSettings       func()
	OnToggleTools         func()
	OnToggleTerminal      func()
	OnOpenHostTerminal    func()
	OnRestartTerminal     func()
	OnCloseTerminal       func()

	Terminal              *termui.Panel
	HostTerminalStore     *hostterminal.Store
	HostTerminalUI        *termui.HostState
	TerminalShellsNote    string
	OnAddSelectionToChat  func(string) error
	OnOpenTerminalTarget  func(string) error
	CanAddSelectionToChat bool

	ConnectionBanner                        string
	ConnectionReady                         bool
	ConnectionConnecting                    bool
	ConnectionError                         string
	ConnectionEpoch                         uint64
	DesktopUpdateInstalling                 bool
	DesktopUpdatePhase, DesktopUpdateDetail string
	OnOpenDesktopUpdate                     func()
	OnReconnect                             func()
	UsageJobsToday                          *int
	UsageBalance                            string
	SessionCost                             costui.Model
	OnOpenUsage                             func()
	EditMessageSeq                          *float64
	FeedbackCommentMsgID                    *string
	FeedbackCommentDraft                    *string
	PlanBoardOpen                           *bool
	GoalDraft                               *string
	// TodoPanelCollapsed mirrors Electron TodoPanel header chevron (composer dock).
	TodoPanelCollapsed           *bool
	ImageLightboxPath            *string
	OnOpenImageLightbox          func(displayPath string)
	OnCloseImageLightbox         func()
	OnOpenCitationURL            func(url string)
	OnOpenPlanResource           func(planpreview.Document)
	OnOpenDeliverable            func(path string)
	OnOpenHistoryReference       func(path string, complete func(error))
	VisualReply                  session.VisualReplyState
	VisualReplyBitmap            *ui.Bitmap
	OnPreviewVisualReply         func(attachmentID string)
	OnInteractVisualReply        func(attachmentID string)
	OnInlineVisualReply          func(attachmentID string) session.InlineVisualReply
	OnStopInteractiveVisualReply func()
	OnReadVisualReply            func(attachmentID string)
	OnSaveVisualReply            func(attachmentID string)
	OnCloseVisualReply           func()
	// ChatPaneTab is ConversationRoot in-column tab: "chat" (default) or "trajectory".
	ChatPaneTab *string
	// HideChatViewTabs mirrors Electron submissionPolicy.viewTabs=false.
	HideChatViewTabs bool
	// Trajectory UI state (owned by main; package fallbacks if nil).
	TrajSearch         *string
	TrajSelectedID     *string
	TrajActualDuration *bool
	TrajTurnsCollapsed *bool
	TrajCallsCollapsed *bool
	TrajDetailTab      *string

	workspacePane     *WorkspacePane
	centerHeight      float32 // resolved height below titlebar for terminal concessions
	captionRightInset float32
	jobsHeaderMounted *bool
}

// View paints the main dialogue shell below the title bar.
func View(c *ui.Context, m Model) {
	jobsMounted := false
	m.jobsHeaderMounted = &jobsMounted
	if m.Interactions != nil {
		m.Interactions.BeginQuestionFocusFrame()
		defer m.Interactions.EndQuestionFocusFrame()
	}
	if m.Sizes == nil {
		fallback := DefaultLayout()
		m.Sizes = &fallback
	}
	if m.ComposerResize != nil {
		m.ComposerResize.Adopt(m.FileBindingID, m.ConversationPreferences.ComposerResizeHeight, m.ConversationPreferences.ComposerResizeWidth)
	}
	prepareWorkspace(m)
	m = m.Sizes.Workspace.beginFileProjection(m)
	defer m.Sizes.Workspace.finishFileProjection(m)
	if !workspaceVisible(m, "browser") && m.OnHideBrowser != nil {
		m.OnHideBrowser()
	}

	if m.FileTabs != nil && !workspaceVisible(m, "files") {
		m.FileTabs.CancelPlanRead()
		m.FileTabs.Pause()
	}
	if m.Store == nil {
		return
	}
	if !workspaceVisible(m, "agents") {
		m.Store.CloseAgentNavigation()
	}
	viewport, viewportHeight := viewportDIP(c)
	m.ViewportW, m.ViewportH = viewport, viewportHeight
	m.Sizes.observeSidebarViewport(viewport)
	registerPageCommands(c, m)
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	border := chrome.Color(colors.Border)
	muted := chrome.Color(colors.Muted)
	if m.DisableDeveloperTools && m.ChatPaneTab != nil && *m.ChatPaneTab == "trajectory" {
		*m.ChatPaneTab = "chat"
	}
	m.Store.LoadStopSequenceConfig()
	snap := m.Store.Snapshot()
	snap.AgentPresetLabels = m.Store.AgentPresetLabelsForDeveloperTools(!m.DisableDeveloperTools)
	if m.HostTerminalUI != nil {
		observeHostTerminalPanes(m)
	}
	if m.FileTabs != nil {
		m.FileTabs.SyncPlanSession(snap.SelectedID)
	}
	if m.Interactions != nil {
		m.Interactions.SyncProjectedQuestionSnapshot(snap.SelectedID, snap.QuestionProjection.Questions, snap.QuestionProjection.Inbox, snap.QuestionProjection.QuestionsSeq, snap.QuestionProjection.InboxSeq, snap.QuestionProjection.QueueSeen, m.Store.QuestionQueueObservation)
		syncQuestionHistory(m.Interactions, snap)
	}
	if m.Sizes == nil {
		fallback := DefaultLayout()
		m.Sizes = &fallback
	}
	cols := m.Sizes.ResolveColumns(viewport, m.RightPaneOpen)
	if m.FilePaneWidth == nil {
		m.FilePaneWidth = &cols.Details
	} else {
		*m.FilePaneWidth = cols.Details
	}

	// Grow below titlebar without percent-height double counting. Theme BG is
	// the center's surface; the sidebar and right track own their own surfaces.
	shell := ui.Column(c).Grow(1).MinHeight(0).Background(chrome.BackdropCanvas(chrome.Color(colors.BG)))
	workspaceFloatDropSurface(c, m, shell)
	_, height := viewportDIP(c)
	// The titlebar is above this shell. Its previous-frame origin remains
	// stable during a window resize; preserve a visible terminal below it.
	m.centerHeight = maxf(0, height-shell.Bounds().Y)
	shell.Children(func() {
		mainRow(c, m, snap, colors, border, muted)
	})
	if !jobsMounted && m.Store != nil {
		m.Store.UnmountNativeJobs()
	}
	workflowMemberDialog(c, m)
}

func clampSidebarWidth(w float32) float32 {
	// ui-layout columns.ts: SIDEBAR_MIN = 264, SIDEBAR_MAX = 420.
	if w < SidebarMin {
		return SidebarMin
	}
	if w > SidebarMax {
		return SidebarMax
	}
	return w
}

// navigationView is shared by chat and the global main panels. Its owning
// shell stays mounted while only the center content changes.
func navigationView(c *ui.Context, m Model, snap session.ViewState) {
	selectSession := m.OnSelectSession
	if selectSession == nil {
		selectSession = m.Store.Select
	}
	selectRoot := selectSession
	selectSession = func(id string) {
		if m.Sizes != nil && m.Sizes.Workspace != nil {
			m.Sizes.Workspace.closeMainChild()
		}
		m.Store.MarkMainConversationViewed(id)
		selectRoot(id)
	}
	createSession := m.OnCreateSession
	if createSession == nil {
		createSession = func() { m.Store.CreateSession(context.Background()) }
	}
	rename := m.Store.SessionRename()
	projectModel := m.Projects
	projectModel.SearchQuery = derefString(m.SessionSearch)
	projectModel.OnResolvedFile = m.OnOpenDeliverable
	projectModel.OnObserver = func(id string) { openProjectObserver(m, id) }
	schedules := schedstore.ViewState{}
	if m.ScheduleStore != nil {
		m.ScheduleStore.EnsureContext(context.Background(), snap.SelectedID, nil)
		schedules = m.ScheduleStore.Snapshot()
	}
	sidebar.View(c, sidebar.Model{
		PendingStatuses:      sidebarPendingStatuses(m),
		Batch:                m.Store.SessionBatch(),
		OnBatchEdit:          m.Store.EditSessionBatch,
		OnBatchRequest:       m.Store.RequestSessionBatch,
		OnBatchConfirm:       func(token uint64) { m.Store.ConfirmSessionBatch(context.Background(), token) },
		OnBatchCancel:        m.Store.CancelSessionBatch,
		HideArchivedList:     m.HideArchivedList,
		Account:              m.Account,
		Schedules:            schedules,
		Projects:             projectModel,
		Collapsed:            m.Sizes.sidebarCollapsed(),
		ViewportW:            m.ViewportW,
		ViewportH:            m.ViewportH,
		OnToggle:             m.Sizes.ToggleSidebar,
		Snapshot:             snap,
		ThemeID:              m.ThemeID,
		SystemDark:           m.SystemDark,
		SearchQuery:          m.SessionSearch,
		SearchFocus:          m.SessionSearchFocus,
		WorkspaceLabel:       m.Store.WorkspaceCwd(),
		RenameSessionID:      rename.SessionID,
		RenameDraft:          &rename.Draft,
		RenameBusy:           rename.Busy,
		RenameError:          rename.Error,
		OnRenameDraftChanged: m.Store.EditSessionRename,
		OnSelect:             selectSession,
		OnRefresh: func() {
			m.Store.RefreshList(context.Background())
		},
		OnCreate:        createSession,
		OnOpenMarket:    m.OnOpenMarket,
		OnOpenSchedule:  m.OnOpenSchedule,
		OnOpenAssistant: m.OnOpenAssistant,
		OnOpenLauncher:  m.OnOpenLauncher,
		OnOpenSettings:  m.OnOpenSettings,
		ConnectionReady: m.ConnectionReady, ConnectionConnecting: m.ConnectionConnecting, ConnectionEpoch: m.ConnectionEpoch,
		ConnectionError:         m.ConnectionError,
		OnReconnect:             m.OnReconnect,
		DesktopUpdateInstalling: m.DesktopUpdateInstalling,
		DesktopUpdatePhase:      m.DesktopUpdatePhase, DesktopUpdateDetail: m.DesktopUpdateDetail, OnOpenDesktopUpdate: m.OnOpenDesktopUpdate,
		OnContactAccount:            m.OnContactAccount,
		OnRetryWorkspacePreferences: m.Store.RetryWorkspacePreferences,
		OnDropWorkspaceBefore: func(generation uint64, id, before string) {
			m.Store.InsertWorkspaceBeforeForHost(context.Background(), generation, id, before)
		},
		OnSelectWorkspace:        m.OnSelectWorkspace,
		OnManageWorkspace:        m.Store.BeginWorkspaceManagement,
		OnWorkspaceGrouping:      m.Store.SetWorkspaceGrouping,
		OnWorkspaceOrdering:      m.Store.SetWorkspaceOrdering,
		OnToggleWorkspaceGroup:   m.Store.ToggleWorkspaceGroup,
		OnMoveWorkspaceBefore:    func(id, before string) { m.Store.InsertWorkspaceBefore(context.Background(), id, before) },
		OnMoveLocalSessionBefore: m.Store.InsertLocalSessionBefore,
		OnMoveWorkspaceSessionBefore: func(workspaceID, id, before string) {
			m.Store.InsertSessionBefore(context.Background(), workspaceID, id, before)
		},
		OnBrowseWorkspace:    m.OnBrowseWorkspace,
		OnNoDirectory:        m.OnNoDirectory,
		OnRevealWorkspace:    m.OnRevealWorkspace,
		OnRenameWorkspace:    m.OnRenameWorkspace,
		OnDeleteWorkspace:    m.OnDeleteWorkspace,
		OnMoveSessionBefore:  m.OnMoveSessionBefore,
		WorkspaceRenameDraft: m.WorkspaceRenameDraft,
		WorkspaceMenuOpen:    m.WorkspaceMenuOpen,
		OnBeginRename:        m.Store.BeginSessionRename,
		OnCommitRename:       m.Store.SubmitSessionRename,
		OnCancelRename:       m.Store.CancelSessionRename,
		OnArchive: func(sessionID string) {
			m.Store.ArchiveSession(context.Background(), sessionID)
		},
		OnUnarchive: func(sessionID string) {
			m.Store.UnarchiveSession(context.Background(), sessionID)
		},
		OnDelete: func(sessionID string) {
			// Electron session delete: confirm before session/delete.
			choice, _ := mygo.Dialog.Message(mygo.MessageOptions{
				Type:    mygo.MessageWarning,
				Title:   i18n.T(i18n.KeySidebarDelete),
				Message: i18n.T(i18n.KeySidebarDeleteConfirm),
				Buttons: []string{i18n.T(i18n.KeySidebarDelete), i18n.T(i18n.KeySidebarRenameCancel)},
			})
			if choice.Button != 0 {
				return
			}
			m.Store.DeleteSession(context.Background(), sessionID)
		},
		OnPin: func(sessionID string) {
			m.Store.PinSession(context.Background(), sessionID)
		},
		OnUnpin: func(sessionID string) {
			m.Store.UnpinSession(context.Background(), sessionID)
		},
		OnCopySessionID: func(sessionID string) {
			if strings.TrimSpace(sessionID) != "" {
				mygo.Clipboard.WriteText(sessionID)
				m.Store.SetLastAction(i18n.T(i18n.KeySidebarCopyID) + ": " + sessionID)
			}
		},
		OnFork: func(sessionID string) {
			m.Store.ForkSession(context.Background(), sessionID)
		},
	})
	archiveConfirmation(c, m)
	sessionFeedbackDialog(c, m)
	permissionRiskConfirmation(c, m)
}

func mainRow(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors, border, muted ui.Color) {
	inspectToolCall := toolCallNavigationFrame(c, m, snap)
	withNavigation := func(body func()) {
		if m.ExternalNavigation {
			body()
			return
		}
		sidebarColumns(c, m.Sizes, func() { navigationView(c, m, snap) }, body)
	}
	withNavigation(func() {
		hero := conversationPhase(snap, hasSessionInteractions(m.Interactions, snap.SelectedID)) == conversationHero
		paintHeader := func() {
			headerHeight := workspaceSharedCaption(c, m, snap, colors, hero)
			m.centerHeight = max(float32(0), m.centerHeight-headerHeight)
		}
		// Electron desktop defaults to 2-column (rail | chat). Files/diff mounts
		// only when FilePaneWidth > 0 (tools toggle); otherwise chat fills.
		paintChat := func() {
			phase := conversationPhase(snap, hasSessionInteractions(m.Interactions, snap.SelectedID))
			emptyHero := phase == conversationHero
			interactionSnapshot := interaction.ViewState{}
			if m.Interactions != nil {
				interactionSnapshot = m.Interactions.SnapshotForAgent(snap.SelectedID)
			}
			interactionSnapshot, takeover := composerInteraction(interactionSnapshot)
			// Electron chat chrome sits ~20 DIP lower than the rail brand (caption
			// overlay). Add top inset on the chat column only 鈥?not the rail.
			conversationArea := ui.Column(c.Key("conversation-files:" + snap.SelectedID)).Grow(1).MinWidth(0).MinHeight(0)
			enter := conversationEnter(c, conversationArea, conversationEnterOwner(snap), emptyHero, snap.PromptAttempted, conversationEnterStructure(m, snap, phase, takeover))
			if takeover || phase == conversationSettling {
				enter.cancel()
			}
			var enterCard, enterViewport ui.Rect
			enterScroll := ui.Local(conversationArea, "conversation-enter-scroll", func() conversationEnterScroll { return conversationEnterScroll{} })
			enterScroll.signal(enter, c.Now())
			conversationArea.Children(func() {
				if mainSubagentConversationBody(c, m) {
					enter.cancel()
					return
				}
				conversationFileDrop(c, m, conversationArea, snap.SelectedID)
				m.Store.MarkMainConversationViewed(snap.SelectedID)
				// Prefer live commands/list from main; do not invent stub slash rows.
				slashCatalog := m.ComposerSlash
				filePaths := m.ComposerPaths
				if kind, _, _, _ := composer.ActiveSuggest(derefString(m.Draft)); kind == composer.SuggestMention {
					if len(filePaths) == 0 && m.OnIndexWorkspace != nil {
						filePaths = m.OnIndexWorkspace(strings.TrimSpace(m.WorkspaceRoot))
					}
				}
				editing := m.ComposerState != nil && m.ComposerState.EditSeq > 0
				permissionOptions, agentPresetOptions := m.Store.PresetPickerOptions(snap.SelectedID, !m.DisableDeveloperTools)
				presetSelection := m.Store.AgentPresetSelection()
				presetPick := m.AgentPresetPick
				if presetSelection.Label != "" {
					label := presetSelection.Label
					presetPick = &label
				}
				paintComposer := func(hero bool) {
					nextPrompt := m.Store.SyncNextPrompt(context.Background(), !hasPendingStopInteraction(m.Interactions, snap.SelectedID) && composer.NextPromptEligible(snap, m.ComposerState, m.ConversationPreferences.NextPrompt, hero))
					if !hero {
						// Electron TodoPanel: todos-only dock above composer (not PlanBoard).
						chat.TodoPanelView(c, chat.TodoPanelModel{
							Todos:      snap.PlanBoard.Todos,
							ThemeID:    m.ThemeID,
							SystemDark: m.SystemDark,
							Collapsed:  m.TodoPanelCollapsed,
						})
					}
					composer.View(c, composer.Model{
						SVGImages:            m.SVGImages,
						GroupMembers:         m.GroupMembers,
						NextPrompt:           nextPrompt,
						OnAcceptNextPrompt:   func() string { return m.Store.AcceptNextPrompt(nextPrompt) },
						ComposerBeam:         m.ConversationPreferences.ComposerBeam,
						ComposerBeamStyle:    m.ConversationPreferences.ComposerBeamStyle,
						TypingFx:             m.ConversationPreferences.TypingFx,
						TypingFxStyle:        m.ConversationPreferences.TypingFxStyle,
						ResizeEnabled:        m.ConversationPreferences.ComposerResize,
						Resize:               m.ComposerResize,
						ResizeMaxWidth:       maxf(0, m.Sizes.Center-conversationScrollGutter-32),
						OnResizeEnd:          m.OnComposerResizeEnd,
						Snapshot:             snap,
						ThemeID:              m.ThemeID,
						SystemDark:           m.SystemDark,
						WorkspaceRoot:        m.WorkspaceRoot,
						Inert:                composer.RequiresWorkspace(snap, hero),
						OnRequestWorkspace:   m.OnPickHeroWorkspace,
						Draft:                m.Draft,
						State:                m.ComposerState,
						OnSubmit:             m.OnSubmitComposer,
						OnSubmitGesture:      m.OnSubmitComposerGesture,
						OnOpenURL:            m.OnOpenCitationURL,
						OnOpenReference:      m.OnOpenDeliverable,
						OnAttachFiles:        m.OnAttachComposerFiles,
						OnRetryAttachment:    m.OnRetryComposerFile,
						OnWorkspaceReference: func(ref filelink.Reference) error { return insertWorkspaceReference(m, ref) },
						OnPasteImage:         m.OnPasteComposerImage,
						Editing:              editing,
						OnCancelEdit: func() {
							if m.ComposerState != nil {
								m.ComposerState.EndEdit()
							}
						},
						Hero:                    hero,
						OnCardLayout:            func(r ui.Rect) { enterCard = r },
						ModelPick:               m.ModelPick,
						PermissionOptions:       permissionOptions,
						AgentPresetOptions:      agentPresetOptions,
						PermissionPick:          m.PermissionPick,
						AgentPresetPick:         presetPick,
						AgentPresetSelection:    presetSelection,
						OnAgentPresetIntroduced: m.Store.AcknowledgeAgentPresetIntroduction,
						SlashCatalog:            slashCatalog,
						OnRetrySlashCommands:    func() { m.Store.RefreshSlashCommands(context.Background()) },
						FilePaths:               filePaths,
						OnSearchFileRefs:        m.OnSearchFileRefs,
						OnSearchReferences:      m.OnSearchReferences,
						SuggestList:             m.ComposerSuggest,
						SkillsOpen:              m.ComposerSkillsOpen,
						Skills:                  snap.SkillOptions,
						SkillsList:              m.ComposerSkillsList,
						SkillsFilter:            m.ComposerSkillsFilter,
						SkillHubHits:            snap.SkillHubHits,
						OnSearchSkillHub:        m.OnSearchSkillHub,
						OnInstallSkillHub:       m.OnInstallSkillHub,
						OnDeleteSkill:           m.OnDeleteSkill,
						SkillCreateName:         m.SkillCreateName,
						OnCreateSkill:           m.OnCreateSkill,
						OnToggleSkillModel:      m.OnToggleSkillModel,
						ChipEdit:                m.ComposerChipEdit,
						UseTextInputClient:      m.UseComposerTextInput,
						TextInput:               m.ComposerTextInput,
						OnSend: func(text string) {
							editSeq := float64(0)
							if m.EditMessageSeq != nil {
								editSeq = *m.EditMessageSeq
								*m.EditMessageSeq = 0
							}
							if editSeq > 0 {
								m.Store.SendPromptEdit(context.Background(), text, editSeq)
							} else {
								m.Store.SendPrompt(context.Background(), text)
							}
						},
						StopBlocked: hasPendingStopInteraction(m.Interactions, snap.SelectedID),
						OnStop:      func() { cancelCapturedResponse(m, snap) },
						OnModelChange: func(label string) {
							if choice, ok := snap.ModelChoices[label]; ok {
								m.Store.SelectModelSelectionForSessionID(context.Background(), snap.SelectedID, choice)
							}
						},
						OnRetryModels: func() {
							m.Store.RefreshModelCatalog(context.Background())
							m.Store.RefreshProjections(context.Background())
						},
						OnModelEffortChange: func(label string) {
							if choice, ok := snap.ModelEffortChoices[label]; ok {
								m.Store.SelectModelEffortSelectionForSessionID(context.Background(), snap.SelectedID, choice)
							}
						},
						OnExitPlanMode: func() { m.Store.ExitPlanMode(context.Background(), snap.SelectedID) },
						OnRetryPermissions: func() {
							m.Store.RefreshPermissionCatalog(context.Background())
							m.Store.RefreshProjections(context.Background())
						},
						OnPermissionChange: func(label string) {
							m.Store.SelectPermissionValueForSession(context.Background(), snap.SelectedID, label)
						},
						OnAgentPresetChange: func(label string) {
							m.Store.SelectAgentPresetIDForSession(context.Background(), snap.SelectedID, label)
						},
						OnClearDraft: func() {
							if m.Draft != nil {
								*m.Draft = ""
							}
							if m.EditMessageSeq != nil {
								*m.EditMessageSeq = 0
							}
						},
						OnNewSession: func() {
							m.Store.CreateSession(context.Background())
						},
						OnExecuteSlash: func(ctx context.Context, line string) {
							_ = m.Store.ExecuteCommandLine(ctx, line)
						},
						OnPickCommand:                   m.Store.PickComposerCommand,
						OnNativeCommandAction:           m.OnNativeCommandAction,
						OnLoadNativeCommandOptions:      m.Store.LoadNativeCommandOptions,
						OnSelectNativeCommandModel:      m.Store.SelectNativeCommandModel,
						OnSelectNativeCommandPermission: m.Store.SelectNativeCommandPermission,
						OnNativeCommandChanged:          m.OnNativeCommandChanged,
						OnToggleSkills:                  m.OnToggleComposerSkills,
						OnPickDirectory:                 m.OnPickComposerDirectory,
						OnPickImage:                     m.OnPickComposerImage,
						OnPickSpeech:                    m.OnPickComposerSpeech,
						Voice:                           m.ComposerVoice,
						OnStartVoice:                    m.OnStartComposerVoice,
						OnVoiceSettings:                 m.OnVoiceSettings,
						OnStopVoice:                     m.OnStopComposerVoice,
						OnCancelVoice:                   m.OnCancelComposerVoice,
						OnInsertVoice:                   m.OnInsertComposerVoice,
						OnCycleSpeech:                   m.OnCycleComposerSpeech,
						OnPickHeroWorkspace:             m.OnPickHeroWorkspace,
						WorkspacePickerAnchor:           m.WorkspacePickerAnchor,
						OnNoDirectory:                   m.OnNoDirectory,
						OnUseBusyEnterSetting:           m.Store.UseBusyEnterSetting,
						OnPromptModeChange: func(mode session.PromptMode) {
							m.Store.SetPromptMode(mode)
						},
					})
				}
				colW := float32(0)
				if m.Sizes != nil {
					colW = m.Sizes.Center
				}
				// The drawer band reserves no sidebar track, so the conversation
				// column owns the full width even though the resolved Center still
				// subtracts one; sizing the hero seat from it would clip the
				// headline's 预览版 badge at the right edge.
				if m.Sizes != nil && m.Sizes.drawerMode() {
					if width, _ := viewportDIP(c); width > 0 {
						colW = width
					}
				}
				contentAxis := conversationContentWidth(colW)
				if m.Sizes != nil {
					if emptyHero || m.Sizes.transcriptGesture.binding != m.FileBindingID {
						m.Sizes.transcriptGesture.side = ""
					}
					contentAxis = m.Sizes.conversationWidth(colW)
				}
				// ConversationRoot owns W. InputBar alone owns the 16 DIP side
				// clearances, so both phases receive the same W+64 seat.
				contentViewportW := maxf(0, colW-conversationScrollGutter)
				chatContentW := min(contentAxis, maxf(0, contentViewportW-64))
				composerOuterW := min(contentAxis+64, contentViewportW)
				if m.ConversationPreferences.ComposerResize && m.ComposerResize != nil && m.ComposerResize.Width > 0 {
					composerOuterW = min(m.ComposerResize.Width+32, contentViewportW)
				}
				chat.StatusView(c, chat.Model{
					Snapshot: snap, ThemeID: m.ThemeID, SystemDark: m.SystemDark,
					ConnectionBanner: m.ConnectionBanner, OnReconnect: m.OnReconnect,
					UsageJobsToday: m.UsageJobsToday, UsageBalance: m.UsageBalance, OnOpenUsage: m.OnOpenUsage,
				})
				conversationHistoryStatus(c, m, snap, muted)
				// ConversationRoot: same chatContent column for transcript + docked composer.
				chatRegion := ui.Column(c).Key("conversation-body").Grow(1).MinHeight(0)
				chatRegion.ObserveLayout(func(r ui.Rect) {
					enterViewport = r
				})
				if emptyHero {
					chatRegion.Padding(0, 0, 32, 0)
				}
				bindChatStopRegion(chatRegion, m, snap)
				chatRegion.Children(func() {
					paneTab := "chat"
					if m.ChatPaneTab != nil && strings.TrimSpace(*m.ChatPaneTab) != "" {
						paneTab = strings.TrimSpace(*m.ChatPaneTab)
					}
					if phase == conversationHero || phase == conversationSettling || (snap.HistoryError != "" && len(snap.Messages) == 0) {
						ui.Box(c).Key("conversation-phase-space").Grow(1).MinHeight(0)
					} else {
						transcript := ui.Box(c).Grow(1).MinHeight(0).Clip()
						m.Sizes.Workspace.setConversationBounds(transcript.Bounds())
						transcript.Children(func() {
							transcriptWidth, sideGrow := chatContentW, float32(1)
							if paneTab == "trajectory" {
								// The original trajectory ledger fills ConversationRoot.
								transcriptWidth, sideGrow = colW, 0
							} else {
								// Native List fills scrollBody; its rows reserve the stable
								// five-DIP bar before centering W. The viewport clears two.
								transcriptWidth, sideGrow = maxf(0, colW-conversationScrollMargin), 0
							}
							ui.Row(c).Fill().Children(func() {
								ui.Box(c).Grow(sideGrow).MinWidth(0)
								ui.Column(c).Width(transcriptWidth).FillHeight().MinWidth(0).Children(func() {
									if paneTab == "trajectory" {
										if snap.HistoryNavigation.Busy || snap.HistoryNavigation.Ready {
											m.Store.CancelHistoryTurnJump(snap.SelectedID, snap.HistoryNavigation.Request)
										}
										if m.ChatList != nil {
											m.ChatList.Key = nil
										}
										// Electron ui-trajectory: toolbar + timeline + ledger (not PlanBoard).
										chat.TrajectoryView(c, chat.TrajectoryModel{
											PaneWidth:      transcriptWidth,
											Snapshot:       snap,
											ThemeID:        m.ThemeID,
											SystemDark:     m.SystemDark,
											List:           m.ChatList,
											SearchQuery:    m.TrajSearch,
											SelectedID:     m.TrajSelectedID,
											ActualDuration: m.TrajActualDuration,
											TurnsCollapsed: m.TrajTurnsCollapsed,
											CallsCollapsed: m.TrajCallsCollapsed,
											DetailTab:      m.TrajDetailTab,
											OnLoadEarlier: func() {
												m.Store.LoadEarlierHistory(context.Background())
											},
											OnCopyDetail: func(text string) {
												if strings.TrimSpace(text) != "" {
													mygo.Clipboard.WriteText(text)
												}
											},
										})
									} else {
										chat.View(c, chat.Model{
											OnInspectToolCall:    inspectActionFor(inspectToolCall, m.toolCallInspector()),
											OnOpenModelSettings: m.OnOpenModelSettings,
											Inspector:         sessionInspectorChatNavigation(m),
											GroupRoom:         m.GroupRoom, GroupMembers: m.GroupMembers, OnReplyGroupThread: m.OnReplyGroupThread,
											DisableDeveloperTools: m.DisableDeveloperTools,
											Preferences:           m.ChatPreferences,
											ScheduleStore:         m.ScheduleStore, OnOpenScheduleTask: m.OnOpenScheduleTask,
											Snapshot:           snap,
											ContentWidth:       contentAxis,
											ThemeID:            m.ThemeID,
											SystemDark:         m.SystemDark,
											List:               m.ChatList,
											Scroll:             &enterScroll.scroll,
											ThinkingExpanded:   m.ThinkingExpanded,
											SuppressStatus:     true,
											FullColumnViewport: true,
											ConnectionBanner:   m.ConnectionBanner,
											OnReconnect:        m.OnReconnect,
											UsageJobsToday:     m.UsageJobsToday,
											UsageBalance:       m.UsageBalance,
											OnOpenUsage:        m.OnOpenUsage,
											FeedbackStore:      m.Store,
											HistoryStore:       m.Store,
											OnMessageFeedback: func(messageID, rating, note string) {
												m.Store.RateMessage(context.Background(), messageID, rating, note)
											},
											OnClearMessageFeedback: func(messageID string) {
												m.Store.ClearMessageFeedback(context.Background(), messageID)
											},
											FeedbackCommentMsgID: m.FeedbackCommentMsgID,
											FeedbackCommentDraft: m.FeedbackCommentDraft,
											OnForkTurn:           func(seq float64) { m.Store.ForkTurn(context.Background(), snap.SelectedID, seq) },
											EditingSeq: func() float64 {
												if m.ComposerState != nil {
													return m.ComposerState.EditSeq
												}
												return 0
											}(),
											EditReturnSeq: func() *float64 {
												if m.ComposerState != nil {
													return &m.ComposerState.EditReturnSeq
												}
												return nil
											}(),
											OnCancelEdit: func() {
												if m.ComposerState != nil {
													snapLastEditSeq := m.ComposerState.EditSeq
													m.ComposerState.EndEdit()
													if !m.ComposerState.Sending {
														m.ComposerState.EditFocus = false
														m.ComposerState.EditReturnSeq = snapLastEditSeq
													}
												}
											},
											OnEditMessage: func(seq float64, body string) {
												if m.ComposerState != nil && session.CanEditMessage(snap, seq) {
													m.ComposerState.BeginEdit(snap.SelectedID, seq, body)
												}
											},
											OnQuoteMessage: func(body string) {
												chat.QuoteIntoDraft(m.Draft, body)
											},
											OnLoadEarlier: func() {
												m.Store.LoadEarlierHistory(context.Background())
											},
											ImageLightboxPath:            m.ImageLightboxPath,
											OnOpenImageLightbox:          m.OnOpenImageLightbox,
											OnCloseImageLightbox:         m.OnCloseImageLightbox,
											OnOpenCitationURL:            m.OnOpenCitationURL,
											OnOpenChangesReview:          func(scope session.TurnFileScope, target session.TurnFileTarget) { OpenChangesReview(m, scope, target) },
											OnOpenPlanResource:           m.OnOpenPlanResource,
											OnOpenDeliverable:            m.OnOpenDeliverable,
											OnOpenHistoryReference:       m.OnOpenHistoryReference,
											VisualReply:                  m.VisualReply,
											VisualReplyBitmap:            m.VisualReplyBitmap,
											OnPreviewVisualReply:         m.OnPreviewVisualReply,
											OnInteractVisualReply:        m.OnInteractVisualReply,
											OnInlineVisualReply:          m.OnInlineVisualReply,
											OpenUIParent:                 m.OpenUIParent,
											OpenUIWindow:                 m.OpenUIWindow,
											OpenUIMediaOcclusions:        openUIMediaOcclusions(m),
											OnStopInteractiveVisualReply: m.OnStopInteractiveVisualReply,
											OnReadVisualReply:            m.OnReadVisualReply,
											OnSaveVisualReply:            m.OnSaveVisualReply,
											OnCloseVisualReply:           m.OnCloseVisualReply,
										})
									}
								})
								ui.Box(c).Grow(sideGrow).MinWidth(0)
							})
							if paneTab != "trajectory" && len(snap.Messages) > 0 && m.ChatList != nil {
								ui.Box(c).Key("turn-rail-seat").Attach(ui.AnchorTopRight, ui.AnchorBottomRight).Children(func() {
									chat.TurnRailView(c, chat.Model{Snapshot: snap, ThemeID: m.ThemeID, SystemDark: m.SystemDark}, transcriptWidth, func(seq float64) {
										for i, msg := range snap.Messages {
											if msg.Seq == seq {
												m.ChatList.ScrollTo(i, ui.Start)
												break
											}
										}
									})
								})
							}
							paintConversationWidthControls(c, m, transcript, contentAxis, paneTab != "trajectory")
						})
						if paneTab != "trajectory" && (snap.PlanBoard.PlanMode || snap.PlanBoard.Goal != nil || len(snap.PlanBoard.Todos) > 0) {
							// Goal/todo/takeover cards share the transcript content axis.
							ui.Row(c).FillWidth().Padding(0, conversationScrollGutter, 0, 0).Children(func() {
								ui.Box(c).Grow(1).MinWidth(0)
								ui.Column(c).Width(chatContentW).MaxWidthPercent(100).MinWidth(0).Children(func() {
									chat.PlanBoardView(c, chat.PlanBoardModel{
										Board:          snap.PlanBoard,
										LoadQueueImage: m.Store.QueueImageLoader(snap.SelectedID, snap.PlanBoard.QueueActions.Scope),
										ThemeID:        m.ThemeID,
										SystemDark:     m.SystemDark,
										HideQueue:      true,
										Open:           m.PlanBoardOpen,
										GoalDraft:      m.GoalDraft,
										OnCreateGoal: func(objective string) {
											m.Store.CreateGoal(context.Background(), objective)
										},
										OnEditGoal: func(objective string) {
											m.Store.EditGoal(context.Background(), objective)
										},
										OnCompleteGoal: func() {
											m.Store.CompleteGoal(context.Background())
										},
										OnClearGoal: func() {
											m.Store.ClearGoal(context.Background())
										},
										OnPauseGoal: func() {
											m.Store.PauseGoal(context.Background())
										},
										OnResumeGoal: func() {
											m.Store.ResumeGoal(context.Background())
										},
										OnQueueRemove: func(itemID string) {
											m.Store.UpdateQueue(context.Background(), snap.SelectedID, itemID, session.QueueActionRemove)
										},
										OnQueueSteer: func(itemID string) {
											m.Store.UpdateQueue(context.Background(), snap.SelectedID, itemID, session.QueueActionSteer)
										},
										QueueEditor: snap.QueueEditor,
										OnQueueEdit: func(itemID, _ string) {
											m.Store.BeginQueueEditForSession(context.Background(), snap.SelectedID, itemID)
										},
										OnChangeQueueEdit: m.Store.ChangeQueueEdit,
										OnCancelQueueEdit: m.Store.CancelQueueEdit,
										OnSaveQueueEdit:   func(token uint64) { m.Store.SaveQueueEdit(context.Background(), token) },
									})
								})
								ui.Box(c).Grow(1).MinWidth(0)
							})
						}
						if len(snap.BandActions) > 0 {
							ui.Row(c).FillWidth().Padding(0, conversationScrollGutter, 0, 0).Children(func() {
								ui.Box(c).Grow(1).MinWidth(0)
								ui.Column(c).Width(chatContentW).MaxWidthPercent(100).MinWidth(0).Gap(6).Padding(0, 0, 6, 0).Children(func() {
									ui.Text(c, i18n.T(i18n.KeyChatBandTitle)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
									ui.Row(c).FillWidth().Gap(6).Children(func() {
										for _, a := range snap.BandActions {
											act := a
											if ui.Button(c, act.Label).FontSize(theme.InterfaceSize(11)).Clicked() {
												m.Store.PressBand(context.Background(), act.ID)
											}
										}
									})
								})
								ui.Box(c).Grow(1).MinWidth(0)
							})
						}
						if note := strings.TrimSpace(snap.ControlNote); note != "" {
							ui.Row(c).FillWidth().Padding(0, conversationScrollGutter, 0, 0).Children(func() {
								ui.Box(c).Grow(1).MinWidth(0)
								ui.Column(c).Width(chatContentW).MaxWidthPercent(100).MinWidth(0).Padding(0, 0, 4, 0).Children(func() {
									ui.Text(c, i18n.T(i18n.KeyChatControlNote)+": "+note).FontSize(theme.InterfaceSize(11)).TextColor(muted)
								})
								ui.Box(c).Grow(1).MinWidth(0)
							})
						}
					}
					// Electron keeps the composer seat mounted on the trajectory tab
					// (data-conversation-composer-overlay); do not hide it.
					// ConversationRoot.composerSeat is flex:none: the transcript
					// yields height before the input and its below-card status rows.
					seat := ui.Row(c).Key("conversation-composer-seat").FillWidth().Shrink(0).Padding(0, conversationScrollGutter, 0, 0)
					seat.Children(func() {
						ui.Box(c).Grow(1).MinWidth(0)
						ui.Column(c).Key("conversation-composer-stack").Width(composerOuterW).MaxWidthPercent(100).MinWidth(0).
							OffsetAfterLayout(func(r ui.Rect) ui.Point { return ui.Point{Y: enter.measure(c.Now(), enterCard, r, enterViewport)} }).Children(func() {
							if phase == conversationSettling {
								// The unknown page settles on the Hero shell: same fish,
								// title slot and input-card geometry, no blank box and no
								// jump when the composer mounts.
								conversationSettlingSkeleton(c, colors)
								return
							}
							if emptyHero {
								conversationHeroHeadline(c, m, colors)
							}
							if m.Interactions != nil {
								region := ui.Box(c.Key("conversation-interaction")).FillWidth().MinWidth(0)
								region.Children(func() {
									paint := func() {
										interui.View(c, interui.Model{Snapshot: interactionSnapshot, ScopeAgentID: snap.SelectedID,
											ThemeID: m.ThemeID, SystemDark: m.SystemDark, Store: m.Interactions,
											OnAutoAllowChanged: m.OnAutoAllowChanged, OnOpenPlanResource: m.OnOpenPlanResource})
									}
									if takeover {
										// ConversationRoot caps takeover text at the same 336 DIP
										// as the draft. Long questions must leave the transcript visible.
										ui.Scroll(c.Key("conversation-takeover-scroll")).FillWidth().MinWidth(0).MaxHeight(336).Children(paint)
									} else {
										paint()
									}
								})
							}
							ordinary := ui.Box(c).Key("conversation-composer-input").FillWidth()
							if takeover {
								// Keep the resident draft identity without painting, focus,
								// or a second occupied footer while approval/questions own it.
								ordinary.Absolute().Top(0).Left(0).Right(0).Invisible()
							}
							ordinary.Children(func() {
								if !emptyHero {
									// Queued prompts sit directly above the composer; the
									// expanded plan board keeps goal/todo/plan sections only.
									chat.QueueDockView(c, chat.PlanBoardModel{
										Board:          snap.PlanBoard,
										LoadQueueImage: m.Store.QueueImageLoader(snap.SelectedID, snap.PlanBoard.QueueActions.Scope),
										ThemeID:        m.ThemeID,
										SystemDark:     m.SystemDark,
										QueueEditor:    snap.QueueEditor,
										OnQueueRemove: func(itemID string) {
											m.Store.UpdateQueue(context.Background(), snap.SelectedID, itemID, session.QueueActionRemove)
										},
										OnQueueSteer: func(itemID string) {
											m.Store.UpdateQueue(context.Background(), snap.SelectedID, itemID, session.QueueActionSteer)
										},
										OnQueueEdit: func(itemID, _ string) {
											m.Store.BeginQueueEditForSession(context.Background(), snap.SelectedID, itemID)
										},
										OnChangeQueueEdit: m.Store.ChangeQueueEdit,
										OnCancelQueueEdit: m.Store.CancelQueueEdit,
										OnSaveQueueEdit:   func(token uint64) { m.Store.SaveQueueEdit(context.Background(), token) },
									})
								}
								paintComposer(emptyHero)
								if !emptyHero {
									// sessioncost owns the whole compact statistics row;
									// painting the chat line as well duplicated it.
									if !costui.OwnsPerformanceStats {
										chat.PerformanceUsageView(c, snap, m.ChatPreferences.PerformanceUsage, muted)
									}
									costModel := m.SessionCost
									costModel.Snapshot = snap
									costui.View(c, costModel, muted)
								}
							})
						})
						ui.Box(c).Grow(1).MinWidth(0)
					})
					// Hero and skeleton share one centered stack: the space below
					// mirrors the growing space above so the column stays centered.
					if emptyHero || phase == conversationSettling {
						ui.Box(c).Key("conversation-hero-bottom-space").Grow(1).MinHeight(0)
					}
				})
			})
		}
		paintCenter := func() {
			centerTerminalColumn(c, m, paintChat, func() {
				termui.View(c, termui.Model{
					Panel: m.Terminal, ThemeID: m.ThemeID, SystemDark: m.SystemDark,
					ShellsNote: m.TerminalShellsNote, OnRestart: m.OnRestartTerminal, OnClose: m.OnCloseTerminal,
					OnAddToChat: m.OnAddSelectionToChat, CanAddToChat: m.CanAddSelectionToChat,
					TargetBinding: m.FileBindingID, OnOpenTarget: m.OnOpenTerminalTarget, OnOpenShortcuts: m.OnOpenSettings,
				})
			})
		}
		workspaceColumns(c, m, paintCenter, paintHeader)
	})
}

// Approval has the original composer's priority 1; a question is the default
// takeover. Continued late-answer cards are history and do not hide the draft.
func composerInteraction(snap interaction.ViewState) (interaction.ViewState, bool) {
	selected := -1
	for index, pending := range snap.Pending {
		if pending.Continued {
			continue
		}
		// QuestionCards.hide unpublishes the question from the original
		// composer chain while retaining its row's reopen action.
		if pending.Event == interaction.EventUserQuestions && snap.QuestionCards[pending.EventID].Hidden {
			continue
		}
		if selected < 0 || pending.Event == interaction.EventApproval {
			selected = index
		}
		if pending.Event == interaction.EventApproval {
			break
		}
	}
	if selected < 0 {
		return snap, false
	}
	snap.Pending = []interaction.Pending{snap.Pending[selected]}
	snap.QuestionHistory, snap.QuestionStatus = nil, ""
	snap.AutoAllow, snap.Banner = false, ""
	return snap, true
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// conversationContentWidth is the shared raw width axis from ConversationRoot:
// clamp(680, 64% column, 920). Each seat clamps its own rendered envelope, so
// transcript W and input-card W+32 remain aligned even on a narrow column.
func conversationContentWidth(colW float32) float32 {
	return min(float32(920), maxf(680, colW*0.64))
}

func rightPane(c *ui.Context, m Model) {
	if m.workspacePane != nil && m.workspacePane.Kind == "project-observer" {
		projectModel := m.Projects
		if projectModel.ObserverModel != nil {
			projectModel = projectModel.ObserverModel(m.workspacePane.ProjectObserver)
		}
		projectModel.OnResolvedFile = m.OnOpenDeliverable
		projects.ObserverView(c, projectModel, m.workspacePane.ProjectObserver)
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	field := chrome.BackdropSurface(chrome.Color(colors.Sidebar))
	border := chrome.Color(colors.Border)

	tab := "files"
	if m.RightPaneTab != nil {
		switch *m.RightPaneTab {
		case "custom-panels", "session-inspector-log", "changes-review", "subagentchat", "plan", "diff", "agents", "browser", "host-terminal", "schedule-task":
			tab = *m.RightPaneTab
		}
	}

	ui.Column(c).Fill().MinWidth(0).Draw(chrome.Glass(field, chrome.MaterialPanel)).Children(func() {
		workspacePaneControls(c, m)
		ui.Box(c).Fill().Height(1).Background(border)
		if tab == "browser" && m.BrowserContent != nil {
			m.BrowserContent(c)
			return
		}
		if m.OnHideBrowser != nil {
			m.OnHideBrowser()
		}
		if tab == "custom-panels" && m.CustomPanelContent != nil {
			m.CustomPanelContent(c)
			return
		}
		if tab == "changes-review" {
			changesReviewPane(c, m)
			return
		}
		if tab == "session-inspector-log" {
			sessionInspectorPane(c, m)
			return
		}
		if tab == "subagentchat" {
			subagentChatPane(c, m)
			return
		}
		if tab == "schedule-task" {
			scheduleTaskPane(c, m)
			return
		}
		if tab == "host-terminal" && m.Store != nil {
			m = hostTerminalPaneModel(m)
			termui.HostView(c, m.HostTerminalUI, m.HostTerminalStore, m.Store.Snapshot().SelectedID, m.ThemeID, m.SystemDark, hostTerminalActions(m))
			return
		}
		if tab == "agents" && m.Store != nil {
			store := m.Store
			navigation := store
			if m.AgentNavigationStore != nil {
				navigation = m.AgentNavigationStore
			}
			agents.View(c, agents.Model{
				OnOpenChat:         subagentChatOpener(m),
				OnOpenConversation: subagentConversationOpener(m),
				Interactions:       m.Interactions,
				OnOpenPlanResource: m.OnOpenPlanResource,
				Store:              navigation,
				Manager:            m.AgentPresetManager,
				Snapshot:           store.Snapshot(),
				ThemeID:            m.ThemeID,
				SystemDark:         m.SystemDark,
				PromptDraft:        m.AgentsPromptDraft,
				OnOpenURL:          m.OnOpenCitationURL,
				OnInterrupt: func(parentID, childID string) {
					navigation.InterruptChild(context.Background(), parentID, childID, "")
				},
				OnOpenDocument:     m.OnOpenAgentDocument,
				OnClose: func() {
					if m.RightPaneTab != nil {
						*m.RightPaneTab = "files"
					}
				},
			})
			return
		}
		if tab == "diff" {
			diffsurface.View(c, diffsurface.Model{
				OnOpenFile: func(path string) error {
					handled, err := files.OpenPresentedPath(files.Model{
						Tabs:                m.FileTabs,
						BindingID:           m.FileBindingID,
						SessionID:           m.Store.Snapshot().SelectedID,
						WorkspaceRoot:       m.WorkspaceRoot,
						SelectedFile:        m.SelectedFile,
						Preview:             m.FilePreview,
						PreviewErr:          m.FilePreviewErr,
						Editing:             m.FileEditing,
						EditDraft:           m.FileEditDraft,
						EditStatus:          m.FileEditStatus,
						StatNote:            m.FileStatNote,
						ReadRemote:          m.OnReadWorkspaceFile,
						ReadRemoteContext:   m.OnReadWorkspaceFileContext,
						ReadTextPageContext: m.OnReadWorkspaceTextPageContext,
						OpenDocumentViewer:  m.OnOpenDocumentViewer,
						ReadTableContext:    m.OnReadWorkspaceTableContext,
						ImageGeneration:     m.FileImageGeneration,
						StatRemoteContext:   m.OnStatWorkspaceFileContext,
						StatRemote:          m.OnStatWorkspaceFile,
					}, path)
					if err != nil {
						return err
					}
					if !handled {
						return errors.New("file is unavailable in this workspace (it may have been deleted)")
					}
					if m.RightPaneTab != nil {
						*m.RightPaneTab = "files"
					}
					return nil
				},
				Workbench:     m.GitWorkbench,
				WorkspaceRoot: m.WorkspaceRoot,
				ThemeID:       m.ThemeID,
				SystemDark:    m.SystemDark,
				PathFilter:    m.DiffPath,
				Text:          m.DiffText,
				ErrMsg:        m.DiffErr,
			})
			return
		}
		if m.workspacePane != nil {
			if tab == "files" && m.RemoteFilesContent != nil {
				m.workspacePane.remoteFilesRendered = true
				if m.workspacePane.RemoteFiles == nil {
					m.workspacePane.RemoteFiles = remoteui.NewFileWorkspaceState()
				}
				if m.RemoteFilesContent(c, m.Store, m.workspacePane.RemoteFiles) {
					return
				}
			}
			m.workspacePane.Files.rendered = true
		}
		treeSplit := m.FileTreeSplit
		if treeSplit == nil {
			treeSplit = &m.Sizes.RightTree
		}
		files.View(c, files.Model{
			SVGImages:           m.SVGImages,
			PlanOnly:            tab == "plan",
			AllowPageShortcuts:  m.FileShortcuts,
			OnNewTab:            m.OnOpenFilePicker,
			PlanSnapshot:        m.Store.Snapshot(),
			OnLoadPlanHistory:   func() { m.Store.LoadEarlierHistory(context.Background()) },
			ReadSubagentPlan:    m.Store.ReadSubagentPlan,
			Tabs:                m.FileTabs,
			BindingID:           m.FileBindingID,
			SessionID:           m.Store.Snapshot().SelectedID,
			OnReferenceFile:     func(ref filelink.Reference) error { return insertWorkspaceReference(m, ref) },
			WorkspaceRoot:       m.WorkspaceRoot,
			ThemeID:             m.ThemeID,
			SystemDark:          m.SystemDark,
			Expanded:            m.FileExpanded,
			SelectedFile:        m.SelectedFile,
			Filter:              m.FileFilter,
			Preview:             m.FilePreview,
			PreviewErr:          m.FilePreviewErr,
			PreviewList:         m.FilePreviewList,
			PreviewSplit:        treeSplit,
			Editing:             m.FileEditing,
			EditDraft:           m.FileEditDraft,
			EditStatus:          m.FileEditStatus,
			EditViewMode:        m.FileEditViewMode,
			EditSplit:           m.FileEditSplit,
			PaneWidth:           m.FilePaneWidth,
			ListRemote:          m.OnListWorkspaceFiles,
			ListRemoteContext:   m.OnListWorkspaceFilesContext,
			WatchRemoteContext:  m.OnWatchWorkspaceDirectoryContext,
			ReadRemote:          m.OnReadWorkspaceFile,
			ReadRemoteContext:   m.OnReadWorkspaceFileContext,
			ReadTextPageContext: m.OnReadWorkspaceTextPageContext,
			OpenDocumentViewer:  m.OnOpenDocumentViewer,
			ReadTableContext:    m.OnReadWorkspaceTableContext,
			ImageGeneration:     m.FileImageGeneration,
			StatRemoteContext:   m.OnStatWorkspaceFileContext,
			StatRemote:          m.OnStatWorkspaceFile,
			StatNote:            m.FileStatNote,
			OnShowDiff: func(rel string) {
				if m.GitWorkbench != nil {
					m.GitWorkbench.CancelComparison()
					m.GitWorkbench.BaseRef = ""
					m.GitWorkbench.Staged = false
				}
				if m.RightPaneTab != nil {
					*m.RightPaneTab = "diff"
				}
				if m.DiffPath != nil {
					*m.DiffPath = rel
				}
				text, errMsg := diffsurface.LoadDiff(m.WorkspaceRoot, rel)
				if m.DiffText != nil {
					*m.DiffText = text
				}
				if m.DiffErr != nil {
					*m.DiffErr = errMsg
				}
			},
			OnSaveEdit: func(rel, text string) error {
				full, err := workspacefs.JoinRoot(m.WorkspaceRoot, rel)
				if err != nil {
					return err
				}
				return workspacefs.WriteFileText(full, text)
			},
			OnOpenFloatingDoc:   m.OnOpenFloatingDoc,
			OnOpenDoc:           m.OnOpenDoc,
			OnOpenMedia:         m.OnOpenMedia,
			OnReveal:            m.OnRevealFile,
			OnOpenWith:          m.OnOpenWithApp,
			OpenWithApps:        m.FileOpenWithApps,
			OnRenderPDF:         m.OnRenderOfficePDF,
			OfficePreviewStatus: m.OfficePreviewStatus,
			OnAddToChat:         m.OnAddSelectionToChat,
			CanAddToChat:        m.CanAddSelectionToChat,
			OnOpenLink:          m.OnOpenCitationURL,
		})
	})
}
