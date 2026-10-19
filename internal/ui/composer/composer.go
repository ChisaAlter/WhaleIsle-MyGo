// Package composer is the multiline prompt input for M4–M6.
package composer

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// Model is composer state for one frame.
type Model struct {
	GroupMembers         []session.GroupMember
	NextPrompt           session.NextPromptSuggestion
	OnAcceptNextPrompt   func() string
	ComposerBeam         bool
	ComposerBeamStyle    json.RawMessage
	TypingFx             bool
	TypingFxStyle        json.RawMessage
	ResizeEnabled        bool
	Resize               *ResizeState
	ResizeMaxWidth       float32
	OnResizeEnd          func(axis string, value float32)
	Snapshot             session.ViewState
	ThemeID              string
	SystemDark           bool
	WorkspaceRoot        string
	Draft                *string
	State                *DraftState
	OnSubmit             func()
	OnSubmitGesture      func(accelerated bool)
	OnOpenURL            func(string)
	OnOpenReference      func(path string) // existing session-owned file preview route
	OnAttachFiles        func([]string)
	OnRetryAttachment    func(int)
	OnWorkspaceReference func(filelink.Reference) error
	OnPasteImage         func(bool)
	ModelPick            *string
	PermissionOptions    []session.PermissionOption
	AgentPresetOptions   []session.AgentPresetOption
	AgentPresetSelection session.AgentPresetSelectionView
	PermissionPick       *string
	AgentPresetPick      *string

	OnRetrySlashCommands func()
	SlashCatalog         []SlashCommand
	FilePaths            []string
	SuggestList          *ui.ListState

	SkillsOpen   bool
	Skills       []session.SkillOption
	SkillsList   *ui.ListState
	SkillsFilter *string
	SkillHubHits []session.SkillHubHit
	// OnSearchSkillHub maps to skillInventory/searchHub.
	OnSearchSkillHub func(query string)
	// OnInstallSkillHub maps to skillInventory/installHub.
	OnInstallSkillHub func(hit session.SkillHubHit)
	// OnDeleteSkill maps to skillInventory/delete for writable skills.
	OnDeleteSkill func(name string)
	// SkillCreateName is optional draft for skillInventory/create.
	SkillCreateName *string
	OnCreateSkill   func(name string)
	// OnToggleSkillModel maps to skillInventory/setInvocation (flip modelInvocable).
	OnToggleSkillModel func(name string, modelInvocable bool)

	// ChipEdit is optional popup state for rewriting one @path / /skill chip.
	ChipEdit *ChipEdit

	// UseTextInputClient paints the shipped native DraftClient path. Main enables
	// it; existing DraftClient ownership is retained. TextArea is a fallback.
	UseTextInputClient bool
	TextInput          *DraftClient

	OnSend              func(text string)
	OnStop              func()
	StopBlocked         bool // pending approval/question owns response input
	stopSuggestionsOpen bool
	OnModelChange       func(label string)
	OnModelEffortChange func(label string)
	OnRetryModels       func()
	OnRetryPermissions  func()
	OnExitPlanMode      func()
	OnPermissionChange  func(value string)
	OnAgentPresetChange func(id string)
	OnClearDraft        func()
	OnNewSession        func()
	OnExecuteSlash      func(ctx context.Context, line string) // remote commands/execute
	OnToggleSkills      func(open bool)
	OnPickDirectory     func()
	// OnPickImage attaches a local image path into the draft (Electron InputBar + menu).
	OnPickImage func()
	// OnPickSpeech picks an audio file and appends speech/transcribe text to the draft.
	OnPickSpeech                                            func()
	Voice                                                   voiceinput.Status
	OnStartVoice, OnStopVoice, OnCancelVoice, OnInsertVoice func()
	OnVoiceSettings                                         func()
	// OnCycleSpeech advances speech/configure provider from catalog.
	OnCycleSpeech func()
	// OnPickHeroWorkspace switches the session workspace (HeroShell chip); not @ mention.
	OnPickHeroWorkspace   func()
	OnNoDirectory         func()
	OnPromptModeChange    func(mode session.PromptMode)
	OnUseBusyEnterSetting func()
	// OnSearchFileRefs calls fileReferences/list for @ mention query (optional).
	OnSearchFileRefs   func(query string) []string
	OnSearchReferences func(query string, quoted bool) []session.ReferenceCandidate

	OnCancelEdit func()
	Editing      bool // draft is a message edit (editMessageSeq)
	// Hero paints the Electron empty-session centered composer (larger card,
	// build-oriented placeholder, workspace/mode chips above the card).
	Hero bool
}

type suggestItem struct {
	action func()
	drill  string
	label  string
	sub    string
	insert string
	kind   SuggestKind
}

// View paints the prompt area with send and stop actions.
func View(c *ui.Context, m Model) {
	viewWithColors(c, m, theme.Resolve(m.ThemeID).Active(m.SystemDark))
}

// viewWithColors keeps semantic surface roles together for the active palette.
func viewWithColors(c *ui.Context, m Model, colors theme.Colors) {
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	border := chrome.Color(colors.Border)
	accent := chrome.Color(colors.Accent)
	panel := chrome.Color(colors.BG)
	beam := readBeamStyle(m.ComposerBeamStyle)
	showBeam := m.ComposerBeam && composerRunning(m) && !c.Preferences().ReduceMotion

	if m.State != nil && m.State.GroupReply != nil {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "回复线程 / Reply to thread: "+m.State.GroupReply.ThreadID).TextColor(muted)
			if ui.Button(c, "取消回复 / Cancel reply").Disabled(m.State.Sending).Clicked() {
				m.State.CancelGroupReply()
			}
		})
	}
	draft := ""
	if m.Draft != nil {
		draft = *m.Draft
	}
	var suggestClient *DraftClient
	if m.UseTextInputClient {
		suggestClient = m.TextInput
	}
	kind, query, tok, quoted, suggestOpen := activeSuggestion(m, suggestClient)
	items := buildSuggestItemsWithQuote(m, kind, query, quoted)
	m.stopSuggestionsOpen = suggestOpen && len(items) > 0
	skillsQuery := ""
	if m.SkillsFilter != nil {
		skillsQuery = *m.SkillsFilter
	}
	skillItems := FilterSkills(m.Skills, skillsQuery, len(m.Skills))
	mentions := ParseDraftMentions(draft)
	selectStart, selectEnd := -1, -1
	selectIsUTF16 := m.UseTextInputClient && m.TextInput != nil

	// InputBar uses its semantic translucent surface over the shell canvas.
	_ = panel
	// InputBar.root: pad 0 side-clearance(16) 4; hero drops bottom pad.
	// HeroShell.root already pads 24 — composer keeps the shared 16 clearance.
	padTop, padRight, padBot, padLeft := float32(0), float32(16), float32(4), float32(16)
	if m.Hero {
		// Hero retains InputBar side clearance inside the HeroShell stack.
		padBot = 0
	}
	composerRegion := ui.Column(c.Key("composer-region")).Padding(padTop, padRight, padBot, padLeft)
	regionClient := m.TextInput
	if regionClient == nil && m.State != nil {
		regionClient = m.State.Client
	}
	BindStopRegion(composerRegion, regionClient, m, false)
	if suggestOpen && composerRegion.PressedOutside() {
		dismissSuggestion(m, suggestClient)
		suggestOpen = false
	}
	composerRegion.Children(func() {
		if m.State != nil && m.State.FailedSendCount() > 0 {
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, "未发送草稿 / Unsent drafts").FontSize(11).TextColor(muted)
				if !m.State.FailedSendsRestored() && ui.Button(c, "恢复 / Restore").FontSize(11).Disabled(m.State.Sending || m.State.Preparing > 0 || m.Editing).Clicked() {
					m.State.RestoreFailedSends()
				}
			})
		}
		markdown.LinkActions(c, markdown.Links(draft), markdown.Style{FontSize: 12, Accent: accent}, m.OnOpenURL)
		if kind == SuggestSlash && !m.Editing {
			if m.Snapshot.SlashCommandsLoading {
				ui.Text(c, i18n.T(i18n.KeyCommandCatalogLoading)).FontSize(12).TextColor(muted).Padding(4, 8)
			} else if m.Snapshot.SlashCommandsErr != "" {
				ui.Row(c).FillWidth().Gap(8).Padding(4, 8).Children(func() {
					ui.Text(c, m.Snapshot.SlashCommandsErr).FontSize(12).TextColor(muted).Grow(1)
					if m.OnRetrySlashCommands != nil && ui.Button(c, i18n.T(i18n.KeyCommandCatalogRetry)).FontSize(11).Clicked() {
						m.OnRetrySlashCommands()
					}
				})
			}
		}
		if m.State != nil {
			m.State.ModelCommand.Sync(m.Snapshot.SelectedID)
			m.State.PermissionCommand.Sync(m.Snapshot.SelectedID)
			if m.State.PermissionCommand.Visible(m.Snapshot.SelectedID) {
				paintPermissionCommand(c, m, panel, border, fg, muted)
			}
			if m.State.ModelCommand.Visible(m.Snapshot.SelectedID) {
				paintModelCommand(c, m, panel, border, fg, muted)
			}
		}
		if m.SkillsOpen {
			paintSkillsPopup(c, m, skillItems, panel, border, fg, muted)
		}
		paintLeadingClaimHint(c, m, muted)
		if suggestOpen && (len(items) > 0 || len(referenceBreadcrumbs(m)) > 0) {
			paintSuggestPopup(c, m, items, kind, tok, panel, border, fg, muted)
		}
		if m.Hero {
			// ConversationRoot.heroWorkspaceRow is inset from the actual card,
			// with 4px above its 28px controls and an 8px gap below.
			wsLabel := strings.TrimSpace(m.WorkspaceRoot)
			folderIcon := "📂"
			if wsLabel == "" {
				wsLabel = i18n.T(i18n.KeyChatHeroChooseWorkspace)
				folderIcon = "📁"
			} else if i := strings.LastIndexAny(wsLabel, `/\`); i >= 0 && i+1 < len(wsLabel) {
				wsLabel = wsLabel[i+1:]
			}
			if m.Snapshot.ScratchCwd != "" && filepathCleanForLabel(m.WorkspaceRoot) == filepathCleanForLabel(m.Snapshot.ScratchCwd) {
				wsLabel = i18n.T(i18n.KeyNoDirectory)
				folderIcon = ""
			}
			ui.Row(c.Key("composer-hero-workspace")).Label("Composer workspace row").FillWidth().MinWidth(0).Gap(2).AlignItems(ui.Center).
				Margin(4, 0, 8, 0).Padding(0, 16, 0, 20).Children(func() {
				wsBtn := ui.Button(c, "").Label("Composer workspace").Height(28).Radius(8).Padding(0, 8, 0, 8).
					Background(ui.Transparent).BorderWidth(0)
				if (m.OnPickHeroWorkspace == nil && m.OnPickDirectory == nil) || (m.State != nil && (m.State.Sending || m.State.Preparing > 0)) {
					wsBtn = wsBtn.Disabled(true)
				}
				wsBtn.Children(func() {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Text(c, folderIcon).FontSize(13)
						ui.Text(c, wsLabel).FontSize(13).FontWeight(500).TextColor(fg)
						ui.Text(c, "▾").FontSize(11).TextColor(muted)
					})
				})
				if wsBtn.Clicked() {
					if m.OnPickHeroWorkspace != nil {
						m.OnPickHeroWorkspace()
					} else if m.OnPickDirectory != nil {
						m.OnPickDirectory()
					}
				}
				if m.OnNoDirectory != nil {
					if ui.Button(c, i18n.T(i18n.KeyNoDirectory)).Height(28).FontSize(13).Disabled(m.State != nil && (m.State.Sending || m.State.Preparing > 0)).Clicked() {
						m.OnNoDirectory()
					}
				}
				if m.AgentPresetSelection.Busy {
					ui.Text(c, "Applying preset…").FontSize(12)
				}
				if m.AgentPresetSelection.Error != "" {
					ui.Text(c, m.AgentPresetSelection.Error).FontSize(12)
				}
				ui.Box(c).Grow(1).MinWidth(0)
				// Hero options follow margin-left:auto after the workspace control.
				presetLabel := i18n.T(i18n.KeyComposerPresetDefault)
				if m.AgentPresetPick != nil && strings.TrimSpace(*m.AgentPresetPick) != "" {
					presetLabel = strings.TrimSpace(*m.AgentPresetPick)
				} else if len(m.Snapshot.AgentPresetLabels) > 0 {
					presetLabel = m.Snapshot.AgentPresetLabels[0]
				}
				if len(m.AgentPresetOptions) > 0 && m.AgentPresetPick != nil && m.AgentPresetSelection.CanSelect && !m.AgentPresetSelection.Busy {
					presetSelect(c, presetLabel, agentPresetChoices(m.AgentPresetOptions), m.OnAgentPresetChange).
						Height(28).FontSize(13).FontWeight(500).
						Background(ui.Transparent).BorderWidth(0).TextColor(fg)
				} else {
					ui.Row(c).Label("Composer hero preset").Gap(4).AlignItems(ui.Center).Height(28).Padding(0, 8, 0, 8).Children(func() {
						ui.Text(c, "⌂").FontSize(13)
						ui.Text(c, presetLabel).FontSize(13).FontWeight(500).TextColor(fg)
						ui.Text(c, "▾").FontSize(11).TextColor(muted)
					})
				}
			})
		}
		// InputBar.card: 8px top pad, 12px gap, 28px radius. The hairline
		// is painted as elevation, so it does not consume draft layout space.
		card := ui.Column(c.Key("composer-card")).Label("Composer card").FillWidth().Background(chrome.BackdropSurface(chrome.Color(colors.InputMajor))).
			Radius(28).Padding(8, 0, 0, 0).Gap(12).
			DrawOver(func(p *ui.Painter, r ui.Rect) {
				p.Stroke(r, border, 28, 0.5)
				if showBeam {
					paintComposerBeam(p, r, beam)
				}
			})
		card.Children(func() {
			paintResizeHandles(c, m, card, muted)
			if m.State != nil {
				paintDraftAttachments(c, m, fg, muted)
			}
			if m.Editing {
				ui.Row(c).FillWidth().Padding(0, 16).Gap(8).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyChatEditHint)).FontSize(11).TextColor(muted)
					if ui.Button(c, i18n.T(i18n.KeyQuitConfirmCancel)).Disabled(m.State != nil && m.State.Sending).Clicked() && m.OnCancelEdit != nil {
						m.OnCancelEdit()
					}
				})
			}
			if len(mentions) > 0 && m.Draft != nil {
				paintMentionChips(c, m, mentions, muted, accent, selectIsUTF16, &selectStart, &selectEnd)
			}
			if m.ChipEdit != nil && m.ChipEdit.Open && m.Draft != nil {
				paintChipEditPopup(c, m, panel, border, muted)
			}
			if m.Draft != nil {
				var area ui.Element
				placeholder := composerPlaceholder(m)
				if m.UseTextInputClient && m.TextInput != nil {
					area = paintDraftTextInput(c, m, fg, chrome.Color(colors.Caption), selectStart, selectEnd)
					// A clicked mention explicitly requests selection and focus.
					// Ordinary rebuilds must preserve sidebar/toolbar/input focus.
					if selectStart >= 0 {
						area.Focus()
					}
				} else {
					// InputBar: docked min-height 36 (~1 line); hero .input 52 (~2×24).
					minLines, maxLines := 1, 8
					if m.Hero {
						minLines, maxLines = 2, 10
					}
					ta := ui.TextArea(c.Key("composer-draft-input"), m.Draft).Label("Composer draft").FillWidth().Lines(minLines, maxLines).Placeholder(placeholder).
						Font(theme.CurrentPreferences().Composer.Family).FontSize(theme.CurrentPreferences().Composer.Size).FixedLineHeight(composerLineHeight(theme.CurrentPreferences().Composer.Size)).Padding(4, 8, 0, 14).
						MinHeight(36).Background(ui.Transparent).BorderWidth(0).Radius(0).TextColor(fg)
					if m.Hero {
						ta = ta.MinHeight(52)
					}
					if m.ResizeEnabled && m.Resize != nil && m.Resize.Height > 0 {
						ta.Height(resizeInputHeight(c, m, 36))
					}
					if selectStart >= 0 {
						ta.SetTextSelection(selectStart, selectEnd).Focus()
					}
					area = ta
					paintNextPrompt(c, m, muted)
				}
				// Toolkit AutoFocus is a first-mount request, never a per-frame
				// keep-focused loop. Do not override an already focused control.
				if !c.Root().FocusWithin() {
					area.AutoFocus()
				}
				if suggestOpen && len(items) > 0 {
					wireSuggestKeys(area, m, items, kind, tok)
				}
				if m.State != nil && m.State.EditFocus {
					area.Focus()
					m.State.EditFocus = false
				}
				if m.NextPrompt.Text != "" && !(suggestOpen && len(items) > 0) && !(m.UseTextInputClient && m.TextInput != nil) {
					area.OnShortcut(0, ui.KeyTab, func() { acceptNextPrompt(m) })
				}
				wireFileDrop(area, m)
				wireWorkspaceReferenceDrop(area, m)
				if _, over := ui.DragOver[filelink.Reference](area); over {
					ui.Text(c, "添加文件引用 / Add file reference").FontSize(11).TextColor(muted)
				}
				if m.UseTextInputClient && m.TextInput != nil {
					m.TextInput.Sync()
				}
				if area.FileDragOver() && strings.TrimSpace(m.WorkspaceRoot) != "" {
					ui.Text(c, i18n.T(i18n.KeyComposerDropHint)).FontSize(11).TextColor(muted)
				}
			}
			// InputBar.row: pad 2 8 6; tools/trailing gap 12; +.add 28 selector fill;
			// .select 28/13/500 transparent; .primary send 34 info-fill.
			btnH := float32(28)
			sendH := float32(34)
			selectorFill := chrome.Color(colors.SelectorFill)
			infoFill := chrome.Color(colors.ButtonInfoFill)
			onAccent := chrome.Color(colors.ButtonFG)
			ui.Row(c).FillWidth().Wrap().Gap(12).AlignItems(ui.Center).Padding(2, 8, 6, 8).Children(func() {
				// Keep tools and trailing controls as separate wrap groups.
				ui.Row(c).MaxWidthPercent(100).MinWidth(0).Wrap().Gap(12).AlignItems(ui.Center).Children(func() {
					plus := ui.Button(c, "+").Label("命令列表 / Commands").Tooltip("命令列表 / Commands").Width(btnH).Height(btnH).Radius(btnH / 2).BorderWidth(0).Background(selectorFill).TextColor(fg).FontSize(18)
					if m.Editing || suggestionState(m, m.TextInput) == nil || m.State != nil && (m.State.Sending || m.State.Preparing > 0) {
						plus = plus.Disabled(true)
					}
					if plus.Clicked() {
						toggleCommandLauncher(m)
					}
					paintVoiceControls(c, m, fg, muted, selectorFill, btnH)
					paintPlanModeControl(c, m, muted, btnH)
					// InputBar .select: permission scope (not PromptMode queue/steer).
					if len(m.PermissionOptions) > 0 && m.PermissionPick != nil {
						presetSelect(c, *m.PermissionPick, permissionChoices(m.PermissionOptions), m.OnPermissionChange).
							Height(btnH).FontSize(13).FontWeight(500).
							Background(ui.Transparent).BorderWidth(0).TextColor(muted)
					} else {
						scope := i18n.T(i18n.KeyComposerPermissionDefault)
						if m.PermissionPick != nil && strings.TrimSpace(*m.PermissionPick) != "" {
							scope = strings.TrimSpace(*m.PermissionPick)
						}
						ui.Text(c, scope+" ▾").FontSize(13).FontWeight(500).TextColor(muted)
					}
				})
				ui.Row(c).Label("Composer trailing controls").MaxWidthPercent(100).MinWidth(0).Gap(12).AlignItems(ui.Center).Margin(0, 0, 0, ui.Auto).Children(func() {
					if m.State == nil || !m.State.ModelCommand.Visible(m.Snapshot.SelectedID) {
						paintContextMeter(c, m.Snapshot, muted, btnH)
						paintModelPicker(c, m, muted, btnH)
					}
					running := m.Snapshot.PromptBusy
					for _, item := range m.Snapshot.Items {
						if item.SessionID == m.Snapshot.SelectedID && item.Running {
							running = true
						}
					}
					child := false
					for _, item := range m.Snapshot.Items {
						if item.SessionID == m.Snapshot.SelectedID && item.Origin == "subagent" {
							child = true
						}
					}
					primaryStops := running && !child && (composerDraftEmpty(m) || m.StopBlocked)
					if primaryStops || running && child && m.OnStop != nil {
						stop := ui.Button(c, "■").Label(i18n.T(i18n.KeyComposerStop)).Tooltip(i18n.T(i18n.KeyComposerStop)).
							Width(sendH).Height(sendH).Padding(0).Radius(sendH / 2).
							Background(infoFill).BorderWidth(0).Top(-2).
							TextColor(onAccent).FontSize(12).Disabled(m.OnStop == nil)
						if stop.Clicked() && m.OnStop != nil {
							m.OnStop()
						}
					}
					if !primaryStops {
						send := ui.Button(c, "↑").Label(i18n.T(i18n.KeyComposerSend)).Tooltip(i18n.T(i18n.KeyComposerSend)).
							Width(sendH).Height(sendH).Padding(0).Radius(sendH / 2).
							Background(infoFill).BorderWidth(0).Top(-2).
							TextColor(onAccent).FontSize(16)
						if composerSendDisabled(m) {
							send = send.Disabled(true)
						}
						if send.Clicked() && handleModelCommand(m) {
							// Local entry consumes no prompt or attachment.
						} else if send.Clicked() && m.OnSubmitGesture != nil {
							m.OnSubmitGesture(false)
						} else if send.Clicked() && m.OnSubmit != nil {
							m.OnSubmit()
						} else if send.Clicked() && m.OnSend != nil && m.Draft != nil {
							text := *m.Draft
							if handleSlashSend(m, text) {
								*m.Draft = ""
							} else {
								*m.Draft = ""
								m.OnSend(text)
							}
							if m.TextInput != nil {
								m.TextInput.ClearHistory()
							}
						}
					}
				})
			})
		})
	})
}

func buildSuggestItems(m Model, kind SuggestKind, query string) []suggestItem {
	return buildSuggestItemsWithQuote(m, kind, query, false)
}

func buildSuggestItemsWithQuote(m Model, kind SuggestKind, query string, quoted bool) []suggestItem {
	switch kind {
	case SuggestSlash:
		return slashMenuItems(m, query)
	case SuggestMention:
		var candidates []session.ReferenceCandidate
		var roomCandidates []suggestItem
		if !quoted && m.GroupMembers != nil {
			for _, item := range session.GroupMentions(m.GroupMembers, query) {
				roomCandidates = append(roomCandidates, suggestItem{label: item.Label, sub: "@" + item.Value, insert: "@" + item.Value + " ", kind: SuggestMention})
			}
		}
		if m.OnSearchReferences != nil {
			candidates = m.OnSearchReferences(query, quoted)
		}
		if len(candidates) == 0 {
			var paths []string
			if m.OnSearchFileRefs != nil {
				paths = m.OnSearchFileRefs(query)
			}
			if len(paths) == 0 {
				paths = FilterPaths(m.FilePaths, query, 8)
			}
			for _, path := range paths {
				kind := "file"
				if strings.HasSuffix(path, "/") {
					kind = "directory"
				}
				candidate := session.ReferenceCandidate{Kind: kind, Path: path, Label: path}
				if strings.HasPrefix(path, "@[") {
					candidate.Kind, candidate.Mention = "session", path
				}
				candidates = append(candidates, candidate)
			}
		}
		out := append([]suggestItem(nil), roomCandidates...)
		for _, candidate := range candidates {
			label, sub, insert := candidate.Label, candidate.Kind, candidate.Mention
			drill := ""
			if candidate.Kind == "session" {
				if quoted || insert == "" {
					continue
				}
				if candidate.Cwd != "" && !candidate.SameWorkspace {
					sub += " · " + candidate.Cwd
				}
				// An opaque identity distinguishes equally titled sessions even within one workspace.
				if candidate.SessionID != "" {
					sub += " · " + candidate.SessionID
				}
				insert += " "
			} else {
				var ok bool
				insert, ok = FormatFileMention(candidate.Path, candidate.Kind == "directory", quoted)
				if !ok {
					continue
				}
				if label == "" {
					label = candidate.Path
				}
				if candidate.Kind == "directory" {
					sub = "directory · Tab to browse"
					drill = insert
					if strings.HasPrefix(insert, `@"`) {
						insert += `"`
					}
					insert += " "
				} else {
					insert += " "
				}
			}
			out = append(out, suggestItem{label: label, sub: sub, insert: insert, drill: drill, kind: SuggestMention})
		}
		return out
	default:
		return nil
	}
}

func paintSkillsPopup(c *ui.Context, m Model, items []session.SkillOption, panel, border, fg, muted ui.Color) {
	ui.Column(c).Gap(1).Padding(0, 0, 4, 0).Children(func() {
		ui.Column(c).Background(panel).Border(1, border).Radius(6).Padding(2, 2, 2, 2).MaxHeight(280).Children(func() {
			if m.SkillsFilter != nil {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(2, 4, 2, 4).Children(func() {
					ui.TextInput(c, m.SkillsFilter).Grow(1).Placeholder(i18n.T(i18n.KeyComposerSkillsFilter))
					q := strings.TrimSpace(*m.SkillsFilter)
					if m.OnSearchSkillHub != nil && q != "" {
						if ui.Button(c, i18n.T(i18n.KeyComposerSkillsHubSearch)).FontSize(11).Clicked() {
							m.OnSearchSkillHub(q)
						}
					}
				})
			}
			if m.SkillCreateName != nil && m.OnCreateSkill != nil {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(2, 4, 2, 4).Children(func() {
					ui.TextInput(c, m.SkillCreateName).Grow(1).
						Placeholder(i18n.T(i18n.KeyComposerSkillsCreateName))
					if ui.Button(c, i18n.T(i18n.KeyComposerSkillsCreate)).FontSize(11).Clicked() {
						name := strings.TrimSpace(*m.SkillCreateName)
						if name != "" {
							m.OnCreateSkill(name)
							*m.SkillCreateName = ""
						}
					}
				})
			}
			if len(m.SkillHubHits) > 0 {
				for _, hit := range m.SkillHubHits {
					h := hit
					ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(4, 6, 4, 6).Children(func() {
						ui.Column(c).Grow(1).Children(func() {
							ui.Text(c, h.SkillName).FontSize(12).Bold().TextColor(fg)
							sub := h.Description
							if sub == "" {
								sub = h.Repo
							}
							if sub != "" {
								ui.Text(c, sub).FontSize(11).TextColor(muted)
							}
						})
						if h.Status != "" {
							ui.Text(c, h.Status).FontSize(11).TextColor(muted)
						}
						if !h.Installed && h.Status != "conflict" && m.OnInstallSkillHub != nil {
							if ui.Button(c, i18n.T(i18n.KeyComposerSkillsHubInstall)).FontSize(11).Clicked() {
								m.OnInstallSkillHub(h)
							}
						}
					})
				}
			}
			if len(items) == 0 {
				msg := i18n.T(i18n.KeyComposerSkillsEmpty)
				filterEmpty := m.SkillsFilter == nil || strings.TrimSpace(*m.SkillsFilter) == ""
				if m.Snapshot.SkillsErr != "" && filterEmpty && len(m.Skills) == 0 {
					msg = m.Snapshot.SkillsErr
				}
				if len(m.SkillHubHits) == 0 {
					ui.Text(c, msg).FontSize(11).TextColor(muted).Padding(6, 8, 6, 8)
				}
				return
			}
			list := m.SkillsList
			if list == nil {
				return
			}
			sel := 0
			if list.Selected != nil {
				sel = *list.Selected
			}
			if sel < 0 || sel >= len(items) {
				sel = 0
				if list.Selected != nil {
					*list.Selected = 0
				}
			}
			l := ui.List(c, list, len(items)).Rows(func(row ui.ListRow) {
				sk := items[row.Index]
				bg := panel
				if row.Index == sel {
					bg = muted
				}
				sub := sk.Description
				if sk.UserOnly {
					prefix := i18n.T(i18n.KeyComposerSkillUserOnly)
					if sub != "" {
						sub = prefix + " · " + sub
					} else {
						sub = prefix
					}
				}
				ui.Column(row.Context).Padding(6, 8, 6, 8).Background(bg).Children(func() {
					ui.Row(row.Context).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(row.Context, "/"+sk.Name).FontSize(12).Bold().TextColor(fg).Grow(1)
						if m.OnToggleSkillModel != nil {
							label := i18n.T(i18n.KeyComposerSkillsToggleModel)
							if ui.Button(row.Context, label).FontSize(10).
								BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
								m.OnToggleSkillModel(sk.Name, sk.UserOnly) // UserOnly ⇒ currently not model-invocable
							}
						}
						if sk.UserOnly && m.OnDeleteSkill != nil {
							if ui.Button(row.Context, i18n.T(i18n.KeyComposerSkillsDelete)).FontSize(10).
								BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
								m.OnDeleteSkill(sk.Name)
							}
						}
					})
					if sub != "" {
						ui.Text(row.Context, sub).FontSize(11).TextColor(muted)
					}
				})
			})
			if l.Submitted() || l.Changed() {
				if list.Selected != nil && *list.Selected >= 0 && *list.Selected < len(items) {
					applySkillPick(m, items[*list.Selected])
				}
			}
		})
	})
}

type chipDrag struct {
	Index int
}

func paintMentionChips(c *ui.Context, m Model, mentions []DraftMention, muted, accent ui.Color, selectUTF16 bool, selectStart, selectEnd *int) {
	ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
		for i, ment := range mentions {
			i, ment := i, ment
			label := ChipLabel(ment)
			color := accent
			if ment.Kind == DraftMentionSkill {
				color = muted
			}
			chip := ui.Button(c, label).TextColor(color).Drag(chipDrag{Index: i})
			if _, over := ui.DragOver[chipDrag](chip); over {
				chip = chip.Background(muted)
			}
			if from, ok := ui.Drop[chipDrag](chip); ok && m.Draft != nil {
				*m.Draft = ReorderDraftMentions(*m.Draft, from.Index, i)
			}
			if chip.Dragging() {
				chip = chip.Opacity(0.45)
			}
			if m.Draft == nil {
				continue
			}
			if chip.RightClicked() {
				openChipEdit(m, ment)
				var start, end int
				var ok bool
				if selectUTF16 {
					start, end, ok = MentionUTF16Range(*m.Draft, ment)
				} else {
					start, end, ok = MentionRuneRange(*m.Draft, ment)
				}
				if ok && selectStart != nil && selectEnd != nil {
					*selectStart, *selectEnd = start, end
				}
			} else if chip.Clicked() && chip.Clicks() == 1 {
				activateDraftReference(m, ment)
			}
			if ui.Button(c, "×").Label("Remove reference " + ment.Text).
				Tooltip("移除引用 / Remove reference").TextColor(muted).
				Disabled(m.State != nil && m.State.Sending).Clicked() {
				removeDraftReference(m, ment)
			}
		}
	})
}

func paintChipEditPopup(c *ui.Context, m Model, panel, border, muted ui.Color) {
	ed := m.ChipEdit
	if ed == nil {
		return
	}
	ui.Column(c).Background(panel).Border(1, border).Radius(6).Padding(6, 8, 6, 8).Gap(6).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyComposerChipEdit)).FontSize(11).TextColor(muted)
		ui.TextInput(c, &ed.Draft).Fill().Placeholder(i18n.T(i18n.KeyComposerChipEditPlaceholder))
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, i18n.T(i18n.KeyComposerChipEditApply)).Clicked() {
				target := DraftMention{Text: ed.Orig, Start: ed.Start, End: ed.Start + len(ed.Orig)}
				*m.Draft = ReplaceDraftMention(*m.Draft, target, ed.Draft)
				closeChipEdit(m)
			}
			if ui.Button(c, i18n.T(i18n.KeyComposerChipEditCancel)).Clicked() {
				closeChipEdit(m)
			}
		})
	})
}

func openChipEdit(m Model, ment DraftMention) {
	if m.ChipEdit == nil {
		return
	}
	m.ChipEdit.Open = true
	m.ChipEdit.Start = ment.Start
	m.ChipEdit.Orig = ment.Text
	m.ChipEdit.Draft = ment.Text
}

func closeChipEdit(m Model) {
	if m.ChipEdit == nil {
		return
	}
	m.ChipEdit.Open = false
	m.ChipEdit.Start = 0
	m.ChipEdit.Orig = ""
	m.ChipEdit.Draft = ""
}

func applySkillPick(m Model, sk session.SkillOption) {
	if m.Draft == nil {
		return
	}
	AppendSkillInvocation(m.Draft, sk.Name)
	if m.SkillsFilter != nil {
		*m.SkillsFilter = ""
	}
	if m.OnToggleSkills != nil {
		m.OnToggleSkills(false)
	}
}

func paintSuggestPopup(c *ui.Context, m Model, items []suggestItem, kind SuggestKind, tok TokenRange, panel, border, fg, muted ui.Color) {
	list := m.SuggestList
	if list == nil {
		return
	}
	sel := 0
	if list.Selected != nil {
		sel = *list.Selected
	}
	if sel < 0 || sel >= len(items) {
		sel = 0
		if list.Selected != nil {
			*list.Selected = 0
		}
	}
	ui.Column(c).Gap(1).Padding(0, 0, 4, 0).Children(func() {
		ui.Column(c).Background(panel).Border(1, border).Radius(6).Padding(2, 2, 2, 2).MaxHeight(180).Children(func() {
			paintReferenceBreadcrumbs(c, m, tok)
			drilled := false
			clicked := -1
			l := ui.List(c, list, len(items)).Rows(func(row ui.ListRow) {
				it := items[row.Index]
				bg := panel
				if row.Index == sel {
					bg = muted
				}
				itemRow := ui.Column(row.Context).Padding(6, 8, 6, 8).Background(bg)
				itemRow.Children(func() {
					ui.Row(row.Context).Children(func() {
						ui.Text(row.Context, it.label).Grow(1).FontSize(12).Bold().TextColor(fg)
						if it.drill != "" && ui.Button(row.Context, "›").Tooltip("浏览目录 / Browse directory").Clicked() {
							drilled = true
							applySuggestDrill(m, it, tok)
						}
					})
					if it.sub != "" {
						ui.Text(row.Context, it.sub).FontSize(11).TextColor(muted)
					}
				})
				if itemRow.Clicked() {
					clicked = row.Index
				}
			})
			if !drilled && clicked >= 0 {
				applySuggest(m, items[clicked], tok)
			} else if !drilled && (l.Submitted() || l.Changed()) {
				if list.Selected != nil && *list.Selected >= 0 && *list.Selected < len(items) {
					applySuggest(m, items[*list.Selected], tok)
				}
			}
		})
	})
	_ = kind
}

func wireFileDrop(area ui.Element, m Model) {
	if m.Draft == nil {
		return
	}
	if dropped := area.DroppedFiles(); len(dropped) > 0 {
		intakeTransferredFiles(m, dropped)
	}
}

func wireSuggestKeys(area ui.Element, m Model, items []suggestItem, kind SuggestKind, tok TokenRange) {
	// The native input handler owns these keys, including the IME guard.
	if m.UseTextInputClient && m.TextInput != nil {
		return
	}
	for _, key := range []ui.Key{ui.KeyUp, ui.KeyDown, ui.KeyTab, ui.KeyEnter, ui.KeyEscape} {
		key := key
		area.OnShortcut(0, key, func() { handleSuggestKey(nil, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: key}) })
	}
	area.OnShortcut(ui.Shift, ui.KeyTab, func() { handleSuggestKey(nil, m, ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyTab, Mods: ui.Shift}) })
}

func applySuggest(m Model, it suggestItem, tok TokenRange) {
	if it.action != nil {
		dismissSuggestion(m, m.TextInput)
		it.action()
		return
	}
	if m.Draft == nil {
		return
	}
	switch it.kind {
	case SuggestSlash:
		*m.Draft = ReplaceToken(*m.Draft, tok, it.insert)
	case SuggestMention:
		*m.Draft = ReplaceToken(*m.Draft, tok, it.insert)
	}
	if m.TextInput != nil {
		m.TextInput.Sync()
		caret := ui.UTF16Len((*m.Draft)[:tok.Start+len(it.insert)])
		m.TextInput.SetSelectionUTF16(caret, caret)
	}
	activeSuggestion(m, m.TextInput)
	dismissSuggestion(m, m.TextInput)
}

func handleSlashSend(m Model, text string) bool {
	if m.Editing {
		return false
	}
	text = strings.TrimSpace(text)
	if text == "" || !strings.HasPrefix(text, "/") {
		return false
	}
	if strings.Contains(text, "\n") {
		return false
	}
	line := NormalizeCommandLine(text)
	if isSkillOnlyInvocation(m, line) {
		return false
	}
	if act := LocalSlashAction(line, StubSlashCommands()); act != "" {
		switch act {
		case "clear":
			if m.OnClearDraft != nil {
				m.OnClearDraft()
			}
		case "new":
			if m.OnNewSession != nil {
				m.OnNewSession()
			}
		}
		return true
	}
	for _, c := range m.SlashCatalog {
		if strings.EqualFold(NormalizeCommandLine(c.Line), line) && c.LocalAction != "" {
			switch c.LocalAction {
			case "clear":
				if m.OnClearDraft != nil {
					m.OnClearDraft()
				}
			case "new":
				if m.OnNewSession != nil {
					m.OnNewSession()
				}
			}
			return true
		}
	}
	if m.OnExecuteSlash != nil {
		m.OnExecuteSlash(context.Background(), line)
		return true
	}
	return false
}

func draftHint(snap session.ViewState) string {
	if snap.ClientReady {
		return snap.SelectedID
	}
	return i18n.T(i18n.KeyComposerWaiting)
}

// BuildSlashCatalog merges stubs with remote commands/list rows.
func BuildSlashCatalog(stubs []SlashCommand, remoteNames []SlashCommand) []SlashCommand {
	if len(stubs) == 0 {
		stubs = StubSlashCommands()
	}
	return MergeSlashCommands(stubs, remoteNames)
}

// FromAPIList converts dsh commands/list into SlashCommand rows.
func FromAPIList(list []api.ListResultItem2) []SlashCommand {
	out := make([]SlashCommand, 0, len(list))
	for _, it := range list {
		name := it.Name
		if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}
		out = append(out, SlashCommand{Line: name, Description: it.Description})
	}
	return out
}

func paintDraftAttachments(c *ui.Context, m Model, fg, muted ui.Color) {
	paintAttachmentRail(c, m, fg, muted)
}

func filepathCleanForLabel(p string) string {
	return strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
}
