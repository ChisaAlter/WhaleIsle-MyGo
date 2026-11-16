package assistantctl

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

// BuildPanelModel assembles the complete native assistantui.Model for one UI
// frame and returns the corresponding FrameObservation flags plus a commit hook
// that writes back the frame's edited draft when View finishes.
func (s *Service) BuildPanelModel(opts PanelBuildOptions) (assistantui.Model, FrameObservation, func()) {
	if s == nil {
		return assistantui.Model{}, FrameObservation{}, func() {}
	}
	host := s.safeHost()
	ep := opts.Endpoint
	origin := ep.Origin
	win := host.Window()

	whaleOn := true
	dshbotOn := false
	if cfgStore := host.Config(); cfgStore != nil {
		cfg := cfgStore.Get()
		whaleOn = cfg.WhaleAssistantEnabled
		dshbotOn = cfg.DshbotEnabled
	}
	whaleVendor, dshbotVendor := s.CachedVendors()
	asst := s.EnsureAssistantStore()
	asst.Attach(func() {
		if win != nil {
			host.WindowUpdate(win, func() {})
		}
	})

	s.mu.Lock()
	assistantDraft := s.draft
	lastNative := s.lastAct
	pluginBusy := s.pluginBusy
	pluginDetail := s.pluginDetail
	roster := s.dshbotRoster
	s.mu.Unlock()

	var dshbotRows []dshbotstore.Row
	var dshbotRosterAt string
	if roster != nil {
		snap := roster.Snapshot()
		if snap.Origin == origin {
			dshbotRows = snap.Rows
			dshbotRosterAt = snap.FetchedAt
		}
	}

	asstSnap := asst.Snapshot()
	assistantDraft = asst.ConsumeAcceptedDraft(assistantDraft)
	dshbotSelectedID := ""
	if asstSnap.DshbotOrigin == origin {
		dshbotSelectedID = asstSnap.DshbotID
	}
	if lastNative == "" {
		lastNative = asstSnap.LastAction
	}
	if lastNative == "" {
		lastNative = opts.SessionLastAction
	}
	history := asstSnap.History
	if len(history) == 0 && len(opts.SessionMessages) > 0 {
		history = assiststore.HistoryFromSessionMessages(opts.SessionMessages, 8)
	}

	taskModel := s.TasksModel(ep, dshbotSelectedID)
	routineModel := s.RoutinesModel(ep, dshbotSelectedID)
	routineCreateModel := s.RoutineCreateModel(ep, dshbotSelectedID, routineModel.State, routineModel.OnRefresh)
	roomCreateRendered := asstSnap.Mode == assiststore.ModeDshbot
	roomCreateModel := s.RoomCreateModel(ep)
	botCreateModel := s.BotCreateModel(ep)
	botVisibilityModel := s.BotVisibilityModel(ep)
	contactsModel := assistantui.ContactsModel{}
	if asstSnap.Mode == assiststore.ModeDshbot {
		contactsModel = s.ContactsModel(ep)
	}
	botDeleteModel := s.BotDeleteModel(ep, dshbotSelectedID)
	botDuplicateModel := s.BotDuplicateModel(ep, dshbotSelectedID)
	roomEditRendered := asstSnap.Mode == assiststore.ModeDshbot
	roomEditModel := s.RoomEditModel(ep, dshbotSelectedID)
	roomRuntimeRendered := asstSnap.Mode == assiststore.ModeDshbot
	roomRuntimeModel := s.RoomRuntimeModel(ep, dshbotSelectedID)
	whaleSettingsModel := s.WhaleSettingsModel(ep)
	whaleTasksModel := s.WhaleTasksModel(ep)
	proactiveModel := s.ProactiveModel()

	obs := FrameObservation{
		RoomCreateRendered:    roomCreateRendered,
		RoomEditRendered:      roomEditRendered,
		RoomRuntimeRendered:   roomRuntimeRendered,
		NotificationsRendered: roomCreateRendered,
	}

	openWeb := func(plugin, actLabel string) {
		s.SuspendViews()
		s.whaleSettings.Suspend()
		s.mu.Lock()
		s.conversationGeneration++
		s.open = false
		s.draft = assistantDraft
		if actLabel != "" {
			s.lastAct = actLabel
		}
		s.mu.Unlock()
		host.MarkWebViewOpen()
		host.OpenWebViewHostPlugin(origin, plugin)
		if win != nil {
			host.WindowUpdate(win, func() {})
		}
	}

	m := assistantui.Model{
		RoutineCreate:         routineCreateModel,
		RoomCreate:            roomCreateModel,
		BotCreate:             botCreateModel,
		BotVisibility:         botVisibilityModel,
		Contacts:              contactsModel,
		BotDelete:             botDeleteModel,
		BotDuplicate:          botDuplicateModel,
		RoomEdit:              roomEditModel,
		RoomRuntime:           roomRuntimeModel,
		Routines:              routineModel,
		WhaleSettings:         whaleSettingsModel,
		WhaleTasks:            whaleTasksModel,
		Proactive:             proactiveModel,
		Tasks:                 taskModel,
		ThemeID:               opts.ThemeID,
		SystemDark:            opts.SystemDark,
		Mode:                  asstSnap.Mode,
		WhaleEnabled:          whaleOn,
		DshbotEnabled:         dshbotOn,
		WhaleVendorPresent:    whaleVendor,
		DshbotVendorPresent:   dshbotVendor,
		HarnessReady:          true,
		Busy:                  asstSnap.Busy || pluginBusy || taskModel.State.Busy || whaleSettingsModel.State.Busy || routineModel.State.Busy || routineCreateModel.State.Busy || roomCreateModel.State.Busy || botCreateModel.State.Busy || botVisibilityModel.State.Busy || contactsModel.State.Busy || botDeleteModel.State.Busy || botDuplicateModel.State.Busy || roomEditModel.State.Busy || roomRuntimeModel.State.Busy,
		LastWebViewURL:        opts.LastWebViewURL,
		LastNativeAction:      lastNative,
		LastErr:               asstSnap.LastErr,
		SessionID:             asstSnap.SessionID,
		History:               history,
		Draft:                 &assistantDraft,
		PluginRPCBusy:         pluginBusy,
		PluginRPCDetail:       pluginDetail,
		DshbotRoster:          dshbotRows,
		DshbotRosterFetched:   dshbotRosterAt,
		BotNotificationStatus: s.BotNotificationStatusText(),
		DshbotSelectedID:      dshbotSelectedID,
		OnDshbotSelect: func(botID string) {
			s.SuspendViews()
			asst.SelectDshbot(botID, origin)
		},
		OnSetMode: func(mode assiststore.Mode) {
			s.SuspendViews()
			s.whaleSettings.Suspend()
			if asst != nil {
				asst.SetMode(mode)
			}
			if win != nil {
				host.WindowUpdate(win, func() {})
			}
		},
		OnOpenModeWeb: func() {
			plugin := PluginWhale
			if assiststore.NormalizeMode(asstSnap.Mode) == assiststore.ModeDshbot {
				plugin = PluginDshbot
			}
			openWeb(plugin, "")
		},
		OnOpenSession: func(_ string) {
			target := asst.Snapshot()
			if target.Mode == assiststore.ModeDshbot && target.DshbotOrigin != ep.Origin {
				asst.SetLastAction(i18n.T(i18n.KeyCmdSelectCurrentRosterBot), i18n.T(i18n.KeyCmdComposerHostChanged))
				return
			}
			s.OpenConversation(ep, target.Mode, target.DshbotID, "")
		},
		OnClose: func() {
			s.whaleSettings.Suspend()
			s.SuspendViews()
			s.mu.Lock()
			s.conversationGeneration++
			s.open = false
			s.draft = assistantDraft
			s.mu.Unlock()
			if win != nil {
				host.WindowUpdate(win, func() {})
			}
		},
		OnToggleWhale: func(enabled bool) {
			s.whaleSettings.Suspend()
			host.SetAssistantFlag(&enabled, nil)
		},
		OnToggleDshbot: func(enabled bool) {
			s.SuspendViews()
			host.SetAssistantFlag(nil, &enabled)
		},
		OnOpenWhaleWeb: func() {
			openWeb(PluginWhale, i18n.T(i18n.KeyAssistantOpenWhaleWeb))
		},
		OnOpenDshbotWeb: func() {
			openWeb(PluginDshbot, i18n.T(i18n.KeyAssistantOpenDshbotWeb))
		},
		OnNativePrompt: func(text string) {
			s.HandleNativePrompt(origin, ep, text)
			if win != nil {
				host.WindowUpdate(win, func() {})
			}
		},
		OnWhaleEnsure: func() {
			s.OpenConversation(ep, assiststore.ModeWhale, "", "")
		},
		OnWhaleCatalog: func() {
			s.RunPluginRPC(win, ep, "catalog", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
				out, err := c.WhaleCatalog(ctx)
				return string(out), err
			})
		},
		OnWhalePetChat: func(text string) {
			s.HandleNativePrompt(origin, ep, text)
		},
		OnWhalePetState: func() {
			s.RunPluginRPC(win, ep, "pet/state", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
				out, err := c.WhalePetState(ctx, false)
				return string(out), err
			})
		},
		OnWhalePetLook: func() {
			paths, err := host.OpenFileDialog(win, FileDialogOptions{
				Title: i18n.T(i18n.KeyAssistantWhalePetLook),
			})
			if err != nil {
				s.mu.Lock()
				s.pluginDetail = "pet/look: " + err.Error()
				s.lastAct = s.pluginDetail
				s.mu.Unlock()
				if win != nil {
					host.WindowUpdate(win, func() {})
				}
				return
			}
			if len(paths) == 0 {
				return
			}
			imgPath := paths[0]
			s.RunPluginRPC(win, ep, "pet/look", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
				st, err := c.WhalePetStateTyped(ctx, false)
				if err != nil {
					return "", err
				}
				if st.Model == nil || strings.TrimSpace(st.Model.Provider) == "" || strings.TrimSpace(st.Model.Model) == "" {
					return "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdWhaleVisionModelMissing))
				}
				raw, err := os.ReadFile(imgPath)
				if err != nil {
					return "", err
				}
				if len(raw) == 0 || len(raw) > 4<<20 {
					return "", fmt.Errorf("%s", i18n.T(i18n.KeyCmdWhaleLookImageLimit))
				}
				b64 := base64.StdEncoding.EncodeToString(raw)
				out, err := c.WhalePetLook(ctx, st.Model.Provider, st.Model.Model, b64)
				return string(out), err
			})
		},
		OnDshbotDescribe: func() {
			s.RunPluginRPC(win, ep, "describe", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
				out, err := c.DshbotDescribe(ctx)
				return string(out), err
			})
		},
		OnDshbotCatalog: func() {
			s.RunPluginRPC(win, ep, "catalog/items", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
				out, err := c.DshbotCatalog(ctx)
				return string(out), err
			})
		},
		OnDshbotActivity: func() {
			s.RunPluginRPC(win, ep, "bot/activity", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
				act, err := c.DshbotActivityTyped(ctx, nil)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf(i18n.T(i18n.KeyCmdActivityEntryCount), len(act.Entries)), nil
			})
		},
		OnDshbotRefreshRoster: func() {
			label := i18n.T(i18n.KeyAssistantDshbotRosterRefresh)
			s.RunPluginRPC(win, ep, label, func(ctx context.Context, c *pluginrpc.Client) (string, error) {
				res, err := c.DshbotDescribeTyped(ctx, "")
				if err != nil {
					return "", err
				}
				if dr := s.DshbotRoster(); dr != nil {
					dr.SetFromDescribeForHost(res, false, origin)
					if act, aerr := c.DshbotActivityTyped(ctx, nil); aerr == nil {
						dr.MergeActivity(act)
					}
					for _, row := range dr.Snapshot().Rows {
						if row.Kind != "room" {
							continue
						}
						if rt, rerr := c.DshbotGroupRuntime(ctx, row.ID); rerr == nil {
							dr.ApplyGroupRuntime(rt)
						}
					}
				}
				rows := pluginrpc.RosterRowsFromDescribe(res, false)
				return fmt.Sprintf("%d", len(rows)), nil
			})
		},
		OnDshbotOpenRosterSession: func(botID string) {
			s.SuspendViews()
			s.OpenDshbot(ep, botID)
		},
	}

	commit := func() {
		s.SetDraft(assistantDraft)
	}
	return m, obs, commit
}
