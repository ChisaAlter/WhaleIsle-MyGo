// Package frame is the M4 main shell: resizable sidebar, chat, optional
// right and bottom regions.
package frame

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pluginui"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/shortcuts"
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
	"whaleisle/internal/ui/workspacepicker"
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
	return sizes.resolveColumnsWithReserve(viewport, 0, rightOpen)
}

// A docked Pad reserves body space without changing the window's responsive
// breakpoints, the persisted split preferences, or the native caption span.
func (sizes *LayoutSizes) resolveColumnsWithReserve(viewport, reserve float32, rightOpen bool) Columns {
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
	available := maxf(0, viewport-maxf(0, reserve))
	cols := computeColumns(maxf(0, available-sidebarDivider-SplitterWidth), sidebar, pref, viewport)
	if cols.Details == 0 {
		// A closed right track has no divider; only the sidebar divider remains.
		cols = computeColumns(maxf(0, available-sidebarDivider), sidebar, 0, viewport)
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
	// The drawer opens at the reference's lastDrawerHeight(), lastDrawer ?? 280
	// (ui-layout client.js:720-722), instead of the previous 180 default; 180 is
	// now only its minimum (columns.ts TERMINAL_DRAWER_MIN).
	return LayoutSizes{Sidebar: SidebarDefault, Center: 680, Main: 560, Bottom: TerminalDrawerDefault, RightTree: 280}
}

// TerminalDrawerDefault is the height the terminal drawer opens at before any
// drag: lastDrawer ?? 280 (ui-layout client.js lastDrawerHeight).
const TerminalDrawerDefault = float32(280)

// terminalDrawerMin is the reference's TERMINAL_DRAWER_MIN: a dragged or
// stored drawer never gets shorter than 180 DIP, so the centre column keeps
// 180 DIP for it however tall the window is (columns.ts TERMINAL_DRAWER_MIN).
const terminalDrawerMin = float32(180)

// Model wires the frame to session state and layout handles.
type Model struct {
	PublicContributions             pluginui.PublicContributions
	InteractionRenderSlot           interui.SlotRenderer
	WorkspaceNavigation             []pluginui.WorkspaceNavigation
	OnWorkspaceSessionAction        func(sessionID, owner, action string)
	GlobalPluginSlot                func(*ui.Context, string, func())
	OnCloseWindow                   func()
	OnCloseForegroundModal          func() bool
	ChildBusyEnter                  string
	RightOverlayReserve             float32
	ConversationViews               []pluginui.ConversationView
	OnSelectConversationView        func(string)
	OnClearConversationView         func()
	OnSelectBuiltinConversationView func(string)
	RenderConversationView          func(*ui.Context) bool
	PluginRightPaneTabs             []pluginui.RightPaneTab
	PluginRightPaneKinds            []PluginRightPaneKind
	OnOpenPluginRightPaneKind       func(kind, replaceGuideID string) error
	OnSelectPluginRightPaneTab      func(string)
	OnClosePluginRightPaneTab       func(string)
	RenderPluginRightPaneTab        func(*ui.Context, string) bool
	SVGImages                       *svgimage.Owner
	OpenUIParent                    uintptr
	OpenUIWindow                    *mygo.Window
	CustomPanelContent              func(*ui.Context)
	OnOpenDocumentViewer            func(string) (files.DocumentSurface, error)
	GroupRoom                       bool
	GroupMembers                    []session.GroupMember
	OnReplyGroupThread              func(string)
	DisableDeveloperTools           bool
	TitlebarPreferences             settings.TitlebarPreferences
	WorkspaceTrailing               func(*ui.Context, string)
	WorkspacePage                   bool
	// ActivePanel is the visible global route, or empty for a conversation.
	ActivePanel             string
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
	ViewportW       float32
	ViewportH       float32
	OnSelectSession func(string)
	OnCreateSession func()

	Projects projects.Model
	// Account is the sidebar's bottom account seat (avatar, name, menu).
	Account sidebar.Account
	Store   *session.Store
	// FrameView is a read-only handle for synchronous view construction. The view
	// it returns is shared with the store and valid for the current pass only.
	// Event callbacks and asynchronous work must use Store.Snapshot instead.
	FrameView               *session.FrameSnapshot
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

	WorkspaceRoot   string
	FileBindingID   string
	FileTabs        *files.Tabs
	GitWorkbench    *diffsurface.Workbench
	FileExpanded    *map[string]bool
	SelectedFile    *string
	FileFilter      *string
	FilePreview     *string
	FilePreviewErr  *string
	FilePreviewList *ui.ListState
	FileTreeSplit   *float32
	FileShortcuts   func() bool
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
	OnOpenTeam              func()
	OrdinaryTeam            func() projects.OrdinaryTeamModel
	NativePages             []pluginui.NativeView
	OnOpenNativePage        func(string)
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
	OnQueryReferences       func(context.Context, string, bool) ([]session.ReferenceCandidate, error)
	ChildDraft              func(string, session.AgentSessionAddress) *composer.DraftState
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
	OnReadWorkspaceImageContext    func(context.Context, string, int64) ([]byte, error)
	OnReadWorkspaceTableContext    func(context.Context, string, workspacefs.XLSXPageRequest) (*workspacefs.XLSXPage, error)
	FileImageGeneration            uint64
	OnStatWorkspaceFileContext     func(context.Context, string) string
	// OnStatWorkspaceFile returns a short workspaceFiles/stat summary.
	OnStatWorkspaceFile              func(rel string) string
	FileStatNote                     *string
	OnPickHeroWorkspace              func()
	WorkspacePickerAnchor            *ui.Handle
	OnNoDirectory                    func()
	OnCreateWorkspaceSession         func(uint64, string)
	OnSelectWorkspace                func(string)
	OnBrowseWorkspace                func() // SidebarRoot 鈽?/ folder row
	DirectoryActions                 []workspacepicker.DirectoryAction
	OnRevealWorkspace                func()
	OnOpenWorkspaceLocal             func()
	OnRenameWorkspace                func(title string)
	OnDeleteWorkspace                func()
	OnMoveSessionBefore              func(sessionID, beforeSessionID string)
	WorkspaceRenameDraft             *string
	WorkspaceMenuOpen                *bool
	OnOpenFloatingDoc                func(string)
	OnOpenDoc                        func(absPath string, kind workspacefs.DocKind, inWebView bool)
	OnOpenMedia                      func(path string, floating bool)
	OnRevealFile                     func(absPath string)
	OnOpenWithApp                    func(absPath, appID string)
	FileOpenWithApps                 []files.OpenWithApp
	FileOpenWithState                *files.OpenWithState
	OnRefreshFileOpenWith            func()
	OnOpenWithEditor                 func(absPath, editorID string, line int)
	FileEditorApps                   []files.OpenWithApp
	OnRenderOfficePDF                func(rel string)
	OfficePreviewStatus              func(path string) string
	OnOpenBrowserPreview             func()
	OnOpenMarket                     func()
	OnCreatePluginDraft              func(region string)
	OnOpenSchedule                   func()
	OnOpenAssistant                  func()
	OnOpenWhaleSettings              func()
	OnNewHostTerminal                func()
	OnPreviewDeliverablePresentation func(string, string) error
	PluginSlot                       func(*ui.Context, string, string, float64, func())
	OnOpenLauncher                   func()
	OnContactAccount                 func()
	OnOpenSettings                   func()
	OnVoiceSettings                  func()
	OnToggleTools                    func()
	OnToggleTerminal                 func()
	OnOpenHostTerminal               func()
	OnRestartTerminal                func()
	OnCloseTerminal                  func()

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
	TodoPanelCollapsed            *bool
	ImageLightboxPath             *string
	OnOpenImageLightbox           func(displayPath string)
	OnCloseImageLightbox          func()
	OnOpenCitationURL             func(url string)
	OnOpenPlanResource            func(planpreview.Document)
	OnOpenDeliverable             func(path string)
	OnPreviewDeliverable          func(path string) error
	OnOpenHistoryReference        func(path string, complete func(error))
	ConversationContributions     func(*ui.Context, session.ViewState, string)
	ConversationTurnContributions func(*ui.Context, session.ViewState, session.Message)
	OnOpenSessionPlugins          func()
	OnCloseSessionPlugins         func()
	SessionPluginsActive          bool
	OnDownloadSessionLog          func(string)
	SessionLogDownloading         func(string) bool
	OnOpenSessionFeedback         func(string)
	VisualReply                   session.VisualReplyState
	VisualReplyBitmap             *ui.Bitmap
	OnPreviewVisualReply          func(attachmentID string)
	OnInteractVisualReply         func(attachmentID string)
	OnInlineVisualReply           func(attachmentID string) session.InlineVisualReply
	OnStopInteractiveVisualReply  func()
	OnReadVisualReply             func(attachmentID string)
	OnSaveVisualReply             func(attachmentID string)
	OnCloseVisualReply            func()
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
	syncExternalAccount(c, m)
	if !workspaceVisible(m, "agents") {
		m.Store.CloseAgentNavigation()
	}
	viewport, viewportHeight := viewportDIP(c)
	m.ViewportW, m.ViewportH = viewport, viewportHeight
	m.Sizes.observeSidebarGeometry(viewport, viewportHeight)
	m.Sizes.ObserveSidebarSessionSelection(m.Store.Snapshot())
	registerPageCommands(c, m)
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	border := chrome.Color(colors.Border)
	muted := chrome.Color(colors.Muted)
	if m.DisableDeveloperTools && m.ChatPaneTab != nil && *m.ChatPaneTab == "trajectory" {
		selectConversationView(m, "chat")
	}
	m.Store.LoadStopSequenceConfig()
	snap := renderSnapshot(m)
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
	cols := m.Sizes.resolveColumnsWithReserve(viewport, m.RightOverlayReserve, m.RightPaneOpen)
	// Below 768 the right dock derives fullscreen from the live frame width.
	// At larger widths an explicit delivery still needs a visible fallback if
	// the normal details track cannot fit. Failed previews never change mode.
	if preview := m.OnPreviewDeliverable; preview != nil {
		m.OnPreviewDeliverable = func(path string) error {
			if err := preview(path); err != nil {
				return err
			}
			// Compute against a copy: opening the drawer updates responsive state,
			// but this check must not mutate the live frame's layout preferences.
			proposed := *m.Sizes
			if m.Sizes.Responsive != nil {
				responsive := *m.Sizes.Responsive
				proposed.Responsive = &responsive
			}
			if viewport >= workspaceAutoFullscreenWidth && proposed.resolveColumnsWithReserve(viewport, m.RightOverlayReserve, true).Details == 0 {
				w := m.Sizes.Workspace
				if w != nil && w.Fullscreen == "" && w.Root != nil && w.Root.find(w.Active) != nil {
					w.Fullscreen = w.Active
				}
			}
			return nil
		}
	}
	if m.FilePaneWidth == nil {
		m.FilePaneWidth = &cols.Details
	} else {
		*m.FilePaneWidth = cols.Details
	}

	// Grow below titlebar without percent-height double counting. Theme BG is
	// the center's surface; the sidebar and right track own their own surfaces.
	shell := ui.Column(c).Grow(1).MinHeight(0)
	// ExternalNavigation's outer NavigationShell already owns the common canvas.
	// Standalone and detached frames retain one canvas layer of their own.
	if !m.ExternalNavigation {
		shell.Background(chrome.BackdropCanvas(chrome.Color(colors.BG)))
	}
	workspaceFloatDropSurface(c, m, shell)
	_, height := viewportDIP(c)
	// The titlebar is above this shell. Its previous-frame origin remains
	// stable during a window resize; preserve a visible terminal below it.
	m.centerHeight = maxf(0, height-shell.Bounds().Y)
	shell.Children(func() {
		mainRow(c, m, snap, colors, border, muted)
		if !m.ExternalNavigation {
			renderGlobalShellSlot(c, m, "shell.bottom", nil)
		}
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

var externalAccounts sync.Map

func syncExternalAccount(c *ui.Context, m Model) {
	if !m.ExternalNavigation || m.Store == nil {
		return
	}
	prev, ok := externalAccounts.Load(m.Store)
	externalAccounts.Store(m.Store, m.Account)
	if !ok {
		if m.Account.Name != "" || m.Account.SignedIn {
			c.Invalidate()
		}
		return
	}
	if old, ok := prev.(sidebar.Account); ok {
		if old.Name != m.Account.Name || old.Detail != m.Account.Detail || old.SignedIn != m.Account.SignedIn || old.Busy != m.Account.Busy {
			c.Invalidate()
		}
	}
}

func resolveNavigationAccount(m Model) sidebar.Account {
	account := m.Account
	if account.Name == "" && !account.SignedIn && m.Store != nil {
		if v, ok := externalAccounts.Load(m.Store); ok {
			if ext, ok := v.(sidebar.Account); ok {
				onOpen := account.OnOpen
				account = ext
				if account.OnOpen == nil {
					account.OnOpen = onOpen
				}
			}
		}
	}
	return account
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
		for _, item := range m.Store.Snapshot().Items {
			if item.SessionID == id && routeRegisteredSessionOpen(m, item) {
				return
			}
		}
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
	menuShortcutLabels := map[string]string{}
	for id, chord := range shortcuts.Resolve(m.PageShortcutBindings) {
		label := chord.String()
		switch runtime.GOOS {
		case "windows":
			label = strings.Replace(label, "Mod", "Ctrl", 1)
		case "darwin":
			label = strings.Replace(label, "Mod", "Cmd", 1)
		}
		menuShortcutLabels[id] = label
	}
	newSessionShortcut := menuShortcutLabels["session.new"]
	m.Store.SetSessionBatchScope(sidebar.BatchSelectionScope(sidebar.Model{Snapshot: snap, Collapsed: m.Sizes.sidebarCollapsed(), SearchQuery: m.SessionSearch}))
	sidebar.View(c, sidebar.Model{
		PublicContributions:      m.PublicContributions,
		WorkspaceNavigation:      m.WorkspaceNavigation,
		OnWorkspaceSessionAction: m.OnWorkspaceSessionAction,
		ActivePanel:              m.ActivePanel,
		NewSessionShortcut:       newSessionShortcut,
		MenuShortcutLabels:       menuShortcutLabels,
		PendingStatuses:          sidebarPendingStatuses(m),
		Batch:                    m.Store.SessionBatch(),
		OnBatchEdit:              m.Store.EditSessionBatch,
		OnBatchRequest:           m.Store.RequestSessionBatch,
		OnBatchConfirm:           func(token uint64) { m.Store.ConfirmSessionBatch(context.Background(), token) },
		OnBatchCancel:            m.Store.CancelSessionBatch,
		HideArchivedList:         m.HideArchivedList,
		Account:                  resolveNavigationAccount(m),
		Schedules:                schedules,
		Projects:                 projectModel,
		Collapsed:                m.Sizes.sidebarCollapsed(),
		ViewportW:                m.ViewportW,
		ViewportH:                m.ViewportH,
		OnToggle:                 m.Sizes.ToggleSidebar,
		Snapshot:                 snap,
		ThemeID:                  m.ThemeID,
		SystemDark:               m.SystemDark,
		SearchQuery:              m.SessionSearch,
		SearchFocus:              m.SessionSearchFocus,
		WorkspaceLabel:           m.Store.WorkspaceCwd(),
		RenameSessionID:          rename.SessionID,
		RenameDraft:              &rename.Draft,
		RenameBusy:               rename.Busy,
		RenameError:              rename.Error,
		OnRenameDraftChanged:     m.Store.EditSessionRename,
		OnSelect:                 selectSession,
		OnRefresh: func() {
			m.Store.RefreshList(context.Background())
		},
		OnCreate:              createSession,
		OnOpenMarket:          m.OnOpenMarket,
		OnOpenSchedule:        m.OnOpenSchedule,
		OnOpenAssistant:       m.OnOpenAssistant,
		NativePages:           m.NativePages,
		OnOpenNativePage:      m.OnOpenNativePage,
		OnOpenLauncher:        m.OnOpenLauncher,
		OnOpenSessionPlugins:  selectedSessionPluginAction(m, snap.SelectedID, m.OnOpenSessionPlugins),
		OnCloseSessionPlugins: selectedSessionPluginAction(m, snap.SelectedID, m.OnCloseSessionPlugins),
		SessionPluginsActive:  m.SessionPluginsActive,
		OnOpenSettings:        m.OnOpenSettings,
		ConnectionReady:       m.ConnectionReady, ConnectionConnecting: m.ConnectionConnecting, ConnectionEpoch: m.ConnectionEpoch,
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
		OnCreateWorkspaceSession: m.OnCreateWorkspaceSession,
		OnManageWorkspace:        m.Store.BeginWorkspaceManagement,
		OnBeginWorkspaceRename:   m.Store.BeginWorkspaceRename,
		OnBeginWorkspaceRemoval:  m.Store.BeginWorkspaceRemoval,
		OnWorkspaceGrouping:      m.Store.SetWorkspaceGrouping,
		OnWorkspaceOrdering:      m.Store.SetWorkspaceOrdering,
		OnArchivedFilter:         m.Store.SetWorkspaceArchivedFilter,
		OnToggleWorkspaceGroup:   m.Store.ToggleWorkspaceGroup,
		OnMoveWorkspaceBefore:    func(id, before string) { m.Store.InsertWorkspaceBefore(context.Background(), id, before) },
		OnMoveLocalSessionBefore: m.Store.InsertLocalSessionBefore,
		OnMoveWorkspaceSessionBefore: func(workspaceID, id, before string) {
			m.Store.InsertSessionBeforeForHost(snap.WorkspaceLayout.HostGeneration, workspaceID, id, before)
		},
		OnBrowseWorkspace:    m.OnBrowseWorkspace,
		DirectoryActions:     m.DirectoryActions,
		OnNoDirectory:        m.OnNoDirectory,
		OnRevealWorkspace:    m.OnRevealWorkspace,
		OnRenameWorkspace:    m.OnRenameWorkspace,
		OnDeleteWorkspace:    m.OnDeleteWorkspace,
		OnMoveSessionBefore:  m.OnMoveSessionBefore,
		WorkspaceRenameDraft: m.WorkspaceRenameDraft,
		WorkspaceMenuOpen:    m.WorkspaceMenuOpen,
		OnBeginRename:        func(id, title string) { beginConversationRename(m, id, title) },
		OnCommitRename:       m.Store.SubmitSessionRename,
		OnCancelRename:       m.Store.CancelSessionRename,
		OnArchive: func(sessionID string) {
			archiveConversation(m, sessionID, false)
		},
		OnUnarchive: func(sessionID string) {
			archiveConversation(m, sessionID, true)
		},
		OnDelete: func(sessionID string) {
			// Electron session delete: confirm before session/delete.
			title := sessionID
			for _, item := range m.Store.Snapshot().Items {
				if item.SessionID == sessionID && item.Title != "" {
					title = item.Title
					break
				}
			}
			choice, err := mygo.Dialog.Message(mygo.MessageOptions{
				Type:          mygo.MessageWarning,
				Title:         i18n.T(i18n.KeySidebarDelete),
				Message:       fmt.Sprintf(i18n.T(i18n.KeySidebarDeleteConfirm), title),
				Buttons:       []string{i18n.T(i18n.KeySidebarDelete), i18n.T(i18n.KeySidebarRenameCancel)},
				DefaultButton: 1,
				CancelButton:  1,
			})
			if err != nil || choice.Button != 0 {
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
			if item, ok := conversationActionItem(m, sessionID); ok && !routeProjectConversationAction(m, item, "fork") {
				m.Store.ForkSession(context.Background(), sessionID)
			}
		},
	})
	archiveConfirmation(c, m)
	sessionFeedbackDialog(c, m)
	permissionRiskConfirmation(c, m)
}

func mainRow(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors, border, muted ui.Color) {
	inspectToolCall := toolCallNavigationFrame(c, m, snap)
	withNavigation := func(body func()) {
		if m.ExternalNavigation || workspaceAutoFullscreen(m) {
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
			if skinLayout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark); skinLayout.Active() {
				conversationArea.Background(chrome.Color(skinLayout.ChatBackground)).Radius(0).Padding(0)
			}
			enter := conversationEnterLazy(c, conversationArea, conversationEnterOwner(snap), emptyHero, snap.PromptAttempted, func() string {
				return conversationEnterStructure(m, snap, phase, takeover)
			})
			if takeover || phase == conversationSettling {
				enter.cancel()
			}
			var enterCard, enterViewport ui.Rect
			residentCard := ui.Local(conversationArea, "interaction-resident-card:"+snap.PlanHistoryBinding+":"+snap.SelectedID, func() ui.Rect { return ui.Rect{} })
			enterScroll := ui.Local(conversationArea, "conversation-enter-scroll", func() conversationEnterScroll { return conversationEnterScroll{} })
			enterScroll.signal(enter, c.Now())
			conversationArea.Children(func() {
				if mainSubagentConversationBody(c, m) {
					enter.cancel()
					return
				}
				if snap.SelectedID != "" && !m.HideChatViewTabs && !conversationIsManaged(snap) && !conversationHeaderHidden(snap, hasSessionInteractions(m.Interactions, snap.SelectedID)) {
					conversationViewTabs(c, m, colors, false)
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
						Hero: hero,
						OnCardLayout: func(r ui.Rect) {
							enterCard = r
							if r.W > 0 && r.H > 0 {
								changed := residentCard.W != r.W || residentCard.H != r.H
								*residentCard = r
								if changed {
									c.Invalidate()
								}
							}
						},
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
						OnQueryReferences:       m.OnQueryReferences,
						RenderSlot:              nativeComposerSlot(m),
						ExtraActions:            func(c *ui.Context) { projects.ProjectStopButton(c, m.Projects, snap.SelectedID) },
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
							if m.OnCreateSession != nil {
								m.OnCreateSession()
							} else {
								m.Store.CreateSession(context.Background())
							}
						},
						OnExecuteSlash: func(ctx context.Context, line string) {
							_ = m.Store.ExecuteCommandLine(ctx, line)
						},
						OnPickCommand: func(generation uint64, sessionID, line string) {
							current := m.Store.Snapshot()
							if generation != current.ComposerHostGeneration || sessionID != current.SelectedID {
								return
							}
							if strings.TrimSpace(line) == "/export" && m.OnDownloadSessionLog != nil {
								for _, descriptor := range current.SlashCommands {
									if descriptor.Name == "export" && descriptor.Input == nil && descriptor.DefinitionId != nil && *descriptor.DefinitionId == "@deepseek-ai/dsh-session-log-export" {
										m.OnDownloadSessionLog(sessionID)
										return
									}
								}
							}
							m.Store.PickComposerCommand(generation, sessionID, line)
						},
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
						colW = maxf(0, width-m.RightOverlayReserve)
					}
				}
				// Native columns meet edge to edge: the conversation root is
				// viewport − sidebar (1160 at 1440 wide, probed in round 186), while
				// Sizes.Center still reserves one DIP per divider between the tracks.
				// The width formulas below are percentages of that native column, so
				// feeding them the divider-shrunk Center lands the axis 0.64×(1..2) DIP
				// short — 741.12 instead of 742.4, i.e. a 773.3 composer card instead
				// of 774.39. Only the no-right-dock case is widened; with a dock open
				// the track arithmetic is the drawn geometry.
				axisColumn := colW
				if m.Sizes != nil && m.Sizes.Right <= 0 && !m.Sizes.drawerMode() {
					if width, _ := viewportDIP(c); width > 0 {
						axisColumn = maxf(colW, width-m.Sizes.Sidebar-m.RightOverlayReserve)
					}
				}
				contentAxis := conversationContentWidth(axisColumn)
				if m.Sizes != nil {
					if emptyHero || m.Sizes.transcriptGesture.binding != m.FileBindingID {
						m.Sizes.transcriptGesture.side = ""
					}
					contentAxis = m.Sizes.conversationWidth(axisColumn)
				}
				if skinLayout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark); skinLayout.Active() {
					contentAxis = skinLayout.ContentWidth
				} else if m.ConversationPreferences.ComposerResize && m.ComposerResize != nil && m.ComposerResize.Width >= 680 && (m.Sizes == nil || m.Sizes.TranscriptWidth <= 0) {
					contentAxis = m.ComposerResize.Width - 32
				}
				// ConversationRoot owns W. InputBar alone owns the 16-DIP side
				// clearances, so both phases receive the same stack seat, and the
				// card inside it is exactly the conversation body: contentAxis+32
				// (0.64 x 1160 + 32 = 774.4 at 1440x920), the native
				// `max-width: 774.4px`. Deriving both from one helper is what keeps
				// the old 772-vs-774.4 residual at a 0-DIP difference.
				contentViewportW := maxf(0, colW-conversationScrollGutter)
				chatContentW := min(contentAxis, maxf(0, contentViewportW-64))
				composerOuterW := min(conversationComposerStackWidth(contentAxis), contentViewportW)
				if m.ConversationPreferences.ComposerResize && m.ComposerResize != nil && m.ComposerResize.Width > 0 {
					composerOuterW = min(m.ComposerResize.Width+32, contentViewportW)
				}
				chat.StatusView(c, chat.Model{
					Snapshot: snap, ThemeID: m.ThemeID, SystemDark: m.SystemDark,
					ConnectionBanner: m.ConnectionBanner, OnReconnect: m.OnReconnect,
				})
				conversationHistoryStatus(c, m, snap, muted)
				// ConversationRoot: same chatContent column for transcript + docked composer.
				chatRegion := ui.Column(c).Key("conversation-body").Label("Conversation body").Grow(1).MinHeight(0)
				chatRegion.ObserveLayout(func(r ui.Rect) {
					enterViewport = r
				})
				bindChatStopRegion(chatRegion, m, snap)
				chatRegion.Children(func() {
					paneTab := "chat"
					if m.ChatPaneTab != nil && strings.TrimSpace(*m.ChatPaneTab) != "" {
						paneTab = strings.TrimSpace(*m.ChatPaneTab)
					}
					if strings.HasPrefix(paneTab, "plugin:") {
						ui.Column(c).Grow(1).MinHeight(0).FillWidth().Children(func() {
							if m.RenderConversationView == nil || !m.RenderConversationView(c) {
								ui.Text(c, paneLabel("插件视图尚不可用，你可以切回对话。", "This plugin view is not available yet. You can return to Chat.")).Padding(12)
								if ui.Button(c, i18n.T(i18n.KeyChatPaneTabChat)).Clicked() {
									*m.ChatPaneTab = "chat"
									if m.OnClearConversationView != nil {
										m.OnClearConversationView()
									}
								}
							}
						})
						return
					}
					// The ledger owns its empty/loading states. Chat's hero or
					// history-error placeholder must not hide the selected view.
					if paneTab != "trajectory" && (phase == conversationHero || phase == conversationSettling || (snap.HistoryError != "" && len(snap.Messages) == 0)) {
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
											OnOpenImage:    m.OnOpenImageLightbox,
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
										navigation := sessionInspectorChatNavigation(m)
										chat.View(c, chat.Model{
											ReasoningBody:                    nativeReasoningSlot(m),
											RenderSlot:                       nativeMessageSlot(m),
											ToolRenderSlot:                   nativeToolSlot(m),
											OnPreviewDeliverablePresentation: m.OnPreviewDeliverablePresentation,
											OnInspectToolCall:                inspectActionFor(inspectToolCall, m.toolCallInspector()),
											TurnContributions:                m.ConversationTurnContributions,
											Inspector:                        navigation,
											GroupRoom:                        m.GroupRoom, GroupMembers: m.GroupMembers, OnReplyGroupThread: m.OnReplyGroupThread,
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
											OnPreviewDeliverable:         m.OnPreviewDeliverable,
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
										syncInspectorChatProjection(m, navigation)
									}
								})
								ui.Box(c).Grow(sideGrow).MinWidth(0)
							})
							if paneTab != "trajectory" && len(snap.Messages) > 0 && m.ChatList != nil {
								ui.Box(c).Key("turn-rail-seat").Attach(ui.AnchorRight, ui.AnchorRight).Right(6).Children(func() {
									railModel := chat.Model{Snapshot: snap, ThemeID: m.ThemeID, SystemDark: m.SystemDark, List: m.ChatList, Preferences: m.ChatPreferences, ThinkingExpanded: m.ThinkingExpanded, OnLoadEarlier: func() { m.Store.LoadEarlierHistory(context.Background()) }}
									chat.TurnRailView(c, railModel, transcriptWidth, func(seq float64) {
										chat.ScrollToMessageSeq(railModel, seq, ui.Start)
										if m.Store != nil {
											for _, turn := range snap.HistoryNavigation.Turns {
												if turn.Seq == seq && !turn.Loaded {
													m.Store.JumpToHistoryTurn(context.Background(), snap.SelectedID, snap.HistoryNavigation.Request, seq)
													break
												}
											}
										}
									})
								})
							}
							paintConversationWidthControls(c, m, transcript, contentAxis, paneTab != "trajectory", &enterScroll.scroll)
						})
						if paneTab != "trajectory" && m.PlanBoardOpen == nil && m.TodoPanelCollapsed == nil && !m.UseComposerTextInput && (snap.PlanBoard.PlanMode || snap.PlanBoard.Goal != nil || len(snap.PlanBoard.Todos) > 0) {
							// Goal/todo/takeover cards share the transcript content axis.
							ui.Row(c).FillWidth().Padding(0, conversationScrollGutter, 0, 0).Children(func() {
								ui.Box(c).Grow(1).MinWidth(0)
								ui.Column(c).Width(chatContentW).MaxWidthPercent(100).MinWidth(0).Children(func() {
									chat.PlanBoardView(c, chat.PlanBoardModel{
										Board:          snap.PlanBoard,
										HideGoal:       true,
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
							chat.GoalDockView(c, chat.PlanBoardModel{
								Board: snap.PlanBoard, ThemeID: m.ThemeID, SystemDark: m.SystemDark,
								OnEditGoal:     func(objective string) { m.Store.EditGoal(context.Background(), objective) },
								OnCompleteGoal: func() { m.Store.CompleteGoal(context.Background()) },
								OnClearGoal:    func() { m.Store.ClearGoal(context.Background()) },
								OnPauseGoal:    func() { m.Store.PauseGoal(context.Background()) },
								OnResumeGoal:   func() { m.Store.ResumeGoal(context.Background()) },
							}, m.Store)
							if m.Interactions != nil {
								viewSnap := interactionSnapshot
								viewSnap.QuestionHistory = nil
								viewSnap.AutoAllow = false
								viewSnap.Banner = ""
								region := ui.Box(c.Key("conversation-interaction")).FillWidth().MinWidth(0)
								region.Children(func() {
									paint := func() {
										interui.View(c, interui.Model{Snapshot: viewSnap, ScopeAgentID: snap.SelectedID,
											RenderSlot: m.InteractionRenderSlot,
											ThemeID:    m.ThemeID, SystemDark: m.SystemDark, Store: m.Interactions,
											OnAutoAllowChanged: m.OnAutoAllowChanged, OnOpenPlanResource: m.OnOpenPlanResource}, *residentCard)
									}
									paint()
								})
							}
							ordinary := ui.Box(c).Key("conversation-composer-input").FillWidth()
							if takeover {
								// Keep the resident draft identity without painting, focus,
								// or a second occupied footer while approval/questions own it.
								ordinary.Absolute().Top(0).Left(0).Right(0).Invisible()
							}
							ordinary.Children(func() {
								if projects.ConversationComposer(c, m.Projects, snap.SelectedID, selectedConversationParent(snap)) {
									return
								}
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
								if emptyHero {
									projects.HeroOption(c, m.Projects, snap)
									// Padding belongs to the centered composerHero item.
									ui.Box(c).Key("conversation-hero-stack-foot").Height(conversationHeroStackFoot).Shrink(0)
								}
								if !emptyHero {
									// sessioncost owns the whole compact statistics row;
									// painting the chat line as well duplicated it.
									if !costui.OwnsPerformanceStats {
										chat.PerformanceUsageView(c, snap, m.ChatPreferences.PerformanceUsage, muted)
									}
									costModel := m.SessionCost
									costModel.Snapshot = snap
									if !composerTelemetryDock(c, snap, costModel, muted) {
										costui.View(c, costModel, muted)
									}
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

// The visible takeover follows the pending-interaction registry, before the
// selected pending value is passed to composer slots. Larger precedence wins:
// plan review 2, question 1, approval 0; newest publication wins within a rank.
// Continued late-answer cards are history and do not hide the draft.
func composerInteraction(snap interaction.ViewState) (interaction.ViewState, bool) {
	return interui.SelectComposerInteraction(snap)
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
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	field := chrome.BackdropSurface(chrome.Color(colors.Sidebar))
	border := chrome.Color(colors.Border)

	tab := "files"
	if m.RightPaneTab != nil {
		switch *m.RightPaneTab {
		case "guide", "project-team", "plugin-native", "custom-panels", "session-inspector-log", "changes-review", "subagentchat", "plan", "diff", "agents", "team", "browser", "host-terminal", "schedule-task":
			tab = *m.RightPaneTab
		}
	}

	artifact := ui.Column(c).Label("Skin artifact boundary").Fill().MinWidth(0)
	if layout := theme.ResolvedSkinLayout(m.ThemeID, m.SystemDark); layout.Active() {
		artifact.Background(chrome.Color(layout.ChatBackground)).Radius(0).Padding(0).Clip().DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: layout.SidebarBorder, H: r.H}, chrome.Color(layout.SidebarBorderColor), 0)
		})
	} else {
		if m.workspacePane == nil || m.workspacePane.Kind != "project-observer" {
			artifact.Draw(chrome.Glass(field, chrome.MaterialPanel))
		}
	}
	artifact.Children(func() {
		if m.workspacePane != nil && m.workspacePane.Kind == "project-observer" {
			workspaceProjectRequirements(c, m)
			return
		}
		workspacePaneControls(c, m)
		ui.Box(c).Fill().Height(1).Background(border)
		if tab == "guide" {
			workspaceResourceGuide(c, m)
			return
		}
		if tab == "project-team" {
			workspaceProjectTeam(c, m)
			return
		}
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
			termui.HostView(c, m.HostTerminalUI, m.HostTerminalStore, m.Store.SelectedSessionID(), m.ThemeID, m.SystemDark, hostTerminalActions(m))
			return
		}
		if tab == "team" {
			workspaceOrdinaryTeam(c, m)
			return
		}
		if tab == "plugin-native" {
			handle := ""
			if m.workspacePane != nil {
				handle = m.workspacePane.PluginRightPaneHandle
			}
			if m.RenderPluginRightPaneTab == nil || handle == "" || !m.RenderPluginRightPaneTab(c, handle) {
				ui.Text(c, paneLabel("此插件视图已不可用，请从资源菜单重新打开。", "This plugin view is no longer available. Reopen it from the resource menu.")).Padding(12).FillWidth()
			}
			return
		}
		if tab == "agents" && m.Store != nil {
			store := m.Store
			navigation := store
			if m.AgentNavigationStore != nil {
				navigation = m.AgentNavigationStore
			}
			agents.ViewWithJobs(c, agents.Model{
				ChatPreferences:       m.ChatPreferences,
				InteractionRenderSlot: m.InteractionRenderSlot,
				OnOpenChat:            subagentChatOpener(m),
				OnOpenConversation:    subagentConversationOpener(m),
				Interactions:          m.Interactions,
				OnOpenPlanResource:    m.OnOpenPlanResource,
				Store:                 navigation,
				Manager:               m.AgentPresetManager,
				Snapshot:              renderSnapshot(m),
				ThemeID:               m.ThemeID,
				SystemDark:            m.SystemDark,
				PromptDraft:           m.AgentsPromptDraft,
				ChildDraft:            m.ChildDraft,
				BusyEnter:             m.ChildBusyEnter,
				ComposerOverride: func(c *ui.Context, address session.AgentSessionAddress) bool {
					return projects.ConversationComposer(c, m.Projects, address.ChildSessionID, address.ParentSessionID)
				},
				OnOpenURL: m.OnOpenCitationURL,
				OnInterrupt: func(parentID, childID string) {
					navigation.InterruptChild(context.Background(), parentID, childID, "")
				},
				OnOpenDocument: m.OnOpenAgentDocument,
				OnClose: func() {
					if m.workspacePane != nil {
						workspacePaneClose(m, m.workspacePane)
					} else if m.RightPaneTab != nil {
						*m.RightPaneTab = "files"
					}
				},
			}, func(ctx *ui.Context) { workspaceAgentJobs(ctx, m) })
			return
		}
		if tab == "diff" {
			diffsurface.View(c, diffsurface.Model{
				OnOpenFile: func(path string) error {
					return openWorkspaceFileResource(m, files.Model{
						Tabs:                m.FileTabs,
						BindingID:           m.FileBindingID,
						SessionID:           m.Store.SelectedSessionID(),
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
			if tab == "files" && m.RemoteFilesContent != nil && (m.workspacePane.Files == nil || m.workspacePane.Files.ResourcePath == "") {
				m.workspacePane.remoteFilesRendered = true
				if m.workspacePane.RemoteFiles == nil {
					m.workspacePane.RemoteFiles = remoteui.NewFileWorkspaceState()
				}
				if m.RemoteFilesContent(c, m.Store, m.workspacePane.RemoteFiles) {
					remote := m.workspacePane.RemoteFiles.Files
					bindWorkspacePageRefresh(m, func() {
						if remote != nil {
							view := remote.Snapshot()
							if view.Resolved && !view.Busy && !view.Local && view.RootPath != "" {
								remote.Browse(view.RootPath)
							}
						}
					})
					return
				}
			}
			m.workspacePane.Files.rendered = true
		}
		treeSplit := m.FileTreeSplit
		if treeSplit == nil {
			treeSplit = &m.Sizes.RightTree
		}
		resourceOnly := m.workspacePane != nil && m.workspacePane.Files != nil && m.workspacePane.Files.ResourcePath != ""
		fileModel := files.Model{
			TreeOnly:            tab == "files" && !resourceOnly,
			ResourceOnly:        resourceOnly,
			SVGImages:           m.SVGImages,
			PlanOnly:            tab == "plan",
			AllowPageShortcuts:  m.FileShortcuts,
			OnNewTab:            m.OnOpenFilePicker,
			PlanSnapshot:        renderSnapshot(m),
			OnLoadPlanHistory:   func() { m.Store.LoadEarlierHistory(context.Background()) },
			ReadSubagentPlan:    m.Store.ReadSubagentPlan,
			Tabs:                m.FileTabs,
			BindingID:           m.FileBindingID,
			SessionID:           m.Store.SelectedSessionID(),
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
			OpenWithState:       m.FileOpenWithState,
			OnRefreshOpenWith:   m.OnRefreshFileOpenWith,
			ReadImageContext:    m.OnReadWorkspaceImageContext,
			OnOpenWithEditor:    m.OnOpenWithEditor,
			EditorApps:          m.FileEditorApps,
			OnRenderPDF:         m.OnRenderOfficePDF,
			OfficePreviewStatus: m.OfficePreviewStatus,
			OnAddToChat:         m.OnAddSelectionToChat,
			CanAddToChat:        m.CanAddSelectionToChat,
			OnOpenLink:          m.OnOpenCitationURL,
		}
		fileModel.OnOpenResource = func(path string) error { return openWorkspaceFileResource(m, fileModel, path) }
		if resourceOnly {
			fileModel.OnNewTab = nil
		}
		if tab == "files" {
			bindWorkspacePageRefresh(m, func() {
				if fileModel.TreeOnly {
					files.RefreshExplorer(fileModel)
				} else {
					files.ReloadSelectedFile(fileModel)
				}
			})
		}
		files.View(c, fileModel)
	})
}

var (
	telemetrySpeedSVG = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M3.4041 13.096C2.49514 12.187 1.87614 11.0288 1.62537 9.76798C1.37459 8.50716 1.50331 7.20028 1.99525 6.01261C2.48719 4.82494 3.32025 3.80981 4.3891 3.09557C5.45795 2.38134 6.71458 2.00008 8.0001 2C9.28563 2.00008 10.5423 2.38134 11.6111 3.09557C12.68 3.80981 13.513 4.82494 14.005 6.01261C14.4969 7.20028 14.6256 8.50716 14.3748 9.76798C14.1241 11.0288 13.5051 12.187 12.5961 13.096" stroke="currentColor"/><path d="M8 8.49994L11.6114 4.88855" stroke="currentColor"/><path d="M8 9.75C8.69036 9.75 9.25 9.19036 9.25 8.5C9.25 7.80964 8.69036 7.25 8 7.25C7.30964 7.25 6.75 7.80964 6.75 8.5C6.75 9.19036 7.30964 9.75 8 9.75Z" fill="currentColor"/></svg>`))
	telemetryCacheSVG = ui.MustParseSVG([]byte(`<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" stroke-width="1"><path d="M13.1967 5.1869C13.7232 4.77378 14.0003 4.30517 14.0001 3.82819C14.0003 3.3512 13.7232 2.88259 13.1967 2.46947C12.6702 2.05635 11.9128 1.71328 11.0006 1.47475C10.0885 1.23621 9.05371 1.11062 8.00039 1.1106C6.94707 1.11057 5.9123 1.23612 5.00009 1.47461C4.08742 1.71301 3.32948 2.05604 2.80249 2.46919C2.2755 2.88235 1.99805 3.35106 1.99805 3.82819C1.99805 4.30531 2.2755 4.77402 2.80249 5.18718C3.32948 5.60033 4.08742 5.94336 5.00009 6.18176C5.9123 6.42025 6.94707 6.5458 8.00039 6.54578C9.05371 6.54575 10.0885 6.42016 11.0006 6.18163C11.9128 5.94309 12.6702 5.60002 13.1967 5.1869Z" stroke="currentColor"/><path d="M2 3.80371V11.7848" stroke="currentColor"/><path d="M14 3.80371V11.7848" stroke="currentColor"/><path d="M2 7.81396C2 8.60524 2.63214 9.36411 3.75736 9.92363C4.88258 10.4832 6.4087 10.7975 8 10.7975C9.5913 10.7975 11.1174 10.4832 12.2426 9.92363C13.3679 9.36411 14 8.60524 14 7.81396" stroke="currentColor"/><path d="M2 11.7847C2 12.6081 2.63214 13.3977 3.75736 13.98C4.88258 14.5622 6.4087 14.8893 8 14.8893C9.5913 14.8893 11.1174 14.5622 12.2426 13.98C13.3679 13.3977 14 12.6081 14 11.7847" stroke="currentColor"/></svg>`))
)

func composerTelemetryDock(c *ui.Context, snap session.ViewState, m costui.Model, muted ui.Color) bool {
	if snap.SelectedID == "" || m.State == nil {
		return false
	}
	for _, ns := range m.Preferences.Namespaces {
		if ns.NS == "ui-conversation" {
			raw := strings.ReplaceAll(string(ns.Value), " ", "")
			if strings.Contains(raw, `"sessionCost":true`) || strings.Contains(raw, `"statsLine":false`) {
				return false
			}
			break
		}
	}
	memo := ui.Local(c.Root(), "dock-performance-history-derived", func() session.HistoryDerivedMemo { return session.HistoryDerivedMemo{} })
	perfTokens, perfMS, _ := memo.RecordedPerformance(snap)
	var prompt, hit float64
	for _, source := range snap.CostSources {
		if source.ID != snap.SelectedID || source.Usage == nil {
			continue
		}
		for _, route := range source.Usage.Models {
			if route.Peak != nil {
				prompt += route.Peak.Miss + route.Peak.Hit
				hit += route.Peak.Hit
			}
			if route.OffPeak != nil {
				prompt += route.OffPeak.Miss + route.OffPeak.Hit
				hit += route.OffPeak.Hit
			}
		}
	}
	if prompt <= 0 {
		for _, source := range snap.CostSources {
			if source.Usage == nil {
				continue
			}
			for _, route := range source.Usage.Models {
				if route.Peak != nil {
					prompt += route.Peak.Miss + route.Peak.Hit
					hit += route.Peak.Hit
				}
				if route.OffPeak != nil {
					prompt += route.OffPeak.Miss + route.OffPeak.Hit
					hit += route.OffPeak.Hit
				}
			}
		}
	}
	hasSpeed := perfMS > 0 && perfTokens > 0
	hasCache := prompt > 0
	if !hasSpeed && !hasCache {
		return false
	}
	ui.Column(c).Key("composer-dock").Label("Composer dock").FillWidth().MinHeight(24).Padding(0, 8, 4, 8).Gap(4).AlignItems(ui.Center).Children(func() {
		ui.Row(c).Key("composer-dock-stats").FillWidth().Justify(ui.Center).AlignItems(ui.Center).Gap(12).Children(func() {
			if hasSpeed {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(1, 8).Children(func() {
					ui.Icon(c, telemetrySpeedSVG).Size(14, 14).TextColor(muted)
					ui.Text(c, fmt.Sprintf("%.0f %s", perfTokens*1000/perfMS, i18n.T(i18n.KeyComposerDockSpeedUnit))).FontSize(12).FixedLineHeight(20).TextColor(muted)
				})
			}
			if hasCache {
				pct := 100 * hit / prompt
				pctText := fmt.Sprintf("%.0f%%", math.Round(pct))
				if pct < 100 && math.Round(pct) >= 100 {
					pctText = fmt.Sprintf("%.1f%%", pct)
				}
				ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(1, 8).Children(func() {
					ui.Icon(c, telemetryCacheSVG).Size(14, 14).TextColor(muted)
					ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerDockCacheHit), pctText)).FontSize(12).FixedLineHeight(20).TextColor(muted)
				})
			}
		})
	})
	return true
}
