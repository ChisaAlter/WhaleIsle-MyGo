// Command whaleisle is the Go desktop shell of Whale Isle MyGo.
//
// Assembles M1 shell chrome and M2 harness boot: single-instance lock,
// config / credentials, custom title bar, tray, close-to-tray, open-at-login,
// vendored harness prepare/launch via Supervisor, and a native boot page.
package main

import (
	"context"
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
	assistantctl "whaleisle/internal/app/assistant"
	launcherctl "whaleisle/internal/app/launcher"
	marketctl "whaleisle/internal/app/market"
	projectctl "whaleisle/internal/app/project"
	"whaleisle/internal/app/startup"
	"whaleisle/internal/app/updater"
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
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/phoneremote"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/desktop/providerdiscovery"
	"whaleisle/internal/desktop/providerprofiles"
	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/desktop/taskguard"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/directorypicker"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/harness"
	"whaleisle/internal/harness/installctrl"
	"whaleisle/internal/hoststate"
	"whaleisle/internal/paths"
	"whaleisle/internal/product"
	"whaleisle/internal/recovery"
	"whaleisle/internal/renderpolicy"
	"whaleisle/internal/shortcutprefs"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/startupsuggest"
	"whaleisle/internal/startuptrace"
	appstate "whaleisle/internal/state"
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
	mainRoute                   mainRoute
	mainRouteEpoch              uint64
	mainRouteApplied            uint64
	mainRouteDirty              atomic.Bool
	liveUIWindowActive          atomic.Bool
	closingOverlayActive        atomic.Bool
	confirmAskActive            atomic.Bool
	minimalOfferActive          atomic.Bool
	configWriterMu              sync.Mutex
	configWriter                *asyncConfigWriter
	settingsTabCacheOnce        sync.Once
	settingsTabCache            *settingsTabSWR
	writebackRevs               writebackRevisions
	lastFileTabsBinding         string
	lastFileTabsRoot            string
	pluginViewInventoryHost     uint64
	pluginViewInventoryRevision uint64
	pluginDraftCreator          *pluginDraftCreation
	remoteWorkspaceLease        atomic.Uint64
	childComposerBindings       map[string]string
	fileOpenWithOwner           fileOpenWithOwner
	fileOpenWithState           files.OpenWithState
	sessionLogDownloads         map[sessionLogExportKey]*sessionLogExportOperation
	voiceDevices                *composerVoiceDevices
	workspaceOpenInApp          *frame.WorkspaceOpenInAppState
	initCtrlMu                  sync.Mutex
	updaterCtl                  *updater.Center
	marketCtl                   *marketctl.Coordinator
	launcherCtl                 *launcherctl.Controller
	assistantCtl                *assistantctl.Service
	projectCtl                  *projectctl.Manager
	media                       *nativeMedia
	mediaPaused                 bool
	customPanels                customPanelState
	applicationMenu             *desktopApplicationMenu
	wallpaperControls           *settings.WallpaperEditor
	wallpaperVisible            bool
	commandSearchFocus          bool
	appearanceEditor            *theme.Editor
	appearanceVisible           bool
	shortcutEditor              *shortcuts.Editor
	shortcutPreferences         *shortcutprefs.Store
	shortcutRevision            string
	shortcutEditorSeq           uint64
	wallpaperChoicesCache       []string
	wallpaperChoicesReady       bool
	wallpaperChoicesLoading     bool
	wallpaperChoicesGen         uint64
	// shortcutBindingsCache holds the accepted bindings for this state; see
	// pageShortcutBindings.
	shortcutBindingsMu    sync.Mutex
	shortcutBindingsCache shortcutBindingsCache
	projectWindowHidden   bool
	nativeRecovery        atomic.Pointer[nativeUIRecoveryController]
	voice                 *composerVoiceOperation
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
	// relaunchPending makes the quit path restart the application instead of
	// exiting: a data-location change only takes effect in a new process.
	relaunchPending bool
	// dataLocationErr and dataLocationAction are the last outcome the
	// 数据位置 row shows; both are locale text, not codes.
	dataLocationErr       string
	dataLocationAction    string
	dataLocationImporting bool
	// pendingDataLocation is a recorded data directory that was not there at
	// startup. It is named to the user before anything is written into the
	// default directory the process fell back to.
	pendingDataLocation           string
	pendingDataLocationUnreadable bool
	// legacyImport is the planned copy of the original Whale Isle profile. It
	// is planned off the render thread because planning walks a profile that
	// holds thousands of sessions.
	legacyImport      importer.FirstRunPlan
	legacyImportOK    bool
	legacyImportKnown bool
	// previousImport is the planned copy from the location this profile
	// replaced, planned by the same background pass.
	previousImport      importer.FirstRunPlan
	previousImportOK    bool
	previousImportKnown bool
	// dataLocationOfferScan is the single-flight guard for that pass, so a
	// frame and a second window cannot plan the same sources twice.
	dataLocationOfferScan dataLocationOfferScan
	// releaseProfile holds the desktop profile lock when startup could not
	// claim it before the first window (a pending data-location warning).
	releaseProfile func()
	// profileClaimDeferred records that the profile lock still has to be
	// claimed inside WhenReady.
	profileClaimDeferred bool
	phonePopupOpen       bool
	phoneRuntime         *phoneremote.Runtime
	phoneDeviceEditor    remoteui.PhoneDeviceEditor
	phoneGatewayEditor   remoteui.GatewayEditor
	phoneGitSession      string
	phoneGitRoot         string
	phoneGitWorkspaces   phoneGitWorkspaceScope
	phoneOp              sync.Mutex
	phoneStopResume      context.CancelFunc
	phoneIntent          uint64
	closeToTray          bool
	desktopTray          *mygo.Tray
	trayHideNotice       trayHideNotice
	trayHideNotification *mygo.Notification
	welcomeWin           *welcomeWindow
	// whaleBridgeWin is the single embedded WhaleBridge management window
	// (port of reference/electron/src/main/whalebridge-window.js).
	whaleBridgeWin *whaleBridgeWindow
	credsMissing   bool
	welcomeDone    bool
	bridgeAlerts   *bridgeAlertUI
	cfgStore       *config.Store
	// minimalSuggestion observes startup responsiveness once per launch; its
	// one answer opens the startup offer, and minimalOfferOpen holds that offer
	// on screen until it is answered or dismissed.
	minimalSuggestion      *startupsuggest.Observer
	minimalOfferOpen       bool
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
	sup                     *harness.Supervisor
	bootStatus              harness.Status
	bootLogs                appstate.Published[[]string] // immutable; replace or appendBootLogLocked, never edit in place
	coldLoading             bool                         // first automatic launch, until the workbench can open
	// coldStartSkipped records that the automatic start was declined (a failed
	// recovery record, a skip-user-plugins recovery, a waiting import) while the
	// window still opens on the startup page, which carries the reason and the
	// retry and diagnostics actions instead of dropping into the workbench.
	coldStartSkipped bool
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
	acpPanel                                acpPanelState
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
	whaleCatalog                            whaleCatalogState
	desktopDraft                            desktopConfigDraft
	desktopSaveErr                          string
	desktopLastAction                       string
	credsDraft                              credentialsDraft
	credsSeedLoading                        bool
	credsSeedGen                            uint64
	credsSaveErr                            string
	credsLastAction                         string
	credsHarnessNote                        string
	analyticsEnabled                        *bool
	credsShowSecrets                        bool
	usage                                   *usagestore.Store
	accountReturn                           accountReturnState
	accountStatusNotices                    accountStatusNoticeState
	accountQuotaNotice                      accountQuotaNoticeState
	im                                      *imstore.Store
	schedule                                *schedstore.Store
	usageOpen                               bool
	imOpen                                  bool
	marketOpen                              bool
	marketManager                           bool
	scheduleOpen                            bool
	scheduleHistoryText                     string
	remoteOpen                              bool
	hasRemoteToken                          bool
	hasRemoteRelayToken                     bool
	// Phone-remote tokens copied from the credential store for the daemon
	// config. Secrets: never logged and never written to config.json.
	remoteToken                  string
	remoteRelayToken             string
	remoteDeviceCount            int
	remoteDeviceNames            []string
	phoneActionMsg               string
	sshProbeLine                 string
	webviewOpen                  bool
	activePluginViewLabel        string
	browserPreviewOpen           bool
	officePreview                officePreviewState
	pdfPreviewRequest            uint64
	mediaPreviewRequest          uint64
	htmlDocumentRequest          uint64
	browserController            *browserpreviewui.Controller
	browserPreviewNavigateErr    string
	webviewHost                  *desktopwebview.Host
	settingsOverlayHost          *desktopwebview.Host
	webviewHostErr               string
	lastWebViewURL               string
	lastWebViewOrigin            string
	pluginViewResumeAfterRestart bool
	pluginViewHostGeneration     int
	maintenanceOpen              bool
	filesPaneOpen                bool // Electron tools → files/diff right dock
	launcherImport               string
	launcherImportJournal        []string
	launcherImportBusy           bool
	importRecoveryPending        bool
	launcherImportCancel         context.CancelFunc
	lifecycleCtx                 context.Context
	lifecycleCancel              context.CancelFunc
	// phoneBootStarted gates the once-per-session phone gateway start when the
	// harness first becomes ready; phoneBootStartAction is nil in production
	// (the real start) and replaced by a counter in tests.
	phoneBootStarted     bool
	phoneBootStartAction func()
	// startupOrch owns readiness levels and the startup tier plan.
	startupOrch              startupOrchestration
	phoneDiagSampling        bool
	launcherImportCategories map[string]bool
	launcherImportCatalog    *importer.Catalog
	launcherImportScan       importCatalogScan
	launcherStartCancel      context.CancelFunc
	quitRequestPending       bool
	quitCleanupFinished      bool
	closingOverlay           bool
	workspaceMenuOpen        bool
	workspacePickerAnchor    ui.Handle
	workspaceRenameDraft     string
	pendingDeeplink          deeplink.Target
	settingsDrafts           []settingsDraftSlot
	workspaceSessions        workspaceSessionLayouts
	layout                   frame.LayoutSizes
	chatList                 ui.ListState
	draft                    string
	editMessageSeq           float64
	feedbackCommentMsgID     string
	feedbackCommentDraft     string
	modelPick                string
	permissionPick           string
	agentPresetPick          string
	thinkingExpanded         map[string]bool
	renameSessionID          string
	renameDraft              string
	sessionSearch            string

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
	composerSlashStore      *session.Store
	composerSlashRev        uint64
	composerSlashLocale     string
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
	dshCtx            context.Context
	dshRefresh        func()
	dshCoalescer      *storeRefreshCoalescer
	dshEndpoint       string
	dshClient         *dsh.Client
	dshMux            *dsh.Mux
	dshMuxReady       bool
	dshMuxLost        bool
	dshBootstrapError string
	dshNetworkOffline bool
	coldStoresDir     string
	coldStoresOnce    sync.Once
	coldStoresDone    chan struct{}
}

var state = mainState{fileEditViewMode: files.EditViewEdit}

// desktopControlOps exposes the native operations behind the authenticated
// install-control desktop routes. Each callback uses an existing Go store or
// dsh RPC; unsupported operations stay explicit 501 responses in installctrl.
func desktopControlOps() installctrl.DesktopOps {
	return installctrl.DesktopOps{
		Proactive: state.dispatchProactive,
		Confirm:   state.confirmPluginSideEffect,
		State: func(context.Context) (any, error) {
			plugins := state.marketCoordinator().InstalledPluginLabels()
			state.mu.Lock()
			cfgStore := state.cfgStore
			status := state.bootStatus
			state.mu.Unlock()
			var cfg config.Config
			if cfgStore != nil {
				cfg = cfgStore.Get()
			}
			public := desktopPublicConfig(cfg)
			return map[string]any{
				"ok": true, "version": buildVersion, "kernel": status.State.String(),
				"plugins": plugins, "disabledPlugins": append([]string(nil), cfg.DisabledPlugins...),
				"config": public,
			}, nil
		},
		ListMarketplace: func(ctx context.Context, opt installctrl.MarketplaceOptions) (any, error) {
			plugins := state.marketCoordinator().ListPluginsForInstallControl(ctx, opt.Refresh, opt.Query)
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

func (s *mainState) applyRemoteCredentials(creds credentials.Credentials) {
	s.hasRemoteToken = strings.TrimSpace(creds.RemoteToken) != ""
	s.hasRemoteRelayToken = strings.TrimSpace(creds.RemoteRelayToken) != ""
	s.remoteToken = strings.TrimSpace(creds.RemoteToken)
	s.remoteRelayToken = strings.TrimSpace(creds.RemoteRelayToken)
	s.remoteDeviceCount = len(creds.RemoteDevices)
	s.remoteDeviceNames = phoneremote.DeviceNames(creds.RemoteDevices)
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
		s.desktopSaveErr = errcopy.Sentence(err)
		s.desktopLastAction = ""
		return
	}
	if err := s.materializeWallpaperDraft(); err != nil {
		s.desktopSaveErr = errcopy.Sentence(err)
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
		s.desktopSaveErr = errcopy.Sentence(err)
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

// saveDesktopConfigAsync validates the form on the UI thread, applies the
// accepted in-memory UI state optimistically, and runs disk persistence and OS
// login-item registration off the UI thread without holding s.mu across I/O.
func (s *mainState) saveDesktopConfigAsync(win *mygo.Window) {
	s.mu.Lock()
	if s.cfgStore == nil {
		s.mu.Unlock()
		s.notifyUser(win, hostNotReadyToast())
		return
	}
	draft := s.desktopDraft
	hostAppearanceAccepted := s.hostAppearanceAccepted
	cfgStore := s.cfgStore
	sessions := s.sessions
	interactions := s.interactions
	port, err := strconv.Atoi(strings.TrimSpace(draft.port))
	if err != nil || port < 1024 || port > 65535 {
		s.desktopSaveErr = i18n.T(i18n.KeyCmdDesktopInvalidPort)
		s.desktopLastAction = ""
		s.mu.Unlock()
		return
	}
	restartPolicy, err := config.ParseRestartPreferences(draft.harnessAutoRestart, draft.harnessRestartMaxAttempts, draft.harnessRestartBaseDelayMs)
	if err != nil {
		s.desktopSaveErr = errcopy.Sentence(err)
		s.desktopLastAction = ""
		s.mu.Unlock()
		return
	}
	if err := s.materializeWallpaperDraft(); err != nil {
		s.desktopSaveErr = errcopy.Sentence(err)
		s.mu.Unlock()
		return
	}
	draft = s.desktopDraft
	workspaceChanged := strings.TrimSpace(draft.workspace) != strings.TrimSpace(draft.workspaceSeed)
	s.desktopSaveErr = ""
	s.desktopLastAction = i18n.T(i18n.KeySettingsDesktopSaved)
	s.closeToTray = draft.closeToTray
	if !hostAppearanceAccepted {
		s.themeID = draft.theme
	}
	s.mu.Unlock()

	i18n.SetLocale(draft.locale)
	if !hostAppearanceAccepted {
		applyAppearance(draft.theme)
	}
	if interactions != nil {
		interactions.SetAutoAllow(draft.autoAllowTools)
	}

	writer := s.configWriterFor(cfgStore)
	go func() {
		// Flush any prior queued toggles first so this explicit form save sees
		// a coherent store baseline and preserves CAS conflict detection.
		_ = writer.Flush()
		err := cfgStore.Update(func(c *config.Config) {
			restartPolicy.Apply(c)
			if workspaceChanged {
				c.Workspace = strings.TrimSpace(draft.workspace)
			}
			c.GoPort = port
			if !hostAppearanceAccepted {
				c.Theme = draft.theme
			}
			c.Locale = draft.locale
			c.CloseToTray = draft.closeToTray
			c.OpenAtLogin = draft.openAtLogin
			c.AutoStartDesktop = draft.autoStartDesktop
			if !hostAppearanceAccepted {
				c.WallpaperPath = strings.TrimSpace(draft.wallpaperPath)
			}
			c.HideArchivedList = !draft.showArchivedList
			c.HideChatViewTabs = !draft.showViewTabs
			c.AutoAllowTools = draft.autoAllowTools
		})
		if err != nil {
			s.mu.Lock()
			s.desktopSaveErr = errcopy.Sentence(err)
			s.desktopLastAction = ""
			s.mu.Unlock()
			if win != nil {
				win.Invalidate()
			}
			return
		}
		cfg := cfgStore.Get()
		if config.RestartPreferencesFrom(cfg) != restartPolicy {
			s.mu.Lock()
			s.desktopSaveErr = i18n.T(i18n.KeyCmdRestartPolicyNotSaved)
			s.desktopLastAction = ""
			s.mu.Unlock()
			if win != nil {
				win.Invalidate()
			}
			return
		}
		var loginErr error
		if err := desktopSystemIntegration(func() error { return mygo.App.SetOpenAtLogin(cfg.OpenAtLogin) }); err != nil {
			loginErr = err
			log.Println("openAtLogin:", err)
		}
		s.mu.Lock()
		s.desktopDraft.workspace, s.desktopDraft.workspaceSeed = cfg.Workspace, cfg.Workspace
		s.closeToTray = cfg.CloseToTray
		if !s.hostAppearanceAccepted {
			s.themeID = cfg.Theme
		}
		if loginErr != nil {
			s.desktopLastAction = ""
			s.desktopSaveErr = fmt.Sprintf("%s: %v", i18n.T(i18n.KeyCmdSettingsSavedButStartupRegistrationFai), loginErr)
		} else {
			s.desktopSaveErr = ""
			s.desktopLastAction = i18n.T(i18n.KeySettingsDesktopSaved)
		}
		s.mu.Unlock()
		if workspaceChanged && sessions != nil {
			sessions.SetWorkspaceCwd(cfg.Workspace)
		}
		if win != nil {
			win.Invalidate()
		}
	}()
}

// pickAndApplyWorkspace opens the Host registry picker. Launcher startup-directory
// selection remains independent and can still be used while the Host is offline.
func (s *mainState) pickAndApplyWorkspace(win *mygo.Window, sessions *session.Store, defaultPath string) {
	if sessions != nil {
		sessions.OpenWorkspacePicker()
	}
}

// applyWorkspacePath adopts a Host path transactionally. mkdirLocal is retained
// for call compatibility; registry adoption never creates desktop directories.
func (s *mainState) applyWorkspacePath(win *mygo.Window, sessions *session.Store, ws string, mkdirLocal bool) {
	if sessions == nil {
		return
	}
	// A first workspace is chosen before any Host exists: workspace/create is
	// unreachable, and the Host reads config.workspace when it launches. Saving
	// it here is what lets the choice start the Host instead of dead-ending.
	if !sessions.Published().Value.WorkspaceReady {
		s.adoptFirstWorkspace(win, sessions, ws)
		return
	}
	s.registerWorkspacePath(win, sessions, ws)
}

// adoptFirstWorkspace persists the workspace for a Host that has not started
// yet, then asks the supervisor to start it under that path.
func (s *mainState) adoptFirstWorkspace(win *mygo.Window, sessions *session.Store, path string) {
	real, err := directorypicker.UsableDirectory(path)
	if err != nil {
		log.Printf("workspace: refuse %q: %v", path, err)
		s.notifyUser(win, directoryPickerText("无法使用该文件夹，请换一个位置后重试。", "That folder cannot be used. Choose another one and try again."))
		return
	}
	s.mu.Lock()
	store := s.cfgStore
	s.mu.Unlock()
	if store == nil {
		log.Printf("workspace: no config store for %q", real)
		s.notifyUser(win, directoryPickerText("设置尚未就绪，请稍后重试。", "Settings are not ready yet. Try again in a moment."))
		return
	}
	if err := store.Update(func(c *config.Config) { c.Workspace = real }); err != nil {
		log.Printf("workspace: save %q: %v", real, err)
		s.notifyUser(win, directoryPickerText("无法保存工作区，请检查配置目录是否可写后重试。", "The workspace could not be saved. Check that the config folder is writable and try again."))
		return
	}
	log.Printf("workspace: adopted %q before Host start", real)
	sessions.SetWorkspaceCwd(real)
	s.startWorkbenchHost(win, nil)
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

// showMainWindow brings the window forward for an action the person at this
// machine took: a tray click, a menu item, a notification they clicked, a
// deeplink they opened, or a second launch they started.
func (s *mainState) showMainWindow() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win == nil {
		return
	}
	// The person asked for the window, so it may activate from here on. Before
	// this call the launch was unasked for and the window could only appear
	// (SetInactiveOnly at creation), which is what keeps a background start from
	// moving the foreground out from under the person using the machine.
	win.SetInactiveOnly(false)
	win.Show()
	win.Restore()
	win.Focus()
}

// revealMainWindow puts the window on screen without taking the foreground, for
// the surfaces no action at this machine asked for: the fallback when the tray
// could not be created during a launch the operator did not start, and a phone
// that navigated the desktop. ShowInactive also brings a minimized window back,
// so the app stays reachable from the taskbar while whatever the operator is
// working in keeps the foreground.
func (s *mainState) revealMainWindow() {
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win == nil {
		return
	}
	win.ShowInactive()
}

func (s *mainState) applyDeeplink(t deeplink.Target) {
	s.applyDeeplinkReveal(t, true)
}

// applyDeeplinkReveal is applyDeeplink with the choice of what the navigation may
// do to the operator's foreground: a destination the person at this machine
// picked comes forward, and one a phone picked only appears.
func (s *mainState) applyDeeplinkReveal(t deeplink.Target, activate bool) {
	if t.Action == deeplink.ActionLauncher {
		t.Action, t.Section = deeplink.ActionSettings, "desktop"
	}
	if t.Action != deeplink.ActionOpen {
		s.assistantService().SuspendViews()
	}
	s.mu.Lock()
	s.pendingDeeplink = t
	switch t.Action {
	case deeplink.ActionSettings:
		s.setMainRouteLocked(routeSettings)
		s.settingsEmbeddedSection = ""
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
		s.setMainRouteLocked(routeUsage)
	case deeplink.ActionMarket:
		s.setMainRouteLocked(routeMarket)
	case deeplink.ActionSchedule:
		s.setMainRouteLocked(routeSchedule)
	case deeplink.ActionAgents:
		s.setMainRouteLocked(routeConversation)
		s.filesPaneOpen = true
		s.rightPaneTab = "agents"
	case deeplink.ActionAuth:
		s.setMainRouteLocked(routeSettings)
		// OAuth return from account/startSignIn (callbackOrigin whaleisle://auth).
		s.settingsEmbeddedSection = ""
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
		s.setMainRouteLocked(routeConversation)
		if s.sessions != nil && t.SessionID != "" {
			s.revokePhoneGit()
			s.sessions.Select(t.SessionID)
		}
	case deeplink.ActionOpen:
		// Raise the current destination without changing its route.
	}
	win := s.win
	usage := s.usage
	sessions := s.sessions
	s.mu.Unlock()
	s.syncMainRouteResources()
	if activate {
		s.showMainWindow()
	} else {
		s.revealMainWindow()
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
	s.closeMainWebView()
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
		s.desktopSaveErr = errcopy.Sentence(err)
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

func (s *mainState) pluginRPCClient(ep harness.Endpoint) *pluginrpc.Client {
	return &pluginrpc.Client{BaseURL: ep.Origin, Cookie: ep.Cookie}
}

func (s *mainState) openWebViewHost(origin string) {
	s.openWebViewHostPlugin(origin, "")
}

func (s *mainState) openWebViewHostPlugin(origin, pluginID string) {
	s.openMainWebViewPlugin(origin, pluginID)
}

func (s *mainState) showBrowserPreview(rawURL string) {
	s.showMainBrowser(rawURL)
}

func (s *mainState) openWebViewURL(browse string) {
	s.openMainWebViewURL(browse)
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTerminalLocked(workspaceRoot, systemDark)
}

func (s *mainState) ensureTerminalLocked(workspaceRoot string, systemDark bool) {
	root := strings.TrimSpace(workspaceRoot)
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
	if override := s.launcherController().DownloadRouteOverride(); override != "" {
		return override
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if cmd == "l-ver" || cmd == "launcher-versions" {
		s.mu.Lock()
		win := s.win
		s.mu.Unlock()
		s.openNativeUpdateCenter(win)
		return true
	}
	if sid, ok := strings.CutPrefix(cmd, "session:"); ok && strings.TrimSpace(sid) != "" {
		s.launcherController().RetireStartupSelection()
	}
	s.mu.Lock()
	defer func() { s.mu.Unlock(); s.syncMainRouteResources() }()
	s.setMainRouteLocked(routeConversation)
	switch cmd {
	case "market":
		s.setMainRouteLocked(routeMarket)
	case "usage":
		s.setMainRouteLocked(routeUsage)
	case "assist", "assistant":
		s.setMainRouteLocked(routeAssistant)
	case "schedule", "sched":
		s.setMainRouteLocked(routeSchedule)
	case "im":
		s.setMainRouteLocked(routeIM)
	case "remote":
		s.setMainRouteLocked(routeRemote)
	case "browser":
		s.setMainRouteLocked(routeBrowser)
	case "webview":
		s.setMainRouteLocked(routeWebView)
	case "settings", "s-desk", "launcher", "l-home":
		s.settingsEmbeddedSection = ""
		s.setMainRouteLocked(routeSettings)
		s.settingsDesktopActive = true
		s.seedDesktopDraft()
	case "components", "l-comp", "launcher-components":
		s.setMainRouteLocked(routeComponents)
	case "import", "l-imp", "launcher-import":
		s.setMainRouteLocked(routeImport)
	case "startup-diagnostics", "l-plug", "launcher-plugins":
		s.setMainRouteLocked(routeStartupDiagnostics)
	case "update-preferences", "l-set", "launcher-settings":
		s.setMainRouteLocked(routeUpdatePreferences)
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
			s.workspaceSessionOpen.generation++
			ctx := s.lifecycleContext()
			go func() {
				var sessions *session.Store
				var snap session.ViewState
				poll := time.NewTimer(100 * time.Millisecond)
				defer poll.Stop()
				for i := 0; i < 100; i++ {
					if ctx.Err() != nil {
						return
					}
					s.mu.Lock()
					sessions = s.sessions
					s.mu.Unlock()
					if sessions != nil {
						snap = sessions.Published().Value
						if len(snap.Items) > 0 {
							break
						}
					}
					poll.Reset(100 * time.Millisecond)
					select {
					case <-ctx.Done():
						return
					case <-poll.C:
					}
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
	if isolatedDesktopRun() || os.Getenv("WHALEISLE_ACCEPT_OPEN_HOOK") != "1" {
		return
	}
	ctx := s.lifecycleContext()
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.pollAcceptOpenHook() {
					s.mu.Lock()
					w := s.win
					s.mu.Unlock()
					if w != nil {
						w.Update(func() {})
					}
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
	if bootProgressOnly(s.bootStatus, st) {
		// Log lines and plugin progress inside one Starting status (at most five
		// a second, coalesced by the supervisor) repaint the page from the
		// published log; the Host-state work below runs on state changes only.
		s.bootStatus = st
		if s.sup != nil {
			s.bootLogs.Publish(s.sup.Logs())
		}
		win := s.win
		s.mu.Unlock()
		if win != nil {
			ui.FrameTraceCause("boot-progress")
			win.Invalidate()
		}
		return
	}
	prev := s.bootStatus.State
	// A start that begins clears the declined mark: the page is then showing a
	// running start, not a skipped one.
	if st.State == harness.Starting {
		s.coldStartSkipped = false
	}
	// The actual supervisor generation/state check above rejects late snapshots.
	if s.bootStatus.Generation != st.Generation || s.bootStatus.State != st.State || s.bootStatus.Endpoint != st.Endpoint {
		s.remoteWorkspaceLease.Add(1)
	}
	s.bootStatus = st
	startuptrace.Mark("boot.status", st.State.String())
	if st.State == harness.Ready {
		startuptrace.Mark("boot.outcome_recorded")
		startuptrace.Finish("ready")
	}
	failedMessage := ""
	if st.State == harness.Failed {
		key := st.Generation
		if key == 0 {
			key = -1
		}
		if st.Err != nil && key != s.crashReportGeneration {
			s.crashReportGeneration = key
			failedMessage = errcopy.Sentence(st.Err)
		}
	} else {
		s.crashReportGeneration = 0
	}
	sup := s.sup
	if sup != nil {
		s.bootLogs.Publish(sup.Logs())
	}
	logs := s.bootLogSnapshot()
	win := s.win
	s.mu.Unlock()

	var startLogs []string
	if sup != nil {
		startLogs = sup.CurrentStartLogs()
	}
	l := s.launcherController()
	l.RecordStartupStatus(st, startLogs, buildVersion)
	l.ObserveBootStatus(prev, st)

	startPhone := false
	s.mu.Lock()
	if s.bootStatus.Generation == st.Generation && s.bootStatus.State == st.State {
		if st.State == harness.Ready {
			startPhone = s.claimPhoneBootStartLocked()
		}
	}
	s.mu.Unlock()
	// The native shell starts its phone gateway as soon as the config with
	// remoteEnabled is in effect; MyGo had no such call, so a boot-time enable
	// never started the daemon. Start it once per session, on the first Ready.
	if startPhone {
		// Tier C: the gateway starts on the remote page or after the idle
		// window, not inside the Host's first seconds.
		if s.phoneBootStartAction != nil {
			s.deferPhoneBootStart(s.phoneBootStartAction)
		} else {
			s.deferPhoneBootStart(func() { s.refreshPhoneRemote() })
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
	if win != nil {
		ui.FrameTraceCause("boot-status")
		win.Update(func() { s.refreshPetTask(); s.refreshDesktopTray(); s.refreshStartupUpdateAttention(win) })
	}
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
		s.endStartupGeneration()
		return
	}
	ep := st.Endpoint
	key := ep.Origin + "\x00" + ep.Cookie
	s.dshMu.Lock()
	if s.dshEndpoint == key && s.dshCancel != nil {
		s.dshMu.Unlock()
		return
	}
	oldLease := s.stopDSHLocked()
	ctx, cancel := context.WithCancel(context.Background())
	s.dshCancel = cancel
	s.dshCtx = ctx
	s.dshEndpoint = key
	// One transport per Host generation: a restarted Host never inherits the
	// previous process's pooled sockets, and stopDSH closes this pool.
	client := dsh.NewGenerationClient(dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie})
	startupGen := s.beginStartupGeneration()
	s.mu.Lock()
	s.dshClient = client
	s.dshMux = nil
	s.dshMuxReady = false
	s.dshMuxLost = false
	s.dshBootstrapError = ""
	s.dshNetworkOffline = false
	s.mu.Unlock()
	s.replaceMCPControllerLocked()
	s.settingsProviderProfiles.ResetHost()
	if s.settingsTabCache != nil {
		s.settingsTabCache.Reset()
	}
	s.remoteWorkspaceLease.Add(1)
	s.accountSignOut.Bind(client)
	s.settingsReferencedCredentials.Attach(client)
	s.settingsProviderDiscovery.Attach(client)
	s.mu.Lock()
	if s.sessions == nil {
		s.replaceSessionsLocked(s.newSessionStoreLocked())
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
	if s.schedule == nil {
		s.schedule = schedstore.New()
	}
	sessions := s.sessions
	inter := s.interactions
	setStore := s.settings
	usageStore := s.usage
	remoteMachines := s.remoteMachines
	imStore := s.im
	schedStore := s.schedule
	win := s.win
	s.mu.Unlock()
	s.dshMu.Unlock()
	closeAccountPlatformLease(oldLease)

	// A retained interaction store still owns the previous Host's refresh hook.
	// SetAutoAllow publishes synchronously; that hook may re-enter mainState.
	if interactionConfig != nil {
		inter.SetAutoAllow(interactionConfig.Get().AutoAllowTools)
	}
	var refreshQueued atomic.Bool
	var viewTabsHydrated atomic.Bool
	var inventoryReleased atomic.Bool
	var wasStreaming atomic.Bool
	flushRefresh := func() {
		s.noteNativeUIActivity()
		s.cancelComposerVoiceIfStale()
		s.tryHydrateViewTabsFromHarness(setStore, &viewTabsHydrated)
		if win == nil {
			uiDiagRefreshNil.Add(1)
			return
		}
		// While the boot page covers the window none of this is visible, and
		// the page repaints on its own 20 fps tick: run the main-thread work
		// without asking for a frame per store notification. The refresh
		// that lowers the page (recordStartupWorkspaceReady above) redraws.
		booting := s.bootPageCovers()
		if refreshQueued.CompareAndSwap(false, true) {
			work := func() {
				refreshQueued.Store(false)
				s.reconcileComposerSubmissions(sessions)
				s.refreshPetTask()
				s.maybeStartupBlankSession(win, sessions)
			}
			if booting {
				go mygo.RunOnMain(work)
			} else {
				ui.FrameTraceCause("session-refresh")
				win.Update(work)
			}
		} else if !booting {
			ui.FrameTraceCause("session-refresh")
			win.Invalidate()
		}
	}
	coalescer := newStoreRefreshCoalescer(storeRefreshCoalesceWindow, flushRefresh)
	refresh := func() {
		uiDiagRefreshHits.Add(1)
		s.observeWorkbenchReady(startupGen, sessions)
		s.recordStartupWorkspaceReady(sessions)
		v := sessions.Published().Value
		if !inventoryReleased.Load() {
			if (v.ClientReady && v.WorkspaceReady && !v.ListLoading) || v.WorkspaceError != nil || v.ListErr != nil {
				if s.marketCoordinator().WorkspaceReady(client) {
					inventoryReleased.Store(true)
				}
			}
		}
		streaming := v.PromptBusy || v.StopTurnSeq != 0
		prevStreaming := wasStreaming.Swap(streaming)
		terminalEdge := prevStreaming && !streaming
		hasError := v.HistoryError != nil || v.ListErr != nil || v.WorkspaceError != nil
		hasPendingInter := inter != nil && len(inter.Snapshot().Value.Pending) > 0
		coalescer.Notify(terminalEdge || hasError || hasPendingInter)
	}
	sessions.Attach(client, refresh)
	retireStaleInlineVisualPages()
	s.bindWorkspacePreferences(sessions, client)
	chat.SetAttachmentResolver(sessions.CaptureAttachmentRead)
	setStore.Attach(client, refresh)
	s.refreshReferencedCredentials()
	s.refreshProviderDiscovery()
	usageStore.Attach(client, refresh)
	remoteMachines.Attach(client, refresh)
	imStore.Attach(client, refresh)
	s.marketCoordinator().AttachHostForWorkspace(client, s.managedMarketProfile(st), refresh)
	schedStore.Attach(client, refresh)
	s.tryHydrateViewTabsFromHarness(setStore, &viewTabsHydrated)
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
	mux := &dsh.Mux{Endpoint: client.Endpoint, HTTP: client.HTTP}
	s.dshMu.Lock()
	if s.dshClient != client || ctx.Err() != nil {
		s.dshMu.Unlock()
		coalescer.Close()
		return
	}
	s.mu.Lock()
	s.dshMux = mux
	s.mu.Unlock()
	if s.dshCoalescer != nil {
		s.dshCoalescer.Close()
	}
	s.dshCoalescer = coalescer
	s.dshRefresh = refresh
	s.dshMu.Unlock()
	s.hostTerminals.Bind(client, mux, false, refresh)
	sessions.AttachMux(mux)
	s.bindSubmissionRecovery(ctx, sessions)
	if s.usage != nil {
		s.usage.AttachMux(mux)
	}
	if inter != nil {
		inter.AttachClient(client, mux)
		s.bindQuestionDraftRecovery(inter)
	}
	go s.runDSH(ctx, startupGen, client, mux, inter)
	// The proactive runner is Tier C: bound here only once its group ran;
	// before that the Tier C job binds the then-current client itself.
	if s.proactiveReady() {
		s.dshMu.Lock()
		if s.dshClient == client && ctx.Err() == nil {
			s.bindProactiveRunner(ctx, client)
		}
		s.dshMu.Unlock()
	}
	// Tier B: catalogs, presets, vision, billing and the product event wait
	// for this generation's WorkbenchReady and then run staggered.
	s.queueHostTierB(ctx, startupGen, client, sessions)
}

func (s *mainState) stopDSH() {
	s.dshMu.Lock()
	lease := s.stopDSHLocked()
	s.dshMu.Unlock()
	retireStaleInlineVisualPages()
	closeAccountPlatformLease(lease)
}

func (s *mainState) stopDSHLocked() *accountPlatformLease {
	s.assistantService().StopDSH()
	s.hostTerminals.Bind(nil, nil, false, nil)
	s.cancelComposerVoice()
	s.mu.Lock()
	s.cancelSessionLogExportsLocked()
	if s.acpPanel.cancel != nil {
		s.acpPanel.cancel()
	}
	s.acpPanel = acpPanelState{generation: s.acpPanel.generation + 1}
	s.invalidateSettingsSkillCreateLocked()
	s.invalidateSettingsSkillHubLocked()
	s.mu.Unlock()
	s.projectManager().Suspend()
	if s.remoteFiles != nil {
		s.remoteFiles.Bind(nil, "", nil)
	}
	if s.remoteMachines != nil {
		s.remoteMachines.Attach(nil, nil)
	}
	if s.dshCancel != nil {
		s.dshCancel()
		s.dshCancel = nil
	}
	if s.dshCoalescer != nil {
		s.dshCoalescer.Close()
		s.dshCoalescer = nil
	}
	s.dshCtx = nil
	s.dshRefresh = nil
	s.dshEndpoint = ""
	s.replaceMCPControllerLocked()
	s.settingsProviderProfiles.ResetHost()
	if s.settingsTabCache != nil {
		s.settingsTabCache.Reset()
	}
	s.remoteWorkspaceLease.Add(1)
	s.mu.Lock()
	// Only idle sockets close; requests still in flight finish on their own.
	s.dshClient.Close()
	s.dshClient = nil
	s.dshMux = nil
	s.dshMuxReady = false
	s.dshMuxLost = false
	s.dshBootstrapError = ""
	s.dshNetworkOffline = false
	s.mu.Unlock()
	lease := s.clearAccountReturnDetached()
	s.accountSignOut.Bind(nil)
	s.marketCoordinator().DetachHost()
	s.settingsReferencedCredentials.Attach(nil)
	s.settingsProviderDiscovery.Attach(nil)
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
	if s.schedule != nil {
		s.schedule.Detach()
	}
	return lease
}

func (s *mainState) refreshSettingsMCP() {
	s.refreshMCPController()
}

func (s *mainState) runDSH(ctx context.Context, startupGen uint64, client *dsh.Client, mux *dsh.Mux, inter *interaction.Store) {
	tracker, plan := s.startupPlan()
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
			ui.FrameTraceCause("connection")
			win.Update(s.refreshPetTask)
		}
	}
	connection := s.newConnectionAttemptState(ctx, client, mux, refresh)
	stopNetwork := connection.watchNetwork(ctx)
	defer stopNetwork()
	startuptrace.Mark("connection.bootstrap_start")
	_ = dsh.MaintainMuxFromBootstrap(ctx, mux, client, func(ctx context.Context) error {
		startuptrace.Mark("connection.carrier_connected")
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
						startuptrace.Mark("connection.events_ready")
						tracker.Reach(startupGen, startup.EventsReady)
						s.marketCoordinator().HostEventsReady(client)
						if schedStore != nil {
							schedStore.OnEventsReady(scheduleEventsOwner)
						}
						if sessions != nil {
							sessions.InitializeDefaultWorkspace(ctx)
							sessions.OnEventsReady(eventsOwner)
							// Tier B for this events generation: after the list baseline.
							plan.AfterWorkbench(ctx, startupGen, "speech.catalog", func() { sessions.WatchSpeechCatalogForEvents(eventsOwner) })
							plan.AfterWorkbench(ctx, startupGen, "permission.catalog", func() { sessions.RefreshPermissionCatalogForEvents(eventsOwner) })
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
		tracker.Lose(startupGen)
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
			log.Print("[boot] failed: ", st.Err)
		}
		return boot.StatusLine(st)
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
	startRaiseTrace()
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
	// A theme switch reaches the window over the interface's transition
	// duration instead of in one frame.
	theme.EnablePaletteReveal()
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
	// A record that names a missing directory, or that cannot be read at all,
	// falls back to the default. Nothing may be written there until the user
	// has seen that: the profile lock would create the directory, so claiming
	// it is postponed to the first window, which asks before EnsureUserData.
	// A run whose directory comes from WHALEISLE_USER_DATA or isolated mode
	// never reads the installed application's record, so it must not warn
	// about one either: that build has no way to fix it and its dialog quits.
	if issue, ok := paths.RecordedDataLocationIssue(); ok && state.dataLocationEditable() {
		state.mu.Lock()
		state.pendingDataLocation = issue.Path
		state.pendingDataLocationUnreadable = issue.Unreadable
		state.profileClaimDeferred = true
		state.mu.Unlock()
	} else {
		releaseProfile, claimErr := claimDesktopProfile()
		if claimErr != nil {
			// Dialog.Message cannot run before App.Run; the failure must stay a
			// logged early exit instead of panicking on the error path.
			log.Println("desktop profile:", claimErr)
			return
		}
		defer releaseProfile()
	}
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
		// The startup questions below run before config.json is read, so the
		// stored locale is not known yet: the system's own UI language speaks
		// for the app until the config is open, which then restores the
		// user's stored preference.
		i18n.SetLocale(i18n.SystemLocale())
		// Nothing below may create the data directory before the user has
		// answered about a recorded location that is missing.
		if !state.resolvePendingDataLocation() {
			return
		}
		if !state.claimDeferredProfile() {
			return
		}
		state.offerFirstRunImport()
		if err := paths.EnsureUserData(); err != nil {
			log.Println("userdata:", err)
		}
		userData := paths.UserDataDir()
		if err := startAppLog(userData); err != nil {
			log.Println("app log:", err)
		}
		state.mu.Lock()
		state.coldStoresDir = userData
		state.mu.Unlock()
		imgDir := imgcache.Dir(userData)
		if err := os.MkdirAll(imgDir, 0o755); err != nil {
			log.Println("imgcache:", err)
			media.Init(media.Config{CacheDir: filepath.Join(userData, "cache", "media")})
		} else {
			cache := imgcache.New(imgDir)
			media.Init(media.Config{CacheDir: filepath.Join(userData, "cache", "media"), Fetch: cache.Fetch})
			chat.SetRemoteImageCache(cache)
			attDir := filepath.Join(userData, "cache", "attachments")
			session.SetAttachmentCacheDir(attDir)
		}

		cfgStore, err := config.Open(userData)
		if err != nil {
			// The parse failure itself is for the log; the dialog names the
			// file and what to do about it.
			log.Println("config:", err)
			_, _ = mygo.Dialog.Message(mygo.MessageOptions{Type: mygo.MessageWarning, Title: i18n.T(i18n.KeyCmdConfigurationLoadFailed), Message: fmt.Sprintf(i18n.T(i18n.KeyConfigLoadFailedBody), filepath.Join(userData, config.FileName))})
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
		// A value left over from an older build names a Host this app was not
		// built against; correcting it here keeps a fallback launch honest
		// without asking the operator what that field means.
		if err := cfgStore.ReconcileHostVersion(launcherctl.ReadHarnessVersion()); err != nil {
			log.Println("host version:", err)
		}
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
		state.launcherController().SetCloseToTray(cfg.CloseToTray)

		if creds, err := credentials.Load(userData); err != nil {
			log.Println("credentials:", err)
		} else {
			state.mu.Lock()
			state.applyRemoteCredentials(creds)
			// Electron welcome-window gate: a profile without a stored API key
			// sees the welcome window before the workbench opens.
			state.credsMissing = strings.TrimSpace(creds.APIKey) == ""
			_, markerErr := os.Stat(filepath.Join(userData, welcomeMarker))
			state.welcomeDone = markerErr == nil
			state.mu.Unlock()
		}
		startuptrace.Mark("config.loaded")
		// The stored policy is in force before the first frame, so a profile
		// that runs minimal mode never paints the effects it pauses. The
		// startup observation starts here too: it decides nothing until the
		// window has been usable in the foreground for a while.
		applyMinimalMode(cfg.MinimalMode)
		applyGlassTheme(migrateGlassTheme(cfgStore))
		state.minimalSuggestion = startupsuggest.New()

		endOpenAtLogin := startuptrace.Begin("boot.open_at_login")
		if err := desktopSystemIntegration(func() error { return mygo.App.SetOpenAtLogin(cfg.OpenAtLogin) }); err != nil {
			log.Println("openAtLogin:", err)
		}
		endOpenAtLogin()

		hidden := isolatedDesktopHidden() || !isolatedDesktopRun() && mygo.App.WasOpenedAtLogin() && cfg.CloseToTray
		// --background is the only launch that shows the window with
		// ShowInactive; creating it visible calls the activating Show(). An
		// isolated QA run needs the same thing, and it is also the run that
		// skips the desktop system integration (URL scheme, autostart, tray),
		// so the flag must not be dropped there.
		background := slices.Contains(os.Args, "--background")
		endTheme := startuptrace.Begin("boot.theme")
		th := theme.Resolve(cfg.Theme)
		bgColor := th.BackgroundColor()
		endTheme()
		flatDarkWindow := flatDarkCanvasPreview(cfgStore, cfg, th)
		// The color the first frame starts on, for the window to hold while it
		// has no frame to show yet: the masked canvas the view paints under
		// everything else, or the window's own background color when that is
		// opaque. A wallpaper or gradient canvas has no single color to hold,
		// and its average is wrong for most images — a muddy mid-tone that
		// draws more attention than the system backdrop, which at least reads
		// as the system rather than as a defect. Those keep the system's.
		startupColor := bgColor
		if flatDarkWindow {
			// The desktop must show through the glass, so the window itself stays
			// transparent (native dark + glassOpacity model).
			startupColor = maskedCanvasHex(th.Active(mygo.Theme.IsDark()).BG)
			bgColor = "rgba(0,0,0,0)"
		}
		endSeed := startuptrace.Begin("boot.state_seed")
		state.mu.Lock()
		// A cold launch with automatic start owns its opening even before a
		// workspace exists: the page then says so and asks for the folder,
		// instead of dropping the person into a workbench whose service is not
		// running and whose every surface would blame the workspace.
		state.coldLoading = cfg.AutoStartDesktop
		state.layout = frame.DefaultLayout()
		if cfg.SidebarWidth >= 264 {
			state.layout.Sidebar = cfg.SidebarWidth
			if state.layout.Sidebar > 420 {
				state.layout.Sidebar = 420
			}
		}
		state.terminalVisible = false
		state.layout.Bottom = 0
		state.terminalSavedBottom = frame.DefaultLayout().Bottom
		if cfg.TerminalHeight >= 80 {
			state.terminalSavedBottom = cfg.TerminalHeight
		}
		state.chatList = ui.ListState{FollowEnd: true}
		state.fileExpanded = map[string]bool{}
		state.planBoardOpen = false
		state.todoPanelCollapsed = true
		state.trajActualDur = false // dsh.trajectory.duration default (sequence view)
		state.trajDetailTab = "overview"
		state.mu.Unlock()
		stateKey := "main"
		if isolatedDesktopRun() {
			stateKey = ""
		}
		endSeed()
		winWidth, winHeight, winMinWidth, winMinHeight := mainWindowSize()
		winOpts := mygo.WindowOptions{
			Title:     i18n.T(i18n.KeyAppName),
			Width:     winWidth,
			Height:    winHeight,
			MinWidth:  winMinWidth,
			MinHeight: winMinHeight,
			StateKey:  stateKey,
			// A launch nobody asked for must not be raisable even by a program that
			// calls SetForegroundWindow from outside, so its window is created
			// non-activatable. A click or an app-initiated reveal clears that.
			NonActivatable: windowNonActivatable(background, isolatedDesktopRun()),
			Hidden:         hidden || background,
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
			// and the onboarding window uses the same 12 DIP inset.
			WindowsCaptionButtons:    &mygo.WindowsCaptionButtonOptions{Width: 32, Height: 32, Top: 12, Right: 8, CornerRadius: 8, GlyphSize: 12},
			WindowsFrameCornerRadius: 20,
			Transparent:              true,
			BackgroundColor:          bgColor,
			WindowsBackdropColor:     startupColor,
			Content:                  ui.View(mainView),
		}
		// Native content paints the accepted appearance; DirectComposition keeps
		// the outer silhouette transparent without adding a caption content row.
		endWindow := startuptrace.Begin("boot.window_new")
		win := mygo.NewWindow(winOpts)
		endWindow()
		// An unasked-for launch may show this window, never take the foreground:
		// every raising call on it becomes ShowInactive until an action at this
		// machine asks for it (showMainWindow). The flag lives on the window, so a
		// raising path added later is covered without being audited.
		win.SetInactiveOnly(background || isolatedDesktopRun())
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
		// A main-window recreation (for example, a recovery reload)
		// closes the companion through OnClosed below; re-arm it when the new
		// window is shown so an enabled pet comes back instead of staying
		// closed for the rest of the session.
		win.OnShow(func() {
			state.desktopPet.SetReady(!isolatedDesktopRun())
			startuptrace.Mark("window.shown")
		})
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
		win.OnHide(state.pauseMedia)
		win.OnMinimize(state.pauseMedia)
		win.OnClosed(state.pauseMedia)
		media.Default().SetProcessNotify(win.Invalidate)
		win.OnShow(state.resumeMedia)
		win.OnRestore(state.resumeMedia)
		asstSvc := state.assistantService()
		win.OnHide(asstSvc.OnWindowHide)
		win.OnMinimize(asstSvc.OnWindowMinimize)
		win.OnClosed(asstSvc.OnWindowClosed)
		win.OnFocus(asstSvc.OnWindowFocus)
		win.OnBlur(state.accountWindowBlurred)
		win.OnFocus(state.returnFromAccountExternal)
		win.OnClosed(state.clearAccountReturn)
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
			state.mu.Unlock()
			state.projectManager().SetWindowHidden(true)
			state.hostTerminalUI.Observe(&state.hostTerminals, "", false)
		})
		win.OnShow(func() {
			state.mu.Lock()
			state.projectWindowHidden = false
			state.mu.Unlock()
			state.projectManager().SetWindowHidden(false)
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
			state.mu.Unlock()
			state.projectManager().Close()
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
		state.liveUIWindowActive.Store(win != nil)
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
		state.trajActualDur = false // dsh.trajectory.duration default (sequence view)
		state.trajDetailTab = "overview"
		state.mu.Unlock()
		state.watchAcceptOpenHook()
		// Update center, proactive monitor, cold stores and the bridge alerts
		// are Tier C (internal/app/startup/plan.go): first use of the
		// feature or 30 s after WorkbenchReady.
		state.registerDeferredStartup(win, userData, !isolatedDesktopRun())
		mygo.App.OnQuit(state.assistantService().CancelProactiveMonitor)
		mygo.App.OnQuit(func() {
			if err := state.flushPendingConfigWrites(); err != nil {
				log.Println("config flush on app quit:", err)
			}
		})
		// Tier OS: the hang watchdog must cover the start itself.
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
		}

		// First-run account onboarding is separate from Host startup.
		if state.welcomeRequired() && !hidden && !background {
			state.openWelcomeWindow(win, cfg)
		} else {
			state.beginWorkbenchBoot(cfg, win)
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

type deferredColdStoreOps struct {
	openIM           func(string) (*imstore.Store, error)
	openAssistant    func(string) (*assiststore.Store, error)
	openProactive    func(string)
	openCustomPanels func(string)
	openDshbotRoster func(string) (*dshbotstore.Store, error)
	loadSSHTunnel    func(string) (sshremote.TunnelConfig, error)
}

func (s *mainState) defaultDeferredColdStoreOps(ops *deferredColdStoreOps) deferredColdStoreOps {
	var resolved deferredColdStoreOps
	if ops != nil {
		resolved = *ops
	}
	if resolved.openIM == nil {
		resolved.openIM = imstore.Open
	}
	if resolved.openAssistant == nil {
		resolved.openAssistant = assiststore.Open
	}
	if resolved.openProactive == nil {
		resolved.openProactive = s.openProactive
	}
	if resolved.openCustomPanels == nil {
		resolved.openCustomPanels = s.openCustomPanels
	}
	if resolved.openDshbotRoster == nil {
		resolved.openDshbotRoster = dshbotstore.Open
	}
	if resolved.loadSSHTunnel == nil {
		resolved.loadSSHTunnel = sshremote.LoadSharedProfileTunnelConfig
	}
	return resolved
}

// triggerDeferredColdStores starts non-critical local store loading asynchronously
// once (primarily after window.shown, or earlier if Host connects in background).
func (s *mainState) triggerDeferredColdStores(userData string, ops *deferredColdStoreOps) <-chan struct{} {
	if s == nil || userData == "" {
		return nil
	}
	s.mu.Lock()
	if s.coldStoresDone == nil {
		s.coldStoresDone = make(chan struct{})
	}
	done := s.coldStoresDone
	s.mu.Unlock()
	s.coldStoresOnce.Do(func() {
		ctx := s.lifecycleContext()
		resolved := s.defaultDeferredColdStoreOps(ops)
		go func() {
			defer close(done)
			s.loadDeferredColdStores(ctx, userData, resolved)
		}()
	})
	return done
}

func (s *mainState) loadDeferredColdStores(ctx context.Context, userData string, ops deferredColdStoreOps) {
	if ctx.Err() != nil {
		return
	}
	var wg sync.WaitGroup
	wg.Add(6)
	go func() {
		defer wg.Done()
		s.commitDeferredIMStore(ctx, userData, ops.openIM)
	}()
	go func() {
		defer wg.Done()
		if ctx.Err() != nil {
			return
		}
		if asst, err := ops.openAssistant(userData); err != nil {
			log.Println("assistant:", err)
			if ctx.Err() == nil {
				s.assistantService().EnsureAssistantStore()
			}
		} else if ctx.Err() == nil {
			s.assistantService().AttachAssistantStore(asst)
		}
	}()
	go func() {
		defer wg.Done()
		s.commitDeferredProactive(ctx, userData, ops.openProactive)
	}()
	go func() {
		defer wg.Done()
		if ctx.Err() != nil {
			return
		}
		ops.openCustomPanels(userData)
	}()
	go func() {
		defer wg.Done()
		if ctx.Err() != nil {
			return
		}
		if roster, err := ops.openDshbotRoster(userData); err != nil {
			log.Println("dshbot roster:", err)
		} else if ctx.Err() == nil {
			s.assistantService().AttachDshbotRoster(roster)
		}
	}()
	go func() {
		defer wg.Done()
		s.commitDeferredSSHTunnel(ctx, userData, ops.loadSSHTunnel)
	}()
	wg.Wait()
	if ctx.Err() != nil {
		return
	}
	startuptrace.Mark("stores.opened")
	s.mu.Lock()
	win := s.win
	s.mu.Unlock()
	if win != nil {
		win.Update(func() {})
	}
}

func (s *mainState) commitDeferredIMStore(ctx context.Context, userData string, openFn func(string) (*imstore.Store, error)) {
	if ctx.Err() != nil {
		return
	}
	imS, err := openFn(userData)
	if ctx.Err() != nil {
		return
	}
	var attachIM, detachPrev *imstore.Store
	s.dshMu.Lock()
	client, refresh, dshCtx := s.dshClient, s.dshRefresh, s.dshCtx
	s.mu.Lock()
	if err != nil {
		log.Println("im:", err)
		if s.im == nil {
			s.im = imstore.New()
			attachIM = s.im
		}
	} else if imS != nil && (s.im == nil || s.im.Root() == "") {
		detachPrev = s.im
		s.im = imS
		// A draft typed while the durable store was still opening moves with
		// the swap; a saved draft on disk wins otherwise.
		if prev := detachPrev; prev != nil {
			if d := prev.Snapshot().Value.Draft; d != "" && imS.Snapshot().Value.Draft == "" {
				imS.SetDraft("", d)
			}
		}
		attachIM = imS
	}
	s.mu.Unlock()
	s.dshMu.Unlock()
	if detachPrev != nil {
		detachPrev.Detach()
	}
	if attachIM != nil && client != nil && (dshCtx == nil || dshCtx.Err() == nil) {
		attachIM.Attach(client, refresh)
		s.dshMu.Lock()
		stale := s.dshClient != client || s.dshCtx != dshCtx || (dshCtx != nil && dshCtx.Err() != nil)
		s.dshMu.Unlock()
		if stale {
			attachIM.Detach()
		}
	}
}

func (s *mainState) commitDeferredProactive(ctx context.Context, userData string, openFn func(string)) {
	if ctx.Err() != nil {
		return
	}
	openFn(userData)
	if ctx.Err() != nil {
		return
	}
	s.dshMu.Lock()
	client, dshCtx := s.dshClient, s.dshCtx
	s.dshMu.Unlock()
	if client != nil && dshCtx != nil && dshCtx.Err() == nil {
		s.bindProactiveRunner(dshCtx, client)
		s.dshMu.Lock()
		stale := s.dshClient != client || s.dshCtx != dshCtx || dshCtx.Err() != nil
		s.dshMu.Unlock()
		if stale {
			s.assistantService().StopProactiveRunner()
		}
	}
	s.assistantService().StartProactiveMonitor()
}

func (s *mainState) commitDeferredSSHTunnel(ctx context.Context, userData string, loadFn func(string) (sshremote.TunnelConfig, error)) {
	if ctx.Err() != nil {
		return
	}
	tcfg, err := loadFn(userData)
	if err != nil {
		log.Println("ssh-tunnel:", err)
		return
	}
	if ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	if s.sshTunnel == nil && s.sshHost == "" && s.sshUser == "" && s.sshPort == "" && s.sshIdentity == "" && s.sshForward == "" {
		s.sshTunnel = sshremote.NewTunnelSession()
		s.sshTunnel.SetConfig(tcfg)
		s.sshHost = tcfg.Host
		s.sshUser = tcfg.User
		if tcfg.Port > 0 {
			s.sshPort = strconv.Itoa(tcfg.Port)
		} else {
			s.sshPort = "22"
		}
		s.sshIdentity = tcfg.IdentityFile
		s.sshForward = tcfg.LocalForward
	}
	s.mu.Unlock()
}

func startHarness(cfg config.Config) {
	startuptrace.Mark("harness.start_requested")
	startupStore := state.launcherController().RecoveryStore()
	state.mu.Lock()
	if state.quitting {
		state.mu.Unlock()
		return
	}
	cfgStore := state.cfgStore
	if state.launcherImportBusy {
		state.launcherImport = i18n.T(i18n.KeyCmdImportStartBlocked)
		state.mu.Unlock()
		state.launcherController().SetStartBusy(false)
		return
	}
	prepareCtx, cancelPreparation := context.WithCancel(context.Background())
	state.launcherStartCancel = cancelPreparation
	state.mu.Unlock()
	state.launcherController().SetStartBusy(true)
	defer func() {
		cancelPreparation()
		state.mu.Lock()
		state.launcherStartCancel = nil
		state.mu.Unlock()
	}()
	if !state.recoverImportBeforeStart() {
		return
	}
	// Every owned start provisions its directory here, so a restart or a start
	// from the launcher cannot hand the Host an empty one either.
	cfg = state.ensureLaunchWorkspace(cfg)
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
		// The record names this launch's Host, so the next launch can end one
		// this shell never got to stop instead of starting a second service
		// beside it.
		RecordPath: paths.HostRecordFile(),
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
	// Publish on the window queue after preparation, so Close/import cannot
	// race a late background bridge result into starting a new runtime.
	launch := func() {
		state.mu.Lock()
		if state.quitting || state.launcherImportBusy {
			state.mu.Unlock()
			state.launcherController().SetStartBusy(false)
			return
		}
		state.sup = sup
		state.mu.Unlock()
		state.launcherController().ResetRecoveryRecording()
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

func (s *mainState) finishBootPreparation(win *mygo.Window) {
	s.launcherController().EndPreparationOnly()
	if win != nil {
		win.Update(func() {})
	}
}

// beginWorkbenchBoot prepares automatic Host startup without changing the user's route.
func (s *mainState) beginWorkbenchBoot(cfg config.Config, win *mygo.Window) {
	s.launcherController().BeginBootPreparation()
	go func() {
		defer s.finishBootPreparation(win)
		canStartAfterRecovery := s.recoverImportBeforeStart()
		cfg = s.ensureLaunchWorkspace(cfg)
		if s.coldStartupAllowed(cfg, canStartAfterRecovery) {
			s.mu.Lock()
			quitting := s.quitting
			s.mu.Unlock()
			if quitting {
				return
			}
			s.launcherController().MarkColdStartBusy()
			startHarness(cfg)
		} else {
			// The page stays. A declined automatic start is still the opening
			// state of a cold launch: it names the reason and offers the action
			// that resolves it, instead of dropping the user into a workbench
			// with a status strip. Only when the window shows the Host starting
			// does the page hand over (recordStartupWorkspaceReady).
			s.mu.Lock()
			s.coldStartSkipped = true
			s.mu.Unlock()
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

// gitStatusLine is the git status strip's one line. A failure names the cause
// and the next step; git's own wording goes to the log with the exit status.
func gitStatusLine(actionOK, actionFail string, out string, err error) string {
	if err != nil {
		log.Printf("git: %s: %v", actionFail, err)
		if line := firstLine(out); line != "" {
			log.Printf("git: %s output: %s", actionFail, line)
		}
		return errcopy.Under(actionFail, err)
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
		s.gitMsg = errcopy.Sentence(reviewErr)
		s.mu.Unlock()
		return
	}
	summary, reviewErr := desktopgit.StagedSummary(root)
	if reviewErr != nil {
		s.mu.Lock()
		s.gitMsg = errcopy.Sentence(reviewErr)
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

func (s *mainState) tryHydrateViewTabsFromHarness(setStore *settingsstore.Store, hydrated *atomic.Bool) {
	if setStore == nil || hydrated == nil || hydrated.Load() {
		return
	}
	on, ok := setStore.ViewTabsEnabled()
	if !ok || !hydrated.CompareAndSwap(false, true) {
		return
	}
	go func() {
		s.mu.Lock()
		s.desktopDraft.showViewTabs = on
		cfgStore := s.cfgStore
		win := s.win
		s.mu.Unlock()
		if cfgStore != nil {
			if err := migrateViewTabsSetting(cfgStore, !on); err != nil {
				log.Printf("desktop: migrate view tabs setting: %v", err)
				hydrated.Store(false)
				return
			}
		}
		if win != nil {
			win.Update(func() {})
		}
	}()
}

func (s *mainState) applyImportCatalog(win *mygo.Window, sourceHome string) {
	s.requestImportCatalog(win, sourceHome, "")
}

func mainView(c *ui.Context) {
	theme.RevealFrame(c)
	state.syncMainRouteResources()
	defer state.beginProjectPresentationFrame()()
	// Recorded first so the deferred call runs after every layer, and the
	// opt-in sampler sees the whole build.
	viewStarted := time.Now()
	state.noteNativeUIActivity()
	defer recordViewFrame(viewStarted)
	if state.paintColdLoading(c) {
		return
	}
	// Registered first so it runs last: the closing layer must sit above chat,
	// dialogs, settings and maintenance on every return path below. It takes
	// the palette this frame resolved below, so it cannot disagree with the
	// window it covers.
	var closingColors theme.Colors
	closingOverlay := false
	minimalOfferOpen := false
	confirmAskPending := false
	defer func() {
		if closingOverlay || state.closingOverlayActive.Load() {
			state.paintClosingOverlayLayer(c, closingColors)
		}
	}()
	// The startup offer paints over the workbench and under the closing layer.
	defer func() {
		if minimalOfferOpen || state.minimalOfferActive.Load() {
			state.syncMinimalModeOffer(c)
		}
	}()
	// Registered before deferred native overlays: their rich seats must collect
	// DrawOver geometry before the final exact-Host placement is flushed.
	// The first-run 预览版说明 layer is intentionally not part of this build:
	// users asked for the blocking notice to be removed.
	sessionViews := state.beginSessionFrameViews()
	defer state.endSessionFrameViews(sessionViews)
	state.syncHostAppearance()
	openInApp := state.observeWorkspaceOpenInApp()
	defer openInApp.FinishFrame(c)
	defer state.observePluginCreator()
	defer state.finishVoiceDeviceSettingsFrame()
	defer state.syncResumeCheckpoint(c)
	standaloneScheduleRendered := false
	defer func() {
		if state.schedulePanelRendered && !standaloneScheduleRendered {
			state.schedulePanel.Deactivate()
		}
		state.schedulePanelRendered = standaloneScheduleRendered
	}()
	asstSvc := state.assistantService()
	asstSvc.BeginGroupRuntimeFrame()
	defer asstSvc.FinishGroupRuntimeFrame()
	state.flushInactiveGeneralFontEdits(false)
	defer state.syncSettingsNativeSurfaces()
	composerRendered := false
	composerSessionID := ""
	defer func() { state.observeComposerEdit(composerRendered, composerSessionID) }()
	defer state.syncAccountStatusNotices()
	defer state.syncAccountQuotaNotice(c)
	defer state.syncResidentAccount(c)
	defer state.syncScheduleDeleteNotice()
	defer func() {
		if confirmAskPending || state.confirmAskActive.Load() {
			state.syncConfirmAsk(c, closingColors)
		}
	}()
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
	var assistantObs assistantctl.FrameObservation
	defer func() { asstSvc.ObserveFrame(assistantObs) }()
	creatorTaskRendered := false
	defer func() { state.observeCreatorTask(creatorTaskRendered) }()
	defer func() { state.observeAgentPresetReader(creatorTaskRendered) }()
	defer state.observeAgentPresetCodingTools()
	providerDiscoveryRendered := false
	defer func() { state.observeProviderDiscovery(providerDiscoveryRendered) }()
	mcpPanelRendered := false
	defer func() { state.observeMCPHealth(mcpPanelRendered) }()
	asstSvc.ObserveWhaleSettings()
	state.cancelComposerVoiceIfStale()
	state.restoreResumeSelection()
	state.refreshPetTask()
	marketStore := state.marketCoordinator().EnsureStore()
	marketSnap := state.marketCoordinator().ViewSnapshot().Value
	marketCanRestart := marketSnap.CanRestart
	assistantOpen := asstSvc.Open()
	state.mu.Lock()
	closingOverlay = state.closingOverlay
	minimalOfferOpen = state.minimalOfferOpen
	confirmAskPending = state.confirmAsk != nil
	themeID := state.themeID
	st := state.bootStatus
	logs := state.bootLogSnapshot()
	sup := state.sup
	sessions := state.sessions
	dshMuxReady := state.dshMuxReady
	dshMuxLost := state.dshMuxLost
	dshBootstrapError := state.dshBootstrapError
	dshNetworkOffline := state.dshNetworkOffline
	filesPaneOpen := state.filesPaneOpen
	agentPresetManager := state.agentPresetManager
	lastFileTabsBinding := state.lastFileTabsBinding
	lastFileTabsRoot := state.lastFileTabsRoot
	nextPromptRendered := false
	defer func() {
		if !nextPromptRendered && sessions != nil {
			sessions.SyncNextPrompt(state.lifecycleContext(), false)
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
			state.setMainRouteLocked(routeSettings)
			state.settingsDesktopActive = true
			state.seedDesktopDraft()
			// syncHostAppearance loads settings through EnsureLoaded outside state.mu.
			// Opening this panel must not launch the Host's configuration document.
		}
	}
	setStore := state.settings
	settingsOpen := state.settingsOpen
	settingsEmbeddedSectionFrame := state.settingsEmbeddedSection
	usageStore := state.usage
	imStore := state.im
	schedStore := state.schedule
	usageOpen := state.usageOpen
	accountActionError := state.accountReturn.err
	imOpen := state.imOpen
	marketOpen := state.marketOpen
	marketManager := state.marketManager
	scheduleOpen := state.scheduleOpen
	remoteOpen := state.remoteOpen
	nodeBin := ""
	webviewOpen := state.webviewOpen
	browserPreviewOpen := state.browserPreviewOpen
	maintenanceOpen := state.maintenanceOpen
	mainDestination := state.mainRoute
	cfgHideArchivedList := false
	cfgRemoteWorkspace := false
	wallpaperPath := ""
	imPluginDisabled := false
	if state.cfgStore != nil {
		cfg := state.effectiveConfigFor(state.cfgStore)
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
	rawGitCommitDraft := state.gitCommitDraft
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
	rawFileEditSplit := state.fileEditSplit
	fileEditSplit := rawFileEditSplit
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
	rawChatPaneTab := chatPaneTab
	if chatPaneTab == "" {
		chatPaneTab = "chat"
	}
	trajSearch := state.trajSearch
	trajSelectedID := state.trajSelectedID
	trajActualDur := state.trajActualDur
	trajTurnsFold := state.trajTurnsFold
	trajCallsFold := state.trajCallsFold
	rawTrajDetailTab := state.trajDetailTab
	trajDetailTab := rawTrajDetailTab
	if trajDetailTab == "" || trajDetailTab == "overview" || trajDetailTab == i18n.T(i18n.KeyTrajTabOverview) {
		trajDetailTab = "overview"
	}
	imageLightboxPath := state.imageLightboxPath
	wbStart := mainViewWritebackSnapshot{
		revs:                 state.writebackRevs,
		layout:               layout,
		draft:                draft,
		editMessageSeq:       editMessageSeq,
		feedbackCommentMsgID: feedbackCommentMsgID,
		feedbackCommentDraft: feedbackCommentDraft,
		modelPick:            modelPick,
		permissionPick:       permissionPick,
		agentPresetPick:      agentPresetPick,
		thinkingExpanded:     thinkingExpanded,
		renameSessionID:      renameSessionID,
		renameDraft:          renameDraft,
		sessionSearch:        sessionSearch,
		fileTabs:             fileTabs,
		gitWorkbench:         gitWorkbench,
		fileExpanded:         fileExpanded,
		selectedFile:         selectedFile,
		fileFilter:           fileFilter,
		filePreview:          filePreview,
		filePreviewErr:       filePreviewErr,
		filePreviewList:      filePreviewList,
		fileEditing:          fileEditing,
		fileEditDraft:        fileEditDraft,
		fileEditStatus:       fileEditStatus,
		fileEditViewMode:     fileEditViewMode,
		fileEditSplit:        rawFileEditSplit,
		rightPaneTab:         rightPaneTab,
		agentsPromptDraft:    agentsPromptDraft,
		goalDraft:            goalDraft,
		diffPath:             diffPath,
		diffText:             diffText,
		diffErr:              diffErr,
		planBoardOpen:        planBoardOpen,
		todoPanelCollapsed:   todoPanelCollapsed,
		chatPaneTab:          rawChatPaneTab,
		trajSearch:           trajSearch,
		trajSelectedID:       trajSelectedID,
		trajActualDur:        trajActualDur,
		trajTurnsFold:        trajTurnsFold,
		trajCallsFold:        trajCallsFold,
		trajDetailTab:        rawTrajDetailTab,
		imageLightboxPath:    imageLightboxPath,
		gitCommitOpen:        gitCommitOpen,
		gitCommitDraft:       rawGitCommitDraft,
		gitPROpen:            gitPROpen,
	}
	workspaceRoot := ""
	if state.cfgStore != nil {
		workspaceRoot = state.effectiveConfigFor(state.cfgStore).Workspace
	}
	// A selected session owns its actual directory, including Host scratch sessions.
	if sessions != nil {
		if cwd := sessions.SelectionSummary().SelectedCwd; cwd != "" {
			workspaceRoot = cwd
		}
	}

	state.mu.Unlock()
	if !assistantOpen || st.State != harness.Ready {
		asstSvc.SuspendViews()
	}
	if !settingsOpen && !usageOpen {
		settings.ClearPreferenceErrorNotices(c)
	}
	imOnboardingRendered := false
	defer func() {
		if !imOnboardingRendered && imStore != nil && imStore.Onboarding.Snapshot().Value.Open {
			imStore.Onboarding.Hide()
		}
	}()
	if !imOpen || maintenanceOpen || settingsOpen || marketOpen || scheduleOpen || remoteOpen || webviewOpen || browserPreviewOpen || assistantOpen || usageOpen {
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
	fileTabsRebound := fileBindingID != lastFileTabsBinding || workspaceRoot != lastFileTabsRoot
	fileTabs.Bind(fileBindingID, workspaceRoot)
	fileTabsPaused := sessions == nil || maintenanceOpen || usageOpen || imOpen || marketOpen || scheduleOpen || remoteOpen || webviewOpen || browserPreviewOpen || assistantOpen
	if fileTabsPaused {
		state.hostTerminalUI.Observe(&state.hostTerminals, "", false)
		fileTabs.CancelPlanRead()
		if sessions != nil {
			sessions.CloseAgentNavigation()
		}
		fileTabs.Pause()
	}
	if fileTabsRebound || fileTabsPaused {
		state.mu.Lock()
		state.fileTabs = fileTabs
		state.lastFileTabsBinding = fileBindingID
		state.lastFileTabsRoot = workspaceRoot
		state.markFileTabsMutatedLocked()
		wbStart.fileTabs = fileTabs
		wbStart.revs.fileTabs = state.writebackRevs.fileTabs
		state.mu.Unlock()
	}
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
	gitChanged := state.refreshGit(workspaceRoot)
	prRefreshed := false
	if gitPROpen && gitBranch != "" && workspaceRoot != "" {
		state.refreshPR(workspaceRoot, false)
		prRefreshed = true
	}
	if gitChanged || prRefreshed {
		state.mu.Lock()
		gitBranch = state.gitBranch
		gitDirty = state.gitDirty
		gitRepositoryStatus = state.gitRepositoryStatus
		gitMsg = state.gitMsg
		gitPROpen = state.gitPROpen
		gitPRReport = state.gitPRReport
		gitPRBusy = state.gitPRBusy
		state.mu.Unlock()
	}
	th := theme.Resolve(themeID)
	systemDark := mygo.Theme.IsDark()
	mediaOwner := state.mediaOwner(win, sessions)
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
		// through the panels; the canvas itself is not painted opaque. Minimal
		// mode paints no blur material to see through, so the masked canvas it
		// used to inherit from the desktop is painted here instead.
		if renderpolicy.NoBackdrop() {
			canvasFill = chrome.FlatBackdropFill(chrome.Color(colors.BG), true)
		} else {
			canvasFill = ui.Transparent
		}
	}
	appendBackdropLog("view: scheme=%q systemDark=%v darkCanvas=%v path=%q flat=%v bg=%q fill=%v", th.Scheme, systemDark, darkCanvas, state.resolvedWallpaperPath(wallpaperPath), flatCanvas, colors.BG, canvasFill)
	wallpaperDraw := state.wallpaperPainter(wallpaperPath, true, c.Preferences().ReduceMotion)
	ui.Column(c).Fill().Background(canvasFill).Draw(func(p *ui.Painter, r ui.Rect) {
		// Opaque window-layer fill: a Transparent DirectComposition window
		// ignores BackgroundColor, so the masked canvas must be painted here.
		if darkCanvas && flatCanvas && canvasFill.A > 0 {
			p.Fill(r, canvasFill, 0)
		}
		wallpaperDraw(p, r)
	}).DrawOver(chrome.WindowFramePainter(win, systemDark)).Children(func() {
		// Exclusive panels (settings/market/…) must not keep desktop title actions.
		desktopChromeLocked := maintenanceOpen || usageOpen || imOpen || marketOpen ||
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
		captionMode := mainTopCaptionModeForOS(runtime.GOOS, st.State == harness.Ready, mainDestination)
		state.mainCaptionFileDrop(c, st.State == harness.Ready, frame.Model{Store: sessions, Sizes: &layout, WorkspaceRoot: workspaceRoot}, func() {
			switch captionMode {
			case topCaptionNone:
				ui.Box(c).Height(0)
			case topCaptionWindowsStandaloneStrip:
				// Windows standalone routes (routeBrowser, routeIM, routeRemote,
				// routeUsage, offline maintenance) align their own headers with
				// c.TitleBar() at y=12..44; keep a clean 12 DIP borderless drag
				// strip so no hairline or gitBadge collides with "- □ ×".
				ui.Row(c).Fill().Height(chrome.EffectiveTitleBarHeight()).Background(ui.Transparent).DragWindow()
			default:
				if frame.SidebarRestoreCaption(c, &layout, titleColors, captionGit, captionNewSession) {
					// Settings and other exclusive routes bypass the navigation shell's
					// writeback. Persist caption restores through the same layout state.
					state.mu.Lock()
					state.layout.Sidebar, state.layout.SidebarExpanded = layout.Sidebar, layout.SidebarExpanded
					state.layout.Responsive = layout.Responsive
					state.mu.Unlock()
				} else if captionMode == topCaptionNonWindowsLocked {
					chrome.TitleBarWithActions(c, "", titleColors, gitBadge, nil)
				} else {
					ui.Row(c).Fill().Height(chrome.EffectiveTitleBarHeight()).Background(titleColors.Background).DragWindow()
				}
			}
		})
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
				},
			})
		}
		sharedNavigation := !usageOpen && !imOpen && !remoteOpen && !webviewOpen && !browserPreviewOpen
		if sessions == nil {
			state.mu.Lock()
			if state.sessions == nil {
				state.replaceSessionsLocked(state.newSessionStoreLocked())
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
				settingsView = setStore.Snapshot().Value
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
					state.ensureCredentialsDraftSeededLocked()
				}
				credsErr := state.credsSaveErr
				credsAction := state.credsLastAction
				credsHarness := state.credsHarnessNote
				wantWallpaperChoices := (desktopActive || appearanceActive) && !embeddedActive
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
						sessSnap = sessions.Published().Value
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
				wallpaperChoices := state.cachedWallpaperChoices(wantWallpaperChoices)
				desktopFields.WallpaperChoices = wallpaperChoices
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
						DataLocation:          state.dataLocationAccess(win),
						OnSetCloseToTray:      state.setCloseToTrayPreference,
						OnSetAutoStartDesktop: state.setAutoStartDesktopPreference,
						OnSave: func() {
							state.saveDesktopConfigAsync(win)
						},
						OnBrowseWorkspace: func() {
							state.mu.Lock()
							def := strings.TrimSpace(state.desktopDraft.workspace)
							state.mu.Unlock()
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
						OnCheckUpdates:          func() { state.showUpdateCheck(win) },
						OnOpenComponents:        func() { state.navigateMainRoute(routeComponents) },
						OnOpenImport:            func() { state.navigateMainRoute(routeImport) },
						OnOpenHostDiagnostics:   func() { state.navigateMainRoute(routeStartupDiagnostics) },
						OnOpenUpdatePreferences: func() { state.navigateMainRoute(routeUpdatePreferences) },
					}
				}
				accountSection := &settings.AccountAccess{}
				if accountActive && !embeddedActive {
					usageSnap := usagestore.ViewState{}
					if usageStore != nil {
						usageSnap = usageStore.Snapshot().Value
					}
					loginState := state.accountLogin.Snapshot()
					signOutState := state.accountSignOutAccess()
					if usageSnap.AuthorizeURL == "" && loginState.URL != "" {
						usageSnap.AuthorizeURL = loginState.URL
					}
					accountSection = &settings.AccountAccess{
						OnContact:        state.contactAccountSupport,
						SignInBusy:       loginState.Busy || signOutState.Open || signOutState.Busy,
						SignOut:          signOutState,
						ClientReady:      usageSnap.ClientReady,
						Loading:          usageSnap.Loading,
						Err:              strings.TrimSpace(usageSnap.Err.SentenceText() + " " + loginState.Error + " " + accountActionError),
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
						OnRefresh: func() {
							if usageStore == nil {
								state.refuseHostAction()
								return
							}
							if profile := usageStore.Snapshot().Value.Details.Profile.Value; profile.AvatarURL != nil {
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
							state.settingsTabMarkRefreshed("usage", settingsTabOwner{gen: st.Generation, ref: usageStore, scope: sid})
							usageStore.Refresh(context.Background(), sid)
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
					accountSection.Status = usageStore.Snapshot().Value.AccountStatus
				}
				credentialsSection := &settings.CredentialsAccess{}
				if credsActive && !embeddedActive {
					credentialsSection = &settings.CredentialsAccess{
						Ready:       state.credentialsLoaded,
						CanSave:     state.credentialsCanSave,
						OnReload:    state.retryCredentialsLoad,
						Fields:      credFields,
						ShowSecrets: &state.credsShowSecrets,
						LastAction:  credsAction,
						HarnessNote: credsHarness,
						Err:         credsErr,
						OnSave: func() {
							state.mu.Lock()
							state.saveCredentials()
							state.mu.Unlock()
						},
						OnDescribe: func() {
							state.dshMu.Lock()
							client := state.dshClient
							state.dshMu.Unlock()
							if client == nil {
								state.mu.Lock()
								state.credsHarnessNote = i18n.T(i18n.KeySettingsCredDescribeUnavailable)
								state.mu.Unlock()
								return
							}
							go func() {
								refs := []string{settingsstore.CredRefAPIKey, settingsstore.CredRefGithubToken}
								desc, err := settingsstore.DescribeCredentials(context.Background(), client, refs)
								state.mu.Lock()
								if err != nil {
									state.credsHarnessNote = errcopy.Sentence(err)
								} else {
									labels := map[string]string{
										settingsstore.CredRefAPIKey:      i18n.T(i18n.KeySettingsCredAPIKey),
										settingsstore.CredRefGithubToken: i18n.T(i18n.KeySettingsCredGithub),
									}
									parts := make([]string, 0, len(refs))
									for _, ref := range refs {
										flag := i18n.T(i18n.KeySettingsCredNotConfigured)
										if desc[ref].Configured {
											flag = i18n.T(i18n.KeySettingsCredConfigured)
										}
										label := labels[ref]
										if label == "" {
											label = ref
										}
										parts = append(parts, fmt.Sprintf(i18n.T(i18n.KeySettingsCredDescribeItem), label, flag))
									}
									state.credsHarnessNote = strings.Join(parts, " · ")
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
						Identifier:            product.AppID,
						UserData:              paths.UserDataDir(),
						VendorRoot:            paths.VendorDir(),
						AnalyticsEnabled:      state.analyticsEnabled,
						OnRevealUserData: func() error {
							return desktopcomponents.OpenInFileManager(paths.UserDataDir())
						},
						OnRevealLogs: func() error {
							return desktopcomponents.OpenInFileManager(filepath.Join(paths.UserDataDir(), "logs"))
						},
						OnRevealVendor: func() error {
							return desktopcomponents.OpenInFileManager(paths.VendorDir())
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
						Minimal:          state.minimalModeAccess(),
						Glass:            state.glassThemeAccess(),
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
						},
						OnSave: func() {
							state.saveDesktopConfigAsync(win)
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
						Err:        sessSnap.ModelsErr.SentenceText(),
						LastAction: settingsModelsSessionAction(sessSnap.LastAction, sessSnap.ModelsErr.SentenceText()),
						OnRefresh: func() {
							state.settingsTabMarkRefreshed("models", settingsTabOwner{gen: st.Generation, ref: sessions})
							if sessions != nil {
								sessions.RefreshModelCatalog(context.Background())
							}
						},
						OnApply: func() {
							state.mu.Lock()
							label := state.settingsModelLabel
							state.mu.Unlock()
							if sessions != nil && label != "" {
								sessions.SelectModelForSessionID(context.Background(), sessSnap.SelectedID, label)
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
					marketViewState := marketstore.ViewState{}
					if marketStore != nil {
						marketStore.EnsureInventory(context.Background())
						marketViewState = marketStore.Snapshot().Value
					}
					pluginRows := make([]settings.PluginRow, 0, len(marketViewState.Rows))
					for _, row := range marketViewState.Rows {
						pluginRows = append(pluginRows, settings.PluginRow{
							Name: row.Label, Status: row.Detail, Disabled: row.Enabled != nil && !*row.Enabled,
							CanToggle: row.CanToggle(), ReadOnlyReason: row.ReadOnlyReason, Error: row.Error,
						})
					}
					lastPluginAction := state.settingsPluginsLastAction
					pluginsSection = &settings.PluginsAccess{
						Inventory:        state.pluginInventoryAccess(),
						Rows:             pluginRows,
						HasListRPC:       marketViewState.HasListRPC,
						Err:              marketViewState.Err.SentenceText(),
						ClientReady:      marketViewState.ClientReady,
						Loading:          marketViewState.Loading,
						Busy:             marketSnap.InstallBusy,
						OperationMessage: marketSnap.InstallMsg,
						Activation:       marketSnap.Activation,
						CanRestart:       marketCanRestart,
						OnRestart:        state.restartMarketPlugins,
						LastAction:       lastPluginAction,
						OnRefresh: func() {
							state.settingsTabMarkRefreshed("plugins", settingsTabOwner{gen: st.Generation, ref: marketStore})
							if marketStore != nil {
								marketStore.Refresh(context.Background())
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
							state.setMainRouteLocked(routeMarket)
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
							state.mu.Unlock()
							if marketStore != nil && state.settingsTabShouldRefresh("market", settingsTabOwner{gen: st.Generation, ref: marketStore}) {
								marketStore.Refresh(context.Background())
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
									sessions.SetLastAction(errcopy.Sentence(err))
									return
								}
								go sessions.OpenCapturedSkillDirectory(context.Background(), ticket)
							}
						},
						SessionID:  sessSnap.SelectedID,
						Err:        sessSnap.SkillCatalog.Err,
						Notice:     sessSnap.SkillCatalog.Notice,
						Busy:       sessSnap.SkillCatalog.Busy || state.settingsSkillEditorSaving,
						LastAction: joinNonEmptyLines(state.settingsSkillsLastAction, settingsSkillsSessionAction(sessSnap.LastAction), sessSnap.SkillCatalog.Action),
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
						},
						OnDelete: func(name string) {
							if sessions != nil {
								sessions.DeleteSkillInventory(context.Background(), name)
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
								state.settingsTabMarkRefreshed("skills", settingsTabOwner{gen: st.Generation, ref: sessions, scope: sessions.SelectedSessionID()})
								sessions.RefreshSkills(state.lifecycleContext())
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
							state.setMainRouteLocked(routeConversation)
							state.mu.Unlock()
						},
						OnSetModelInvocable: func(name string, modelInvocable bool) {
							if sessions != nil {
								sessions.SetCatalogSkillInvocation(state.lifecycleContext(), name, modelInvocable)
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
							state.ensureCredentialsDraftSeededLocked()
						}
						state.mu.Unlock()
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
						state.ensureNativeUpdateCenterAsync(win)
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
						if state.settingsTabShouldRefresh("models", settingsTabOwner{gen: st.Generation, ref: sessions}) {
							state.refreshProviderProfiles()
							state.refreshVisionSettings()
							if sessions != nil {
								sessions.RefreshModelCatalog(context.Background())
							}
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
						if state.settingsTabShouldRefresh("mcp", settingsTabOwner{gen: st.Generation}) {
							state.refreshSettingsMCP()
						}
					},
					OnSelectPlugins: func() {
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
						if state.settingsTabShouldRefresh("plugins", settingsTabOwner{gen: st.Generation, ref: marketStore}) {
							state.refreshPluginInventory()
							if marketStore != nil {
								marketStore.RefreshShared(context.Background())
							}
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
						if sessions != nil && state.settingsTabShouldRefresh("skills", settingsTabOwner{gen: st.Generation, ref: sessions, scope: sessions.SelectedSessionID()}) {
							sessions.RefreshSkills(state.lifecycleContext())
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
						if manager != nil && state.settingsTabShouldRefresh("agent-presets", settingsTabOwner{gen: st.Generation, ref: manager}) {
							go manager.Refresh(context.Background())
						}
					},
					OnSelectVoice: func() {
						state.mu.Lock()
						state.settingsEmbeddedSection = settings.SectionVoice
						state.mu.Unlock()
					},
					OnSelectDeepSeekAPIs: func() {
						state.mu.Lock()
						state.settingsEmbeddedSection = settings.SectionDeepSeekAPIs
						state.mu.Unlock()
						if state.settingsTabShouldRefresh("deepseek-apis", settingsTabOwner{gen: st.Generation}) {
							state.refreshDeepSeekAPIs()
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
					},
					OnClose: func() {
						state.mu.Lock()
						state.setMainRouteLocked(routeConversation)
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
						state.credsDraft = credentialsDraft{}
						state.credsSeedGen++
						state.credsSeedLoading = false
						state.credsSaveErr = ""
						state.credsLastAction = ""
						state.credsShowSecrets = false
						state.mu.Unlock()
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
						if marketStore != nil && state.settingsTabShouldRefresh("market", settingsTabOwner{gen: st.Generation, ref: marketStore}) {
							marketStore.Refresh(context.Background())
						}
					},
					OnOpenRemote: func() {
						state.mu.Lock()
						state.settingsEmbeddedSection = settings.SectionRemote
						state.mu.Unlock()
					},
					OnOpenRemoteWorkspace: func() {
						state.mu.Lock()
						state.settingsEmbeddedSection = settings.SectionRemoteWorkspace
						state.mu.Unlock()
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
							if state.settingsTabShouldRefresh("usage", settingsTabOwner{gen: st.Generation, ref: usageStore, scope: sid}) {
								usageStore.Refresh(context.Background(), sid)
							}
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
				settingsModel = state.settingsSectionModel(settingsModel, func(_ *ui.Context, section settings.SectionID) {
					paintMainContent(section)
				})
				settings.ModalView(c, settingsModel)
				return
			}
		}
		paintMainContent = func(embeddedSection settings.SectionID) {
			if mainDestination.isMaintenance() {
				state.maintenancePanel(c, win, mainDestination)
				return
			}
			if acceptPanelOK && embeddedSection == "" && marketOpen && marketManager {
				state.pluginManagerView(c, themeID)
				return
			}
			if ((acceptPanelOK && embeddedSection == "" && marketOpen) || embeddedSection == settings.SectionMarket) && marketStore != nil {
				marketStore.EnsureInventory(context.Background())
				marketStore.SetNotify(func() {
					if win != nil {
						win.Update(func() {})
					}
				})
				marketStore.EnsureCatalog(i18n.Locale())
				state.mu.Lock()
				muxReadyMarket := state.dshMuxReady
				state.mu.Unlock()
				marketOwner := marketSnap.Owner
				spec := marketSnap.InstallSpec
				marketui.View(c, marketui.Model{
					Media:         mediaOwner,
					OnOpenManager: state.openNativePluginManager,
					OnOpenURL: func(rawURL string) {
						if err := mygo.Shell.OpenExternal(rawURL); err != nil {
							log.Println("market README link:", err)
						}
					},
					Embedded:         embeddedSection == settings.SectionMarket,
					Store:            marketStore,
					ThemeID:          themeID,
					SystemDark:       mygo.Theme.IsDark(),
					HarnessReady:     st.State == harness.Ready,
					MuxConnected:     muxReadyMarket,
					PluginDisabled:   false,
					InstallReady:     state.marketInstallReadyFor(nodeBin),
					InstallSpec:      &spec,
					InstallBusy:      marketSnap.InstallBusy,
					InstallMessage:   marketSnap.InstallMsg,
					InstallLog:       marketSnap.InstallLog,
					Activation:       marketSnap.Activation,
					CanRestart:       marketCanRestart,
					OnRestart:        state.restartMarketPlugins,
					NeedsAllowBuilds: marketSnap.NeedsAllowBuilds,
					AllowBuilds:      marketSnap.AllowBuilds,
					CanCancelInstall: marketSnap.CanCancelInstall,
					Incompatible:     marketSnap.Incompatible,
					OnClose: func() {
						state.mu.Lock()
						state.setMainRouteLocked(routeConversation)
						state.mu.Unlock()
					},
					OnRefresh: func() {
						state.settingsTabMarkRefreshed("market", settingsTabOwner{gen: st.Generation, ref: marketStore})
						marketStore.Refresh(context.Background())
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
							mygo.Dialog.Error(i18n.T(i18n.KeyCmdMarketExportErrorTitle), errcopy.Sentence(err))
							return
						}
						if path == "" {
							return
						}
						if err = os.WriteFile(path, data, 0600); err != nil {
							mygo.Dialog.Error(i18n.T(i18n.KeyCmdMarketExportErrorTitle), errcopy.Sentence(err))
						}
					},
					OnCopyInstallLog: func(log string) {
						if strings.TrimSpace(log) != "" {
							mygo.Clipboard.WriteText(log)
						}
					},
				})
				state.marketCoordinator().CommitInstallSpec(marketSnap.InstallSpec, spec)
				return
			}
			if acceptPanelOK && embeddedSection == "" && scheduleOpen && schedStore != nil {
				sid := ""
				sessionLinks := map[string]string{}
				sessionSnap := session.ViewState{}
				if sessions != nil {
					sessionSnap = sessions.Published().Value
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
							if err == nil {
								state.scheduleHistoryText = text
							}
							state.mu.Unlock()
							if err != nil {
								state.notifyUser(win, saveFailedToast(i18n.T(i18n.KeyCmdReadTaskHistory), err))
								return
							}
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
						state.setMainRouteLocked(routeConversation)
						state.mu.Unlock()
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
				state.mu.Unlock()
				imui.View(c, imui.Model{
					Embedded:       embeddedSection == "channels",
					Store:          imStore,
					ThemeID:        themeID,
					SystemDark:     mygo.Theme.IsDark(),
					HarnessReady:   st.State == harness.Ready,
					MuxConnected:   muxReadyIM,
					PluginDisabled: imPluginDisabled,
					Draft:          imStore.DraftBuffer(),
					OnClose: func() {
						if err := imStore.Persist(); err != nil {
							state.notifyUser(win, errcopy.Sentence(err))
							return
						}
						state.mu.Lock()
						state.setMainRouteLocked(routeConversation)
						state.mu.Unlock()
					},
					OnRefresh: func() {
						imStore.Refresh(context.Background())
					},
					OnOpenPluginWeb: func() {
						state.openWebViewHostPlugin(st.Endpoint.Origin, desktopwebview.PluginIM)
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
								if err := dropDisabledPlugin(state.cfgStore, "dsh-im"); err != nil {
									state.notifyUser(win, saveFailedToast(i18n.T(i18n.KeyCmdEnableDshIm), err))
									return
								}
							}
							state.notifyUser(win, i18n.T(i18n.KeyCmdDshImEnabledRestartTheAppIfThePageS))
							if win != nil {
								win.Update(func() {})
							}
						}()
					},
					OnSend: func(channelID, text string) {
						imStore.SendDraft(context.Background(), channelID, text)
					},
				})
				return
			}
			if embeddedSection == settings.SectionVoice {
				state.renderVoiceInputSection(c)
				return
			}
			if embeddedSection == settings.SectionDeepSeekAPIs {
				state.renderDeepSeekAPIsSection(c)
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
				state.mu.Lock()
				lastWebURL := state.lastWebViewURL
				state.mu.Unlock()
				var sessionLastAction string
				var sessionMsgs []struct{ Role, Body string }
				if sessions != nil {
					sessSnap := sessions.Published().Value
					sessionLastAction = sessSnap.LastAction
					for _, m := range sessSnap.Messages {
						sessionMsgs = append(sessionMsgs, struct{ Role, Body string }{m.Role, m.Body})
					}
				}
				model, obs, commitDraft := asstSvc.BuildPanelModel(assistantctl.PanelBuildOptions{
					Endpoint:          st.Endpoint,
					ThemeID:           themeID,
					SystemDark:        mygo.Theme.IsDark(),
					LastWebViewURL:    lastWebURL,
					SessionLastAction: sessionLastAction,
					SessionMessages:   sessionMsgs,
				})
				assistantObs = obs
				assistant.View(c, model)
				commitDraft()
				return
			}
			if acceptPanelOK && embeddedSection == "" && browserPreviewOpen {
				if state.browserController == nil {
					state.browserController = state.newBrowserController(win)
				}
				state.browserController.View(c, themeID, mygo.Theme.IsDark(), func() {
					state.mu.Lock()
					state.setMainRouteLocked(routeConversation)
					state.mu.Unlock()
				})
				return
			}

			if acceptPanelOK && embeddedSection == "" && webviewOpen {
				origin := st.Endpoint.Origin
				state.mu.Lock()
				host := state.webviewHost
				hostErr := state.webviewHostErr
				pluginTitle := strings.TrimSpace(state.activePluginViewLabel)
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
					Title:         pluginTitle,
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
						state.setMainRouteLocked(routeConversation)
						state.activePluginViewLabel = ""
						state.mu.Unlock()
						state.returnFromAccountPage()
					},
					OnOpenHost: func() {
						state.mu.Lock()
						state.lastWebViewOrigin = origin
						state.mu.Unlock()
						state.openWebViewURL(resumeURL)
					},
					OnCloseHost: func() {
						state.closeWebViewHost()
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
					},
				})
				return
			}
			if (acceptPanelOK && embeddedSection == "" && usageOpen && usageStore != nil) || embeddedSection == settings.SectionUsageStats {
				exportFailure := ui.Local(c.Root(), "usage-export-failure", func() string { return "" })
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
						settings.UsagePreferences(c, host, fg, muted)
					}
				}
				usageui.View(c, usageui.Model{
					Preferences: usagePreferences,
					Embedded:    embeddedSection == settings.SectionUsageStats,
					ActionError: firstNonEmptyString(accountActionError, *exportFailure),
					Store:       usageStore,
					OnExportStats: func(format string) {
						*exportFailure = ""
						data, err := usageStore.ExportStats(format)
						if err != nil {
							*exportFailure = errcopy.Sentence(err)
							return
						}
						ext := "csv"
						if format == "json" {
							ext = "json"
						}
						path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: win, Title: i18n.T(i18n.KeyCmdUsageExportStatistics), DefaultPath: "usage-" + format + "." + ext})
						if err != nil {
							*exportFailure = errcopy.Sentence(err)
							return
						}
						if path == "" {
							return
						}
						if err = os.WriteFile(path, data, 0600); err != nil {
							*exportFailure = errcopy.Sentence(err)
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
						state.setMainRouteLocked(routeConversation)
						state.mu.Unlock()
					},
					SessionID:  sid,
					ThemeID:    themeID,
					SystemDark: mygo.Theme.IsDark(),
					OnClose: func() {
						state.mu.Lock()
						state.setMainRouteLocked(routeConversation)
						state.mu.Unlock()
					},
					OnRefresh: func() {
						state.settingsTabMarkRefreshed("usage", settingsTabOwner{gen: st.Generation, ref: usageStore, scope: sid})
						state.settingsTabMarkRefreshed("usage-stats", usageStore.StatisticsOwner())
						usageStore.Refresh(context.Background(), sid)
					},
					OnOpenUsageWeb: func() {
						state.openAccountUsage(usageStore.Snapshot().Value.UsageURL, "usage")
					},
					OnOpenTopUp: func() {
						state.openAccountTopUp(usageStore.Snapshot().Value.TopUpURL)
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
			// The workbench remains available before Host connection and after failure.
			if sessions == nil {
				state.mu.Lock()
				if state.sessions == nil {
					state.replaceSessionsLocked(state.newSessionStoreLocked())
				}
				sessions = state.sessions
				state.mu.Unlock()
			}
			if sessions != nil {
				// Body under the title bar must Grow into remaining height (not Fill%).
				ui.Column(c).Grow(1).MinHeight(0).Children(func() {
					connBanner := connectionBanner(st, dshMuxReady, dshMuxLost)
					if st.State != harness.Ready {
						connBanner = "" // The workbench Host status owns startup and recovery.
					} else {
						if unavailable := connectionUnavailableMessage(dshBootstrapError, dshNetworkOffline, dshMuxLost); unavailable != "" {
							connBanner = unavailable
						}
					}
					var usageJobsToday *int
					usageBalance := ""
					if usageStore != nil {
						us := usageStore.Snapshot().Value
						usageBalance = us.BalanceSummary
						if n, ok := usagestore.TodayJobCount(us.DayBuckets); ok {
							usageJobsToday = &n
						}
					}
					published := sessions.Published()
					snap := published.Value
					slashLocale := i18n.Locale()
					state.mu.Lock()
					if state.agentPresetManager == nil {
						state.agentPresetManager = session.NewAgentPresetManager(sessions)
					}
					agentPresetManager = state.agentPresetManager
					activeDraft := state.composerDraftLocked(snap.SelectedID)
					composerRendered, composerSessionID = true, snap.SelectedID
					draft = activeDraft.Text
					state.composerTextInput = activeDraft.Client
					if state.composerSuggest.Selected == nil {
						state.composerSuggest.Selected = &state.composerSuggestSel
					}
					needSlashCommands := strings.HasPrefix(draft, "/") && snap.ClientReady
					if state.composerSlashStore != sessions || state.composerSlashRev != published.Rev || state.composerSlashLocale != slashLocale || state.composerSlash == nil {
						state.composerSlash = composer.BuildSlashCatalog(composer.StubSlashCommands(), composer.FromAPIList(snap.SlashCommands))
						state.composerSlashStore = sessions
						state.composerSlashRev = published.Rev
						state.composerSlashLocale = slashLocale
					}
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
					needSkillsRefresh := false
					if composerSkillsOpen && snap.SelectedID != "" && snap.SelectedID != state.composerSkillsSession {
						needSkillsRefresh = true
						state.composerSkillsSession = snap.SelectedID
					}
					if composerSkillsList.Selected == nil {
						composerSkillsList.Selected = &state.composerSkillsSel
					}
					compStart := captureComposerWriteback(state.writebackRevs.composerSuggest, state.writebackRevs.composerSkills, composerSuggest, composerSkillsList, composerSkillsOpen, composerSkillsFilter, skillCreateName, composerChipEdit)
					state.ensureTerminalLocked(workspaceRoot, systemDark)
					termPanel := state.terminalPanel
					termShellsNote := state.terminalShellsNote
					filesPaneOpen = state.filesPaneOpen
					state.mu.Unlock()
					state.bindComposerFileIntake(win, sessions, snap.SelectedID, activeDraft)
					if snap.SelectedModel != "" && modelPick != snap.SelectedModel {
						modelPick = snap.SelectedModel
					}
					if permissionPick != snap.SelectedPermission {
						permissionPick = snap.SelectedPermission
					}
					if snap.SelectedAgentPreset != "" && agentPresetPick != snap.SelectedAgentPreset {
						agentPresetPick = snap.SelectedAgentPreset
					}
					if needSlashCommands {
						sessions.EnsureSlashCommands(state.lifecycleContext())
					}
					if needSkillsRefresh {
						sessions.RefreshSkills(state.lifecycleContext())
					}
					filePaneWidth := float32(0)
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
								sessions.SetLastAction(errcopy.Sentence(err))
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
						finish(nil)
					}
					// The sidebar account seat mirrors the settings account row.
					accountUsage := usagestore.ViewState{}
					if state.usage != nil {
						accountUsage = state.usage.Snapshot().Value
					}
					avatarURL := ""
					if accountUsage.Details.Profile.Value.AvatarURL != nil {
						avatarURL = strings.TrimSpace(*accountUsage.Details.Profile.Value.AvatarURL)
					}
					uFrame := state.updaterCenter().FrameSnapshot()
					assistantNavName := asstSvc.WhaleNavName()
					connReady := st.State == harness.Ready && dshMuxReady
					updateInstalling := uFrame.Installing
					updatePhase := uFrame.Phase
					updateDetail := uFrame.Detail
					// One settings copy per build: the frame model only reads it, and each
					// Snapshot copies every namespace's documents.
					settingsView := currentSettingsView()
					// One report for every surface that speaks about a service that
					// is not ready: the strip, the sidebar's empty list and the
					// composer cannot disagree about why there are no sessions.
					hostReport := hoststate.Of(st)
					frameModel := frame.Model{
						Media:                 mediaOwner,
						HostNote:              hostReport.Note,
						HostPlaceholder:       hostReport.Placeholder,
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
							cfgStore := state.cfgStore
							inter := state.interactions
							state.mu.Unlock()
							if cfgStore == nil {
								state.refuseHostAction()
								return
							}
							if inter != nil {
								inter.SetAutoAllow(on)
							}
							state.enqueueConfigUpdate(cfgStore, func(c *config.Config) {
								c.AutoAllowTools = on
							})
						},
						ConnectionBanner: connBanner,
						HostStatus: func(c *ui.Context) {
							if mainDestination == routeConversation {
								state.workbenchHostStatus(c, win, st, sup, logs)
							}
						},
						OnReconnect: state.requestConnectionReconnect,
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
							state.setMainRouteLocked(routeUsage)
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
							state.setMainRouteLocked(routeUsage)
							state.mu.Unlock()
							sid := ""
							if sessions != nil {
								sid = sessions.SelectedSessionID()
							}
							if state.settingsTabShouldRefresh("usage", settingsTabOwner{gen: st.Generation, ref: usageStore, scope: sid}) {
								usageStore.Refresh(context.Background(), sid)
							}
						},
						TitlebarPreferences:     settingsstore.TitlebarPreferencesFrom(settingsView),
						DisableDeveloperTools:   !settingsstore.DeveloperToolsEnabled(settingsView),
						DeveloperToolsPending:   setStore != nil && !(settingsView.Described && settingsView.HasDocument) && settingsView.Err == "",
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
							state.submitComposer(win, sessions, activeDraft, snap.SelectedID, workspaceRoot, accelerated, snap.ComposerHostGeneration)
						},
						OnNativeCommandAction: func(owner composer.NativeCommandOwner, name string) bool {
							return state.nativeComposerCommandAction(win, sessions, activeDraft, snap.SelectedID, workspaceRoot, owner, name)
						},
						OnNativeCommandChanged: func() {
							if win != nil {
								win.Update(func() {})
							}
						},
						OnAttachComposerFiles: func(paths []string) {
							state.addTransferredComposerFiles(win, sessions, snap.SelectedID, activeDraft, paths)
						},
						OnRetryComposerFile:  func(index int) { state.prepareComposerFile(win, sessions, snap.SelectedID, activeDraft, index, true) },
						OnPasteComposerImage: func(explicit bool) { state.pasteComposerImage(win, sessions, snap.SelectedID, activeDraft, explicit) },
						EditMessageSeq:       &editMessageSeq,
						FeedbackCommentMsgID: &feedbackCommentMsgID,
						FeedbackCommentDraft: &feedbackCommentDraft,
						ModelPick:            &modelPick,
						PermissionPick:       &permissionPick,
						AgentPresetPick:      &agentPresetPick,
						ThinkingExpanded:     &thinkingExpanded,
						RenameSessionID:      &renameSessionID,
						RenameDraft:          &renameDraft,
						SessionSearch:        &sessionSearch,
						WorkspaceRoot:        workspaceRoot,
						FileExpanded:         &fileExpanded,
						SelectedFile:         &selectedFile,
						FileFilter:           &fileFilter,
						FilePreview:          &filePreview,
						FilePreviewErr:       &filePreviewErr,
						FilePreviewList:      &filePreviewList,
						FileEditing:          &fileEditing,
						FileEditDraft:        &fileEditDraft,
						FileTabs:             &fileTabs,
						FileBindingID:        fileBindingID,
						GitWorkbench:         &gitWorkbench,
						FileEditStatus:       &fileEditStatus,
						FileEditViewMode:     &fileEditViewMode,
						FileEditSplit:        &fileEditSplit,
						FilePaneWidth:        &filePaneWidth,
						RightPaneOpen:        filesPaneOpen,
						RightPaneTab:         &rightPaneTab,
						AgentsPromptDraft:    &agentsPromptDraft,
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
						},
						OnCloseImageLightbox: func() {
							imageLightboxPath = ""
							state.mu.Lock()
							state.imageLightboxPath = ""
							state.mu.Unlock()
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
						},
						OnInstallSkillHub: func(hit session.SkillHubHit) {
							if sessions != nil {
								sessions.InstallSkillHub(context.Background(), hit, session.SkillRootUser)
							}
						},
						OnDeleteSkill: func(name string) {
							state.mu.Lock()
							state.settingsEmbeddedSection = ""
							state.setMainRouteLocked(routeSettings)
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
						},
						SkillCreateName: &skillCreateName,
						OnCreateSkill: func(name string) {
							if sessions != nil {
								sessions.CreateSkillInventory(context.Background(), name, name, "", session.SkillRootUser)
							}
						},
						OnToggleSkillModel: func(name string, modelInvocable bool) {
							if sessions != nil {
								sessions.SetSkillInvocation(state.lifecycleContext(), name, modelInvocable, true)
							}
						},
						OnToggleComposerSkills: func(open bool) {
							state.mu.Lock()
							state.composerSkillsOpen = open
							state.mu.Unlock()
							if open && sessions != nil {
								sessions.RefreshSkills(state.lifecycleContext())
							}
						},
						OnNoDirectory:    func() { state.openNoDirectory(win, sessions, activeDraft, snap.SelectedID) },
						OnCreateSession:  func() { state.openGlobalWorkspaceSession(win, sessions) },
						DirectoryActions: state.projectDirectoryActions(win),
						OnCreateWorkspaceSession: func(generation uint64, id string) {
							state.openWorkspaceSession(win, sessions, generation, id)
						},
						OnSelectWorkspace: func(id string) {
							if sessions == nil {
								state.refuseHostAction()
								return
							}
							sessions.OpenWorkspacePicker()
							request := sessions.Published().Value.WorkspaceNavigation
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
							state.revealInFileManager(win, root)
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
						},
						OnMoveSessionBefore: func(sessionID, beforeSessionID string) {
							if sessions == nil {
								state.refuseHostAction()
								return
							}
							sessions.InsertSessionBefore(context.Background(), sessions.WorkspaceID(), sessionID, beforeSessionID)
						},
						WorkspaceMenuOpen: &state.workspaceMenuOpen,
						OnOpenAgents: func() {
							state.mu.Lock()
							state.filesPaneOpen = true
							state.rightPaneTab = "agents"
							state.mu.Unlock()
							filesPaneOpen, rightPaneTab = true, "agents"
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
										sessions.SetLastAction(errcopy.Sentence(err))
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
						},
						OnPickComposerImage: func() {
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: win, Title: i18n.T(i18n.KeyCmdAttachFilesImages), DefaultPath: workspaceRoot, Multiple: true})
							if err != nil {
								activeDraft.Error = errcopy.Sentence(err)
								return
							}
							state.addComposerFiles(win, sessions, snap.SelectedID, activeDraft, paths)
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
								sessions.SetLastAction(errcopy.Sentence(err))
								return
							}
							if len(paths) == 0 {
								return
							}
							go func() {
								text, err := sessions.TranscribeAudioFile(context.Background(), paths[0])
								if err != nil {
									sessions.SetLastAction(errcopy.Sentence(err))
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
							state.setMainRouteLocked(routeMarket)
							state.mu.Unlock()
						},
						SidebarPlugins: state.sidebarPluginEntries(i18n.Locale()),
						OnOpenSidebarPlugin: func(id string) {
							state.openInstalledPluginViewWithLabel(id, state.sidebarPluginLabel(id))
						},
						OnCreatePluginDraft: func(region string) {
							state.startCreatorPluginDraft(win, sessions, region)
						},
						OnOpenSchedule: func() {
							state.mu.Lock()
							state.setMainRouteLocked(routeSchedule)
							state.mu.Unlock()
						},
						OnOpenAssistant: func() {
							state.openWhaleFromSidebar(win)
						},
						OnOpenDesktopSettings:   state.openWorkbenchSettings,
						ConnectionReady:         connReady,
						AssistantName:           assistantNavName,
						DesktopUpdateInstalling: updateInstalling,
						DesktopUpdatePhase:      updatePhase,
						DesktopUpdateDetail:     updateDetail,
						OnOpenDesktopUpdate:     func() { state.openNativeUpdateCenter(win) },
						OnContactAccount:        state.contactAccountSupport,
						Account: sidebar.Account{
							Name:      strings.TrimSpace(accountUsage.Name),
							Detail:    strings.TrimSpace(accountUsage.Email),
							AvatarURL: avatarURL,
							SignedIn:  accountUsage.AccountStatus == "credential-stored" || strings.TrimSpace(accountUsage.Name) != "" || strings.TrimSpace(accountUsage.Email) != "",
							OnSignOut: func() { state.runAccountSignOut(win, state.accountSignOut.Request) },
							OnOpen:    state.openAccountSettings,
							OnSignIn:  state.startAccountSignIn,
							OnContact: state.contactAccountSupport, LauncherActions: state.phoneLauncherActions,
							RenderLauncherPopup: state.renderPhonePopup,
						},
						OnOpenSettings: func() { state.toggleApplicationSettings(c) },
						OnToggleTools: func() {
							state.mu.Lock()
							// Electron ▦ opens the files/diff right dock (not a full-width menu).
							next := !state.filesPaneOpen
							state.mu.Unlock()
							state.setFilesPaneOpen(next)
						},
						OnCloseTerminal: func() {
							state.mu.Lock()
							state.terminalSessions.CloseWorkspace(workspaceRoot)
							state.terminalPanel = nil
							state.terminalVisible = false
							state.layout.Bottom = 0
							state.mu.Unlock()
						},
						OnRestartTerminal: func() {
							state.restartTerminal(workspaceRoot, systemDark)
						},
						OnToggleTerminal: func() {
							state.toggleTerminal(workspaceRoot, systemDark)
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
							state.revealInFileManager(win, absPath)
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
								sessions.SetLastAction(errcopy.Sentence(err))
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
					})
					state.consumeNavigationFile(frameModel)
					state.gateAssistantNavigation(&frameModel)
					nextPromptRendered = true
					frame.View(c, frameModel)
					state.workspaceNavigationView(c, win, sessions, themeID, systemDark, func() { state.openNoDirectory(win, sessions, activeDraft, snap.SelectedID) })
					state.directoryPickerView(c, win, sessions, themeID, systemDark, func(mode, path string) {
						if mode == "mention" {
							root := strings.TrimSpace(workspaceRoot)
							if root == "" && state.cfgStore != nil {
								root = strings.TrimSpace(state.effectiveConfigFor(state.cfgStore).Workspace)
							}
							if err := composer.InsertDirectoryMention(&activeDraft.Text, root, path); err != nil && sessions != nil {
								sessions.SetLastAction(i18n.T(i18n.KeyComposerDirOutside))
							}
							return
						}
						state.applyWorkspacePath(win, sessions, path, false)
					})
					state.mu.Lock()
					state.commitComposerWritebackLocked(compStart, captureComposerWriteback(compStart.suggestRev, compStart.skillsListRev, composerSuggest, composerSkillsList, composerSkillsOpen, composerSkillsFilter, skillCreateName, composerChipEdit))
					previousChatPaneTab := state.commitMainViewWritebackLocked(wbStart, mainViewWritebackSnapshot{
						layout:               layout,
						draft:                activeDraft.Text,
						editMessageSeq:       editMessageSeq,
						feedbackCommentMsgID: feedbackCommentMsgID,
						feedbackCommentDraft: feedbackCommentDraft,
						modelPick:            modelPick,
						permissionPick:       permissionPick,
						agentPresetPick:      agentPresetPick,
						thinkingExpanded:     thinkingExpanded,
						renameSessionID:      renameSessionID,
						renameDraft:          renameDraft,
						sessionSearch:        sessionSearch,
						fileTabs:             fileTabs,
						gitWorkbench:         gitWorkbench,
						fileExpanded:         fileExpanded,
						selectedFile:         selectedFile,
						fileFilter:           fileFilter,
						filePreview:          filePreview,
						filePreviewErr:       filePreviewErr,
						filePreviewList:      filePreviewList,
						fileEditing:          fileEditing,
						fileEditDraft:        fileEditDraft,
						fileEditStatus:       fileEditStatus,
						fileEditViewMode:     fileEditViewMode,
						fileEditSplit:        fileEditSplit,
						rightPaneTab:         rightPaneTab,
						agentsPromptDraft:    agentsPromptDraft,
						goalDraft:            goalDraft,
						diffPath:             diffPath,
						diffText:             diffText,
						diffErr:              diffErr,
						planBoardOpen:        planBoardOpen,
						todoPanelCollapsed:   todoPanelCollapsed,
						chatPaneTab:          chatPaneTab,
						trajSearch:           trajSearch,
						trajSelectedID:       trajSelectedID,
						trajActualDur:        trajActualDur,
						trajTurnsFold:        trajTurnsFold,
						trajCallsFold:        trajCallsFold,
						trajDetailTab:        trajDetailTab,
						imageLightboxPath:    imageLightboxPath,
						gitCommitOpen:        gitCommitOpen,
						gitCommitDraft:       gitCommitDraft,
						gitPROpen:            gitPROpen,
					})
					if sessionSearch == wbStart.sessionSearch {
						sessionSearch = state.sessionSearch
					}
					if chatPaneTab == wbStart.chatPaneTab {
						chatPaneTab = state.chatPaneTab
					}
					state.sessionSearch = sessionSearch
					state.chatPaneTab = chatPaneTab
					state.mu.Unlock()
					frame.TraceConversationTabWriteback(previousChatPaneTab, chatPaneTab)
					// Search and lazy trajectory gating synchronously notify the
					// attached session store. Its refresh callback also takes
					// state.mu, so publish after unlocking.
					if sessions != nil {
						sessions.SetTrajectoryEnabled(chatPaneTab == "trajectory" && settingsstore.DeveloperToolsEnabled(settingsView))
						sessions.UpdateSessionSearch(state.lifecycleContext(), sessionSearch)
					}
				})
				return
			}

		}
		navAccountUsage := usagestore.ViewState{}
		if state.usage != nil {
			navAccountUsage = state.usage.Snapshot().Value
		}
		navAvatarURL := ""
		if navAccountUsage.Details.Profile.Value.AvatarURL != nil {
			navAvatarURL = strings.TrimSpace(*navAccountUsage.Details.Profile.Value.AvatarURL)
		}
		navigationModel := frame.Model{
			Store: sessions, Sizes: &layout, ThemeID: themeID, SystemDark: systemDark,
			FrameView:     sessionViews.view(sessions),
			SessionSearch: &sessionSearch, RenameSessionID: &renameSessionID, RenameDraft: &renameDraft,
			Interactions: interactions, HideArchivedList: cfgHideArchivedList, RightPaneTab: &rightPaneTab,
			OnToggleTools: func() {
				state.mu.Lock()
				next := !state.filesPaneOpen
				state.mu.Unlock()
				state.setFilesPaneOpen(next)
			},
			WorkspaceRoot: workspaceRoot, WorkspacePage: sharedNativeCaption && desktopChromeLocked,
			GitWorkbench: &gitWorkbench,
			Account: sidebar.Account{
				Name:      strings.TrimSpace(navAccountUsage.Name),
				Detail:    strings.TrimSpace(navAccountUsage.Email),
				AvatarURL: navAvatarURL,
				SignedIn:  navAccountUsage.AccountStatus == "credential-stored" || strings.TrimSpace(navAccountUsage.Name) != "" || strings.TrimSpace(navAccountUsage.Email) != "",
				OnSignOut: func() { state.runAccountSignOut(win, state.accountSignOut.Request) },
				OnOpen:    state.openAccountSettings,
				OnSignIn:  state.startAccountSignIn,
				OnContact: state.contactAccountSupport, LauncherActions: state.phoneLauncherActions,
				RenderLauncherPopup: state.renderPhonePopup,
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
			if settingsOpen && setStore != nil && settingsEmbeddedSectionFrame != "" {
				ui.Box(c).Fill().Background(chrome.BackdropCanvas(chrome.Color(colors.BG)))
				return
			}
			paintMainContent("")
		})
		if navigationModel.WorkspacePage {
			state.commitGitWorkbenchByRev(wbStart.revs.gitWorkbench, gitWorkbench)
		}
		paintSettingsOverlay()

	})
}

func trayIcon() []byte {
	return trayIconCached()
}

// firstNonEmptyString is the line the usage page writes above its cards: the
// failure that came last takes the place of the one that came before it.
func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
