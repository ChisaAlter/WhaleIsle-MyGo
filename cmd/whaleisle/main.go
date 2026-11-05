// Command whaleisle is the Go desktop shell of DeepSeekHarness Go.
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
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/pad"
	"whaleisle/internal/paths"
	"whaleisle/internal/product"
	"whaleisle/internal/recovery"
	"whaleisle/internal/shortcutprefs"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/startuprec"
	"whaleisle/internal/startuptrace"
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
	"whaleisle/internal/ui/projects"
	remoteui "whaleisle/internal/ui/remote"
	scheduleui "whaleisle/internal/ui/schedule"
	costui "whaleisle/internal/ui/sessioncost"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/sidebar"
	termui "whaleisle/internal/ui/terminal"
	"whaleisle/internal/ui/theme"
	usageui "whaleisle/internal/ui/usage"
	webviewui "whaleisle/internal/ui/webview"
)

type mainState struct {
	remoteWorkspaceLease  atomic.Uint64
	childComposerBindings map[string]string
	padDesktop            *pad.Desktop
	padDesktopAttempted   bool
	pluginComposerOwner   *nativePluginComposerOwner
	fileOpenWithOwner     fileOpenWithOwner
	fileOpenWithState     files.OpenWithState
	// pendingToast holds at most one message for the next painted frame; see
	// notifyUser. It is never read from background timers.
	pendingToast          string
	sessionLogDownloads   map[sessionLogExportKey]*sessionLogExportOperation
	voiceDevices          *composerVoiceDevices
	workspaceOpenInApp    *frame.WorkspaceOpenInAppState
	svgImages             *nativeSVGImages
	svgImagesPaused       bool
	nativePlugins         nativePluginManagerState
	customPanels          customPanelState
	applicationMenu       *desktopApplicationMenu
	wallpaperControls     *settings.WallpaperEditor
	wallpaperVisible      bool
	commandSearchFocus    bool
	appearanceEditor      *theme.Editor
	appearanceVisible     bool
	shortcutEditor        *shortcuts.Editor
	shortcutPreferences   *shortcutprefs.Store
	shortcutRevision      string
	shortcutEditorSeq     uint64
	wallpaperChoicesCache []string
	wallpaperChoicesReady bool
	// shortcutBindingsCache holds the accepted bindings for this state; see
	// pageShortcutBindings.
	shortcutBindingsMu    sync.Mutex
	shortcutBindingsCache shortcutBindingsCache
	projectWindowHidden   bool
	startupRecovery       startupRecoveryState
	voice                 *composerVoiceOperation
	projectFlow           projectFlowState
	featurePreferences    featurePreferenceState
	resumeStore           *recovery.Store
	resumeCheckpoint      *resumeCheckpoint
	resumeError           string
	resumePendingSession  string
	resumeWorkspace       string
	mu                    sync.Mutex
	win                   *mygo.Window
	taskRestartPending    bool
	quitting              bool
	phonePopupOpen        bool
	phoneRuntime          *phoneremote.Runtime
	phoneDeviceEditor     remoteui.PhoneDeviceEditor
	phoneGatewayEditor    remoteui.GatewayEditor
	phoneGitSession       string
	phoneGitRoot          string
	phoneGitWorkspaces    phoneGitWorkspaceScope
	phoneOp               sync.Mutex
	closeToTray           bool
	desktopTray           *mygo.Tray
	trayHideNotice        trayHideNotice
	trayHideNotification  *mygo.Notification
	welcomeWin            *welcomeWindow
	// whaleBridgeWin is the single embedded WhaleBridge management window
	// (port of reference/electron/src/main/whalebridge-window.js).
	whaleBridgeWin         *whaleBridgeWindow
	credsMissing           bool
	welcomeDone            bool
	bridgeAlerts           *bridgeAlertUI
	cfgStore               *config.Store
	viewPreferences        *workspaceViewPreferences
	desktopPet             *pet.Companion
	settingsLaunchHandled  bool
	petEvents              petEventController
	petPersonality         petPersonalityMirror
	navigationFile         *navigationFileRequest
	themeID                string
	hostAppearance         *settings.HostThemeBinding
	hostAppearanceOwner    string
	hostAppearanceAccepted bool
	hostWallpaperImage     string
	confirmAsk             *confirmAskRequest
	// confirmAskTechnicalOpen is the disclosure state of the queued question's
	// evidence block; askConfirm resets it for each new request.
	confirmAskTechnicalOpen bool
	desktopOnboarding       *settingsstore.DesktopOnboarding
	sup                     *harness.Supervisor
	bootStatus              harness.Status
	bootLogs                []string
	// crashReportGeneration is the failed supervisor generation that already
	// wrote a host crash report, so one failure writes one file.
	crashReportGeneration int
	crashReportPath       string

	sessions                                *session.Store
	frameViews                              sessionFrameViews
	agentPresetManager                      *session.AgentPresetManager
	creatorTask                             *session.CreatorTask
	pluginCreatorTask                       *session.CreatorTask
	productShortcutCloses                   []productShortcutClose
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
	settingsConversationActive              bool
	settingsFeaturesActive                  bool
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
	settingsSkillHub                        settingsSkillHubState
	settingsSkillsLastAction                string
	settingsSkillsDetail                    string
	settingsSkillEditSession                string
	settingsSkillInspectVersion             uint64
	settingsSkillEditGroups                 []string
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
	whaleCatalog                            whaleCatalogState
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
	// Phone-remote tokens copied from the credential store for the daemon
	// config. Secrets: never logged and never written to config.json.
	remoteToken                     string
	remoteRelayToken                string
	remoteDeviceCount               int
	remoteDeviceNames               []string
	phoneActionMsg                  string
	sshProbeLine                    string
	webviewOpen                     bool
	browserPreviewOpen              bool
	officePreview                   officePreviewState
	pdfPreviewRequest               uint64
	mediaPreviewRequest             uint64
	htmlDocumentRequest             uint64
	browserController               *browserpreviewui.Controller
	browserPreviewDraft             string
	browserPreviewNavigateErr       string
	webviewHost                     *desktopwebview.Host
	settingsOverlayHost             *desktopwebview.Host
	webviewHostErr                  string
	lastWebViewURL                  string
	lastWebViewOrigin               string
	pluginViewResumeAfterRestart    bool
	pluginViewHostGeneration        int
	assistantOpen                   bool
	assistantConversationGeneration uint64
	assistantDraft                  string
	assistantLastAct                string
	assistantPluginBusy             bool
	assistantPluginDetail           string
	launcherOpen                    bool
	// launcherWin is the separate 1060x660 launcher window (G04). Nil keeps the
	// in-window launcher panel: isolated/QA desktops, --background, --settings,
	// headless tests and a failed window creation all stay on that fallback.
	launcherWin *launcherWindow
	// launcherWindowAllowed is decided once at startup: only a cold foreground
	// start hides the main window behind a separate launcher window.
	launcherWindowAllowed   bool
	toolsMenuOpen           bool
	filesPaneOpen           bool // Electron tools → files/diff right dock
	launcherTab             string
	launcherWorkspaceDraft  string
	launcherImport          string
	launcherImportJournal   []string
	launcherImportBusy      bool
	launcherImportCancel    context.CancelFunc
	launcherUpdate          string
	launcherUpdateCheckBusy bool
	updateCheckPending      bool
	// updateProgress is the visible update row; updateJournal keeps the update
	// evidence outside the installation directory (port of the reference shell
	// update dialog and update-journal.js).
	updateProgress          *updateProgressState
	updateJournal           updateJournalBook
	launcherPrefsError      string
	nativeReleases          nativeReleaseState
	launcherUpdateAvailable bool
	launcherUpdateSynced    bool
	launcherNote            string
	launcherTriedStart      bool
	launcherStartBusy       bool
	launcherStartPreparing  bool
	launcherStartCancel     context.CancelFunc
	launcherStartFailed     bool
	launcherStartErr        string
	launcherPrefsSeeded     bool
	launcherQuitAfterStart  bool
	// phoneBootStarted gates the once-per-session phone gateway start when the
	// harness first becomes ready; phoneBootStartAction is nil in production
	// (the real start) and replaced by a counter in tests.
	phoneBootStarted              bool
	phoneBootStartAction          func()
	phoneDiagSampling             bool
	launcherAutoStartDesktop      bool
	launcherAskOnUpdate           bool
	launcherDownloadRoute         string
	launcherImportOverwrite       bool
	launcherImportAttachments     bool
	launcherImportCategories      map[string]bool
	launcherImportCatalog         *importer.Catalog
	launcherImportScan            importCatalogScan
	launcherComponents            []desktopcomponents.Row
	launcherComponentsFresh       bool
	launcherComponentsRefreshBusy bool
	launcherComponentStatus       string
	launcherHarnessVersion        string
	quitRequestPending            bool
	quitCleanupFinished           bool
	closingOverlay                bool
	launcherRoutePopOpen          bool
	workspaceMenuOpen             bool
	workspacePickerAnchor         ui.Handle
	workspaceRenameDraft          string
	pendingDeeplink               deeplink.Target
	settingsDrafts                []settingsDraftSlot
	workspaceSessions             workspaceSessionLayouts
	layout                        frame.LayoutSizes
	chatList                      ui.ListState
	draft                         string
	editMessageSeq                float64
	feedbackCommentMsgID          string
	feedbackCommentDraft          string
	modelPick                     string
	permissionPick                string
	agentPresetPick               string
	thinkingExpanded              map[string]bool
	renameSessionID               string
	renameDraft                   string
	sessionSearch                 string

	gitBranch           string
	gitDirty            bool
	gitRepositoryStatus *desktopgit.RepositoryStatus
	gitMsg              string
	gitBusy             bool
	gitProgress         *desktopgit.Progress
	gitCommitOpen       bool
	gitCommitDraft      string
	gitCheckedRoot      string
	gitCheckedAt        time.Time
	gitRefresh          *gitRefreshRead
	gitPROpen           bool
	gitPRBusy           bool
	gitPRReport         desktopgit.PRReport
	gitPRRoot           string
	gitPRAt             time.Time
	// gitCommitGenerating owns the in-flight title-bar message generation; the
	// draft stays user-editable while it runs (port of git-generate.js).
	gitCommitGenerating bool
	gitCommitGenErr     string
	// gitCommitGenPending is the generated copy handed to the next painted
	// frame, so a worker that finishes mid-frame cannot be overwritten by that
	// frame's draft save.
	gitCommitGenPending *string

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

	schedulePanel         scheduleui.Panel
	schedulePanelRendered bool
	scheduleEditID        string
	scheduleEditTitle     string
	scheduleEditPrompt    string

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

	composerResize     composer.ResizeState
	sessionCostState   costui.State
	composerSuggestSel int
	composerSuggest    ui.ListState
	composerPathsRoot  string
	composerPaths      []string
	// composerPathsLoading is the root a worker is indexing; the Failed pair
	// holds off retrying a root whose index failed (see composerPathsLocked).
	composerPathsLoading    string
	composerPathsFailedRoot string
	composerPathsRetry      time.Time
	composerSlash           []composer.SlashCommand
	composerSlashKey        string
	composerSkillsOpen      bool
	composerSkillsSel       int
	composerSkillsList      ui.ListState
	composerSkillsFilter    string
	skillCreateName         string
	composerSkillsSession   string
	composerChipEdit        composer.ChipEdit
	composerTextInput       *composer.DraftClient
	composerDrafts          map[string]*composer.DraftState
	workspaceSessionOpen    workspaceSessionOpenState
	startupSelection        startupSelectionState
	projectPreference       projectPreferenceState

	terminalPanel         *termui.Panel
	terminalSessions      termui.Sessions
	hostTerminals         hostterminal.Store
	hostTerminalUI        termui.HostState
	terminalShellsNote    string
	terminalVisible       bool
	terminalSavedBottom   float32
	terminalResumePending bool

	dshMu             sync.Mutex
	dshCancel         context.CancelFunc
	dshEndpoint       string
	dshClient         *dsh.Client
	dshMux            *dsh.Mux
	dshMuxReady       bool
	dshMuxLost        bool
	dshBootstrapError string
	dshNetworkOffline bool
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
			public := desktopPublicConfig(cfg)
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
		ApplyConfig: state.applyDesktopConfigPatch,
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
	workspaceSeed             string
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

func effectiveGoPort(cfg config.Config) int {
	if cfg.GoPort >= 1024 && cfg.GoPort <= 65535 && cfg.GoPort != 3080 {
		return cfg.GoPort
	}
	if cfg.Port >= 1024 && cfg.Port <= 65535 && cfg.Port != 3080 {
		return cfg.Port
	}
	return config.DefaultGoPort
}

func effectiveGoRemotePort(cfg config.Config) int {
	if cfg.GoRemotePort >= 1024 && cfg.GoRemotePort <= 65535 && cfg.GoRemotePort != 6767 {
		return cfg.GoRemotePort
	}
	if cfg.RemotePort >= 1024 && cfg.RemotePort <= 65535 && cfg.RemotePort != 6767 {
		return cfg.RemotePort
	}
	return config.DefaultGoRemotePort
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
		workspaceSeed:             cfg.Workspace,
		port:                      strconv.Itoa(effectiveGoPort(cfg)),
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
	s.remoteToken = strings.TrimSpace(creds.RemoteToken)
	s.remoteRelayToken = strings.TrimSpace(creds.RemoteRelayToken)
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
		s.phoneRuntime.SetHostActions(phoneremote.HostActions{OpenSettings: s.openPhoneSettingsSection, SaveConfig: s.savePhoneConfigPatch})
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
		return i18n.T(i18n.KeyCmdRemoteConfigUnavailable)
	}
	cfg := s.cfgStore.Get()
	s.mu.Lock()
	ep := s.bootStatus.Endpoint
	quitting := s.quitting
	creds := remoteRuntimeCredentials{Local: s.remoteToken, Relay: s.remoteRelayToken}
	s.mu.Unlock()
	if quitting {
		return "offline"
	}
	if !cfg.RemoteEnabled {
		manager.Stop()
		return "disabled"
	}
	gitRoots := s.phoneGitRoots()
	cfgForRemote := cfg
	cfgForRemote.RemotePort = effectiveGoRemotePort(cfg)
	runtimeConfig := buildRemoteRuntimeConfig(cfgForRemote, creds, gitRoots, ep.Origin, ep.Cookie)
	if manager.Snapshot().Status == "starting" {
		return "starting"
	}
	if manager.Snapshot().Listening {
		manager.Refresh(runtimeConfig)
	} else {
		manager.Start(paths.UserDataDir(), paths.VendorDir(), runtimeConfig)
	}
	snapshot := manager.Snapshot()
	s.writePhoneDiag(snapshot)
	s.startPhoneDiagSampler(manager)
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
		s.notifyUser(s.win, hostNotReadyToast())
		return
	}
	port, err := strconv.Atoi(strings.TrimSpace(s.desktopDraft.port))
	if err != nil || port < 1024 || port > 65535 {
		s.desktopSaveErr = i18n.T(i18n.KeyCmdDesktopInvalidPort)
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
	workspaceChanged := strings.TrimSpace(s.desktopDraft.workspace) != strings.TrimSpace(s.desktopDraft.workspaceSeed)
	err = s.cfgStore.Update(func(c *config.Config) {
		restartPolicy.Apply(c)
		if workspaceChanged {
			c.Workspace = strings.TrimSpace(s.desktopDraft.workspace)
		}
		c.GoPort = port
		if !s.hostAppearanceAccepted {
			c.Theme = s.desktopDraft.theme
		}
		c.Locale = s.desktopDraft.locale
		c.CloseToTray = s.desktopDraft.closeToTray
		c.OpenAtLogin = s.desktopDraft.openAtLogin
		c.AutoStartDesktop = s.desktopDraft.autoStartDesktop
		if !s.hostAppearanceAccepted {
			c.WallpaperPath = strings.TrimSpace(s.desktopDraft.wallpaperPath)
		}
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
	s.desktopDraft.workspace, s.desktopDraft.workspaceSeed = cfg.Workspace, cfg.Workspace
	if config.RestartPreferencesFrom(cfg) != restartPolicy {
		s.desktopSaveErr = i18n.T(i18n.KeyCmdRestartPolicyNotSaved)
		s.desktopLastAction = ""
		return
	}
	s.desktopSaveErr = ""
	s.desktopLastAction = i18n.T(i18n.KeySettingsDesktopSaved)
	s.closeToTray = cfg.CloseToTray
	if !s.hostAppearanceAccepted {
		s.themeID = cfg.Theme
	}
	i18n.SetLocale(cfg.Locale)
	if !s.hostAppearanceAccepted {
		applyAppearance(cfg.Theme)
	}
	if err := desktopSystemIntegration(func() error { return mygo.App.SetOpenAtLogin(cfg.OpenAtLogin) }); err != nil {
		s.desktopLastAction = ""
		s.desktopSaveErr = fmt.Sprintf("%s: %v", i18n.T(i18n.KeyCmdSettingsSavedButStartupRegistrationFai), err)
		log.Println("openAtLogin:", err)
	}
	if workspaceChanged && s.sessions != nil {
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
	s.registerWorkspacePath(win, sessions, ws)
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
	lw := s.launcherWin
	open := s.launcherOpen
	s.mu.Unlock()
	if open && !lw.destroyed() && (win == nil || !win.IsVisible()) {
		lw.show()
		return
	}
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
	// Session and Agents links navigate to the conversation workspace. Tool
	// pages return before that workspace is painted, so retire their routes.
	// A plain Open link continues to raise the current page without navigating.
	if t.Action == deeplink.ActionSession || t.Action == deeplink.ActionAgents {
		s.imOpen, s.remoteOpen, s.marketOpen = false, false, false
		s.webviewOpen, s.browserPreviewOpen = false, false
		s.assistantOpen, s.toolsMenuOpen = false, false
		s.assistantConversationGeneration++
	}
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
		s.settingsConversationActive = false
		s.settingsFeaturesActive = false
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
			s.settingsConversationActive = false
			s.settingsFeaturesActive = false
			s.seedDesktopDraft()
		case "conversation", "chat":
			s.settingsConversationActive = true
		case "features", "feature":
			s.settingsFeaturesActive = true
		case "models", "model":
			s.settingsModelsActive = true
		case "remote", "phone":
			// Remote pairing lives in the embedded 远程 section, not a separate page.
			s.settingsEmbeddedSection = settings.SectionRemote
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
		s.settingsConversationActive = false
		s.settingsFeaturesActive = false
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
	if t.Action == deeplink.ActionLauncher {
		// The launcher deep link opens/reuses the one launcher window
		// (index.js:340-348); the panel fallback still needs the main window.
		s.syncLauncherStage()
	} else {
		// Every other route lands on the main window; an open launcher window
		// would mask it.
		s.closeLauncherWindow()
		s.showMainWindow()
	}
	if t.Action == deeplink.ActionAuth && usage != nil {
		sid := ""
		if sessions != nil {
			sid = sessions.SelectedSessionID()
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
	if whale != nil && dshbot == nil {
		s.setWhaleAssistantPreference(*whale)
		return
	}
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
		s.mu.Lock()
		s.desktopSaveErr = err.Error()
		s.mu.Unlock()
		if win != nil {
			win.Update(func() {})
		}
		return
	}
	s.mu.Lock()
	s.desktopSaveErr = ""
	s.mu.Unlock()
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
		asst.SetLastAction(i18n.T(i18n.KeyCmdSelectCurrentRosterBot), i18n.T(i18n.KeyCmdComposerHostChanged))
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
		s.webviewHostErr = i18n.T(i18n.KeyCmdURLUnavailable)
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
		Untrusted:             true,
		SessionOrigin:         sessionOrigin,
		ShortcutPreferences:   s.pageShortcutPreferences(),
		OnShortcutCloseWindow: s.queueProductShortcutClose,
		SessionCookie:         sessionCookie,
		OnChange:              func() { win.Update(func() {}) },
		OnFailure:             s.onEmbeddedPageFailure,
		Parent:                win,
		URL:                   browse,
		Title:                 hostTitle,
		TopInset:              inset,
		// Native media policy: the loopback Harness session may ask for the
		// microphone, the camera is refused, notifications follow the shell gate.
		PermissionPolicy: desktopwebview.PermissionPolicy{
			SessionHost:          strings.TrimSpace(sessionOrigin) != "",
			NotificationsAllowed: systemNotificationsSupported,
		}.Decide,
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.webviewHost = nil
		s.webviewHostErr = err.Error()
		// A hosted document that never came up is this shell's nearest
		// web-boot surface (MyGo has no boot web process): keep a diagnostic.
		go state.recordWebBootFailure(err.Error())
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
	s.terminalResumePending = false
	if s.terminalVisible {
		s.flushTerminalResizeLocked()
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
	if !s.terminalVisible || s.layout.Bottom <= terminalMinBottom || s.terminalResumePending {
		s.flushTerminalResizeLocked()
		return
	}
	if s.terminalPanel != nil && s.terminalPanel.Workspace() == root {
		return
	}
	s.flushTerminalResizeLocked()
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
	s.terminalResumePending = false
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
	s.showElectronImportReportNative(win)
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
		s.terminalResumePending = false
		s.terminalVisible = true
		if s.layout.Bottom <= terminalMinBottom {
			if s.terminalSavedBottom > terminalMinBottom {
				s.layout.Bottom = s.terminalSavedBottom
			} else {
				s.layout.Bottom = frame.DefaultLayout().Bottom
			}
		}
	default:
		if sid, ok := strings.CutPrefix(cmd, "session:"); ok && strings.TrimSpace(sid) != "" {
			target := strings.TrimSpace(sid)
			s.todoPanelCollapsed = true
			s.startupSelection.consumed = true
			s.workspaceSessionOpen.generation++
			go func() {
				var sessions *session.Store
				var snap session.ViewState
				for i := 0; i < 100; i++ {
					s.mu.Lock()
					sessions = s.sessions
					s.mu.Unlock()
					if sessions != nil {
						snap = sessions.Snapshot()
						if len(snap.Items) > 0 {
							break
						}
					}
					time.Sleep(100 * time.Millisecond)
				}
				if sessions == nil {
					return
				}
				resolved := target
				for _, it := range snap.Items {
					if strings.EqualFold(it.SessionID, target) ||
						strings.EqualFold(strings.TrimPrefix(it.SessionID, "session-"), strings.TrimPrefix(target, "session-")) ||
						strings.Contains(strings.ToLower(it.Title), target) {
						resolved = it.SessionID
						break
					}
				}
				owners := session.WorkspaceSessionOwners(snap.Workspaces, snap.Items)
				if wsID := owners[resolved]; wsID != "" && session.WorkspaceGroupCollapsed(snap, wsID) {
					sessions.ToggleWorkspaceGroup(wsID)
				}
				sessions.Select(resolved)
				s.mu.Lock()
				w := s.win
				s.mu.Unlock()
				if w != nil {
					w.Update(func() {})
				}
			}()
			break
		}
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
				// A route may hand the window to the launcher or back to the
				// desktop; the move is its own main-thread task, never a nested
				// content update.
				mygo.RunOnMain(func() { s.syncLauncherStage() })
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
	if s.bootStatus.Generation != st.Generation || s.bootStatus.State != st.State || s.bootStatus.Endpoint != st.Endpoint {
		s.remoteWorkspaceLease.Add(1)
	}
	s.bootStatus = st
	startuptrace.Mark("boot.status", st.State.String())
	persistenceFailed := s.recordStartupStatusLocked(st)
	if st.State == harness.Ready {
		// The startup outcome file is what an external observer reads as Ready, so
		// this mark is the end of the measured cold-start chain.
		startuptrace.Mark("boot.outcome_recorded")
		startuptrace.Finish("ready")
	}
	// A fatal local-service failure writes one bounded diagnostic for the error
	// surface to name (native crash-report.js). One file per failure: keyed by
	// the supervisor generation, and by the pre-supervisor error text otherwise.
	failedMessage := ""
	if st.State == harness.Failed {
		key := st.Generation
		if key == 0 {
			key = -1
		}
		if st.Err != nil && key != s.crashReportGeneration {
			s.crashReportGeneration = key
			failedMessage = st.Err.Error()
		}
	} else {
		s.crashReportGeneration = 0
	}
	// Starting the phone gateway is decided under the same lock as the ready
	// transition, then run after unlock (it spawns the daemon and takes phoneOp).
	startPhone := false
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
		startPhone = s.claimPhoneBootStartLocked()
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
	logs := append([]string(nil), s.bootLogs...)
	win := s.win
	lw := s.launcherWin
	s.mu.Unlock()
	// The native shell starts its phone gateway as soon as the config with
	// remoteEnabled is in effect; MyGo had no such call, so a boot-time enable
	// never started the daemon. Start it once per session, on the first Ready.
	if startPhone {
		if s.phoneBootStartAction != nil {
			s.phoneBootStartAction()
		} else {
			go s.refreshPhoneRemote()
		}
	}
	if failedMessage != "" {
		// Best effort: a report that cannot be written never masks the failure.
		if path := recordShellCrash("host", failedMessage, strings.Join(logs, "\n")); path != "" {
			s.publishCrashReport(win, path)
		}
	}
	s.syncPluginViewHost(st)
	s.syncDialogue(st)
	s.observeMarketActivation(st)
	if !lw.destroyed() {
		lw.win.Update(func() {})
	}
	if win != nil {
		win.Update(func() { s.refreshPetTask(); s.refreshDesktopTray(); s.refreshStartupUpdateAttention(win) })
	}
	// Window moves run as their own main-thread task, never inside a content
	// update: creating or destroying a native window from a paint callback
	// re-enters the message loop and wedges it.
	mygo.RunOnMain(func() { s.syncLauncherStage() })
}

// claimPhoneBootStartLocked decides the once-per-session phone gateway start
// when the harness first reports ready. It mirrors the native shell, which
// starts the remote daemon from the effective config rather than waiting for a
// settings action. Caller must hold s.mu. Disabled config, a quitting shell and
// repeated Ready frames (harness restarts included) must not start it again;
// the user-facing enable/disable path keeps owning later changes.
func (s *mainState) claimPhoneBootStartLocked() bool {
	if s.phoneBootStarted || s.quitting || s.cfgStore == nil {
		return false
	}
	if !s.cfgStore.Get().RemoteEnabled {
		return false
	}
	s.phoneBootStarted = true
	return true
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
	s.mu.Lock()
	s.dshMuxReady = false
	s.dshMuxLost = false
	s.dshBootstrapError = ""
	s.dshNetworkOffline = false
	s.mu.Unlock()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}}
	s.replaceMCPControllerLocked()
	s.settingsProviderProfiles.ResetHost()
	s.remoteWorkspaceLease.Add(1)
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
	interactionConfig := s.cfgStore
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

	// A retained interaction store still owns the previous Host's refresh hook.
	// SetAutoAllow publishes synchronously; that hook may re-enter mainState.
	if interactionConfig != nil {
		inter.SetAutoAllow(interactionConfig.Get().AutoAllowTools)
	}
	var refreshQueued atomic.Bool
	refresh := func() {
		s.cancelComposerVoiceIfStale()
		if win != nil {
			if refreshQueued.CompareAndSwap(false, true) {
				win.Update(func() {
					refreshQueued.Store(false)
					s.refreshPetTask()
					s.maybeStartupBlankSession(win, sessions)
				})
			} else {
				win.Invalidate()
			}
		}
	}
	sessions.Attach(client, refresh)
	s.bindWorkspacePreferences(sessions, client)
	chat.SetAttachmentResolver(sessions.CaptureAttachmentRead)
	setStore.Attach(client, refresh)
	s.refreshPluginInventory()
	s.refreshReferencedCredentials()
	s.refreshProviderDiscovery()
	usageStore.Attach(client, refresh)
	usageStore.LoadBilling(context.Background())
	remoteMachines.Attach(client, refresh)
	imStore.Attach(client, refresh)
	marketStore.SetUpdateProfileDir(s.managedMarketProfile(st))
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
		s.bindQuestionDraftRecovery(inter)
	}
	go s.runDSH(ctx, client, mux, inter)
	sessions.InitializeDefaultModel(context.Background())
	sessions.RefreshModelCatalog(context.Background())
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
	s.cancelSessionLogExportsLocked()
	s.stopProjectProcessLocked()
	s.invalidateSettingsSkillCreateLocked()
	s.invalidateSettingsSkillHubLocked()
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
	s.remoteWorkspaceLease.Add(1)
	s.dshClient = nil
	s.bindNativePluginManager(nil)
	s.clearAccountReturn()
	s.accountSignOut.Bind(nil)
	s.attachPluginInventory(nil)
	s.settingsReferencedCredentials.Attach(nil)
	s.settingsProviderDiscovery.Attach(nil)
	s.dshMux = nil
	s.mu.Lock()
	s.dshMuxReady = false
	s.dshMuxLost = false
	s.dshBootstrapError = ""
	s.dshNetworkOffline = false
	s.mu.Unlock()
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
		s.market.SetUpdateProfileDir("")
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
	connection := s.newConnectionAttemptState(ctx, client, mux, refresh)
	stopNetwork := connection.watchNetwork(ctx)
	defer stopNetwork()
	_ = dsh.MaintainMuxFromBootstrap(ctx, mux, client, func(ctx context.Context) error {
		if !connection.begin(ctx) {
			return context.Canceled
		}
		generationDone := make(chan struct{})
		if !mux.BindGeneration(ctx, generationDone) {
			close(generationDone)
			return context.Canceled
		}
		var scheduleEventsOwner *schedstore.EventsOwner
		if schedStore != nil {
			scheduleEventsOwner = schedStore.BeginEvents(ctx, client)
		}
		var eventsOwner *session.EventsOwner
		if sessions != nil {
			eventsOwner = sessions.BeginEvents(ctx, client)
		}
		var delivery *interaction.EventsDeliveryOwner
		if inter != nil {
			delivery = inter.BeginEventsDelivery(ctx, client, mux)
		}
		go func() {
			defer close(generationDone)
			if inter != nil {
				defer inter.RetireEventsDelivery(delivery)
			}
			err := client.FollowEvents(ctx, mux, dsh.EventsFuncs{
				Ready: func(_ dsh.EventsReady) {
					connection.ready(ctx, func() {
						pluginEvent(dsh.EventsEmit{Event: "plugin-manager/changed"})
						pluginEvent(dsh.EventsEmit{Event: "plugin-market/changed"})
						if schedStore != nil {
							schedStore.OnEventsReady(scheduleEventsOwner)
						}
						if sessions != nil {
							sessions.InitializeDefaultWorkspace(ctx)
							sessions.OnEventsReady(eventsOwner)
							sessions.WatchSpeechCatalogForEvents(eventsOwner)
							sessions.WatchNativeContributionsForEvents(eventsOwner)
							sessions.RefreshPermissionCatalogForEvents(eventsOwner)
						}
					})
				},
				Emit: func(emit dsh.EventsEmit) {
					connection.withCurrent(ctx, func() {
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
					})
				},
				Waterfall: func(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
					if inter == nil {
						return nil, nil
					}
					return inter.HandleWaterfallForDelivery(delivery, wf)
				},
				Result: func(result dsh.EventResult, err error) bool {
					if inter != nil {
						return inter.OnResultForDelivery(delivery, result, err)
					}
					return false
				},
				Cancel: func(c dsh.EventsCancel) {
					if inter != nil {
						inter.OnCancelForDelivery(delivery, c)
					}
				},
				Retire: func() {
					if inter != nil {
						inter.RetireEventsDelivery(delivery)
					}
				},
			})
			if ctx.Err() == nil {
				connection.retry(ctx, err)
			}
		}()
		if sessions != nil {
			sessions.FollowWorkspace(ctx)
			sessions.FollowSelected(ctx)
			sessions.WatchControl(ctx)
			sessions.RefreshProjections(ctx)
		}
		go settingsstore.WatchAnalyticsPolicy(ctx, mux, func(on bool) {
			connection.withCurrent(ctx, func() {
				s.mu.Lock()
				s.analyticsEnabled = &on
				s.mu.Unlock()
				refresh()
			})
		})
		return nil
	}, func(err error) {
		if err != nil {
			log.Print("[connection] lost: ", err)
		}
		connection.lost()
	}, func(warning error) {
		log.Print("[connection] ", warning)
	}, func(err error) {
		connection.bootstrap(err)
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
	if handleNativeRecoveryProcess() {
		return
	}
	// WHALEISLE_STARTUP_TRACE is the opt-in cold-start phase timer; this is the
	// earliest Go code the process runs.
	startuptrace.Mark("process.start")
	startUIDiagnostics()
	startEditorProbe()
	// WHALEISLE_STACK_DUMP is a diagnostic hook: after a few seconds it writes
	// every goroutine's stack to the named file, so a wedged UI thread can be
	// inspected from a headless run. Unset in normal operation.
	if dump := strings.TrimSpace(os.Getenv("WHALEISLE_STACK_DUMP")); dump != "" {
		go func() {
			time.Sleep(8 * time.Second)
			if f, err := os.Create(dump); err == nil {
				_ = pprof.Lookup("goroutine").WriteTo(f, 2)
				_ = f.Close()
			}
		}()
	}
	mygo.App.SetName(product.Name)
	if err := preparePortableDesktop(); err != nil {
		log.Println("portable desktop:", err)
		return
	}
	configureDesktopName()
	if err := prepareIsolatedDesktop(); err != nil {
		log.Println("isolated desktop:", err)
		return
	}
	startuptrace.Mark("shell.prepared")
	startuptrace.Meta("isolated", strconv.FormatBool(isolatedDesktopRun()))
	startuptrace.Meta("userData", paths.UserDataDir())
	startuptrace.Meta("vendor", paths.VendorDir())

	if !receiveSystemToastActivation() {
		return
	}
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	// A stale stop request must not cold-start the desktop.
	if len(os.Args) > 1 && os.Args[1] == "--managed-launcher-stop" {
		return
	}
	releaseProfile, err := claimDesktopProfile()
	if err != nil {
		log.Println("desktop profile:", err)
		_, _ = mygo.Dialog.Message(mygo.MessageOptions{Type: mygo.MessageError, Title: product.Name, Message: err.Error()})
		return
	}
	defer releaseProfile()
	mygo.App.OnOpenURL(func(rawURL string) {
		if t, ok := deeplink.Parse(rawURL); ok {
			state.applyDeeplink(t)
			return
		}
		log.Print("deeplink: ignored unsupported route")
	})
	if err := desktopSystemIntegration(func() error { return mygo.App.RegisterURLScheme(deeplink.Scheme) }); err != nil {
		log.Println("RegisterURLScheme:", err)
	}
	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		if consumeSystemToastActivation(args) {
			return
		}
		if managedLauncherStop(args) {
			state.quitDesktopTray()
			return
		}
		if slices.Contains(args, "--background") {
			return
		}
		state.showMainWindow()
	})

	go func() {
		theme.EnsureSkinFonts()
		chrome.WarmBrandAssets()
		_ = windowIcon()
		_ = trayIcon()
		chrome.WarmHeroBrandInks()
	}()

	mygo.App.WhenReady(func() {
		startuptrace.Mark("app.when_ready")
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
			sidebar.SetRemoteImageCache(cache)
			settings.SetAccountImageCache(cache)
			files.SetRemoteImageCache(cache)
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
		startuptrace.Mark("stores.opened")

		cfgStore, err := config.Open(userData)
		if err != nil {
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{Type: mygo.MessageWarning, Title: i18n.T(i18n.KeyCmdConfigurationLoadFailed), Message: err.Error() + "\n\nThe original file is preserved. Back up and repair config.json, then restart. / 原文件已保留，请备份并修复 config.json 后重启。"})
			mygo.App.Quit()
			return
		}
		if p, err := cfgStore.Appearance(); err == nil {
			theme.SetPreferences(p)
		}
		go func(themeID string) {
			th := theme.Resolve(themeID)
			for _, dark := range []bool{false, true} {
				c := th.Active(dark)
				_ = chrome.HeroBrandMark(chrome.Color(c.FG))
				_ = chrome.HeroBrandMark(chrome.Color(c.Muted))
			}
		}(cfgStore.Get().Theme)
		// Window-scoped Ctrl+Shift+I / Cmd+Alt+I, gated like the native shell
		// (devtools-shortcut.js:16-19): development always allows it, packaged
		// builds only with the openDevTools setting on.
		desktopwebview.SetDevToolsShortcutAllowed(func() bool {
			return !mygo.App.IsPackaged() || cfgStore.Get().OpenDevTools
		})
		state.openStartupRecovery(userData, cfgStore)
		cfg := cfgStore.Get()
		state.hostTerminals.ConfigureShellPreference(cfg.HostTerminalShell, func(path string) error {
			return cfgStore.Update(func(c *config.Config) { c.HostTerminalShell = path })
		})
		i18n.SetLocale(cfg.Locale)
		accountlogin.SetClientVersion(buildVersion)
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
			// Electron welcome-window gate: a profile without a stored API key
			// sees the welcome window before the launcher/boot flow.
			state.credsMissing = strings.TrimSpace(creds.APIKey) == ""
			_, markerErr := os.Stat(filepath.Join(userData, welcomeMarker))
			state.welcomeDone = markerErr == nil
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
		startuptrace.Mark("config.loaded")

		if err := desktopSystemIntegration(func() error { return mygo.App.SetOpenAtLogin(cfg.OpenAtLogin) }); err != nil {
			log.Println("openAtLogin:", err)
		}

		hidden := isolatedDesktopHidden() || !isolatedDesktopRun() && mygo.App.WasOpenedAtLogin() && cfg.CloseToTray
		background := !isolatedDesktopRun() && slices.Contains(os.Args, "--background")
		// The reference shell never creates the main window while the launcher
		// owns the stage (window.js:900-952); MyGo creates it hidden and shows a
		// separate 1060x660 launcher window instead. Isolated/QA desktops, login
		// starts, --background and --settings keep the single-window panel path.
		launcherFirst := launcherFirstWindow(isolatedDesktopRun(), hidden, background, slices.Contains(os.Args, "--settings"))
		if os.Getenv("WHALEISLE_FORCE_MAIN") != "" {
			launcherFirst = false
		}
		th := theme.Resolve(cfg.Theme)
		bgColor := th.BackgroundColor()
		flatDarkWindow := flatDarkCanvasPreview(cfgStore, cfg, th)
		if flatDarkWindow {
			// The desktop must show through the glass, so the window itself stays
			// transparent (native dark + glassOpacity model).
			bgColor = "rgba(0,0,0,0)"
		}
		winWidth, winHeight, winMinWidth, winMinHeight := mainWindowSize()
		winOpts := mygo.WindowOptions{
			Title:          i18n.T(i18n.KeyAppName),
			Width:          winWidth,
			Height:         winHeight,
			MinWidth:       winMinWidth,
			MinHeight:      winMinHeight,
			StateKey:       "main",
			Hidden:         hidden || background || launcherFirst,
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: int(chrome.EffectiveTitleBarHeight()),
			// The native shell builds an application menu for accelerators but
			// keeps the OS menu bar hidden (chrome.js:48 autoHideMenuBar,
			// :69-70 setAutoHideMenuBar/setMenuBarVisibility). Without this the
			// mygo menu bar would push the 12 DIP title band down.
			AutoHideMenuBar: true,
			// Native controls plate: 48 tall with 12 top / 4 bottom padding and
			// 32x32 buttons, so the buttons occupy y=12..44
			// (harness-chrome-inject.js:12,89,96-97). Top is the plate's own
			// inset, not a centred 8: the caption band paints its titles at y=12
			// and the launcher window already draws the same plate at 12
			// (launcher_window.go:69, launcher.css:21).
			WindowsCaptionButtons:    &mygo.WindowsCaptionButtonOptions{Width: 32, Height: 32, Top: 12, Right: 8, CornerRadius: 8, GlyphSize: 12},
			WindowsFrameCornerRadius: 20,
			Transparent:              true,
			BackgroundColor:          bgColor,
			Content:                  ui.View(mainView),
		}
		// Native content paints the accepted appearance; DirectComposition keeps
		// the outer silhouette transparent without adding a caption content row.
		win := mygo.NewWindow(winOpts)
		frame.SetDisplayWindow(win)
		startuptrace.Mark("window.created")
		// Native window.js:185 gives the Windows window a 48 DIP icon.
		applyWindowIcon(win)
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
		// A main-window recreation (launcher -> desktop, or a recovery reload)
		// closes the companion through OnClosed below; re-arm it when the new
		// window is shown so an enabled pet comes back instead of staying
		// closed for the rest of the session.
		win.OnShow(func() { state.desktopPet.SetReady(!isolatedDesktopRun()); startuptrace.Mark("window.shown") })
		win.OnClosed(func() { state.desktopPet.Close() })
		win.OnBlur(state.cancelComposerVoiceOnBlur)
		win.OnHide(state.cancelComposerVoice)
		win.OnClosed(state.cancelComposerVoice)
		win.OnHide(state.retireNativeComposerCommands)
		win.OnClosed(state.retireNativeComposerCommands)
		win.OnHide(state.pauseVoiceDevices)
		win.OnMinimize(state.pauseVoiceDevices)
		win.OnClosed(state.pauseVoiceDevices)
		win.OnClosed(state.closeWorkspaceOpenInApp)
		win.OnClosed(state.cancelGitRefresh)
		win.OnHide(state.pauseSVGImages)
		win.OnMinimize(state.pauseSVGImages)
		win.OnClosed(state.pauseSVGImages)
		win.OnShow(state.resumeSVGImages)
		win.OnRestore(state.resumeSVGImages)
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
		win.OnHide(state.checkpointResumeOnVisibilityChange)
		win.OnMinimize(state.checkpointResumeOnVisibilityChange)
		win.OnHide(func() { state.layout.Workspace.PauseContent() })
		win.OnMinimize(func() { state.layout.Workspace.PauseContent() })
		flushTerminalResize := func() { state.mu.Lock(); state.flushTerminalResizeLocked(); state.mu.Unlock() }
		win.OnHide(flushTerminalResize)
		win.OnMinimize(flushTerminalResize)
		win.OnClosed(func() { state.disposeWorkspaceSessions() })
		win.OnClosed(state.closeResumeCheckpoint)
		win.OnHide(state.hideOfficeSettings)
		win.OnClosed(state.hideOfficeSettings)
		win.OnHide(state.suspendCreatorTask)
		win.OnHide(state.suspendPluginCreator)
		win.OnMinimize(state.suspendPluginCreator)
		win.OnClosed(state.suspendPluginCreator)
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
		state.launcherWindowAllowed = launcherFirst
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
		state.startNativeUpdateCenter(win)
		state.startNativeUIRecovery(win)
		win.OnClosed(state.cancelImportCatalogScan)
		win.OnClosed(state.stopNativeUpdateCenter)
		if background && !hidden {
			win.ShowInactive()
		}

		state.openResume(userData)
		state.mu.Lock()
		resumeErr := state.resumeError
		state.mu.Unlock()
		if resumeErr != "" {
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageWarning, Title: i18n.T(i18n.KeyCmdResumeStateNeedsAttention), Message: resumeErr + "\n\n" + i18n.T(i18n.KeyCmdTheOriginalFileIsPreservedBackItUpBe)})
		}

		if !isolatedDesktopRun() {
			state.setupDesktopTray()
			if err := setupSystemNotifications(); err != nil {
				log.Println("system notifications:", err)
			}
			consumeSystemToastActivation(os.Args[1:])
			state.startWhaleBridgeAlerts(win)
		}

		// Electron welcome-window.js: a profile without any credential sees the
		// welcome window first; launcher and boot continue after it completes or
		// is skipped. Everyone else enters the launcher.js flow right away.
		if state.welcomeRequired() && !hidden && !background {
			state.openWelcomeWindow(win, cfg)
		} else {
			state.beginLauncherBoot(cfg, win)
		}
	})

	if err := mygo.App.Run(); err != nil {
		// The event loop failing is a fatal main-process failure: keep the
		// diagnostic, then still report the original error.
		if path := recordShellCrash("main", err.Error(), ""); path != "" {
			log.Println(i18n.T(i18n.KeyCrashReportWritten), path)
		}
		log.Fatal(err)
	}
}

func startHarness(cfg config.Config) {
	startuptrace.Mark("harness.start_requested")
	state.mu.Lock()
	if state.quitting {
		state.mu.Unlock()
		return
	}
	cfgStore := state.cfgStore
	startupStore := state.startupRecovery.store
	if state.launcherImportBusy {
		state.launcherStartBusy = false
		state.launcherImport = i18n.T(i18n.KeyCmdImportStartBlocked)
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
	port := effectiveGoPort(cfg)
	// Re-read the accepted optional Project mount on every owned start/restart.
	projectDispatch, readProjectPatches := state.projectLaunchAccess(cfg)
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
		Install:           installSrv,
		ReadLaunchPatches: readProjectPatches,
		Launch: harness.LaunchOptions{
			Host: host,
			Port: port,
			// Pairing can be enabled after boot. Keep its published fixed SPA
			// port free without restarting an active Host when that happens.
			ReservedPorts:   []int{phoneremote.MobileWebPort},
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
				return fmt.Errorf("%s", i18n.T(i18n.KeyCmdStartupRecoveryUnavailable))
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

// beginLauncherBoot is the Electron launcher.js entry: show the launcher first;
// auto-start the desktop only when the workspace is set AND autoStartDesktop is
// enabled (default true). The welcome window calls it once after completion or
// skip, so the ordinary flow runs exactly once on either path.
func (s *mainState) beginLauncherBoot(cfg config.Config, win *mygo.Window) {
	s.mu.Lock()
	s.launcherOpen = true
	s.launcherStartPreparing = true
	s.mu.Unlock()
	// launcher.js opens the launcher before any start decision. UI thread only:
	// the startup callback and the welcome window's completion both run there.
	s.syncLauncherStage()
	go func() {
		defer s.finishLauncherPreparation(win)
		canStartAfterRecovery := s.recoverImportBeforeStart()
		if s.coldStartupAllowed(cfg, canStartAfterRecovery) {
			s.mu.Lock()
			if s.quitting {
				s.mu.Unlock()
				return
			}
			s.launcherTriedStart = true
			s.launcherStartBusy = true
			s.mu.Unlock()
			startHarness(cfg)
		}
	}()
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
	valid := s.capturePRViewOwner(root)
	if !valid() {
		return
	}
	s.runGitRemote("pr-view", root, func(ctx context.Context) (string, error) {
		if !valid() {
			return "", errors.New(resumeText("工作区已更改，请重新打开 PR。", "The workspace changed. Open the PR again."))
		}
		st := desktopgit.ViewPR(ctx, root)
		if !st.OK {
			return "", fmt.Errorf("%s", st.Err)
		}
		if !desktopgit.ValidPullRequestURLForHost(st.URL, st.Host, st.Repository) {
			return "", errors.New(resumeText("PR 地址与当前仓库不匹配。", "The PR URL does not match this repository."))
		}
		var openErr error = errors.New(resumeText("工作区已更改，请重新打开 PR。", "The workspace changed. Open the PR again."))
		mygo.RunOnMain(func() {
			if valid() {
				openErr = mygo.Shell.OpenExternal(st.URL)
			}
		})
		if openErr != nil {
			return "", openErr
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
		Message:       i18n.T(i18n.KeyCmdCommitStagedChangesOnly) + summary + "\n\nUnstaged files and editor drafts are excluded; no push. / 不含未暂存文件或编辑器草稿，不会推送。",
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
		s.cancelGitRefresh()

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
		// The next frame refreshes its currently selected root. A completed
		// operation from a previous workspace cannot replace that read owner.
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
	s.requestImportCatalog(win, sourceHome, "")
}

func (s *mainState) startCreatorPluginDraft(win *mygo.Window, sessions *session.Store, region string) {
	prompt := resumeText("创建一个 DSH 会话视图插件。", "Create a DSH conversation view plugin.")
	switch region {
	case "leftbar":
		prompt = resumeText("创建一个带左侧栏入口的 DSH 插件。", "Create a DSH plugin with a left sidebar entry.")
	case "rightbar":
		prompt = resumeText("创建一个带右侧栏入口的 DSH 插件。", "Create a DSH plugin with a right sidebar entry.")
	}
	sid := ""
	if sessions != nil {
		sid = sessions.SelectedSessionID()
	}
	s.mu.Lock()
	s.agentPresetPick = "cordis"
	draft := s.composerDraftLocked(sid)
	if strings.TrimSpace(draft.Text) == "" {
		draft.Text = prompt
	} else {
		draft.Text = strings.TrimRight(draft.Text, "\n") + "\n\n" + prompt
	}
	draft.Client.Sync()
	s.marketOpen = false
	s.scheduleOpen = false
	s.assistantConversationGeneration++
	s.assistantOpen = false
	s.settingsOpen = false
	s.launcherOpen = false
	s.mu.Unlock()
	if sessions != nil {
		sessions.SelectAgentPresetIDForSession(context.Background(), sid, "cordis")
	}
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
	// Recorded first so the deferred call runs after every layer, and the
	// opt-in sampler sees the whole build.
	viewStarted := time.Now()
	defer recordViewFrame(viewStarted)
	// Registered first so it runs last: the closing layer must sit above chat,
	// dialogs, settings and the launcher on every return path below. It takes
	// the palette this frame resolved below, so it cannot disagree with the
	// window it covers.
	var closingColors theme.Colors
	defer func() { state.paintClosingOverlayLayer(c, closingColors) }()
	// Registered before deferred native overlays: their rich seats must collect
	// DrawOver geometry before the final exact-Host placement is flushed.
	defer state.endPluginRichFrame(c)
	// The first-run 预览版说明 layer is intentionally not part of this build:
	// users asked for the blocking notice to be removed.
	consumePendingToast(c)
	sessionViews := state.beginSessionFrameViews()
	defer state.endSessionFrameViews(sessionViews)
	state.syncHostAppearance()
	openInApp := state.observeWorkspaceOpenInApp()
	defer openInApp.FinishFrame(c)
	defer state.observePluginCreator()
	defer state.paintPad(c)
	defer state.paintPluginShellOverlayLayer(c)
	defer state.finishVoiceDeviceSettingsFrame()
	defer state.syncResumeCheckpoint(c)
	standaloneScheduleRendered := false
	defer func() {
		if state.schedulePanelRendered && !standaloneScheduleRendered {
			state.schedulePanel.Deactivate()
		}
		state.schedulePanelRendered = standaloneScheduleRendered
	}()
	state.beginAssistantGroupRuntimeFrame()
	defer state.finishAssistantGroupRuntimeFrame()
	state.flushInactiveGeneralFontEdits(false)
	defer state.syncSettingsNativeSurfaces()
	composerRendered := false
	composerSessionID := ""
	defer func() { state.observeComposerEdit(composerRendered, composerSessionID) }()
	defer state.syncAccountBonusNotice(c)
	defer state.syncAccountStatusNotices(c)
	defer state.syncAccountQuotaNotice(c)
	defer state.syncResidentAccount(c)
	defer state.syncScheduleDeleteNotice(c)
	defer func() { state.syncConfirmAsk(c, closingColors) }()
	defer state.syncNativeUpdateModal(c, state.win)
	state.beginWallpaperFrame()
	defer state.endWallpaperFrame()
	state.registerApplicationCommands(c)
	defer state.processNonFramePageClose(c)
	defer state.processProductShortcutClose(c)
	state.syncApplicationMenu(c)
	defer state.processApplicationMenu(c)
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
	defer state.observeAgentPresetCodingTools()
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
	if !state.settingsLaunchHandled {
		state.settingsLaunchHandled = true
		if slices.Contains(os.Args, "--settings") {
			state.settingsOpen = true
			state.settingsDesktopActive = true
			state.seedDesktopDraft()
			// syncHostAppearance loads settings through EnsureLoaded outside state.mu.
			// Opening this panel must not launch the Host's configuration document.
		}
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
	launcherOpen := state.launcherOpen && state.launcherWin.destroyed()
	toolsMenuOpen := state.toolsMenuOpen
	cfgHideArchivedList := false
	cfgRemoteWorkspace := false
	wallpaperPath := ""
	imPluginDisabled := false
	if state.cfgStore != nil {
		cfg := state.cfgStore.Get()
		nodeBin = cfg.NodeBin
		cfgHideArchivedList = cfg.HideArchivedList
		cfgRemoteWorkspace = cfg.RemoteWorkspaceEnabled
		wallpaperPath, _ = theme.WindowWallpaperPath(cfg.WallpaperPath)
		for _, p := range cfg.DisabledPlugins {
			if p == "dsh-im" {
				imPluginDisabled = true
			}
		}
	}
	state.selectWorkspaceSessionLocked()
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
	// List layout commits after this view returns. Keep one UI-thread owner
	// across build passes so deferred navigation and cancellation share it.
	chatList := &state.chatList
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
	gitRepositoryStatus := state.gitRepositoryStatus
	gitMsg := state.gitMsg
	gitProgress := state.gitProgress
	gitBusy := state.gitBusy
	gitCommitOpen := state.gitCommitOpen
	gitCommitDraft := state.gitCommitDraft
	// This frame already holds state.mu (the snapshot above); the Locked
	// accessors assume that and never re-lock.
	if generated, ok := state.takeGeneratedCommitMessageLocked(); ok {
		gitCommitDraft = generated
	}
	gitCommitGenerating, gitCommitGenErr := state.gitCommitGenerationStateLocked()
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
	fileOpenWithState := state.fileOpenWithState
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
		if cwd := sessions.SelectionSummary().SelectedCwd; cwd != "" {
			workspaceRoot = cwd
		}
	}

	state.mu.Unlock()
	state.syncPadDesktop()
	state.syncPluginNativeNavigation(st.State == harness.Ready)
	if !settingsOpen {
		state.syncPluginPublicSettingsSection("", false)
	}
	if !settingsOpen && !usageOpen {
		settings.ClearPreferenceErrorNotices(c)
	}
	state.syncPluginContributions(st.State == harness.Ready && marketOpen && marketManager && !settingsOpen && !launcherOpen && !usageOpen && !imOpen && !scheduleOpen && !remoteOpen && !webviewOpen && !browserPreviewOpen && !assistantOpen)
	// Session-scoped contributions stay bound to the selected conversation; the
	// sync is throttled because reading the full session snapshot every frame is
	// expensive (see plugin_session_host.go).
	state.syncPluginSession(st.State == harness.Ready && !launcherOpen && !usageOpen && !imOpen && !marketOpen && !scheduleOpen && !remoteOpen && !webviewOpen && !browserPreviewOpen && !assistantOpen && !settingsOpen)
	state.beginPluginRichFrame(state.pluginRichFrameVisible())
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
		fileSessionID = sessions.SelectedSessionID()
	}
	state.dshMu.Lock()
	fileClient := state.dshClient
	fileMux := state.dshMux
	state.dshMu.Unlock()
	fileBindingID := fmt.Sprintf("%p\x00%s", fileClient, fileSessionID)
	var openWithOwner fileOpenWithOwner
	if sessions != nil && st.State == harness.Ready && selectedFile != "" {
		openWithOwner = fileOpenWithOwner{sessions: sessions, client: fileClient, endpoint: st.Endpoint, generation: st.Generation, sessionID: fileSessionID, root: workspaceRoot, path: selectedFile, epoch: sessions.ComposerSessionGeneration()}
		state.refreshFileOpenWith(win, openWithOwner, false)
		state.mu.Lock()
		fileOpenWithApps = append([]files.OpenWithApp(nil), state.fileOpenWithApps...)
		fileOpenWithState = state.fileOpenWithState
		state.mu.Unlock()
	} else {
		fileOpenWithApps = nil
		fileOpenWithState = files.OpenWithState{}
	}
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
	if gitPROpen && gitBranch != "" && workspaceRoot != "" {
		state.refreshPR(workspaceRoot, false)
	}
	state.mu.Lock()
	gitBranch = state.gitBranch
	gitDirty = state.gitDirty
	gitRepositoryStatus = state.gitRepositoryStatus
	gitMsg = state.gitMsg
	gitPROpen = state.gitPROpen
	gitPRReport = state.gitPRReport
	gitPRBusy = state.gitPRBusy
	state.mu.Unlock()
	th := theme.Resolve(themeID)
	systemDark := mygo.Theme.IsDark()
	svgImages := state.svgImageOwner(win, sessions)
	defer state.renderDesktopOnboarding(c, themeID, systemDark)
	colors := th.Active(systemDark)
	// The quit layer paints from the same palette as everything above it.
	closingColors = colors
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
		if gitRepositoryStatus != nil {
			gitBadge.Ahead, gitBadge.Behind = gitRepositoryStatus.Ahead, gitRepositoryStatus.Behind
		}
		gitBadge.PRSummary = gitPRSummary(gitPRReport)
		gitBadge.PRPanelLabel = i18n.T(i18n.KeyGitPRPanel)
		gitBadge.PRViewLabel = i18n.T(i18n.KeyGitPRView)
		gitBadge.PRCreateLabel = i18n.T(i18n.KeyGitPRCreate)
		if st.State == harness.Ready && workspaceRoot != "" && !gitBusy {
			root := workspaceRoot
			gitBadge.OnFetch = func() { state.runGitFetch(root) }
			gitBadge.OnPull = func() { state.runGitPull(root) }
			if gitRepositoryStatus != nil && gitRepositoryStatus.CanPush() {
				gitBadge.OnPush = func() { state.runGitPush(root) }
			}
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
			if gitRepositoryStatus == nil || !gitRepositoryStatus.CanCreatePR() {
				gitBadge.OnPRCreate = nil
			}
		}
	}
	titleColors := chrome.TitleBarColorsFrom(colors)
	// Electron desk caption is white/transparent on light; dark keeps theme bg
	// so the strip under traffic lights matches the desk wash.
	if !systemDark {
		titleColors.Background = chrome.Color("#ffffff")
	}
	// The canvas alone carries the native flat mask; the sidebar, cards and
	// caption keep their own tokens (round228/229).
	canvasFill := chrome.Color(colors.BG)
	darkCanvas := th.Scheme == theme.SchemeDark || (th.Scheme != theme.SchemeLight && systemDark)
	flatCanvas := state.wallpaperCanvasFlatFor(wallpaperPath)
	if darkCanvas && flatCanvas {
		// Native glass model: the window stays transparent and the desktop shows
		// through the panels; the canvas itself is not painted opaque.
		canvasFill = ui.Transparent
	}
	appendBackdropLog("view: scheme=%q systemDark=%v darkCanvas=%v path=%q flat=%v bg=%q fill=%v", th.Scheme, systemDark, darkCanvas, state.resolvedWallpaperPath(wallpaperPath), flatCanvas, colors.BG, canvasFill)
	ui.Column(c).Fill().Background(canvasFill).Draw(func(p *ui.Painter, r ui.Rect) {
		// Opaque window-layer fill: a Transparent DirectComposition window
		// ignores BackgroundColor, so the masked canvas must be painted here.
		if darkCanvas && flatCanvas {
			p.Fill(r, canvasFill, 0)
		}
	}).Draw(state.wallpaperPainter(wallpaperPath, !launcherOpen, c.Preferences().ReduceMotion)).DrawOver(chrome.WindowFramePainter(win, systemDark)).Children(func() {
		if state.paintPluginRootClient(c, st.State == harness.Ready, launcherOpen || settingsOpen, titleColors) {
			return
		}
		if launcherOpen {
			// One launcher model, two surfaces: the separate 1060x660 launcher
			// window and this in-window fallback panel (launcher_window.go).
			state.launcherPanel(c, win, launcherPanelHost{
				ThemeID:    themeID,
				Version:    buildVersion,
				Status:     st,
				Logs:       logs,
				Workspace:  workspaceRoot,
				Supervisor: sup,
				Sessions:   sessions != nil,
			})
			return
		}
		// Exclusive panels (settings/market/…) must not keep desktop title actions.
		desktopChromeLocked := usageOpen || imOpen || marketOpen ||
			scheduleOpen || remoteOpen || webviewOpen || browserPreviewOpen || assistantOpen
		sharedNativeCaption := runtime.GOOS == "windows" && st.State == harness.Ready &&
			!usageOpen && !imOpen && !remoteOpen && !webviewOpen && !browserPreviewOpen
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
				state.openGlobalWorkspaceSession(win, sessions)
			}
		}
		state.mainCaptionFileDrop(c, st.State == harness.Ready, frame.Model{Store: sessions, Sizes: &layout, WorkspaceRoot: workspaceRoot}, func() {
			if sharedNativeCaption {
				ui.Box(c).Height(0)
			} else if frame.SidebarRestoreCaption(c, &layout, titleColors, captionGit, captionNewSession) {
				// Settings and other exclusive routes bypass the navigation shell's
				// writeback. Persist caption restores through the same layout state.
				state.mu.Lock()
				state.layout.Sidebar, state.layout.SidebarExpanded = layout.Sidebar, layout.SidebarExpanded
				state.layout.Responsive = layout.Responsive
				state.mu.Unlock()
			} else if desktopChromeLocked {
				chrome.TitleBarWithActions(c, "", titleColors, gitBadge, nil)
			} else if runtime.GOOS == "windows" {
				// The workspace owns its shared auto-sized caption row. An extra
				// top spacer shifts conversation and dock controls away from it.
				ui.Box(c).Height(0)
			} else {
				ui.Row(c).Fill().Height(chrome.EffectiveTitleBarHeight()).Background(titleColors.Background).DragWindow()
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
							sid = sessions.SelectedSessionID()
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
							sid = sessions.SelectedSessionID()
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
		state.paintUpdateProgress(c, titleColors)
		if gitBranch != "" {
			root := workspaceRoot
			chrome.DrawGitCommitStrip(c, titleColors, gitCommitOpen, chrome.GitCommitStrip{
				Message:       &gitCommitDraft,
				Dirty:         gitDirty,
				Placeholder:   i18n.T(i18n.KeyGitCommitPlaceholder),
				CommitLabel:   i18n.T(i18n.KeyGitCommit),
				CancelLabel:   i18n.T(i18n.KeyGitCommitCancel),
				GenerateLabel: i18n.T(i18n.KeyCmdGenerate),
				Generating:    gitCommitGenerating,
				GenerateError: gitCommitGenErr,
				OnGenerate: func() {
					if st.State == harness.Ready && root != "" && !gitBusy && !gitCommitGenerating {
						state.runGitCommitMessageGeneration(win, root)
					}
				},
				OnCommit: func() {
					if st.State == harness.Ready && root != "" && !gitBusy {
						gitWorkbench.RequestCommit(root, gitCommitDraft)
						rightPaneTab = "diff"
						gitCommitOpen = false
						state.mu.Lock()
						state.filesPaneOpen, state.rightPaneTab = true, "diff"
						state.gitCommitOpen = false
						state.gitCommitGenErr = ""
						state.mu.Unlock()
					}
				},
				OnCancel: func() {
					state.mu.Lock()
					state.gitCommitOpen = false
					state.gitCommitGenErr = ""
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
		var (
			settingsView      settingsstore.ViewState
			settingsViewReady bool
		)
		currentSettingsView := func() settingsstore.ViewState {
			if !settingsViewReady && setStore != nil {
				settingsView = setStore.Snapshot()
				settingsViewReady = true
			}
			return settingsView
		}
		var paintMainContent func(settings.SectionID)
		paintSettingsOverlay := func() {
			if settingsOpen && setStore != nil {
				snap := currentSettingsView()
				state.mu.Lock()
				embeddedSection := state.settingsEmbeddedSection
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
				conversationActive := state.settingsConversationActive
				featuresActive := state.settingsFeaturesActive
				creatorTaskRendered = agentPresetsActive
				embeddedActive := embeddedSection == settings.SectionMarket ||
					embeddedSection == settings.SectionRemote ||
					embeddedSection == settings.SectionRemoteWorkspace ||
					embeddedSection == settings.SectionUsageStats
				if (desktopActive || appearanceActive || interfaceActive || aboutActive) && !embeddedActive && !state.desktopDraft.seeded {
					state.seedDesktopDraft()
				}
				desktopErr := state.desktopSaveErr
				desktopAction := state.desktopLastAction
				if credsActive && !embeddedActive && !state.credsDraft.seeded {
					state.seedCredentialsDraft()
				}
				credsErr := state.credsSaveErr
				credsAction := state.credsLastAction
				credsHarness := state.credsHarnessNote
				var wallpaperChoices []string
				if (desktopActive || appearanceActive) && !embeddedActive {
					if !state.wallpaperChoicesReady {
						exeDir, _ := os.Executable()
						state.wallpaperChoicesCache = theme.ListWallpapers(theme.WallpaperGalleryDirs(paths.UserDataDir(), filepath.Dir(exeDir))...)
						state.wallpaperChoicesReady = true
					}
					wallpaperChoices = state.wallpaperChoicesCache
				} else {
					state.wallpaperChoicesReady = false
				}
				var restartPolicy *config.RestartPreferences
				if (desktopActive || aboutActive) && !embeddedActive && state.cfgStore != nil {
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
					if (modelsActive || skillsActive) && !embeddedActive {
						sessSnap = sessionViews.snapshot(sessions)
					}
					if modelsActive && !embeddedActive {
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
					}
					if agentPresetsActive && !embeddedActive && state.agentPresetManager == nil {
						state.agentPresetManager = session.NewAgentPresetManager(sessions)
					}
				}
				state.mu.Unlock()
				var referencedCredentials *settings.ReferencedCredentialsAccess
				if !embeddedActive && (credsActive || modelsActive || snap.SelectedNS == "llm-deepseek" || snap.SelectedNS == "llm-pi-ai" || snap.SelectedNS == "web-search-deepseek") {
					referencedCredentials = state.referencedCredentialsAccess()
				}
				desktopSection := &settings.DesktopAccess{}
				if (desktopActive || aboutActive) && !embeddedActive {
					desktopSection = &settings.DesktopAccess{
						Version:               state.harnessVersionSnapshot(),
						RestartPolicy:         restartPolicy,
						Fields:                desktopFields,
						LastAction:            desktopAction,
						Err:                   desktopErr,
						OnSetCloseToTray:      state.setCloseToTrayPreference,
						OnSetAutoStartDesktop: state.setAutoStartDesktopPreference,
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
					}
				}
				accountSection := &settings.AccountAccess{}
				if accountActive && !embeddedActive {
					usageSnap := usagestore.ViewState{}
					if usageStore != nil {
						usageSnap = usageStore.Snapshot()
					}
					loginState := state.accountLogin.Snapshot()
					signOutState := state.accountSignOutAccess()
					if usageSnap.AuthorizeURL == "" && loginState.URL != "" {
						usageSnap.AuthorizeURL = loginState.URL
					}
					bonuses := make([]settings.AccountBonus, 0, len(usageSnap.Bonuses))
					for _, b := range usageSnap.Bonuses {
						label := strings.TrimSpace(b.Amount + " " + b.Currency)
						if label == "" {
							label = b.Campaign
						}
						if b.Title != nil {
							label = *b.Title
						}
						bonuses = append(bonuses, settings.AccountBonus{
							AccountID: b.AccountID,
							OrderID:   b.OrderID,
							Label:     label,
							Detail:    b.Message,
						})
					}
					accountSection = &settings.AccountAccess{
						OnContact:        state.contactAccountSupport,
						SignInBusy:       loginState.Busy || signOutState.Open || signOutState.Busy,
						SignOut:          signOutState,
						ClientReady:      usageSnap.ClientReady,
						Loading:          usageSnap.Loading,
						Err:              strings.TrimSpace(usageSnap.Err + " " + loginState.Error + " " + accountActionError),
						Status:           usageSnap.AccountStatus,
						Email:            usageSnap.Email,
						Name:             usageSnap.Name,
						Plan:             usageSnap.Plan,
						UsageURL:         usageSnap.UsageURL,
						TopUpURL:         usageSnap.TopUpURL,
						BalanceSummary:   usageSnap.BalanceSummary,
						Details:          usageSnap.Details,
						HasRunningTasks:  usageSnap.HasRunningTasks,
						AttemptPhase:     usageSnap.AttemptPhase,
						AttemptErrorCode: usageSnap.AttemptErrorCode,
						SignInAttemptID:  usageSnap.AttemptID,
						AuthorizeURL:     usageSnap.AuthorizeURL,
						BonusesLoading:   usageSnap.BonusesLoading,
						BonusesError:     usageSnap.BonusesError,
						BonusAckState:    usageSnap.BonusFeedback.State,
						BonusAckError:    usageSnap.BonusFeedback.Error,
						BonusAckAccount:  usageSnap.BonusFeedback.AccountID,
						BonusAckOrder:    usageSnap.BonusFeedback.OrderID,
						Bonuses:          bonuses,
						OnAckBonus: func(accountID, orderID string) {
							if usageStore == nil {
								state.refuseHostAction()
								return
							}
							sid := ""
							if sessions != nil {
								sid = sessions.SelectedSessionID()
							}
							usageStore.AckBonusNotified(context.Background(), accountID, orderID, sid)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnRefresh: func() {
							if usageStore == nil {
								state.refuseHostAction()
								return
							}
							if profile := usageStore.Snapshot().Details.Profile.Value; profile.AvatarURL != nil {
								settings.RetryAccountAvatar(*profile.AvatarURL, func() {
									if win != nil {
										win.Invalidate()
									}
								})
							}
							sid := ""
							if sessions != nil {
								sid = sessions.SelectedSessionID()
							}
							usageStore.Refresh(context.Background(), sid)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnOpenAuthorize: state.openAccountAuthorization,
						OnCopyAuthorize: state.copyAccountAuthorization,
						OnOpenUsage:     func(u string) { state.openAccountUsage(u, "settings") },
						OnOpenTopUp:     state.openAccountTopUp,
						OnSignIn:        state.startAccountSignIn,
						OnSignOut:       func() { state.runAccountSignOut(win, state.accountSignOut.Request) },
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
								state.refuseHostAction()
								return
							}
							state.cancelAccountSignIn(client, attemptID, usageStore, sessions)
						},
					}
				} else if usageStore != nil {
					accountSection.Status = usageStore.Snapshot().AccountStatus
				}
				credentialsSection := &settings.CredentialsAccess{}
				if credsActive && !embeddedActive {
					credentialsSection = &settings.CredentialsAccess{
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
					}
				}
				aboutSection := &settings.AboutAccess{}
				if aboutActive && !embeddedActive {
					aboutSection = &settings.AboutAccess{
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
								state.refuseHostAction()
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
					}
				}
				appearanceSection := &settings.AppearanceAccess{}
				if appearanceActive && !embeddedActive {
					appearanceSection = &settings.AppearanceAccess{
						OnSetTheme:       state.setAppearanceTheme,
						Background:       state.wallpaperSettings(true),
						Preferences:      state.themeFontEditor(true),
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
					}
				}
				interfaceSection := &settings.InterfaceAccess{}
				if interfaceActive && !embeddedActive {
					interfaceSection = &settings.InterfaceAccess{
						Shortcuts:             state.pageShortcutEditor(),
						Locale:                &state.desktopDraft.locale,
						CloseToTray:           &state.desktopDraft.closeToTray,
						OpenAtLogin:           &state.desktopDraft.openAtLogin,
						AutoStartDesktop:      &state.desktopDraft.autoStartDesktop,
						ShowArchivedList:      &state.desktopDraft.showArchivedList,
						ShowViewTabs:          &state.desktopDraft.showViewTabs,
						AutoAllowTools:        &state.desktopDraft.autoAllowTools,
						LastAction:            desktopAction,
						Err:                   desktopErr,
						OnSave:                state.saveInterfacePreferences,
						OnSetShowArchivedList: state.setShowArchivedListPreference,
						OnSetViewTabs:         state.setLocalViewTabsPreference,
						OnSetAutoAllowTools:   state.setAutoAllowToolsPreference,
					}
				}
				// 对话 and 内置功能 pages: the Host and the restart-owning setters are bound
				// in wireSettingsBodies while the page is open.
				conversationSection := &settings.ConversationAccess{}
				featuresSection := &settings.FeaturesAccess{}
				modelsSection := &settings.ModelsAccess{}
				if modelsActive && !embeddedActive {
					modelsSection = &settings.ModelsAccess{
						ProviderExtras:    state.renderPluginModelExtras,
						FooterExtras:      state.renderPluginModelFooter,
						ProviderDiscovery: state.providerDiscoveryAccess(),
						ProviderProfiles:  state.providerProfilesAccess(),
						Vision:            state.visionSettingsAccess(),
						ProviderEnv: func(namespace string) ([]string, bool) {
							state.mu.Lock()
							set := state.settings
							state.mu.Unlock()
							if set == nil {
								return nil, false
							}
							return set.ProviderEnv(namespace)
						},
						Labels:     sessSnap.ModelLabels,
						Selected:   &state.settingsModelLabel,
						SessionID:  sessSnap.SelectedID,
						Err:        sessSnap.ModelsErr,
						LastAction: sessSnap.LastAction,
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
					}
				}
				mcpSection := &settings.MCPAccess{}
				if mcpActive && !embeddedActive {
					mcpSection = state.settingsMCPAccess()
				}
				pluginsSection := &settings.PluginsAccess{}
				if pluginsActive && !embeddedActive {
					marketSnap := marketstore.ViewState{}
					if marketStore != nil {
						marketSnap = marketStore.Snapshot()
					}
					pluginRows := make([]settings.PluginRow, 0, len(marketSnap.Rows))
					for _, row := range marketSnap.Rows {
						pluginRows = append(pluginRows, settings.PluginRow{
							Name: row.Label, Status: row.Detail, Disabled: row.Enabled != nil && !*row.Enabled,
							CanToggle: row.CanToggle(), ReadOnlyReason: row.ReadOnlyReason, Error: row.Error,
						})
					}
					lastPluginAction := state.settingsPluginsLastAction
					if lastPluginAction == "" {
						lastPluginAction = marketSnap.ListMethod
					}
					pluginsSection = &settings.PluginsAccess{
						Inventory:        state.pluginInventoryAccess(),
						Tabs:             state.pluginSettingsTabs(),
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
						LastAction:       lastPluginAction,
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
						OnOpenView:    state.openInstalledPluginView,
						OnInspect:     state.inspectInstalledPlugin,
						OnOpenManager: state.openNativePluginManager,
						OnOpenMarket: func() {
							state.mu.Lock()
							state.settingsOpen = false
							state.settingsDesktopActive = false
							state.settingsAccountActive = false
							state.settingsCredentialsActive = false
							state.settingsAboutActive = false
							state.settingsAppearanceActive = false
							state.settingsInterfaceActive = false
							state.settingsConversationActive = false
							state.settingsFeaturesActive = false
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
					}
				}
				skillsSection := &settings.SkillsAccess{}
				if skillsActive && !embeddedActive {
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
					projectCwd := ""
					if sessions != nil {
						removal = sessions.SkillRemovalReview()
						if target, err := sessions.SkillInventoryTarget(session.SkillRootUser); err == nil {
							projectCwd = target.Cwd
						}
					}
					skillsSection = &settings.SkillsAccess{
						Create:       state.settingsSkillCreateAccess(),
						Hub:          state.settingsSkillHubAccess(),
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
						OnSetVisibleGroupInvocable: func(rows []settings.SkillRow, enabled bool) {
							state.setVisibleSkillGroupInvocation(sessions, sessSnap.SkillCatalog, rows, enabled)
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
						OnReloadEdit:       state.reloadSettingsSkillEdit,
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
						OnInspect:    state.inspectSettingsSkill,
						TargetRoot:   state.settingsSkillRoot,
						ProjectCwd:   projectCwd,
						OnSelectRoot: state.selectSettingsSkillHubRoot,
						HubQuery:     &state.settingsSkillHubQuery,
						HubHits:      append([]settings.SkillHubHit(nil), state.settingsSkillHubHits...),
						OnRefresh: func() {
							if sessions != nil {
								sessions.RefreshSkills(context.Background())
							}
							if win != nil {
								win.Update(func() {})
							}
						},
						OnSearchHub:  state.searchSettingsSkillHub,
						OnInstallHub: state.requestSettingsSkillHubInstall,
						OnInsert: func(name string) {
							name = strings.TrimSpace(name)
							if name == "" {
								return
							}
							state.mu.Lock()
							id := ""
							if sessions != nil {
								id = sessions.SelectedSessionID()
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
					}
				}
				agentPresetsSection := &settings.AgentPresetsAccess{}
				if agentPresetsActive && !embeddedActive {
					agentPresetsSection = &settings.AgentPresetsAccess{
						DisableDeveloperTools: !settingsstore.DeveloperToolsEnabled(snap),
						OnSaveDefault:         state.saveDefaultAgentPreset,
						Creator:               state.creatorTaskAccess(sessions),
						Manager:               state.agentPresetManager,
						Selected:              &state.settingsAgentPresetLabel,
						Reader:                state.agentPresetReaderAccess(sessions),
					}
				}
				settingsModel := settings.Model{
					PublicContributions:   state.pluginPublicContributions(),
					Pet:                   state.desktopPet,
					ReferencedCredentials: referencedCredentials,
					Store:                 setStore,
					ThemeID:               themeID,
					SystemDark:            mygo.Theme.IsDark(),
					DesktopActive:         desktopActive,
					Desktop:               desktopSection,
					AccountActive:         accountActive,
					Account:               accountSection,
					CredentialsActive:     credsActive,
					Credentials:           credentialsSection,
					AboutActive:           aboutActive,
					About:                 aboutSection,
					AppearanceActive:      appearanceActive,
					Appearance:            appearanceSection,
					InterfaceActive:       interfaceActive,
					Interface:             interfaceSection,
					ConversationActive:    conversationActive,
					Conversation:          conversationSection,
					FeaturesActive:        featuresActive,
					Features:              featuresSection,
					ModelsActive:          modelsActive,
					Models:                modelsSection,
					MCPActive:             mcpActive,
					MCP:                   mcpSection,
					PluginsActive:         pluginsActive,
					Plugins:               pluginsSection,
					SkillsActive:          skillsActive,
					Skills:                skillsSection,
					AgentPresetsActive:    agentPresetsActive,
					AgentPresets:          agentPresetsSection,
					OnSelectDesktop: func() {
						state.mu.Lock()
						state.settingsDesktopActive = true
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
					OnSelectAccount: state.openAccountSettings,
					OnSignIn:        state.startAccountSignIn,
					OnSelectCredentials: func() {
						state.mu.Lock()
						state.settingsCredentialsActive = true
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
					OnSelectConversation: func() {
						state.mu.Lock()
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsFeaturesActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.settingsConversationActive = true
						state.mu.Unlock()
						if win != nil {
							win.Update(func() {})
						}
					},
					OnSelectFeatures: func() {
						state.mu.Lock()
						state.settingsDesktopActive = false
						state.settingsAccountActive = false
						state.settingsCredentialsActive = false
						state.settingsAboutActive = false
						state.settingsAppearanceActive = false
						state.settingsInterfaceActive = false
						state.settingsConversationActive = false
						state.settingsModelsActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						state.settingsFeaturesActive = true
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
						state.settingsMCPActive = false
						state.settingsPluginsActive = false
						state.settingsSkillsActive = false
						state.settingsAgentPresetsActive = false
						if state.settingsModelLabel == "" && sessions != nil {
							state.settingsModelLabel = sessions.SelectedModel()
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
						state.settingsConversationActive = false
						state.settingsFeaturesActive = false
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
					OnOpenRemoteWorkspace: func() {
						state.mu.Lock()
						state.settingsEmbeddedSection = settings.SectionRemoteWorkspace
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
								sid = sessions.SelectedSessionID()
							}
							usageStore.Refresh(context.Background(), sid)
						}
						if win != nil {
							win.Update(func() {})
						}
					},
					FilterPtr:     &state.settingsNSFilter,
					RevealSecrets: &state.settingsRevealSecrets,
					OnRequestReplace: func(confirm func()) {
						state.askConfirm(confirmAskRequest{
							Owner:        win,
							Title:        i18n.T(i18n.KeySettingsReplaceConfirmTitle),
							Body:         i18n.T(i18n.KeySettingsReplaceConfirmMessage),
							CancelLabel:  i18n.T(i18n.KeySettingsReplaceCancel),
							ConfirmLabel: i18n.T(i18n.KeySettingsReplace),
							Run:          confirm,
						})
					},
				}
				state.wireSettingsBodies(&settingsModel)
				settingsModel = state.withPluginSettingsSections(state.settingsSectionModel(settingsModel, func(_ *ui.Context, section settings.SectionID) {
					paintMainContent(section)
				}))
				state.syncPluginPublicSettingsSection(string(settings.ActiveSectionID(settingsModel)), true)
				settings.ModalView(c, settingsModel)
				return
			}
		}
		paintMainContent = func(embeddedSection settings.SectionID) {
			if acceptPanelOK && embeddedSection == "" && !marketOpen && !scheduleOpen && !assistantOpen && !usageOpen && !imOpen && !remoteOpen && !webviewOpen && !browserPreviewOpen && state.selectedPluginMainView() != "" && state.renderPluginNativeView(c) {
				return
			}
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
					SVGImages:     svgImages,
					OnOpenManager: state.openNativePluginManager,
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
					HarnessMarketOverlay: true,
					PluginDisabled:       false,
					InstallReady:         state.marketInstallReadyFor(nodeBin),
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
						path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: win, Title: i18n.T(i18n.KeyCmdExportInstallationHistory), DefaultPath: "whaleisle-marketplace.log"})
						if err != nil {
							mygo.Dialog.Error(i18n.T(i18n.KeyCmdMarketExportErrorTitle), err.Error())
							return
						}
						if path == "" {
							return
						}
						if err = os.WriteFile(path, data, 0600); err != nil {
							mygo.Dialog.Error(i18n.T(i18n.KeyCmdMarketExportErrorTitle), err.Error())
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
				sessionSnap := session.ViewState{}
				if sessions != nil {
					sessionSnap = sessionViews.snapshot(sessions)
					sid = sessionSnap.SelectedID
					sessionLinks = scheduleSessionLinks(sessionSnap)
				}
				standaloneScheduleRendered = true
				state.schedulePanel.View(c, scheduleui.Model{Store: schedStore, SessionID: sid, ThemeID: themeID, SystemDark: mygo.Theme.IsDark(),
					SessionLinks:     sessionLinks,
					SessionLinkState: func(id string) string { return scheduleSessionLinkState(sessionSnap, id) },
					OnRefresh: func() {
						if schedStore == nil {
							state.refuseHostAction()
							return
						}
						scheduler := schedStore
						go scheduler.Refresh(context.Background(), sid)
					},
					OnDelete: func(id string) {
						if schedStore == nil {
							state.refuseHostAction()
							return
						}
						scheduler := schedStore
						go scheduler.Delete(context.Background(), sid, id)
					},
					OnHistory: func(id string) {
						if schedStore == nil {
							state.refuseHostAction()
							return
						}
						scheduler := schedStore
						go func() {
							text, err := scheduler.History(context.Background(), sid, id)
							state.mu.Lock()
							if err != nil {
								state.pendingToast = saveFailedToast(i18n.T(i18n.KeyCmdReadTaskHistory), err)
							} else {
								state.scheduleHistoryText = text
							}
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						}()
					},
					OnOpenSession: state.openScheduleSession,
					OnSaveTitle: func(id, title string) {
						if schedStore == nil {
							state.notifyUser(win, hostNotReadyToast())
							return
						}
						scheduler := schedStore
						go func() {
							if err := scheduler.UpdateTitle(context.Background(), sid, id, title); err != nil {
								state.notifyUser(win, saveFailedToast(i18n.T(i18n.KeyCmdSaveTitle), err))
								return
							}
							if win != nil {
								win.Update(func() {})
							}
						}()
					},
					OnSavePrompt: func(id, prompt string) {
						if schedStore == nil {
							state.notifyUser(win, hostNotReadyToast())
							return
						}
						scheduler := schedStore
						go func() {
							if err := scheduler.UpdatePrompt(context.Background(), sid, id, prompt); err != nil {
								state.notifyUser(win, saveFailedToast(i18n.T(i18n.KeyCmdSavePrompt), err))
								return
							}
							if win != nil {
								win.Update(func() {})
							}
						}()
					},
					OnNewTask: func() {
						state.startScheduleTask(win, sessions)
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
						if err := imStore.Persist(); err != nil {
							state.notifyUser(win, err.Error())
							return
						}
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
					OnEnablePlugin: func() {
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
							defer cancel()
							if _, err := callDesktopPlugin(ctx, "pluginManager/setBundleEnabled", map[string]any{"name": "dsh-im", "enabled": true}); err != nil {
								state.notifyUser(win, saveFailedToast(i18n.T(i18n.KeyCmdEnableDshIm), err))
								return
							}
							if state.cfgStore != nil {
								_ = state.cfgStore.Update(func(c *config.Config) {
									kept := c.DisabledPlugins[:0]
									for _, name := range c.DisabledPlugins {
										if name != "dsh-im" {
											kept = append(kept, name)
										}
									}
									c.DisabledPlugins = kept
								})
							}
							state.notifyUser(win, i18n.T(i18n.KeyCmdDshImEnabledRestartTheAppIfThePageS))
							if win != nil {
								win.Update(func() {})
							}
						}()
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
				var channelsContent func(*ui.Context)
				if imStore != nil && !imPluginDisabled {
					channelsContent = func(c *ui.Context) { paintMainContent("channels") }
				}
				state.renderRemoteView(c, embeddedSection, channelsContent)
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
					lastNative = sessionViews.snapshot(sessions).LastAction
				}
				history := asstSnap.History
				if len(history) == 0 && sessions != nil {
					sessSnap := sessionViews.snapshot(sessions)
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
							asst.SetLastAction(i18n.T(i18n.KeyCmdSelectCurrentRosterBot), i18n.T(i18n.KeyCmdComposerHostChanged))
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
							return fmt.Sprintf(i18n.T(i18n.KeyCmdActivityEntryCount), len(act.Entries)), nil
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
					OnOpenURL: func(u string) {
						if err := mygo.Shell.OpenExternal(u); err != nil {
							state.notifyUser(win, saveFailedToast(i18n.T(i18n.KeyCmdOpenTheDownloadPage), err))
						}
					},
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
					sid = sessions.SelectedSessionID()
				}
				var usagePreferences func(*ui.Context, ui.Color, ui.Color)
				if embeddedSection == settings.SectionUsageStats && setStore != nil {
					snapshot, owner, write := setStore.BindDescribedPreferences()
					host := &settings.BodySettingsAccess{Snapshot: snapshot, Owner: owner, OnChange: func(ns, key, value string) {
						write(context.Background(), ns, map[string]string{key: value})
					}}
					usagePreferences = func(c *ui.Context, fg, muted ui.Color) {
						settings.UsagePreferencesWithContributions(c, host, state.pluginPublicContributions(), fg, muted)
					}
				}
				usageui.View(c, usageui.Model{
					Preferences: usagePreferences,
					Embedded:    embeddedSection == settings.SectionUsageStats,
					ActionError: accountActionError,
					Store:       usageStore,
					OnExportStats: func(format string) {
						c.CloseToast("usage-export-error")
						data, err := usageStore.ExportStats(format)
						if err != nil {
							c.AddToast(ui.Toast{ID: "usage-export-error", Title: i18n.T(i18n.KeyCmdUsageExportErrorTitle), Description: err.Error(), Type: "error", Timeout: -1})
							return
						}
						ext := "csv"
						if format == "json" {
							ext = "json"
						}
						path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: win, Title: i18n.T(i18n.KeyCmdUsageExportStatistics), DefaultPath: "usage-" + format + "." + ext})
						if err != nil {
							c.AddToast(ui.Toast{ID: "usage-export-error", Title: i18n.T(i18n.KeyCmdUsageExportErrorTitle), Description: err.Error(), Type: "error", Timeout: -1})
							return
						}
						if path == "" {
							return
						}
						if err = os.WriteFile(path, data, 0600); err != nil {
							c.AddToast(ui.Toast{ID: "usage-export-error", Title: i18n.T(i18n.KeyCmdUsageExportErrorTitle), Description: err.Error(), Type: "error", Timeout: -1})
						}
					},
					OnOpenSession: func(id string) {
						if sessions == nil {
							state.refuseHostAction()
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
					if st.State == harness.Ready {
						if unavailable := connectionUnavailableMessage(state.dshBootstrapError, state.dshNetworkOffline, state.dshMuxLost); unavailable != "" {
							connBanner = unavailable
						}
					}
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
					snap := sessionViews.snapshot(sessions)
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
						// Indexing runs on a worker (see composerPathsLocked); this
						// only starts it. state.mu is held here.
						state.composerPathsLocked(strings.TrimSpace(workspaceRoot))
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
					fileNavigation := files.Model{
						BindingID: fileBindingID, SessionID: snap.SelectedID, WorkspaceRoot: snap.SelectedCwd,
						Tabs: &fileTabs, SelectedFile: &selectedFile,
						Preview: &filePreview, PreviewErr: &filePreviewErr,
						Editing: &fileEditing, EditDraft: &fileEditDraft,
						EditStatus: &fileEditStatus, StatNote: &state.fileStatNote,
						ReadTextPageContext: func(ctx context.Context, rel string, offset int64) (workspacefs.TextPage, error) {
							return workspacefs.ReadRemoteTextPage(ctx, fileClient, fileSessionID, rel, offset)
						},
						ReadRemoteContext: func(ctx context.Context, rel string) (string, error) {
							return workspacefs.ReadRemoteText(ctx, fileClient, rel, workspacefs.DefaultMaxReadBytes)
						},
						StatRemoteContext: func(ctx context.Context, rel string) string {
							return workspacePreviewStat(ctx, fileClient, fileSessionID, rel)
						},
						ReadTableContext: remoteWorkbookReader(sessions, fileClient, snap.SelectedID),
						ImageGeneration:  snap.StopGeneration,
						ReadImageContext: state.workspaceImageReader(sessions, fileClient, st, snap.SelectedID, snap.SelectedCwd),
					}
					// Copy the immutable scope value out of the read-only frame view
					// before callbacks can outlive this render pass.
					turnFileScope := snap.TurnFiles.Scope
					previewOwner := state.captureDeliverablePreviewOwner(sessions, turnFileScope, fileBindingID, snap.SelectedCwd, st, fileClient)
					openDelivered := func(path string, complete func(error)) {
						current := func() bool {
							if sessions == nil {
								return false
							}
							// checked via SelectedSessionID and TurnFilesScope
							if sessions.SelectedSessionID() != snap.SelectedID || complete != nil && sessions.TurnFilesScope() != turnFileScope {
								return false
							}
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							return fmt.Sprintf("%p\x00%s", client, snap.SelectedID) == fileBindingID
						}
						finish := func(err error) {
							if !current() {
								return
							}
							if complete != nil {
								complete(err)
								if win != nil {
									win.Invalidate()
								}
							} else if err != nil {
								sessions.SetLastAction(err.Error())
							}
						}
						if !current() {
							return
						}
						fileTabs.Plans.ShowFiles(snap.SelectedID)
						if mediapreview.Supports(path) {
							state.openMediaFile(sessions, fileClient, snap.SelectedID, snap.SelectedCwd, path, false)
							finish(nil)
							return
						}
						handled, err := layout.Workspace.OpenFileResource(fileNavigation, path)
						if err != nil {
							finish(err)
							return
						}
						if !handled {
							if st.State != harness.Ready {
								finish(fmt.Errorf("%s", i18n.T(i18n.KeyCmdWorkspaceOpenUnavailable)))
							} else if complete == nil {
								sessions.OpenWorkspacePath(context.Background(), path, "", "")
							} else {
								go func() {
									ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
									defer cancel()
									err := sessions.OpenHistoryReferenceOnHost(ctx, turnFileScope, path)
									mygo.RunOnMain(func() { finish(err) })
								}()
							}
							return
						}
						if workspacefs.ClassifyDoc(selectedFile) == workspacefs.DocHTML {
							state.openLocalHTML(sessions, snap.SelectedID, snap.SelectedCwd, selectedFile)
							finish(nil)
							return
						}
						if workspacefs.ClassifyDoc(selectedFile) == workspacefs.DocPDF {
							state.openLocalPDF(sessions, snap.SelectedID, snap.SelectedCwd, selectedFile)
							finish(nil)
							return
						}
						state.mu.Lock()
						state.filesPaneOpen, state.rightPaneTab = true, "files"
						state.mu.Unlock()
						filesPaneOpen, rightPaneTab = true, "files"
						if win != nil {
							win.Update(func() {})
						}
						finish(nil)
					}
					// The sidebar account seat mirrors the settings account row.
					accountUsage := usagestore.ViewState{}
					if state.usage != nil {
						accountUsage = state.usage.Snapshot()
					}
					avatarURL := ""
					if accountUsage.Details.Profile.Value.AvatarURL != nil {
						avatarURL = strings.TrimSpace(*accountUsage.Details.Profile.Value.AvatarURL)
					}
					state.mu.Lock()
					connReady := st.State == harness.Ready && state.dshMuxReady
					updateInstalling := state.nativeReleases.installing
					updatePhase := state.nativeReleases.phase
					updateDetail := strings.TrimSpace(state.nativeReleases.status + " " + state.nativeReleases.err)
					state.mu.Unlock()
					// One settings copy per build: the frame model only reads it, and each
					// Snapshot copies every namespace's documents.
					settingsView := currentSettingsView()
					frameModel := frame.Model{
						PublicContributions:             state.pluginPublicContributions(),
						ConversationViews:               state.pluginConversationViews(),
						OnSelectConversationView:        state.selectPluginConversationView,
						OnClearConversationView:         state.clearPluginConversationView,
						OnSelectBuiltinConversationView: state.selectBuiltinPluginConversationView,
						RenderConversationView:          state.renderPluginConversationView,
						PluginRightPaneTabs:             state.pluginRightPaneTabs(),
						PluginRightPaneKinds:            state.pluginRightPaneKinds(),
						OnOpenPluginRightPaneKind:       state.pluginRightPaneKindOpener(win, sessions, snap.SelectedID, layout.Workspace),
						OnSelectPluginRightPaneTab:      state.selectPluginRightPaneTab,
						OnClosePluginRightPaneTab:       state.closePluginRightPaneTab,
						RenderPluginRightPaneTab:        state.renderPluginRightPaneTab,

						SVGImages:             svgImages,
						PluginSlot:            state.renderPluginNativeSlot,
						InteractionRenderSlot: state.renderPluginInteractionSlot,
						RightOverlayReserve:   state.padReservedWidth(c),
						ChildBusyEnter:        setStore.BusyEnterFor(fileClient),
						ChildDraft:            state.childComposerDraft,
						OnQueryReferences:     state.boundReferenceQuery(sessions, snap),
						OnOpenWhaleSettings:   state.openPetSettings,
						OnDownloadSessionLog:  state.sessionLogExportAction(win, sessions, snap),
						SessionLogDownloading: state.sessionLogDownloading,
						OnOpenSessionFeedback: state.sessionFeedbackAction(win, sessions, snap),
						OrdinaryTeam: func() projects.OrdinaryTeamModel {
							m := state.ordinaryTeamModel(win)
							m.ThemeID, m.SystemDark = themeID, systemDark
							return m
						},
						CustomPanelContent: state.customPanelContent(snap.SelectedID, themeID),
						GroupRoom:          groupRoom, GroupMembers: groupMembers,
						OnReplyGroupThread: func(threadID string) {
							state.beginAssistantGroupReply(sessions, activeDraft, snap.SelectedID, threadID)
						},
						ExternalNavigation:   sharedNavigation,
						PageShortcutBindings: state.pageShortcutBindings(),
						SessionSearchFocus:   &state.commandSearchFocus,
						OnOpenWorkspaceFiles: func() { filesPaneOpen = true; rightPaneTab = "files" },
						Store:                sessions,
						FrameView:            sessionViews.view(sessions),
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
							missing := state.cfgStore == nil
							var err error
							if !missing {
								err = state.cfgStore.Update(func(c *config.Config) {
									c.AutoAllowTools = on
								})
							}
							state.mu.Unlock()
							if missing {
								state.refuseHostAction()
							} else if err != nil {
								state.notifyUser(win, saveFailedToast(i18n.T(i18n.KeyCmdAutoAllowTools), err))
							}
						},
						ConnectionBanner: connBanner,
						OnReconnect:      state.requestConnectionReconnect,
						SessionCost: costui.Model{Billing: func() usagestore.BillingView {
							if usageStore != nil {
								return usageStore.Billing()
							}
							return usagestore.BillingView{}
						}(), Preferences: settingsView, State: &state.sessionCostState, OnPrices: func() {
							if usageStore != nil {
								usageStore.RequestBillingEditor()
							}
							state.mu.Lock()
							state.usageOpen = true
							state.mu.Unlock()
						}},
						UsageJobsToday: usageJobsToday,
						UsageBalance:   usageBalance,
						OnOpenUsage: func() {
							if usageStore == nil {
								state.refuseHostAction()
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
								sid = sessions.SelectedSessionID()
							}
							usageStore.Refresh(context.Background(), sid)
							if win != nil {
								win.Update(func() {})
							}
						},
						TitlebarPreferences:     settingsstore.TitlebarPreferencesFrom(settingsView),
						DisableDeveloperTools:   !settingsstore.DeveloperToolsEnabled(settingsView),
						HideArchivedList:        cfgHideArchivedList,
						ChatPreferences:         settingsstore.ChatPreferencesFrom(settingsView),
						ConversationPreferences: settingsstore.ConversationPreferencesFrom(settingsView),
						ComposerResize:          &state.composerResize,
						OnComposerResizeEnd:     func(axis string, value float32) { setStore.SaveComposerSize(context.Background(), axis, value) },
						Sizes:                   &layout,
						ChatList:                chatList,
						ThemeID:                 themeID,
						SystemDark:              systemDark,
						Draft:                   &activeDraft.Text,
						ComposerState:           activeDraft,
						OnSubmitComposerGesture: func(accelerated bool) {
							state.submitComposer(win, sessions, activeDraft, snap.SelectedID, workspaceRoot, editMessageSeq, accelerated, snap.ComposerHostGeneration)
						},
						OnNativeCommandAction: func(owner composer.NativeCommandOwner, name string) bool {
							return state.nativeComposerCommandAction(win, sessions, activeDraft, snap.SelectedID, workspaceRoot, owner, name)
						},
						OnNativeCommandChanged: func() {
							if win != nil {
								win.Update(func() {})
							}
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
						HideChatViewTabs:   state.conversationViewTabsHidden(settingsView),
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
							state.settingsConversationActive = false
							state.settingsFeaturesActive = false
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
						OnCreateWorkspaceSession: func(generation uint64, id string) {
							state.openWorkspaceSession(win, sessions, generation, id)
						},
						OnSelectWorkspace: func(id string) {
							if sessions == nil {
								state.refuseHostAction()
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
								state.refuseHostAction()
								return
							}
							id := sessions.WorkspaceID()
							if id == "" {
								sessions.SetLastAction(i18n.T(i18n.KeyCmdWorkspaceRenameIDMissing))
								return
							}
							sessions.RenameWorkspace(context.Background(), id, title)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnDeleteWorkspace: func() {
							if sessions == nil {
								state.refuseHostAction()
								return
							}
							id := sessions.WorkspaceID()
							if id == "" {
								sessions.SetLastAction(i18n.T(i18n.KeyCmdWorkspaceDeleteIDMissing))
								return
							}
							sessions.DeleteWorkspace(context.Background(), id)
							if win != nil {
								win.Update(func() {})
							}
						},
						OnMoveSessionBefore: func(sessionID, beforeSessionID string) {
							if sessions == nil {
								state.refuseHostAction()
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
							filesPaneOpen, rightPaneTab = true, "agents"
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
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: i18n.T(i18n.KeyCmdAttachFilesImages), DefaultPath: workspaceRoot, Multiple: true})
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
								state.refuseHostAction()
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
								state.refuseHostAction()
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
						OnReadWorkspaceTableContext: fileNavigation.ReadTableContext,
						OnReadWorkspaceImageContext: fileNavigation.ReadImageContext,
						FileImageGeneration:         fileNavigation.ImageGeneration,
						OnOpenDocumentViewer: func(path string) (files.DocumentSurface, error) {
							return state.openDocumentViewer(fileClient, snap.SelectedID, path)
						},
						OnReadWorkspaceTextPageContext: fileNavigation.ReadTextPageContext,
						OnReadWorkspaceFileContext:     fileNavigation.ReadRemoteContext,
						OnStatWorkspaceFileContext:     fileNavigation.StatRemoteContext,
						OnReadWorkspaceFile: func(rel string) (string, error) {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							return workspacefs.ReadRemoteText(context.Background(), client, rel, workspacefs.DefaultMaxReadBytes)
						},
						FileStatNote: &state.fileStatNote,
						OnStatWorkspaceFile: func(rel string) string {
							return workspacePreviewStat(context.Background(), fileClient, fileSessionID, rel)
						},
						OnIndexWorkspace: func(root string) []string {
							state.mu.Lock()
							defer state.mu.Unlock()
							return state.composerPathsLocked(root)
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
						OnVoiceSettings:     state.openVoiceSettings,
						RemoteFilesContent: func(c *ui.Context, paneSessions *session.Store, owner *remoteui.FileWorkspaceState) bool {
							return remoteui.FileWorkspace(c, state.remoteFileWorkspaceModel(win, paneSessions, themeID, systemDark, st.State == harness.Ready, cfgRemoteWorkspace, owner))
						},
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
						OnCreatePluginDraft: func(region string) {
							agentPresetPick = "cordis"
							state.startCreatorPluginDraft(win, sessions, region)
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
						ConnectionReady:         connReady,
						DesktopUpdateInstalling: updateInstalling,
						DesktopUpdatePhase:      updatePhase,
						DesktopUpdateDetail:     updateDetail,
						OnOpenDesktopUpdate: func() {
							state.mu.Lock()
							state.launcherOpen = true
							state.launcherTab = launcher.TabVersions
							state.toolsMenuOpen = false
							state.settingsOpen = false
							state.mu.Unlock()
							if win != nil {
								win.Update(func() {})
							}
						},
						OnContactAccount: state.contactAccountSupport,
						Account: sidebar.Account{
							Name:                strings.TrimSpace(accountUsage.Name),
							Detail:              strings.TrimSpace(accountUsage.Email),
							AvatarURL:           avatarURL,
							SignedIn:            accountUsage.AccountStatus == "credential-stored" || strings.TrimSpace(accountUsage.Name) != "" || strings.TrimSpace(accountUsage.Email) != "",
							OnSignOut:           func() { state.runAccountSignOut(win, state.accountSignOut.Request) },
							OnOpen:              state.openAccountSettings,
							OnSignIn:            state.startAccountSignIn,
							OnContact:           state.contactAccountSupport,
							Notices:             state.sidebarAccountNotices(),
							LauncherActions:     state.pluginAccountLauncherActions,
							RenderLauncherPopup: state.renderPluginAccountLauncherPopup,
						},
						OnOpenSettings: func() { state.toggleApplicationSettings(c) },
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
							if sessions == nil || sessions.SelectedSessionID() != snap.SelectedID {
								return
							}
							state.dshMu.Lock()
							currentClient := state.dshClient
							state.dshMu.Unlock()
							if fmt.Sprintf("%p\x00%s", currentClient, snap.SelectedID) != fileBindingID {
								return
							}
							if err := layout.Workspace.OpenPlanResource(snap.SelectedID, fileBindingID, workspaceRoot, doc); err != "" {
								sessions.SetLastAction(err)
								return
							}
							state.mu.Lock()
							state.filesPaneOpen, state.rightPaneTab = true, "plan"
							state.mu.Unlock()
							filesPaneOpen, rightPaneTab = true, "plan"
							if win != nil {
								win.Update(func() {})
							}
						},
						OnOpenDeliverable: func(path string) { openDelivered(path, nil) },
						OnPreviewDeliverable: func(path string) error {
							if accepted, err := state.previewDeliverableOnHost(previewOwner, path); accepted {
								if err == nil {
									filesPaneOpen, rightPaneTab = true, "browser"
								}
								return err
							}
							if !state.localDeliverablePreviewScope(previewOwner) {
								return fmt.Errorf("%s", resumeText("无法预览：当前会话的本地工作区不可用", "Cannot preview: the local workspace for this session is unavailable"))
							}
							documentAccepted := false
							localPreview := fileNavigation
							localPreview.OpenDeliveryDocument = func(absPath string, kind files.DeliveryPreviewKind) error {
								if err := state.openLocalDeliverableDocument(previewOwner, absPath, kind); err != nil {
									return err
								}
								documentAccepted = true
								return nil
							}
							if err := files.PreviewDeliveryPath(localPreview, path); err != nil {
								return err
							}
							if !state.localDeliverablePreviewScope(previewOwner) {
								return fmt.Errorf("%s", resumeText("无法预览：会话已更改", "Cannot preview: the session has changed"))
							}
							if documentAccepted {
								// The frame publishes these locals after the callback; retain
								// the route selected by the document owner.
								filesPaneOpen, rightPaneTab = true, "browser"
								return nil
							}
							fileTabs.Plans.ShowFiles(snap.SelectedID)
							state.mu.Lock()
							state.filesPaneOpen, state.rightPaneTab = true, "files"
							state.mu.Unlock()
							filesPaneOpen, rightPaneTab = true, "files"
							if win != nil {
								win.Update(func() {})
							}
							return nil
						},
						OnOpenFilePicker: func() {
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: i18n.T(i18n.KeyCmdOpenFile)})
							if err != nil || len(paths) == 0 || strings.TrimSpace(paths[0]) == "" {
								return
							}
							openDelivered(paths[0], nil)
						},
						OnOpenHistoryReference: openDelivered,
						FileOpenWithApps:       fileOpenWithApps,
						FileOpenWithState:      &fileOpenWithState,
						OnRefreshFileOpenWith:  func() { state.refreshFileOpenWith(win, openWithOwner, true) },
						FileEditorApps:         editorOpenWithApps(),
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
							if err := mygo.Shell.OpenPath(absPath); err != nil {
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
							state.openWorkspacePathWithOwner(openWithOwner, absPath, appID)
						},
						OnOpenWithEditor: func(absPath, editorID string, line int) {
							err := openInEditor(absPath, editorID, line, func(path string) error {
								if sessions != nil && st.State == harness.Ready {
									sessions.OpenWorkspacePath(context.Background(), path, "", "")
									return nil
								}
								return mygo.Shell.OpenPath(path)
							})
							if err != nil {
								sessions.SetLastAction(err.Error())
							}
						},
						OnRenderOfficePDF: func(rel string) {
							state.openOfficePDF(sessions, snap.SelectedID, snap.SelectedCwd, rel)
						},
						OfficePreviewStatus: func(path string) string {
							return state.officePreviewMessage(sessions, snap.SelectedID, path)
						},
					}
					frameModel.OnNewHostTerminal = func() {
						state.mu.Lock()
						current := state.sessions == sessions && state.bootStatus.Generation == st.Generation && state.bootStatus.Endpoint == st.Endpoint && state.bootStatus.State == harness.Ready && !state.quitting
						state.mu.Unlock()
						state.dshMu.Lock()
						sameClient := state.dshClient == fileClient
						state.dshMu.Unlock()
						if !current || !sameClient || sessions.SelectedSessionID() != snap.SelectedID || sessions.ComposerSessionGeneration() != snap.ComposerSessionGeneration {
							return
						}
						layout.Workspace.NewHostTerminal()
						state.mu.Lock()
						state.filesPaneOpen, state.rightPaneTab = true, "host-terminal"
						state.mu.Unlock()
						filesPaneOpen, rightPaneTab = true, "host-terminal"
						if win != nil {
							win.Invalidate()
						}
					}
					frameModel.OnPreviewDeliverablePresentation = func(path, presentation string) error {
						if presentation != "" && presentation != "default" && presentation != "dock" && presentation != "mini" {
							return fmt.Errorf("unsupported resource presentation %q", presentation)
						}
						beforeOpen := filesPaneOpen
						if err := frameModel.OnPreviewDeliverable(path); err != nil {
							return err
						}
						if err := layout.Workspace.PresentOpenedResource(presentation); err != nil {
							return err
						}
						if presentation == "mini" {
							filesPaneOpen = beforeOpen
							state.mu.Lock()
							state.filesPaneOpen = beforeOpen
							state.mu.Unlock()
						}
						if win != nil {
							win.Invalidate()
						}
						return nil
					}
					state.bindPluginNativeNavigation(&frameModel)
					state.bindShellParity(c, &frameModel, win)
					state.bindWorkspaceTrailing(win, &frameModel, gitBadge, gitRepositoryStatus, gitBusy, workspaceRoot, gitCommitDraft)
					frameModel.WorkspaceOpenInApp = openInApp
					frameModel.OnOpenWorkspaceLocal = func() { openInApp.OpenCurrent(sessions) }
					state.bindProjectModel(win, &frameModel)
					state.bindProjectNativeConsumers(win, &frameModel.Projects)
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
					frameModel = state.withPluginSessionFrame(frameModel)
					state.gateAssistantNavigation(&frameModel)
					nextPromptRendered = true
					frame.View(c, frameModel)
					state.workspaceNavigationView(c, win, sessions, themeID, systemDark, func() { state.openNoDirectory(win, sessions, activeDraft, snap.SelectedID) })
					state.directoryPickerView(c, win, sessions, themeID, systemDark, func(mode, path string) {
						if state.completePluginDirectoryAction(mode, path) {
							return
						}
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
					state.renderPluginSessionSlot(c)
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
					previousChatPaneTab := state.chatPaneTab
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
					frame.TraceConversationTabWriteback(previousChatPaneTab, chatPaneTab)
					// Search synchronously notifies the attached session store. Its
					// refresh callback also takes state.mu, so publish after unlocking.
					if sessions != nil {
						sessions.UpdateSessionSearch(context.Background(), sessionSearch)
					}
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
		navAccountUsage := usagestore.ViewState{}
		if state.usage != nil {
			navAccountUsage = state.usage.Snapshot()
		}
		navAvatarURL := ""
		if navAccountUsage.Details.Profile.Value.AvatarURL != nil {
			navAvatarURL = strings.TrimSpace(*navAccountUsage.Details.Profile.Value.AvatarURL)
		}
		navigationModel := frame.Model{
			PublicContributions: state.pluginPublicContributions(),
			Store:               sessions, Sizes: &layout, ThemeID: themeID, SystemDark: systemDark,
			FrameView:     sessionViews.view(sessions),
			SessionSearch: &sessionSearch, RenameSessionID: &renameSessionID, RenameDraft: &renameDraft,
			WorkspaceRoot: workspaceRoot, WorkspacePage: sharedNativeCaption && desktopChromeLocked,
			GitWorkbench: &gitWorkbench,
			Account: sidebar.Account{
				Name:                strings.TrimSpace(navAccountUsage.Name),
				Detail:              strings.TrimSpace(navAccountUsage.Email),
				AvatarURL:           navAvatarURL,
				SignedIn:            navAccountUsage.AccountStatus == "credential-stored" || strings.TrimSpace(navAccountUsage.Name) != "" || strings.TrimSpace(navAccountUsage.Email) != "",
				OnSignOut:           func() { state.runAccountSignOut(win, state.accountSignOut.Request) },
				OnOpen:              state.openAccountSettings,
				OnSignIn:            state.startAccountSignIn,
				OnContact:           state.contactAccountSupport,
				Notices:             state.sidebarAccountNotices(),
				LauncherActions:     state.pluginAccountLauncherActions,
				RenderLauncherPopup: state.renderPluginAccountLauncherPopup,
			},
		}
		if navigationModel.WorkspacePage {
			state.bindGlobalWorkspace(win, &navigationModel, workspaceRoot, systemDark)
			if setStore != nil {
				navigationModel.TitlebarPreferences = settingsstore.TitlebarPreferencesFrom(currentSettingsView())
			}
			state.bindWorkspaceTrailing(win, &navigationModel, gitBadge, gitRepositoryStatus, gitBusy, workspaceRoot, gitCommitDraft)
		}
		state.withMainNavigation(c, sharedNavigation, navigationModel, func() {
			if settingsOpen && setStore != nil && state.settingsEmbeddedSection != "" {
				ui.Box(c).Fill().Background(chrome.BackdropCanvas(chrome.Color(colors.BG)))
				return
			}
			paintMainContent("")
		})
		if navigationModel.WorkspacePage {
			state.mu.Lock()
			state.gitWorkbench = gitWorkbench
			state.mu.Unlock()
		}
		paintSettingsOverlay()

	})
}

func trayIcon() []byte {
	return trayIconCached()
}
