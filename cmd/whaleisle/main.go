// Command whaleisle is the Go desktop shell of Whale Isle.
//
// Assembles M1 shell chrome and M2 harness boot: single-instance lock,
// config / credentials, custom title bar, tray, close-to-tray, open-at-login,
// vendored harness prepare/launch via Supervisor, and a native boot page.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/accountlogin"
	"whaleisle/internal/accountlogout"
	"whaleisle/internal/config"
	"whaleisle/internal/credentials"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/desktop/credentialrefs"
	"whaleisle/internal/desktop/deeplink"
	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/desktop/imagepreview"
	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/desktop/importer"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/desktop/providerdiscovery"
	"whaleisle/internal/desktop/providerprofiles"
	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/desktop/taskguard"
	desktopupdate "whaleisle/internal/desktop/update"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/paths"
	"whaleisle/internal/recovery"
	"whaleisle/internal/shortcutprefs"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/startuprec"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/store/hostterminal"
	imstore "whaleisle/internal/store/im"
	"whaleisle/internal/store/interaction"
	marketstore "whaleisle/internal/store/market"
	remotestore "whaleisle/internal/store/remote"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/store/session"
	settingsstore "whaleisle/internal/store/settings"
	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/boot"
	browserpreviewui "whaleisle/internal/ui/browserpreview"
	"whaleisle/internal/ui/chat"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/dirpick"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
	imui "whaleisle/internal/ui/im"
	"whaleisle/internal/ui/launcher"
	marketui "whaleisle/internal/ui/market"
	"whaleisle/internal/ui/planpreview"
	remoteui "whaleisle/internal/ui/remote"
	scheduleui "whaleisle/internal/ui/schedule"
	costui "whaleisle/internal/ui/sessioncost"
	"whaleisle/internal/ui/settings"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
	usageui "whaleisle/internal/ui/usage"
	webviewui "whaleisle/internal/ui/webview"
)

type mainState struct {
	nativePlugins        nativePluginManagerState
	customPanels         customPanelState
	applicationMenu      *desktopApplicationMenu
	wallpaperControls    *settings.WallpaperEditor
	wallpaperVisible     bool
	commandSearchFocus   bool
	appearanceEditor     *theme.Editor
	appearanceVisible    bool
	shortcutEditor       *shortcuts.Editor
	shortcutPreferences  *shortcutprefs.Store
	shortcutRevision     string
	projectWindowHidden  bool
	startupRecovery      startupRecoveryState
	voice                *composerVoiceOperation
	projectFlow          projectFlowState
	resumeStore          *recovery.Store
	resumeError          string
	resumePendingSession string
	resumeWorkspace      string
	mu                   sync.Mutex
	win                  *mygo.Window
	taskRestartPending   bool
	quitting             bool
	phoneRuntime         *phoneremote.Runtime
	phoneDeviceEditor    remoteui.PhoneDeviceEditor
	phoneGatewayEditor   remoteui.GatewayEditor
	phoneGitSession      string
	phoneGitRoot         string
	phoneGitWorkspaces   phoneGitWorkspaceScope
	phoneOp              sync.Mutex
	closeToTray          bool
	desktopTray          *mygo.Tray
	trayHideNotice       trayHideNotice
	trayHideNotification *mygo.Notification
	bridgeAlerts         *bridgeAlertUI
	cfgStore             *config.Store
	viewPreferences      *workspaceViewPreferences
	desktopPet           *pet.Companion
	petEvents            petEventController
	petPersonality       petPersonalityMirror
	navigationFile       *navigationFileRequest
	themeID              string
	sup                  *harness.Supervisor
	bootStatus           harness.Status
	bootLogs             []string

	sessions                                *session.Store
	agentPresetManager                      *session.AgentPresetManager
	creatorTask                             *session.CreatorTask
	interactions                            *interaction.Store
	settings                                *settingsstore.Store
	settingsOpen                            bool
	settingsEmbeddedSection                 settings.SectionID
	settingsRemoteTab                       string
	settingsNSFilter                        string
	settingsRevealSecrets                   bool
	settingsDesktopActive                   bool
	settingsAccountActive                   bool
	accountLogin                            accountlogin.Flow
	accountSignOut                          accountlogout.Flow
	settingsCredentialsActive               bool
	settingsAboutActive                     bool
	settingsAppearanceActive                bool
	settingsInterfaceActive                 bool
	settingsModelsActive                    bool
	settingsMCPActive                       bool
	settingsPluginsActive                   bool
	settingsPluginsLastAction               string
	settingsSkillsActive                    bool
	settingsSkillRoot                       string
	settingsSkillHubTarget                  session.SkillTarget
	settingsSkillHubQuery                   string
	settingsSkillHubHits                    []settings.SkillHubHit
	settingsSkillCreate                     settingsSkillCreateState
	settingsSkillsLastAction                string
	settingsSkillsDetail                    string
	settingsSkillEditSession                string
	settingsSkillInspectVersion             uint64
	settingsSkillEditGroups                 string
	settingsSkillEditWhen                   string
	settingsSkillEditTarget                 session.SkillEditTarget
	settingsSkillEditorSaving               bool
	settingsSkillQuery, settingsSkillSource string
	settingsSkillEditName                   string
	settingsSkillEditDesc                   string
	settingsSkillEditContent                string
	settingsSkillEditModel                  bool
	settingsSkillEditUser                   bool
	settingsSkillEditWritable               bool
	settingsAgentPresetsActive              bool
	settingsModelLabel                      string
	settingsModelSession                    string
	settingsModelDrafts                     map[string]string
	settingsAgentPresetLabel                string
	agentPresetReader                       *session.AgentPresetReader
	agentPresetReaderSource                 *session.Store
	settingsMCPController                   *mcpconfig.Controller
	settingsProviderProfiles                providerprofiles.Controller
	settingsReferencedCredentials           credentialrefs.Controller
	settingsProviderDiscovery               providerdiscovery.Controller
	settingsPluginInventory                 plugininventory.Controller
	remoteMachines                          *remotestore.Store
	remoteFiles                             *remotestore.Files
	remoteFileEditor                        remoteui.FileEditor
	remoteMachineEditor                     remoteui.MachineEditor
	sshTunnel                               *sshremote.TunnelSession
	sshHost                                 string
	sshUser                                 string
	sshPort                                 string
	sshIdentity                             string
	sshForward                              string
	sshTunnelMsg                            string
	sshConnectEpoch                         uint64
	sshConnectCancel                        context.CancelFunc
	imDraft                                 string
	assistant                               *assiststore.Store
	dshbotRoster                            *dshbotstore.Store
	assistantTasks                          dshbotstore.TaskControls
	assistantRoutines                       dshbotstore.RoutineControls
	botNotifications                        *botNotificationRun
	botNotificationMessage                  string
	botNotificationShowing                  bool
	botNotificationDenied                   bool
	botNotificationReadSlot                 chan struct{}
	assistantRoutineCreator                 dshbotstore.RoutineCreator
	assistantRoomCreator                    dshbotstore.RoomCreator
	assistantBotVisibility                  dshbotstore.BotVisibilityManager
	assistantContacts                       dshbotstore.ContactControls
	assistantBotCreator                     dshbotstore.BotCreator
	assistantBotDeleter                     dshbotstore.BotDeleter
	assistantBotDuplicator                  dshbotstore.BotDuplicator
	assistantRoomEditor                     dshbotstore.RoomEditor
	assistantRoomRuntime                    dshbotstore.RoomRuntimeReader
	assistantGroupLive                      assistantGroupLiveState
	whaleSettings                           assiststore.WhaleSettingsEditor
	desktopDraft                            desktopConfigDraft
	desktopSaveErr                          string
	desktopLastAction                       string
	credsDraft                              credentialsDraft
	credsSaveErr                            string
	credsLastAction                         string
	credsHarnessNote                        string
	analyticsEnabled                        *bool
	credsShowSecrets                        bool
	usage                                   *usagestore.Store
	accountReturn                           accountReturnState
	accountNotice                           accountBonusNoticeState
	accountStatusNotices                    accountStatusNoticeState
	accountQuotaNotice                      accountQuotaNoticeState
	im                                      *imstore.Store
	market                                  *marketstore.Store
	schedule                                *schedstore.Store
	usageOpen                               bool
	imOpen                                  bool
	marketOpen                              bool
	marketManager                           bool
	scheduleOpen                            bool
	scheduleHistoryText                     string
	remoteOpen                              bool
	marketBatchReceipts                     map[string]string
	marketBatch                             []marketstore.InstallTarget
	marketActive                            *marketAttempt
	marketWorker                            *marketAttempt
	marketApproval                          *marketAttempt
	marketCatalogTarget                     *marketstore.InstallTarget
	marketInstallSpec                       string
	marketInstallBusy                       bool
	marketInstallRequestID                  string
	marketInstallMsg                        string
	marketInstallLog                        string
	marketActivation                        marketstore.Activation
	marketNeedsAllowBuilds                  bool
	marketAllowBuilds                       []string
	marketIncompatible                      []api.ChangeResultErrorIncompatibleItem
	hasRemoteToken                          bool
	hasRemoteRelayToken                     bool
	remoteDeviceCount                       int
	remoteDeviceNames                       []string
	phoneActionMsg                          string
	sshProbeLine                            string
	webviewOpen                             bool
	browserPreviewOpen                      bool
	officePreview                           officePreviewState
	pdfPreviewRequest                       uint64
	mediaPreviewRequest                     uint64
	htmlDocumentRequest                     uint64
	browserController                       *browserpreviewui.Controller
	browserPreviewDraft                     string
	browserPreviewNavigateErr               string
	webviewHost                             *desktopwebview.Host
	settingsOverlayHost                     *desktopwebview.Host
	webviewHostErr                          string
	lastWebViewURL                          string
	lastWebViewOrigin                       string
	pluginViewResumeAfterRestart            bool
	pluginViewHostGeneration                int
	assistantOpen                           bool
	assistantConversationGeneration         uint64
	assistantDraft                          string
	assistantLastAct                        string
	assistantPluginBusy                     bool
	assistantPluginDetail                   string
	launcherOpen                            bool
	toolsMenuOpen                           bool
	filesPaneOpen                           bool // Electron tools → files/diff right dock
	launcherTab                             string
	launcherWorkspaceDraft                  string
	launcherImport                          string
	launcherImportJournal                   []string
	launcherImportBusy                      bool
	launcherImportCancel                    context.CancelFunc
	launcherUpdate                          string
	launcherUpdateCheckBusy                 bool
	updateCheckPending                      bool
	launcherPrefsError                      string
	nativeReleases                          nativeReleaseState
	launcherUpdateAvailable                 bool
	launcherUpdateSynced                    bool
	launcherNote                            string
	launcherTriedStart                      bool
	launcherStartBusy                       bool
	launcherStartPreparing                  bool
	launcherStartCancel                     context.CancelFunc
	launcherStartFailed                     bool
	launcherStartErr                        string
	launcherPrefsSeeded                     bool
	launcherQuitAfterStart                  bool
	launcherAutoStartDesktop                bool
	launcherAskOnUpdate                     bool
	launcherDownloadRoute                   string
	launcherImportOverwrite                 bool
	launcherImportAttachments               bool
	launcherImportCategories                map[string]bool
	launcherImportCatalog                   *importer.Catalog
	launcherComponents                      []desktopcomponents.Row
	launcherComponentsFresh                 bool
	launcherComponentsRefreshBusy           bool
	launcherComponentStatus                 string
	launcherHarnessVersion                  string
	quitRequestPending                      bool
	quitCleanupFinished                     bool
	launcherRoutePopOpen                    bool
	workspaceMenuOpen                       bool
	workspacePickerAnchor                   ui.Handle
	workspaceRenameDraft                    string
	pendingDeeplink                         deeplink.Target
	settingsDrafts                          []settingsDraftSlot
	workspaceSessions                       workspaceSessionLayouts
	layout                                  frame.LayoutSizes
	chatList                                ui.ListState
	draft                                   string
	editMessageSeq                          float64
	feedbackCommentMsgID                    string
	feedbackCommentDraft                    string
	modelPick                               string
	permissionPick                          string
	agentPresetPick                         string
	thinkingExpanded                        map[string]bool
	renameSessionID                         string
	renameDraft                             string
	sessionSearch                           string

	gitBranch      string
	gitDirty       bool
	gitMsg         string
	gitBusy        bool
	gitProgress    *desktopgit.Progress
	gitCommitOpen  bool
	gitCommitDraft string
	gitCheckedRoot string
	gitCheckedAt   time.Time
	gitPROpen      bool
	gitPRBusy      bool
	gitPRReport    desktopgit.PRReport
	gitPRRoot      string
	gitPRAt        time.Time

	fileTabs         files.Tabs
	gitWorkbench     diffsurface.Workbench
	fileExpanded     map[string]bool
	selectedFile     string
	fileFilter       string
	filePreview      string
	filePreviewErr   string
	fileStatNote     string
	filePreviewList  ui.ListState
	fileEditing      bool
	fileEditDraft    string
	fileEditStatus   string
	fileEditViewMode files.EditViewMode
	fileEditSplit    float32

	schedulePanel      scheduleui.Panel
	scheduleEditID     string
	scheduleEditTitle  string
	scheduleEditPrompt string

	rightPaneTab          string
	agentsPromptDraft     string
	goalDraft             string
	fileOpenWithApps      []files.OpenWithApp
	fileOpenWithFor       string
	diffPath              string
	diffText              string
	diffErr               string
	planBoardOpen         bool
	todoPanelCollapsed    bool
	chatPaneTab           string // "chat" | "trajectory"
	trajSearch            string
	trajSelectedID        string
	trajActualDur         bool
	trajTurnsFold         bool
	trajCallsFold         bool
	trajDetailTab         string
	imageLightboxPath     string
	dirPickOpen           bool
	dirPickBrowser        dirpick.BrowserState
	dirPickLoading        bool
	dirPickRequest        uint64
	dirPickCancel         context.CancelFunc
	dirPickHostGeneration uint64
	dirPickMode           string // "workspace" | "mention"
	dirPickPath           string
	dirPickListing        api.DirectoryListing
	dirPickErr            string
	dirPickNewName        string

	composerResize        composer.ResizeState
	sessionCostState      costui.State
	composerSuggestSel    int
	composerSuggest       ui.ListState
	composerPathsRoot     string
	composerPaths         []string
	composerSlash         []composer.SlashCommand
	composerSlashKey      string
	composerSkillsOpen    bool
	composerSkillsSel     int
	composerSkillsList    ui.ListState
	composerSkillsFilter  string
	skillCreateName       string
	composerSkillsSession string
	composerChipEdit      composer.ChipEdit
	composerTextInput     *composer.DraftClient
	composerDrafts        map[string]*composer.DraftState

	terminalPanel       *termui.Panel
	terminalSessions    termui.Sessions
	hostTerminals       hostterminal.Store
	hostTerminalUI      termui.HostState
	terminalShellsNote  string
	terminalVisible     bool
	terminalSavedBottom float32

	dshMu       sync.Mutex
	dshCancel   context.CancelFunc
	dshEndpoint string
	dshClient   *dsh.Client
	dshMux      *dsh.Mux
	dshMuxReady bool
	dshMuxLost  bool
}

var state = mainState{fileEditViewMode: files.EditViewEdit}

// desktopControlOps exposes the native operations behind the authenticated
// install-control desktop routes. Each callback uses an existing Go store or
// dsh RPC; unsupported operations stay explicit 501 responses in installctrl.
func desktopControlOps() installctrl.DesktopOps {
	return installctrl.DesktopOps{
		State: func(context.Context) (any, error) {
			state.mu.Lock()
			cfgStore := state.cfgStore
			status := state.bootStatus
			market := state.market
			state.mu.Unlock()
			var cfg config.Config
			if cfgStore != nil {
				cfg = cfgStore.Get()
			}
			var plugins []string
			if market != nil {
				for _, row := range market.Snapshot().Rows {
					if strings.TrimSpace(row.Label) != "" {
						plugins = append(plugins, row.Label)
					}
				}
			}
			public := map[string]any{
				"workspace":                 cfg.Workspace,
				"host":                      cfg.Host,
				"port":                      cfg.Port,
				"theme":                     cfg.Theme,
				"locale":                    cfg.Locale,
				"closeToTray":               cfg.CloseToTray,
				"openAtLogin":               cfg.OpenAtLogin,
				"dshbotEnabled":             cfg.DshbotEnabled,
				"whaleAssistantEnabled":     cfg.WhaleAssistantEnabled,
				"remoteWorkspaceEnabled":    cfg.RemoteWorkspaceEnabled,
				"harnessAutoRestart":        cfg.HarnessAutoRestart,
				"harnessRestartMaxAttempts": cfg.HarnessRestartMaxAttempts,
				"harnessRestartBaseDelayMs": cfg.HarnessRestartBaseDelayMs,
			}
			return map[string]any{
				"ok": true, "version": buildVersion, "kernel": status.State.String(),
				"plugins": plugins, "disabledPlugins": append([]string(nil), cfg.DisabledPlugins...),
				"config": public,
			}, nil
		},
		ListMarketplace: func(ctx context.Context, opt installctrl.MarketplaceOptions) (any, error) {
			state.mu.Lock()
			market := state.market
			state.mu.Unlock()
			if market == nil {
				return map[string]any{"ok": true, "plugins": []any{}}, nil
			}
			if opt.Refresh {
				market.Refresh(ctx)
			}
			query := strings.ToLower(strings.TrimSpace(opt.Query))
			rows := market.Snapshot().Rows
			plugins := make([]map[string]any, 0, len(rows))
			for _, row := range rows {
				if query != "" && !strings.Contains(strings.ToLower(row.Label+" "+row.Detail), query) {
					continue
				}
				plugins = append(plugins, map[string]any{"id": row.Label, "name": row.Label, "summary": row.Detail})
			}
			return map[string]any{"ok": true, "plugins": plugins}, nil
		},
		ApplyConfig: func(_ context.Context, patch map[string]any) (any, error) {
			state.mu.Lock()
			store := state.cfgStore
			sup := state.sup
			state.mu.Unlock()
			if store == nil {
				return nil, errors.New("desktop config store unavailable")
			}
			for key, value := range patch {
				switch key {
				case "theme", "locale":
					if _, ok := value.(string); !ok {
						return nil, fmt.Errorf("%s must be a string", key)
					}
				case "closeToTray", "openAtLogin", "dshbotEnabled", "whaleAssistantEnabled", "remoteWorkspaceEnabled", "harnessAutoRestart":
					if _, ok := value.(bool); !ok {
						return nil, fmt.Errorf("%s must be a boolean", key)
					}
				case "harnessRestartMaxAttempts", "harnessRestartBaseDelayMs":
					if _, ok := value.(float64); !ok {
						return nil, fmt.Errorf("%s must be a number", key)
					}
				default:
					return nil, fmt.Errorf("config field is not desktop-writable: %s", key)
				}
			}
			restart := false
			if err := store.Update(func(c *config.Config) {
				for key, value := range patch {
					switch key {
					case "theme":
						c.Theme = value.(string)
					case "locale":
						c.Locale = value.(string)
					case "closeToTray":
						c.CloseToTray = value.(bool)
					case "openAtLogin":
						c.OpenAtLogin = value.(bool)
					case "dshbotEnabled":
						c.DshbotEnabled, restart = value.(bool), true
					case "whaleAssistantEnabled":
						c.WhaleAssistantEnabled, restart = value.(bool), true
					case "remoteWorkspaceEnabled":
						c.RemoteWorkspaceEnabled, restart = value.(bool), true
					case "harnessAutoRestart":
						c.HarnessAutoRestart = value.(bool)
					case "harnessRestartMaxAttempts":
						c.HarnessRestartMaxAttempts = int(value.(float64))
					case "harnessRestartBaseDelayMs":
						c.HarnessRestartBaseDelayMs = int(value.(float64))
					}
				}
			}); err != nil {
				return nil, err
			}
			if restart && sup != nil {
				sup.Restart()
			}
			return map[string]any{"ok": true, "restarting": restart}, nil
		},
		RemovePlugin: func(ctx context.Context, name string) (any, error) {
			return callDesktopPlugin(ctx, "pluginManager/removeBundle", map[string]any{"name": strings.TrimSpace(name)})
		},
		DisablePlugins: func(ctx context.Context, names []string) (any, error) {
			if len(names) == 0 {
				return nil, errors.New("plugin name required")
			}
			return callDesktopPlugin(ctx, "pluginManager/setBundleEnabled", map[string]any{"name": strings.TrimSpace(names[0]), "enabled": false})
		},
		EnablePlugin: func(ctx context.Context, name string) (any, error) {
			return callDesktopPlugin(ctx, "pluginManager/setBundleEnabled", map[string]any{"name": strings.TrimSpace(name), "enabled": true})
		},
	}
}

func callDesktopPlugin(ctx context.Context, method string, payload map[string]any) (any, error) {
	state.dshMu.Lock()
	client := state.dshClient
	state.dshMu.Unlock()
	if client == nil {
		return nil, errors.New("harness client unavailable")
	}
	var out api.ChangeResult
	if err := client.Call(ctx, method, payload, &out); err != nil {
		return nil, err
	}
	return map[string]any{"ok": out.Error == nil, "changed": out.Changed, "target": out.Target, "enabled": out.Enabled}, nil
}

type settingsDraftSlot struct {
	ns    string
	field string
	val   string
	boolV bool
}

type desktopConfigDraft struct {
	harnessAutoRestart        bool
	harnessRestartMaxAttempts string
	harnessRestartBaseDelayMs string
	workspace                 string
	port                      string
	theme                     string
	locale                    string
	closeToTray               bool
	openAtLogin               bool
	autoStartDesktop          bool
	wallpaperPath             string
	showArchivedList          bool
	showViewTabs              bool // Electron viewTabs; stored inverted as HideChatViewTabs
	autoAllowTools            bool
	seeded                    bool
}

type credentialsDraft struct {
	apiKey           string
	baseURL          string
	githubToken      string
	remoteToken      string
	remoteRelayToken string
	seeded           bool
}

func (s *mainState) settingsDraftPtr(ns, field, seed string) *string {
	for i := range s.settingsDrafts {
		if s.settingsDrafts[i].ns == ns && s.settingsDrafts[i].field == field {
			s.settingsDrafts[i].val = seed
			return &s.settingsDrafts[i].val
		}
	}
	s.settingsDrafts = append(s.settingsDrafts, settingsDraftSlot{ns: ns, field: field, val: seed})
	return &s.settingsDrafts[len(s.settingsDrafts)-1].val
}

func (s *mainState) seedDesktopDraft() {
	if s.desktopDraft.seeded {
		return
	}
	if s.cfgStore == nil {
		return
	}
	cfg := s.cfgStore.Get()
	s.desktopDraft = desktopConfigDraft{
		harnessAutoRestart:        cfg.HarnessAutoRestart,
		harnessRestartMaxAttempts: strconv.Itoa(cfg.HarnessRestartMaxAttempts),
		harnessRestartBaseDelayMs: strconv.Itoa(cfg.HarnessRestartBaseDelayMs),
		workspace:                 cfg.Workspace,
		port:                      strconv.Itoa(cfg.Port),
		theme:                     cfg.Theme,
		locale:                    cfg.Locale,
		closeToTray:               cfg.CloseToTray,
		openAtLogin:               cfg.OpenAtLogin,
		autoStartDesktop:          cfg.AutoStartDesktop,
		wallpaperPath:             cfg.WallpaperPath,
		showArchivedList:          !cfg.HideArchivedList,
		showViewTabs:              !cfg.HideChatViewTabs,
		autoAllowTools:            cfg.AutoAllowTools,
		seeded:                    true,
	}
	s.desktopSaveErr = ""
}

func (s *mainState) seedCredentialsDraft() {
	creds, err := credentials.Load(paths.UserDataDir())
	if err != nil {
		s.credsSaveErr = err.Error()
		s.credsDraft.seeded = true
		return
	}
	s.credsDraft = credentialsDraft{
		apiKey:           creds.APIKey,
		baseURL:          creds.BaseURL,
		githubToken:      creds.GithubToken,
		remoteToken:      creds.RemoteToken,
		remoteRelayToken: creds.RemoteRelayToken,
		seeded:           true,
	}
	s.credsSaveErr = ""
}

func (s *mainState) saveCredentials() {
	dir := paths.UserDataDir()
	existing, err := credentials.Load(dir)
	if err != nil {
		s.credsSaveErr = err.Error()
		s.credsLastAction = ""
		return
	}
	existing.APIKey = strings.TrimSpace(s.credsDraft.apiKey)
	existing.BaseURL = strings.TrimSpace(s.credsDraft.baseURL)
	existing.GithubToken = strings.TrimSpace(s.credsDraft.githubToken)
	existing.RemoteToken = strings.TrimSpace(s.credsDraft.remoteToken)
	existing.RemoteRelayToken = strings.TrimSpace(s.credsDraft.remoteRelayToken)
	if err := credentials.Save(dir, existing); err != nil {
		s.credsSaveErr = err.Error()
		s.credsLastAction = ""
		return
	}
	s.credsSaveErr = ""
	s.credsLastAction = i18n.T(i18n.KeySettingsCredSaved)
	s.applyRemoteCredentials(existing)
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if client != nil {
		go func(key string) {
			if err := settingsstore.SyncAPIKeyToHarness(context.Background(), client, key); err != nil {
				s.mu.Lock()
				s.credsLastAction = err.Error()
				s.mu.Unlock()
				return
			}
			s.mu.Lock()
			s.credsLastAction = i18n.T(i18n.KeySettingsCredSaved) + " · credentials/set"
			s.mu.Unlock()
		}(existing.APIKey)
	}
}

func (s *mainState) applyRemoteCredentials(creds credentials.Credentials) {
	s.hasRemoteToken = strings.TrimSpace(creds.RemoteToken) != ""
	s.hasRemoteRelayToken = strings.TrimSpace(creds.RemoteRelayToken) != ""
	s.remoteDeviceCount = len(creds.RemoteDevices)
	s.remoteDeviceNames = phoneremote.DeviceNames(creds.RemoteDevices)
}

func (s *mainState) phoneRuntimeManager() *phoneremote.Runtime {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.phoneRuntime == nil {
		s.phoneRuntime = phoneremote.NewRuntime()
		s.phoneRuntime.SetGitAuthorizer(s.authorizePhoneGit)
		s.phoneRuntime.SetGitReviewer(s.reviewPhonePullRequest)
	}
	return s.phoneRuntime
}
func (s *mainState) stopPhoneRuntime() {
	s.revokePhoneGit()
	s.phoneOp.Lock()
	defer s.phoneOp.Unlock()
	s.phoneRuntimeManager().Stop()
}
func (s *mainState) refreshPhoneRemote() (msg string) {
	s.phoneOp.Lock()
	defer s.phoneOp.Unlock()
	defer func() {
		s.mu.Lock()
		s.phoneActionMsg = msg
		s.mu.Unlock()
	}()
	manager := s.phoneRuntimeManager()
	if s.cfgStore == nil {
		return "remote configuration unavailable"
	}
	cfg := s.cfgStore.Get()
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	quitting := s.quitting
	s.mu.Unlock()
	if quitting {
		return "offline"
	}
	if !cfg.RemoteEnabled {
		manager.Stop()
		return "disabled"
	}
	gitRoots := s.phoneGitRoots()
	runtimeConfig := map[string]any{
		"remoteGitEnabled": len(gitRoots) > 0, "remoteGitRoots": gitRoots,
		"remoteEnabled": cfg.RemoteEnabled, "remoteMode": cfg.RemoteMode,
		"remotePort": cfg.RemotePort, "remoteBindAddress": cfg.RemoteBindAddress,
		"remoteLanTls": cfg.RemoteLanTls, "remoteAppBaseUrl": cfg.RemoteAppBaseUrl,
		"remoteRelayEndpoint": cfg.RemoteRelayEndpoint, "remoteRelayUseTls": cfg.RemoteRelayUseTls,
		"harnessOrigin": ep.Origin, "harnessCookie": ep.Cookie,
	}
	if manager.Snapshot().Status == "starting" {
		return "starting"
	}
	if manager.Snapshot().Listening {
		manager.Refresh(runtimeConfig)
	} else {
		manager.Start(paths.UserDataDir(), paths.VendorDir(), runtimeConfig)
	}
	snapshot := manager.Snapshot()
	msg = snapshot.Status
	if snapshot.Message != "" {
		msg += ": " + snapshot.Message
	}
	return msg
}
func (s *mainState) phoneConnectURL() string {
	if s.cfgStore == nil || !s.cfgStore.Get().RemoteEnabled {
		return ""
	}
	return s.phoneRuntimeManager().Snapshot().PairingURL
}

func (s *mainState) saveDesktopConfig() {
	if s.cfgStore == nil {
		return
	}
	port, err := strconv.Atoi(strings.TrimSpace(s.desktopDraft.port))
	if err != nil || port < 1024 || port > 65535 {
		s.desktopSaveErr = "invalid port (1024-65535)"
		s.desktopLastAction = ""
		return
	}
	restartPolicy, err := config.ParseRestartPreferences(s.desktopDraft.harnessAutoRestart, s.desktopDraft.harnessRestartMaxAttempts, s.desktopDraft.harnessRestartBaseDelayMs)
	if err != nil {
		s.desktopSaveErr = err.Error()
		s.desktopLastAction = ""
		return
	}
	if err := s.materializeWallpaperDraft(); err != nil {
		s.desktopSaveErr = err.Error()
		return
	}
	err = s.cfgStore.Update(func(c *config.Config) {
		restartPolicy.Apply(c)
		c.Workspace = strings.TrimSpace(s.desktopDraft.workspace)
		c.Port = port
		c.Theme = s.desktopDraft.theme
		c.Locale = s.desktopDraft.locale
		c.CloseToTray = s.desktopDraft.closeToTray
		c.OpenAtLogin = s.desktopDraft.openAtLogin
		c.AutoStartDesktop = s.desktopDraft.autoStartDesktop
		c.WallpaperPath = strings.TrimSpace(s.desktopDraft.wallpaperPath)
		c.HideArchivedList = !s.desktopDraft.showArchivedList
		c.HideChatViewTabs = !s.desktopDraft.showViewTabs
		c.AutoAllowTools = s.desktopDraft.autoAllowTools
	})
	if err != nil {
		s.desktopSaveErr = err.Error()
		s.desktopLastAction = ""
		return
	}
	cfg := s.cfgStore.Get()
	if config.RestartPreferencesFrom(cfg) != restartPolicy {
		s.desktopSaveErr = "Harness restart policy was not saved; draft retained"
		s.desktopLastAction = ""
		return
	}
	s.desktopSaveErr = ""
	s.desktopLastAction = i18n.T(i18n.KeySettingsDesktopSaved)
	s.closeToTray = cfg.CloseToTray
	s.themeID = cfg.Theme
	i18n.SetLocale(cfg.Locale)
	applyAppearance(cfg.Theme)
	if err := desktopSystemIntegration(func() error { return mygo.App.SetOpenAtLogin(cfg.OpenAtLogin) }); err != nil {
		s.desktopLastAction = ""
		s.desktopSaveErr = fmt.Sprintf("%s: %v", resumeText("设置已保存，但开机启动未生效", "Settings saved, but startup registration failed"), err)
		log.Println("openAtLogin:", err)
	}
	if s.sessions != nil {
		s.sessions.SetWorkspaceCwd(cfg.Workspace)
	}
	if s.interactions != nil {
		s.interactions.SetAutoAllow(cfg.AutoAllowTools)
	}
}

// pickAndApplyWorkspace opens the Host registry picker. Launcher startup-directory
// selection remains independent and can still be used while the Host is offline.
func (s *mainState) pickAndApplyWorkspace(win *mygo.Window, sessions *session.Store, defaultPath string) {
	if sessions != nil {
		sessions.OpenWorkspacePicker()
	}
	if win != nil {
		win.Update(func() {})
	}
}

// applyWorkspacePath adopts a Host path transactionally. mkdirLocal is retained
// for call compatibility; registry adoption never creates desktop directories.
func (s *mainState) applyWorkspacePath(win *mygo.Window, sessions *session.Store, ws string, mkdirLocal bool) {
	if sessions == nil {
		return
	}
	sessions.OpenWorkspacePicker()
	request := sessions.Snapshot().WorkspaceNavigation
	if request == nil || request.Mode != "pick" || request.Busy {
		return
	}
	sessions.RegisterWorkspace(context.Background(), request.Token, ws, s.persistRegisteredWorkspace, func(row session.WorkspaceRow) { s.workspaceSelectionCommitted(win, sessions, row.Path) })
}

func (s *mainState) settingsBoolPtr(ns, field, seed string) *bool {
	for i := range s.settingsDrafts {
		if s.settingsDrafts[i].ns == ns && s.settingsDrafts[i].field == field {
			s.settingsDrafts[i].val = seed
			s.settingsDrafts[i].boolV = seed == "true"
			return &s.settingsDrafts[i].boolV
		}
	}
	s.settingsDrafts = append(s.settingsDrafts, settingsDraftSlot{
		ns: ns, field: field, val: seed, boolV: seed == "true",
	})
	return &s.settingsDrafts[len(s.settingsDrafts)-1].boolV
}

func (s *mainState) showMainWindow() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win == nil {
		return
	}
	win.Show()
	win.Restore()
	win.Focus()
}

func (s *mainState) applyDeeplink(t deeplink.Target) {
	s.assistantBotCreator.Suspend()
	s.assistantBotVisibility.Suspend()
	s.assistantContacts.Suspend()
	s.assistantBotDeleter.Suspend()
	s.assistantBotDuplicator.Suspend()
	s.assistantRoomCreator.Suspend()
	s.assistantRoomEditor.Suspend()
	s.assistantRoomRuntime.Suspend()
	s.assistantRoutines.Bind(dshbotstore.RoutineScope{})
	s.mu.Lock()
	s.pendingDeeplink = t
	switch t.Action {
	case deeplink.ActionLauncher:
		s.launcherOpen = true
		s.settingsOpen = false
		s.usageOpen = false
		s.scheduleOpen = false
		s.webviewOpen = false
		s.browserPreviewOpen = false
		s.assistantConversationGeneration++
		s.assistantOpen = false
	case deeplink.ActionSettings:
		s.settingsEmbeddedSection = ""
		s.settingsOpen = true
		s.launcherOpen = false
		s.usageOpen = false
		s.scheduleOpen = false
		s.webviewOpen = false
		s.browserPreviewOpen = false
		s.assistantConversationGeneration++
		s.assistantOpen = false
		s.settingsCredentialsActive = false
		s.settingsDesktopActive = false
		s.settingsAccountActive = false
		s.settingsAppearanceActive = false
		s.settingsInterfaceActive = false
		s.settingsModelsActive = false
		s.settingsMCPActive = false
		s.settingsPluginsActive = false
		s.settingsSkillsActive = false
		s.settingsAgentPresetsActive = false
		s.settingsAboutActive = false
		switch strings.ToLower(t.Section) {
		case "credentials", "cred":
			s.settingsCredentialsActive = true
		case "account", "login":
			s.settingsAccountActive = true
		case "appearance", "theme":
			s.settingsAppearanceActive = true
			s.seedDesktopDraft()
		case "interface", "ui":
			s.settingsInterfaceActive = true
			s.seedDesktopDraft()
		case "models", "model":
			s.settingsModelsActive = true
		case "mcp":
			s.settingsMCPActive = true
		case "plugins", "plugin":
			s.settingsPluginsActive = true
		case "skills", "skill":
			s.settingsSkillsActive = true
		case "agent-presets", "agentpresets", "agents":
			s.settingsAgentPresetsActive = true
		case "about":
			s.settingsAboutActive = true
		case "desktop", "":
			s.settingsDesktopActive = true
			s.seedDesktopDraft()
		default:
			s.settingsDesktopActive = true
			s.seedDesktopDraft()
		}
	case deeplink.ActionUsage:
		s.usageOpen = true
		s.launcherOpen = false
		s.settingsOpen = false
		s.scheduleOpen = false
		s.marketOpen = false
		s.webviewOpen = false
		s.browserPreviewOpen = false
		s.assistantConversationGeneration++
		s.assistantOpen = false
	case deeplink.ActionMarket:
		s.marketOpen = true
		s.marketManager = false
		s.launcherOpen = false
		s.settingsOpen = false
		s.usageOpen = false
		s.scheduleOpen = false
		s.webviewOpen = false
		s.browserPreviewOpen = false
		s.assistantConversationGeneration++
		s.assistantOpen = false
	case deeplink.ActionSchedule:
		s.scheduleOpen = true
		s.launcherOpen = false
		s.settingsOpen = false
		s.usageOpen = false
		s.marketOpen = false
		s.webviewOpen = false
		s.browserPreviewOpen = false
		s.assistantConversationGeneration++
		s.assistantOpen = false
	case deeplink.ActionAgents:
		s.filesPaneOpen = true
		s.rightPaneTab = "agents"
		s.launcherOpen = false
		s.settingsOpen = false
		s.usageOpen = false
		s.scheduleOpen = false
		s.marketOpen = false
	case deeplink.ActionAuth:
		// OAuth return from account/startSignIn (callbackOrigin whaleisle://auth).
		s.settingsEmbeddedSection = ""
		s.settingsOpen = true
		s.launcherOpen = false
		s.usageOpen = false
		s.scheduleOpen = false
		s.marketOpen = false
		s.webviewOpen = false
		s.browserPreviewOpen = false
		s.assistantConversationGeneration++
		s.assistantOpen = false
		s.settingsCredentialsActive = false
		s.settingsDesktopActive = false
		s.settingsAccountActive = true
		s.settingsAppearanceActive = false
		s.settingsInterfaceActive = false
		s.settingsModelsActive = false
		s.settingsMCPActive = false
		s.settingsPluginsActive = false
		s.settingsSkillsActive = false
		s.settingsAgentPresetsActive = false
		s.settingsAboutActive = false
	case deeplink.ActionSession:
		if s.sessions != nil && t.SessionID != "" {
			s.revokePhoneGit()
			s.sessions.Select(t.SessionID)
		}
		s.launcherOpen = false
		s.settingsOpen = false
		s.usageOpen = false
		s.scheduleOpen = false
	case deeplink.ActionOpen:
		s.launcherOpen = false
	}
	win := s.win
	usage := s.usage
	sessions := s.sessions
	s.mu.Unlock()
	s.showMainWindow()
	if t.Action == deeplink.ActionAuth && usage != nil {
		sid := ""
		if sessions != nil {
			sid = sessions.Snapshot().SelectedID
		}
		usage.Refresh(context.Background(), sid)
	}
	if win != nil {
		win.Update(func() {})
	}
	log.Printf("deeplink: action=%s session=%s section=%s", t.Action, t.SessionID, t.Section)
}

func (s *mainState) closeWebViewHost() {
	s.closeAccountPlatform()
	s.mu.Lock()
	host := s.webviewHost
	if host != nil {
		s.lastWebViewURL = host.URL()
	}
	s.webviewHost = nil
	s.pluginViewResumeAfterRestart = false
	s.webviewHostErr = ""
	s.mu.Unlock()
	if host != nil {
		_ = host.Close()
	}
}

func (s *mainState) syncHarnessFromConfig() {
	s.mu.Lock()
	store := s.cfgStore
	sup := s.sup
	s.mu.Unlock()
	if store == nil || sup == nil {
		return
	}
	cfg := store.Get()
	sup.SetSkipUserPlugins(cfg.SkipUserPlugins)
	sup.SetPrepareOptions(harness.PrepareOptions{
		DisabledPlugins: cfg.DisabledPlugins,
		RemoteSet:       true,
		Remote:          cfg.RemoteWorkspaceEnabled,
		SkipUserPlugins: cfg.SkipUserPlugins,
		Whale:           cfg.WhaleAssistantEnabled,
		Dshbot:          cfg.DshbotEnabled,
	})
}

func (s *mainState) setAssistantFlag(whale *bool, dshbot *bool) {
	s.mu.Lock()
	store := s.cfgStore
	sup := s.sup
	win := s.win
	s.mu.Unlock()
	if store == nil {
		return
	}
	if err := store.Update(func(c *config.Config) {
		if whale != nil {
			c.WhaleAssistantEnabled = *whale
		}
		if dshbot != nil {
			c.DshbotEnabled = *dshbot
		}
	}); err != nil {
		log.Printf("assistant: config update: %v", err)
		return
	}
	if sup != nil {
		s.syncHarnessFromConfig()
		state.restartUnlessImport(sup)
	}
	if win != nil {
		win.Update(func() {})
	}
}

// Assistant entry hands the resolved canonical session to the existing chat owner.
func (s *mainState) handleAssistantNativePrompt(_ string, ep harness.Endpoint, text string) {
	s.mu.Lock()
	asst := s.assistant
	s.mu.Unlock()
	if asst == nil || strings.TrimSpace(text) == "" {
		return
	}
	target := asst.Snapshot()
	if target.Mode == assiststore.ModeDshbot && target.DshbotOrigin != ep.Origin {
		asst.SetLastAction("Select a Bot from the current Host roster first", "Host changed; draft retained")
		return
	}
	s.openAssistantConversation(ep, target.Mode, target.DshbotID, strings.TrimSpace(text))
}

func (s *mainState) pluginRPCClient(ep harness.Endpoint) *pluginrpc.Client {
	return &pluginrpc.Client{BaseURL: ep.Origin, Cookie: ep.Cookie}
}

func (s *mainState) runAssistantPluginRPC(win *mygo.Window, ep harness.Endpoint, label string, fn func(context.Context, *pluginrpc.Client) (string, error)) {
	s.mu.Lock()
	if s.assistantPluginBusy {
		s.mu.Unlock()
		return
	}
	s.assistantPluginBusy = true
	s.assistantPluginDetail = label + "…"
	s.assistantLastAct = s.assistantPluginDetail
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
	go func() {
		detail, err := fn(context.Background(), s.pluginRPCClient(ep))
		msg := detail
		if err != nil {
			msg = label + ": " + err.Error()
		} else if msg == "" {
			msg = label + ": ok"
		} else {
			msg = label + ": " + truncateRunes(msg, 180)
		}
		s.mu.Lock()
		s.assistantPluginBusy = false
		s.assistantPluginDetail = msg
		s.assistantLastAct = msg
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}()
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func (s *mainState) openWebViewHost(origin string) {
	s.openWebViewHostPlugin(origin, "")
}

func (s *mainState) openWebViewHostPlugin(origin, pluginID string) {
	s.mu.Lock()
	s.lastWebViewOrigin = origin
	s.mu.Unlock()
	s.openWebViewURL(desktopwebview.SlotURL(origin, pluginID, nil))
}

func (s *mainState) showBrowserPreview(rawURL string) {
	s.mu.Lock()
	s.browserPreviewOpen = true
	s.webviewOpen = false
	win := s.win
	s.mu.Unlock()
	s.closeWebViewHost()
	if s.browserController == nil {
		s.browserController = s.newBrowserController(win)
	}
	if strings.TrimSpace(rawURL) != "" {
		s.browserController.Navigate(rawURL)
	}
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) openWebViewURL(browse string) {
	s.closeAccountPlatform()
	s.mu.Lock()
	s.accountReturn.embedded = ""
	win := s.win
	existing := s.webviewHost
	ep := s.bootStatus.Endpoint
	hostGeneration := s.bootStatus.Generation
	s.mu.Unlock()
	if win == nil {
		return
	}
	browse = strings.TrimSpace(browse)
	if browse == "" {
		s.mu.Lock()
		s.webviewHostErr = "URL unavailable"
		s.mu.Unlock()
		return
	}
	sessionOrigin, sessionCookie := "", ""
	if desktopwebview.IsHostURL(browse, ep.Origin) {
		sessionOrigin, sessionCookie = ep.Origin, ep.Cookie
	}
	if existing != nil && existing.Alive() && existing.MatchesSession(sessionOrigin, sessionCookie) {
		if err := existing.Navigate(browse); err != nil {
			s.mu.Lock()
			s.webviewHostErr = err.Error()
			s.mu.Unlock()
			return
		}
		existing.SetVisible(true)
		s.mu.Lock()
		s.lastWebViewURL = browse
		s.pluginViewResumeAfterRestart = false
		s.webviewHostErr = ""
		s.mu.Unlock()
		return
	}
	if existing != nil {
		_ = existing.Close()
	}
	s.mu.Lock()
	browserPanel := s.browserPreviewOpen
	s.mu.Unlock()
	panelH := webviewui.PanelChromeHeight
	hostTitle := i18n.T(i18n.KeyWebViewTitle)
	if browserPanel {
		panelH = browserpreviewui.PanelChromeHeight
		hostTitle = i18n.T(i18n.KeyBrowserPreviewTitle)
	}
	inset := chrome.TitleBarHeight + panelH
	host, err := desktopwebview.NewHost(desktopwebview.HostOptions{
		Untrusted:           true,
		SessionOrigin:       sessionOrigin,
		ShortcutPreferences: s.pageShortcutPreferences(),
		SessionCookie:       sessionCookie,
		OnChange:            func() { win.Update(func() {}) },
		Parent:              win,
		URL:                 browse,
		Title:               hostTitle,
		TopInset:            inset,
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.webviewHost = nil
		s.webviewHostErr = err.Error()
		return
	}
	s.webviewHost = host
	s.pluginViewResumeAfterRestart = false
	s.pluginViewHostGeneration = hostGeneration
	s.lastWebViewURL = browse
	s.webviewHostErr = ""
	host.SetVisible(true)
}

const terminalMinBottom = float32(40)

func (s *mainState) terminalShown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminalVisible && s.layout.Bottom > terminalMinBottom
}

func (s *mainState) toggleTerminal(workspaceRoot string, systemDark bool) {
	s.mu.Lock()
	if s.terminalVisible {
		s.terminalSavedBottom = s.layout.Bottom
		if s.terminalSavedBottom <= terminalMinBottom {
			s.terminalSavedBottom = frame.DefaultLayout().Bottom
		}
		s.layout.Bottom = 0
		s.terminalVisible = false
	} else {
		s.terminalVisible = true
		if s.terminalSavedBottom > terminalMinBottom {
			s.layout.Bottom = s.terminalSavedBottom
		} else {
			s.layout.Bottom = frame.DefaultLayout().Bottom
		}
	}
	s.mu.Unlock()
	s.ensureTerminal(workspaceRoot, systemDark)
}

func (s *mainState) ensureTerminal(workspaceRoot string, systemDark bool) {
	root := strings.TrimSpace(workspaceRoot)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.terminalVisible || s.layout.Bottom <= terminalMinBottom {
		return
	}
	if s.terminalPanel != nil && s.terminalPanel.Workspace() == root {
		return
	}
	s.terminalPanel = s.terminalSessions.Get(root, systemDark)
}

func (s *mainState) setFilesPaneOpen(open bool) {
	s.mu.Lock()
	s.filesPaneOpen = open
	s.mu.Unlock()
	// The Files tree owns directory subscriptions, including cancellation when
	// hidden, collapsed, or rebound. No separate unscoped status-only watch.
}

func (s *mainState) closeTerminal() {
	s.mu.Lock()
	s.terminalSessions.Close()
	s.terminalPanel = nil
	s.terminalShellsNote = ""
	s.mu.Unlock()
}

func (s *mainState) restartTerminal(workspaceRoot string, systemDark bool) {
	s.mu.Lock()
	s.terminalPanel = s.terminalSessions.Restart(workspaceRoot, systemDark)
	s.terminalShellsNote = ""
	s.terminalVisible = true
	if s.layout.Bottom <= terminalMinBottom {
		if s.terminalSavedBottom > terminalMinBottom {
			s.layout.Bottom = s.terminalSavedBottom
		} else {
			s.layout.Bottom = frame.DefaultLayout().Bottom
		}
	}
	s.mu.Unlock()
	s.ensureTerminal(workspaceRoot, systemDark)
}

func (s *mainState) sessionRunningCount() int {
	s.mu.Lock()
	store := s.sessions
	s.mu.Unlock()
	if store == nil {
		return 0
	}
	return store.RunningCount()
}

// promptAcceptUnknownSSHHost asks whether to trust an unknown host key (TOFU).
// Returns true when the user chooses Accept.
func promptAcceptUnknownSSHHost(win *mygo.Window, unknown *sshremote.UnknownHostKey, beforeShow func() bool) bool {
	msg := fmt.Sprintf(
		i18n.T(i18n.KeyRemoteSSHHostUnknownMessage),
		unknown.Address, unknown.KeyType, unknown.Fingerprint,
	)
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:        win,
		Type:          mygo.MessageQuestion,
		Title:         i18n.T(i18n.KeyRemoteSSHHostUnknownTitle),
		Message:       msg,
		Buttons:       []string{i18n.T(i18n.KeyRemoteSSHHostDecline), i18n.T(i18n.KeyRemoteSSHHostAccept)},
		DefaultButton: 0,
		CancelButton:  0,
		BeforeShow:    beforeShow,
	})
	return err == nil && res.Button == 1
}

// showSSHHostKeyChanged refuses a mismatched host key with a clear dialog.
func showSSHHostKeyChanged(win *mygo.Window, changed *sshremote.HostKeyChanged, beforeShow func() bool) {
	msg := fmt.Sprintf(
		i18n.T(i18n.KeyRemoteSSHHostChangedMessage),
		changed.Address, changed.KeyType, changed.Fingerprint,
	)
	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Parent:     win,
		Type:       mygo.MessageWarning,
		Title:      i18n.T(i18n.KeyRemoteSSHHostChangedTitle),
		Message:    msg,
		Buttons:    []string{"OK"},
		BeforeShow: beforeShow,
	})
}

func (s *mainState) showElectronImportReport(win *mygo.Window) {
	rep, err := importer.ImportReport(importer.ElectronUserDataDir())
	msg := importer.FormatReport(rep)
	if err != nil {
		msg = err.Error()
	}
	_, _ = mygo.Dialog.Message(mygo.MessageOptions{
		Parent:  win,
		Type:    mygo.MessageInfo,
		Title:   i18n.T(i18n.KeySettingsDesktopDetectElectronTitle),
		Message: msg,
		Buttons: []string{"OK"},
	})
}

func (s *mainState) updateDownloadRoute() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.launcherDownloadRoute != "" {
		return s.launcherDownloadRoute
	}
	if s.cfgStore != nil {
		return s.cfgStore.Get().DownloadRoute
	}
	return ""
}

func (s *mainState) shouldCloseToTray() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closeToTray && !isolatedDesktopRun()
}

// acceptOpenHookPath is a one-shot acceptance driver: write a panel id, app opens it.
// File: %TEMP%/whaleisle-accept-open  (contents: market|usage|assist|schedule|im|remote|browser|webview|settings|launcher|desktop|term)
func acceptOpenHookPath() string {
	return filepath.Join(os.TempDir(), "whaleisle-accept-open")
}

func (s *mainState) pollAcceptOpenHook() bool {
	if isolatedDesktopRun() {
		return false
	}
	path := acceptOpenHookPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	_ = os.Remove(path)
	cmd := strings.TrimSpace(strings.ToLower(string(b)))
	if cmd == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.launcherOpen = false
	s.launcherComponentsFresh = false
	s.toolsMenuOpen = false
	s.settingsOpen = false
	s.usageOpen = false
	s.imOpen = false
	s.marketOpen = false
	s.scheduleOpen = false
	s.remoteOpen = false
	s.webviewOpen = false
	s.browserPreviewOpen = false
	s.assistantConversationGeneration++
	s.assistantOpen = false
	switch cmd {
	case "market":
		s.marketOpen = true
	case "usage":
		s.usageOpen = true
	case "assist", "assistant":
		s.assistantConversationGeneration++
		s.assistantOpen = true
	case "schedule", "sched":
		s.scheduleOpen = true
	case "im":
		s.imOpen = true
	case "remote":
		s.remoteOpen = true
	case "browser":
		s.browserPreviewOpen = true
	case "webview":
		s.webviewOpen = true
	case "settings", "s-desk":
		s.settingsEmbeddedSection = ""
		s.settingsOpen = true
		s.settingsDesktopActive = true
		s.seedDesktopDraft()
	case "launcher", "l-home":
		s.launcherOpen = true
		s.launcherTab = launcher.TabHome
	case "l-set", "launcher-settings":
		s.launcherOpen = true
		s.launcherTab = launcher.TabSettings
	case "l-comp", "launcher-components":
		s.launcherOpen = true
		s.launcherTab = launcher.TabComponents
	case "l-ver", "launcher-versions":
		s.launcherOpen = true
		s.launcherTab = launcher.TabVersions
	case "l-imp", "launcher-import":
		s.launcherOpen = true
		s.launcherTab = launcher.TabImport
	case "l-plug", "launcher-plugins":
		s.launcherOpen = true
		s.launcherTab = launcher.TabPlugins
	case "desktop", "d-main":
		// all overlays already cleared
	case "term", "terminal":
		s.terminalVisible = true
		if s.layout.Bottom <= terminalMinBottom {
			if s.terminalSavedBottom > terminalMinBottom {
				s.layout.Bottom = s.terminalSavedBottom
			} else {
				s.layout.Bottom = frame.DefaultLayout().Bottom
			}
		}
	default:
		log.Printf("accept-open: unknown %q", cmd)
		return false
	}
	log.Printf("accept-open: %s", cmd)
	return true
}

func (s *mainState) watchAcceptOpenHook() {
	if isolatedDesktopRun() {
		return
	}
	go func() {
		for {
			time.Sleep(200 * time.Millisecond)
			if s.pollAcceptOpenHook() {
				s.mu.Lock()
				w := s.win
				s.mu.Unlock()
				if w != nil {
					w.Update(func() {})
				}
			}
		}
	}()
}

func (s *mainState) refreshBoot(st harness.Status, owner ...*harness.Supervisor) {
	s.mu.Lock()
	if len(owner) > 0 && owner[0] != s.sup {
		s.mu.Unlock()
		return
	}
	if st.Generation != 0 && s.sup != nil {
		actual := s.sup.Status()
		if st.Generation != actual.Generation || st.State != actual.State {
			s.mu.Unlock()
			return
		}
		st = actual
	}
	prev := s.bootStatus.State
	// The actual supervisor generation/state check above rejects late snapshots.
	s.bootStatus = st
	persistenceFailed := s.recordStartupStatusLocked(st)
	switch st.State {
	case harness.Starting:
		s.launcherStartBusy = true
		if !st.RecoveryActive && !st.RecoveryCancelled {
			s.launcherOpen = true
		}
	case harness.Ready:
		s.launcherStartBusy = false
		s.launcherStartFailed = false
		s.launcherStartErr = ""
		quitLauncher := true
		if s.cfgStore != nil {
			quitLauncher = s.cfgStore.Get().QuitAfterStart
		}
		if st.RecoveryCancelled {
			// Cancellation keeps the current page, including an explicitly opened launcher.
		} else if startuprec.ShouldCloseLauncher(true, quitLauncher, st.Recovery, persistenceFailed || s.startupRecovery.updateHold) {
			s.launcherOpen = false
		} else {
			s.launcherOpen = true
		}
	case harness.Failed:
		s.launcherStartBusy = false
		if !st.RecoveryCancelled {
			s.launcherOpen = true
		}
		if s.launcherTriedStart || prev == harness.Starting {
			s.launcherStartFailed = true
			if st.Err != nil {
				s.launcherStartErr = st.Err.Error()
			}
		}
	}
	if s.sup != nil {
		s.bootLogs = s.sup.Logs()
	}
	win := s.win
	s.mu.Unlock()
	s.syncPluginViewHost(st)
	s.syncDialogue(st)
	s.observeMarketActivation(st)
	if win != nil {
		win.Update(func() { s.refreshPetTask(); s.refreshDesktopTray(); s.refreshStartupUpdateAttention(win) })
	}
}

func (s *mainState) syncDialogue(st harness.Status) {
	if st.State != harness.Ready {
		s.stopDSH()
		return
	}
	ep := st.Endpoint
	key := ep.Origin + "\x00" + ep.Cookie
	s.dshMu.Lock()
	if s.dshEndpoint == key && s.dshCancel != nil {
		s.dshMu.Unlock()
		return
	}
	s.stopDSHLocked()
	ctx, cancel := context.WithCancel(context.Background())
	s.dshCancel = cancel
	s.dshEndpoint = key
	s.dshMuxReady = false
	s.dshMuxLost = false
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}}
	s.replaceMCPControllerLocked()
	s.settingsProviderProfiles.ResetHost()
	s.dshClient = client
	s.accountSignOut.Bind(client)
	s.attachPluginInventory(client)
	s.settingsReferencedCredentials.Attach(client)
	s.settingsProviderDiscovery.Attach(client)
	s.mu.Lock()
	if s.sessions == nil {
		s.sessions = session.New()
	}
	if s.interactions == nil {
		s.interactions = interaction.New()
	}
	if s.cfgStore != nil {
		s.interactions.SetAutoAllow(s.cfgStore.Get().AutoAllowTools)
	}
	if s.settings == nil {
		s.settings = settingsstore.New()
		s.settings.SetVisualReplyEnableCheck(s.verifyVisualReplyEnable)
	}
	if s.usage == nil {
		s.usage = usagestore.New()
		s.usage.SetStatsCachePath(filepath.Join(paths.DshHome(), "cache", "mygo-usage-overview-v4.json"))
	}
	if s.remoteMachines == nil {
		s.remoteMachines = remotestore.New()
	}
	if s.im == nil {
		s.im = imstore.New()
	}
	if s.market == nil {
		s.market = marketstore.New()
	}
	if s.schedule == nil {
		s.schedule = schedstore.New()
	}
	sessions := s.sessions
	inter := s.interactions
	setStore := s.settings
	usageStore := s.usage
	remoteMachines := s.remoteMachines
	imStore := s.im
	marketStore := s.market
	schedStore := s.schedule
	win := s.win
	s.mu.Unlock()
	s.dshMu.Unlock()

	refresh := func() {
		s.cancelComposerVoiceIfStale()
		if win != nil {
			win.Update(s.refreshPetTask)
		}
	}
	sessions.Attach(client, refresh)
	s.bindWorkspacePreferences(sessions, client)
	chat.SetAttachmentFetcher(func(ctx context.Context, attachmentID string) (string, error) {
		return sessions.FetchAttachment(ctx, attachmentID)
	})
	setStore.Attach(client, refresh)
	s.refreshPluginInventory()
	s.refreshReferencedCredentials()
	s.refreshProviderDiscovery()
	usageStore.Attach(client, refresh)
	usageStore.RefreshStats(context.Background(), false)
	usageStore.LoadBilling(context.Background())
	remoteMachines.Attach(client, refresh)
	imStore.Attach(client, refresh)
	marketStore.Attach(client, refresh)
	s.bindNativePluginManager(client)
	schedStore.Attach(client, refresh)
	go func() {
		_ = settingsstore.ReportProductEvent(context.Background(), client, api.ProductEvent(
			[]byte(`{"name":"desktop.ready","source":"whaleisle"}`),
		))
	}()
	setStore.RefreshVision(context.Background())
	go s.hydrateViewTabsFromHarness(setStore)
	s.mu.Lock()
	workspace := ""
	if s.cfgStore != nil {
		workspace = s.cfgStore.Get().Workspace
	}
	s.mu.Unlock()
	sessions.SetWorkspaceCwd(workspace)
	if inter != nil {
		inter.Attach(refresh)
	}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	s.dshMu.Lock()
	s.dshMux = mux
	s.dshMu.Unlock()
	s.hostTerminals.Bind(client, mux, false, refresh)
	sessions.AttachMux(mux)
	if s.usage != nil {
		s.usage.AttachMux(mux)
	}
	if inter != nil {
		inter.AttachClient(client, mux)
	}
	go s.runDSH(ctx, client, mux, inter)
	sessions.FollowWorkspace(ctx)
	sessions.WatchSpeechCatalog(ctx)
	sessions.InitializeDefaultWorkspace(context.Background())
	sessions.RefreshList(context.Background())
	sessions.InitializeDefaultModel(context.Background())
	sessions.RefreshModelCatalog(context.Background())
	sessions.RefreshPermissionCatalog(context.Background())
	sessions.RefreshAgentPresets(context.Background())
	sessions.RefreshSkills(context.Background())
}

func (s *mainState) stopDSH() {
	s.dshMu.Lock()
	defer s.dshMu.Unlock()
	s.stopDSHLocked()
}

func (s *mainState) stopDSHLocked() {
	s.assistantBotCreator.Suspend()
	s.assistantBotVisibility.Suspend()
	s.assistantContacts.Suspend()
	s.assistantBotDeleter.Suspend()
	s.assistantBotDuplicator.Suspend()
	s.assistantRoomCreator.Suspend()
	s.assistantRoomEditor.Suspend()
	s.assistantRoomRuntime.Suspend()
	s.hostTerminals.Bind(nil, nil, false, nil)
	s.cancelComposerVoice()
	s.mu.Lock()
	s.stopProjectProcessLocked()
	s.invalidateSettingsSkillCreateLocked()
	s.stopProjectRefreshLocked()
	s.closeProjectHistoricalLocked()
	s.mu.Unlock()
	if s.remoteFiles != nil {
		s.remoteFiles.Bind(nil, "", nil)
	}
	if s.remoteMachines != nil {
		s.remoteMachines.Attach(nil, nil)
	}
	s.invalidateMarketLocked()
	if s.dshCancel != nil {
		s.dshCancel()
		s.dshCancel = nil
	}
	s.dshEndpoint = ""
	s.replaceMCPControllerLocked()
	s.settingsProviderProfiles.ResetHost()
	s.dshClient = nil
	s.bindNativePluginManager(nil)
	s.clearAccountReturn()
	s.accountSignOut.Bind(nil)
	s.attachPluginInventory(nil)
	s.settingsReferencedCredentials.Attach(nil)
	s.settingsProviderDiscovery.Attach(nil)
	s.dshMux = nil
	s.dshMuxReady = false
	s.dshMuxLost = false
	if s.sessions != nil {
		s.sessions.Detach()
	}
	if s.interactions != nil {
		s.interactions.Detach()
	}
	if s.settings != nil {
		s.settings.Detach()
	}
	if s.usage != nil {
		s.usage.Detach()
	}
	if s.im != nil {
		s.im.Detach()
	}
	if s.market != nil {
		s.market.Detach()
	}
	if s.schedule != nil {
		s.schedule.Detach()
	}
}

func (s *mainState) refreshSettingsMCP() {
	s.refreshMCPController()
}

func (s *mainState) runDSH(ctx context.Context, client *dsh.Client, mux *dsh.Mux, inter *interaction.Store) {
	pluginEvent := s.nativePluginEventHandler(client)
	s.mu.Lock()
	sessions := s.sessions
	schedStore := s.schedule
	setStore := s.settings
	usageStore := s.usage
	win := s.win
	s.mu.Unlock()
	refresh := func() {
		if win != nil {
			win.Update(s.refreshPetTask)
		}
	}
	setMuxState := func(ready, lost bool) {
		s.mu.Lock()
		s.dshMuxReady = ready
		s.dshMuxLost = lost
		s.mu.Unlock()
		s.hostTerminals.Transport(client, ready)
		refresh()
	}
	_ = dsh.MaintainMux(ctx, mux, dsh.Backoff{}, func(ctx context.Context) error {
		setMuxState(true, false)
		var scheduleEventsOwner *schedstore.EventsOwner
		if schedStore != nil {
			scheduleEventsOwner = schedStore.BeginEvents(ctx, client)
		}
		var eventsOwner *session.EventsOwner
		if sessions != nil {
			eventsOwner = sessions.BeginEvents(ctx, client)
		}
		go func() {
			err := client.FollowEvents(ctx, mux, dsh.EventsFuncs{
				Ready: func(_ dsh.EventsReady) {
					pluginEvent(dsh.EventsEmit{Event: "plugin-manager/changed"})
					if schedStore != nil {
						schedStore.OnEventsReady(scheduleEventsOwner)
					}
					if sessions != nil {
						sessions.OnEventsReady(eventsOwner)
					}
				},
				Emit: func(emit dsh.EventsEmit) {
					pluginEvent(emit)
					if usageStore != nil {
						usageStore.OnAccountNotice(ctx, client, emit)
					}
					if schedStore != nil {
						schedStore.OnEmit(scheduleEventsOwner, emit)
					}
					if setStore != nil {
						setStore.OnEmitMaybeRefresh(ctx, emit)
					}
					if sessions != nil {
						sessions.OnEmit(eventsOwner, emit)
					}
				},
				Waterfall: func(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
					if inter == nil {
						return nil, nil
					}
					return inter.HandleWaterfall(wf)
				},
				Result: func(result dsh.EventResult, err error) bool {
					if inter != nil {
						return inter.OnResult(result, err)
					}
					return false
				},
				Cancel: func(c dsh.EventsCancel) {
					if inter != nil {
						inter.OnCancel(c)
					}
				},
			})
			if err != nil && ctx.Err() == nil {
				setMuxState(false, true)
			}
		}()
		if sessions != nil {
			sessions.FollowWorkspace(ctx)
			sessions.FollowSelected(ctx)
			sessions.WatchControl(ctx)
			sessions.RefreshProjections(ctx)
		}
		go settingsstore.WatchAnalyticsPolicy(ctx, mux, func(on bool) {
			s.mu.Lock()
			s.analyticsEnabled = &on
			s.mu.Unlock()
			refresh()
		})
		return nil
	}, func(error) {
		setMuxState(false, true)
	})
}

func connectionBanner(st harness.Status, muxReady, muxLost bool) string {
	switch st.State {
	case harness.Ready:
		// fall through to mux checks
	case harness.Failed:
		if st.Err != nil {
			return i18n.T(i18n.KeyBootFailed) + ": " + st.Err.Error()
		}
		return i18n.T(i18n.KeyBootFailed)
	case harness.Starting:
		return i18n.T(i18n.KeyBootStarting)
	default:
		return i18n.T(i18n.KeyChatConnConnecting)
	}
	if muxLost {
		return i18n.T(i18n.KeyChatConnReconnecting)
	}
	if !muxReady {
		return i18n.T(i18n.KeyChatConnConnecting)
	}
	return ""
}

func main() {
	mygo.App.SetName("Whale Isle")
	if err := preparePortableDesktop(); err != nil {
		log.Println("portable desktop:", err)
		return
	}
	if err := prepareIsolatedDesktop(); err != nil {
		log.Println("isolated desktop:", err)
		return
	}

	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	// A stale stop request must not cold-start the desktop.
	if len(os.Args) > 1 && os.Args[1] == "--managed-launcher-stop" {
		return
	}
	mygo.App.OnOpenURL(func(rawURL string) {
		if t, ok := deeplink.Parse(rawURL); ok {
			state.applyDeeplink(t)
			return
		}
		log.Printf("deeplink: ignored %s", rawURL)
	})
	if err := desktopSystemIntegration(func() error { return mygo.App.RegisterURLScheme(deeplink.Scheme) }); err != nil {
		log.Println("RegisterURLScheme:", err)
	}
	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		if managedLauncherStop(args) {
			state.quitDesktopTray()
			return
		}
		state.showMainWindow()
	})

	mygo.App.WhenReady(func() {
		if err := paths.EnsureUserData(); err != nil {
			log.Println("userdata:", err)
		}
		userData := paths.UserDataDir()
		imgDir := imgcache.Dir(userData)
		if err := os.MkdirAll(imgDir, 0o755); err != nil {
			log.Println("imgcache:", err)
		} else {
			cache := imgcache.New(imgDir)
			chat.SetRemoteImageCache(cache)
			marketui.SetRemoteImageCache(cache)
			attDir := filepath.Join(userData, "cache", "attachments")
			session.SetAttachmentCacheDir(attDir)
		}
		if imS, err := imstore.Open(userData); err != nil {
			log.Println("im:", err)
			state.mu.Lock()
			if state.im == nil {
				state.im = imstore.New()
			}
			state.mu.Unlock()
		} else {
			state.mu.Lock()
			state.im = imS
			state.imDraft = imS.Snapshot().Draft
			state.mu.Unlock()
		}
		if asst, err := assiststore.Open(userData); err != nil {
			log.Println("assistant:", err)
			state.mu.Lock()
			if state.assistant == nil {
				state.assistant = assiststore.New()
			}
			state.mu.Unlock()
		} else {
			state.mu.Lock()
			state.assistant = asst
			state.mu.Unlock()
		}
		state.openCustomPanels(userData)
		if roster, err := dshbotstore.Open(userData); err != nil {
			log.Println("dshbot roster:", err)
		} else {
			state.mu.Lock()
			state.dshbotRoster = roster
			state.mu.Unlock()
		}

		cfgStore, err := config.Open(userData)
		if err != nil {
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{Type: mygo.MessageWarning, Title: "Configuration could not be loaded / 配置读取失败", Message: err.Error() + "\n\nThe original file is preserved. Back up and repair config.json, then restart. / 原文件已保留，请备份并修复 config.json 后重启。"})
			mygo.App.Quit()
			return
		}
		if p, err := cfgStore.Appearance(); err == nil {
			theme.SetPreferences(p)
		}
		state.openStartupRecovery(userData, cfgStore)
		cfg := cfgStore.Get()
		state.hostTerminals.ConfigureShellPreference(cfg.HostTerminalShell, func(path string) error {
			return cfgStore.Update(func(c *config.Config) { c.HostTerminalShell = path })
		})
		i18n.SetLocale(cfg.Locale)
		applyAppearance(cfg.Theme)
		state.mu.Lock()
		state.cfgStore = cfgStore
		state.closeToTray = cfg.CloseToTray
		state.themeID = cfg.Theme
		state.mu.Unlock()

		if creds, err := credentials.Load(userData); err != nil {
			log.Println("credentials:", err)
		} else {
			state.mu.Lock()
			state.applyRemoteCredentials(creds)
			state.mu.Unlock()
		}
		if tcfg, err := sshremote.LoadTunnelConfig(userData); err != nil {
			log.Println("ssh-tunnel:", err)
		} else {
			state.mu.Lock()
			state.sshTunnel = sshremote.NewTunnelSession()
			state.sshTunnel.SetConfig(tcfg)
			state.sshHost = tcfg.Host
			state.sshUser = tcfg.User
			if tcfg.Port > 0 {
				state.sshPort = strconv.Itoa(tcfg.Port)
			} else {
				state.sshPort = "22"
			}
			state.sshIdentity = tcfg.IdentityFile
			state.sshForward = tcfg.LocalForward
			state.mu.Unlock()
		}

		if err := desktopSystemIntegration(func() error { return mygo.App.SetOpenAtLogin(cfg.OpenAtLogin) }); err != nil {
			log.Println("openAtLogin:", err)
		}

		hidden := isolatedDesktopHidden() || !isolatedDesktopRun() && mygo.App.WasOpenedAtLogin() && cfg.CloseToTray
		th := theme.Resolve(cfg.Theme)
		bgColor := th.BackgroundColor()
		winOpts := mygo.WindowOptions{
			Title:           i18n.T(i18n.KeyAppName),
			Width:           1200,
			Height:          800,
			StateKey:        "main",
			Hidden:          hidden,
			TitleBarStyle:   mygo.TitleBarHidden,
			TitleBarHeight:  int(chrome.EffectiveTitleBarHeight()),
			BackgroundColor: bgColor,
			Content:         ui.View(mainView),
		}
		// Wallpaper: config goWallpaperPath -> mainView chrome.WallpaperBackdrop
		// (WindowOptions has BackgroundColor only in mygo v0.3.5).
		win := mygo.NewWindow(winOpts)
		state.accountSignOut.SetNotify(win.Invalidate)
		win.OnClosed(func() { state.accountSignOut.Bind(nil) })
		state.desktopPet = pet.New(cfgStore)
		state.desktopPet.OnOpenSession = state.openPetSession
		state.desktopPet.OnNotificationsClosed = state.cancelPetEvents
		state.desktopPet.OnNotificationsReady = state.syncPetEvents
		state.desktopPet.OnOpenNotificationSession = state.openPetNotificationSession
		state.desktopPet.OnOpenChat = state.openPetChat
		state.desktopPet.OnOpenSettings = state.openPetSettings
		state.desktopPet.OnLook = state.lookFromPet
		state.desktopPet.OnPersonality = state.queuePetPersonality
		state.desktopPet.OnQuickChat = state.openPetQuickChat
		win.OnClosed(state.cancelPetPersonality)
		state.desktopPet.OnRefreshGrowth = state.refreshPetGrowth
		state.desktopPet.OnAutoRefreshGrowth = state.autoRefreshPetGrowth
		state.desktopPet.OnFeedTokens = state.feedPetTokens
		state.desktopPet.SetReady(!isolatedDesktopRun())
		win.OnClosed(func() { state.desktopPet.Close() })
		win.OnBlur(state.cancelComposerVoiceOnBlur)
		win.OnHide(state.cancelComposerVoice)
		win.OnClosed(state.cancelComposerVoice)
		win.OnClosed(state.whaleSettings.Suspend)
		win.OnClosed(state.stopBotNotifications)
		win.OnFocus(state.clearFocusedBotNotifications)
		win.OnBlur(state.accountWindowBlurred)
		win.OnFocus(state.returnFromAccountExternal)
		win.OnClosed(state.clearAccountReturn)
		win.OnHide(func() { state.assistantRoutines.Bind(dshbotstore.RoutineScope{}) })
		win.OnMinimize(func() { state.assistantRoutines.Bind(dshbotstore.RoutineScope{}) })
		win.OnClosed(func() { state.assistantRoutines.Bind(dshbotstore.RoutineScope{}) })
		win.OnHide(state.assistantRoomCreator.Suspend)
		win.OnHide(state.assistantBotCreator.Suspend)
		win.OnHide(state.assistantBotVisibility.Suspend)
		win.OnHide(state.assistantContacts.Suspend)
		win.OnHide(state.assistantBotDeleter.Suspend)
		win.OnHide(state.assistantBotDuplicator.Suspend)
		win.OnHide(state.assistantRoomEditor.Suspend)
		win.OnHide(state.assistantRoomRuntime.Suspend)
		win.OnMinimize(state.assistantRoomCreator.Suspend)
		win.OnMinimize(state.assistantBotCreator.Suspend)
		win.OnMinimize(state.assistantBotVisibility.Suspend)
		win.OnMinimize(state.assistantContacts.Suspend)
		win.OnMinimize(state.assistantBotDeleter.Suspend)
		win.OnMinimize(state.assistantBotDuplicator.Suspend)
		win.OnMinimize(state.assistantRoomEditor.Suspend)
		win.OnMinimize(state.assistantRoomRuntime.Suspend)
		win.OnClosed(state.assistantRoomCreator.Suspend)
		win.OnClosed(state.assistantBotCreator.Suspend)
		win.OnClosed(state.assistantBotVisibility.Suspend)
		win.OnClosed(state.assistantContacts.Suspend)
		win.OnClosed(state.assistantBotDeleter.Suspend)
		win.OnClosed(state.assistantBotDuplicator.Suspend)
		win.OnClosed(state.assistantRoomEditor.Suspend)
		win.OnClosed(state.assistantRoomRuntime.Suspend)
		win.OnHide(state.suspendMCPHealth)
		win.OnMinimize(state.suspendMCPHealth)
		win.OnClosed(state.suspendMCPHealth)
		win.OnHide(func() { state.layout.Workspace.PauseContent() })
		win.OnMinimize(func() { state.layout.Workspace.PauseContent() })
		win.OnClosed(func() { state.disposeWorkspaceSessions() })
		win.OnHide(state.hideOfficeSettings)
		win.OnClosed(state.hideOfficeSettings)
		win.OnHide(state.suspendCreatorTask)
		win.OnMinimize(state.suspendCreatorTask)
		win.OnClosed(state.suspendCreatorTask)
		win.OnHide(state.suspendAgentPresetReader)
		win.OnMinimize(state.suspendAgentPresetReader)
		win.OnClosed(state.suspendAgentPresetReader)
		win.OnHide(state.settingsProviderDiscovery.Suspend)
		win.OnMinimize(state.settingsProviderDiscovery.Suspend)
		win.OnClosed(state.settingsProviderDiscovery.Suspend)
		win.OnHide(func() {
			state.mu.Lock()
			state.fileTabs.CancelPlanRead()
			if state.sessions != nil {
				state.sessions.CloseAgentNavigation()
			}
			state.fileTabs.Pause()
			state.projectWindowHidden = true
			state.stopProjectRefreshLocked()
			state.mu.Unlock()
			state.hostTerminalUI.Observe(&state.hostTerminals, "", false)
		})
		win.OnShow(func() {
			state.mu.Lock()
			state.projectWindowHidden = false
			state.mu.Unlock()
			win.Invalidate()
		})
		win.OnClosed(func() {
			state.hostTerminals.Bind(nil, nil, false, nil)
			state.hostTerminalUI.Observe(&state.hostTerminals, "", false)
			state.mu.Lock()
			state.fileTabs.Dispose()
			if state.sessions != nil {
				state.sessions.CloseAgentNavigation()
			}
			state.stopProjectProcessLocked()
			state.stopProjectRefreshLocked()
			state.closeProjectHistoricalLocked()
			state.mu.Unlock()
		})

		win.OnClose(func(e *mygo.CloseEvent) {
			state.mu.Lock()
			quitting := state.quitting
			sup := state.sup
			closeToTray := state.closeToTray && !isolatedDesktopRun()
			state.mu.Unlock()
			if quitting {
				state.mu.Lock()
				finished := state.quitCleanupFinished
				state.mu.Unlock()
				if !finished {
					e.PreventDefault()
				}
				return
			}
			if taskguard.HideOnClose(quitting, closeToTray) {
				e.PreventDefault()
				state.hideToDesktopTray(win, sup)
				return
			}
			e.PreventDefault()
			state.requestQuitProtected(win, sup)
		})

		state.mu.Lock()
		state.win = win
		state.watchAcceptOpenHook()
		state.layout = frame.DefaultLayout()
		if cfg.SidebarWidth >= 264 {
			state.layout.Sidebar = cfg.SidebarWidth
			if state.layout.Sidebar > 420 {
				state.layout.Sidebar = 420
			}
		}
		// Desktop opens with the dialogue filling the shell; terminal is opt-in.
		state.terminalVisible = false
		state.layout.Bottom = 0
		state.terminalSavedBottom = frame.DefaultLayout().Bottom
		if cfg.TerminalHeight >= 80 {
			state.terminalSavedBottom = cfg.TerminalHeight
		}
		state.chatList = ui.ListState{FollowEnd: true}
		state.fileExpanded = map[string]bool{}
		state.planBoardOpen = false // Electron chat shell keeps the plan board collapsed by default
		state.todoPanelCollapsed = true
		state.trajActualDur = true
		state.trajDetailTab = "overview"
		state.mu.Unlock()

		state.openResume(userData)
		state.mu.Lock()
		resumeErr := state.resumeError
		state.mu.Unlock()
		if resumeErr != "" {
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Title: resumeText("恢复状态需要处理", "Resume state needs attention"), Message: resumeErr + "\n\n" + resumeText("原文件已保留，请备份后修复。当前窗口不会自动覆盖该文件。", "The original file is preserved. Back it up before repairing it. This window will not automatically overwrite it.")})
		}

		if !isolatedDesktopRun() {
			state.setupDesktopTray()
			state.startWhaleBridgeAlerts(win)
		}

		// Electron launcher.js: show launcher first; auto-start desktop only when
		// workspace is set AND autoStartDesktop is enabled (default true).
		state.mu.Lock()
		state.launcherOpen = true
		state.mu.Unlock()
		state.mu.Lock()
		state.launcherStartPreparing = true
		state.mu.Unlock()
		go func() {
			defer state.finishLauncherPreparation(win)
			canStartAfterRecovery := state.recoverImportBeforeStart()
			if state.coldStartupAllowed(cfg, canStartAfterRecovery) {
				state.mu.Lock()
				if state.quitting {
					state.mu.Unlock()
					return
				}
				state.launcherTriedStart = true
				state.launcherStartBusy = true
				state.mu.Unlock()
				startHarness(cfg)
			}
		}()
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func startHarness(cfg config.Config) {
	state.mu.Lock()
	if state.quitting {
		state.mu.Unlock()
		return
	}
	cfgStore := state.cfgStore
	startupStore := state.startupRecovery.store
	if state.launcherImportBusy {
		state.launcherStartBusy = false
		state.launcherImport = "导入尚未结束，暂不能启动内核。"
		state.mu.Unlock()
		return
	}
	state.launcherStartBusy = true
	prepareCtx, cancelPreparation := context.WithCancel(context.Background())
	state.launcherStartCancel = cancelPreparation
	state.mu.Unlock()
	defer func() {
		cancelPreparation()
		state.mu.Lock()
		state.launcherStartCancel = nil
		state.mu.Unlock()
	}()
	if !state.recoverImportBeforeStart() {
		return
	}
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 3080
	}
	// Project is mounted only with matching staged Host artifacts and a real bridge.
	var projectDispatch func(context.Context, map[string]any) map[string]any
	var projectPatches []string
	currentHost := harness.ReadCurrentHostCapabilities(paths.VendorDir(), harness.CurrentHostBridges{})
	if currentHost.ArtifactsReady && !cfg.SkipUserPlugins {
		bridge := harness.NewProjectBridge(cfg.NodeBin, paths.DshHome(), cfg.Workspace, desktopcomponents.OpenInFileManager, state.selectProjectDirectory)
		if bridge.Ready(prepareCtx) {
			overlay, err := bridge.PrepareOverlay(prepareCtx, filepath.Join(paths.VendorDir(), "dsh-project"), paths.WebProfileDir(), paths.HarnessRoot())
			if err == nil {
				projectDispatch = bridge.Dispatch
				projectPatches = append(projectPatches, overlay)
			} else {
				log.Println("Project unavailable: desktop overlay preparation failed")
			}
		}
	}
	if prepareCtx.Err() != nil {
		return
	}
	installSrv := harness.NewInstallServer(harness.InstallServerOptions{
		Project:     projectDispatch,
		HTMLPreview: helperHTMLPreview.Handler(),
		NodeBin:     cfg.NodeBin,
		Desktop:     desktopControlOps(),
		OnRestart: func() {
			state.mu.Lock()
			s := state.sup
			state.mu.Unlock()
			if s != nil {
				state.restartUnlessImport(s)
			}
		},
	})
	var sup *harness.Supervisor
	sup = harness.New(harness.Options{
		Install: installSrv,
		Launch: harness.LaunchOptions{
			PatchFiles:      projectPatches,
			Host:            host,
			Port:            port,
			Home:            paths.DshHome(),
			VendorDir:       paths.VendorDir(),
			HarnessRoot:     paths.HarnessRoot(),
			NodeBin:         cfg.NodeBin,
			DshBin:          cfg.DshBin,
			SkipUserPlugins: cfg.SkipUserPlugins,
		},
		Prepare: harness.PrepareOptions{
			DisabledPlugins: cfg.DisabledPlugins,
			RemoteSet:       true,
			Remote:          cfg.RemoteWorkspaceEnabled,
			SkipUserPlugins: cfg.SkipUserPlugins,
			Whale:           cfg.WhaleAssistantEnabled,
			Dshbot:          cfg.DshbotEnabled,
		},
		Workspace:   cfg.Workspace,
		AutoRestart: cfg.HarnessAutoRestart,
		MaxAttempts: cfg.HarnessRestartMaxAttempts,
		BaseDelay:   time.Duration(cfg.HarnessRestartBaseDelayMs) * time.Millisecond,
		ReadRestartPolicy: func() harness.RestartPolicy {
			// Store.Get returns one committed snapshot. Do not take state.mu here.
			current := cfg
			if cfgStore != nil {
				current = cfgStore.Get()
			}
			return harness.RestartPolicy{AutoRestart: current.HarnessAutoRestart, MaxAttempts: current.HarnessRestartMaxAttempts, BaseDelay: time.Duration(current.HarnessRestartBaseDelayMs) * time.Millisecond}
		},
		ReadyTimeout: 3 * time.Minute,
		OnChange:     func(st harness.Status) { state.refreshBoot(st, sup) },
		OnPluginRecovery: func(reason string, logs []string) error {
			if cfgStore == nil || startupStore == nil {
				return fmt.Errorf("startup recovery storage is unavailable")
			}
			if err := cfgStore.RecordAutomaticPluginRecovery(buildVersion, reason, logs); err != nil {
				return err
			}
			return startupStore.Record(false, reason, logs, buildVersion)
		},
	})
	if strings.TrimSpace(cfg.Workspace) == "" {
		state.mu.Lock()
		state.launcherOpen = true
		state.launcherStartBusy = false
		state.launcherStartFailed = true
		state.launcherStartErr = i18n.T(i18n.KeyLauncherNoWorkspace)
		state.mu.Unlock()
		state.refreshBoot(harness.Status{
			State: harness.Failed,
			Err:   fmt.Errorf("workspace not configured"),
		})
		return
	}
	if st, err := os.Stat(cfg.Workspace); err != nil || !st.IsDir() {
		state.refreshBoot(harness.Status{
			State: harness.Failed,
			Err:   fmt.Errorf("workspace missing: %s", cfg.Workspace),
		})
		return
	}
	// Publish on the window queue after preparation, so Close/import cannot
	// race a late background bridge result into starting a new runtime.
	launch := func() {
		state.mu.Lock()
		if state.quitting || state.launcherImportBusy {
			state.launcherStartBusy = false
			state.mu.Unlock()
			return
		}
		state.sup = sup
		state.startupRecovery.recorded = false
		state.mu.Unlock()
		sup.Start()
	}
	state.mu.Lock()
	win := state.win
	state.mu.Unlock()
	if win != nil {
		win.Update(launch)
	} else {
		launch()
	}
}

func (s *mainState) finishLauncherPreparation(win *mygo.Window) {
	s.mu.Lock()
	s.launcherStartPreparing = false
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func applyAppearance(id string) {
	switch theme.Resolve(id).Scheme {
	case theme.SchemeDark:
		mygo.Theme.SetSource(mygo.ThemeDark)
	case theme.SchemeLight:
		mygo.Theme.SetSource(mygo.ThemeLight)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

const gitRefreshInterval = 3 * time.Second
const prRefreshInterval = 30 * time.Second

func (s *mainState) refreshGit(workspaceRoot string) {
	root := strings.TrimSpace(workspaceRoot)
	s.mu.Lock()
	stale := root != s.gitCheckedRoot || time.Since(s.gitCheckedAt) >= gitRefreshInterval
	s.mu.Unlock()
	if !stale {
		return
	}
	branch, dirty := "", false
	if root != "" {
		if snap, err := desktopgit.WorkspaceSnapshot(root); err == nil {
			branch, dirty = snap.Branch, snap.Dirty
		}
	}
	s.mu.Lock()
	s.gitCheckedRoot = root
	s.gitCheckedAt = time.Now()
	s.gitBranch = branch
	s.gitDirty = dirty
	s.mu.Unlock()
}

func gitStatusLine(actionOK, actionFail string, out string, err error) string {
	if err != nil {
		msg := actionFail
		if out != "" {
			msg = firstLine(out)
		} else if err != nil {
			msg = err.Error()
		}
		return msg
	}
	if out != "" {
		return firstLine(out)
	}
	return actionOK
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func (s *mainState) runGitFetch(root string) {
	s.runGitRemote("fetch", root, func(ctx context.Context) (string, error) {
		return desktopgit.FetchContext(ctx, root)
	}, i18n.T(i18n.KeyGitFetchOK), i18n.T(i18n.KeyGitActionFailed))
}

func (s *mainState) runGitPull(root string) {
	s.runGitRemote("pull", root, func(ctx context.Context) (string, error) {
		return desktopgit.PullContext(ctx, root)
	}, i18n.T(i18n.KeyGitPullOK), i18n.T(i18n.KeyGitActionFailed))
}

func (s *mainState) runGitPush(root string) {
	s.reviewGitPush(root)
}

func (s *mainState) runGitCommit(root, message, reviewedToken string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	s.runGitRemote("commit", root, func(ctx context.Context) (string, error) {
		return desktopgit.CommitReviewed(root, message, reviewedToken)
	}, i18n.T(i18n.KeyGitCommitOK), i18n.T(i18n.KeyGitActionFailed))
}

func (s *mainState) runGitPRView(root string) {
	s.runGitRemote("pr-view", root, func(ctx context.Context) (string, error) {
		st := desktopgit.ViewPR(ctx, root)
		if !st.OK {
			return "", fmt.Errorf("%s", st.Err)
		}
		if st.URL != "" {
			return st.URL, nil
		}
		return st.Summary, nil
	}, i18n.T(i18n.KeyGitPRView), i18n.T(i18n.KeyGitActionFailed))
}

func (s *mainState) runGitPRCheckout(root string, number int) {
	s.runGitRemote("pr-checkout", root, func(ctx context.Context) (string, error) {
		st := desktopgit.CheckoutPR(ctx, root, number)
		if !st.OK {
			return "", fmt.Errorf("%s", st.Err)
		}
		s.mu.Lock()
		s.gitPRAt = time.Time{}
		s.gitCheckedAt = time.Time{}
		s.mu.Unlock()
		if st.Summary != "" {
			return st.Summary, nil
		}
		return fmt.Sprintf("#%d", number), nil
	}, i18n.T(i18n.KeyGitPRCheckoutOK), i18n.T(i18n.KeyGitActionFailed))
}

func (s *mainState) refreshPR(workspaceRoot string, force bool) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return
	}
	s.mu.Lock()
	if s.gitPRBusy {
		s.mu.Unlock()
		return
	}
	stale := force || root != s.gitPRRoot || time.Since(s.gitPRAt) >= prRefreshInterval
	if !stale {
		s.mu.Unlock()
		return
	}
	s.gitPRBusy = true
	win := s.win
	s.mu.Unlock()

	go func() {
		rep := desktopgit.PRStatusReport(context.Background(), root)
		s.mu.Lock()
		s.gitPRBusy = false
		s.gitPRReport = rep
		s.gitPRRoot = root
		s.gitPRAt = time.Now()
		if rep.MissingGH {
			s.gitMsg = i18n.T(i18n.KeyGitPRNoGH)
		} else if rep.Err != "" && !rep.OK {
			s.gitMsg = firstLine(rep.Err)
		}
		win = s.win
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
	}()
}

func gitPRSummary(rep desktopgit.PRReport) string {
	if rep.MissingGH {
		return ""
	}
	if rep.Current != nil {
		p := *rep.Current
		if p.Number > 0 && strings.TrimSpace(p.State) != "" {
			return fmt.Sprintf("#%d %s", p.Number, p.State)
		}
		return desktopgit.FormatPRInfo(p)
	}
	return ""
}

func (s *mainState) confirmGitCommit(win *mygo.Window, root, message string, dirty bool) {
	if !dirty {
		s.mu.Lock()
		s.gitMsg = i18n.T(i18n.KeyGitCommitNotDirty)
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
		return
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	_, reviewedToken, reviewErr := desktopgit.IndexReview(root)
	if reviewErr != nil {
		s.mu.Lock()
		s.gitMsg = reviewErr.Error()
		s.mu.Unlock()
		return
	}
	summary, reviewErr := desktopgit.StagedSummary(root)
	if reviewErr != nil {
		s.mu.Lock()
		s.gitMsg = reviewErr.Error()
		s.mu.Unlock()
		return
	}
	res, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent:        win,
		Type:          mygo.MessageQuestion,
		Title:         i18n.T(i18n.KeyGitCommitConfirmTitle),
		Message:       "Commit only these staged changes / 仅提交以下已暂存变更：\n" + summary + "\n\nUnstaged files and editor drafts are excluded; no push. / 不含未暂存文件或编辑器草稿，不会推送。",
		Buttons:       []string{i18n.T(i18n.KeyGitCommitCancel), i18n.T(i18n.KeyGitCommit)},
		DefaultButton: 0,
		CancelButton:  0,
	})
	if err != nil || res.Button != 1 {
		return
	}
	s.mu.Lock()
	s.gitCommitOpen = false
	s.gitCommitDraft = ""
	s.mu.Unlock()
	s.runGitCommit(root, message, reviewedToken)
}

func (s *mainState) runGitRemote(action string, root string, run func(context.Context) (string, error), okLabel, failLabel string) {
	root = strings.TrimSpace(root)
	if root == "" {
		return
	}
	s.mu.Lock()
	if s.gitBusy {
		s.mu.Unlock()
		return
	}
	s.gitBusy = true
	s.gitMsg = ""
	win := s.win
	sessions := s.sessions
	progressWindow := win
	progress := desktopgit.NewProgress(root, action+"…", func() {
		if progressWindow != nil {
			progressWindow.Invalidate()
		}
	})
	s.gitProgress = progress
	s.mu.Unlock()

	go func() {
		ctx, cancel := context.WithTimeout(desktopgit.WithProgress(context.Background(), progress), desktopgit.DefaultCommandTimeout)
		defer cancel()
		out, err := run(ctx)
		msg := gitStatusLine(okLabel, failLabel, out, err)
		progress.Finish(msg, err != nil)

		s.mu.Lock()
		s.gitBusy = false
		s.gitMsg = msg
		s.gitCheckedAt = time.Time{}
		win = s.win
		sessions = s.sessions
		s.mu.Unlock()

		if sessions != nil {
			sessions.SetLastAction(msg)
		}
		if win != nil {
			win.Update(func() {})
		}
		s.refreshGit(root)
		if win != nil {
			win.Update(func() {})
		}
	}()
}

func (s *mainState) hydrateViewTabsFromHarness(setStore *settingsstore.Store) {
	if setStore == nil {
		return
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		on, ok := setStore.ViewTabsEnabled()
		if !ok {
			time.Sleep(40 * time.Millisecond)
			continue
		}
		s.mu.Lock()
		s.desktopDraft.showViewTabs = on
		cfgStore := s.cfgStore
		win := s.win
		s.mu.Unlock()
		if cfgStore != nil {
			_ = cfgStore.Update(func(c *config.Config) {
				c.HideChatViewTabs = !on
			})
		}
		if win != nil {
			win.Update(func() {})
		}
		return
	}
}

func (s *mainState) applyImportCatalog(win *mygo.Window, sourceHome string) {
	s.mu.Lock()
	busy := s.launcherImportBusy
	s.mu.Unlock()
	if busy {
		return
	}
	src := strings.TrimSpace(sourceHome)
	if src == "" {
		src = importer.OfficialDshHome()
		if src == "" {
			src = filepath.Join(importer.ElectronUserDataDir(), "dsh-home")
		} else if st, err := os.Stat(src); err != nil || !st.IsDir() {
			src = filepath.Join(importer.ElectronUserDataDir(), "dsh-home")
		}
	}
	cat, err := importer.BuildCatalog(src, paths.DshHome())
	s.mu.Lock()
	if err != nil {
		s.launcherImport = err.Error()
		s.launcherImportCategories = nil
		s.launcherImportCatalog = nil
	} else {
		s.launcherImportCatalog = &cat
		s.launcherImportAttachments = false
		if cat.HasAttachments {
			for _, item := range cat.Sessions {
				if !item.Unsupported {
					s.launcherImportAttachments = true
					break
				}
			}
		}
		s.launcherImport = fmt.Sprintf(i18n.T(i18n.KeyLauncherImportSummary),
			cat.SourceHome, cat.DestHome,
			len(cat.Sessions), len(cat.Skills), len(cat.Plugins),
			len(cat.MCP), len(cat.Settings), len(cat.Presets))
		counts := cat.CategoryCounts()
		s.launcherImportCategories = map[string]bool{}
		for id, pair := range counts {
			s.launcherImportCategories[id] = pair[1] > 0
		}
	}
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func marketInstallRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func mainView(c *ui.Context) {
	state.flushInactiveGeneralFontEdits(false)
	defer state.syncSettingsNativeSurfaces()
	composerRendered := false
	composerSessionID := ""
	defer func() { state.observeComposerEdit(composerRendered, composerSessionID) }()
	defer state.syncAccountBonusNotice(c)
	defer state.syncAccountStatusNotices(c)
	defer state.syncAccountQuotaNotice(c)
	state.beginWallpaperFrame()
	defer state.endWallpaperFrame()
	state.registerApplicationCommands(c)
	state.syncApplicationMenu(c)
	state.beginAppearanceFrame(c)
	state.configurePointerEffects(c)
	defer state.endAppearanceFrame()
	defer state.observeAssistantBotRoute()
	roomRuntimeRendered := false
	defer func() { state.observeAssistantRoomRuntime(roomRuntimeRendered) }()
	roomEditRendered := false
	defer func() { state.observeAssistantRoomEditor(roomEditRendered) }()
	roomCreateRendered := false
	defer func() { state.observeAssistantRoomCreator(roomCreateRendered) }()
	defer func() { state.observeAssistantBotCreator(roomCreateRendered) }()
	defer func() { state.observeAssistantBotVisibility(roomCreateRendered) }()
	defer func() { state.observeAssistantContacts(roomCreateRendered) }()
	defer func() { state.observeAssistantBotDeleter(roomCreateRendered) }()
	defer func() { state.observeAssistantBotDuplicator(roomCreateRendered) }()
	defer func() { state.observeBotNotifications(roomCreateRendered) }()
	creatorTaskRendered := false
	defer func() { state.observeCreatorTask(creatorTaskRendered) }()
	defer func() { state.observeAgentPresetReader(creatorTaskRendered) }()
	providerDiscoveryRendered := false
	defer func() { state.observeProviderDiscovery(providerDiscoveryRendered) }()
	mcpPanelRendered := false
	defer func() { state.observeMCPHealth(mcpPanelRendered) }()
	state.observeWhaleSettings()
	state.cancelComposerVoiceIfStale()
	state.restoreResumeSelection()
	state.refreshPetTask()
	state.mu.Lock()
	if state.market == nil {
		state.market = marketstore.New()
	}
	themeID := state.themeID
	st := state.bootStatus
	logs := append([]string(nil), state.bootLogs...)
	sup := state.sup
	sessions := state.sessions
	nextPromptRendered := false
	defer func() {
		if !nextPromptRendered && sessions != nil {
			sessions.SyncNextPrompt(context.Background(), false)
			sessions.UnmountNativeJobs()
		}
	}()
	interactions := state.interactions
	if state.settings == nil {
		state.settings = settingsstore.New()
		state.settings.SetVisualReplyEnableCheck(state.verifyVisualReplyEnable)
	}
	setStore := state.settings
	settingsOpen := state.settingsOpen
	usageStore := state.usage
	imStore := state.im
	marketStore := state.market
	schedStore := state.schedule
	usageOpen := state.usageOpen
	accountActionError := state.accountReturn.err
	imOpen := state.imOpen
	marketOpen := state.marketOpen
	marketManager := state.marketManager
	scheduleOpen := state.scheduleOpen
	marketInstallSpecPtr := &state.marketInstallSpec
	marketInstallBusy := state.marketInstallBusy
	marketCanCancel := state.marketInstallBusy && state.marketInstallRequestID != ""
	marketInstallMsg := state.marketInstallMsg
	marketInstallLog := state.marketInstallLog
	marketActivation := state.marketActivation
	marketCanRestart := state.sup != nil && state.marketWorker == nil && !state.taskRestartPending && st.State != harness.Starting
	marketNeedsAllowBuilds := state.marketNeedsAllowBuilds
	marketAllowBuilds := append([]string(nil), state.marketAllowBuilds...)
	marketIncompatible := append([]api.ChangeResultErrorIncompatibleItem(nil), state.marketIncompatible...)
	remoteOpen := state.remoteOpen
	nodeBin := ""
	hasRemoteToken := state.hasRemoteToken
	hasRemoteRelayToken := state.hasRemoteRelayToken
	remoteDeviceCount := state.remoteDeviceCount
	remoteDeviceNames := append([]string(nil), state.remoteDeviceNames...)
	phoneActionMsg := state.phoneActionMsg
	webviewOpen := state.webviewOpen
	browserPreviewOpen := state.browserPreviewOpen
	assistantOpen := state.assistantOpen
	if !assistantOpen || st.State != harness.Ready {
		state.assistantBotCreator.Suspend()
		state.assistantBotVisibility.Suspend()
		state.assistantContacts.Suspend()
		state.assistantBotDeleter.Suspend()
		state.assistantBotDuplicator.Suspend()
		state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
		state.assistantRoomCreator.Suspend()
		state.assistantRoomEditor.Suspend()
		state.assistantRoomRuntime.Suspend()
	}
	launcherOpen := state.launcherOpen
	toolsMenuOpen := state.toolsMenuOpen
	launcherTab := state.launcherTab
	if launcherTab == "" {
		launcherTab = launcher.TabHome
	}
	launcherImport := state.launcherImport
	launcherImportJournal := append([]string(nil), state.launcherImportJournal...)
	launcherImportBusy := state.launcherImportBusy
	launcherUpdate := state.launcherUpdate
	launcherUpdateAvailable := state.launcherUpdateAvailable
	launcherNote := state.launcherNote
	launcherStartBusy := state.launcherStartBusy
	launcherStartFailed := state.launcherStartFailed
	launcherStartErr := state.launcherStartErr
	cfgHideArchivedList := false
	cfgRemoteWorkspace := false
	cfgPhoneGitEnabled := state.phoneGitSession != ""
	cfgRemoteEnabled := false
	cfgRemoteMode := ""
	cfgRemotePort := 0
	cfgRemoteBind := ""
	cfgRemoteLanTls := false
	cfgRemoteAppBase := ""
	cfgRemoteRelayEndpoint := ""
	cfgRemoteRelayUseTLS := false
	wallpaperPath := ""
	imPluginDisabled := false
	marketPluginDisabled := false
	if state.cfgStore != nil {
		cfg := state.cfgStore.Get()
		nodeBin = cfg.NodeBin
		cfgHideArchivedList = cfg.HideArchivedList
		cfgRemoteWorkspace = cfg.RemoteWorkspaceEnabled
		cfgRemoteEnabled = cfg.RemoteEnabled
		cfgRemoteMode = cfg.RemoteMode
		cfgRemotePort = cfg.RemotePort
		cfgRemoteBind = cfg.RemoteBindAddress
		cfgRemoteLanTls = cfg.RemoteLanTls
		cfgRemoteAppBase = cfg.RemoteAppBaseUrl
		cfgRemoteRelayEndpoint = cfg.RemoteRelayEndpoint
		cfgRemoteRelayUseTLS = cfg.RemoteRelayUseTls
		wallpaperPath, _ = theme.WindowWallpaperPath(cfg.WallpaperPath)
		for _, p := range cfg.DisabledPlugins {
			if p == "dsh-im" {
				imPluginDisabled = true
			}
			if p == "ui-settings-market" || p == "dsh-client-ui-settings-market" {
				marketPluginDisabled = true
			}
		}
	}
	state.selectWorkspaceSessionLocked()
	marketInstallReady := desktopmarket.InstallReady(nodeBin)
	if state.layout.Workspace == nil {
		state.layout.Workspace = &frame.WorkspacePanes{}
	}
	layout := state.layout
	win := state.win
	// The very first view build happens inside mygo.NewWindow (main.go:2232),
	// before state.win is assigned at main.go:2358, so win is still nil there.
	// Resolve the native parent defensively instead of dereferencing it: the
	// OpenUI native surface seat is skipped while it is 0 (chat/openui.go) and
	// attaches on the next render, once the window exists.
	var openUIParent uintptr
	if win != nil {
		openUIParent = win.NativeHandle()
	}
	chatList := state.chatList
	draft := state.draft
	editMessageSeq := state.editMessageSeq
	feedbackCommentMsgID := state.feedbackCommentMsgID
	feedbackCommentDraft := state.feedbackCommentDraft
	modelPick := state.modelPick
	permissionPick := state.permissionPick
	agentPresetPick := state.agentPresetPick
	thinkingExpanded := state.thinkingExpanded
	renameSessionID := state.renameSessionID
	renameDraft := state.renameDraft
	sessionSearch := state.sessionSearch
	gitBranch := state.gitBranch
	gitDirty := state.gitDirty
	gitMsg := state.gitMsg
	gitProgress := state.gitProgress
	gitBusy := state.gitBusy
	gitCommitOpen := state.gitCommitOpen
	gitCommitDraft := state.gitCommitDraft
	gitPROpen := state.gitPROpen
	gitPRReport := state.gitPRReport
	gitPRBusy := state.gitPRBusy
	fileTabs := state.fileTabs
	gitWorkbench := state.gitWorkbench
	fileExpanded := state.fileExpanded
	selectedFile := state.selectedFile
	fileFilter := state.fileFilter
	filePreview := state.filePreview
	filePreviewErr := state.filePreviewErr
	filePreviewList := state.filePreviewList
	fileEditing := state.fileEditing
	fileEditDraft := state.fileEditDraft
	fileEditStatus := state.fileEditStatus
	fileEditViewMode := state.fileEditViewMode
	fileEditSplit := state.fileEditSplit
	if fileEditSplit <= 0 {
		fileEditSplit = 200
	}
	rightPaneTab := state.rightPaneTab
	agentsPromptDraft := state.agentsPromptDraft
	goalDraft := state.goalDraft
	fileOpenWithApps := append([]files.OpenWithApp(nil), state.fileOpenWithApps...)
	diffPath := state.diffPath
	diffText := state.diffText
	diffErr := state.diffErr
	planBoardOpen := state.planBoardOpen
	todoPanelCollapsed := state.todoPanelCollapsed
	chatPaneTab := state.chatPaneTab
	if chatPaneTab == "" {
		chatPaneTab = "chat"
	}
	trajSearch := state.trajSearch
	trajSelectedID := state.trajSelectedID
	trajActualDur := state.trajActualDur
	trajTurnsFold := state.trajTurnsFold
	trajCallsFold := state.trajCallsFold
	trajDetailTab := state.trajDetailTab
	if trajDetailTab == "" || trajDetailTab == "overview" || trajDetailTab == i18n.T(i18n.KeyTrajTabOverview) {
		trajDetailTab = "overview"
	}
	imageLightboxPath := state.imageLightboxPath
	workspaceRoot := ""
	if state.cfgStore != nil {
		workspaceRoot = state.cfgStore.Get().Workspace
	}
	// A selected session owns its actual directory, including Host scratch sessions.
	if sessions != nil {
		if cwd := sessions.Snapshot().SelectedCwd; cwd != "" {
			workspaceRoot = cwd
		}
	}
	// Refresh open-with apps when the selected file changes (Electron workspacePathApplications).
	if selectedFile != "" && selectedFile != state.fileOpenWithFor && sessions != nil && st.State == harness.Ready {
		rel := selectedFile
		abs, err := workspacefs.JoinRoot(workspaceRoot, rel)
		if err == nil && abs != "" {
			state.fileOpenWithFor = rel
			go func() {
				apps, err := sessions.ListWorkspacePathApplications(context.Background(), abs)
				state.mu.Lock()
				if state.fileOpenWithFor != rel {
					state.mu.Unlock()
					return
				}
				state.fileOpenWithApps = state.fileOpenWithApps[:0]
				if err == nil {
					for _, a := range apps {
						name := strings.TrimSpace(a.Name)
						if name == "" {
							name = a.Id
						}
						state.fileOpenWithApps = append(state.fileOpenWithApps, files.OpenWithApp{ID: a.Id, Name: name})
					}
				}
				state.mu.Unlock()
				if win != nil {
					win.Update(func() {})
				}
			}()
		}
	} else if selectedFile == "" && state.fileOpenWithFor != "" {
		state.fileOpenWithFor = ""
		state.fileOpenWithApps = nil
		fileOpenWithApps = nil
	}
	state.mu.Unlock()
	imOnboardingRendered := false
	defer func() {
		if !imOnboardingRendered && imStore != nil && imStore.Onboarding.Snapshot().Open {
			imStore.Onboarding.Hide()
		}
	}()
	if !imOpen || launcherOpen || settingsOpen || marketOpen || scheduleOpen || remoteOpen || webviewOpen || browserPreviewOpen || assistantOpen || usageOpen {
		state.hideOfficeSettings()
	}
	// Global native pages return before frame.View. Retire old bindings and
	// pause debounce work here as well, without flushing hidden drafts.
	fileSessionID := ""
	if sessions != nil {
		fileSessionID = sessions.Snapshot().SelectedID
	}
	state.dshMu.Lock()
	fileClient := state.dshClient
	fileMux := state.dshMux
	state.dshMu.Unlock()
	fileBindingID := fmt.Sprintf("%p\x00%s", fileClient, fileSessionID)
	layout.Workspace.BeginContentFrame(fileBindingID, workspaceRoot)
	defer func() { layout.Workspace.EndContentFrame() }()
	fileTabs.Bind(fileBindingID, workspaceRoot)
	if sessions == nil || launcherOpen || usageOpen || imOpen || marketOpen || scheduleOpen || remoteOpen || webviewOpen || browserPreviewOpen || assistantOpen {
		state.hostTerminalUI.Observe(&state.hostTerminals, "", false)
		fileTabs.CancelPlanRead()
		if sessions != nil {
			sessions.CloseAgentNavigation()
		}
		fileTabs.Pause()
	}
	state.mu.Lock()
	state.fileTabs = fileTabs
	state.mu.Unlock()
	if state.browserController != nil {
		state.browserController.PrunePDFDocuments()
		state.browserController.PruneMediaDocuments()
		state.browserController.PruneHTMLDocuments()
		state.browserController.BeginFrame()
	}
	defer func() {
		if state.browserController != nil {
			state.browserController.EndFrame()
		}
	}()
	if !webviewOpen && !browserPreviewOpen {
		state.parkPluginViewForSettings(settingsOpen)
	}
	state.refreshGit(workspaceRoot)
	if gitBranch != "" && workspaceRoot != "" {
		state.refreshPR(workspaceRoot, false)
	}
	state.mu.Lock()
	gitBranch = state.gitBranch
	gitDirty = state.gitDirty
	gitMsg = state.gitMsg
	gitPROpen = state.gitPROpen
	gitPRReport = state.gitPRReport
	gitPRBusy = state.gitPRBusy
	state.mu.Unlock()
	th := theme.Resolve(themeID)
	systemDark := mygo.Theme.IsDark()
	colors := th.Active(systemDark)
	var gitBadge *chrome.GitBadge
	if gitBranch != "" {
		gitBadge = &chrome.GitBadge{
			Branch:      gitBranch,
			Dirty:       gitDirty,
			FetchLabel:  i18n.T(i18n.KeyGitFetch),
			PullLabel:   i18n.T(i18n.KeyGitPull),
			PushLabel:   i18n.T(i18n.KeyGitPush),
			CommitLabel: i18n.T(i18n.KeyGitCommit),
			StatusText:  gitMsg,
		}
		gitBadge.PRSummary = gitPRSummary(gitPRReport)
		gitBadge.PRPanelLabel = i18n.T(i18n.KeyGitPRPanel)
		gitBadge.PRViewLabel = i18n.T(i18n.KeyGitPRView)
		gitBadge.PRCreateLabel = i18n.T(i18n.KeyGitPRCreate)
		if st.State == harness.Ready && workspaceRoot != "" && !gitBusy {
			root := workspaceRoot
			gitBadge.OnFetch = func() { state.runGitFetch(root) }
			gitBadge.OnPull = func() { state.runGitPull(root) }
			gitBadge.OnPush = func() { state.runGitPush(root) }
			gitBadge.OnCommit = func() {
				state.mu.Lock()
				state.gitCommitOpen = false
				state.filesPaneOpen = true
				state.rightPaneTab = "diff"
				rightPaneTab = "diff"
				gitCommitOpen = false
				state.gitPROpen = false
				state.mu.Unlock()
				gitWorkbench.RequestCommit(root, gitCommitDraft)
				if win != nil {
					win.Update(func() {})
				}
			}
			gitBadge.OnPRPanel = func() {
				gitPROpen = !gitPROpen
				state.mu.Lock()
				state.gitPROpen = gitPROpen
				state.gitCommitOpen = false
				open := gitPROpen
				state.mu.Unlock()
				gitCommitOpen = false
				if open {
					state.refreshPR(root, true)
				}
				if win != nil {
					win.Update(func() {})
				}
			}
			gitBadge.OnPRView = func() { state.runGitPRView(root) }
			gitBadge.OnPRCreate = func() {
				state.mu.Lock()
				state.filesPaneOpen = true
				state.rightPaneTab = "diff"
				state.gitPROpen = false
				state.mu.Unlock()
				rightPaneTab, gitPROpen = "diff", false
				gitWorkbench.RequestPR(root)
				if win != nil {
					win.Invalidate()
				}
			}
		}
	}
	titleColors := chrome.TitleBarColorsFrom(colors)
	// Electron desk caption is white/transparent on light; dark keeps theme bg
	// so the strip under traffic lights matches the desk wash.
	if !systemDark {
		titleColors.Background = chrome.Color("#ffffff")
	}
	ui.Column(c).Fill().Background(chrome.Color(colors.BG)).Draw(state.wallpaperPainter(wallpaperPath, !launcherOpen, c.Preferences().ReduceMotion)).Children(func() {
		if launcherOpen {
			// The pinned native release catalog owns the one background lookup.
			// Full-window launcher shell (Electron parity): no main chrome.
			vendorReady := false
			if info, err := os.Stat(filepath.Join(paths.VendorDir(), "deepseek-harness", "apps", "cli", "lib", "bin.js")); err == nil && !info.IsDir() {
				vendorReady = true
			}
			disabled := []string(nil)
			skipUserPlugins := false
			pluginRecoveryReason := ""
			cfgStore := state.cfgStore
			harnessSup := sup
			if cfgStore != nil {
				cfg := cfgStore.Get()
				disabled = append([]string(nil), cfg.DisabledPlugins...)
				rec := cfgStore.PluginRecoveryEvidence()
				skipUserPlugins = rec.SkipUserPlugins
				pluginRecoveryReason = rec.Reason
			}
			if strings.TrimSpace(state.launcherWorkspaceDraft) == "" {
				seed := workspaceRoot
				if seed == "" && cfgStore != nil {
					seed = cfgStore.Get().Workspace
				}
				state.launcherWorkspaceDraft = seed
			}
			wsDraft := &state.launcherWorkspaceDraft
			if !state.launcherPrefsSeeded && cfgStore != nil {
				cfg := cfgStore.Get()
				state.launcherQuitAfterStart = cfg.QuitAfterStart
				state.launcherAutoStartDesktop = cfg.AutoStartDesktop
				state.launcherAskOnUpdate = cfg.AskOnUpdate
				state.launcherDownloadRoute = cfg.DownloadRoute
				state.closeToTray = cfg.CloseToTray
				state.launcherPrefsSeeded = true
			}
			state.mu.Lock()
			componentsFresh := state.launcherComponentsFresh
			state.mu.Unlock()
			if !componentsFresh {
				state.refreshLauncherComponents()
			}
			state.mu.Lock()
			compRows := append([]desktopcomponents.Row(nil), state.launcherComponents...)
			prefsError := state.launcherPrefsError
			state.mu.Unlock()
			componentActivity := state.launcherComponentActivitySnapshot()
			quitPtr := &state.launcherQuitAfterStart
			trayPtr := &state.closeToTray
			autoPtr := &state.launcherAutoStartDesktop
			askPtr := &state.launcherAskOnUpdate
			routePtr := &state.launcherDownloadRoute
			// Electron「桌面端已就绪」means the desk shell window is up, not only
			// that the harness has reached Ready (MyGo keeps desk visible while Starting).
			deskRunning := st.State == harness.Ready || sessions != nil
			launcher.View(c, state.withNativeReleases(win, launcher.Model{
				ThemeID:         themeID,
				SystemDark:      mygo.Theme.IsDark(),
				Version:         buildVersion,
				Workspace:       *wsDraft,
				WorkspacePtr:    wsDraft,
				VendorReady:     vendorReady,
				LauncherPackage: isLauncherPackage(),
				DesktopRunning:  deskRunning,
				// Keep the running chrome even while harness is still Starting.
				StartBusy:            launcherStartBusy && !deskRunning,
				StartFailed:          launcherStartFailed,
				StartError:           launcherStartErr,
				BootLogs:             logs,
				PluginLoad:           st.PluginLoad,
				ActiveTab:            launcherTab,
				ImportSummary:        launcherImport,
				ImportJournal:        launcherImportJournal,
				UpdateStatus:         launcherUpdate,
				UpdateAvailable:      launcherUpdateAvailable,
				DisabledPlugins:      disabled,
				SkipUserPlugins:      skipUserPlugins,
				PluginRecoveryReason: pluginRecoveryReason,
				StatusNote:           launcherNote,
				QuitAfterStart:       quitPtr,
				CloseToTray:          trayPtr,
				AutoStartDesktop:     autoPtr,
				AskOnUpdate:          askPtr,
				DownloadRoute:        routePtr,
				RoutePopOpen:         &state.launcherRoutePopOpen,
				ImportOverwrite:      &state.launcherImportOverwrite,
				ImportAttachments:    &state.launcherImportAttachments,
				ImportCategories:     state.launcherImportCategories,
				ImportCatalog:        state.launcherImportCatalog,
				Components:           compRows,
				ComponentPending:     componentActivity.Pending,
				ComponentStatus:      componentActivity.Status,
				LauncherPrefsError:   prefsError,
				ComponentBusy:        bridgeChangeActive.Load(),
				HostRuntimeState:     st.State,
				OnRefreshComponents: func() {
					state.refreshLauncherComponents()
					if win != nil {
						win.Update(func() {})
					}
				},
				OnComponentAction: func(id, action string) {
					if bridgeChangeActive.Load() {
						return
					}
					if id == desktopcomponents.HostComponentID {
						state.changeHostBundle(win, action)
						return
					}
					if id == "whalebridge" && (action == "update" || action == "rollback" || action == "uninstall" || action == "stop") {
						state.changeWhaleBridge(win, action)
						return
					}
					if id == "whalebridge" && action == "start" {
						release, ok := state.claimLauncherComponentAction(win, id, action)
						if !ok {
							return
						}
						state.mu.Lock()
						state.launcherUpdate = "正在启动鲸桥并验证就绪状态…"
						state.launcherComponentStatus = state.launcherUpdate
						state.mu.Unlock()
						go func() {
							defer release()
							err := desktopcomponents.RunAction(paths.UserDataDir(), id, action)
							state.mu.Lock()
							state.launcherUpdate = desktopcomponents.FormatActionError(id, action, err)
							state.launcherComponentStatus = state.launcherUpdate
							state.mu.Unlock()
							state.refreshLauncherComponents()
							if win != nil {
								win.Update(func() {})
							}
						}()
						return
					}
					if id == "whalebridge" && action == "install" {
						release, ok := state.claimLauncherComponentAction(win, id, action)
						if !ok {
							return
						}
						route := state.updateDownloadRoute()
						state.mu.Lock()
						state.launcherUpdate = "正在下载并校验鲸桥组件…"
						state.launcherComponentStatus = state.launcherUpdate
						state.mu.Unlock()
						go func() {
							defer release()
							err := desktopcomponents.InstallWhaleBridge(context.Background(), paths.UserDataDir(), route)
							state.mu.Lock()
							state.launcherUpdate = desktopcomponents.FormatActionError(id, action, err)
							state.launcherComponentStatus = state.launcherUpdate
							if err == nil {
								state.launcherUpdate = "鲸桥已安装并校验，可点击启动；未自动运行"
								state.launcherComponentStatus = state.launcherUpdate
							}
							state.mu.Unlock()
							state.refreshLauncherComponents()
							if win != nil {
								win.Update(func() {})
							}
						}()
						return
					}
					state.runGenericComponentAction(win, id, action)
				},
				OnSaveLauncherPrefs: func() {
					if cfgStore == nil {
						return
					}
					state.mu.Lock()
					quit := state.launcherQuitAfterStart
					tray := state.closeToTray
					auto := state.launcherAutoStartDesktop
					ask := state.launcherAskOnUpdate
					route := state.launcherDownloadRoute
					state.mu.Unlock()
					err := cfgStore.Update(func(c *config.Config) {
						c.QuitAfterStart = quit
						c.CloseToTray = tray
						c.AutoStartDesktop = auto
						c.AskOnUpdate = ask
						c.DownloadRoute = route
					})
					state.mu.Lock()
					state.launcherPrefsError = ""
					if err != nil {
						state.launcherPrefsError = err.Error()
					}
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
				OnTab: func(tab string) {
					state.mu.Lock()
					state.launcherTab = tab
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
				OnClose: func() {
					state.mu.Lock()
					state.launcherOpen = false
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
				OnStop: func() {
					state.mu.Lock()
					sup := state.sup
					state.mu.Unlock()
					state.stopProtected(win, sup)
				},
				OnStart: func() {
					state.mu.Lock()
					if state.launcherStartPreparing || state.launcherStartBusy || state.launcherImportBusy || state.quitting {
						state.mu.Unlock()
						return
					}
					state.launcherStartPreparing = true
					store := state.cfgStore
					ws := strings.TrimSpace(state.launcherWorkspaceDraft)
					state.launcherTriedStart = true
					state.launcherStartBusy = true
					state.launcherStartErr = ""
					state.startupRecovery.updateHold = false
					state.mu.Unlock()
					go func() {
						defer state.finishLauncherPreparation(win)
						if store == nil {
							state.mu.Lock()
							state.launcherStartBusy = false
							state.launcherStartFailed = true
							state.launcherStartErr = "config store is not ready"
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
							return
						}
						if ws == "" {
							ws = paths.DefaultWorkspace()
						}
						if err := os.MkdirAll(ws, 0o755); err != nil {
							state.mu.Lock()
							state.launcherStartBusy = false
							state.launcherStartFailed = true
							state.launcherStartErr = err.Error()
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
							return
						}
						if err := store.Update(func(c *config.Config) {
							c.Workspace = ws
						}); err != nil {
							state.mu.Lock()
							state.launcherStartBusy = false
							state.launcherStartFailed = true
							state.launcherStartErr = err.Error()
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
							return
						}
						state.mu.Lock()
						state.launcherWorkspaceDraft = ws
						state.launcherOpen = true
						state.mu.Unlock()
						startHarness(store.Get())
						if win != nil {
							win.Update(func() {})
						}
					}()
				},
				OnScanImport: func() {
					state.applyImportCatalog(win, "")
				},
				OnPickImportSource: func() {
					picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
						Parent:    win,
						Title:     i18n.T(i18n.KeyLauncherImportPickSourceTitle),
						Directory: true,
					})
					if err != nil || len(picked) == 0 {
						return
					}
					state.applyImportCatalog(win, picked[0])
				},
				OnAddSkillImportDir: func() {
					picked, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
						Parent:    win,
						Title:     i18n.T(i18n.KeyLauncherImportAddSkillTitle),
						Directory: true,
					})
					if err != nil || len(picked) == 0 {
						return
					}
					state.mu.Lock()
					cat := state.launcherImportCatalog
					state.mu.Unlock()
					if cat == nil {
						state.applyImportCatalog(win, "")
						state.mu.Lock()
						cat = state.launcherImportCatalog
						state.mu.Unlock()
					}
					if cat == nil {
						return
					}
					n := importer.AppendSkillDirectory(cat, picked[0])
					state.mu.Lock()
					state.launcherImportCatalog = cat
					state.launcherImport = fmt.Sprintf(i18n.T(i18n.KeyLauncherImportSummarySkillDir),
						cat.SourceHome, cat.DestHome,
						len(cat.Sessions), len(cat.Skills), len(cat.Plugins),
						len(cat.MCP), len(cat.Settings), len(cat.Presets),
						picked[0], n)
					counts := cat.CategoryCounts()
					state.launcherImportCategories = map[string]bool{}
					for id, pair := range counts {
						state.launcherImportCategories[id] = pair[1] > 0
					}
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
				OnCopyImportJournal: func(lines []string) {
					if len(lines) == 0 {
						return
					}
					mygo.Clipboard.WriteText(strings.Join(lines, "\n"))
				},
				OnStartImport: func() { state.startImport(win) },
				ImportBusy:    launcherImportBusy,
				OnCancelImport: func() {
					state.mu.Lock()
					cancel := state.launcherImportCancel
					state.mu.Unlock()
					if cancel != nil {
						cancel()
					}
				},
				OnInstallUpdate: func() {
					// Electron #btn-home-update → check + download/stage (same as settings update).
					state.showUpdateCheck(win)
					res, err := desktopupdate.CheckForUpdatesVersionRoute(context.Background(), buildVersion, state.updateDownloadRoute())
					state.mu.Lock()
					if err != nil {
						state.launcherUpdateAvailable = false
					} else {
						state.launcherUpdateAvailable = res.Status == desktopupdate.StatusUpdateAvailable
						if state.launcherUpdateAvailable {
							state.launcherUpdate = fmt.Sprintf(i18n.T(i18n.KeyLauncherUpdateAvailable), res.Version, buildVersion)
						}
					}
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
				OnCheckUpdate: func() {
					state.checkLauncherUpdate(win)
				},
				OnOpenSettings: func() {
					state.mu.Lock()
					state.launcherOpen = false
					state.settingsEmbeddedSection = ""
					state.settingsOpen = true
					state.settingsDesktopActive = true
					state.seedDesktopDraft()
					state.mu.Unlock()
					if state.settings != nil {
						state.settings.OpenSettingsDocument(context.Background())
					}
					if win != nil {
						win.Update(func() {})
					}
				},
				OnEnableSkipUserPlugins: func() {
					if cfgStore == nil {
						return
					}
					err := cfgStore.UpdatePluginRecovery(func(m map[string]any) {
						m["skipUserPlugins"] = true
						m["appVersion"] = "" // explicit manual recovery survives upgrades
						if _, ok := m["reason"]; !ok {
							m["reason"] = "launcher"
						}
					})
					state.mu.Lock()
					if err != nil {
						state.launcherNote = err.Error()
					} else {
						state.launcherNote = i18n.T(i18n.KeyLauncherSkipUserPluginsOn)
					}
					state.mu.Unlock()
					if err == nil {
						state.syncHarnessFromConfig()
						if harnessSup != nil {
							state.restartUnlessImport(harnessSup)
						}
					}
					if win != nil {
						win.Update(func() {})
					}
				},
				OnClearPluginRecovery: func() {
					if cfgStore == nil {
						return
					}
					err := cfgStore.ClearPluginRecovery()
					state.mu.Lock()
					if err != nil {
						state.launcherNote = err.Error()
					} else {
						state.launcherNote = i18n.T(i18n.KeyLauncherSkipUserPluginsOff)
					}
					state.mu.Unlock()
					if err == nil {
						state.syncHarnessFromConfig()
						if harnessSup != nil {
							state.restartUnlessImport(harnessSup)
						}
					}
					if win != nil {
						win.Update(func() {})
					}
				},
				OnRestartHarness: func() {
					state.syncHarnessFromConfig()
					if harnessSup != nil {
						state.restartUnlessImport(harnessSup)
					}
					state.mu.Lock()
					state.launcherNote = i18n.T(i18n.KeyLauncherRestartHarness)
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
			}))
			return
		}
		// Exclusive panels (settings/market/…) must not keep desktop title actions.
		desktopChromeLocked := usageOpen || imOpen || marketOpen ||
			scheduleOpen || remoteOpen || webviewOpen || browserPreviewOpen || assistantOpen
		acceptPanelOK := st.State == harness.Ready
		// Keep the restore action outside the collapsed navigation track on
		// desktop. Preserve existing page-specific title actions.
		var captionGit *chrome.GitBadge
		if desktopChromeLocked {
			captionGit = gitBadge
		}
		var captionNewSession func()
		if st.State == harness.Ready && sessions != nil {
			captionNewSession = func() {
				state.revealMainConversation()
				sessions.CreateSession(context.Background())
			}
		}
		state.mainCaptionFileDrop(c, st.State == harness.Ready, frame.Model{Store: sessions, Sizes: &layout, WorkspaceRoot: workspaceRoot}, func() {
			if frame.SidebarRestoreCaption(c, &layout, titleColors, captionGit, captionNewSession) {
				// Settings and other exclusive routes bypass the navigation shell's
				// writeback. Persist caption restores through the same layout state.
				state.mu.Lock()
				state.layout.Sidebar, state.layout.SidebarExpanded = layout.Sidebar, layout.SidebarExpanded
				state.layout.Responsive = layout.Responsive
				state.mu.Unlock()
			} else if desktopChromeLocked {
				chrome.TitleBarWithActions(c, "", titleColors, gitBadge, nil)
			} else {
				// Thin drag strip so system traffic lights keep hit targets; no chip row.
				ui.Row(c).Fill().Height(chrome.EffectiveTitleBarHeight()).
					Background(titleColors.Background).DragWindow()
			}
		})
		// In-window tools menu (UIA-clickable). Replaces native Popup which automation cannot invoke reliably.
		if toolsMenuOpen && !desktopChromeLocked && st.State == harness.Ready {
			openTool := func(apply func()) {
				state.mu.Lock()
				state.launcherOpen = false
				state.toolsMenuOpen = false
				state.settingsOpen = false
				state.usageOpen = false
				state.imOpen = false
				state.marketOpen = false
				state.scheduleOpen = false
				state.remoteOpen = false
				state.webviewOpen = false
				state.browserPreviewOpen = false
				state.assistantConversationGeneration++
				state.assistantOpen = false
				apply()
				state.mu.Unlock()
				if win != nil {
					win.Update(func() {})
				}
			}
			ui.Column(c).FillWidth().Background(chrome.Color(colors.Sidebar)).
				BorderWidth(0, 0, 1, 0).BorderColor(chrome.Color(colors.Border)).
				Padding(6, 12, 6, 12).Gap(4).Children(func() {
				toolBtn := func(label string, fn func()) {
					if ui.Button(c, label).Clicked() {
						fn()
					}
				}
				toolBtn(i18n.T(i18n.KeyBrowserPreviewOpen), func() {
					state.showBrowserPreview(state.browserPreviewDraft)
					openTool(func() { state.browserPreviewOpen = true })
				})
				toolBtn(i18n.T(i18n.KeyAssistantOpen), func() {
					openTool(func() { state.assistantConversationGeneration++; state.assistantOpen = true })
				})
				toolBtn(i18n.T(i18n.KeyWebViewOpen), func() {
					openTool(func() { state.webviewOpen = true })
				})
				if cfgRemoteWorkspace {
					toolBtn(i18n.T(i18n.KeyRemoteOpen), func() {
						openTool(func() { state.remoteOpen = true })
					})
				}
				if marketStore != nil {
					toolBtn(i18n.T(i18n.KeyMarketOpen), func() {
						openTool(func() { state.marketOpen = true })
						marketStore.Refresh(context.Background())
						marketStore.EnsureCatalog(i18n.Locale())
					})
				}
				if schedStore != nil {
					toolBtn(i18n.T(i18n.KeyScheduleOpen), func() {
						openTool(func() { state.scheduleOpen = true })
						sid := ""
						if sessions != nil {
							sid = sessions.Snapshot().SelectedID
						}
						schedStore.Refresh(context.Background(), sid)
					})
				}
				if imStore != nil {
					toolBtn(i18n.T(i18n.KeyIMOpen), func() {
						openTool(func() {
							state.imOpen = true
							state.imDraft = imStore.Snapshot().Draft
						})
						imStore.Refresh(context.Background())
					})
				}
				if usageStore != nil {
					toolBtn(i18n.T(i18n.KeyUsageOpen), func() {
						openTool(func() { state.usageOpen = true })
						sid := ""
						if sessions != nil {
							sid = sessions.Snapshot().SelectedID
						}
						usageStore.Refresh(context.Background(), sid)
					})
				}
				toolBtn(i18n.T(i18n.KeyFilesTitle), func() {
					openTool(func() {
						state.filesPaneOpen = true
						state.rightPaneTab = "files"
					})
				})
				toolBtn(i18n.T(i18n.KeyAgentsTitle), func() {
					openTool(func() {
						state.filesPaneOpen = true
						state.rightPaneTab = "agents"
					})
				})
			})
		}
		diffsurface.DrawGitProgress(c, gitProgress, chrome.Color(colors.FG), chrome.Color(colors.Muted), func() { state.mu.Lock(); state.gitProgress = nil; state.mu.Unlock() })
		if gitBranch != "" {
			root := workspaceRoot
			chrome.DrawGitCommitStrip(c, titleColors, gitCommitOpen, chrome.GitCommitStrip{
				Message:     &gitCommitDraft,
				Dirty:       gitDirty,
				Placeholder: i18n.T(i18n.KeyGitCommitPlaceholder),
				CommitLabel: i18n.T(i18n.KeyGitCommit),
				CancelLabel: i18n.T(i18n.KeyGitCommitCancel),
				OnCommit: func() {
					if st.State == harness.Ready && root != "" && !gitBusy {
						gitWorkbench.RequestCommit(root, gitCommitDraft)
						rightPaneTab = "diff"
						gitCommitOpen = false
						state.mu.Lock()
						state.filesPaneOpen, state.rightPaneTab = true, "diff"
						state.gitCommitOpen = false
						state.mu.Unlock()
					}
				},
				OnCancel: func() {
					state.mu.Lock()
					state.gitCommitOpen = false
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
			})
			prCurrent := ""
			if gitPRReport.Current != nil {
				prCurrent = desktopgit.FormatPRInfo(*gitPRReport.Current)
			}
			var recent []chrome.PRListItem
			for _, item := range gitPRReport.Recent {
				info := item
				label := desktopgit.FormatPRInfo(info)
				n := info.Number
				row := chrome.PRListItem{Label: label}
				if st.State == harness.Ready && root != "" && !gitBusy && !gitPRBusy && n > 0 {
					row.CheckoutLabel = i18n.T(i18n.KeyGitPRCheckout)
					row.OnCheckout = func() { state.runGitPRCheckout(root, n) }
				}
				recent = append(recent, row)
			}
			chrome.DrawPRPanel(c, titleColors, gitPROpen, chrome.PRPanel{
				CurrentLabel:  prCurrent,
				MissingGH:     gitPRReport.MissingGH,
				MissingGHText: i18n.T(i18n.KeyGitPRNoGH),
				EmptyText:     i18n.T(i18n.KeyGitPRNone),
				RefreshLabel:  i18n.T(i18n.KeyGitPRRefresh),
				ViewLabel:     i18n.T(i18n.KeyGitPRView),
				CreateLabel:   i18n.T(i18n.KeyGitPRCreate),
				CloseLabel:    i18n.T(i18n.KeyGitPRClose),
				Recent:        recent,
				OnRefresh: func() {
					if root != "" {
						state.refreshPR(root, true)
					}
				},
				OnView: func() {
					if st.State == harness.Ready && root != "" && !gitBusy {
						state.runGitPRView(root)
					}
				},
				OnCreate: func() {
					if st.State == harness.Ready && root != "" && !gitBusy {
						state.mu.Lock()
						state.filesPaneOpen = true
						state.rightPaneTab = "diff"
						state.gitPROpen = false
						state.mu.Unlock()
						rightPaneTab, gitPROpen = "diff", false
						gitWorkbench.RequestPR(root)
						if win != nil {
							win.Invalidate()
						}
					}
				},
				OnClose: func() {
					gitPROpen = false
					state.mu.Lock()
					state.gitPROpen = false
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
			})
		}
		sharedNavigation := !usageOpen && !imOpen && !remoteOpen && !webviewOpen && !browserPreviewOpen
		if sessions == nil {
			state.mu.Lock()
			if state.sessions == nil {
				state.sessions = session.New()
			}
			sessions = state.sessions
			state.mu.Unlock()
		}
		var paintMainContent func(settings.SectionID)
		paintSettingsOverlay := func() {
			if settingsOpen && setStore != nil {
				snap := setStore.Snapshot()
				state.mu.Lock()
				if !state.desktopDraft.seeded {
					state.seedDesktopDraft()
				}
				desktopActive := state.settingsDesktopActive
				accountActive := state.settingsAccountActive
				credsActive := state.settingsCredentialsActive
				aboutActive := state.settingsAboutActive
				appearanceActive := state.settingsAppearanceActive
				interfaceActive := state.settingsInterfaceActive
				modelsActive := state.settingsModelsActive
				providerDiscoveryRendered = modelsActive
				mcpActive := state.settingsMCPActive
				mcpPanelRendered = mcpActive
				pluginsActive := state.settingsPluginsActive
				skillsActive := state.settingsSkillsActive
				agentPresetsActive := state.settingsAgentPresetsActive
				creatorTaskRendered = agentPresetsActive
				desktopErr := state.desktopSaveErr
				desktopAction := state.desktopLastAction
				if !state.credsDraft.seeded {
					state.seedCredentialsDraft()
				}
				credsErr := state.credsSaveErr
				credsAction := state.credsLastAction
				credsHarness := state.credsHarnessNote
				exeDir, _ := os.Executable()
				wallpaperChoices := theme.ListWallpapers(theme.WallpaperGalleryDirs(paths.UserDataDir(), filepath.Dir(exeDir))...)
				var restartPolicy *config.RestartPreferences
				if state.cfgStore != nil {
					savedPolicy := config.RestartPreferencesFrom(state.cfgStore.Get())
					restartPolicy = &savedPolicy
				}
				desktopFields := settings.DesktopFields{
					HarnessAutoRestart:        &state.desktopDraft.harnessAutoRestart,
					HarnessRestartMaxAttempts: &state.desktopDraft.harnessRestartMaxAttempts,
					HarnessRestartBaseDelayMs: &state.desktopDraft.harnessRestartBaseDelayMs,
					Workspace:                 &state.desktopDraft.workspace,
					Port:                      &state.desktopDraft.port,
					Theme:                     &state.desktopDraft.theme,
					Locale:                    &state.desktopDraft.locale,
					CloseToTray:               &state.desktopDraft.closeToTray,
					OpenAtLogin:               &state.desktopDraft.openAtLogin,
					WallpaperPath:             &state.desktopDraft.wallpaperPath,
					WallpaperChoices:          wallpaperChoices,
				}
				credFields := settings.CredentialsFields{
					APIKey:           &state.credsDraft.apiKey,
					BaseURL:          &state.credsDraft.baseURL,
					GithubToken:      &state.credsDraft.githubToken,
					RemoteToken:      &state.credsDraft.remoteToken,
					RemoteRelayToken: &state.credsDraft.remoteRelayToken,
				}
				sessSnap := session.ViewState{}
				if sessions != nil {
					sessSnap = sessions.Snapshot()
					if state.settingsModelSession != sessSnap.SelectedID {
						if state.settingsModelDrafts == nil {
							state.settingsModelDrafts = map[string]string{}
						}
						state.settingsModelDrafts[state.settingsModelSession] = state.settingsModelLabel
						state.settingsModelSession = sessSnap.SelectedID
						state.settingsModelLabel = state.settingsModelDrafts[sessSnap.SelectedID]
					}
					if state.settingsModelLabel == "" && sessSnap.SelectedModel != "" {
						state.settingsModelLabel = sessSnap.SelectedModel
					}
					if state.agentPresetManager == nil {
						state.agentPresetManager = session.NewAgentPresetManager(sessions)
					}
				}
				state.mu.Unlock()
				marketSnap := marketstore.ViewState{}
				if marketStore != nil {
					marketSnap = marketStore.Snapshot()
				}
				usageSnap := usagestore.ViewState{}
				if usageStore != nil {
					usageSnap = usageStore.Snapshot()
				}
				state.accountLogin.Observe(api.AccountView{Status: usageSnap.AccountStatus, Attempt: api.AccountViewAttempt{
					Id: usageSnap.AttemptID, Phase: usageSnap.AttemptPhase, AuthorizeUrl: &usageSnap.AuthorizeURL,
				}}, desktopwebview.OpenExternal)
				loginState := state.accountLogin.Snapshot()
				signOutState := state.accountSignOutAccess()
				if usageSnap.AuthorizeURL == "" && loginState.URL != "" {
					usageSnap.AuthorizeURL = loginState.URL
				}
				pluginRows := make([]settings.PluginRow, 0, len(marketSnap.Rows))
				for _, row := range marketSnap.Rows {
					pluginRows = append(pluginRows, settings.PluginRow{
						Name: row.Label, Status: row.Detail, Disabled: row.Enabled != nil && !*row.Enabled,
						CanToggle: row.CanToggle(), ReadOnlyReason: row.ReadOnlyReason, Error: row.Error,
					})
				}
				skillRows := make([]settings.SkillRow, 0, len(sessSnap.SkillCatalog.Skills))
				for _, sk := range sessSnap.SkillCatalog.Skills {
					skillRows = append(skillRows, settings.SkillRow{
						Name:   sk.Name,
						Source: sk.Source, Path: skillPathValue(sk.Path), Directory: skillPathValue(sk.Directory),
						Description:    sk.Description,
						WhenToUse:      skillPathValue(sk.WhenToUse),
						ModelInvocable: sk.ModelInvocable,
						Groups:         sk.Groups, Writable: sk.Writable, UserInvocable: sk.UserInvocable, Error: sessSnap.SkillCatalog.Errors[sk.Name],
					})
				}
				removal := session.SkillRemovalReview{}
				if sessions != nil {
					removal = sessions.SkillRemovalReview()
				}
				settingsModel := settings.Model{
					Pet:                   state.desktopPet,
					ReferencedCredentials: state.referencedCredentialsAccess(),
					Store:                 setStore,
					ThemeID:               themeID,
					SystemDark:            mygo.Theme.IsDark(),
					DesktopActive:         desktopActive,
					Desktop: &settings.DesktopAccess{
						Version:       state.harnessVersionSnapshot(),
						RestartPolicy: restartPolicy,
						Fields:        desktopFields,
						LastAction:    desktopAction,
						Err:           desktopErr,
						OnSave: func() {
							state.mu.Lock()
							state.saveDesktopConfig()
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnBrowseWorkspace: func() {
							def := strings.TrimSpace(state.desktopDraft.workspace)
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
								Parent:      win,
								Title:       i18n.T(i18n.KeySettingsDesktopBrowseWorkspace),
								DefaultPath: def,
								Directory:   true,
							})
							if err != nil || len(paths) == 0 {
								return
							}
							state.mu.Lock()
							state.desktopDraft.workspace = paths[0]
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnDetectElectron: func() {
							state.showElectronImportReport(win)
						},
						OnCheckUpdates: func() {
							state.showUpdateCheck(win)
						},
					},
					AccountActive: accountActive,
					Account: &settings.AccountAccess{
						OnContact:       state.contactAccountSupport,
						SignInBusy:      loginState.Busy || signOutState.Open || signOutState.Busy,
						SignOut:         signOutState,
						ClientReady:     usageSnap.ClientReady,
						Loading:         usageSnap.Loading,
						Err:             strings.TrimSpace(usageSnap.Err + " " + loginState.Error + " " + accountActionError),
						Status:          usageSnap.AccountStatus,
						Email:           usageSnap.Email,
						Name:            usageSnap.Name,
						Plan:            usageSnap.Plan,
						UsageURL:        usageSnap.UsageURL,
						TopUpURL:        usageSnap.TopUpURL,
						BalanceSummary:  usageSnap.BalanceSummary,
						Details:         usageSnap.Details,
						HasRunningTasks: usageSnap.HasRunningTasks,
						AttemptPhase:    usageSnap.AttemptPhase,
						SignInAttemptID: usageSnap.AttemptID,
						AuthorizeURL:    usageSnap.AuthorizeURL,
						BonusesLoading:  usageSnap.BonusesLoading,
						BonusesError:    usageSnap.BonusesError,
						BonusAckState:   usageSnap.BonusFeedback.State,
						BonusAckError:   usageSnap.BonusFeedback.Error,
						BonusAckAccount: usageSnap.BonusFeedback.AccountID,
						BonusAckOrder:   usageSnap.BonusFeedback.OrderID,
						Bonuses: func() []settings.AccountBonus {
							out := make([]settings.AccountBonus, 0, len(usageSnap.Bonuses))
							for _, b := range usageSnap.Bonuses {
								label := strings.TrimSpace(b.Amount + " " + b.Currency)
								if label == "" {
									label = b.Campaign
								}
								out = append(out, settings.AccountBonus{
									AccountID: b.AccountID,
									OrderID:   b.OrderID,
									Label:     label,
									Detail:    b.Message,
								})
							}
							return out
						}(),
						OnAckBonus: func(accountID, orderID string) {
							if usageStore == nil {
								return
							}
							sid := ""
							if sessions != nil {
								sid = sessions.Snapshot().SelectedID
							}
							usageStore.AckBonusNotified(context.Background(), accountID, orderID, sid)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnRefresh: func() {
							if usageStore == nil {
								return
							}
							sid := ""
							if sessions != nil {
								sid = sessions.Snapshot().SelectedID
							}
							usageStore.Refresh(context.Background(), sid)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnOpenAuthorize: state.openAccountAuthorization,
						OnOpenUsage:     func(u string) { state.openAccountUsage(u, "settings") },
						OnOpenTopUp:     state.openAccountTopUp,
						OnSignIn: func() {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							if client == nil || !state.accountLogin.Begin() {
								return
							}
							go func() {
								_, zoneOff := time.Now().Zone()
								clientMeta := api.AccountClientMetadata{
									Version:               "whaleisle",
									Locale:                "zh-CN",
									TimezoneOffsetSeconds: float64(zoneOff),
								}
								var view api.AccountView
								err := client.Call(context.Background(), "account/startSignIn", map[string]any{
									"client":         clientMeta,
									"callbackOrigin": "whaleisle://auth",
									"loginSource":    "desktop",
								}, &view)
								state.accountLogin.Started(view, err, desktopwebview.OpenExternal)
								if usageStore != nil {
									sid := ""
									if sessions != nil {
										sid = sessions.Snapshot().SelectedID
									}
									usageStore.Refresh(context.Background(), sid)
								}
								if win != nil {
									win.Update(func() {})
								}
							}()
						},
						OnSignOut: func() { state.runAccountSignOut(win, state.accountSignOut.Request) },
						OnConfirmSignOut: func() {
							state.runAccountSignOut(win, func(ctx context.Context, meta any) { state.accountSignOut.Confirm(ctx, meta, signOutState.ReviewID) })
						},
						OnCancelSignOut: func() { state.accountSignOut.Cancel(signOutState.ReviewID) },
						OnCheckSignOut: func() {
							state.runAccountSignOut(win, func(ctx context.Context, meta any) { state.accountSignOut.Check(ctx, meta, signOutState.ReviewID) })
						},
						OnCancelSignIn: func() {
							attemptID := strings.TrimSpace(usageSnap.AttemptID)
							if attemptID == "" {
								return
							}
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							if client == nil {
								return
							}
							state.cancelAccountSignIn(client, attemptID, usageStore, sessions)
						},
					},
					CredentialsActive: credsActive,
					Credentials: &settings.CredentialsAccess{
						Fields:      credFields,
						ShowSecrets: &state.credsShowSecrets,
						LastAction:  credsAction,
						HarnessNote: credsHarness,
						Err:         credsErr,
						OnSave: func() {
							state.mu.Lock()
							state.saveCredentials()
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnDescribe: func() {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							if client == nil {
								state.mu.Lock()
								state.credsHarnessNote = "credentials/describe: no client"
								state.mu.Unlock()
								return
							}
							go func() {
								desc, err := settingsstore.DescribeCredentials(context.Background(), client, []string{
									settingsstore.CredRefAPIKey, settingsstore.CredRefGithubToken,
								})
								state.mu.Lock()
								if err != nil {
									state.credsHarnessNote = err.Error()
								} else {
									parts := make([]string, 0, len(desc))
									for ref, v := range desc {
										flag := "off"
										if v.Configured {
											flag = "on"
										}
										parts = append(parts, ref+"="+flag)
									}
									state.credsHarnessNote = "credentials/describe: " + strings.Join(parts, ", ")
								}
								state.mu.Unlock()
								if win != nil {
									win.Update(func() {})
								}
							}()
						},
					},
					AboutActive: aboutActive,
					About: &settings.AboutAccess{
						ReadCredentialStorage: func() credentials.StorageStatus { return credentials.InspectStorage(paths.UserDataDir()) },
						OnOpenLink:            desktopwebview.OpenExternal,
						AppName:               i18n.T(i18n.KeyAppName),
						Version:               buildVersion,
						Identifier:            "ai.deepseek.harness.gui",
						UserData:              paths.UserDataDir(),
						VendorRoot:            paths.VendorDir(),
						AnalyticsEnabled:      state.analyticsEnabled,
						OnRevealUserData: func() {
							_ = desktopcomponents.OpenInFileManager(paths.UserDataDir())
						},
						OnRevealLogs: func() {
							dir := filepath.Join(paths.UserDataDir(), "logs")
							_ = os.MkdirAll(dir, 0o755)
							_ = desktopcomponents.OpenInFileManager(dir)
						},
						OnRevealVendor: func() {
							_ = desktopcomponents.OpenInFileManager(paths.VendorDir())
						},
						OnCopyUserData: func() {
							mygo.Clipboard.WriteText(paths.UserDataDir())
						},
						OnCopyVersion: func() {
							mygo.Clipboard.WriteText(buildVersion)
						},
						OnCheckUpdates: func() {
							state.showUpdateCheck(win)
						},
						OnRefreshAnalytics: func() {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							if client == nil {
								return
							}
							go func() {
								on, err := settingsstore.ProductAnalyticsEnabled(context.Background(), client)
								state.mu.Lock()
								if err == nil {
									state.analyticsEnabled = &on
								}
								state.mu.Unlock()
								if win != nil {
									win.Update(func() {})
								}
							}()
						},
					},
					AppearanceActive: appearanceActive,
					Appearance: &settings.AppearanceAccess{
						OnSetTheme:       state.setAppearanceTheme,
						Background:       state.wallpaperSettings(appearanceActive),
						Preferences:      state.themeFontEditor(appearanceActive),
						Theme:            &state.desktopDraft.theme,
						WallpaperPath:    &state.desktopDraft.wallpaperPath,
						WallpaperChoices: wallpaperChoices,
						LastAction:       desktopAction,
						Err:              desktopErr,
						OnBrowseWallpaper: func() {
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
								Parent: win,
								Title:  i18n.T(i18n.KeySettingsAppearanceBrowseWallpaper),
							})
							if err != nil || len(paths) == 0 {
								return
							}
							if gallery := state.wallpaperEditor(); gallery != nil {
								gallery.ImportFile(paths[0])
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnSave: func() {
							state.mu.Lock()
							state.saveDesktopConfig()
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
					},
					InterfaceActive: interfaceActive,
					Interface: &settings.InterfaceAccess{
						Shortcuts:        state.pageShortcutEditor(),
						Locale:           &state.desktopDraft.locale,
						CloseToTray:      &state.desktopDraft.closeToTray,
						OpenAtLogin:      &state.desktopDraft.openAtLogin,
						AutoStartDesktop: &state.desktopDraft.autoStartDesktop,
						ShowArchivedList: &state.desktopDraft.showArchivedList,
						ShowViewTabs:     &state.desktopDraft.showViewTabs,
						AutoAllowTools:   &state.desktopDraft.autoAllowTools,
						LastAction:       desktopAction,
						Err:              desktopErr,
						OnSave: func() {
							state.mu.Lock()
							showTabs := state.desktopDraft.showViewTabs
							state.saveDesktopConfig()
							set := state.settings
							state.mu.Unlock()
							if set != nil {
								set.SyncViewTabs(context.Background(), showTabs)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
					},
					ModelsActive: modelsActive,
					Models: &settings.ModelsAccess{
						ProviderDiscovery: state.providerDiscoveryAccess(),
						ProviderProfiles:  state.providerProfilesAccess(),
						Vision:            state.visionSettingsAccess(),
						Labels:            sessSnap.ModelLabels,
						Selected:          &state.settingsModelLabel,
						SessionID:         sessSnap.SelectedID,
						Err:               sessSnap.ModelsErr,
						LastAction:        sessSnap.LastAction,
						OnRefresh: func() {
							if sessions != nil {
								sessions.RefreshModelCatalog(context.Background())
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnApply: func() {
							state.mu.Lock()
							label := state.settingsModelLabel
							state.mu.Unlock()
							if sessions != nil && label != "" {
								sessions.SelectModelForSessionID(context.Background(), sessSnap.SelectedID, label)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
					},
					MCPActive:     mcpActive,
					MCP:           state.settingsMCPAccess(),
					PluginsActive: pluginsActive,
					Plugins: &settings.PluginsAccess{
						Inventory:        state.pluginInventoryAccess(),
						Rows:             pluginRows,
						HasListRPC:       marketSnap.HasListRPC,
						Err:              marketSnap.Err,
						ClientReady:      marketSnap.ClientReady,
						Loading:          marketSnap.Loading,
						Busy:             marketInstallBusy,
						OperationMessage: marketInstallMsg,
						Activation:       marketActivation,
						CanRestart:       marketCanRestart,
						OnRestart:        state.restartMarketPlugins,
						LastAction: func() string {
							if state.settingsPluginsLastAction != "" {
								return state.settingsPluginsLastAction
							}
							return marketSnap.ListMethod
						}(),
						OnRefresh: func() {
							if marketStore != nil {
								marketStore.Refresh(context.Background())
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnSetDisabled: func(name string, disabled bool) {
							state.runMarketChange("pluginManager/setBundleEnabled", map[string]any{"name": name, "enabled": !disabled})
						},
						OnOpenView: state.openInstalledPluginView,
						OnInspect:  state.inspectInstalledPlugin,
						OnOpenMarket: func() {
							state.mu.Lock()
							state.settingsOpen = false
							state.settingsDesktopActive = false
							state.settingsAccountActive = false
							state.settingsCredentialsActive = false
							state.settingsAboutActive = false
							state.settingsAppearanceActive = false
							state.settingsInterfaceActive = false
							state.settingsModelsActive = false
							state.settingsMCPActive = false
							state.settingsPluginsActive = false
							state.settingsSkillsActive = false
							state.settingsAgentPresetsActive = false
							// MyGo uses its native market. The Host market requires Electron's
							// shell IPC and cannot register in the untrusted WebView.
							state.marketOpen = true
							state.webviewOpen = false
							state.browserPreviewOpen = false
							state.mu.Unlock()
							if marketStore != nil {
								marketStore.Refresh(context.Background())
							}
							if win != nil {
								win.Update(func() {})
							}
						},
					},
					SkillsActive: skillsActive,
					Skills: &settings.SkillsAccess{
						Create:       state.settingsSkillCreateAccess(),
						OnOpenCreate: state.openSettingsSkillCreate,
						Skills:       skillRows,
						Query:        &state.settingsSkillQuery, SourceFilter: &state.settingsSkillSource,
						RemovalID: removal.ID, RemovalLabel: removal.Label, RemovalMessage: removal.Message, RemovalReady: removal.Ready, RemovalBusy: removal.Busy,
						OnCancelRemoval: func() {
							if sessions != nil {
								sessions.CancelSkillRemoval(removal.ID)
							}
						},
						OnConfirmRemoval: func() {
							if sessions != nil {
								go sessions.ConfirmSkillRemoval(context.Background(), removal.ID)
							}
						},
						OnOpenDirectory: func(name string) {
							if sessions != nil {
								ticket, err := sessions.CaptureSkillDirectory(name)
								if err != nil {
									sessions.SetLastAction(err.Error())
									return
								}
								go sessions.OpenCapturedSkillDirectory(context.Background(), ticket)
							}
						},
						SessionID:  sessSnap.SelectedID,
						Err:        sessSnap.SkillCatalog.Err,
						Notice:     sessSnap.SkillCatalog.Notice,
						Busy:       sessSnap.SkillCatalog.Busy || state.settingsSkillEditorSaving,
						LastAction: strings.TrimSpace(strings.Join([]string{state.settingsSkillsLastAction, sessSnap.LastAction, sessSnap.SkillCatalog.Action}, "\n")),
						EditGroups: &state.settingsSkillEditGroups,
						OnSetGroupInvocable: func(group string, enabled bool) {
							if sessions != nil {
								sessions.SetSkillGroupInvocation(context.Background(), group, enabled)
							}
						},
						CreateName:         &state.skillCreateName,
						DetailPreview:      state.settingsSkillsDetail,
						EditName:           state.settingsSkillEditName,
						EditDesc:           &state.settingsSkillEditDesc,
						EditContent:        &state.settingsSkillEditContent,
						EditWritable:       state.settingsSkillEditWritable,
						EditWhenToUse:      &state.settingsSkillEditWhen,
						EditModelInvocable: &state.settingsSkillEditModel,
						EditUserInvocable:  &state.settingsSkillEditUser,
						EditCanSave:        sessions != nil && sessions.SkillEditorCurrent(state.settingsSkillEditTarget),
						OnCancelEdit:       state.cancelSettingsSkillEdit,
						OnSaveUpdate:       state.saveSettingsSkillEdit,
						OnCreate: func(name string) {
							if sessions != nil {
								sessions.CreateSkillInventory(context.Background(), name, name, "", state.settingsSkillRoot)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnDelete: func(name string) {
							if sessions != nil {
								sessions.DeleteSkillInventory(context.Background(), name)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnInspect:  state.inspectSettingsSkill,
						TargetRoot: state.settingsSkillRoot,
						ProjectCwd: func() string {
							if sessions == nil {
								return ""
							}
							target, _ := sessions.SkillInventoryTarget(session.SkillRootUser)
							return target.Cwd
						}(),
						OnSelectRoot: func(root string) {
							state.mu.Lock()
							state.settingsSkillRoot = root
							state.settingsSkillHubHits = nil
							state.settingsSkillHubTarget = session.SkillTarget{}
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						HubQuery: &state.settingsSkillHubQuery,
						HubHits:  append([]settings.SkillHubHit(nil), state.settingsSkillHubHits...),
						OnRefresh: func() {
							if sessions != nil {
								sessions.RefreshSkills(context.Background())
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnSearchHub: func(query string) {
							query = strings.TrimSpace(query)
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							if client == nil || sessions == nil || query == "" {
								return
							}
							target, targetErr := sessions.SkillInventoryTarget(state.settingsSkillRoot)
							if targetErr == nil && !target.MatchesClient(client) {
								return
							}
							if targetErr != nil {
								state.mu.Lock()
								state.settingsSkillsLastAction = targetErr.Error()
								state.mu.Unlock()
								if win != nil {
									win.Update(func() {})
								}
								return
							}
							go func() {
								req := api.SkillHubSearchRequest{Query: query, Root: target.Root}
								if target.SessionID != "" {
									req.SessionId = &target.SessionID
								}
								if target.Cwd != "" {
									req.Cwd = &target.Cwd
								}
								var out api.SearchHubResult
								err := client.Call(context.Background(), "skillInventory/searchHub", map[string]any{"request": req}, &out)
								hits := make([]settings.SkillHubHit, 0, len(out))
								if err == nil {
									for _, item := range out {
										st := ""
										if item.InstallationStatus != nil {
											st = *item.InstallationStatus
										}
										hits = append(hits, settings.SkillHubHit{
											SkillName:   item.SkillName,
											Repo:        item.Repo,
											Path:        item.Path,
											Description: item.Description,
											Stars:       item.Stars,
											Status:      st,
										})
									}
								}
								state.mu.Lock()
								currentTarget, currentErr := sessions.SkillInventoryTarget(state.settingsSkillRoot)
								if currentErr != nil || currentTarget != target {
									state.mu.Unlock()
									return
								}
								if err != nil {
									state.settingsSkillsLastAction = err.Error()
									state.settingsSkillHubHits = nil
								} else {
									state.settingsSkillsLastAction = fmt.Sprintf("hub: %d", len(hits))
									state.settingsSkillHubHits = hits
									state.settingsSkillHubTarget = target
								}
								state.mu.Unlock()
								if win != nil {
									win.Update(func() {})
								}
							}()
						},
						OnInstallHub: func(repo, path, skillName string) {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							if client == nil || sessions == nil || repo == "" || path == "" || skillName == "" {
								return
							}
							target, targetErr := sessions.SkillInventoryTarget(state.settingsSkillRoot)
							if targetErr == nil && !target.MatchesClient(client) {
								return
							}
							if targetErr != nil {
								state.mu.Lock()
								state.settingsSkillsLastAction = targetErr.Error()
								state.mu.Unlock()
								if win != nil {
									win.Update(func() {})
								}
								return
							}
							state.mu.Lock()
							if state.settingsSkillHubTarget != target {
								state.settingsSkillsLastAction = "Skill target changed; search again before installing."
								state.settingsSkillHubHits = nil
								state.mu.Unlock()
								if win != nil {
									win.Update(func() {})
								}
								return
							}
							state.mu.Unlock()
							go func() {
								req := api.SkillHubInstallRequest{
									Repo: repo, Path: path, SkillName: skillName, Root: target.Root,
								}
								if target.SessionID != "" {
									req.SessionId = &target.SessionID
								}
								if target.Cwd != "" {
									req.Cwd = &target.Cwd
								}
								var out api.SkillHubInstallResult
								err := client.Call(context.Background(), "skillInventory/installHub", map[string]any{"request": req}, &out)
								message := ""
								if err == nil {
									message, err = session.SkillHubInstallMessage(out.Status, out.SkillName, out.Directory)
								}
								state.mu.Lock()
								currentTarget, currentErr := sessions.SkillInventoryTarget(state.settingsSkillRoot)
								if currentErr != nil || currentTarget != target {
									state.mu.Unlock()
									return
								}
								if err != nil {
									state.settingsSkillsLastAction = err.Error()
								} else {
									state.settingsSkillsLastAction = message
									for i := range state.settingsSkillHubHits {
										row := &state.settingsSkillHubHits[i]
										if row.Repo == repo && row.Path == path && row.SkillName == skillName {
											row.Status = "installed"
										}
									}
								}
								state.mu.Unlock()
								if err == nil {
									sessions.RefreshSkills(context.Background())
								}
								if win != nil {
									win.Update(func() {})
								}
							}()
						},
						OnInsert: func(name string) {
							name = strings.TrimSpace(name)
							if name == "" {
								return
							}
							state.mu.Lock()
							id := ""
							if sessions != nil {
								id = sessions.Snapshot().SelectedID
							}
							skillDraft := state.composerDraftLocked(id)
							draft := strings.TrimRight(skillDraft.Text, " ")
							if draft != "" {
								draft += " "
							}
							skillDraft.Text = draft + "/" + name + " "
							skillDraft.Client.Sync()
							state.settingsOpen = false
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnSetModelInvocable: func(name string, modelInvocable bool) {
							if sessions != nil {
								sessions.SetCatalogSkillInvocation(context.Background(), name, modelInvocable)
							}
						},
					},
					AgentPresetsActive: agentPresetsActive,
					AgentPresets: &settings.AgentPresetsAccess{
						DisableDeveloperTools: !settingsstore.DeveloperToolsEnabled(setStore.Snapshot()),
						Creator:               state.creatorTaskAccess(sessions),
						Manager:               state.agentPresetManager,
						Selected:              &state.settingsAgentPresetLabel,
						Reader:                state.agentPresetReaderAccess(sessions),
					},
					OnSelectDesktop: func() {
						state.mu.Lock()
						state.settingsDesktopActive = true
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectAccount: func() {
						state.mu.Lock()
						state.settingsAccountActive = true
						state.settingsDesktopActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.mu.Unlock()
						if usageStore != nil {
							sid := ""
							if sessions != nil {
								sid = sessions.Snapshot().SelectedID
							}
							usageStore.Refresh(context.Background(), sid)
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectCredentials: func() {
						state.mu.Lock()
						state.settingsCredentialsActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						if !state.credsDraft.seeded {
							state.seedCredentialsDraft()
						}
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectAbout: func() {
						state.mu.Lock()
						state.settingsAboutActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectAppearance: func() {
						state.mu.Lock()
						state.settingsAppearanceActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						if !state.desktopDraft.seeded {
							state.seedDesktopDraft()
						}
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectInterface: func() {
						state.mu.Lock()
						state.settingsInterfaceActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						if !state.desktopDraft.seeded {
							state.seedDesktopDraft()
						}
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectModels: func() {
						state.mu.Lock()
						state.settingsModelsActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						if state.settingsModelLabel == "" && sessions != nil {
							state.settingsModelLabel = sessions.Snapshot().SelectedModel
						}
						state.mu.Unlock()
						state.refreshProviderProfiles()
						state.refreshVisionSettings()
						if sessions != nil {
							sessions.RefreshModelCatalog(context.Background())
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectMCP: func() {
						state.mu.Lock()
						state.settingsMCPActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.mu.Unlock()
						state.refreshSettingsMCP()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectPlugins: func() {
						state.refreshPluginInventory()
						state.mu.Lock()
						state.settingsPluginsActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.mu.Unlock()
						if marketStore != nil {
							marketStore.Refresh(context.Background())
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectSkills: func() {
						state.mu.Lock()
						state.settingsSkillsActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsAgentPresetsActive = false
						state.mu.Unlock()
						if sessions != nil {
							sessions.RefreshSkills(context.Background())
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectAgentPresets: func() {
						state.mu.Lock()
						state.settingsAgentPresetsActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetLabel = ""
						manager := state.agentPresetManager
						state.mu.Unlock()
						if manager != nil {
							go manager.Refresh(context.Background())
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectNamespace: func(ns string) {
						state.mu.Lock()
						if ns == "remote-workspace" {
							state.settingsEmbeddedSection = settings.SectionRemoteWorkspace
						}
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.mu.Unlock()
						setStore.Select(ns)
						if win != nil {
							win.Update(func() {})
						}
					},
					OnClose: func() {
						state.mu.Lock()
						state.settingsOpen = false
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.settingsAgentPresetLabel = ""
						state.settingsDrafts = nil
						state.settingsNSFilter = ""
						state.settingsRevealSecrets = false
						state.desktopSaveErr = ""
						state.desktopLastAction = ""
						state.credsDraft.seeded = false
						state.credsSaveErr = ""
						state.credsLastAction = ""
						state.credsShowSecrets = false
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnDraft: setStore.SetDraft,
					DraftPtr: func(key, seed string) *string {
						return state.settingsDraftPtr(snap.SelectedNS, key, seed)
					},
					BoolPtr: func(key, seed string) *bool {
						return state.settingsBoolPtr(snap.SelectedNS, key, seed)
					},
					OnOpenMarket: func() {
						state.mu.Lock()
						state.settingsEmbeddedSection = settings.SectionMarket
						state.mu.Unlock()
						if marketStore != nil {
							marketStore.Refresh(context.Background())
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenRemote: func() {
						state.mu.Lock()
						state.settingsEmbeddedSection = settings.SectionRemote
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenUsage: func() {
						state.mu.Lock()
						state.settingsEmbeddedSection = settings.SectionUsageStats
						state.mu.Unlock()
						if usageStore != nil {
							sid := ""
							if sessions != nil {
								sid = sessions.Snapshot().SelectedID
							}
							usageStore.Refresh(context.Background(), sid)
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					FilterPtr:     &state.settingsNSFilter,
					RevealSecrets: &state.settingsRevealSecrets,
					ConfirmReplace: func() bool {
						res, err := mygo.Dialog.Message(mygo.MessageOptions{
							Parent:        win,
							Type:          mygo.MessageQuestion,
							Title:         i18n.T(i18n.KeySettingsReplaceConfirmTitle),
							Message:       i18n.T(i18n.KeySettingsReplaceConfirmMessage),
							Buttons:       []string{i18n.T(i18n.KeySettingsReplaceCancel), i18n.T(i18n.KeySettingsReplace)},
							DefaultButton: 0,
							CancelButton:  0,
						})
						return err == nil && res.Button == 1
					},
				}
				state.wireSettingsBodies(&settingsModel)
				settings.ModalView(c, state.settingsSectionModel(settingsModel, func(_ *ui.Context) {
					state.mu.Lock()
					section := state.settingsEmbeddedSection
					state.mu.Unlock()
					paintMainContent(section)
				}))
				return
			}
		}
		paintMainContent = func(embeddedSection settings.SectionID) {
			if acceptPanelOK && embeddedSection == "" && marketOpen && marketManager {
				state.pluginManagerView(c, themeID)
				return
			}
			if ((acceptPanelOK && embeddedSection == "" && marketOpen) || embeddedSection == settings.SectionMarket) && marketStore != nil {
				marketStore.SetNotify(func() {
					if win != nil {
						win.Update(func() {})
					}
				})
				marketStore.EnsureCatalog(i18n.Locale())
				state.mu.Lock()
				muxReadyMarket := state.dshMuxReady
				state.mu.Unlock()
				marketOwner := state.captureMarketOwner(marketStore)
				marketui.View(c, marketui.Model{
					OnOpenURL: func(rawURL string) {
						if err := mygo.Shell.OpenExternal(rawURL); err != nil {
							log.Println("market README link:", err)
						}
					},
					Embedded:             embeddedSection == settings.SectionMarket,
					Store:                marketStore,
					ThemeID:              themeID,
					SystemDark:           mygo.Theme.IsDark(),
					HarnessReady:         st.State == harness.Ready,
					MuxConnected:         muxReadyMarket,
					HarnessMarketOverlay: !marketPluginDisabled,
					PluginDisabled:       marketPluginDisabled,
					InstallReady:         marketInstallReady,
					InstallSpec:          marketInstallSpecPtr,
					InstallBusy:          marketInstallBusy,
					InstallMessage:       marketInstallMsg,
					InstallLog:           marketInstallLog,
					Activation:           marketActivation,
					CanRestart:           marketCanRestart,
					OnRestart:            state.restartMarketPlugins,
					NeedsAllowBuilds:     marketNeedsAllowBuilds,
					AllowBuilds:          marketAllowBuilds,
					CanCancelInstall:     marketCanCancel,
					Incompatible:         marketIncompatible,
					OnClose: func() {
						state.mu.Lock()
						state.marketOpen = false
						state.scheduleOpen = false
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnRefresh: func() {
						marketStore.Refresh(context.Background())
						if win != nil {
							win.Update(func() {})
						}
					},
					OnBatchUpdate:    state.runMarketBatchUpdates,
					OnStopBatch:      state.stopMarketBatch,
					OnInstallCatalog: state.runMarketCatalogInstall,
					OnInstallSpec: func(spec string, allowBuilds []string) {
						state.runMarketInstallForOwner(spec, allowBuilds, &marketOwner)
					},
					OnCancelInstall: func() {
						state.cancelMarketInstall()
					},
					OnAcceptExemption: state.acceptMarketExemption,
					OnRemoveBundle: func(name string) {
						state.runMarketChangeForOwner("pluginManager/removeBundle", map[string]any{"name": name}, &marketOwner)
					},
					OnSetBundleEnabled: func(name string, enabled bool) {
						state.runMarketChangeForOwner("pluginManager/setBundleEnabled", map[string]any{"name": name, "enabled": enabled}, &marketOwner)
					},
					OnExportHistory: func() {
						data := marketStore.ExportHistory()
						path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: win, Title: "导出安装历史 / Export installation history", DefaultPath: "whaleisle-marketplace.log"})
						if err != nil {
							mygo.Dialog.Error("Market export", err.Error())
							return
						}
						if path == "" {
							return
						}
						if err = os.WriteFile(path, data, 0600); err != nil {
							mygo.Dialog.Error("Market export", err.Error())
						}
					},
					OnCopyInstallLog: func(log string) {
						if strings.TrimSpace(log) != "" {
							mygo.Clipboard.WriteText(log)
						}
					},
					OnOpenHarnessMarket: func() {
						origin := ""
						state.mu.Lock()
						if state.sup != nil {
							st := state.sup.Status()
							if st.State == harness.Ready {
								origin = st.Endpoint.Origin
							}
						}
						state.mu.Unlock()
						if origin != "" {
							state.openWebViewHostPlugin(origin, desktopwebview.PluginMarket)
						}
					},
				})
				return
			}
			if acceptPanelOK && embeddedSection == "" && scheduleOpen && schedStore != nil {
				sid := ""
				sessionLinks := map[string]string{}
				if sessions != nil {
					sessionSnap := sessions.Snapshot()
					sid = sessionSnap.SelectedID
					sessionLinks = scheduleSessionLinks(sessionSnap)
				}
				state.schedulePanel.View(c, scheduleui.Model{Store: schedStore, SessionID: sid, ThemeID: themeID, SystemDark: mygo.Theme.IsDark(),
					SessionLinks:  sessionLinks,
					OnOpenSession: state.openScheduleSession,
					OnNewTask: func() {
						state.selectMainPanel("")
						if sessions != nil {
							sessions.CreateSession(context.Background())
						}
					},
					OnClose: func() {
						state.mu.Lock()
						state.scheduleOpen = false
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnInvalidate: func() {
						if win != nil {
							win.Update(func() {})
						}
					},
					OnCopyHistory: func(text string) {
						if strings.TrimSpace(text) != "" {
							mygo.Clipboard.WriteText(text)
						}
					},
				})
				return
			}

			if ((acceptPanelOK && embeddedSection == "" && imOpen) || embeddedSection == "channels") && imStore != nil {
				imOnboardingRendered = true
				state.mu.Lock()
				muxReadyIM := state.dshMuxReady
				imDraft := state.imDraft
				state.mu.Unlock()
				imui.View(c, imui.Model{
					Embedded:         embeddedSection == "channels",
					Store:            imStore,
					ThemeID:          themeID,
					SystemDark:       mygo.Theme.IsDark(),
					HarnessReady:     st.State == harness.Ready,
					MuxConnected:     muxReadyIM,
					HarnessIMOverlay: !imPluginDisabled,
					PluginDisabled:   imPluginDisabled,
					Draft:            &imDraft,
					OnClose: func() {
						if sel := imStore.Snapshot().SelectedID; sel != "" {
							imStore.SetDraft(sel, imDraft)
						}
						_ = imStore.Persist()
						state.mu.Lock()
						state.imOpen = false
						state.imDraft = imDraft
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnRefresh: func() {
						imStore.Refresh(context.Background())
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenPluginWeb: func() {
						state.openWebViewHostPlugin(st.Endpoint.Origin, desktopwebview.PluginIM)
						state.mu.Lock()
						state.webviewOpen = true
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSend: func(channelID, text string) {
						imStore.SendDraft(context.Background(), channelID, text)
						if win != nil {
							win.Update(func() {})
						}
					},
				})
				state.mu.Lock()
				state.imDraft = imDraft
				state.mu.Unlock()
				return
			}
			if (acceptPanelOK && embeddedSection == "" && remoteOpen) || embeddedSection == settings.SectionRemote || embeddedSection == settings.SectionRemoteWorkspace {
				state.mu.Lock()
				muxReady := state.dshMuxReady
				remoteMachines := state.remoteMachines
				sshProbeLine := state.sshProbeLine
				sshHost := state.sshHost
				sshUser := state.sshUser
				sshPort := state.sshPort
				sshIdentity := state.sshIdentity
				sshForward := state.sshForward
				sshTunnelMsg := state.sshTunnelMsg
				tunnelState := string(sshremote.TunnelIdle)
				if state.sshTunnel != nil {
					_, stt, errLine := state.sshTunnel.Snapshot()
					tunnelState = string(stt)
					if errLine != "" && sshTunnelMsg == "" {
						sshTunnelMsg = errLine
					}
				}
				state.mu.Unlock()
				var channelsContent func(*ui.Context)
				if imStore != nil && !imPluginDisabled {
					channelsContent = func(c *ui.Context) { paintMainContent("channels") }
				}
				remoteui.View(c, remoteui.Model{
					Embedded:            embeddedSection != "",
					SettingsPage:        map[settings.SectionID]string{settings.SectionRemote: "remote", settings.SectionRemoteWorkspace: "workspace"}[embeddedSection],
					ActiveTab:           &state.settingsRemoteTab,
					ChannelsContent:     channelsContent,
					GatewayProtocol:     "chisacode-v2",
					LegacyRelayTokenSet: hasRemoteRelayToken,
					OnSetRelayToken:     state.clearLegacyRelayToken,
					OnRotateToken:       func() error { return state.phoneDeviceAction("rotate", "", "") },
					OnUseWorkspaceMirror: func(localMirror string) {
						go state.applyWorkspacePath(win, sessions, localMirror, false)
					},
					Machines:               remoteMachines,
					Files:                  state.remoteSessionFiles(win, sessions),
					FileEditor:             &state.remoteFileEditor,
					MachineEditor:          &state.remoteMachineEditor,
					ThemeID:                themeID,
					SystemDark:             mygo.Theme.IsDark(),
					RemoteWorkspaceEnabled: cfgRemoteWorkspace,
					RemoteEnabled:          cfgRemoteEnabled,
					PhoneGitEnabled:        cfgPhoneGitEnabled,
					PhoneGitWorkspaces:     state.phoneGitWorkspaceRows(),
					OnTogglePhoneGitWorkspace: func(id string, enabled bool) {
						state.setPhoneGitWorkspace(id, enabled)
						go state.refreshPhoneRemote()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnTogglePhoneGit: func(enabled bool) {
						state.setPhoneGit(enabled)
						go state.refreshPhoneRemote()
						if win != nil {
							win.Update(func() {})
						}
					},
					RemoteMode:           cfgRemoteMode,
					RemotePort:           cfgRemotePort,
					RemoteBindAddress:    cfgRemoteBind,
					RemoteLanTls:         cfgRemoteLanTls,
					RemoteAppBaseURL:     cfgRemoteAppBase,
					RemoteRelayEndpoint:  cfgRemoteRelayEndpoint,
					RemoteRelayUseTLS:    cfgRemoteRelayUseTLS,
					RemoteDeviceCount:    remoteDeviceCount,
					RemoteDeviceNames:    remoteDeviceNames,
					PhoneActionMsg:       phoneActionMsg,
					PhoneRuntime:         state.phoneRuntimeManager().Snapshot(),
					PhoneDeviceEditor:    &state.phoneDeviceEditor,
					GatewayEditor:        &state.phoneGatewayEditor,
					OnSaveGateway:        state.savePhoneGateway,
					OnPhoneDeviceAction:  state.phoneDeviceAction,
					HarnessRemoteOverlay: cfgRemoteWorkspace,
					MuxConnected:         muxReady,
					HarnessReady:         st.State == harness.Ready,
					HasRemoteToken:       hasRemoteToken,
					HasRemoteRelayToken:  hasRemoteRelayToken,
					SSHProbeLine:         sshProbeLine,
					SSHHost:              &sshHost,
					SSHUser:              &sshUser,
					SSHPort:              &sshPort,
					SSHIdentity:          &sshIdentity,
					SSHForward:           &sshForward,
					SSHTunnelState:       tunnelState,
					SSHTunnelMsg:         sshTunnelMsg,
					OnClose: func() {
						state.revokePhoneGit()
						state.mu.Lock()
						state.remoteOpen = false
						state.sshHost = sshHost
						state.sshUser = sshUser
						state.sshPort = sshPort
						state.sshIdentity = sshIdentity
						state.sshForward = sshForward
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnToggleWorkspace: func(enabled bool) {
						if state.cfgStore == nil {
							return
						}
						_ = state.cfgStore.Update(func(c *config.Config) {
							c.RemoteWorkspaceEnabled = enabled
						})
						if win != nil {
							win.Update(func() {})
						}
					},
					OnTogglePairing: func(enabled bool) {
						if state.cfgStore == nil {
							return
						}
						if err := state.cfgStore.Update(func(c *config.Config) {
							c.RemoteEnabled = enabled
						}); err != nil {
							state.mu.Lock()
							state.phoneActionMsg = "手机 Remote 设置未保存，请重试。 / Phone Remote setting was not saved; retry."
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
							return
						}
						if !enabled {
							// Revoke before the asynchronous stop; no phone Git
							// request may use the previous desktop grant meanwhile.
							state.revokePhoneGit()
						}
						go func() {
							// Read the latest saved setting inside phoneOp. A queued
							// disable must not stop a more recent enable request.
							_ = state.refreshPhoneRemote()
							if win != nil {
								win.Update(func() {})
							}
						}()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnCopyConnectURL: func() {
						url := state.phoneConnectURL()
						state.mu.Lock()
						if url == "" {
							state.phoneActionMsg = i18n.T(i18n.KeyRemotePhoneNoURL)
						} else {
							mygo.Clipboard.WriteText(url)
							state.phoneActionMsg = i18n.T(i18n.KeyRemotePhoneCopied)
						}
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnCopySSHTarget: func(target string) {
						target = strings.TrimSpace(target)
						if target == "" {
							return
						}
						mygo.Clipboard.WriteText(target)
						state.mu.Lock()
						state.sshTunnelMsg = i18n.T(i18n.KeyChatCopy) + ": " + target
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenConnectURL: func() {
						url := state.phoneConnectURL()
						message := i18n.T(i18n.KeyRemotePhoneNoURL)
						if url != "" {
							if err := desktopwebview.OpenExternal(url); err == nil {
								message = i18n.T(i18n.KeyRemotePhoneOpened)
							} else {
								message = i18n.T(i18n.KeyRemotePhoneOpenFailed)
							}
						}
						// Never put an offer into persisted webview navigation/history.
						state.mu.Lock()
						state.phoneActionMsg = message
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnRefreshPhone: func() {
						go func() {
							_ = state.refreshPhoneRemote()
							if win != nil {
								win.Update(func() {})
							}
						}()
					},
					OnProbeSSH: func() {
						line := remoteui.RunSSHProbe(remoteui.Model{
							RemoteWorkspaceEnabled: cfgRemoteWorkspace,
							RemoteEnabled:          cfgRemoteEnabled,
							RemoteMode:             cfgRemoteMode,
							RemotePort:             cfgRemotePort,
							RemoteBindAddress:      cfgRemoteBind,
							RemoteLanTls:           cfgRemoteLanTls,
							HasRemoteToken:         hasRemoteToken,
							HasRemoteRelayToken:    hasRemoteRelayToken,
						})
						state.mu.Lock()
						state.sshProbeLine = line
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSaveTunnel: func() {
						port, _ := strconv.Atoi(strings.TrimSpace(sshPort))
						cfg := sshremote.TunnelConfig{
							Host:         strings.TrimSpace(sshHost),
							User:         strings.TrimSpace(sshUser),
							Port:         port,
							IdentityFile: strings.TrimSpace(sshIdentity),
							LocalForward: strings.TrimSpace(sshForward),
						}
						state.mu.Lock()
						state.invalidateSSHConnectLocked()
						err := sshremote.SaveTunnelConfig(paths.UserDataDir(), cfg)
						state.sshHost = cfg.Host
						state.sshUser = cfg.User
						state.sshPort = strconv.Itoa(cfg.Port)
						state.sshIdentity = cfg.IdentityFile
						state.sshForward = cfg.LocalForward
						if state.sshTunnel == nil {
							state.sshTunnel = sshremote.NewTunnelSession()
						}
						state.sshTunnel.SetConfig(cfg)
						if err != nil {
							state.sshTunnelMsg = err.Error()
						} else {
							state.sshTunnelMsg = "saved ssh-tunnel.json"
						}
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnConnectSSH: func() {
						port, _ := strconv.Atoi(strings.TrimSpace(sshPort))
						cfg := sshremote.TunnelConfig{
							Host:         strings.TrimSpace(sshHost),
							User:         strings.TrimSpace(sshUser),
							Port:         port,
							IdentityFile: strings.TrimSpace(sshIdentity),
							LocalForward: strings.TrimSpace(sshForward),
						}
						state.mu.Lock()
						op := state.beginSSHConnectLocked(cfg)
						state.mu.Unlock()
						kh := sshremote.KnownHostsPath(paths.UserDataDir())
						go op.run(sshConnectEffects{
							connect: op.tunnel.Connect,
							prompt: func(unknown *sshremote.UnknownHostKey, beforeShow func() bool) bool {
								return promptAcceptUnknownSSHHost(win, unknown, beforeShow)
							},
							changed: func(changed *sshremote.HostKeyChanged, beforeShow func() bool) {
								showSSHHostKeyChanged(win, changed, beforeShow)
							},
							accept: func(unknown *sshremote.UnknownHostKey) error {
								return sshremote.AcceptHostKey(kh, unknown.Address, unknown.Key)
							},
							redraw: func() {
								if win != nil {
									win.Update(func() {})
								}
							},
						})
					},
					OnDisconnectSSH: func() {
						state.mu.Lock()
						state.invalidateSSHConnectLocked()
						if state.sshTunnel != nil {
							state.sshTunnel.Disconnect()
						}
						state.sshTunnelMsg = "disconnected"
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
				})
				state.mu.Lock()
				state.sshHost = sshHost
				state.sshUser = sshUser
				state.sshPort = sshPort
				state.sshIdentity = sshIdentity
				state.sshForward = sshForward
				state.mu.Unlock()
				return
			}
			if acceptPanelOK && embeddedSection == "" && assistantOpen {
				origin := st.Endpoint.Origin
				ep := st.Endpoint
				whaleOn := true
				dshbotOn := false
				if state.cfgStore != nil {
					cfg := state.cfgStore.Get()
					whaleOn = cfg.WhaleAssistantEnabled
					dshbotOn = cfg.DshbotEnabled
				}
				whaleVendor := false
				dshbotVendor := false
				if info, err := os.Stat(filepath.Join(paths.VendorDir(), "dsh-whale")); err == nil && info.IsDir() {
					whaleVendor = true
				}
				if info, err := os.Stat(filepath.Join(paths.VendorDir(), "dshbot")); err == nil && info.IsDir() {
					dshbotVendor = true
				}
				state.mu.Lock()
				assistantDraft := state.assistantDraft
				lastWebURL := state.lastWebViewURL
				lastNative := state.assistantLastAct
				pluginBusy := state.assistantPluginBusy
				pluginDetail := state.assistantPluginDetail
				asst := state.assistant
				var dshbotRows []dshbotstore.Row
				var dshbotRosterAt string
				if dr := state.dshbotRoster; dr != nil {
					snap := dr.Snapshot()
					if snap.Origin == origin {
						dshbotRows = snap.Rows
						dshbotRosterAt = snap.FetchedAt
					}
				}
				state.mu.Unlock()
				if asst == nil {
					asst = assiststore.New()
					state.mu.Lock()
					state.assistant = asst
					state.mu.Unlock()
				}
				asst.Attach(func() {
					if win != nil {
						win.Update(func() {})
					}
				})
				asstSnap := asst.Snapshot()
				assistantDraft = asst.ConsumeAcceptedDraft(assistantDraft)
				dshbotSelectedID := ""
				if asstSnap.DshbotOrigin == origin {
					dshbotSelectedID = asstSnap.DshbotID
				}
				if lastNative == "" {
					lastNative = asstSnap.LastAction
				}
				if lastNative == "" && sessions != nil {
					lastNative = sessions.Snapshot().LastAction
				}
				history := asstSnap.History
				if len(history) == 0 && sessions != nil {
					sessSnap := sessions.Snapshot()
					var pairs []struct{ Role, Body string }
					for _, m := range sessSnap.Messages {
						pairs = append(pairs, struct{ Role, Body string }{m.Role, m.Body})
					}
					history = assiststore.HistoryFromSessionMessages(pairs, 8)
				}
				taskModel := state.assistantTasksModel(ep, dshbotSelectedID)
				routineModel := state.assistantRoutinesModel(ep, dshbotSelectedID)
				routineCreateModel := state.assistantRoutineCreateModel(ep, dshbotSelectedID, routineModel.State, routineModel.OnRefresh)
				roomCreateRendered = asstSnap.Mode == assiststore.ModeDshbot
				roomCreateModel := state.assistantRoomCreateModel(ep)
				botCreateModel := state.assistantBotCreateModel(ep)
				botVisibilityModel := state.assistantBotVisibilityModel(ep)
				contactsModel := assistant.ContactsModel{}
				if asstSnap.Mode == assiststore.ModeDshbot {
					contactsModel = state.assistantContactsModel(ep)
				}
				botDeleteModel := state.assistantBotDeleteModel(ep, dshbotSelectedID)
				botDuplicateModel := state.assistantBotDuplicateModel(ep, dshbotSelectedID)
				roomEditRendered = asstSnap.Mode == assiststore.ModeDshbot
				roomEditModel := state.assistantRoomEditModel(ep, dshbotSelectedID)
				roomRuntimeRendered = asstSnap.Mode == assiststore.ModeDshbot
				roomRuntimeModel := state.assistantRoomRuntimeModel(ep, dshbotSelectedID)
				whaleSettingsModel := state.assistantWhaleSettingsModel(ep)
				assistant.View(c, assistant.Model{
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
					Tasks:                 taskModel,
					ThemeID:               themeID,
					SystemDark:            mygo.Theme.IsDark(),
					Mode:                  asstSnap.Mode,
					WhaleEnabled:          whaleOn,
					DshbotEnabled:         dshbotOn,
					WhaleVendorPresent:    whaleVendor,
					DshbotVendorPresent:   dshbotVendor,
					HarnessReady:          true,
					Busy:                  asstSnap.Busy || pluginBusy || taskModel.State.Busy || whaleSettingsModel.State.Busy || routineModel.State.Busy || routineCreateModel.State.Busy || roomCreateModel.State.Busy || botCreateModel.State.Busy || botVisibilityModel.State.Busy || contactsModel.State.Busy || botDeleteModel.State.Busy || botDuplicateModel.State.Busy || roomEditModel.State.Busy || roomRuntimeModel.State.Busy,
					LastWebViewURL:        lastWebURL,
					LastNativeAction:      lastNative,
					LastErr:               asstSnap.LastErr,
					SessionID:             asstSnap.SessionID,
					History:               history,
					Draft:                 &assistantDraft,
					PluginRPCBusy:         pluginBusy,
					PluginRPCDetail:       pluginDetail,
					DshbotRoster:          dshbotRows,
					DshbotRosterFetched:   dshbotRosterAt,
					BotNotificationStatus: state.botNotificationStatusText(),
					DshbotSelectedID:      dshbotSelectedID,
					OnDshbotSelect: func(botID string) {
						state.assistantBotCreator.Suspend()
						state.assistantBotVisibility.Suspend()
						state.assistantContacts.Suspend()
						state.assistantContacts.Suspend()
						state.assistantBotDeleter.Suspend()
						state.assistantBotDuplicator.Suspend()
						state.assistantRoomEditor.Suspend()
						state.assistantRoomRuntime.Suspend()
						state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
						state.assistantTasks.Bind(dshbotstore.TaskScope{})
						asst.SelectDshbot(botID, origin)
					},
					OnSetMode: func(mode assiststore.Mode) {
						state.assistantBotCreator.Suspend()
						state.assistantBotVisibility.Suspend()
						state.assistantContacts.Suspend()
						state.assistantContacts.Suspend()
						state.assistantBotDeleter.Suspend()
						state.assistantBotDuplicator.Suspend()
						state.assistantRoomCreator.Suspend()
						state.assistantRoomEditor.Suspend()
						state.assistantRoomRuntime.Suspend()
						state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
						state.whaleSettings.Suspend()
						state.assistantTasks.Bind(dshbotstore.TaskScope{})
						if asst != nil {
							asst.SetMode(mode)
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenModeWeb: func() {
						state.assistantBotCreator.Suspend()
						state.assistantBotVisibility.Suspend()
						state.assistantContacts.Suspend()
						state.assistantContacts.Suspend()
						state.assistantBotDeleter.Suspend()
						state.assistantBotDuplicator.Suspend()
						state.assistantRoomCreator.Suspend()
						state.assistantRoomEditor.Suspend()
						state.assistantRoomRuntime.Suspend()
						state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
						state.whaleSettings.Suspend()
						plugin := desktopwebview.PluginWhale
						if assiststore.NormalizeMode(asstSnap.Mode) == assiststore.ModeDshbot {
							plugin = desktopwebview.PluginDshbot
						}
						state.mu.Lock()
						state.webviewOpen = true
						state.assistantConversationGeneration++
						state.assistantOpen = false
						state.assistantDraft = assistantDraft
						state.mu.Unlock()
						state.openWebViewHostPlugin(origin, plugin)
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenSession: func(_ string) {
						target := asst.Snapshot()
						if target.Mode == assiststore.ModeDshbot && target.DshbotOrigin != ep.Origin {
							asst.SetLastAction("Select a Bot from the current Host roster first", "Host changed; draft retained")
							return
						}
						state.openAssistantConversation(ep, target.Mode, target.DshbotID, "")
					},
					OnClose: func() {
						state.whaleSettings.Suspend()
						state.assistantBotCreator.Suspend()
						state.assistantBotVisibility.Suspend()
						state.assistantContacts.Suspend()
						state.assistantContacts.Suspend()
						state.assistantBotDeleter.Suspend()
						state.assistantBotDuplicator.Suspend()
						state.assistantRoomCreator.Suspend()
						state.assistantRoomEditor.Suspend()
						state.assistantRoomRuntime.Suspend()
						state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
						state.assistantTasks.Bind(dshbotstore.TaskScope{})
						state.mu.Lock()
						state.assistantConversationGeneration++
						state.assistantOpen = false
						state.assistantDraft = assistantDraft
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnToggleWhale: func(enabled bool) {
						state.whaleSettings.Suspend()
						state.setAssistantFlag(&enabled, nil)
					},
					OnToggleDshbot: func(enabled bool) {
						state.assistantBotCreator.Suspend()
						state.assistantBotVisibility.Suspend()
						state.assistantContacts.Suspend()
						state.assistantContacts.Suspend()
						state.assistantBotDeleter.Suspend()
						state.assistantBotDuplicator.Suspend()
						state.assistantRoomCreator.Suspend()
						state.assistantRoomEditor.Suspend()
						state.assistantRoomRuntime.Suspend()
						state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
						state.assistantTasks.Bind(dshbotstore.TaskScope{})
						state.setAssistantFlag(nil, &enabled)
					},
					OnOpenWhaleWeb: func() {
						state.assistantBotCreator.Suspend()
						state.assistantBotVisibility.Suspend()
						state.assistantContacts.Suspend()
						state.assistantContacts.Suspend()
						state.assistantBotDeleter.Suspend()
						state.assistantBotDuplicator.Suspend()
						state.assistantRoomCreator.Suspend()
						state.assistantRoomEditor.Suspend()
						state.assistantRoomRuntime.Suspend()
						state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
						state.whaleSettings.Suspend()
						state.mu.Lock()
						state.webviewOpen = true
						state.assistantConversationGeneration++
						state.assistantOpen = false
						state.assistantDraft = assistantDraft
						state.assistantLastAct = i18n.T(i18n.KeyAssistantOpenWhaleWeb)
						state.mu.Unlock()
						state.openWebViewHostPlugin(origin, desktopwebview.PluginWhale)
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenDshbotWeb: func() {
						state.assistantBotCreator.Suspend()
						state.assistantBotVisibility.Suspend()
						state.assistantContacts.Suspend()
						state.assistantContacts.Suspend()
						state.assistantBotDeleter.Suspend()
						state.assistantBotDuplicator.Suspend()
						state.assistantRoomCreator.Suspend()
						state.assistantRoomEditor.Suspend()
						state.assistantRoomRuntime.Suspend()
						state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
						state.whaleSettings.Suspend()
						state.mu.Lock()
						state.webviewOpen = true
						state.assistantConversationGeneration++
						state.assistantOpen = false
						state.assistantDraft = assistantDraft
						state.assistantLastAct = i18n.T(i18n.KeyAssistantOpenDshbotWeb)
						state.mu.Unlock()
						state.openWebViewHostPlugin(origin, desktopwebview.PluginDshbot)
						if win != nil {
							win.Update(func() {})
						}
					},
					OnNativePrompt: func(text string) {
						state.handleAssistantNativePrompt(origin, ep, text)
						if win != nil {
							win.Update(func() {})
						}
					},
					OnWhaleEnsure: func() {
						state.openAssistantConversation(ep, assiststore.ModeWhale, "", "")
					},
					OnWhaleCatalog: func() {
						state.runAssistantPluginRPC(win, ep, "catalog", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
							out, err := c.WhaleCatalog(ctx)
							return string(out), err
						})
					},
					OnWhalePetChat: func(text string) {
						state.handleAssistantNativePrompt(origin, ep, text)
					},
					OnWhalePetState: func() {
						state.runAssistantPluginRPC(win, ep, "pet/state", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
							out, err := c.WhalePetState(ctx, false)
							return string(out), err
						})
					},
					OnWhalePetLook: func() {
						paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
							Parent: win,
							Title:  i18n.T(i18n.KeyAssistantWhalePetLook),
						})
						if err != nil {
							state.mu.Lock()
							state.assistantPluginDetail = "pet/look: " + err.Error()
							state.assistantLastAct = state.assistantPluginDetail
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
							return
						}
						if len(paths) == 0 {
							return
						}
						imgPath := paths[0]
						state.runAssistantPluginRPC(win, ep, "pet/look", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
							st, err := c.WhalePetStateTyped(ctx, false)
							if err != nil {
								return "", err
							}
							if st.Model == nil || strings.TrimSpace(st.Model.Provider) == "" || strings.TrimSpace(st.Model.Model) == "" {
								return "", fmt.Errorf("no whale model selected (open whale WebView and pick a vision model)")
							}
							raw, err := os.ReadFile(imgPath)
							if err != nil {
								return "", err
							}
							if len(raw) == 0 || len(raw) > 4<<20 {
								return "", fmt.Errorf("image empty or larger than 4MiB")
							}
							b64 := base64.StdEncoding.EncodeToString(raw)
							out, err := c.WhalePetLook(ctx, st.Model.Provider, st.Model.Model, b64)
							return string(out), err
						})
					},
					OnDshbotDescribe: func() {
						state.runAssistantPluginRPC(win, ep, "describe", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
							out, err := c.DshbotDescribe(ctx)
							return string(out), err
						})
					},
					OnDshbotCatalog: func() {
						state.runAssistantPluginRPC(win, ep, "catalog/items", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
							out, err := c.DshbotCatalog(ctx)
							return string(out), err
						})
					},
					OnDshbotActivity: func() {
						state.runAssistantPluginRPC(win, ep, "bot/activity", func(ctx context.Context, c *pluginrpc.Client) (string, error) {
							act, err := c.DshbotActivityTyped(ctx, nil)
							if err != nil {
								return "", err
							}
							return fmt.Sprintf("%d entries", len(act.Entries)), nil
						})
					},
					OnDshbotRefreshRoster: func() {
						label := i18n.T(i18n.KeyAssistantDshbotRosterRefresh)
						state.runAssistantPluginRPC(win, ep, label, func(ctx context.Context, c *pluginrpc.Client) (string, error) {
							res, err := c.DshbotDescribeTyped(ctx, "")
							if err != nil {
								return "", err
							}
							state.mu.Lock()
							roster := state.dshbotRoster
							state.mu.Unlock()
							if roster != nil {
								roster.SetFromDescribeForHost(res, false, origin)
								if act, aerr := c.DshbotActivityTyped(ctx, nil); aerr == nil {
									roster.MergeActivity(act)
								}
								for _, row := range roster.Snapshot().Rows {
									if row.Kind != "room" {
										continue
									}
									if rt, rerr := c.DshbotGroupRuntime(ctx, row.ID); rerr == nil {
										roster.ApplyGroupRuntime(rt)
									}
								}
							}
							rows := pluginrpc.RosterRowsFromDescribe(res, false)
							return fmt.Sprintf("%d", len(rows)), nil
						})
					},
					OnDshbotOpenRosterSession: func(botID string) {
						state.assistantBotCreator.Suspend()
						state.assistantBotVisibility.Suspend()
						state.assistantContacts.Suspend()
						state.assistantContacts.Suspend()
						state.assistantBotDeleter.Suspend()
						state.assistantBotDuplicator.Suspend()
						state.assistantRoomCreator.Suspend()
						state.assistantRoomEditor.Suspend()
						state.assistantRoomRuntime.Suspend()
						state.assistantRoutines.Bind(dshbotstore.RoutineScope{})
						state.openAssistantDshbot(ep, botID)
					},
				})
				state.mu.Lock()
				state.assistantDraft = assistantDraft
				state.mu.Unlock()
				return
			}
			if acceptPanelOK && embeddedSection == "" && browserPreviewOpen {
				if state.browserController == nil {
					state.browserController = state.newBrowserController(win)
				}
				state.browserController.View(c, themeID, mygo.Theme.IsDark(), func() {
					state.mu.Lock()
					state.browserPreviewOpen = false
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				})
				return
			}

			if acceptPanelOK && embeddedSection == "" && webviewOpen {
				origin := st.Endpoint.Origin
				state.mu.Lock()
				host := state.webviewHost
				hostErr := state.webviewHostErr
				hostActive := host != nil && host.Alive()
				hostURL := ""
				if hostActive {
					hostState := host.State()
					hostURL = hostState.URL
					if hostState.Error != "" {
						hostErr = hostState.Error
					}
				}
				resumeURL := desktopwebview.RestorePanelURL(origin, state.lastWebViewOrigin, state.lastWebViewURL)
				state.mu.Unlock()
				if hostActive {
					host.SyncBounds()
				}
				webviewui.View(c, webviewui.Model{
					HarnessOrigin: origin,
					ThemeID:       themeID,
					SystemDark:    mygo.Theme.IsDark(),
					HostActive:    hostActive,
					HostURL:       hostURL,
					HostError:     hostErr,
					OnClose: func() {
						state.closeWebViewHost()
						state.mu.Lock()
						state.webviewOpen = false
						state.assistantConversationGeneration++
						state.assistantOpen = false
						state.mu.Unlock()
						state.returnFromAccountPage()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenHost: func() {
						state.mu.Lock()
						state.lastWebViewOrigin = origin
						state.mu.Unlock()
						state.openWebViewURL(resumeURL)
						if win != nil {
							win.Update(func() {})
						}
					},
					OnCloseHost: func() {
						state.closeWebViewHost()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnCopyURL: func(u string) {
						if strings.TrimSpace(u) != "" {
							mygo.Clipboard.WriteText(u)
						}
					},
					OnReload: func(u string) {
						u = strings.TrimSpace(u)
						if u == "" {
							return
						}
						state.mu.Lock()
						state.webviewHostErr = ""
						state.mu.Unlock()
						if host != nil && host.Alive() {
							host.Reload()
						} else {
							state.openWebViewURL(u)
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenBrowser: func() {
						u := resumeURL
						if hostActive {
							u = hostURL
						}
						if u != "" {
							if err := desktopwebview.OpenExternal(u); err != nil {
								log.Printf("webview: open browser: %v", err)
							}
						}
						if win != nil {
							win.Update(func() {})
						}
					},
				})
				return
			}
			if (acceptPanelOK && embeddedSection == "" && usageOpen && usageStore != nil) || embeddedSection == settings.SectionUsageStats {
				sid := ""
				if sessions != nil {
					sid = sessions.Snapshot().SelectedID
				}
				usageui.View(c, usageui.Model{
					Embedded:    embeddedSection == settings.SectionUsageStats,
					ActionError: accountActionError,
					Store:       usageStore,
					OnExportStats: func(format string) {
						data, err := usageStore.ExportStats(format)
						if err != nil {
							mygo.Dialog.Error("Usage export", err.Error())
							return
						}
						ext := "csv"
						if format == "json" {
							ext = "json"
						}
						path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: win, Title: "Export usage statistics", DefaultPath: "usage-" + format + "." + ext})
						if err != nil {
							mygo.Dialog.Error("Usage export", err.Error())
							return
						}
						if path == "" {
							return
						}
						if err = os.WriteFile(path, data, 0600); err != nil {
							mygo.Dialog.Error("Usage export", err.Error())
						}
					},
					OnOpenSession: func(id string) {
						if sessions == nil {
							return
						}
						state.revokePhoneGit()
						sessions.Select(id)
						sessions.FollowSelected(context.Background())
						state.mu.Lock()
						state.usageOpen = false
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					SessionID:  sid,
					ThemeID:    themeID,
					SystemDark: mygo.Theme.IsDark(),
					OnClose: func() {
						state.mu.Lock()
						state.usageOpen = false
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnRefresh: func() {
						usageStore.Refresh(context.Background(), sid)
						if win != nil {
							win.Update(func() {})
						}
					},
					OnOpenUsageWeb: func() {
						state.openAccountUsage(usageStore.Snapshot().UsageURL, "usage")
					},
					OnOpenTopUp: func() {
						state.openAccountTopUp(usageStore.Snapshot().TopUpURL)
					},
					OnCopyUsageURL: func(u string) {
						if strings.TrimSpace(u) != "" {
							mygo.Clipboard.WriteText(u)
						}
					},
					OnCopyTopUpURL: func(u string) {
						if strings.TrimSpace(u) != "" {
							mygo.Clipboard.WriteText(u)
						}
					},
					OnCopyBalance: func(text string) {
						if strings.TrimSpace(text) != "" {
							mygo.Clipboard.WriteText(text)
						}
					},
				})
				return
			}

			if embeddedSection != "" {
				ui.Text(c, i18n.T(i18n.KeySettingsLoading)).FillWidth().TextColor(c.Theme().TextMuted)
				return
			}
			// Electron keeps the desk rail+chat chrome visible while harness boots
			// or fails; exclusive boot.View hid the entire left rail and blocked
			// pixel acceptance. Paint the dialogue shell whenever we have a store.
			if sessions == nil {
				state.mu.Lock()
				if state.sessions == nil {
					state.sessions = session.New()
				}
				sessions = state.sessions
				state.mu.Unlock()
			}
			if sessions != nil {
				state.mu.Lock()
				if state.agentPresetManager == nil {
					state.agentPresetManager = session.NewAgentPresetManager(sessions)
				}
				agentPresetManager := state.agentPresetManager
				state.mu.Unlock()
				// Body under the title bar must Grow into remaining height (not Fill%).
				ui.Column(c).Grow(1).MinHeight(0).Children(func() {
					boot.RecoveryControl(c, boot.Model{
						Status: st, Logs: logs, ThemeID: themeID, SystemDark: mygo.Theme.IsDark(),
						OnConfigure: func() {
							state.mu.Lock()
							state.launcherOpen = true
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnExport: func() {
							if _, err := harness.ExportBootLogsDialog(win, buildVersion, st, logs, i18n.T(i18n.KeyBootExport)); err != nil {
								log.Println("export log:", err)
							}
						},
						OnCancelRecovery: func() {
							if sup != nil {
								sup.CancelRecovery()
							}
						},
						OnRetry: func() {
							if sup != nil {
								state.restartUnlessImport(sup)
							}
						},
					})
					state.mu.Lock()
					connBanner := connectionBanner(st, state.dshMuxReady, state.dshMuxLost)
					state.mu.Unlock()
					var usageJobsToday *int
					usageBalance := ""
					if usageStore != nil {
						us := usageStore.Snapshot()
						usageBalance = us.BalanceSummary
						if n, ok := usagestore.TodayJobCount(us.DayBuckets); ok {
							usageJobsToday = &n
						}
					}
					snap := sessions.Snapshot()
					state.mu.Lock()
					activeDraft := state.composerDraftLocked(snap.SelectedID)
					composerRendered, composerSessionID = true, snap.SelectedID
					draft = activeDraft.Text
					state.composerTextInput = activeDraft.Client
					state.mu.Unlock()
					if snap.SelectedModel != "" && modelPick != snap.SelectedModel {
						modelPick = snap.SelectedModel
					}
					if permissionPick != snap.SelectedPermission {
						permissionPick = snap.SelectedPermission
					}
					if snap.SelectedAgentPreset != "" && agentPresetPick != snap.SelectedAgentPreset {
						agentPresetPick = snap.SelectedAgentPreset
					}
					state.mu.Lock()
					if state.composerSuggest.Selected == nil {
						state.composerSuggest.Selected = &state.composerSuggestSel
					}
					if strings.HasPrefix(draft, "/") && snap.ClientReady {
						sessions.EnsureSlashCommands(context.Background())
					}
					state.composerSlash = composer.BuildSlashCatalog(composer.StubSlashCommands(), composer.FromAPIList(snap.SlashCommands))
					if kind, _, _, ok := composer.ActiveSuggest(draft); ok && kind == composer.SuggestMention {
						root := strings.TrimSpace(workspaceRoot)
						if root != "" && root != state.composerPathsRoot {
							if paths, err := workspacefs.IndexRelPaths(root, 4000); err == nil {
								state.composerPathsRoot = root
								state.composerPaths = paths
							}
						}
					}
					composerSlash := state.composerSlash
					composerPaths := state.composerPaths
					composerSuggest := state.composerSuggest
					composerSkillsOpen := state.composerSkillsOpen
					composerSkillsList := state.composerSkillsList
					composerSkillsFilter := state.composerSkillsFilter
					skillCreateName := state.skillCreateName
					composerChipEdit := state.composerChipEdit
					if composerSkillsOpen && snap.SelectedID != "" && snap.SelectedID != state.composerSkillsSession {
						sessions.RefreshSkills(context.Background())
						state.composerSkillsSession = snap.SelectedID
					}
					if composerSkillsList.Selected == nil {
						composerSkillsList.Selected = &state.composerSkillsSel
					}
					state.mu.Unlock()
					state.ensureTerminal(workspaceRoot, mygo.Theme.IsDark())
					state.mu.Lock()
					termPanel := state.terminalPanel
					termShellsNote := state.terminalShellsNote
					state.mu.Unlock()
					filePaneWidth := float32(0)
					state.mu.Lock()
					filesPaneOpen := state.filesPaneOpen
					state.mu.Unlock()
					visualReplyState := sessions.VisualReplySnapshot()
					if visualReplyState.SessionID != snap.SelectedID {
						visualReplyState = session.VisualReplyState{}
					}
					groupRoom, groupMembers := state.assistantGroupMembers(snap.SelectedID)
					frameModel := frame.Model{
						CustomPanelContent: state.customPanelContent(snap.SelectedID, themeID),
						GroupRoom:          groupRoom, GroupMembers: groupMembers,
						OnReplyGroupThread: func(threadID string) {
							state.beginAssistantGroupReply(sessions, activeDraft, snap.SelectedID, threadID)
						},
						ExternalNavigation:   sharedNavigation,
						PageShortcutBindings: state.pageShortcutBindings(),
						SessionSearchFocus:   &state.commandSearchFocus,
						OnOpenWorkspaceFiles: func() { filesPaneOpen = true; rightPaneTab = "files" },
						Projects:             state.projectModel(win),
						Store:                sessions,
						ScheduleStore:        schedStore,
						OnOpenScheduleTask: func(row schedstore.Entry) {
							state.openScheduleTask(row)
							state.mu.Lock()
							filesPaneOpen, rightPaneTab = state.filesPaneOpen, state.rightPaneTab
							layout.Workspace = state.layout.Workspace
							state.mu.Unlock()
						},
						OnOpenScheduleSession: state.openScheduleSession,
						OnScheduleInvalidate: func() {
							if win != nil {
								win.Update(func() {})
							}
						},
						AgentPresetManager: agentPresetManager,
						Interactions:       interactions,
						OnAutoAllowChanged: func(on bool) {
							state.mu.Lock()
							state.desktopDraft.autoAllowTools = on
							if state.cfgStore != nil {
								_ = state.cfgStore.Update(func(c *config.Config) {
									c.AutoAllowTools = on
								})
							}
							state.mu.Unlock()
						},
						ConnectionBanner: connBanner,
						OnReconnect: func() {
							state.mu.Lock()
							sup := state.sup
							state.mu.Unlock()
							if sup != nil {
								state.restartUnlessImport(sup)
							}
						},
						SessionCost: costui.Model{Billing: func() usagestore.BillingView {
							if usageStore != nil {
								return usageStore.Billing()
							}
							return usagestore.BillingView{}
						}(), Preferences: setStore.Snapshot(), State: &state.sessionCostState, OnPrices: func() { state.mu.Lock(); state.usageOpen = true; state.mu.Unlock() }},
						UsageJobsToday: usageJobsToday,
						UsageBalance:   usageBalance,
						OnOpenUsage: func() {
							if usageStore == nil {
								return
							}
							state.mu.Lock()
							state.usageOpen = true
							state.imOpen = false
							state.settingsOpen = false
							state.webviewOpen = false
							state.browserPreviewOpen = false
							state.assistantConversationGeneration++
							state.assistantOpen = false
							state.remoteOpen = false
							state.marketOpen = false
							state.scheduleOpen = false
							state.mu.Unlock()
							sid := ""
							if sessions != nil {
								sid = sessions.Snapshot().SelectedID
							}
							usageStore.Refresh(context.Background(), sid)
							if win != nil {
								win.Update(func() {})
							}
						},
						TitlebarPreferences:     settingsstore.TitlebarPreferencesFrom(setStore.Snapshot()),
						DisableDeveloperTools:   !settingsstore.DeveloperToolsEnabled(setStore.Snapshot()),
						HideArchivedList:        cfgHideArchivedList,
						ChatPreferences:         settingsstore.ChatPreferencesFrom(setStore.Snapshot()),
						ConversationPreferences: settingsstore.ConversationPreferencesFrom(setStore.Snapshot()),
						ComposerResize:          &state.composerResize,
						OnComposerResizeEnd:     func(axis string, value float32) { setStore.SaveComposerSize(context.Background(), axis, value) },
						Sizes:                   &layout,
						ChatList:                &chatList,
						ThemeID:                 themeID,
						SystemDark:              mygo.Theme.IsDark(),
						Draft:                   &activeDraft.Text,
						ComposerState:           activeDraft,
						OnSubmitComposerGesture: func(accelerated bool) {
							state.submitComposer(win, sessions, activeDraft, snap.SelectedID, workspaceRoot, editMessageSeq, accelerated, snap.ComposerHostGeneration)
						},
						OnAttachComposerFiles: func(paths []string) { state.addComposerFiles(win, sessions, snap.SelectedID, activeDraft, paths) },
						OnRetryComposerFile:   func(index int) { state.prepareComposerFile(win, sessions, snap.SelectedID, activeDraft, index, true) },
						OnPasteComposerImage:  func(explicit bool) { state.pasteComposerImage(win, sessions, snap.SelectedID, activeDraft, explicit) },
						EditMessageSeq:        &editMessageSeq,
						FeedbackCommentMsgID:  &feedbackCommentMsgID,
						FeedbackCommentDraft:  &feedbackCommentDraft,
						ModelPick:             &modelPick,
						PermissionPick:        &permissionPick,
						AgentPresetPick:       &agentPresetPick,
						ThinkingExpanded:      &thinkingExpanded,
						RenameSessionID:       &renameSessionID,
						RenameDraft:           &renameDraft,
						SessionSearch:         &sessionSearch,
						WorkspaceRoot:         workspaceRoot,
						FileExpanded:          &fileExpanded,
						SelectedFile:          &selectedFile,
						FileFilter:            &fileFilter,
						FilePreview:           &filePreview,
						FilePreviewErr:        &filePreviewErr,
						FilePreviewList:       &filePreviewList,
						FileEditing:           &fileEditing,
						FileEditDraft:         &fileEditDraft,
						FileTabs:              &fileTabs,
						FileBindingID:         fileBindingID,
						GitWorkbench:          &gitWorkbench,
						FileEditStatus:        &fileEditStatus,
						FileEditViewMode:      &fileEditViewMode,
						FileEditSplit:         &fileEditSplit,
						FilePaneWidth:         &filePaneWidth,
						RightPaneOpen:         filesPaneOpen,
						RightPaneTab:          &rightPaneTab,
						AgentsPromptDraft:     &agentsPromptDraft,
						OnOpenAgentPaneDocument: func(paneID string, owner *session.Store, token uint64, address session.AgentSessionAddress, path string) {
							state.openAgentPaneDocument(paneID, owner, token, address, path)
						},
						OnOpenAgentDocument: func(token uint64, address session.AgentSessionAddress, path string) {
							state.openAgentDocument(sessions, token, address, path)
						},
						GoalDraft:          &goalDraft,
						DiffPath:           &diffPath,
						DiffText:           &diffText,
						DiffErr:            &diffErr,
						PlanBoardOpen:      &planBoardOpen,
						TodoPanelCollapsed: &todoPanelCollapsed,
						ChatPaneTab:        &chatPaneTab,
						HideChatViewTabs:   !settingsstore.ConversationPreferencesFrom(setStore.Snapshot()).ViewTabs,
						TrajSearch:         &trajSearch,
						TrajSelectedID:     &trajSelectedID,
						TrajActualDuration: &trajActualDur,
						TrajTurnsCollapsed: &trajTurnsFold,
						TrajCallsCollapsed: &trajCallsFold,
						TrajDetailTab:      &trajDetailTab,
						ImageLightboxPath:  &imageLightboxPath,
						OnOpenImageLightbox: func(p string) {
							imageLightboxPath = p
							state.mu.Lock()
							state.imageLightboxPath = p
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnCloseImageLightbox: func() {
							imageLightboxPath = ""
							state.mu.Lock()
							state.imageLightboxPath = ""
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						ComposerSuggest:      &composerSuggest,
						ComposerSlash:        composerSlash,
						ComposerPaths:        composerPaths,
						ComposerSkillsOpen:   composerSkillsOpen,
						ComposerSkillsList:   &composerSkillsList,
						ComposerSkillsFilter: &composerSkillsFilter,
						ComposerChipEdit:     &composerChipEdit,
						ComposerTextInput:    state.composerTextInput,
						// The native client and composer share the stable session-owned draft.
						UseComposerTextInput: true,
						OnSearchSkillHub: func(query string) {
							if sessions != nil {
								sessions.SearchSkillHub(context.Background(), query, session.SkillRootUser)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnInstallSkillHub: func(hit session.SkillHubHit) {
							if sessions != nil {
								sessions.InstallSkillHub(context.Background(), hit, session.SkillRootUser)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnDeleteSkill: func(name string) {
							state.mu.Lock()
							state.settingsEmbeddedSection = ""
							state.settingsOpen = true
							state.settingsSkillsActive = true
							state.settingsDesktopActive = false
							state.settingsAccountActive = false
							state.settingsCredentialsActive = false
							state.settingsAboutActive = false
							state.settingsAppearanceActive = false
							state.settingsInterfaceActive = false
							state.settingsModelsActive = false
							state.settingsMCPActive = false
							state.settingsPluginsActive = false
							state.settingsAgentPresetsActive = false
							state.mu.Unlock()
							if sessions != nil {
								sessions.DeleteSkillInventory(context.Background(), name)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						SkillCreateName: &skillCreateName,
						OnCreateSkill: func(name string) {
							if sessions != nil {
								sessions.CreateSkillInventory(context.Background(), name, name, "", session.SkillRootUser)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnToggleSkillModel: func(name string, modelInvocable bool) {
							if sessions != nil {
								sessions.SetSkillInvocation(context.Background(), name, modelInvocable, true)
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnToggleComposerSkills: func(open bool) {
							state.mu.Lock()
							state.composerSkillsOpen = open
							state.mu.Unlock()
							if open && sessions != nil {
								sessions.RefreshSkills(context.Background())
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnNoDirectory: func() { state.openNoDirectory(win, sessions, activeDraft, snap.SelectedID) },
						OnSelectWorkspace: func(id string) {
							if sessions == nil {
								return
							}
							sessions.OpenWorkspacePicker()
							request := sessions.Snapshot().WorkspaceNavigation
							if request != nil && request.Mode == "pick" && sessions.ChooseRegisteredWorkspace(request.Token, id, state.persistRegisteredWorkspace) {
								state.workspaceSelectionCommitted(win, sessions, sessions.WorkspaceCwd())
							}
						},
						OnPickHeroWorkspace: func() {
							state.pickAndApplyWorkspace(win, sessions, workspaceRoot)
						},
						WorkspacePickerAnchor: &state.workspacePickerAnchor,
						OnBrowseWorkspace: func() {
							state.openDirectoryPicker(sessions, win, "workspace")
						},
						OnRevealWorkspace: func() {
							root := strings.TrimSpace(workspaceRoot)
							if root == "" && state.cfgStore != nil {
								root = strings.TrimSpace(state.cfgStore.Get().Workspace)
							}
							if root == "" {
								return
							}
							_ = desktopcomponents.OpenInFileManager(root)
						},
						WorkspaceRenameDraft: &state.workspaceRenameDraft,
						OnRenameWorkspace: func(title string) {
							if sessions == nil {
								return
							}
							id := sessions.WorkspaceID()
							if id == "" {
								sessions.SetLastAction("workspace rename: no workspace id")
								return
							}
							sessions.RenameWorkspace(context.Background(), id, title)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnDeleteWorkspace: func() {
							if sessions == nil {
								return
							}
							id := sessions.WorkspaceID()
							if id == "" {
								sessions.SetLastAction("workspace delete: no workspace id")
								return
							}
							sessions.DeleteWorkspace(context.Background(), id)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnMoveSessionBefore: func(sessionID, beforeSessionID string) {
							if sessions == nil {
								return
							}
							sessions.InsertSessionBefore(context.Background(), sessions.WorkspaceID(), sessionID, beforeSessionID)
							if win != nil {
								win.Update(func() {})
							}
						},
						WorkspaceMenuOpen: &state.workspaceMenuOpen,
						OnOpenAgents: func() {
							state.mu.Lock()
							state.filesPaneOpen = true
							state.rightPaneTab = "agents"
							state.mu.Unlock()
							rightPaneTab = "agents"
							if win != nil {
								win.Update(func() {})
							}
						},
						OnPickComposerDirectory: func() {
							root := strings.TrimSpace(workspaceRoot)
							if root == "" {
								if sessions != nil {
									sessions.SetLastAction(i18n.T(i18n.KeyComposerDirNoWorkspace))
								}
								return
							}
							picked := ""
							state.mu.Lock()
							tunnel := state.sshTunnel
							state.mu.Unlock()
							hostPick := false
							if tunnel != nil {
								if _, st, _ := tunnel.Snapshot(); st == sshremote.TunnelConnected {
									hostPick = true
								}
							}
							if hostPick && sessions != nil {
								state.openDirectoryPicker(sessions, win, "mention")
								return
							} else {
								paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
									Parent:      win,
									Title:       i18n.T(i18n.KeyComposerDirDialogTitle),
									DefaultPath: root,
									Directory:   true,
								})
								if err != nil {
									if sessions != nil {
										sessions.SetLastAction(err.Error())
									}
									return
								}
								if len(paths) == 0 {
									return // cancelled
								}
								picked = paths[0]
							}
							if picked == "" {
								return
							}
							if err := composer.InsertDirectoryMention(&activeDraft.Text, root, picked); err != nil {
								if sessions != nil {
									sessions.SetLastAction(i18n.T(i18n.KeyComposerDirOutside))
								}
								return
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnPickComposerImage: func() {
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: "Attach files or images", DefaultPath: workspaceRoot, Multiple: true})
							if err != nil {
								activeDraft.Error = err.Error()
								return
							}
							state.addComposerFiles(win, sessions, snap.SelectedID, activeDraft, paths)
							if win != nil {
								win.Update(func() {})
							}
						},
						ComposerVoice:         state.composerVoiceStatus(),
						OnStartComposerVoice:  func() { state.startComposerVoice(win, sessions, snap.SelectedID, activeDraft) },
						OnStopComposerVoice:   state.stopComposerVoice,
						OnCancelComposerVoice: state.cancelComposerVoice,
						OnInsertComposerVoice: state.insertComposerVoice,
						OnPickComposerSpeech: func() {
							if sessions == nil {
								return
							}
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
								Parent: win,
								Title:  i18n.T(i18n.KeyComposerSpeech),
							})
							if err != nil {
								sessions.SetLastAction(err.Error())
								return
							}
							if len(paths) == 0 {
								return
							}
							go func() {
								text, err := sessions.TranscribeAudioFile(context.Background(), paths[0])
								if err != nil {
									sessions.SetLastAction(err.Error())
									if win != nil {
										win.Update(func() {})
									}
									return
								}
								if win != nil {
									win.Update(func() { composer.AppendTranscript(&activeDraft.Text, text) })
								}
							}()
						},
						OnCycleComposerSpeech: func() {
							if sessions == nil {
								return
							}
							sessions.CycleSpeechProvider(context.Background())
							if win != nil {
								win.Update(func() {})
							}
						},
						OnWatchWorkspaceDirectoryContext: func(ctx context.Context, rel string, changed func()) error {
							path := strings.TrimRight(strings.ReplaceAll(workspaceRoot, `\`, "/"), "/")
							if rel != "" {
								path += "/" + rel
							}
							if path == "" {
								path = "/"
							}
							return workspacefs.WatchRemoteDirectory(ctx, fileMux, fileSessionID, path, changed)
						},
						OnListWorkspaceFilesContext: func(ctx context.Context, rel string) ([]workspacefs.Entry, error) {
							return workspacefs.ListRemoteDirScoped(ctx, fileClient, fileSessionID, rel)
						},
						OnListWorkspaceFiles: func(rel string) ([]workspacefs.Entry, error) {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							return workspacefs.ListRemoteDir(context.Background(), client, rel)
						},
						OnReadWorkspaceTableContext: remoteWorkbookReader(sessions, fileClient, snap.SelectedID),
						OnOpenDocumentViewer: func(path string) (files.DocumentSurface, error) {
							return state.openDocumentViewer(fileClient, snap.SelectedID, path)
						},
						OnReadWorkspaceTextPageContext: func(ctx context.Context, rel string, offset int64) (workspacefs.TextPage, error) {
							return workspacefs.ReadRemoteTextPage(ctx, fileClient, fileSessionID, rel, offset)
						},
						OnReadWorkspaceFileContext: func(ctx context.Context, rel string) (string, error) {
							return workspacefs.ReadRemoteText(ctx, fileClient, rel, workspacefs.DefaultMaxReadBytes)
						},
						OnStatWorkspaceFileContext: func(ctx context.Context, rel string) string {
							return workspacePreviewStat(ctx, fileClient, rel)
						},
						OnReadWorkspaceFile: func(rel string) (string, error) {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							return workspacefs.ReadRemoteText(context.Background(), client, rel, workspacefs.DefaultMaxReadBytes)
						},
						FileStatNote: &state.fileStatNote,
						OnStatWorkspaceFile: func(rel string) string {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							st, err := workspacefs.StatRemote(context.Background(), client, rel)
							if err != nil {
								return ""
							}
							note := strings.TrimSpace(st.Version)
							if st.Bytes != nil {
								if note != "" {
									note += " · "
								}
								note += fmt.Sprintf("%.0f B", *st.Bytes)
							}
							if note == "" {
								note = strings.TrimSpace(st.AbsolutePath)
							}
							return note
						},
						OnIndexWorkspace: func(root string) []string {
							if root == "" {
								return nil
							}
							paths, err := workspacefs.IndexRelPaths(root, 4000)
							if err != nil {
								return nil
							}
							state.mu.Lock()
							state.composerPathsRoot = root
							state.composerPaths = paths
							state.mu.Unlock()
							return paths
						},
						OnSearchReferences: func(query string, quoted bool) []session.ReferenceCandidate {
							if sessions == nil || st.State != harness.Ready {
								return nil
							}
							return sessions.ListReferenceCandidates(context.Background(), snap.SelectedID, query, quoted)
						},
						Terminal:          termPanel,
						HostTerminalStore: &state.hostTerminals,
						HostTerminalUI:    &state.hostTerminalUI,
						OnOpenHostTerminal: func() {
							state.mu.Lock()
							state.filesPaneOpen, state.rightPaneTab = true, "host-terminal"
							state.mu.Unlock()
							filesPaneOpen, rightPaneTab = true, "host-terminal"
							if win != nil {
								win.Update(func() {})
							}
						},
						TerminalShellsNote:    termShellsNote,
						CanAddSelectionToChat: sessions != nil && snap.SelectedID != "" && !activeDraft.Sending,
						OnAddSelectionToChat: func(fragment string) error {
							return appendSelectionToComposer(sessions, snap.SelectedID, activeDraft, fragment)
						},
						NewWorkspaceBrowser: func() *browserpreviewui.Controller { return state.newBrowserController(win) },
						BrowserContent: func(c *ui.Context) {
							if state.browserController == nil {
								state.browserController = state.newBrowserController(win)
							}
							state.browserController.ViewDocked(c, themeID, mygo.Theme.IsDark(), func() { rightPaneTab = layout.Workspace.CloseKind("browser"); state.browserController.Hide() })
						},
						OnHideBrowser: func() {
							if state.browserController != nil {
								state.browserController.Hide()
							}
						},
						OnOpenBrowserPreview: func() {
							rightPaneTab = "browser"
							filesPaneOpen = true
						},
						OnOpenMarket: func() {
							state.mu.Lock()
							state.marketOpen = true
							state.scheduleOpen = false
							state.assistantConversationGeneration++
							state.assistantOpen = false
							state.settingsOpen = false
							state.launcherOpen = false
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnOpenSchedule: func() {
							state.mu.Lock()
							state.scheduleOpen = true
							state.marketOpen = false
							state.assistantConversationGeneration++
							state.assistantOpen = false
							state.settingsOpen = false
							state.launcherOpen = false
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnOpenAssistant: func() {
							state.mu.Lock()
							state.assistantConversationGeneration++
							state.assistantOpen = true
							state.marketOpen = false
							state.scheduleOpen = false
							state.settingsOpen = false
							state.launcherOpen = false
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnOpenLauncher: func() {
							state.mu.Lock()
							state.launcherOpen = true
							state.launcherTab = launcher.TabHome
							state.toolsMenuOpen = false
							state.settingsOpen = false
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnContactAccount: state.contactAccountSupport,
						OnOpenSettings: func() {
							state.mu.Lock()
							state.settingsEmbeddedSection = ""
							state.settingsOpen = true
							state.settingsDesktopActive = true
							state.launcherOpen = false
							state.toolsMenuOpen = false
							state.seedDesktopDraft()
							state.mu.Unlock()
							if state.settings != nil {
								state.settings.OpenSettingsDocument(context.Background())
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnToggleTools: func() {
							state.mu.Lock()
							// Electron ▦ opens the files/diff right dock (not a full-width menu).
							next := !state.filesPaneOpen
							state.toolsMenuOpen = false
							state.mu.Unlock()
							state.setFilesPaneOpen(next)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnCloseTerminal: func() {
							state.mu.Lock()
							state.terminalSessions.CloseWorkspace(workspaceRoot)
							state.terminalPanel = nil
							state.terminalVisible = false
							state.layout.Bottom = 0
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnRestartTerminal: func() {
							state.restartTerminal(workspaceRoot, systemDark)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnToggleTerminal: func() {
							state.toggleTerminal(workspaceRoot, systemDark)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnOpenCitationURL:            state.openChatLink,
						VisualReply:                  visualReplyState,
						VisualReplyBitmap:            publishedHTMLBitmap(visualReplyState),
						OnPreviewVisualReply:         func(id string) { previewPublishedHTML(sessions, snap.SelectedID, id) },
						OnInteractVisualReply:        interactivePublishedHTMLHandler(win, sessions, snap.SelectedID, mygo.Theme.IsDark()),
						OnInlineVisualReply:          inlinePublishedHTMLHandler(win, sessions, snap.SelectedID, mygo.Theme.IsDark()),
						OpenUIParent:                 openUIParent,
						OpenUIWindow:                 win,
						OnStopInteractiveVisualReply: sessions.StopInteractiveVisualReply,
						OnReadVisualReply:            func(id string) { readPublishedHTML(sessions, snap.SelectedID, id) },
						OnSaveVisualReply:            func(id string) { savePublishedHTML(win, sessions, snap.SelectedID, id) },
						OnCloseVisualReply:           sessions.CloseVisualReply,
						OnOpenPlanResource: func(doc planpreview.Document) {
							if sessions == nil || sessions.Snapshot().SelectedID != snap.SelectedID {
								return
							}
							state.dshMu.Lock()
							currentClient := state.dshClient
							state.dshMu.Unlock()
							if fmt.Sprintf("%p\x00%s", currentClient, snap.SelectedID) != fileBindingID {
								return
							}
							if err := fileTabs.OpenPlanResource(snap.SelectedID, fileBindingID, doc); err != "" {
								sessions.SetLastAction(err)
								return
							}
							state.mu.Lock()
							state.filesPaneOpen, state.rightPaneTab = true, "files"
							state.mu.Unlock()
							filesPaneOpen, rightPaneTab = true, "files"
							if win != nil {
								win.Update(func() {})
							}
						},
						OnOpenDeliverable: func(path string) {
							if sessions == nil || sessions.Snapshot().SelectedID != snap.SelectedID {
								return
							}
							state.dshMu.Lock()
							currentClient := state.dshClient
							state.dshMu.Unlock()
							if fmt.Sprintf("%p\x00%s", currentClient, snap.SelectedID) != fileBindingID {
								return
							}
							fileTabs.Plans.ShowFiles(snap.SelectedID)
							if mediapreview.Supports(path) {
								state.openMediaFile(sessions, fileClient, snap.SelectedID, snap.SelectedCwd, path, false)
								return
							}
							handled, err := files.OpenPresentedPath(files.Model{
								BindingID: fileBindingID, SessionID: snap.SelectedID,
								WorkspaceRoot: snap.SelectedCwd,
								Tabs:          &fileTabs, SelectedFile: &selectedFile,
								Preview: &filePreview, PreviewErr: &filePreviewErr,
								Editing: &fileEditing, EditDraft: &fileEditDraft,
								EditStatus: &fileEditStatus, StatNote: &state.fileStatNote,
							}, path)
							if err != nil {
								sessions.SetLastAction(err.Error())
								return
							}
							if !handled {
								// Keep remote, missing-cwd and outside-cwd paths intact.
								// Only the explicit card click invokes the existing Host opener.
								if st.State == harness.Ready {
									sessions.OpenWorkspacePath(context.Background(), path, "", "")
								} else {
									sessions.SetLastAction("workspace path open unavailable")
								}
								return
							}
							if workspacefs.ClassifyDoc(selectedFile) == workspacefs.DocHTML {
								state.openLocalHTML(sessions, snap.SelectedID, snap.SelectedCwd, selectedFile)
								return
							}
							if workspacefs.ClassifyDoc(selectedFile) == workspacefs.DocPDF {
								state.openLocalPDF(sessions, snap.SelectedID, snap.SelectedCwd, selectedFile)
								return
							}
							state.mu.Lock()
							state.filesPaneOpen = true
							state.rightPaneTab = "files"
							state.mu.Unlock()
							filesPaneOpen = true
							rightPaneTab = "files"
							if win != nil {
								win.Update(func() {})
							}
						},
						FileOpenWithApps: fileOpenWithApps,
						OnOpenMedia: func(path string, floating bool) {
							state.openMediaFile(sessions, fileClient, snap.SelectedID, snap.SelectedCwd, path, floating)
						},
						OnOpenFloatingDoc: func(path string) {
							kind := browserpreviewui.DocumentHTML
							if imagepreview.SupportsSnapshot(path) {
								kind = browserpreviewui.DocumentImage
							} else if strings.EqualFold(filepath.Ext(path), ".pdf") {
								kind = browserpreviewui.DocumentPDF
							}
							state.openWorkspaceDocument(sessions, snap.SelectedID, snap.SelectedCwd, path, kind, true)
						},
						OnOpenDoc: func(absPath string, kind workspacefs.DocKind, inWebView bool) {
							if inWebView && supportsImagePreview(absPath) {
								state.openLocalImage(sessions, snap.SelectedID, snap.SelectedCwd, absPath)
								return
							}
							if inWebView && kind == workspacefs.DocHTML {
								state.openLocalHTML(sessions, snap.SelectedID, snap.SelectedCwd, absPath)
								return
							}
							if inWebView && kind == workspacefs.DocPDF {
								state.openLocalPDF(sessions, snap.SelectedID, snap.SelectedCwd, absPath)
								return
							}
							if sessions != nil && st.State == harness.Ready {
								sessions.OpenWorkspacePath(context.Background(), absPath, "", "")
								return
							}
							if err := workspacefs.OpenExternal(absPath); err != nil {
								log.Println("open document:", err)
							}
						},
						OnRevealFile: func(absPath string) {
							if sessions != nil && st.State == harness.Ready {
								sessions.OpenWorkspacePath(context.Background(), absPath, "reveal", "")
								return
							}
							_ = desktopcomponents.OpenInFileManager(absPath)
						},
						OnOpenWithApp: func(absPath, appID string) {
							if sessions != nil && st.State == harness.Ready {
								sessions.OpenWorkspacePath(context.Background(), absPath, "", appID)
							}
						},
						OnRenderOfficePDF: func(rel string) {
							state.openOfficePDF(sessions, snap.SelectedID, snap.SelectedCwd, rel)
						},
						OfficePreviewStatus: func(path string) string {
							return state.officePreviewMessage(sessions, snap.SelectedID, path)
						},
					}
					frameModel.OnOpenTerminalTarget = state.terminalTargetHandler(win, frameModel, func(tab string) {
						state.mu.Lock()
						state.filesPaneOpen, state.rightPaneTab = true, tab
						state.mu.Unlock()
						filesPaneOpen, rightPaneTab = true, tab
						if win != nil {
							win.Update(func() {})
						}
					})
					state.consumeNavigationFile(frameModel)
					nextPromptRendered = true
					frame.View(c, frameModel)
					state.workspaceNavigationView(c, win, sessions, themeID, systemDark, func() { state.openNoDirectory(win, sessions, activeDraft, snap.SelectedID) })
					state.directoryPickerView(c, win, sessions, themeID, systemDark, func(mode, path string) {
						if mode == "mention" {
							root := strings.TrimSpace(workspaceRoot)
							if root == "" && state.cfgStore != nil {
								root = strings.TrimSpace(state.cfgStore.Get().Workspace)
							}
							if err := composer.InsertDirectoryMention(&activeDraft.Text, root, path); err != nil && sessions != nil {
								sessions.SetLastAction(i18n.T(i18n.KeyComposerDirOutside))
							}
							if win != nil {
								win.Update(func() {})
							}
							return
						}
						state.applyWorkspacePath(win, sessions, path, false)
					})
					state.mu.Lock()
					// Splitters mutate Sidebar/Right/Main/RightTree on the paint-local
					// copy. Do not clobber Bottom: toggleTerminal writes it on state and
					// a concurrent paint used to wipe the terminal open height back to 0.
					bottom := state.layout.Bottom
					state.layout = layout
					state.layout.Bottom = bottom
					state.composerSuggest = composerSuggest
					state.composerSkillsList = composerSkillsList
					state.composerSkillsOpen = composerSkillsOpen
					state.composerSkillsFilter = composerSkillsFilter
					state.skillCreateName = skillCreateName
					state.composerChipEdit = composerChipEdit
					state.chatList = chatList
					state.draft = activeDraft.Text
					state.editMessageSeq = editMessageSeq
					state.feedbackCommentMsgID = feedbackCommentMsgID
					state.feedbackCommentDraft = feedbackCommentDraft
					state.modelPick = modelPick
					state.permissionPick = permissionPick
					state.agentPresetPick = agentPresetPick
					state.thinkingExpanded = thinkingExpanded
					state.renameSessionID = renameSessionID
					state.renameDraft = renameDraft
					state.sessionSearch = sessionSearch
					if sessions != nil {
						sessions.UpdateSessionSearch(context.Background(), sessionSearch)
					}
					state.fileTabs = fileTabs
					state.gitWorkbench = gitWorkbench
					state.fileExpanded = fileExpanded
					state.selectedFile = selectedFile
					state.fileFilter = fileFilter
					state.filePreview = filePreview
					state.filePreviewErr = filePreviewErr
					state.filePreviewList = filePreviewList
					state.fileEditing = fileEditing
					state.fileEditDraft = fileEditDraft
					state.fileEditStatus = fileEditStatus
					state.fileEditViewMode = fileEditViewMode
					state.fileEditSplit = fileEditSplit
					state.rightPaneTab = rightPaneTab
					state.agentsPromptDraft = agentsPromptDraft
					state.goalDraft = goalDraft
					state.diffPath = diffPath
					state.diffText = diffText
					state.diffErr = diffErr
					state.planBoardOpen = planBoardOpen
					state.todoPanelCollapsed = todoPanelCollapsed
					state.chatPaneTab = chatPaneTab
					state.trajSearch = trajSearch
					state.trajSelectedID = trajSelectedID
					state.trajActualDur = trajActualDur
					state.trajTurnsFold = trajTurnsFold
					state.trajCallsFold = trajCallsFold
					state.trajDetailTab = trajDetailTab
					state.imageLightboxPath = imageLightboxPath
					state.gitCommitOpen = gitCommitOpen
					state.gitCommitDraft = gitCommitDraft
					state.gitPROpen = gitPROpen
					state.mu.Unlock()
				})
				return
			}
			boot.View(c, boot.Model{
				OnCancelRecovery: func() {
					if sup != nil {
						sup.CancelRecovery()
					}
				},
				OnConfigure: func() {
					state.mu.Lock()
					state.launcherOpen = true
					state.mu.Unlock()
					if win != nil {
						win.Update(func() {})
					}
				},
				Status:     st,
				Logs:       logs,
				ThemeID:    themeID,
				SystemDark: mygo.Theme.IsDark(),
				OnRetry: func() {
					if sup != nil {
						state.restartUnlessImport(sup)
					}
				},
				OnExport: func() {
					path, err := harness.ExportBootLogsDialog(win, buildVersion, st, logs, i18n.T(i18n.KeyBootExport))
					if err != nil {
						log.Println("export log:", err)
						return
					}
					if path != "" {
						log.Println(i18n.T(i18n.KeyBootExportSaved), path)
					}
				},
			})
		}
		state.withMainNavigation(c, sharedNavigation, frame.Model{
			Store: sessions, Sizes: &layout, ThemeID: themeID, SystemDark: systemDark,
			SessionSearch: &sessionSearch, RenameSessionID: &renameSessionID, RenameDraft: &renameDraft,
			WorkspaceRoot: workspaceRoot,
		}, func() { paintMainContent("") })
		paintSettingsOverlay()

	})
}

func trayIcon() []byte {
	for _, base := range []string{".", filepath.Dir(os.Args[0])} {
		data, err := os.ReadFile(filepath.Join(base, "resources", "icon.png"))
		if err != nil {
			continue
		}
		if icon, err := scalePNG(data, 32); err == nil {
			return icon
		}
		return data
	}
	log.Println("tray icon not found; running without one")
	return nil
}
