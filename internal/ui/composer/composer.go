// Package composer is the multiline prompt input for M4–M6.
package composer

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/svgimage"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// Model is composer state for one frame.
type Model struct {
	SVGImages             *svgimage.Owner
	GroupMembers          []session.GroupMember
	NextPrompt            session.NextPromptSuggestion
	OnAcceptNextPrompt    func() string
	ComposerBeam          bool
	ComposerBeamStyle     json.RawMessage
	TypingFx              bool
	TypingFxStyle         json.RawMessage
	ResizeEnabled         bool
	Resize                *ResizeState
	ResizeMaxWidth        float32
	OnResizeEnd           func(axis string, value float32)
	Snapshot              session.ViewState
	ThemeID               string
	SystemDark            bool
	WorkspaceRoot         string
	WorkspacePickerAnchor *ui.Handle
	Inert                 bool // no Session or no owning workspace: the card opens the workspace picker
	OnRequestWorkspace    func()
	Draft                 *string
	State                 *DraftState
	OnSubmit              func()
	OnSubmitGesture       func(accelerated bool)
	OnOpenURL             func(string)
	OnOpenReference       func(path string) // existing session-owned file preview route
	OnAttachFiles         func([]string)
	OnRetryAttachment     func(int)
	OnWorkspaceReference  func(filelink.Reference) error
	OnPasteImage          func(bool)
	ModelPick             *string
	PermissionOptions     []session.PermissionOption
	AgentPresetOptions    []session.AgentPresetOption
	AgentPresetSelection  session.AgentPresetSelectionView
	PermissionPick        *string
	AgentPresetPick       *string

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

	OnSend                  func(text string)
	OnStop                  func()
	StopBlocked             bool // pending approval/question owns response input
	stopSuggestionsOpen     bool
	OnModelChange           func(label string)
	OnModelEffortChange     func(label string)
	OnRetryModels           func()
	OnRetryPermissions      func()
	OnExitPlanMode          func()
	OnPermissionChange      func(value string)
	OnAgentPresetChange     func(id string)
	OnAgentPresetIntroduced func(hostGeneration uint64, sessionID string, sequence uint64)
	OnClearDraft            func()
	OnNewSession            func()
	OnExecuteSlash          func(ctx context.Context, line string) // remote commands/execute
	// OnPickCommand captures the rendered Host epoch and Session before a
	// detached, attachment-free menu execution leaves the editor.
	OnPickCommand                   func(hostGeneration uint64, sessionID, line string)
	OnNativeCommandAction           func(owner NativeCommandOwner, name string) bool
	OnLoadNativeCommandOptions      func(context.Context, NativeCommandOwner, string) ([]NativeCommandOption, error)
	OnSelectNativeCommandModel      func(context.Context, NativeCommandOwner, api.ModelCatalogDefault) error
	OnSelectNativeCommandPermission func(context.Context, NativeCommandOwner, string, bool) error
	OnNativeCommandChanged          func()
	OnToggleSkills                  func(open bool)
	OnPickDirectory                 func()
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
	// OnCardLayout observes the current unshifted card before its enclosing
	// stack receives the first-send displacement. It must not build controls.
	OnCardLayout func(ui.Rect)
}

type suggestItem struct {
	action  func()
	drill   string
	label   string
	sub     string
	insert  string
	kind    SuggestKind
	command bool
}

// stopShortcutHint names the composer's double-Escape stop gesture in the stop
// button's tooltip, where the reference surfaces the binding.
const stopShortcutHint = "Esc Esc"

// View paints the prompt area with send and stop actions.
func View(c *ui.Context, m Model) {
	viewWithColors(c, m, theme.Resolve(m.ThemeID).Active(m.SystemDark))
}

// viewWithColors keeps semantic surface roles together for the active palette.
func viewWithColors(c *ui.Context, m Model, colors theme.Colors) {
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	if !m.SystemDark {
		// The light card's muted gray is close to the fill; secondary keeps
		// the permission, model and hint labels readable.
		muted = chrome.Color(colors.Secondary)
	}
	border := chrome.Color(colors.Border)
	accent := chrome.Color(colors.Accent)
	panel := chrome.Color(colors.BG)
	beam := readBeamStyle(m.ComposerBeamStyle)
	reducedMotion := c.Preferences().ReduceMotion
	showBeam := m.ComposerBeam && composerRunning(m) && !reducedMotion

	if !m.Inert && m.State != nil && m.State.GroupReply != nil {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerReplyThread), m.State.GroupReply.ThreadID)).TextColor(muted)
			if ui.Button(c, i18n.T(i18n.KeyComposerReplyCancel)).Disabled(m.State.Sending).Clicked() {
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
	if m.Inert {
		suggestOpen = false
		m.SkillsOpen = false
	}
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
	requestWorkspaceFocus := false
	// Drag state observed on the draft surface, painted as one overlay later so
	// the capsule keeps its size while a file hovers over it.
	fileDrag, referenceDrag := false, false

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
				ui.Text(c, i18n.T(i18n.KeyComposerUnsentDraft)).FontSize(11).TextColor(muted)
				if !m.State.FailedSendsRestored() && ui.Button(c, i18n.T(i18n.KeyComposerUnsentRestore)).FontSize(11).Disabled(m.State.Sending || m.State.Preparing > 0 || m.Editing).Clicked() {
					m.State.RestoreFailedSends()
				}
			})
		}
		markdown.LinkActions(c, markdown.Links(draft), markdown.Style{FontSize: 12, Accent: accent}, m.OnOpenURL)
		if !m.Inert && kind == SuggestSlash && !m.Editing {
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
			if m.Snapshot.NativeContributions.Error != "" {
				ui.Text(c, m.Snapshot.NativeContributions.Error).FontSize(12).TextColor(muted).Padding(4, 8)
			}
		}
		if m.State != nil {
			m.State.ModelCommand.Sync(m.Snapshot.SelectedID)
			m.State.PermissionCommand.Sync(m.Snapshot.SelectedID)
		}
		if m.SkillsOpen {
			paintSkillsPopup(c, m, skillItems, panel, border, fg, muted)
		}
		paintLeadingClaimHint(c, m, muted)
		if m.Hero {
			// One 28px row above the card carries both entries: the workspace
			// picker on the left and the agent preset on the right.
			// Gap 18 (native: workspace chip ends ~620, 标准模式 at 638.3); it only
				// separates the two chips, so it is the native chip-to-chip inset.
				ui.Row(c.Key("composer-hero-preset")).Label("Composer hero preset row").FillWidth().MinWidth(0).Gap(3.33).AlignItems(ui.Center).
				// Round217 UIA (1440x860): the native workspace chip starts at x=502
				// (33 DIP leading inset) and 标准模式 follows at 655.33, i.e. a 3.33 DIP
				// chip-to-chip gap; the old 20/18 pair is what put us at 489.33/675.33.
				Margin(4, 0, 8, 0).Padding(0, 16, 0, 32.67).Children(func() {
				wsLabel := strings.TrimSpace(m.WorkspaceRoot)
				// Session groups without a directory carry sentinel ids such as
				// "no-workspace"; only an absolute root names a workspace.
				if !filepath.IsAbs(wsLabel) {
					wsLabel = ""
				} else if i := strings.LastIndexAny(wsLabel, `/\`); i >= 0 && i+1 < len(wsLabel) {
					wsLabel = wsLabel[i+1:]
					// The harness parks directory-less sessions in a
					// dsh-home/no-workspace folder; that is not a workspace name.
					if wsLabel == "no-workspace" {
						wsLabel = ""
					}
				}
				if m.Inert || wsLabel == "" {
					wsLabel = i18n.T(i18n.KeyChatHeroChooseWorkspace)
				}
				// Native rule (round205/round211): the workspace chip is a fixed 150
				// DIP wide with 8 DIP side padding, a 16 DIP folder icon, a 4 DIP
				// gap, and a trailing chevron; longer names truncate, shorter ones
				// keep the width so the preset chip lands at the native x.
				workspace := chip(c, colors, false, fg).Label("Composer workspace").Padding(0, 8, 0, 8).Width(150)
				if m.WorkspacePickerAnchor != nil {
					workspace.Bind(m.WorkspacePickerAnchor)
				}
				if m.OnPickHeroWorkspace == nil || (m.State != nil && (m.State.Sending || m.State.Preparing > 0)) {
					workspace = workspace.Disabled(true)
				}
				workspace.Children(func() {
					ui.Icon(c, folderIcon).Size(16, 16).TextColor(muted).Shrink(0)
					// Long names truncate inside the fixed 150-DIP chip instead of
					// pushing the chevron or the preset chip.
					ui.Text(c, wsLabel).FontSize(13).FontWeight(500).TextColor(fg).SingleLine().MaxLines(1).Shrink(1).Grow(1)
					chevron(c, false)
				})
				if workspace.Clicked() && m.OnPickHeroWorkspace != nil {
					m.OnPickHeroWorkspace()
				}
				if m.AgentPresetSelection.Busy {
					ui.Text(c, i18n.T(i18n.KeyComposerPresetApplying)).FontSize(12)
				}
				if m.AgentPresetSelection.Error != "" {
					ui.Text(c, m.AgentPresetSelection.Error).FontSize(12)
				}
				// The installed hero keeps the preset chip in the left flow right
				// after the workspace chip (probe at 1440x860: the workspace chip ends
				// near x=620 and 标准模式 sits at x=638.3); a grow spacer here
				// right-aligned it at x≈1127.
				presetLabel := i18n.T(i18n.KeyComposerPresetDefault)
				if m.AgentPresetPick != nil && strings.TrimSpace(*m.AgentPresetPick) != "" {
					presetLabel = presetDisplayName(*m.AgentPresetPick)
				} else if len(m.Snapshot.AgentPresetLabels) > 0 {
					presetLabel = presetDisplayName(m.Snapshot.AgentPresetLabels[0])
				}
				if len(m.AgentPresetOptions) > 0 && m.AgentPresetPick != nil && m.AgentPresetSelection.CanSelect && !m.AgentPresetSelection.Busy {
					presetSelect(c, presetLabel, agentPresetChoices(m.AgentPresetOptions), m.OnAgentPresetChange, colors, presetOptions{
						Leading:  func() { paintPresetIntroduction(c, m, presetLabel, fg) },
						Text:     fg,
						FontSize: 13,
						Weight:   500,
					})
				} else {
					ui.Row(c).Label("Composer hero preset").Gap(4).AlignItems(ui.Center).Height(28).Padding(0, 8, 0, 8).Children(func() {
						ui.Icon(c, presetAgentIcon).Size(16, 16).TextColor(fg)
						ui.Text(c, presetLabel).FontSize(13).FontWeight(500).TextColor(fg)
						ui.Text(c, "▾").FontSize(11).TextColor(muted)
					})
				}
			})
		}
		// InputBar.card: 8px top pad, 12px gap, --dsw-radius-panel (28px). The
		// hairline is painted as elevation, so it does not consume draft layout
		// space, and the inset white bloom below is the reference's
		// `inset 0 0 12px 1px rgba(255,255,255,.25)` (visible in dark themes).
		cardRadius := float32(28)
		card := ui.Column(c.Key("composer-card")).Label("Composer card").FillWidth().
			Radius(cardRadius)
		if m.Hero {
			// HeroShell.card: `padding: 8px 0 0` around the 52-DIP writing well, a
			// 12-DIP gap and the 34-DIP control row, which owns the card's 8-DIP
			// side inset. The rendered dialog is 774.39x114 at (469.3,460) with
			// 添加文件 at x=477.3 and the send disc ending 8 before the right edge.
			card = card.Padding(8, 0, 8, 0).Gap(12)
		} else {
			card = card.Padding(8, 0, 0, 0).Gap(12)
		}
		if m.OnCardLayout != nil {
			card.ObserveLayout(m.OnCardLayout)
		}
		beamState := ui.Local(card, "composer-beam-timeline", func() beamTimeline { return beamTimeline{} })
		card.Draw(func(p *ui.Painter, r ui.Rect) {
			// The card's material paints under the beam and the inner bloom, so
			// the wallpaper shows through the glass and the beam still glows on
			// top of it.
			chrome.DrawGlass(p, r, chrome.Color(colors.InputMajor), chrome.MaterialPanel, cardRadius)
			paintComposerBeam(p, r, beam, beamState, showBeam, reducedMotion)
			paintInnerBloom(p, r, cardRadius)
		})
		card.DrawOver(func(p *ui.Painter, r ui.Rect) { p.Stroke(r, border, cardRadius, 0.5) })
		card.Children(func() {
			if !m.Inert {
				paintResizeHandles(c, m, card, muted)
			}
			// The suggestion menu floats above the card rather than growing it,
			// and is built here so it precedes the toolbar in tree order (labels
			// shared with the toolbar resolve to the menu while it is open).
			if !m.Inert && suggestOpen && (len(items) > 0 || len(referenceBreadcrumbs(m)) > 0) {
				paintSuggestPopup(c, m, card, items, kind, tok, colors)
			}
			if m.State != nil && (len(m.State.Attachments) > 0 || m.State.Error != "") {
				ui.Box(c).FillWidth().Disabled(m.Inert).Children(func() { paintDraftAttachments(c, m, fg, muted) })
			}
			if m.Editing {
				// InputBar.editRow floats above the card and never moves its draft.
				ui.Row(c.Key("composer-edit-status")).Label("Composer edit status").Attach(ui.AnchorTopRight, ui.AnchorBottomRight).
					Right(8).Bottom(8).MaxWidthPercent(100).Padding(4, 8).Gap(8).AlignItems(ui.Center).
					Radius(8).Background(panel).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyChatEditHint)).MinWidth(0).SingleLine().FontSize(12).FixedLineHeight(18).TextColor(muted)
					if ui.Button(c, "×").Label(i18n.T(i18n.KeyQuitConfirmCancel)).Tooltip(i18n.T(i18n.KeyQuitConfirmCancel)).
						Width(22).Height(22).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent).TextColor(muted).
						Disabled(m.State != nil && m.State.Sending).Clicked() && m.OnCancelEdit != nil {
						m.OnCancelEdit()
					}
				})
			}
			if len(mentions) > 0 && m.Draft != nil {
				paintMentionChips(c, m, mentions, muted, accent, selectIsUTF16, &selectStart, &selectEnd)
			}
			if !m.Inert && m.ChipEdit != nil && m.ChipEdit.Open && m.Draft != nil {
				paintChipEditPopup(c, m, panel, border, muted)
			}
			if m.Draft != nil {
				var area ui.Element
				placeholder := composerPlaceholder(m)
				if m.UseTextInputClient && m.TextInput != nil {
					ink := readablePlaceholder(chrome.Color(colors.Caption), chrome.Color(colors.Muted), chrome.Color(colors.InputMajor))
					area = paintDraftTextInput(c, m, fg, ink, selectStart, selectEnd)
					// A clicked mention explicitly requests selection and focus.
					// Ordinary rebuilds must preserve sidebar/toolbar/input focus.
					if selectStart >= 0 {
						area.Focus()
					}
				} else {
					// InputBar: docked min-height 36 (~1 line). The hero input is
					// three lines tall, matching the reference's taller
					// `.hero .input` writing well and 148 DIP card.
					minLines, maxLines := 1, 8
					if m.Hero {
						minLines, maxLines = 3, 10
					}
					ta := ui.TextArea(c.Key("composer-draft-input"), m.Draft).Label("Composer draft").FillWidth().Lines(minLines, maxLines).Placeholder(placeholder).
						Font(theme.RenderPreferences().Composer.Family).FontSize(theme.CurrentPreferences().Composer.Size).FixedLineHeight(composerLineHeight(theme.CurrentPreferences().Composer.Size)).Padding(4, 8, 0, 14).
						MinHeight(36).Background(ui.Transparent).BorderWidth(0).Radius(0).TextColor(fg)
					ta.Disabled(m.Inert)
					if m.Hero {
						// The card keeps 0 side padding, so the writing well owns the
						// 12-DIP text inset (the native placeholder starts 12 in).
						ta = ta.Padding(0, 12, 0, 12)
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
				requestWorkspaceFocus = m.Inert && area.Focused()
				// Toolkit AutoFocus is a first-mount request, never a per-frame
				// keep-focused loop. Do not override an already focused control.
				if !m.Inert && !c.Root().FocusWithin() {
					area.AutoFocus()
				}
				if suggestOpen && len(items) > 0 {
					wireSuggestKeys(area, m, items, kind, tok)
				}
				if !m.Inert && m.State != nil && m.State.EditFocus {
					area.Focus()
					m.State.EditFocus = false
				}
				if !m.Inert && m.NextPrompt.Text != "" && !(suggestOpen && len(items) > 0) && !(m.UseTextInputClient && m.TextInput != nil) {
					area.OnShortcut(0, ui.KeyTab, func() { acceptNextPrompt(m) })
				}
				if !m.Inert {
					wireFileDrop(area, m)
					wireWorkspaceReferenceDrop(area, m)
				}
				_, referenceDrag = ui.DragOver[filelink.Reference](area)
				fileDrag = area.FileDragOver() && strings.TrimSpace(m.WorkspaceRoot) != ""
				if m.UseTextInputClient && m.TextInput != nil {
					m.TextInput.Sync()
				}
			}
			// InputBar.row: pad 2 8 6; tools/trailing gap 12; +.add 28 selector fill;
			// .select 28/13/500 transparent; .primary send 34 info-fill.
			btnH := float32(28)
			sendH := float32(34)
			selectorFill := chrome.Color(colors.SelectorFill)
			selectorHover := chrome.Color(colors.Pressed)
			selectorPressed := selectorFill.Mix(chrome.Color(colors.FG), 0.10)
			infoFill := chrome.Color(colors.ButtonInfoFill)
			infoHover := chrome.Color(colors.ButtonInfoHover)
			infoPressed := infoHover.Mix(chrome.Color(colors.FG), 0.12)
			onAccent := chrome.Color(colors.ButtonFG)
			tools := ui.Row(c).FillWidth().Wrap().Gap(12).AlignItems(ui.Center).Disabled(m.Inert)
			if m.Hero {
				// HeroShell: the 34-DIP control row carries the card's 8-DIP side
				// inset, so the 34px primary disc fills its height.
				tools = tools.Padding(0, 8, 0, 8).Height(34)
			} else {
				tools = tools.Padding(2, 8, 6, 8)
			}
			tools.Children(func() {
				// Keep tools and trailing controls as separate wrap groups.
				ui.Row(c).MaxWidthPercent(100).MinWidth(0).Wrap().Gap(12).AlignItems(ui.Center).Children(func() {
					plus := ui.Button(c, "+").Label(i18n.T(i18n.KeyComposerCommands)).Tooltip(i18n.T(i18n.KeyComposerCommands)).
						Width(btnH).Height(btnH).Padding(0).Radius(btnH / 2).BorderWidth(0).Background(selectorFill).TextColor(fg).FontSize(16)
					if m.Editing || suggestionState(m, m.TextInput) == nil || m.State != nil && (m.State.Sending || m.State.Preparing > 0) {
						plus = plus.Disabled(true)
					}
					if !plus.IsDisabled() {
						// A button with its own background skips styleButton's hover
						// (third_party/mygo/ui widgets.go), so the fill is chosen here.
						if plus.Pressed() {
							plus = plus.Background(selectorPressed).Top(1)
						} else if plus.Hovered() {
							plus = plus.Background(selectorHover)
						}
					}
					if plus.Clicked() {
						toggleCommandLauncher(m)
					}
					paintVoiceControls(c, m, fg, muted, selectorFill, btnH)
					paintPlanModeControl(c, m, muted, btnH)
					// InputBar .select: permission scope (not PromptMode queue/steer).
					if m.Snapshot.NativeContributions.Known && m.Snapshot.NativeContributions.PermissionCommand && len(m.PermissionOptions) > 0 && m.PermissionPick != nil && m.Snapshot.SelectedPermission != "" && m.Snapshot.PermissionsErr == "" {
						presetSelect(c, *m.PermissionPick, permissionChoices(m.PermissionOptions), m.OnPermissionChange, colors, presetOptions{
							Icon:     shieldIcon,
							Text:     muted,
							FontSize: 13,
							Weight:   500,
						})
					}
				})
				ui.Row(c).Label("Composer trailing controls").MaxWidthPercent(100).MinWidth(0).Gap(12).AlignItems(ui.Center).Margin(0, 0, 0, ui.Auto).Children(func() {
					// The hero keeps the reference's control set: the occupancy ring
					// only appears once the Host reports token usage.
					if !m.Hero || m.Snapshot.ContextOccupancy != nil {
						paintContextMeter(c, m.Snapshot, colors, btnH)
					}
					paintModelPicker(c, m, colors, btnH)
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
					// The pressed state sinks the disc one DIP and darkens its fill,
					// the way the reference's :active composes. The metallic sheen is
					// the framework's decorativeButtons path (chrome.CursorEffectConfig
					// → ui.SetDecorativeEffects), so it is not painted twice here.
					primaryFill := func(b ui.Element) ui.Element {
						if b.IsDisabled() {
							return b
						}
						if b.Pressed() {
							return b.Background(infoPressed).Top(-1)
						}
						if b.Hovered() {
							return b.Background(infoHover)
						}
						return b
					}
					// Disabled discs keep the info fill; the glyph is nudged to the
					// readable side so the 0.5 opacity frame still shows an arrow.
					primaryInk := onAccent
					if composerSendDisabled(m) {
						primaryInk = chrome.EnsureContrast(onAccent, infoFill)
					}
					if primaryStops || running && child && m.OnStop != nil {
						stop := ui.Button(c, "").Label(i18n.T(i18n.KeyComposerStop)).
							Tooltip(i18n.T(i18n.KeyComposerStop) + "  " + stopShortcutHint).
							Width(sendH).Height(sendH).Padding(0).Radius(sendH / 2).
							Background(infoFill).BorderWidth(0).Top(-2).
							TextColor(onAccent).Disabled(m.OnStop == nil)
						stop = primaryFill(stop)
						stop.Children(func() { ui.Icon(c, stopIcon).Size(16, 16).PassThrough() })
						if stop.Clicked() && m.OnStop != nil {
							m.OnStop()
						}
					}
					if !primaryStops {
						send := ui.Button(c, "").Label(i18n.T(i18n.KeyComposerSend)).
							Tooltip(i18n.T(i18n.KeyComposerSend) + "  ·  " + i18n.T(i18n.KeyComposerShortcutEnter)).
							Width(sendH).Height(sendH).Padding(0).Radius(sendH / 2).
							Background(infoFill).BorderWidth(0).Top(-2).
							TextColor(primaryInk)
						if composerSendDisabled(m) {
							send = send.Disabled(true)
						}
						send = primaryFill(send)
						send.Children(func() { ui.Icon(c, sendIcon).Size(16, 16).PassThrough() })
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
			if m.Inert && m.OnRequestWorkspace != nil {
				// One transparent hit target covers disabled child controls as
				// well as the editor. No native text/IME owner is registered.
				trigger := ui.Box(c.Key("composer-workspace-trigger")).Absolute().Top(0).Left(0).Right(0).Bottom(0).
					Focusable().Role(ui.RoleButton).Description(i18n.T(i18n.KeyChatHeroChooseWorkspace)).
					OnClick(m.OnRequestWorkspace).
					OnShortcut(0, ui.KeyEnter, m.OnRequestWorkspace).
					OnShortcut(0, ui.KeySpace, m.OnRequestWorkspace).
					OnShortcut(0, ui.KeyDown, m.OnRequestWorkspace)
				if requestWorkspaceFocus {
					trigger.Focus()
				}
			}
			if !m.Inert && (fileDrag || referenceDrag) {
				// PassThrough keeps the drop target and hover routing on the draft
				// surface underneath, so the overlay is only a visual promise.
				images, bytes := composerIntake(m.State)
				overlay := ui.Box(c.Key("composer-drop-overlay")).Absolute().Top(0).Left(0).Right(0).Bottom(0).
					Radius(cardRadius).Background(chrome.Color(colors.BG).Alpha(0.92)).
					Border(1, accent).Justify(ui.Center).AlignItems(ui.Center).PassThrough()
				overlay.Children(func() {
					ui.Column(c).Gap(4).AlignItems(ui.Center).Children(func() {
						title := i18n.T(i18n.KeyComposerDropTitle)
						detail := i18n.T(i18n.KeyComposerDropDetail)
						limit := fmt.Sprintf(i18n.T(i18n.KeyComposerDropLimit),
							composerMaxImages, composerFileSize(composerMaxImageBytes))
						danger := ""
						if referenceDrag {
							title = i18n.T(i18n.KeyComposerSuggestAddReference)
							detail = i18n.T(i18n.KeyComposerSuggestDropReference)
							limit = ""
						} else {
							danger = composerDropWarning(images, bytes)
						}
						ui.Text(c, title).FontSize(13).FontWeight(500).TextColor(fg)
						ui.Text(c, detail).FontSize(11).TextColor(muted)
						if danger != "" {
							// A refusal replaces the capacity line, in the rail's
							// error ink so it reads as a warning.
							ui.Text(c, danger).FontSize(11).
								TextColor(chrome.EnsureContrast(attachmentErrorInk, chrome.Color(colors.BG)))
						} else if limit != "" {
							ui.Text(c, limit).FontSize(11).TextColor(muted)
						}
					})
				})
			}
			paintNativeCommandPopup(c, m, card, panel, border, fg, muted)
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
	ui.Row(c).Gap(4).AlignItems(ui.Center).Disabled(m.Inert).Children(func() {
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
				Tooltip(i18n.T(i18n.KeyComposerRemoveReference)).TextColor(muted).
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

// paintSuggestPopup renders the `/` command and `@` reference menu. It floats
// above the card, edge to edge with it, instead of pushing the capsule taller:
// the reference's popup is an overlay anchored at the composer's top edge.
func paintSuggestPopup(c *ui.Context, m Model, card ui.Element, items []suggestItem, kind SuggestKind, tok TokenRange, colors theme.Colors) {
	list := m.SuggestList
	if list == nil {
		return
	}
	fg := chrome.Color(colors.FG)
	hover := chrome.Color(colors.Hover)
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
	// Prefer the room above the card, where the composer sits at the window's
	// bottom edge; a short window flips the menu below so it stays on screen.
	height := float32(320)
	below := false
	if bounds := card.Bounds(); bounds.W > 0 {
		if bounds.Y < 160 {
			below = true
		} else {
			height = min(height, max(float32(120), bounds.Y-8))
		}
	}
	title := i18n.T(i18n.KeyComposerSlashCommandMenuTitle)
	if kind == SuggestMention {
		title = i18n.T(i18n.KeyComposerMentionMenuTitle)
	}
	popup := ui.Column(c.Key("composer-suggest")).Absolute().Left(0).Right(0)
	if below {
		popup = popup.TopPercent(100).Margin(4, 0, 0, 0)
	} else {
		popup = popup.BottomPercent(100).Margin(0, 0, 4, 0)
	}
	popup = menuSurface(c, popup.MaxHeight(height).Padding(4).Gap(2), colors)
	popup.Children(func() {
		paintReferenceBreadcrumbs(c, m, tok)
		ui.Text(c, title).FontSize(11).FontWeight(500).TextColor(chrome.Color(colors.Caption)).Padding(2, 6)
		drilled := false
		clicked := -1
		ui.Scroll(c).FillWidth().MaxHeight(max(80, height-40)).Children(func() {
			// Rows are plain buttons: the list widget paints its own accent
			// selection over the reference's single hover fill, and the arrows and
			// Enter already belong to the editor's keymap.
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				for i, it := range items {
					i, it := i, it
					itemRow := ui.Button(c, "").FillWidth().MinHeight(34).Padding(6, 8).Radius(8).
						AlignItems(ui.Center).BorderWidth(0).Background(ui.Transparent)
					if i == sel || itemRow.Hovered() {
						itemRow = itemRow.Background(hover)
					}
					itemRow.Children(func() {
						ui.Text(c, it.label).Grow(1).MinWidth(0).SingleLine().FontSize(13).TextColor(fg)
						if it.drill != "" && i == sel {
							ui.Row(c).Shrink(0).Gap(3).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "Tab").FontSize(10).TextColor(chrome.Color(colors.Caption)).
									Padding(0, 4).Radius(4).Background(hover)
								ui.Icon(c, chevronDown).Size(12, 12)
							})
						}
						if it.sub != "" {
							ui.Text(c, it.sub).Shrink(0).MaxWidthPercent(45).SingleLine().FontSize(12).
								TextColor(chrome.Color(colors.Caption))
						}
						if it.drill != "" && ui.Button(c, "").Label(i18n.T(i18n.KeyComposerSuggestBrowse)).
							Tooltip(i18n.T(i18n.KeyComposerSuggestBrowse)).Width(18).Height(18).Padding(0).
							BorderWidth(0).Background(ui.Transparent).Clicked() {
							drilled = true
							applySuggestDrill(m, it, tok)
						}
					})
					if itemRow.Clicked() {
						clicked = i
					}
				}
			})
			// Keyboard hints for the overlay; the rows above stay Tab/Enter
			// focusable and the editor keymap handles the arrow keys.
			ui.Text(c, i18n.T(i18n.KeyComposerMenuFooter)).FontSize(11).
				TextColor(chrome.Color(colors.Caption)).Padding(2, 6)
			if !drilled && clicked >= 0 {
				applySuggest(m, items[clicked], tok)
				if m.State != nil {
					// A row click moves the focus to the row; the draft takes it back.
					m.State.EditFocus = true
				}
			}
		})
	})
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
		if it.command && pickSlashCommand(m, it, tok) {
			return
		}
		replaceSuggestionToken(m, tok, it.insert)
	case SuggestMention:
		replaceSuggestionToken(m, tok, it.insert)
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
