// Package frame is the M4 main shell: resizable sidebar, chat, optional
// right and bottom regions.
package frame

import (
	"context"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

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
	"whaleisle/internal/ui/sidebar"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
)

// LayoutSizes persist split pane dimensions (DIPs) across frames.
type LayoutSizes struct {
	Sidebar   float32
	Center    float32 // width of chat column before the optional right pane
	Main      float32 // height of the dialogue column when the terminal is open
	Bottom    float32 // >40 shows the terminal (saved height preference when hidden)
	RightTree float32 // height of file tree above preview in the right pane
}

// Desktop ui-layout columns.ts contract (anywhere-labs harness-desktop).
const (
	SidebarMin       = float32(264)
	SidebarMax       = float32(420)
	SidebarDefault   = float32(280)
	SidebarCollapsed = float32(56)
	CenterMin        = float32(640) // protected while details is open
	DetailsMin       = float32(300)
	DetailsMax       = float32(520)
	DetailsDefault   = float32(360)
)

// Columns is the resolved three-column geometry for one frame.
type Columns struct {
	Sidebar float32
	Center  float32
	Details float32 // 0 = visually closed (preference may still be open)
}

// ComputeColumns mirrors ui-layout columns.ts computeColumns.
// detailsPref 0 closes the right track; otherwise it is clamped to DETAILS_MIN..MAX
// and may shrink / auto-close to keep Center >= CenterMin.
func ComputeColumns(viewport, sidebarPref, detailsPref float32) Columns {
	s := clampSidebarWidth(sidebarPref)
	if sidebarPref <= 0 {
		s = SidebarCollapsed
	}
	if detailsPref <= 0 {
		return Columns{Sidebar: s, Center: maxf(0, viewport-s), Details: 0}
	}
	d0 := detailsPref
	if d0 < DetailsMin {
		d0 = DetailsMin
	}
	if d0 > DetailsMax {
		d0 = DetailsMax
	}
	// Step 1: preferred widths fit.
	if s+d0+CenterMin <= viewport {
		return Columns{Sidebar: s, Center: viewport - s - d0, Details: d0}
	}
	// Step 2: shrink details toward minimum.
	d1 := viewport - s - CenterMin
	if d1 < DetailsMin {
		d1 = DetailsMin
	}
	if s+d1+CenterMin <= viewport {
		return Columns{Sidebar: s, Center: CenterMin, Details: d1}
	}
	// Step 3: auto-close details; center absorbs the deficit.
	return Columns{Sidebar: s, Center: maxf(0, viewport-s), Details: 0}
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
	Store              *session.Store
	Interactions       *interaction.Store
	OnAutoAllowChanged func(on bool)
	Sizes              *LayoutSizes
	ChatList           *ui.ListState
	ThemeID            string
	SystemDark         bool
	Draft              *string
	ModelPick          *string
	PermissionPick     *string
	AgentPresetPick    *string
	ThinkingExpanded   *map[string]bool
	RenameSessionID    *string
	RenameDraft        *string
	SessionSearch      *string

	WorkspaceRoot    string
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
	FilePaneWidth    *float32

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
	// OnListWorkspaceFiles is optional workspaceFiles/list fallback for the files pane.
	OnListWorkspaceFiles func(rel string) ([]workspacefs.Entry, error)
	// OnReadWorkspaceFile is optional workspaceFiles/read fallback for preview.
	OnReadWorkspaceFile func(rel string) (string, error)
	// OnStatWorkspaceFile returns a short workspaceFiles/stat summary.
	OnStatWorkspaceFile func(rel string) string
	FileStatNote        *string
	OnPickHeroWorkspace     func()
	OnBrowseWorkspace       func() // SidebarRoot ☰ / folder row
	OnRevealWorkspace       func()
	OnRenameWorkspace       func(title string)
	OnDeleteWorkspace       func()
	OnMoveSessionBefore     func(sessionID, beforeSessionID string)
	WorkspaceRenameDraft    *string
	WorkspaceMenuOpen       *bool
	OnOpenDoc               func(absPath string, kind workspacefs.DocKind, inWebView bool)
	OnRevealFile            func(absPath string)
	OnOpenWithApp           func(absPath, appID string)
	FileOpenWithApps        []files.OpenWithApp
	OnRenderOfficePDF       func(rel string)
	OnOpenBrowserPreview    func()
	OnOpenMarket            func()
	OnOpenSchedule          func()
	OnOpenAssistant         func()
	OnOpenLauncher          func()
	OnOpenSettings          func()
	OnToggleTools           func()
	OnToggleTerminal        func()
	OnRestartTerminal       func()

	Terminal           *termui.Panel
	TerminalShellsNote string

	ConnectionBanner     string
	OnReconnect          func()
	UsageJobsToday       *int
	UsageBalance         string
	OnOpenUsage          func()
	EditMessageSeq       *float64
	FeedbackCommentMsgID *string
	FeedbackCommentDraft *string
	PlanBoardOpen        *bool
	GoalDraft            *string
	// TodoPanelCollapsed mirrors Electron TodoPanel header chevron (composer dock).
	TodoPanelCollapsed   *bool
	ImageLightboxPath    *string
	OnOpenImageLightbox  func(displayPath string)
	OnCloseImageLightbox func()
	OnOpenCitationURL    func(url string)
	OnOpenDeliverable    func(path string)
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
}

// View paints the main dialogue shell below the title bar.
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	border := chrome.Color(colors.Border)
	muted := chrome.Color(colors.Muted)
	snap := m.Store.Snapshot()
	// Electron desk wash (sampled accept crops / D-MAIN): cool grey ~#DDE2E9.
	deskWash := ui.LinearGradient{
		From:  chrome.Color("rgb(221, 226, 233)"),
		To:    chrome.Color("rgb(206, 214, 222)"),
		Angle: 135,
		End:   0.88,
		Oklab: false,
	}
	if th.Scheme == theme.SchemeDark || (th.Scheme == theme.SchemeSystem && m.SystemDark) {
		deskWash = ui.LinearGradient{
			From:  chrome.Color("rgb(36, 40, 48)"),
			To:    chrome.Color("rgb(28, 32, 38)"),
			Angle: 135,
			End:   0.88,
			Oklab: false,
		}
	}

	// Grow into remaining height under the title bar. Do not combine with Fill()
	// height (percent 100): that double-counts and pushes the shell off-screen.
	ui.Column(c).Grow(1).MinHeight(0).LinearGradient(deskWash).Children(func() {
		if m.Sizes.Bottom > 40 {
			if m.Sizes.Main < 80 {
				m.Sizes.Main = DefaultLayout().Main
			}
			// First pane = dialogue (Main height); second pane = terminal (grows).
			ui.SplitVertical(c, &m.Sizes.Main, func() {
				mainRow(c, m, snap, colors, border, muted)
			}, func() {
				termui.View(c, termui.Model{
					Panel:      m.Terminal,
					ThemeID:    m.ThemeID,
					SystemDark: m.SystemDark,
					ShellsNote: m.TerminalShellsNote,
					OnRestart:  m.OnRestartTerminal,
				})
			}).Grow(1).MinHeight(0)
		} else {
			mainRow(c, m, snap, colors, border, muted)
		}
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

func mainRow(c *ui.Context, m Model, snap session.ViewState, colors theme.Colors, border, muted ui.Color) {
	if m.Sizes != nil {
		m.Sizes.Sidebar = clampSidebarWidth(m.Sizes.Sidebar)
	}
	ui.Split(c, &m.Sizes.Sidebar, func() {
		sidebar.View(c, sidebar.Model{
			Snapshot:        snap,
			ThemeID:         m.ThemeID,
			SystemDark:      m.SystemDark,
			SearchQuery:     m.SessionSearch,
			WorkspaceLabel:  m.WorkspaceRoot,
			RenameSessionID: derefString(m.RenameSessionID),
			RenameDraft:     m.RenameDraft,
			OnSelect:        m.Store.Select,
			OnRefresh: func() {
				m.Store.RefreshList(context.Background())
			},
			OnCreate: func() {
				m.Store.CreateSession(context.Background())
			},
			OnOpenMarket:      m.OnOpenMarket,
			OnOpenSchedule:    m.OnOpenSchedule,
			OnOpenAssistant:   m.OnOpenAssistant,
			OnOpenLauncher:    m.OnOpenLauncher,
			OnOpenSettings:    m.OnOpenSettings,
			OnBrowseWorkspace: m.OnBrowseWorkspace,
			OnRevealWorkspace:    m.OnRevealWorkspace,
			OnRenameWorkspace:    m.OnRenameWorkspace,
			OnDeleteWorkspace:    m.OnDeleteWorkspace,
			OnMoveSessionBefore:  m.OnMoveSessionBefore,
			WorkspaceRenameDraft: m.WorkspaceRenameDraft,
			WorkspaceMenuOpen:    m.WorkspaceMenuOpen,
			OnBeginRename: func(sessionID, title string) {
				if m.RenameSessionID != nil {
					*m.RenameSessionID = sessionID
				}
				if m.RenameDraft != nil {
					*m.RenameDraft = title
				}
			},
			OnCommitRename: func(sessionID, title string) {
				m.Store.RenameSession(context.Background(), sessionID, title)
				if m.RenameSessionID != nil {
					*m.RenameSessionID = ""
				}
				if m.RenameDraft != nil {
					*m.RenameDraft = ""
				}
			},
			OnCancelRename: func() {
				if m.RenameSessionID != nil {
					*m.RenameSessionID = ""
				}
				if m.RenameDraft != nil {
					*m.RenameDraft = ""
				}
			},
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
	}, func() {
		// Electron desktop defaults to 2-column (rail | chat). Files/diff mounts
		// only when FilePaneWidth > 0 (tools toggle); otherwise chat fills.
		paintChat := func() {
			emptyHero := len(snap.Messages) == 0
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
				editing := m.EditMessageSeq != nil && *m.EditMessageSeq > 0
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
						Snapshot:           snap,
						ThemeID:            m.ThemeID,
						SystemDark:         m.SystemDark,
						WorkspaceRoot:      m.WorkspaceRoot,
						Draft:              m.Draft,
						Editing:            editing,
						Hero:               hero,
						ModelPick:          m.ModelPick,
						PermissionPick:     m.PermissionPick,
						AgentPresetPick:    m.AgentPresetPick,
						SlashCatalog:       slashCatalog,
						FilePaths:          filePaths,
						OnSearchFileRefs:   m.OnSearchFileRefs,
						SuggestList:        m.ComposerSuggest,
						SkillsOpen:         m.ComposerSkillsOpen,
						Skills:             snap.SkillOptions,
						SkillsList:         m.ComposerSkillsList,
						SkillsFilter:       m.ComposerSkillsFilter,
						SkillHubHits:       snap.SkillHubHits,
						OnSearchSkillHub:   m.OnSearchSkillHub,
						OnInstallSkillHub:  m.OnInstallSkillHub,
						OnDeleteSkill:      m.OnDeleteSkill,
						SkillCreateName:    m.SkillCreateName,
						OnCreateSkill:      m.OnCreateSkill,
						OnToggleSkillModel: m.OnToggleSkillModel,
						ChipEdit:           m.ComposerChipEdit,
						UseTextInputClient: m.UseComposerTextInput,
						TextInput:          m.ComposerTextInput,
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
							m.Store.SelectModelForSession(context.Background(), label)
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
						OnCycleSpeech:       m.OnCycleComposerSpeech,
						OnPickHeroWorkspace: m.OnPickHeroWorkspace,
						OnPromptModeChange: func(mode session.PromptMode) {
							m.Store.SetPromptMode(mode)
						},
					})
				}
				colW := float32(0)
				if m.Sizes != nil {
					colW = m.Sizes.Center
				}
				chatContentW := conversationContentWidth(colW)
				composerCardW := chatContentW + 32
				if emptyHero {
					// HeroShell.module.css .root: pad 0 24; .stack gap 12.
					ui.Column(c).Grow(1).MinHeight(0).Padding(0, 24, 0, 24).Children(func() {
						ui.Box(c).Grow(1).MinHeight(0)
						ui.Row(c).FillWidth().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(composerCardW).Gap(12).Children(func() {
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
							ui.Column(c).Width(chatContentW).Fill().MinWidth(0).Children(func() {
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
										ThemeID:          m.ThemeID,
										SystemDark:       m.SystemDark,
										List:             m.ChatList,
										ThinkingExpanded: m.ThinkingExpanded,
										ConnectionBanner: m.ConnectionBanner,
										OnReconnect:      m.OnReconnect,
										UsageJobsToday:   m.UsageJobsToday,
										UsageBalance:     m.UsageBalance,
										OnOpenUsage:      m.OnOpenUsage,
										OnMessageFeedback: func(messageID, rating, note string) {
											m.Store.RateMessage(context.Background(), messageID, rating, note)
										},
										OnClearMessageFeedback: func(messageID string) {
											m.Store.ClearMessageFeedback(context.Background(), messageID)
										},
										FeedbackCommentMsgID: m.FeedbackCommentMsgID,
										FeedbackCommentDraft: m.FeedbackCommentDraft,
										OnEditMessage: func(seq float64, body string) {
											if m.Draft != nil {
												*m.Draft = body
											}
											if m.EditMessageSeq != nil {
												*m.EditMessageSeq = seq
											}
										},
										OnQuoteMessage: func(body string) {
											chat.QuoteIntoDraft(m.Draft, body)
										},
										OnLoadEarlier: func() {
											m.Store.LoadEarlierHistory(context.Background())
										},
										ImageLightboxPath:    m.ImageLightboxPath,
										OnOpenImageLightbox:  m.OnOpenImageLightbox,
										OnCloseImageLightbox: m.OnCloseImageLightbox,
										OnOpenCitationURL:    m.OnOpenCitationURL,
										OnOpenDeliverable:    m.OnOpenDeliverable,
									})
								}
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
					})
					if paneTab != "trajectory" {
						// Electron GoalBar width tracks --dsh-composer-card-max-width.
						ui.Row(c).FillWidth().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(composerCardW).Children(func() {
								chat.PlanBoardView(c, chat.PlanBoardModel{
									Board:      snap.PlanBoard,
									ThemeID:    m.ThemeID,
									SystemDark: m.SystemDark,
									Open:       m.PlanBoardOpen,
									GoalDraft:  m.GoalDraft,
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
									OnQueueEdit: func(itemID, preview string) {
										if m.Draft != nil {
											*m.Draft = preview
										}
										m.Store.UpdateQueue(context.Background(), snap.SelectedID, itemID, session.QueueActionEdit, preview)
									},
								})
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
					}
					if m.Interactions != nil {
						ui.Row(c).FillWidth().Children(func() {
							ui.Box(c).Grow(1).MinWidth(0)
							ui.Column(c).Width(composerCardW).Children(func() {
								interui.View(c, interui.Model{
									Snapshot:           m.Interactions.Snapshot(),
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
							ui.Column(c).Width(composerCardW).Gap(6).Padding(0, 0, 6, 0).Children(func() {
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
							ui.Column(c).Width(composerCardW).Padding(0, 0, 4, 0).Children(func() {
								ui.Text(c, i18n.T(i18n.KeyChatControlNote)+": "+note).FontSize(11).TextColor(muted)
							})
							ui.Box(c).Grow(1).MinWidth(0)
						})
					}
					// Electron keeps the composer seat mounted on the trajectory tab
					// (data-conversation-composer-overlay); do not hide it.
					ui.Row(c).FillWidth().Children(func() {
						ui.Box(c).Grow(1).MinWidth(0)
						ui.Column(c).Width(composerCardW).Children(func() {
							paintComposer(false)
						})
						ui.Box(c).Grow(1).MinWidth(0)
					})
				})
			})
		}
		showRight := m.FilePaneWidth != nil && *m.FilePaneWidth >= DetailsMin
		if showRight {
			// Keep chat column ≥ CENTER_MIN while details is open (ui-layout).
			if m.Sizes != nil && m.Sizes.Center < CenterMin {
				m.Sizes.Center = CenterMin
			}
			ui.Split(c, &m.Sizes.Center, func() {
				paintChat()
			}, func() {
				rightPane(c, m)
			}).Grow(1).MinWidth(0).MinHeight(0)
			return
		}
		paintChat()
	}).Grow(1).MinWidth(0).MinHeight(0)
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// conversationContentWidth mirrors ui-layout chatContent = clamp(680, 64% col, 920),
// then never exceeds the dialogue column (minus ~32 DIP side clearance for the card).
func conversationContentWidth(colW float32) float32 {
	if colW < 400 {
		colW = 920
	}
	w := colW * 0.64
	if w < 680 {
		w = 680
	}
	if w > 920 {
		w = 920
	}
	// Keep composerCardW (= content + 32) inside the column when the desk is narrow.
	maxInCol := colW - 32
	if maxInCol < 240 {
		maxInCol = 240
	}
	if w > maxInCol {
		w = maxInCol
	}
	return w
}

func rightPane(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	field := chrome.Color(colors.Field)
	border := chrome.Color(colors.Border)
	muted := chrome.Color(colors.Muted)
	fg := chrome.Color(colors.FG)
	accent := chrome.Color(colors.Accent)

	tab := "files"
	if m.RightPaneTab != nil {
		switch *m.RightPaneTab {
		case "diff", "agents":
			tab = *m.RightPaneTab
		}
	}

	ui.Column(c).Fill().MinWidth(0).Background(field).Children(func() {
		ui.Row(c).Padding(8, 8, 0, 8).Gap(4).Fill().Children(func() {
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
				if ui.Button(c, i18n.T(i18n.KeyBrowserPreviewOpen)).Clicked() {
					m.OnOpenBrowserPreview()
				}
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)
		if tab == "agents" && m.Store != nil {
			store := m.Store
			agents.View(c, agents.Model{
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
			WorkspaceRoot: m.WorkspaceRoot,
			ThemeID:       m.ThemeID,
			SystemDark:    m.SystemDark,
			Expanded:      m.FileExpanded,
			SelectedFile:  m.SelectedFile,
			Filter:        m.FileFilter,
			Preview:       m.FilePreview,
			PreviewErr:    m.FilePreviewErr,
			PreviewList:   m.FilePreviewList,
			PreviewSplit:  &m.Sizes.RightTree,
			Editing:       m.FileEditing,
			EditDraft:     m.FileEditDraft,
			EditStatus:    m.FileEditStatus,
			EditViewMode:  m.FileEditViewMode,
			EditSplit:     m.FileEditSplit,
			PaneWidth:     m.FilePaneWidth,
			ListRemote:    m.OnListWorkspaceFiles,
			ReadRemote:    m.OnReadWorkspaceFile,
			StatRemote:    m.OnStatWorkspaceFile,
			StatNote:      m.FileStatNote,
			OnShowDiff: func(rel string) {
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
