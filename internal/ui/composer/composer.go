// Package composer is the multiline prompt input for M4–M6.
package composer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/voiceinput"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/filelink"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model is composer state for one frame.
type Model struct {
	Media                 *media.Owner
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
	// HostPlaceholder replaces the inert card's workspace copy while the local
	// service is not ready (internal/hoststate): the blocker is the service, not
	// a workspace the person has yet to choose.
	HostPlaceholder           string
	Disabled                  bool   // session composer unavailable
	ParentOffline             bool   // continuable child session whose parent session is offline
	Placeholder               string // explicit placeholder override
	OnRequestWorkspace        func()
	Draft                     *string
	State                     *DraftState
	OnSubmit                  func()
	OnSubmitGesture           func(accelerated bool)
	OnOpenURL                 func(string)
	OnOpenReference           func(path string) // existing session-owned file preview route
	OnAttachFiles             func([]string)
	OnRetryAttachment         func(int)
	OnWorkspaceReference      func(filelink.Reference) error
	OnPasteImage              func(bool)
	ModelPick                 *string
	PermissionOptions         []session.PermissionOption
	PermissionSwitchFailed    bool
	ConfirmPermissionInPicker bool
	AgentPresetOptions        []session.AgentPresetOption
	AgentPresetSelection      session.AgentPresetSelectionView
	PermissionPick            *string
	AgentPresetPick           *string

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
	OnSelectNativeCommandModel      func(context.Context, NativeCommandOwner, session.ModelSelection) error
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
	OnSearchFileRefs    func(query string) []string
	OnSearchReferences  func(query string, quoted bool) []session.ReferenceCandidate
	OnQueryReferences   func(context.Context, string, bool) ([]session.ReferenceCandidate, error)
	ExtraActions        func(*ui.Context)
	Child               bool // addressed child: literal commands, native editing/stop gestures
	referenceInvalidate func()

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
	action func()
	drill  string
	label  string
	sub    string
	insert string
	// section is the menu caption this row opens (launcher menus only); rows
	// sharing the previous row's section and section-less rows draw none.
	section string
	kind    SuggestKind
	command bool
}

// stopShortcutHint names the composer's double-Escape stop gesture in the stop
// button's tooltip, where the reference surfaces the binding.
const stopShortcutHint = "Esc Esc"

// View paints the prompt area with send and stop actions.
func View(c *ui.Context, m Model) {
	m.referenceInvalidate = c.Services().Invalidate
	t := design.For(c)
	fg := t.Foreground
	muted := t.MutedForeground
	accent := t.Primary
	beam := readBeamStyle(m.ComposerBeamStyle)
	reducedMotion := c.Preferences().ReduceMotion
	showBeam := m.ComposerBeam && composerRunning(m) && !reducedMotion

	if !m.Inert && m.State != nil && m.State.GroupReply != nil {
		ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyComposerReplyThread), m.State.GroupReply.ThreadID)).FontSize(design.TextXS.Size).TextColor(muted)
			if coss.Button(c, i18n.T(i18n.KeyComposerReplyCancel), coss.ButtonOptions{
				Variant:  coss.ButtonOutline,
				Size:     coss.ButtonXS,
				Disabled: m.State.Sending,
			}).Clicked() {
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
	if !suggestOpen && m.State != nil {
		m.State.references.cancelQuery()
	}
	inputDisabled := m.Inert || m.Disabled || composerParentOffline(m)
	if inputDisabled {
		suggestOpen = false
		m.SkillsOpen = false
	}
	var items []suggestItem
	if suggestOpen {
		items = buildSuggestItemsWithQuote(m, kind, query, quoted)
	}
	m.stopSuggestionsOpen = suggestOpen && (len(items) > 0 || kind == SuggestMention && referenceQueryActive(m))
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

	// InputBar.root: pad 0 side-clearance(16) 4; hero drops bottom pad.
	// HeroShell.root already pads 24 — composer keeps the shared 16 clearance.
	padBot := design.Sp(1)
	if m.Hero {
		// Hero retains InputBar side clearance inside the HeroShell stack.
		padBot = 0
	}
	composerRegion := ui.Column(c.Key("composer-region")).Padding(0, design.Sp(4), padBot, design.Sp(4))
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
			ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyComposerUnsentDraft)).FontSize(design.TextXS.Size).TextColor(muted)
				if !m.State.FailedSendsRestored() && coss.Button(c, i18n.T(i18n.KeyComposerUnsentRestore), coss.ButtonOptions{
					Variant:  coss.ButtonOutline,
					Size:     coss.ButtonXS,
					Disabled: m.State.Sending || m.State.Preparing > 0 || m.Editing,
				}).Clicked() {
					m.State.RestoreFailedSends()
				}
			})
		}
		paintDraftLinkActions(c, draft, m.OnOpenURL)
		if !inputDisabled && kind == SuggestSlash && !m.Editing {
			if m.Snapshot.SlashCommandsLoading {
				ui.Text(c, i18n.T(i18n.KeyCommandCatalogLoading)).FontSize(design.TextXS.Size).TextColor(muted).Padding(design.Sp(1), design.Sp(2))
			} else if m.Snapshot.SlashCommandsErr != nil {
				ui.Row(c).FillWidth().Gap(design.Sp(2)).Padding(design.Sp(1), design.Sp(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, m.Snapshot.SlashCommandsErr.SentenceText()).FontSize(design.TextXS.Size).TextColor(t.DestructiveForeground).Grow(1)
					if m.OnRetrySlashCommands != nil && coss.Button(c, i18n.T(i18n.KeyCommandCatalogRetry), coss.ButtonOptions{
						Variant: coss.ButtonOutline,
						Size:    coss.ButtonXS,
					}).Clicked() {
						m.OnRetrySlashCommands()
					}
				})
			}
			if m.Snapshot.NativeContributions.Error != "" {
				ui.Text(c, m.Snapshot.NativeContributions.Error).FontSize(design.TextXS.Size).TextColor(t.DestructiveForeground).Padding(design.Sp(1), design.Sp(2))
			}
		}
		if m.State != nil {
			m.State.ModelCommand.Sync(m.Snapshot.SelectedID)
			m.State.PermissionCommand.Sync(m.Snapshot.SelectedID)
		}
		if m.SkillsOpen {
			paintSkillsPopup(c, m, skillItems, skillsQuery)
		}
		paintLeadingClaimHint(c, m, muted)
		if m.Hero {
			// ConversationRoot.heroWorkspaceRow has a 4-DIP top margin and
			// a 28-DIP row, followed by the composer's 8-DIP stack gap.
			ui.Row(c.Key("composer-hero-preset")).Label("Composer hero preset row").FillWidth().MinWidth(0).Gap(design.Sp(0.5)).AlignItems(ui.Center).
				Margin(design.Sp(1), 0, design.Sp(2), 0).Padding(0, design.Sp(4), 0, design.Sp(5)).Children(func() {
				wsLabel := strings.TrimSpace(m.WorkspaceRoot)
				// Session groups without a directory carry sentinel ids such as
				// "no-workspace"; only an absolute root (POSIX or Windows) names a workspace.
				if !isAbsoluteWorkspacePath(wsLabel) {
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
				workspace := chip(c, false, fg).Label("Composer workspace").Padding(0, design.Sp(2), 0, design.Sp(2)).MaxWidth(360)
				if m.WorkspacePickerAnchor != nil {
					workspace.Bind(m.WorkspacePickerAnchor)
				}
				if m.OnPickHeroWorkspace == nil || (m.State != nil && (m.State.Sending || m.State.Preparing > 0)) {
					workspace = workspace.Disabled(true)
				}
				workspace.Children(func() {
					ui.Icon(c, folderIcon).Size(design.IconSM, design.IconSM).TextColor(muted).Shrink(0)
					ui.Text(c, wsLabel).FontSize(design.TextSM.Size).FontWeight(design.WeightMedium).TextColor(fg).SingleLine().MaxLines(1).Shrink(1).MinWidth(0)
					chevron(c, false)
				})
				if workspace.Clicked() && m.OnPickHeroWorkspace != nil {
					m.OnPickHeroWorkspace()
				}
				if m.AgentPresetSelection.Error != "" {
					ui.Text(c, m.AgentPresetSelection.Error).FontSize(design.TextXS.Size).TextColor(t.DestructiveForeground)
				}
				presetLabel := i18n.T(i18n.KeyComposerPresetDefault)
				if m.AgentPresetPick != nil && strings.TrimSpace(*m.AgentPresetPick) != "" {
					presetLabel = presetDisplayName(*m.AgentPresetPick)
				} else if len(m.Snapshot.AgentPresetLabels) > 0 {
					presetLabel = presetDisplayName(m.Snapshot.AgentPresetLabels[0])
				}
				if len(m.AgentPresetOptions) > 0 && m.AgentPresetPick != nil && m.AgentPresetSelection.CanSelect && !m.AgentPresetSelection.Busy {
					presetSelect(c, presetLabel, agentPresetChoices(m.AgentPresetOptions), m.OnAgentPresetChange, presetOptions{
						Leading:  func() { paintPresetIntroduction(c, m, presetLabel, fg) },
						Text:     fg,
						FontSize: design.TextSM.Size,
						Weight:   design.WeightMedium,
					})
				} else {
					ui.Row(c).Label("Composer hero preset").Gap(design.Sp(1)).AlignItems(ui.Center).Height(chipHeight).Padding(0, design.Sp(2), 0, design.Sp(2)).Children(func() {
						ui.Icon(c, PresetAgentIcon).Size(design.IconSM, design.IconSM).TextColor(fg)
						ui.Text(c, presetLabel).FontSize(design.TextSM.Size).FontWeight(design.WeightMedium).TextColor(fg)
						ui.Icon(c, coss.IconChevronDown.SVG()).Size(design.IconXS, design.IconXS).TextColor(muted)
					})
				}
			})
		}
		// coss Composer frame (RadiusXL, coss input border/ring, 8 top pad, 12 gap;
		// Hero retains the 114-DIP card height contract).
		cardRadius := design.RadiusXL
		card := ui.Column(c.Key("composer-card")).Label("Composer card").FillWidth().
			Radius(cardRadius).Padding(design.Sp(2), 0, 0, 0).Gap(design.Sp(3))
		if m.Hero {
			card = card.MinHeight(114)
		}
		if m.OnCardLayout != nil {
			card.ObserveLayout(m.OnCardLayout)
		}
		focused := !inputDisabled && card.FocusWithin()
		cardFill := composerCardFill(t)
		cardBorder := composerCardBorder(t, focused)
		composerCardElevation(card, t, focused, inputDisabled)
		beamState := ui.Local(card, "composer-beam-timeline", func() beamTimeline { return beamTimeline{} })
		card.Draw(func(p *ui.Painter, r ui.Rect) {
			p.Fill(r, cardFill, cardRadius)
			paintComposerBeam(p, r, beam, beamState, showBeam, reducedMotion)
			if t.IsDark() {
				paintInnerBloom(p, r, cardRadius)
			}
		})
		card.DrawOver(func(p *ui.Painter, r ui.Rect) {
			p.Stroke(r, cardBorder, cardRadius, design.BorderWidth)
		})
		card.Children(func() {
			if !inputDisabled {
				paintResizeHandles(c, m, card, muted)
			}
			// The suggestion menu floats above the card rather than growing it,
			// and is built here so it precedes the toolbar in tree order (labels
			// shared with the toolbar resolve to the menu while it is open).
			if !inputDisabled && suggestOpen && (len(items) > 0 || len(referenceBreadcrumbs(m)) > 0 || referenceQueryActive(m)) {
				paintSuggestPopup(c, m, card, items, kind, tok)
			}
			if !inputDisabled && m.ChipEdit != nil && m.ChipEdit.Open && m.Draft != nil {
				paintChipEditPopup(c, m)
			}
			if m.State != nil && (len(m.State.Attachments) > 0 || m.State.Error != "") {
				ui.Box(c).FillWidth().Disabled(inputDisabled).Children(func() { paintDraftAttachments(c, m, fg, muted) })
			}
			if m.Editing {
				// InputBar.editRow floats above the card and never moves its draft.
				editRow := ui.Row(c.Key("composer-edit-status")).Label("Composer edit status").Attach(ui.AnchorTopRight, ui.AnchorBottomRight).
					Right(design.Sp(2)).Bottom(design.Sp(2)).MaxWidthPercent(100).Padding(design.Sp(1), design.Sp(2)).Gap(design.Sp(2)).
					AlignItems(ui.Center).Radius(design.RadiusMD).Background(t.Popover).Border(design.BorderWidth, t.Border)
				applyShadows(editRow, design.ShadowMd, design.TW.Black, 0.05)
				editRow.Children(func() {
					ui.Text(c, i18n.T(i18n.KeyChatEditHint)).MinWidth(0).SingleLine().FontSize(design.TextXS.Size).FixedLineHeight(18).TextColor(muted)
					if coss.Button(c, "", coss.ButtonOptions{
						Variant:  coss.ButtonGhost,
						Size:     coss.ButtonIconXS,
						Icon:     coss.IconX,
						Label:    i18n.T(i18n.KeyQuitConfirmCancel),
						Disabled: m.State != nil && m.State.Sending,
					}).Size(22, 22).Radius(design.RadiusSM).Tooltip(i18n.T(i18n.KeyQuitConfirmCancel)).Clicked() && m.OnCancelEdit != nil {
						m.OnCancelEdit()
					}
				})
			}
			if len(mentions) > 0 && m.Draft != nil && !selectIsUTF16 {
				paintMentionChips(c, m, mentions, muted, accent, selectIsUTF16, &selectStart, &selectEnd)
			}
			if m.Draft != nil {
				var area ui.Element
				placeholder := composerPlaceholder(m)
				if m.UseTextInputClient && m.TextInput != nil {
					placeholderInk := t.MutedForeground.Alpha(design.OpacityPlaceholder)
					if contrastRatio(placeholderInk, cardFill) < 3 {
						placeholderInk = t.MutedForeground
					}
					area = paintDraftTextInput(c, m, fg, placeholderInk, selectStart, selectEnd)
					// A clicked mention explicitly requests selection and focus.
					// Ordinary rebuilds must preserve sidebar/toolbar/input focus.
					if selectStart >= 0 {
						area.Focus()
					}
				} else {
					// InputBar keeps the same text insets; Hero raises the empty
					// writing well from 36 to 52 DIP without adding a third line.
					minLines, maxLines := 1, 8
					if m.Hero {
						minLines, maxLines = 2, 10
					}
					taPlaceholder := placeholder
					if nextPromptVisible(m) {
						taPlaceholder = ""
					}
					ta := ui.TextAreaBase(c.Key("composer-draft-input"), m.Draft).Label("Composer draft").FillWidth().Lines(minLines, maxLines).Placeholder(taPlaceholder).
						Font(theme.FontForSkin(theme.RenderPreferences(), theme.ResolvedSkin(m.ThemeID, m.SystemDark), theme.SkinComposerFont).Family).FontSize(theme.CurrentPreferences().Composer.Size).FixedLineHeight(composerLineHeight(theme.CurrentPreferences().Composer.Size)).Padding(design.Sp(1), design.Sp(2), 0, design.Sp(3.5)).
						MinHeight(36).TextColor(fg).FocusRing(false)
					ta.Disabled(inputDisabled)
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
				requestWorkspaceFocus = m.Inert && area.Focused()
				// Toolkit AutoFocus is a first-mount request, never a per-frame
				// keep-focused loop. Do not override an already focused control.
				if !inputDisabled && !c.Root().FocusWithin() {
					area.AutoFocus()
				}
				if suggestOpen && len(items) > 0 {
					wireSuggestKeys(area, m, items, kind, tok)
				}
				if !inputDisabled && m.State != nil && m.State.EditFocus {
					area.Focus()
					m.State.EditFocus = false
				}
				if !inputDisabled && m.NextPrompt.Text != "" && !(suggestOpen && len(items) > 0) && !(m.UseTextInputClient && m.TextInput != nil) {
					area.OnShortcut(0, ui.KeyTab, func() { acceptNextPrompt(m) })
				}
				if !inputDisabled {
					wireFileDrop(area, m)
					wireWorkspaceReferenceDrop(area, m)
				}
				_, referenceDrag = ui.DragOver[filelink.Reference](area)
				fileDrag = area.FileDragOver() && strings.TrimSpace(m.WorkspaceRoot) != ""
				if m.UseTextInputClient && m.TextInput != nil {
					m.TextInput.syncFont(theme.FontForSkin(theme.RenderPreferences(), theme.ResolvedSkin(m.ThemeID, m.SystemDark), theme.SkinComposerFont))
				}
			}
			// InputBar keeps ONE toolbar row at every width: no Wrap, so the
			// trailing group's shrinkable chip (model name) ellipsizes instead
			// of dropping beneath the row.
			tools := ui.Row(c).FillWidth().Gap(design.Sp(3)).AlignItems(ui.Center).Padding(design.Sp(0.5), design.Sp(2), design.Sp(1.5), design.Sp(2)).Disabled(m.Inert)
			tools.Children(func() {
				// Keep tools and trailing controls as separate groups; the tools
				// keep their natural width so only the model chip shrinks.
				ui.Row(c).Shrink(0).Gap(design.Sp(3)).AlignItems(ui.Center).Children(func() {
					plusDisabled := inputDisabled || m.Editing || suggestionState(m, m.TextInput) == nil || m.State != nil && (m.State.Sending || m.State.Preparing > 0)
					plus := coss.Button(c, "", coss.ButtonOptions{
						Variant:  coss.ButtonSecondary,
						Size:     coss.ButtonIconSM,
						Icon:     coss.IconPlus,
						Label:    i18n.T(i18n.KeyComposerCommands),
						Disabled: plusDisabled,
					}).Radius(design.RadiusFull).Tooltip(i18n.T(i18n.KeyComposerCommands))
					if plus.Clicked() {
						toggleCommandLauncher(m)
					}
					paintVoiceControls(c, m, chipHeight)
					paintPlanModeControl(c, m, muted, chipHeight)
					// InputBar .select: permission scope (not PromptMode queue/steer).
					paintPermissionPicker(c, m, muted)
				})
				ui.Row(c).Label("Composer trailing controls").MaxWidthPercent(100).MinWidth(0).Gap(design.Sp(3)).AlignItems(ui.Center).Margin(0, 0, 0, ui.Auto).Children(func() {
					if m.ExtraActions != nil {
						m.ExtraActions(c)
					}
					// The hero keeps the reference's control set: the occupancy ring
					// only appears once the Host reports token usage.
					if !m.Hero || m.Snapshot.ContextOccupancy != nil {
						paintContextMeter(c, m.Snapshot, chipHeight)
					}
					paintModelPicker(c, m, chipHeight)
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
					parentOffline := composerParentOffline(m)
					primaryStops := running && (!child || parentOffline) && (composerDraftEmpty(m) || m.StopBlocked || parentOffline)
					if primaryStops || running && child && m.OnStop != nil {
						stop := coss.Button(c, "", coss.ButtonOptions{
							Variant:  coss.ButtonDefault,
							Size:     coss.ButtonIcon,
							Label:    i18n.T(i18n.KeyComposerStop),
							Disabled: m.OnStop == nil,
							Content: func() {
								ui.Icon(c, stopIcon).Size(design.IconSM, design.IconSM).TextColor(t.PrimaryForeground).PassThrough()
							},
						}).Radius(design.RadiusFull).Tooltip(i18n.T(i18n.KeyComposerStop) + "  " + stopShortcutHint)
						if stop.Clicked() && m.OnStop != nil {
							m.OnStop()
						}
					}
					if !primaryStops {
						send := coss.Button(c, "", coss.ButtonOptions{
							Variant:  coss.ButtonDefault,
							Size:     coss.ButtonIcon,
							Icon:     coss.IconArrowUp,
							Label:    i18n.T(i18n.KeyComposerSend),
							Disabled: composerSendDisabled(m),
						}).Radius(design.RadiusFull).Tooltip(i18n.T(i18n.KeyComposerSend) + "  ·  " + i18n.T(i18n.KeyComposerShortcutEnter))
						if send.Clicked() && (handleModelCommand(m) || m.Draft != nil && refuseCommandAttachments(m, *m.Draft)) {
							// Local entry or attachment refusal consumes no prompt or attachment.
						} else if send.Clicked() && m.OnSubmitGesture != nil {
							m.OnSubmitGesture(false)
						} else if send.Clicked() && m.OnSubmit != nil {
							m.OnSubmit()
						} else if send.Clicked() && m.OnSend != nil && m.Draft != nil {
							text := *m.Draft
							if handleSlashSend(m, text) {
								if m.State == nil || m.State.Error == "" {
									*m.Draft = ""
								}
							} else {
								*m.Draft = ""
								m.OnSend(text)
							}
							if m.TextInput != nil && (m.State == nil || m.State.Error == "") {
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
					Focusable().Role(ui.RoleButton).Cursor(ui.CursorPointer).Label(WorkspacePlaceholder()).Description(i18n.T(i18n.KeyChatHeroChooseWorkspace)).
					OnClick(m.OnRequestWorkspace).
					OnShortcut(0, ui.KeyEnter, m.OnRequestWorkspace).
					OnShortcut(0, ui.KeySpace, m.OnRequestWorkspace).
					OnShortcut(0, ui.KeyDown, m.OnRequestWorkspace)
				if requestWorkspaceFocus {
					trigger.Focus()
				}
			}
			if !inputDisabled && (fileDrag || referenceDrag) {
				// PassThrough keeps the drop target and hover routing on the draft
				// surface underneath, so the overlay is only a visual promise.
				images, bytes := composerIntake(m.State)
				overlay := ui.Box(c.Key("composer-drop-overlay")).Absolute().Top(0).Left(0).Right(0).Bottom(0).
					Radius(cardRadius).Background(t.Background.Alpha(0.92)).
					Border(design.BorderWidth, accent).Justify(ui.Center).AlignItems(ui.Center).PassThrough()
				overlay.Children(func() {
					ui.Column(c).Gap(design.Sp(1)).AlignItems(ui.Center).Children(func() {
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
						ui.Text(c, title).FontSize(design.TextSM.Size).FontWeight(design.WeightMedium).TextColor(fg)
						ui.Text(c, detail).FontSize(design.TextXS.Size).TextColor(muted)
						if danger != "" {
							// A refusal replaces the capacity line, in the destructive
							// ink so it reads as a warning.
							ui.Text(c, danger).FontSize(design.TextXS.Size).TextColor(t.DestructiveForeground)
						} else if limit != "" {
							ui.Text(c, limit).FontSize(design.TextXS.Size).TextColor(muted)
						}
					})
				})
			}
			paintNativeCommandPopup(c, m, card)
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
		if m.OnQueryReferences != nil && m.State != nil {
			candidates = m.State.references.query(m, query, quoted)
		} else if m.OnSearchReferences != nil {
			candidates = m.OnSearchReferences(query, quoted)
		}
		if len(candidates) == 0 && m.OnQueryReferences == nil {
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

type draftLink struct {
	Label, URL string
}

func extractDraftLinks(source string) []draftLink {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	md := goldmark.New(goldmark.WithExtensions(extension.Linkify))
	doc := md.Parser().Parse(text.NewReader([]byte(source)))
	var out []draftLink
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination, label string
		switch link := n.(type) {
		case *ast.Link:
			destination = string(link.Destination)
			var buf bytes.Buffer
			for ch := link.FirstChild(); ch != nil; ch = ch.NextSibling() {
				if tx, ok := ch.(*ast.Text); ok {
					buf.Write(tx.Segment.Value([]byte(source)))
				}
			}
			label = strings.TrimSpace(buf.String())
		case *ast.AutoLink:
			destination = string(link.URL([]byte(source)))
			label = string(link.Label([]byte(source)))
		default:
			return ast.WalkContinue, nil
		}
		destination = strings.TrimSpace(destination)
		parsed, err := url.Parse(destination)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
			return ast.WalkContinue, nil
		}
		for _, existing := range out {
			if existing.URL == destination {
				return ast.WalkContinue, nil
			}
		}
		if label == "" {
			label = destination
		}
		out = append(out, draftLink{Label: label, URL: destination})
		return ast.WalkContinue, nil
	})
	return out
}

func paintDraftLinkActions(c *ui.Context, source string, onLink func(string)) {
	if onLink == nil {
		return
	}
	links := extractDraftLinks(source)
	if len(links) == 0 {
		return
	}
	t := design.For(c)
	ui.Column(c).FillWidth().Gap(design.Sp(1)).Children(func() {
		for _, link := range links {
			if coss.Button(c, link.Label, coss.ButtonOptions{
				Variant: coss.ButtonLink,
				Size:    coss.ButtonXS,
				Icon:    coss.IconExternalLink,
			}).Label("↗ " + link.Label).Tooltip(link.URL).MaxWidthPercent(100).MinWidth(0).TextColor(t.Primary).Clicked() {
				onLink(link.URL)
			}
		}
	})
}

func paintSkillsPopup(c *ui.Context, m Model, items []session.SkillOption, skillsQuery string) {
	t := design.For(c)
	ui.Column(c).Gap(design.Sp(0.5)).Padding(0, 0, design.Sp(1), 0).Children(func() {
		menuSurface(c, ui.Column(c).Padding(design.Sp(1)).MaxHeight(280)).Children(func() {
			if m.SkillsFilter != nil {
				ui.Row(c).Gap(design.Sp(1.5)).AlignItems(ui.Center).Padding(design.Sp(0.5), design.Sp(1), design.Sp(0.5), design.Sp(1)).Children(func() {
					coss.Input(c, m.SkillsFilter, coss.InputOptions{
						Size:        coss.InputSM,
						Placeholder: i18n.T(i18n.KeyComposerSkillsFilter),
					}).Grow(1)
					q := strings.TrimSpace(skillsQuery)
					if m.OnSearchSkillHub != nil && q != "" {
						if coss.Button(c, i18n.T(i18n.KeyComposerSkillsHubSearch), coss.ButtonOptions{
							Variant: coss.ButtonOutline,
							Size:    coss.ButtonXS,
						}).Clicked() {
							m.OnSearchSkillHub(q)
						}
					}
				})
			}
			if m.SkillCreateName != nil && m.OnCreateSkill != nil {
				ui.Row(c).Gap(design.Sp(1.5)).AlignItems(ui.Center).Padding(design.Sp(0.5), design.Sp(1), design.Sp(0.5), design.Sp(1)).Children(func() {
					coss.Input(c, m.SkillCreateName, coss.InputOptions{
						Size:        coss.InputSM,
						Placeholder: i18n.T(i18n.KeyComposerSkillsCreateName),
					}).Grow(1)
					if coss.Button(c, i18n.T(i18n.KeyComposerSkillsCreate), coss.ButtonOptions{
						Variant: coss.ButtonOutline,
						Size:    coss.ButtonXS,
					}).Clicked() {
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
					ui.Row(c).Gap(design.Sp(1.5)).AlignItems(ui.Center).Padding(design.Sp(1), design.Sp(1.5), design.Sp(1), design.Sp(1.5)).Children(func() {
						ui.Column(c).Grow(1).Children(func() {
							ui.Text(c, h.SkillName).FontSize(design.TextXS.Size).FontWeight(design.WeightSemibold).TextColor(t.PopoverForeground)
							sub := h.Description
							if sub == "" {
								sub = h.Repo
							}
							if sub != "" {
								ui.Text(c, sub).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
							}
						})
						if h.Status != "" {
							ui.Text(c, h.Status).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
						}
						if !h.Installed && h.Status != "conflict" && m.OnInstallSkillHub != nil {
							if coss.Button(c, i18n.T(i18n.KeyComposerSkillsHubInstall), coss.ButtonOptions{
								Variant: coss.ButtonOutline,
								Size:    coss.ButtonXS,
							}).Clicked() {
								m.OnInstallSkillHub(h)
							}
						}
					})
				}
			}
			if len(items) == 0 {
				msg := i18n.T(i18n.KeyComposerSkillsEmpty)
				filterEmpty := strings.TrimSpace(skillsQuery) == ""
				if m.Snapshot.SkillsErr != nil && filterEmpty && len(m.Skills) == 0 {
					msg = m.Snapshot.SkillsErr.SentenceText()
				}
				if len(m.SkillHubHits) == 0 {
					ui.Text(c, msg).FontSize(design.TextXS.Size).TextColor(t.MutedForeground).Padding(design.Sp(1.5), design.Sp(2), design.Sp(1.5), design.Sp(2))
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
				bg := t.Popover
				rowFg := t.PopoverForeground
				if row.Index == sel {
					bg = t.Accent
					rowFg = t.AccentForeground
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
				ui.Column(row.Context).Padding(design.Sp(1.5), design.Sp(2), design.Sp(1.5), design.Sp(2)).Radius(design.RadiusSM).Background(bg).Children(func() {
					ui.Row(row.Context).Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
						ui.Text(row.Context, "/"+sk.Name).FontSize(design.TextXS.Size).FontWeight(design.WeightSemibold).TextColor(rowFg).Grow(1)
						if m.OnToggleSkillModel != nil {
							label := i18n.T(i18n.KeyComposerSkillsToggleModel)
							if coss.Button(row.Context, label, coss.ButtonOptions{
								Variant: coss.ButtonGhost,
								Size:    coss.ButtonXS,
							}).Clicked() {
								m.OnToggleSkillModel(sk.Name, sk.UserOnly) // UserOnly ⇒ currently not model-invocable
							}
						}
						if sk.UserOnly && m.OnDeleteSkill != nil {
							if coss.Button(row.Context, i18n.T(i18n.KeyComposerSkillsDelete), coss.ButtonOptions{
								Variant: coss.ButtonGhost,
								Size:    coss.ButtonXS,
							}).Clicked() {
								m.OnDeleteSkill(sk.Name)
							}
						}
					})
					if sub != "" {
						ui.Text(row.Context, sub).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
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
	t := design.For(c)
	ui.Row(c).Gap(design.Sp(1)).AlignItems(ui.Center).Disabled(m.Inert || m.Disabled || composerParentOffline(m)).Children(func() {
		for i, ment := range mentions {
			i, ment := i, ment
			label := ChipLabel(ment)
			color := t.Primary
			if ment.Kind == DraftMentionSkill {
				color = t.MutedForeground
			}
			chip := ui.ButtonBase(c).Radius(design.RadiusSM).Padding(design.Sp(0.5), design.Sp(1.5)).
				TextColor(color).FocusRing(false).Cursor(ui.CursorPointer).Drag(chipDrag{Index: i}).Children(func() {
				ui.Text(c, label).FontSize(design.TextXS.Size).FontWeight(design.WeightMedium).TextColor(color).SingleLine()
			})
			if _, over := ui.DragOver[chipDrag](chip); over {
				chip = chip.Background(t.Accent)
			}
			if from, ok := ui.Drop[chipDrag](chip); ok && m.Draft != nil {
				before := *m.Draft
				after := ReorderDraftMentions(before, from.Index, i)
				if after != before && m.TextInput != nil && m.TextInput.Draft == m.Draft {
					if _, marked := m.TextInput.MarkedRange(); !marked {
						m.TextInput.ReplaceText(&ui.TextInputRange{End: ui.UTF16Len(before)}, after)
					}
				} else if m.TextInput == nil {
					*m.Draft = after
				}
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
			if coss.Button(c, "", coss.ButtonOptions{
				Variant:  coss.ButtonGhost,
				Size:     coss.ButtonIconXS,
				Icon:     coss.IconX,
				Label:    "Remove reference " + ment.Text,
				Disabled: m.State != nil && m.State.Sending,
			}).Tooltip(i18n.T(i18n.KeyComposerRemoveReference)).Clicked() {
				removeDraftReference(m, ment)
			}
		}
	})
}

func paintChipEditPopup(c *ui.Context, m Model) {
	ed := m.ChipEdit
	if ed == nil {
		return
	}
	t := design.For(c)
	menuPanelLook(c, ui.Column(c).Padding(design.Sp(1.5), design.Sp(2)).Gap(design.Sp(1.5))).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyComposerChipEdit)).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
		coss.Input(c, &ed.Draft, coss.InputOptions{
			Size:        coss.InputSM,
			Placeholder: i18n.T(i18n.KeyComposerChipEditPlaceholder),
		}).FillWidth()
		ui.Row(c).Gap(design.Sp(2)).Children(func() {
			if coss.Button(c, i18n.T(i18n.KeyComposerChipEditApply), coss.ButtonOptions{
				Variant: coss.ButtonDefault,
				Size:    coss.ButtonXS,
			}).Clicked() {
				target := DraftMention{Text: ed.Orig, Start: ed.Start, End: ed.Start + len(ed.Orig)}
				replaceDraftReference(m, target, ed.Draft)
				closeChipEdit(m)
			}
			if coss.Button(c, i18n.T(i18n.KeyComposerChipEditCancel), coss.ButtonOptions{
				Variant: coss.ButtonOutline,
				Size:    coss.ButtonXS,
			}).Clicked() {
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
func paintSuggestPopup(c *ui.Context, m Model, card ui.Element, items []suggestItem, kind SuggestKind, tok TokenRange) {
	list := m.SuggestList
	if list == nil {
		return
	}
	t := design.For(c)
	fg := t.PopoverForeground
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
		popup = popup.TopPercent(100).Margin(design.Sp(1), 0, 0, 0)
	} else {
		popup = popup.BottomPercent(100).Margin(0, 0, design.Sp(1), 0)
	}
	popup = menuPanelLook(c, popup.MaxHeight(height).Padding(design.Sp(1)).Gap(0))
	popup.Children(func() {
		paintReferenceBreadcrumbs(c, m, tok)
		ui.Text(c, title).FontSize(design.TextXS.Size).FontWeight(design.WeightMedium).FixedLineHeight(design.TextXS.LineHeight).TextColor(t.MutedForeground).Padding(design.Sp(1), design.Sp(2), design.Sp(0.5), design.Sp(2))
		if kind == SuggestMention && referenceQueryActive(m) {
			pending, failure := referenceQueryStatus(m)
			if pending {
				ui.Text(c, "正在查找引用…").FontSize(design.TextXS.Size).TextColor(fg).Padding(design.Sp(1.5))
			}
			if failure != "" {
				ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, failure).Grow(1).FontSize(design.TextXS.Size).TextColor(t.DestructiveForeground)
					if coss.Button(c, "重试", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() {
						m.State.references.cancelQuery()
					}
				})
			} else if !pending && len(items) == 0 {
				ui.Text(c, "没有匹配的引用").FontSize(design.TextXS.Size).TextColor(t.MutedForeground).Padding(design.Sp(1.5))
			}
		}
		drilled := false
		clicked := -1
		type suggestNavState struct {
			LastSel      int
			HoveredIndex int
		}
		nav := ui.Local(popup, "suggest-nav", func() suggestNavState {
			return suggestNavState{LastSel: -1, HoveredIndex: -1}
		})
		scrollActive := nav.LastSel != sel
		nav.LastSel = sel
		coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
			// Rows are plain buttons: the list widget paints its own accent
			// selection over the reference's single hover fill, and the arrows and
			// Enter already belong to the editor's keymap.
			ui.Column(c).FillWidth().Gap(0).Children(func() {
				section := ""
				hoveredRow := -1
				for i, it := range items {
					i, it := i, it
					if it.section != "" && it.section != section {
						ui.Text(c, it.section).FontSize(design.TextXS.Size).FontWeight(design.WeightMedium).FixedLineHeight(design.TextXS.LineHeight).TextColor(t.MutedForeground).Padding(design.Sp(1), design.Sp(2), design.Sp(0.5), design.Sp(2))
					}
					section = it.section
					itemRow := ui.ButtonBase(c).FillWidth().MinHeight(menuRowHeight).Padding(design.Sp(1.5), design.Sp(2)).Radius(design.RadiusLG).
						AlignItems(ui.Center).Justify(ui.Start).Gap(design.Sp(1.5)).FocusRing(false).Cursor(ui.CursorPointer)
					if itemRow.Hovered() {
						hoveredRow = i
						if nav.HoveredIndex != i {
							nav.HoveredIndex = i
							sel = i
							nav.LastSel = i
							if list != nil && list.Selected != nil {
								*list.Selected = i
							}
						}
					}
					rowFg := fg
					if i == sel {
						itemRow = itemRow.Background(t.Accent)
						rowFg = t.AccentForeground
						if scrollActive {
							itemRow.ScrollIntoView()
							scrollActive = false
						}
					}
					itemRow.Children(func() {
						ui.Text(c, it.label).Grow(1).MinWidth(0).SingleLine().FontSize(design.TextSM.Size).TextColor(rowFg)
						if it.drill != "" && i == sel {
							ui.Row(c).Shrink(0).Gap(design.Sp(0.5)).AlignItems(ui.Center).Children(func() {
								coss.Kbd(c, "Tab")
								ui.Icon(c, coss.IconChevronDown.SVG()).Size(design.IconXS, design.IconXS).TextColor(t.MutedForeground)
							})
						}
						if it.sub != "" {
							ui.Text(c, it.sub).Shrink(0).MaxWidthPercent(45).SingleLine().FontSize(design.TextXS.Size).
								TextColor(t.MutedForeground)
						}
						if it.drill != "" && coss.Button(c, "", coss.ButtonOptions{
							Variant: coss.ButtonGhost,
							Size:    coss.ButtonIconXS,
							Icon:    coss.IconChevronDown,
							Label:   i18n.T(i18n.KeyComposerSuggestBrowse),
						}).Size(18, 18).Tooltip(i18n.T(i18n.KeyComposerSuggestBrowse)).Clicked() {
							drilled = true
							applySuggestDrill(m, it, tok)
						}
					})
					if itemRow.Clicked() {
						clicked = i
					}
				}
				if hoveredRow < 0 {
					nav.HoveredIndex = -1
				}
			})
			// Keyboard hints for the overlay; the rows above stay Tab/Enter
			// focusable and the editor keymap handles the arrow keys.
			ui.Text(c, i18n.T(i18n.KeyComposerMenuFooter)).FontSize(design.TextXS.Size).
				TextColor(t.MutedForeground).Padding(design.Sp(0.5), design.Sp(1.5))
			if !drilled && clicked >= 0 {
				applySuggest(m, items[clicked], tok)
				if m.State != nil {
					// A row click moves the focus to the row; the draft takes it back.
					m.State.EditFocus = true
				}
			}
		}).FillWidth().MaxHeight(max(80, height-40))
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
	if refuseCommandAttachments(m, text) {
		return true
	}
	if m.OnExecuteSlash != nil {
		if m.State != nil && len(m.State.Attachments) > 0 {
			m.State.Error = CommandAttachmentsUnsupportedNotice(line)
			return true
		}
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
		row := SlashCommand{Line: name, Description: it.Description}
		if it.DefinitionId != nil {
			if label, description, ok := builtinSlashFace(*it.DefinitionId); ok {
				row.Label, row.Description = label, description
			}
		}
		out = append(out, row)
	}
	return out
}

func paintDraftAttachments(c *ui.Context, m Model, fg, muted ui.Color) {
	paintAttachmentRail(c, m, fg, muted)
}

func filepathCleanForLabel(p string) string {
	return strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
}

// isAbsoluteWorkspacePath recognizes both POSIX (/...) and Windows (X:\, X:/,
// \\...) roots so remote POSIX Host workspaces remain valid on Windows.
func isAbsoluteWorkspacePath(p string) bool {
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\\`) {
		return true
	}
	if len(p) >= 3 && ((p[0] >= 'a' && p[0] <= 'z') || (p[0] >= 'A' && p[0] <= 'Z')) && p[1] == ':' && (p[2] == '/' || p[2] == '\\') {
		return true
	}
	return filepath.IsAbs(p)
}
