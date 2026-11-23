package assistantctl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/state"
	assiststore "whaleisle/internal/store/assistant"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/i18n"
)

const whaleNavRefetch = 30 * time.Second

func whaleTaskText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func (s *Service) cachedWhaleDigestLocked(client *dsh.Client, ep harness.Endpoint, generation int) string {
	if client == nil {
		return ""
	}
	if s.whaleOwnerKey.client == client && s.whaleOwnerKey.origin == ep.Origin && s.whaleOwnerKey.cookie == ep.Cookie && s.whaleOwnerKey.generation == generation && s.whaleOwnerKey.digest != "" {
		return s.whaleOwnerKey.digest
	}
	// Do not expose the authentication cookie or fingerprint in the UI/logs.
	digest := endpointScope(clientBinding(client), ep.Origin, ep.Cookie, fmt.Sprintf("%d", generation))
	s.whaleOwnerKey = whaleKeyCache{
		client:     client,
		origin:     ep.Origin,
		cookie:     ep.Cookie,
		generation: generation,
		digest:     digest,
	}
	return digest
}

func (s *Service) whaleSettingsOwner(ep harness.Endpoint) (*dsh.Client, string) {
	host := s.safeHost()
	petSettings := host.PetSettingsActive()
	readyEndpoint, bootState, generation := host.BootEndpoint()
	cfgStore := host.Config()
	client := host.CurrentClient()
	asst := s.AssistantStore()

	enabled := bootState == harness.Ready && cfgStore != nil && cfgStore.Get().WhaleAssistantEnabled
	if !enabled || readyEndpoint.Origin != ep.Origin || readyEndpoint.Cookie != ep.Cookie {
		return nil, ""
	}
	if client == nil || client.Endpoint.Origin != ep.Origin || client.Endpoint.Cookie != ep.Cookie {
		return nil, ""
	}
	if !petSettings && (asst == nil || asst.Snapshot().Value.Mode != assiststore.ModeWhale) {
		return nil, ""
	}
	s.mu.Lock()
	active := (s.open && !s.proactivePanel.open) || petSettings
	if !active {
		s.mu.Unlock()
		return nil, ""
	}
	digest := s.cachedWhaleDigestLocked(client, ep, generation)
	s.mu.Unlock()
	return client, digest
}

// WhalePreferencesModel builds the Whale model/IM/skill preferences editor model.
func (s *Service) WhalePreferencesModel(ep harness.Endpoint) assistantui.WhalePreferencesModel {
	if s == nil {
		return assistantui.WhalePreferencesModel{}
	}
	host := s.safeHost()
	client, hostKey := s.whaleSettingsOwner(ep)
	editor := &s.whaleSettings.Preferences
	if hostKey != "" {
		editor.Bind(hostKey)
	}
	current := func() bool {
		now, key := s.whaleSettingsOwner(ep)
		return client != nil && now == client && hostKey != "" && key == hostKey
	}
	rpc := host.PluginRPCClient(ep)
	run := func(cmd *assiststore.WhalePreferencesCommand) {
		if !current() {
			return
		}
		release := func() {}
		if cmd != nil {
			var allowed bool
			release, allowed = s.whaleSettings.BeginMutation(hostKey)
			if !allowed {
				return
			}
		}
		ticket, ok := editor.Begin(hostKey, cmd != nil)
		if !ok {
			release()
			return
		}
		s.RefreshWhaleSettingsUI()
		go func() {
			defer func() { release(); s.RefreshWhaleSettingsUI() }()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var value pluginrpc.WhalePreferences
			var catalog *api.ModelCatalog
			var modelErr *state.Fault
			execute := func() error {
				if !current() || !editor.Current(ticket) {
					return fmt.Errorf("Whale editor changed")
				}
				if cmd != nil {
					var err error
					switch cmd.Action {
					case "model":
						err = rpc.WhaleSaveModel(ctx, cmd.Route)
					case "im":
						err = rpc.WhaleSaveIMDefault(ctx, cmd.IMDefault)
					case "skill":
						err = rpc.WhaleToggleSkill(ctx, cmd.Skill, cmd.Enabled)
					default:
						return fmt.Errorf("unknown Whale preference")
					}
					if err != nil {
						return err
					}
				}
				if !current() || !editor.Current(ticket) {
					return fmt.Errorf("Whale editor changed")
				}
				var err error
				value, err = rpc.WhalePreferences(ctx)
				if err != nil {
					return err
				}
				if cmd == nil {
					catalog = &api.ModelCatalog{}
					if !current() || !editor.Current(ticket) {
						return fmt.Errorf("Whale editor changed")
					}
					if e := client.Call(ctx, "session/modelCatalog", map[string]any{}, catalog); e != nil {
						modelErr = state.FaultOf(e)
					}
				}
				return nil
			}
			err := execute()
			if current() {
				editor.Finish(ticket, value, catalog, modelErr, cmd, err)
			}
		}()
	}
	state := editor.Snapshot().Value
	if hostKey == "" {
		state = assiststore.WhalePreferencesSnapshot{}
	}
	return assistantui.WhalePreferencesModel{
		State:     state,
		OnRefresh: func() { run(nil) },
		OnEdit: func(route pluginrpc.WhaleModelRoute, im bool) {
			if current() {
				editor.Edit(hostKey, route, im)
				s.RefreshWhaleSettingsUI()
			}
		},
		OnSave: func(action string) {
			if !current() {
				return
			}
			draft := editor.Snapshot().Value
			run(&assiststore.WhalePreferencesCommand{Action: action, Route: draft.Route, IMDefault: draft.IMDefault})
		},
		OnToggleSkill: func(name string, enabled bool) {
			run(&assiststore.WhalePreferencesCommand{Action: "skill", Skill: name, Enabled: enabled})
		},
	}
}

// WhaleSettingsModel builds the Whale identity/persona/memory settings editor model.
func (s *Service) WhaleSettingsModel(ep harness.Endpoint) assistantui.WhaleSettingsModel {
	if s == nil {
		return assistantui.WhaleSettingsModel{}
	}
	host := s.safeHost()
	client, hostKey := s.whaleSettingsOwner(ep)
	// A mode switch hides this form but retains its draft for the same Host.
	// Explicit Close/disable handlers revoke in-flight tickets separately.
	if hostKey != "" {
		s.whaleSettings.Bind(hostKey)
	}
	current := func() bool {
		now, key := s.whaleSettingsOwner(ep)
		return client != nil && now == client && hostKey != "" && key == hostKey
	}
	cli := host.PluginRPCClient(ep)
	write := func(action string) {
		if !current() || s.whaleSettings.Preferences.Snapshot().Value.Busy {
			return
		}
		release, allowed := s.whaleSettings.BeginMutation(hostKey)
		if !allowed {
			return
		}
		ticket, command, ok := s.whaleSettings.BeginWrite(hostKey, action)
		if !ok {
			release()
			s.RefreshWhaleSettingsUI()
			return
		}
		s.RefreshWhaleSettingsUI()
		go func() {
			defer func() { release(); s.RefreshWhaleSettingsUI() }()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			var actual *pluginrpc.WhaleSettings
			accepted := false
			execute := func() error {
				if !current() || !s.whaleSettings.Current(ticket) {
					return fmt.Errorf("Host or Whale editor changed before submission")
				}
				before, err := cli.WhaleSettings(ctx)
				if err != nil {
					return err
				}
				if err = command.Check(before); err != nil {
					return err
				}
				if !current() || !s.whaleSettings.Current(ticket) {
					return fmt.Errorf("Host or Whale editor changed before submission")
				}
				// Exactly one user-requested mutation; never ensure/open a session, prompt
				// a model, retry a write, or use dshbot task/catalog RPCs here.
				switch command.Action {
				case "memory":
					err = cli.WhaleReplaceMemory(ctx, command.Value.Memory)
				case "clear":
					err = cli.WhaleClearMemory(ctx)
				case "identity":
					_, err = cli.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{Name: &command.Value.Name, UserTitle: &command.Value.UserTitle})
				case "name":
					_, err = cli.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{Name: &command.Value.Name})
				case "title":
					_, err = cli.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{UserTitle: &command.Value.UserTitle})
				case "persona":
					_, err = cli.WhaleUpdateSettings(ctx, pluginrpc.WhaleSettingsPatch{PersonaText: &command.Value.PersonaText})
				}
				if err != nil {
					return err
				}
				accepted = true
				if !current() || !s.whaleSettings.Current(ticket) {
					return fmt.Errorf("Host changed after confirmation; refresh to verify")
				}
				read, err := cli.WhaleSettings(ctx)
				if err == nil {
					actual = &read
				}
				return err
			}
			err := execute()
			if !current() {
				return
			}
			if s.whaleSettings.FinishWrite(ticket, command, actual, accepted, err) {
				if accepted && err == nil && (command.Action == "identity" || command.Action == "name") {
					sessions := host.Sessions()
					s.mu.Lock()
					if actual != nil {
						s.whaleNavName = strings.TrimSpace(actual.Name)
						s.whaleNavFetchedAt = time.Now()
					} else {
						// Read-back failed: expire the published name so the
						// observer refetches on the next frame.
						s.whaleNavFetchedAt = time.Time{}
					}
					s.mu.Unlock()
					if sessions != nil {
						sessions.RefreshList(ctx)
					}
				}
			}
		}()
	}
	preferences := s.WhalePreferencesModel(ep)
	state := s.whaleSettings.Snapshot().Value
	state.Busy = state.Busy || preferences.State.Busy
	if hostKey == "" {
		state = assiststore.WhaleSettingsSnapshot{}
	}
	return assistantui.WhaleSettingsModel{
		Preferences: preferences,
		State:       state,
		OnRefresh: func() {
			if !current() {
				return
			}
			ticket, ok := s.whaleSettings.BeginLoad(hostKey)
			if !ok {
				return
			}
			s.RefreshWhaleSettingsUI()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if !current() || !s.whaleSettings.Current(ticket) {
					return
				}
				value, err := cli.WhaleSettings(ctx)
				if current() && s.whaleSettings.FinishLoad(ticket, value, err) {
					s.RefreshWhaleSettingsUI()
				}
			}()
		},
		OnEdit: func(field, value string) {
			if current() {
				s.whaleSettings.Edit(hostKey, field, value)
				s.RefreshWhaleSettingsUI()
			}
		},
		OnRestore: func(field string) {
			if current() {
				s.whaleSettings.Restore(hostKey, field)
				s.RefreshWhaleSettingsUI()
			}
		},
		OnSave: write,
		OnReviewClear: func() {
			if current() {
				s.whaleSettings.ReviewClear(hostKey)
				s.RefreshWhaleSettingsUI()
			}
		},
		OnCancelClear: func() {
			if current() {
				s.whaleSettings.CancelClear(hostKey)
				s.RefreshWhaleSettingsUI()
			}
		},
		OnConfirmClear: func() { write("clear") },
	}
}

// ObserveWhaleSettings reconciles Whale settings/task bindings and the sidebar
// Whale navigation label on every frame.
func (s *Service) ObserveWhaleSettings() {
	if s == nil {
		return
	}
	ep, _, _ := s.safeHost().BootEndpoint()
	_, hostKey := s.whaleSettingsOwner(ep)
	if hostKey == "" {
		s.whaleSettings.Suspend()
		s.whaleTasks.Suspend()
	} else {
		s.whaleSettings.Bind(hostKey)
	}
	s.ObserveWhaleNavLabel(ep)
}

// ObserveWhaleNavLabel fetches and refreshes the Whale sidebar display name
// periodically without performing I/O on the draw path.
func (s *Service) ObserveWhaleNavLabel(ep harness.Endpoint) {
	if s == nil {
		return
	}
	host := s.safeHost()
	client := host.CurrentClient()
	endpoint, bootState, generation := host.BootEndpoint()
	cfgStore := host.Config()
	lifecycleCtx := host.LifecycleContext()
	enabled := bootState == harness.Ready && cfgStore != nil && cfgStore.Get().WhaleAssistantEnabled

	s.mu.Lock()
	key := ""
	if enabled && ep.Origin != "" && client != nil && client.Endpoint.Origin == ep.Origin && client.Endpoint.Cookie == ep.Cookie && endpoint.Origin == ep.Origin && endpoint.Cookie == ep.Cookie {
		key = s.cachedWhaleDigestLocked(client, ep, generation)
	}
	if s.whaleNavKey != key {
		s.whaleNavKey = key
		s.whaleNavName = ""
		s.whaleNavBusy = false
		s.whaleNavFetchedAt = time.Time{}
		s.whaleNavSerial++
	}
	if key == "" || s.whaleNavBusy || time.Since(s.whaleNavFetchedAt) < whaleNavRefetch {
		s.mu.Unlock()
		return
	}
	s.whaleNavBusy = true
	serial := s.whaleNavSerial
	s.mu.Unlock()

	rpc := host.PluginRPCClient(ep)
	go func() {
		ctx, cancel := context.WithTimeout(lifecycleCtx, 15*time.Second)
		defer cancel()
		settings, err := rpc.WhaleSettings(ctx)
		s.mu.Lock()
		if s.whaleNavKey == key && s.whaleNavSerial == serial {
			s.whaleNavBusy = false
			if err == nil {
				s.whaleNavName = strings.TrimSpace(settings.Name)
				s.whaleNavFetchedAt = time.Now()
			} else {
				// Preserve the last known navigation label across transient RPC errors
				// and back off briefly before retrying.
				s.whaleNavFetchedAt = time.Now().Add(-whaleNavRefetch + 5*time.Second)
			}
		}
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	}()
}

// WhaleTasksModel builds the Whale background reading-task control model.
func (s *Service) WhaleTasksModel(ep harness.Endpoint) assistantui.WhaleTasksModel {
	if s == nil {
		return assistantui.WhaleTasksModel{}
	}
	host := s.safeHost()
	client, hostKey := s.whaleSettingsOwner(ep)
	if hostKey != "" {
		s.whaleTasks.Bind(hostKey)
	}
	current := func() bool {
		now, key := s.whaleSettingsOwner(ep)
		return client != nil && now == client && hostKey != "" && key == hostKey
	}
	rpc := host.PluginRPCClient(ep)
	var execute func(string, string)
	execute = func(action, id string) {
		if !current() {
			return
		}
		ticket, draft, ok := s.whaleTasks.Begin(hostKey, action)
		if !ok {
			return
		}
		s.RefreshWhaleSettingsUI()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			accepted := false
			notice := ""
			var catalog *pluginrpc.WhaleTasks
			err := func() error {
				if !current() || !s.whaleTasks.Current(ticket) {
					return errors.New("task panel or Host changed")
				}
				switch action {
				case "start":
					if _, err := rpc.WhaleStartTask(ctx, draft.Draft()); err != nil {
						return err
					}
					accepted = true
					notice = whaleTaskText("任务已接收，可离开此面板。", "Task accepted. You can leave this panel.")
				case "stop", "resume":
					if _, err := rpc.WhaleControlTask(ctx, id, action); err != nil {
						return err
					}
					accepted = true
				case "owner", "revoke-owner":
					if err := rpc.WhaleTaskOwner(ctx, strings.TrimSpace(draft.BotID), strings.TrimSpace(draft.Actor), action == "revoke-owner"); err != nil {
						return err
					}
					accepted = true
					notice = whaleTaskText("飞书任务授权已更新。", "Feishu task authorization updated.")
				case "refresh":
				default:
					return errors.New("unknown task action")
				}
				value, err := rpc.WhaleTasks(ctx)
				if err != nil {
					return err
				}
				catalog = &value
				return nil
			}()
			if !current() {
				s.whaleTasks.Cancel(ticket)
				s.RefreshWhaleSettingsUI()
				return
			}
			if !s.whaleTasks.Finish(ticket, action, catalog, accepted, notice, err) {
				return
			}
			s.RefreshWhaleSettingsUI()
			if err == nil && catalog != nil {
				poll := false
				for _, task := range catalog.Tasks {
					poll = poll || task.Active()
					for _, effect := range task.Effects {
						poll = poll || effect.State == "pending" || effect.State == "sending"
					}
				}
				if poll {
					lctx := host.LifecycleContext()
					if lctx.Err() == nil {
						var timer *time.Timer
						var stopWatcher func() bool
						timer = time.AfterFunc(2*time.Second, func() {
							if stopWatcher != nil {
								stopWatcher()
							}
							if lctx.Err() == nil && current() {
								execute("refresh", "")
							}
						})
						stopWatcher = context.AfterFunc(lctx, func() {
							timer.Stop()
						})
					}
				}
			}
		}()
	}
	snap := s.whaleTasks.Snapshot().Value
	if current() && !snap.Loaded && !snap.Busy && snap.Error == nil {
		execute("refresh", "")
		snap = s.whaleTasks.Snapshot().Value
	}
	return assistantui.WhaleTasksModel{
		State:       snap,
		OnRefresh:   func() { execute("refresh", "") },
		OnStart:     func() { execute("start", "") },
		OnControl:   func(id, action string) { execute(action, id) },
		OnAuthorize: func() { execute("owner", "") },
		OnRevoke:    func() { execute("revoke-owner", "") },
		OnEdit: func(field, value string) {
			if current() {
				s.whaleTasks.Edit(hostKey, field, value)
				s.RefreshWhaleSettingsUI()
			}
		},
		OnPickFiles: func() {
			if !current() {
				return
			}
			win := host.Window()
			files, err := host.OpenFileDialog(win, FileDialogOptions{
				Title:    whaleTaskText("选择整理资料", "Choose reading inputs"),
				Multiple: true,
				Filters:  []FileFilter{{Name: whaleTaskText("文本资料", "Text files"), Extensions: []string{"txt", "md", "mdx", "csv", "json", "log", "rst"}}},
			})
			if err != nil {
				s.whaleTasks.SetError(hostKey, err)
				s.RefreshWhaleSettingsUI()
				return
			}
			if len(files) == 0 || !current() {
				return
			}
			s.whaleTasks.Edit(hostKey, "files", strings.Join(files, "\n"))
			s.RefreshWhaleSettingsUI()
		},
		OnOpenSession: func(id string) {
			if !current() {
				return
			}
			found := false
			for _, task := range s.whaleTasks.Snapshot().Value.Catalog.Tasks {
				found = found || task.Execution != nil && task.Execution.SessionID == id
			}
			if !found {
				return
			}
			sessions := host.Sessions()
			s.SetOpen(false)
			host.RevealMainConversation()
			s.whaleTasks.Suspend()
			if sessions != nil {
				sessions.Select(id)
			}
			s.RefreshWhaleSettingsUI()
		},
	}
}

// WhaleNavRefetch is the refresh interval for the Whale sidebar navigation label.
const WhaleNavRefetch = whaleNavRefetch

// WhaleNavStatusSnapshot exposes cached Whale nav label state for unit tests.
type WhaleNavStatusSnapshot struct {
	Name      string
	Busy      bool
	FetchedAt time.Time
}

// WhaleNavStatus returns a snapshot of the cached Whale nav label state.
func (s *Service) WhaleNavStatus() WhaleNavStatusSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return WhaleNavStatusSnapshot{
		Name:      s.whaleNavName,
		Busy:      s.whaleNavBusy,
		FetchedAt: s.whaleNavFetchedAt,
	}
}

// ExpireWhaleNavForTest ages the Whale nav cache so the next ObserveWhaleNavLabel refetches.
func (s *Service) ExpireWhaleNavForTest() {
	s.mu.Lock()
	s.whaleNavFetchedAt = time.Now().Add(-whaleNavRefetch - time.Second)
	s.mu.Unlock()
}

// WhaleDigest computes or returns the cached Host generation digest under Service.mu.
func (s *Service) WhaleDigest(client *dsh.Client, ep harness.Endpoint, generation int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cachedWhaleDigestLocked(client, ep, generation)
}
