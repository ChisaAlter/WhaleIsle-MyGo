package launcherctl

import (
	"context"
	"sync"

	"whaleisle/internal/app/updater"
	"whaleisle/internal/config"
	desktopcomponents "whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/startuprec"
)

// Drafts captures editable launcher preferences with generation protection so
// UI frames only write back fields modified on the user gesture.
type Drafts struct {
	Workspace         string
	WorkspaceGen      uint64
	QuitAfterStart    bool
	CloseToTray       bool
	AutoStartDesktop  bool
	AskOnUpdate       bool
	DownloadRoute     string
	RoutePopOpen      bool
	ImportOverwrite   bool
	ImportAttachments bool
}

// PanelSnapshot returns the full read-only launcher UI state under one lock.
type PanelSnapshot struct {
	Tab                   string
	Note                  string
	StartBusy             bool
	StartPreparing        bool
	StartFailed           bool
	StartErr              string
	TriedStart            bool
	PrefsError            string
	UpdateStatus          string
	UpdateAvailable       bool
	UpdateCheckBusy       bool
	InstallUpdateBusy     bool
	ComponentStatus       string
	Components            []desktopcomponents.Row
	ComponentsFresh       bool
	HarnessVersion        string
	Drafts                Drafts
	PersistenceFailedNote string
}

// Controller owns launcher lifecycle, cold-start recovery, diagnostics/repair,
// component discovery, isolated/portable guards, startup blank-session claim,
// and launcher-level update probing.
type Controller struct {
	deps HostDeps

	mu             sync.Mutex
	recoveryDiskMu sync.Mutex // Acquired before mu only when persisting recovery records; never nested inside mu.

	recovery  RecoveryState
	selection startupSelectionState

	componentsRefreshBusy bool
	componentsFresh       bool
	components            []desktopcomponents.Row
	harnessVersion        string

	vendorProbe    bgProbe[bool]
	inventoryProbe bgProbe[startupInventoryResult]

	tab               string
	note              string
	startBusy         bool
	startPreparing    bool
	startFailed       bool
	startErr          string
	triedStart        bool
	prefsError        string
	componentStatus   string
	updateStatus      string
	updateAvailable   bool
	updateCheckBusy   bool
	updateCheckGen    uint64
	installUpdateBusy bool
	installUpdateGen  uint64

	prefsSeeded       bool
	workspaceDraft    string
	workspaceGen      uint64
	quitAfterStart    bool
	autoStartDesktop  bool
	askOnUpdate       bool
	downloadRoute     string
	routePopOpen      bool
	closeToTray       bool
	importOverwrite   bool
	importAttachments bool
}

// NewController creates a launcher Controller bound to optional HostDeps.
func NewController(deps HostDeps) *Controller {
	return &Controller{deps: deps}
}

func (c *Controller) hostEnv() HostEnv {
	if c != nil && c.deps != nil {
		return c.deps.Env()
	}
	return HostEnv{}
}

func (c *Controller) updaterCenter() *updater.Center {
	if c != nil && c.deps != nil {
		return c.deps.Updater()
	}
	return nil
}

func (c *Controller) lifecycleContext() context.Context {
	if c != nil && c.deps != nil {
		if ctx := c.deps.LifecycleContext(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

// PanelState seeds uninitialized draft preferences from cfg and returns a snapshot.
func (c *Controller) PanelState(cfg *config.Config, fallbackWorkspace string) PanelSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if stringsTrim(c.workspaceDraft) == "" {
		seed := fallbackWorkspace
		if seed == "" && cfg != nil {
			seed = cfg.Workspace
		}
		if seed != "" {
			c.workspaceDraft = seed
			c.workspaceGen++
		}
	}
	if !c.prefsSeeded && cfg != nil {
		c.quitAfterStart = cfg.QuitAfterStart
		c.autoStartDesktop = cfg.AutoStartDesktop
		c.askOnUpdate = cfg.AskOnUpdate
		c.downloadRoute = cfg.DownloadRoute
		c.closeToTray = cfg.CloseToTray
		c.prefsSeeded = true
	}
	return c.panelSnapshotLocked()
}

func (c *Controller) panelSnapshotLocked() PanelSnapshot {
	tab := c.tab
	if tab == "" {
		tab = "home"
	}
	return PanelSnapshot{
		Tab:               tab,
		Note:              c.note,
		StartBusy:         c.startBusy,
		StartPreparing:    c.startPreparing,
		StartFailed:       c.startFailed,
		StartErr:          c.startErr,
		TriedStart:        c.triedStart,
		PrefsError:        c.prefsError,
		UpdateStatus:      c.updateStatus,
		UpdateAvailable:   c.updateAvailable,
		UpdateCheckBusy:   c.updateCheckBusy,
		InstallUpdateBusy: c.installUpdateBusy,
		ComponentStatus:   c.componentStatus,
		Components:        append([]desktopcomponents.Row(nil), c.components...),
		ComponentsFresh:   c.componentsFresh,
		HarnessVersion:    c.harnessVersion,
		Drafts: Drafts{
			Workspace:         c.workspaceDraft,
			WorkspaceGen:      c.workspaceGen,
			QuitAfterStart:    c.quitAfterStart,
			CloseToTray:       c.closeToTray,
			AutoStartDesktop:  c.autoStartDesktop,
			AskOnUpdate:       c.askOnUpdate,
			DownloadRoute:     c.downloadRoute,
			RoutePopOpen:      c.routePopOpen,
			ImportOverwrite:   c.importOverwrite,
			ImportAttachments: c.importAttachments,
		},
	}
}

// CommitDrafts applies only fields that changed relative to before, guarding
// workspace mutations by WorkspaceGen to avoid clobbering newer updates.
func (c *Controller) CommitDrafts(before *Drafts, next Drafts) {
	if c == nil || before == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if next.Workspace != before.Workspace && c.workspaceGen == before.WorkspaceGen {
		c.workspaceDraft = next.Workspace
		c.workspaceGen++
		before.WorkspaceGen = c.workspaceGen
		before.Workspace = next.Workspace
	}
	if next.QuitAfterStart != before.QuitAfterStart {
		c.quitAfterStart = next.QuitAfterStart
		before.QuitAfterStart = next.QuitAfterStart
	}
	if next.CloseToTray != before.CloseToTray {
		c.closeToTray = next.CloseToTray
		before.CloseToTray = next.CloseToTray
	}
	if next.AutoStartDesktop != before.AutoStartDesktop {
		c.autoStartDesktop = next.AutoStartDesktop
		before.AutoStartDesktop = next.AutoStartDesktop
	}
	if next.AskOnUpdate != before.AskOnUpdate {
		c.askOnUpdate = next.AskOnUpdate
		before.AskOnUpdate = next.AskOnUpdate
	}
	if next.DownloadRoute != before.DownloadRoute {
		c.downloadRoute = next.DownloadRoute
		before.DownloadRoute = next.DownloadRoute
	}
	if next.RoutePopOpen != before.RoutePopOpen {
		c.routePopOpen = next.RoutePopOpen
		before.RoutePopOpen = next.RoutePopOpen
	}
	if next.ImportOverwrite != before.ImportOverwrite {
		c.importOverwrite = next.ImportOverwrite
		before.ImportOverwrite = next.ImportOverwrite
	}
	if next.ImportAttachments != before.ImportAttachments {
		c.importAttachments = next.ImportAttachments
		before.ImportAttachments = next.ImportAttachments
	}
}

// DownloadRouteOverride returns the staged launcher download route draft if non-empty.
func (c *Controller) DownloadRouteOverride() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.downloadRoute
}

// CloseToTrayDraft returns the controller's CloseToTray toggle (seeded from config on panel load).
func (c *Controller) CloseToTrayDraft() (seeded bool, value bool) {
	if c == nil {
		return false, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.prefsSeeded, c.closeToTray
}

// SetCloseToTray initializes or updates the CloseToTray draft toggle.
func (c *Controller) SetCloseToTray(v bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.closeToTray = v
	c.mu.Unlock()
}

// ImportPrefs returns the controller's import overwrite/attachments toggles.
func (c *Controller) ImportPrefs() (overwrite, attachments bool) {
	if c == nil {
		return false, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.importOverwrite, c.importAttachments
}

// SetTab selects the active launcher navigation tab.
func (c *Controller) SetTab(tab string) {
	c.mu.Lock()
	c.tab = tab
	c.mu.Unlock()
}

// SetNote updates the launcher status banner text.
func (c *Controller) SetNote(note string) {
	c.mu.Lock()
	c.note = note
	c.mu.Unlock()
}

// SetPrefsError sets launcher preferences save error text.
func (c *Controller) SetPrefsError(errMsg string) {
	c.mu.Lock()
	c.prefsError = errMsg
	c.mu.Unlock()
}

// SetStatusMessages updates component & update status banner lines together.
func (c *Controller) SetStatusMessages(updateMsg, componentMsg string) {
	c.mu.Lock()
	if updateMsg != "" {
		c.updateStatus = updateMsg
	}
	c.componentStatus = componentMsg
	c.mu.Unlock()
}

// SetUpdateMessage replaces the launcher update banner status text.
func (c *Controller) SetUpdateMessage(msg string) {
	c.mu.Lock()
	c.updateStatus = msg
	c.mu.Unlock()
}

// TryBeginStart reserves launcher start preparation unless busy, quitting, importing, or restarting.
func (c *Controller) TryBeginStart() (workspaceDraft string, draftGen uint64, ok bool) {
	env := c.hostEnv()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.startPreparing || c.startBusy || env.LauncherImportBusy || env.TaskRestartPending || env.Quitting {
		return "", 0, false
	}
	c.startPreparing = true
	c.triedStart = true
	c.startBusy = true
	c.startErr = ""
	return stringsTrim(c.workspaceDraft), c.workspaceGen, true
}

// FinishStartPreparation clears startPreparing and applies updated start flags safely.
func (c *Controller) FinishStartPreparation(failed bool, errMsg string, ws string, expectedDraftGen uint64) {
	c.mu.Lock()
	c.startPreparing = false
	if failed {
		c.startBusy = false
		c.startFailed = true
		c.startErr = errMsg
	} else if ws != "" && c.workspaceGen == expectedDraftGen {
		c.workspaceDraft = ws
		c.workspaceGen++
	}
	c.mu.Unlock()
}

// BeginBootPreparation marks cold-start preparation active before the async launch check runs.
func (c *Controller) BeginBootPreparation() {
	c.mu.Lock()
	c.startPreparing = true
	c.mu.Unlock()
}

// EndPreparationOnly clears startPreparing on early completion without error overwrite.
func (c *Controller) EndPreparationOnly() {
	c.mu.Lock()
	c.startPreparing = false
	c.mu.Unlock()
}

type StartupActivity struct {
	Busy, Preparing bool
	Error           string
}

// StartupActivity is the small workbench projection; it does not load any
// maintenance inventory or seed preference drafts.
func (c *Controller) StartupActivity() StartupActivity {
	c.mu.Lock()
	defer c.mu.Unlock()
	return StartupActivity{Busy: c.startBusy, Preparing: c.startPreparing, Error: c.startErr}
}

// ObserveBootStatus updates startBusy/startFailed/startErr from a supervisor status transition.
func (c *Controller) ObserveBootStatus(prev harness.State, st harness.Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch st.State {
	case harness.Starting:
		c.startBusy = true
	case harness.Ready:
		c.startBusy = false
		c.startFailed = false
		c.startErr = ""
	case harness.Failed:
		c.startBusy = false
		if c.triedStart || prev == harness.Starting {
			c.startFailed = true
			if st.Err != nil {
				c.startErr = st.Err.Error()
			}
		}
	case harness.Stopped:
		c.startBusy = false
	}
}

// StartBusy reports whether the launcher currently considers a harness start in progress.
func (c *Controller) StartBusy() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startBusy
}

// SetStartBusy updates startBusy cleanly.
func (c *Controller) SetStartBusy(busy bool) {
	c.mu.Lock()
	c.startBusy = busy
	c.mu.Unlock()
}

// MarkColdStartBusy records triedStart and startBusy before launching the harness.
func (c *Controller) MarkColdStartBusy() {
	c.mu.Lock()
	c.triedStart = true
	c.startBusy = true
	c.mu.Unlock()
}

// AppendStartError appends line to startErr when non-empty (used by crash reporting).
func (c *Controller) AppendStartError(line string) {
	if c == nil || line == "" {
		return
	}
	c.mu.Lock()
	if c.startErr != "" {
		c.startErr += "\n" + line
	}
	c.mu.Unlock()
}

// ComponentStatus returns the last status banner line for component actions.
func (c *Controller) ComponentStatus() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.componentStatus
}

// SetComponentActivity updates component/update status lines and optionally invalidates components cache.
func (c *Controller) SetComponentActivity(updateMsg, componentMsg string, markStale bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if updateMsg != "" {
		c.updateStatus = updateMsg
	}
	c.componentStatus = componentMsg
	if markStale {
		c.componentsFresh = false
	}
	c.mu.Unlock()
}

// SetImportRecoveryStatus records recovered import state into launcher controller fields.
func (c *Controller) SetImportRecoveryStatus(tab string, busy, failed bool, errMsg string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if tab != "" {
		c.tab = tab
	}
	c.startBusy = busy
	c.startFailed = failed
	c.startErr = errMsg
	c.mu.Unlock()
}

// SetImportAttachments configures the attachment checkbox draft when catalog discovery populates it.
func (c *Controller) SetImportAttachments(v bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.importAttachments = v
	c.mu.Unlock()
}

// StartErr returns the current launcher start error message.
func (c *Controller) StartErr() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startErr
}

// SetStartError explicitly marks start failure/status (used in tests or error handling).
func (c *Controller) SetStartError(busy, failed bool, errMsg string) {
	c.mu.Lock()
	c.startBusy = busy
	c.startFailed = failed
	c.startErr = errMsg
	c.mu.Unlock()
}

// ResetWorkspaceDraft resets the draft workspace after user confirmation (or test).
func (c *Controller) ResetWorkspaceDraft(ws string) {
	c.mu.Lock()
	c.workspaceDraft = ws
	c.workspaceGen++
	c.mu.Unlock()
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 {
		ch := s[len(s)-1]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			s = s[:len(s)-1]
		} else {
			break
		}
	}
	return s
}

var _ = startuprec.UpdateAttentionScope{}
