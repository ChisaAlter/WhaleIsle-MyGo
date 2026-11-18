// Package projectctl owns the Project, Team, requirements, deliveries, and visual preview application workflows.
package projectctl

import (
	"context"
	"sync"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/projectpreference"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

// HostSnapshot captures the host connection and UI visibility state outside Manager.mu.
type HostSnapshot struct {
	Client           *dsh.Client
	Mux              *dsh.Mux
	Status           harness.Status
	HasSupervisor    bool
	SupervisorID     any
	SupervisorStatus harness.Status
	Quitting         bool
	OverlayOpen      bool
	ImportBusy       bool
	RemoteConnected  bool
}

// VisualCaptureResult mirrors the PNG and height measurements returned by the HTML capture helper.
type VisualCaptureResult struct {
	PNG            string
	ContentHeight  int
	CapturedHeight int
}

// VisualSurfaceOptions configures an interactive isolated HTML preview surface.
type VisualSurfaceOptions struct {
	HTML       string
	Width      int
	Appearance string
	Inline     bool
}

// VisualSurface abstracts the interactive preview handle without coupling projectctl to windowing packages.
type VisualSurface interface {
	session.VisualReplySurface
	session.VisualReplyInteractive
	ContentHeight() int
	SetHooks(onExpire, onHeight func())
}

// AuthorizedFileRequest carries the verified project file admission context for the host editor panes.
type AuthorizedFileRequest struct {
	Target         projectflow.FileTarget
	Binding        string
	Client         *dsh.Client
	HostGeneration int
	ThemeID        string
	SystemDark     bool
	Local          bool
	AccessCurrent  func() bool
	CheckPath      func(path string) error
}

// LeadDraftSnapshot summarizes whether a Lead composer draft is free to receive recovery text.
type LeadDraftSnapshot struct {
	Text           string
	HasAttachments bool
	Sending        bool
	Preparing      int
	EditSeq        float64
	EditSessionID  string
}

// RecoveryDraftTarget represents a Lead composer draft that can be staged for recovery submission.
type RecoveryDraftTarget interface {
	Snapshot() LeadDraftSnapshot
	StageRecoveryText(text string)
}

// HostDeps supplies host shell capabilities to Manager without coupling projectctl to mygo or mainState internals.
type HostDeps struct {
	HostSnapshot func() HostSnapshot

	LifecycleContext func() context.Context
	Sessions         func() *session.Store
	Config           func() *config.Store
	MainWindow       func() any

	RunOnMain    func(func())
	UpdateWindow func(win any, fn func())
	Invalidate   func(win any)

	OpenDirectoryDialog    func(win any, title, defaultPath string) ([]string, error)
	OpenWorkspaceDirectory func(path string) error
	EnsureWorkspace        func(ctx context.Context, ep harness.Endpoint, dir string) error

	OpenModelSettings    func()
	ValidateRelativeFile func(root, canonicalPath string) (string, error)
	OpenAuthorizedFile   func(win any, req AuthorizedFileRequest) error
	ManagedMarketProfile func(st harness.Status) string

	CaptureWorkspaceAdoption func(sessions *session.Store) any
	ApplyWorkspaceAdoption   func(adoption any, sessions *session.Store, sourceSessionID, leadSessionID string) bool
	PlaceLeadDraft           func(sessionID, text string) error
	SubmitLeadDraft          func(win any, sessions *session.Store, sessionID, text string) error
	OccupiedDraftSessionIDs  func() []string

	AcquireExternalMaintenance func() (func(), error)
	ProtectTaskAction          func(ctx context.Context, win any, operation string, commit func() error) (bool, error)
	RestartHost                func(ctx context.Context) error

	VisualReaderAvailable func() (bool, string)
	CaptureVisualHTML     func(ctx context.Context, html string, width int, appearance string) (VisualCaptureResult, error)
	DecodeVisualBitmap    func(preview *projects.VisualPagePreview, png []byte) error
	NewVisualSurface      func(ctx context.Context, win any, opts VisualSurfaceOptions) (VisualSurface, error)
}

// Manager coordinates all Project, observer, team, requirement, delivery, and visual preview controllers.
// Lock invariant: Manager.mu is strictly a leaf lock and is never held across HostDeps callbacks, RPCs, or file I/O.
type Manager struct {
	mu                sync.Mutex
	deps              HostDeps
	primary           *Controller
	observers         map[*projects.ObserverState]*Controller
	observerReadings  map[observerIdentity]*observerReadingState
	ordinaryTeam      *ordinaryTeamState
	projectTeams      map[string]*projectTeamState
	preference        projectpreference.Controller
	windowHidden      bool
	presentationFrame uint64
	primaryPresented  uint64
}

// Controller owns one Project navigation/presentation scope (either the primary sidebar or an observer).
type Controller struct {
	mgr       *Manager
	isPrimary bool
	flow      flowState
	presented uint64
}

type flowState struct {
	materialCollection            *materialCollectionState
	teamQuery                     string
	teamHistory                   bool
	loadAttempted                 bool
	pendingCreate                 string
	createAdoption                any
	directoryIntent               *directoryIntent
	directoryChecks               map[string]*directoryCheck
	createError                   string
	searchQuery                   string
	history                       bool
	historical                    *historicalState
	historicalGeneration          uint64
	activity                      *activityState
	deliveries                    *deliveryState
	documents                     *documentState
	refreshLoop                   *refreshLoop
	refreshNote                   string
	processCancel                 context.CancelFunc
	processMux                    *dsh.Mux
	processWatchGeneration        uint64
	reports                       *projectflow.ReportBrowser
	materials                     map[string]*projectflow.MaterialDraft
	materialKey                   string
	archiveCandidate              *projectflow.Project
	lifecycleNote                 string
	assetDiff                     *projectflow.WorkDiff
	assetNote                     string
	requestRevertError            string
	visualResults                 *visualResultsState
	requestProjectID, requestKind string
	requestPage                   projectflow.RequestPage
	requestCard                   *projectflow.RequestCard
	requestCards                  map[string]projectflow.RequestCard
	requestReverts                *requestRevertState
	requestCursors                []string
	requestLoaded                 bool
	requestGeneration             uint64

	detail                     *projectflow.WorkDetail
	process                    *projectflow.ProcessSnapshot
	detailGeneration           uint64
	teamID, teamNote           string
	team                       *projectflow.Overview
	navigation                 uint64
	client                     *dsh.Client
	listing                    projectflow.Listing
	expanded, busy, loaded     bool
	err, renameID, renameDraft string
	generation                 uint64
	observerSection            string
	observerOpen               bool
	readOperation              *readOperation
	catalogRead                *readOperation
}

func (f *flowState) visible() bool { return f.expanded || f.observerOpen }

// NewManager creates a Manager backed by the supplied HostDeps.
func NewManager(deps HostDeps) *Manager {
	m := &Manager{
		deps:         deps,
		observers:    make(map[*projects.ObserverState]*Controller),
		projectTeams: make(map[string]*projectTeamState),
	}
	m.primary = &Controller{mgr: m, isPrimary: true}
	return m
}

// Primary returns the primary sidebar Project controller.
func (m *Manager) Primary() *Controller { return m.primary }

// Observer returns or lazily creates an independent Controller for the given ObserverState.
func (m *Manager) Observer(o *projects.ObserverState) *Controller {
	snap := m.hostSnapshot()
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.observerLocked(o, snap.Client)
}

func (m *Manager) observerLocked(o *projects.ObserverState, client *dsh.Client) *Controller {
	if m.observers == nil {
		m.observers = make(map[*projects.ObserverState]*Controller)
	}
	c := m.observers[o]
	if c == nil && o != nil {
		// A header-local state may be recreated while an accepted action still
		// owns the hidden controller. Reattach that owner instead of admitting a
		// second action against a fresh, apparently idle controller.
		for previous, pending := range m.observers {
			if previous != nil && pending.flow.client == client && !pending.flow.observerOpen && readingIdentity(previous, client) == readingIdentity(o, client) {
				c = pending
				o.RestoreReading(previous.Reading())
				previous.DetachReads()
				delete(m.observers, previous)
				m.observers[o] = c
				break
			}
		}
	}
	if c == nil {
		listing := m.primary.flow.listing
		listing.Projects = append([]projectflow.Project(nil), listing.Projects...)
		section := ""
		if o != nil {
			section = o.Section
			o.Initialized = false
		}
		c = &Controller{
			mgr: m,
			flow: flowState{
				client:          client,
				listing:         listing,
				loaded:          m.primary.flow.loaded,
				observerOpen:    true,
				observerSection: section,
			},
		}
		if o != nil {
			if saved := m.observerReadings[readingIdentity(o, client)]; saved != nil {
				saved.restore(&c.flow)
				o.RestoreReading(saved.reading)
			}
		}
		m.observers[o] = c
	}
	c.flow.observerOpen = true
	c.presented = m.presentationFrame
	return c
}

// BindModel constructs the primary projects.Model with frame theme and observer bindings.
func (m *Manager) BindModel(win any, themeID string, systemDark bool) projects.Model {
	if cfg := m.configStore(); cfg != nil && !MountEnabled(cfg.Get(), DevelopmentOverride()) {
		return projects.Model{}
	}
	m.mu.Lock()
	m.primaryPresented = m.presentationFrame
	m.mu.Unlock()
	model := m.primary.Model(win)
	m.bindTeamModel(win, &model)
	model.ThemeID, model.SystemDark = themeID, systemDark
	m.primary.bindVisualResults(win, &model)
	// Opening an observer does not make the sidebar own its controller lifetime.
	model.OnObserver = nil
	model.OnObserverSection = nil
	model.CloseObserver = m.CloseObserver
	observerThemeID, observerSystemDark := themeID, systemDark
	model.ObserverModel = func(o *projects.ObserverState) projects.Model {
		snap := m.hostSnapshot()
		m.mu.Lock()
		c := m.observerLocked(o, snap.Client)
		m.mu.Unlock()
		observer := c.Model(win)
		c.bindRequirements(win, &observer, o)
		observer.ThemeID, observer.SystemDark = observerThemeID, observerSystemDark
		c.bindVisualResults(win, &observer)
		return observer
	}
	return model
}

// BindFrameProjects populates dst with BindModel and attaches native file/settings consumers.
func (m *Manager) BindFrameProjects(win any, dst *projects.Model, themeID string, systemDark bool) {
	if dst == nil {
		return
	}
	*dst = m.BindModel(win, themeID, systemDark)
	m.BindNativeConsumers(win, dst)
}

// LoadedProjects returns a copy of the active Host's loaded projects when available.
func (m *Manager) LoadedProjects(client *dsh.Client) ([]projectflow.Project, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f := &m.primary.flow
	if client == nil || f.client != client || !f.loaded || !f.listing.Available {
		return nil, false
	}
	return append([]projectflow.Project(nil), f.listing.Projects...), true
}

// SetWindowHidden updates window visibility and pauses background refresh loops while hidden.
func (m *Manager) SetWindowHidden(hidden bool) {
	m.mu.Lock()
	m.windowHidden = hidden
	if hidden {
		m.primary.stopRefreshLocked()
		m.primary.stopProcessLocked()
		m.primary.closeVisualPagesLocked()
		for _, c := range m.observers {
			c.stopRefreshLocked()
			c.stopProcessLocked()
			c.closeVisualPagesLocked()
		}
	}
	m.mu.Unlock()
}

// Suspend stops live process watchers, refresh loops, and historical browsers across all controllers.
func (m *Manager) Suspend() {
	m.mu.Lock()
	m.primary.stopProcessLocked()
	m.primary.stopRefreshLocked()
	m.primary.closeHistoricalLocked()
	for _, c := range m.observers {
		c.stopProcessLocked()
		c.stopRefreshLocked()
		c.closeHistoricalLocked()
	}
	m.mu.Unlock()
}

// Close disposes the primary controller and every retained observer controller.
func (m *Manager) Close() {
	m.mu.Lock()
	m.primary.disposeLocked()
	for _, c := range m.observers {
		c.disposeLocked()
	}
	m.observers = nil
	m.observerReadings = nil
	m.mu.Unlock()
}

// Dispose resets the controller's active streams, browsers, and visual previews.
func (c *Controller) Dispose() {
	c.mgr.mu.Lock()
	c.disposeLocked()
	c.mgr.mu.Unlock()
}

func (c *Controller) disposeLocked() {
	c.closeObservationLocked()
	c.flow.requestCards = nil
	c.flow.generation++
	c.flow.client = nil
}

func (m *Manager) hostSnapshot() HostSnapshot {
	if m.deps.HostSnapshot != nil {
		return m.deps.HostSnapshot()
	}
	return HostSnapshot{}
}

func (m *Manager) lifecycleContext() context.Context {
	if m.deps.LifecycleContext != nil {
		if ctx := m.deps.LifecycleContext(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

func (m *Manager) sessionsStore() *session.Store {
	if m.deps.Sessions != nil {
		return m.deps.Sessions()
	}
	return nil
}

func (m *Manager) configStore() *config.Store {
	if m.deps.Config != nil {
		return m.deps.Config()
	}
	return nil
}

func (m *Manager) updateWindow(win any, fn func()) {
	if fn == nil {
		fn = func() {}
	}
	if m.deps.UpdateWindow != nil {
		m.deps.UpdateWindow(win, fn)
		return
	}
	if m.deps.RunOnMain != nil {
		m.deps.RunOnMain(fn)
		return
	}
	fn()
}

func (m *Manager) invalidateWindow(win any) {
	if m.deps.Invalidate != nil {
		m.deps.Invalidate(win)
		return
	}
	if m.deps.UpdateWindow != nil {
		m.deps.UpdateWindow(win, func() {})
	}
}

func projectRPC(client *dsh.Client) projectflow.Client {
	return projectflow.Client{Origin: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTP: client.HTTP}
}

func currentWorkspace(cfgStore *config.Store) string {
	workspace := ""
	if cfgStore != nil {
		workspace = cfgStore.Get().Workspace
	}
	if workspace == "" {
		workspace = paths.DefaultWorkspace()
	}
	return workspace
}

func localizedText(zh, en string) string {
	return i18n.Pick(zh, en)
}
