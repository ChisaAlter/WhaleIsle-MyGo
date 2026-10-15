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
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/agents"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/i18n"
	interui "whaleisle/internal/ui/interaction"
	"whaleisle/internal/ui/projects"
	costui "whaleisle/internal/ui/sessioncost"
	"whaleisle/internal/ui/sidebar"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

// LayoutSizes persist split pane dimensions (DIPs) across frames.
type LayoutSizes struct {
	Sidebar         float32
	SidebarExpanded float32 // last expanded preference; Sidebar=0 is the compact rail
	Center          float32 // resolved chat width; not a saved right-pane preference
	Right           float32 // remembered right width, 0 = first-open viewport default
	Main            float32 // height of the dialogue column when the terminal is open
	Bottom          float32 // >40 shows the terminal (saved height preference when hidden)
	RightTree       float32 // height of file tree above preview in the right pane
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
		s = SidebarCollapsed
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
	pref := float32(0)
	if rightOpen {
		if sizes.Right <= 0 {
			sizes.Right = DefaultDetailsWidth(viewport)
		}
		pref = sizes.Right
	}
	cols := computeColumns(maxf(0, viewport-2*SplitterWidth), sizes.Sidebar, pref, viewport)
	if cols.Details == 0 {
		// A closed right track has no divider; only the sidebar divider remains.
		cols = computeColumns(maxf(0, viewport-SplitterWidth), sizes.Sidebar, 0, viewport)
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

// DefaultLayout returns initial split sizes for a 1200×800 window.
// SplitVertical *size is the FIRST (top) pane height — that must be Main, not Bottom.
func DefaultLayout() LayoutSizes {
	// ui-layout columns.ts: SIDEBAR_DEFAULT = 280 (min 264, max 420).
	return LayoutSizes{Sidebar: SidebarDefault, Center: 680, Main: 560, Bottom: 180, RightTree: 280}
}

// Model wires the frame to session state and layout handles.
type Model struct {
	BrowserContent func(*ui.Context)
	OnHideBrowser  func()
	// ExternalNavigation reuses NavigationShell instead of mounting a second rail.
	ExternalNavigation bool
	OnSelectSession    func(string)
	OnCreateSession    func()

	Projects              projects.Model
	Store                 *session.Store
	AgentPresetManager    *session.AgentPresetManager
	Interactions          *interaction.Store
	OnAutoAllowChanged    func(on bool)
	Sizes                 *LayoutSizes
	ChatList              *ui.ListState
	ThemeID               string
	SystemDark            bool
	Draft                 *string
	ComposerState         *composer.DraftState
	OnSubmitComposer      func()
	OnAttachComposerFiles func([]string)
	OnPasteComposerImage  func(bool)
	ModelPick             *string
	PermissionPick        *string
	AgentPresetPick       *string
	ThinkingExpanded      *map[string]bool
	RenameSessionID       *string
	RenameDraft           *string
	SessionSearch         *string

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
	FileEditing      *bool
	FileEditDraft    *string
	FileEditStatus   *string
	FileEditViewMode *files.EditViewMode
	FileEditSplit    *float32
	FilePaneWidth    *float32 // resolved width for responsive file-preview layout
	RightPaneOpen    bool     // explicit visibility; Sizes.Right remains remembered when closed

	RightPaneTab      *string // "files" | "diff" | "agents"
	OnOpenAgents      func()
	AgentsPromptDraft *string
	DiffPath          *string
	DiffText          *string
	DiffErr           *string

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
	OnListWorkspaceFiles        func(rel string) ([]workspacefs.Entry, error)
	OnListWorkspaceFilesContext workspacefs.RemoteDirectoryLister
	// OnReadWorkspaceFile is optional workspaceFiles/read fallback for preview.
	OnReadWorkspaceFile func(rel string) (string, error)
	// OnStatWorkspaceFile returns a short workspaceFiles/stat summary.
	OnStatWorkspaceFile  func(rel string) string
	FileStatNote         *string
	OnPickHeroWorkspace  func()
	OnNoDirectory        func()
	OnSelectWorkspace    func(string)
	OnBrowseWorkspace    func() // SidebarRoot ☰ / folder row
	OnRevealWorkspace    func()
	OnRenameWorkspace    func(title string)
	OnDeleteWorkspace    func()
	OnMoveSessionBefore  func(sessionID, beforeSessionID string)
	WorkspaceRenameDraft *string
	WorkspaceMenuOpen    *bool
	OnOpenDoc            func(absPath string, kind workspacefs.DocKind, inWebView bool)
	OnRevealFile         func(absPath string)
	OnOpenWithApp        func(absPath, appID string)
	FileOpenWithApps     []files.OpenWithApp
	OnRenderOfficePDF    func(rel string)
	OnOpenBrowserPreview func()
	OnOpenMarket         func()
	OnOpenSchedule       func()
	OnOpenAssistant      func()
	OnOpenLauncher       func()
	OnOpenSettings       func()
	OnToggleTools        func()
	OnToggleTerminal     func()
	OnRestartTerminal    func()
	OnCloseTerminal      func()

	Terminal              *termui.Panel
	TerminalShellsNote    string
	OnAddSelectionToChat  func(string) error
	CanAddSelectionToChat bool

	ConnectionBanner     string
	OnReconnect          func()
	UsageJobsToday       *int
	UsageBalance         string
	SessionCost          costui.Model
	OnOpenUsage          func()
	EditMessageSeq       *float64
	FeedbackCommentMsgID *string
	FeedbackCommentDraft *string
	PlanBoardOpen        *bool
	GoalDraft            *string
	// TodoPanelCollapsed mirrors Electron TodoPanel header chevron (composer dock).
	TodoPanelCollapsed           *bool
	ImageLightboxPath            *string
	OnOpenImageLightbox          func(displayPath string)
	OnCloseImageLightbox         func()
	OnOpenCitationURL            func(url string)
	OnOpenDeliverable            func(path string)
	VisualReply                  session.VisualReplyState
	VisualReplyBitmap            *ui.Bitmap
	OnPreviewVisualReply         func(attachmentID string)
	OnInteractVisualReply        func(attachmentID string)
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

	centerHeight float32 // resolved height below titlebar for terminal concessions
}

// View paints the main dialogue shell below the title bar.
func View(c *ui.Context, m Model) {
	if m.FileTabs != nil && (!m.RightPaneOpen || (m.RightPaneTab != nil && *m.RightPaneTab != "" && *m.RightPaneTab != "files")) {
		m.FileTabs.Pause()
	}
	if m.Store == nil {
		return
	}
	registerPageCommands(c, m)
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	border := chrome.Color(colors.Border)
	muted := chrome.Color(colors.Muted)
	snap := m.Store.Snapshot()
	if m.Interactions != nil {
		m.Interactions.SyncProjectedQuestions(snap.SelectedID, snap.QuestionProjection.Questions, snap.QuestionProjection.Inbox, snap.QuestionProjection.QuestionsSeq, snap.QuestionProjection.InboxSeq)
		syncQuestionHistory(m.Interactions, snap)
	}
	if m.Sizes == nil {
		fallback := DefaultLayout()
		m.Sizes = &fallback
	}
	viewport, _ := c.Size()
	cols := m.Sizes.ResolveColumns(viewport, m.RightPaneOpen)
	if m.FilePaneWidth == nil {
		m.FilePaneWidth = &cols.Details
	} else {
		*m.FilePaneWidth = cols.Details
	}

	// Grow below titlebar without percent-height double counting. Theme BG is
	// the center's surface; the sidebar and right track own their own surfaces.
	shell := ui.Column(c).Grow(1).MinHeight(0).Background(chrome.Color(colors.BG))
	_, height := c.Size()
	// The titlebar is above this shell. Its previous-frame origin remains
	// stable during a window resize; preserve a visible terminal below it.
	m.centerHeight = maxf(0, height-shell.Bounds().Y)
	shell.Children(func() {
		mainRow(c, m, snap, colors, border, muted)
	})
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
	createSession := m.OnCreateSession
	if createSession == nil {
		createSession = func() { m.Store.CreateSession(context.Background()) }
	}
	rename := m.Store.SessionRename()
	projectModel := m.Projects
	projectModel.OnResolvedFile = m.OnOpenDeliverable
	sidebar.View(c, sidebar.Model{
		Projects:             projectModel,
		Collapsed:            m.Sizes.Sidebar <= 0,
		OnToggle:             m.Sizes.ToggleSidebar,
		Snapshot:             snap,
		ThemeID:              m.ThemeID,
		SystemDark:           m.SystemDark,
		SearchQuery:          m.SessionSearch,
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
		OnCreate:                    createSession,
		OnOpenMarket:                m.OnOpenMarket,
		OnOpenSchedule:              m.OnOpenSchedule,
		OnOpenAssistant:             m.OnOpenAssistant,
		OnOpenLauncher:              m.OnOpenLauncher,
		OnOpenSettings:              m.OnOpenSettings,
		OnRetryWorkspacePreferences: m.Store.RetryWorkspacePreferences,
		OnDropWorkspaceBefore: func(generation uint64, id, before string) {
			m.Store.InsertWorkspaceBeforeForHost(context.Background(), generation, id, before)
		},
		OnSelectWorkspace:      m.OnSelectWorkspace,
		OnManageWorkspace:      m.Store.BeginWorkspaceManagement,
		OnWorkspaceGrouping:    m.Store.SetWorkspaceGrouping,
		OnWorkspaceOrdering:    m.Store.SetWorkspaceOrdering,
		OnToggleWorkspaceGroup: m.Store.ToggleWorkspaceGroup,
		OnMoveWorkspaceBefore:  func(id, before string) { m.Store.InsertWorkspaceBefore(context.Background(), id, before) },
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
	withNavigation := func(body func()) {
		if m.ExternalNavigation {
			body()
			return
		}
		sidebarColumns(c, m.Sizes, func() { navigationView(c, m, snap) }, body)
	}
	withNavigation(func() {
		// Electron desktop defaults to 2-column (rail | chat). Files/diff mounts
		// only when FilePaneWidth > 0 (tools toggle); otherwise chat fills.
		paintChat := func() {
			emptyHero := len(snap.Messages) == 0 && !hasSessionInteractions(m.Interactions, snap.SelectedID)
			// Electron chat chrome sits ~20 DIP lower than the rail brand (caption
			// overlay). Add top inset on the chat column only — not the rail.
			ui.Column(c).Grow(1).MinWidth(0).MinHeight(0).Children(func() {
				if emptyHero {
					// EmptyHero: blank header band (no Initialize Git / layout glyphs).
					// Keep a thin drag strip for the Windows caption region only.
					ui.Row(c).FillWidth().Height(40).Padding(0, 140, 0, 20).DragWindow().Children(func() {
						ui.Box(c).Grow(1).MinWidth(0)
					})
				} else {
					// Electron: caption row (workspace + actions) then 对话/轨迹 tabs.
					ui.Column(c).FillWidth().Children(func() {
						ui.Row(c).FillWidth().Padding(2, 16, 0, 16).AlignItems(ui.Center).Gap(10).DragWindow().Children(func() {
							title := m.WorkspaceRoot
							if title == "" {
								title = i18n.T(i18n.KeyAppName)
							} else if i := strings.LastIndexAny(title, `/\`); i >= 0 && i+1 < len(title) {
								title = title[i+1:]
							}
							ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(chrome.Color(colors.FG))
							if m.OnOpenLauncher != nil {
								if ui.Button(c, "···").Tooltip(i18n.T(i18n.KeyLauncherOpen)).
									FontSize(14).BorderWidth(0).Background(ui.Transparent).
									TextColor(muted).Clicked() {
									m.OnOpenLauncher()
								}
							} else {
								ui.Text(c, "···").FontSize(14).TextColor(muted)
							}
							ui.Box(c).Grow(1).MinWidth(0)
							agentsCaption := i18n.T(i18n.KeyFrameCaptionAgents)
							docsCaption := i18n.T(i18n.KeyFrameCaptionProjectDocs)
							if m.OnOpenAgents != nil {
								if ui.Button(c, agentsCaption).FontSize(12).BorderWidth(0).
									Background(ui.Transparent).TextColor(muted).Clicked() {
									m.OnOpenAgents()
								}
							} else if m.OnOpenAssistant != nil {
								if ui.Button(c, agentsCaption).FontSize(12).BorderWidth(0).
									Background(ui.Transparent).TextColor(muted).Clicked() {
									m.OnOpenAssistant()
								}
							} else {
								ui.Text(c, agentsCaption).FontSize(12).TextColor(muted)
							}
							if m.OnToggleTools != nil {
								if ui.Button(c, docsCaption).FontSize(12).BorderWidth(0).
									Background(ui.Transparent).TextColor(muted).Clicked() {
									m.OnToggleTools()
								}
							} else {
								ui.Text(c, docsCaption).FontSize(12).TextColor(muted)
							}
							if m.OnOpenLauncher != nil {
								if ui.Button(c, "⌂").Tooltip(i18n.T(i18n.KeyLauncherOpen)).
									FontSize(13).BorderWidth(0).Background(ui.Transparent).
									TextColor(muted).Clicked() {
									m.OnOpenLauncher()
								}
							}
							if m.OnToggleTerminal != nil {
								if ui.Button(c, "▤").Tooltip(i18n.T(i18n.KeyTerminalTitle)).
									FontSize(13).BorderWidth(0).Background(ui.Transparent).
									TextColor(muted).Clicked() {
									m.OnToggleTerminal()
								}
							}
							if m.OnToggleTools != nil {
								if ui.Button(c, "▦").Tooltip(i18n.T(i18n.KeyDesktopTools)).
									FontSize(13).BorderWidth(0).Background(ui.Transparent).
									TextColor(muted).Clicked() {
									m.OnToggleTools()
								}
							}
							if m.OnOpenSettings != nil {
								if ui.Button(c, "⚙").Tooltip(i18n.T(i18n.KeySettingsOpen)).
									FontSize(13).BorderWidth(0).Background(ui.Transparent).
									TextColor(muted).Clicked() {
									m.OnOpenSettings()
								}
							}
						})
						if !m.HideChatViewTabs {
							ui.Row(c).FillWidth().Padding(1, 16, 0, 16).AlignItems(ui.End).Gap(20).Children(func() {
								tab := "chat"
								if m.ChatPaneTab != nil && strings.TrimSpace(*m.ChatPaneTab) != "" {
									tab = strings.TrimSpace(*m.ChatPaneTab)
								}
								paintTab := func(id, label string) {
									active := tab == id
									ink := muted
									weight := 400
									if active {
										ink = chrome.Color(colors.Accent)
										weight = 700
									}
									ui.Column(c).Gap(3).Children(func() {
										b := ui.Button(c, label).FontSize(14).FontWeight(weight).
											BorderWidth(0).Background(ui.Transparent).TextColor(ink).
											Padding(0, 0, 6, 0)
										if b.Clicked() && m.ChatPaneTab != nil {
											*m.ChatPaneTab = id
										}
										if active {
											ui.Box(c).Width(38).Height(2).Background(chrome.Color(colors.Accent))
										}
									})
								}
								paintTab("chat", i18n.T(i18n.KeyChatPaneTabChat))
								paintTab("trajectory", i18n.T(i18n.KeyChatPaneTabTrajectory))
								ui.Box(c).Grow(1).MinWidth(0)
							})
							ui.Box(c).FillWidth().Height(1).Background(border)
						}
					})
				}
				// Prefer live commands/list from main; do not invent stub slash rows.
				slashCatalog := m.ComposerSlash
				filePaths := m.ComposerPaths
				if kind, _, _, _ := composer.ActiveSuggest(derefString(m.Draft)); kind == composer.SuggestMention {
					if len(filePaths) == 0 && m.OnIndexWorkspace != nil {
						filePaths = m.OnIndexWorkspace(strings.TrimSpace(m.WorkspaceRoot))
					}
				}
				editing := m.ComposerState != nil && m.ComposerState.EditSeq > 0
				paintComposer := func(hero bool) {
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
						Snapshot:      snap,
						ThemeID:       m.ThemeID,
						SystemDark:    m.SystemDark,
						WorkspaceRoot: m.WorkspaceRoot,
						Draft:         m.Draft,
						State:         m.ComposerState,
						OnSubmit:      m.OnSubmitComposer,
						OnOpenURL:     m.OnOpenCitationURL,
						OnAttachFiles: m.OnAttachComposerFiles,
						OnPasteImage:  m.OnPasteComposerImage,
						Editing:       editing,
						OnCancelEdit: func() {
							if m.ComposerState != nil {
								m.ComposerState.EndEdit()
							}
						},
						Hero:                 hero,
						ModelPick:            m.ModelPick,
						PermissionPick:       m.PermissionPick,
						AgentPresetPick:      m.AgentPresetPick,
						SlashCatalog:         slashCatalog,
						OnRetrySlashCommands: func() { m.Store.RefreshSlashCommands(context.Background()) },
						FilePaths:            filePaths,
						OnSearchFileRefs:     m.OnSearchFileRefs,
						OnSearchReferences:   m.OnSearchReferences,
						SuggestList:          m.ComposerSuggest,
						SkillsOpen:           m.ComposerSkillsOpen,
						Skills:               snap.SkillOptions,
						SkillsList:           m.ComposerSkillsList,
						SkillsFilter:         m.ComposerSkillsFilter,
						SkillHubHits:         snap.SkillHubHits,
						OnSearchSkillHub:     m.OnSearchSkillHub,
						OnInstallSkillHub:    m.OnInstallSkillHub,
						OnDeleteSkill:        m.OnDeleteSkill,
						SkillCreateName:      m.SkillCreateName,
						OnCreateSkill:        m.OnCreateSkill,
						OnToggleSkillModel:   m.OnToggleSkillModel,
						ChipEdit:             m.ComposerChipEdit,
						UseTextInputClient:   m.UseComposerTextInput,
						TextInput:            m.ComposerTextInput,
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
						OnStop: func() {
							m.Store.CancelRunning(context.Background())
						},
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
						OnPermissionChange: func(label string) {
							m.Store.SelectPermissionPreset(context.Background(), label)
						},
						OnAgentPresetChange: func(label string) {
							m.Store.SelectAgentPresetForSession(context.Background(), label)
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
						OnToggleSkills:      m.OnToggleComposerSkills,
						OnPickDirectory:     m.OnPickComposerDirectory,
						OnPickImage:         m.OnPickComposerImage,
						OnPickSpeech:        m.OnPickComposerSpeech,
						Voice:               m.ComposerVoice,
						OnStartVoice:        m.OnStartComposerVoice,
						OnStopVoice:         m.OnStopComposerVoice,
						OnCancelVoice:       m.OnCancelComposerVoice,
						OnInsertVoice:       m.OnInsertComposerVoice,
						OnCycleSpeech:       m.OnCycleComposerSpeech,
						OnPickHeroWorkspace: m.OnPickHeroWorkspace,
						OnNoDirectory:       m.OnNoDirectory,
						OnPromptModeChange: func(mode session.PromptMode) {
							m.Store.SetPromptMode(mode)
						},
					})
				}
				colW := float32(0)
				if m.Sizes != nil {
					colW = m.Sizes.Center
				}
				contentAxis := conversationContentWidth(colW)
				// Transcript clears32 per side. InputBar owns its16 side pads,
				// so its outer envelope must be content+64 to produce a card W+32.
				chatContentW := min(contentAxis, maxf(0, colW-64))
				composerOuterW := min(contentAxis+64, maxf(0, colW))
				heroStackW := min(contentAxis+32, maxf(0, colW-48))
				if emptyHero {
					// HeroShell.module.css .root: pad 0 24; .stack gap 12.
					ui.Column(c).Grow(1).MinHeight(0).Padding(0, 24, 0, 24).Children(func() {
						ui.Box(c).Grow(1).MinHeight(0)
						ui.Row(c).FillWidth().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(heroStackW).MaxWidthPercent(100).MinWidth(0).Gap(12).Children(func() {
								ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
									ui.Box(c).Grow(1).MinWidth(0)
									if bmp := chrome.FishLogo(); bmp != nil {
										ui.Image(c, bmp).Width(34).Height(25).Fit(ui.Contain)
									}
									ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
										ui.Text(c, i18n.T(i18n.KeyChatHeroSlogan)).FontSize(26).FontWeight(500).
											TextColor(chrome.Color(colors.FG))
										previewFG := chrome.Color("rgb(70, 110, 160)")
										previewBG := chrome.Color("rgba(120, 160, 210, 0.18)")
										previewBD := chrome.Color("rgba(120, 160, 210, 0.35)")
										heroTh := theme.Resolve(m.ThemeID)
										if heroTh.Scheme == theme.SchemeDark || (heroTh.Scheme == theme.SchemeSystem && m.SystemDark) {
											previewFG = chrome.Color(colors.Accent)
											previewBG = chrome.Color("rgba(103, 158, 254, 0.16)")
											previewBD = chrome.Color("rgba(103, 158, 254, 0.35)")
										}
										ui.Text(c, i18n.T(i18n.KeyChatHeroPreview)).FontSize(12).FontWeight(500).
											TextColor(previewFG).
											Padding(1, 7, 0, 7).
											Background(previewBG).
											Border(1, previewBD).
											Radius(999)
									})
									ui.Box(c).Grow(1).MinWidth(0)
								})
								paintComposer(true)
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
						ui.Box(c).Grow(1).MinHeight(0)
					})
					return
				}
				// ConversationRoot: same chatContent column for transcript + docked composer.
				ui.Column(c).Grow(1).MinHeight(0).Padding(20, 0, 0, 0).Children(func() {
					paneTab := "chat"
					if m.ChatPaneTab != nil && strings.TrimSpace(*m.ChatPaneTab) != "" {
						paneTab = strings.TrimSpace(*m.ChatPaneTab)
					}
					ui.Box(c).Grow(1).MinHeight(0).Clip().Children(func() {
						ui.Row(c).Fill().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(chatContentW).FillHeight().MinWidth(0).Children(func() {
								if paneTab == "trajectory" {
									// Electron ui-trajectory: toolbar + timeline + ledger (not PlanBoard).
									chat.TrajectoryView(c, chat.TrajectoryModel{
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
										Snapshot:         snap,
										ContentWidth:     contentAxis,
										ThemeID:          m.ThemeID,
										SystemDark:       m.SystemDark,
										List:             m.ChatList,
										ThinkingExpanded: m.ThinkingExpanded,
										ConnectionBanner: m.ConnectionBanner,
										OnReconnect:      m.OnReconnect,
										UsageJobsToday:   m.UsageJobsToday,
										UsageBalance:     m.UsageBalance,
										OnOpenUsage:      m.OnOpenUsage,
										FeedbackStore:    m.Store,
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
										OnOpenDeliverable:            m.OnOpenDeliverable,
										VisualReply:                  m.VisualReply,
										VisualReplyBitmap:            m.VisualReplyBitmap,
										OnPreviewVisualReply:         m.OnPreviewVisualReply,
										OnInteractVisualReply:        m.OnInteractVisualReply,
										OnStopInteractiveVisualReply: m.OnStopInteractiveVisualReply,
										OnReadVisualReply:            m.OnReadVisualReply,
										OnSaveVisualReply:            m.OnSaveVisualReply,
										OnCloseVisualReply:           m.OnCloseVisualReply,
									})
								}
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
					})
					if paneTab != "trajectory" {
						// Goal/todo/takeover cards share the transcript content axis.
						ui.Row(c).FillWidth().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(chatContentW).MaxWidthPercent(100).MinWidth(0).Children(func() {
								chat.PlanBoardView(c, chat.PlanBoardModel{
									Board:          snap.PlanBoard,
									LoadQueueImage: m.Store.QueueImageLoader(snap.SelectedID, snap.PlanBoard.QueueActions.Scope),
									ThemeID:        m.ThemeID,
									SystemDark:     m.SystemDark,
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
									QueueEditor:       snap.QueueEditor,
									OnQueueEdit:       func(itemID, _ string) { m.Store.BeginQueueEdit(context.Background(), itemID) },
									OnChangeQueueEdit: m.Store.ChangeQueueEdit,
									OnCancelQueueEdit: m.Store.CancelQueueEdit,
									OnSaveQueueEdit:   func(token uint64) { m.Store.SaveQueueEdit(context.Background(), token) },
								})
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
					}
					if m.Interactions != nil {
						ui.Row(c).FillWidth().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(chatContentW).MaxWidthPercent(100).MinWidth(0).Children(func() {
								interui.View(c, interui.Model{
									Snapshot:           m.Interactions.SnapshotForAgent(snap.SelectedID),
									ThemeID:            m.ThemeID,
									SystemDark:         m.SystemDark,
									Store:              m.Interactions,
									OnAutoAllowChanged: m.OnAutoAllowChanged,
								})
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
					}
					if len(snap.BandActions) > 0 {
						ui.Row(c).FillWidth().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(chatContentW).MaxWidthPercent(100).MinWidth(0).Gap(6).Padding(0, 0, 6, 0).Children(func() {
								ui.Text(c, i18n.T(i18n.KeyChatBandTitle)).FontSize(11).TextColor(muted)
								ui.Row(c).FillWidth().Gap(6).Children(func() {
									for _, a := range snap.BandActions {
										act := a
										if ui.Button(c, act.Label).FontSize(11).Clicked() {
											m.Store.PressBand(context.Background(), act.ID)
										}
									}
								})
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
					}
					if note := strings.TrimSpace(snap.ControlNote); note != "" {
						ui.Row(c).FillWidth().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(chatContentW).MaxWidthPercent(100).MinWidth(0).Padding(0, 0, 4, 0).Children(func() {
								ui.Text(c, i18n.T(i18n.KeyChatControlNote)+": "+note).FontSize(11).TextColor(muted)
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
					}
					// Electron keeps the composer seat mounted on the trajectory tab
					// (data-conversation-composer-overlay); do not hide it.
					ui.Row(c).FillWidth().Children(func() {
						ui.Box(c).Grow(1).MinWidth(0)
						ui.Column(c).Width(composerOuterW).MaxWidthPercent(100).MinWidth(0).Children(func() {
							paintComposer(false)
							costModel := m.SessionCost
							costModel.Snapshot = snap
							costui.View(c, costModel, muted)
						})
						ui.Box(c).Grow(1).MinWidth(0)
					})
				})
			})
		}
		paintCenter := func() {
			centerTerminalColumn(c, m, paintChat, func() {
				termui.View(c, termui.Model{
					Panel: m.Terminal, ThemeID: m.ThemeID, SystemDark: m.SystemDark,
					ShellsNote: m.TerminalShellsNote, OnRestart: m.OnRestartTerminal, OnClose: m.OnCloseTerminal,
					OnAddToChat: m.OnAddSelectionToChat, CanAddToChat: m.CanAddSelectionToChat,
				})
			})
		}
		centerRightColumns(c, m, paintCenter, func() { rightPane(c, m) })
	})
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
	field := chrome.Color(colors.Sidebar)
	border := chrome.Color(colors.Border)
	muted := chrome.Color(colors.Muted)
	fg := chrome.Color(colors.FG)
	accent := chrome.Color(colors.Accent)

	tab := "files"
	if m.RightPaneTab != nil {
		switch *m.RightPaneTab {
		case "diff", "agents", "browser":
			tab = *m.RightPaneTab
		}
	}

	ui.Column(c).Fill().MinWidth(0).Background(field).Children(func() {
		ui.Row(c).Padding(8, 8, 0, 8).Gap(4).FillWidth().Children(func() {
			paintRightTab(c, tab == "files", i18n.T(i18n.KeyRightPaneFiles), fg, muted, accent, func() {
				if m.RightPaneTab != nil {
					*m.RightPaneTab = "files"
				}
			})
			paintRightTab(c, tab == "diff", i18n.T(i18n.KeyRightPaneDiff), fg, muted, accent, func() {
				if m.RightPaneTab != nil {
					*m.RightPaneTab = "diff"
				}
			})
			paintRightTab(c, tab == "agents", i18n.T(i18n.KeyRightPaneAgents), fg, muted, accent, func() {
				if m.RightPaneTab != nil {
					*m.RightPaneTab = "agents"
				}
			})
			if m.OnOpenBrowserPreview != nil {
				ui.Box(c).Grow(1)
				paintRightTab(c, tab == "browser", i18n.T(i18n.KeyBrowserPreviewTitle), fg, muted, accent, m.OnOpenBrowserPreview)
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)
		if tab == "browser" && m.BrowserContent != nil {
			m.BrowserContent(c)
			return
		}
		if m.OnHideBrowser != nil {
			m.OnHideBrowser()
		}
		if tab == "agents" && m.Store != nil {
			store := m.Store
			agents.View(c, agents.Model{
				Manager:     m.AgentPresetManager,
				Snapshot:    store.Snapshot(),
				ThemeID:     m.ThemeID,
				SystemDark:  m.SystemDark,
				PromptDraft: m.AgentsPromptDraft,
				OnSelect:    store.Select,
				OnStop: func(sessionID string) {
					store.CancelSession(context.Background(), sessionID)
				},
				OnInterrupt: func(parentID, childID string) {
					store.InterruptChild(context.Background(), parentID, childID, "cancel")
				},
				OnPrompt: func(parentID, childID, text string) {
					store.PromptChild(context.Background(), parentID, childID, text)
				},
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
						Tabs:          m.FileTabs,
						BindingID:     m.FileBindingID,
						WorkspaceRoot: m.WorkspaceRoot,
						SelectedFile:  m.SelectedFile,
						Preview:       m.FilePreview,
						PreviewErr:    m.FilePreviewErr,
						Editing:       m.FileEditing,
						EditDraft:     m.FileEditDraft,
						EditStatus:    m.FileEditStatus,
						StatNote:      m.FileStatNote,
						ReadRemote:    m.OnReadWorkspaceFile,
						StatRemote:    m.OnStatWorkspaceFile,
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
		files.View(c, files.Model{
			Tabs:              m.FileTabs,
			BindingID:         m.FileBindingID,
			WorkspaceRoot:     m.WorkspaceRoot,
			ThemeID:           m.ThemeID,
			SystemDark:        m.SystemDark,
			Expanded:          m.FileExpanded,
			SelectedFile:      m.SelectedFile,
			Filter:            m.FileFilter,
			Preview:           m.FilePreview,
			PreviewErr:        m.FilePreviewErr,
			PreviewList:       m.FilePreviewList,
			PreviewSplit:      &m.Sizes.RightTree,
			Editing:           m.FileEditing,
			EditDraft:         m.FileEditDraft,
			EditStatus:        m.FileEditStatus,
			EditViewMode:      m.FileEditViewMode,
			EditSplit:         m.FileEditSplit,
			PaneWidth:         m.FilePaneWidth,
			ListRemote:        m.OnListWorkspaceFiles,
			ListRemoteContext: m.OnListWorkspaceFilesContext,
			ReadRemote:        m.OnReadWorkspaceFile,
			StatRemote:        m.OnStatWorkspaceFile,
			StatNote:          m.FileStatNote,
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
			OnOpenDoc:    m.OnOpenDoc,
			OnReveal:     m.OnRevealFile,
			OnOpenWith:   m.OnOpenWithApp,
			OpenWithApps: m.FileOpenWithApps,
			OnRenderPDF:  m.OnRenderOfficePDF,
			OnAddToChat:  m.OnAddSelectionToChat,
			CanAddToChat: m.CanAddSelectionToChat,
			OnOpenLink:   m.OnOpenCitationURL,
		})
	})
}

func paintRightTab(c *ui.Context, active bool, label string, fg, muted, accent ui.Color, onSelect func()) {
	color := muted
	weight := 400
	if active {
		color = accent
		weight = 700
	}
	ui.Column(c).Gap(3).Padding(0, 4, 0, 4).Children(func() {
		b := ui.Button(c, label).FontSize(12).FontWeight(weight).
			BorderWidth(0).Background(ui.Transparent).TextColor(color).
			Padding(4, 4, 2, 4)
		if b.Clicked() && onSelect != nil {
			onSelect()
		}
		if active {
			ui.Box(c).Height(2).Background(accent).FillWidth()
		}
	})
	_ = fg
}
