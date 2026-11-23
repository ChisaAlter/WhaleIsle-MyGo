package marketctl

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"reflect"
	"slices"
	"strings"
	"sync"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/state"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
)

// Owner identifies a consistent (client, store, storeGeneration) snapshot
// used to reject UI actions triggered before a Host reconnect or store swap.
type Owner struct {
	Client     *dsh.Client
	Store      *marketstore.Store
	Generation uint64
}

// Attempt tracks one active or awaiting-approval marketplace install/change RPC.
type Attempt struct {
	Client        *dsh.Client
	Store         *marketstore.Store
	Generation    uint64
	Spec          string
	Method        string
	Target        *marketstore.InstallTarget
	ReceiptID     string
	RequestID     string
	Ctx           context.Context
	Cancel        context.CancelFunc
	CancelPending bool
	ContinueBatch bool
}

// ViewSnapshot is the published, immutable view of all UI-relevant market
// coordinator state; AllowBuilds and Incompatible are shared, not copied.
type ViewSnapshot struct {
	Store            *marketstore.Store
	Open             bool
	Manager          bool
	InstallSpec      string
	InstallBusy      bool
	CanCancelInstall bool
	InstallMsg       string
	InstallLog       string
	Activation       marketstore.Activation
	CanRestart       bool
	NeedsAllowBuilds bool
	AllowBuilds      []string
	Incompatible     []api.ChangeResultErrorIncompatibleItem
	Owner            Owner
	WorkerBusy       bool
	RepairBusy       bool

	env restartEnv // the shell environment this view was built against
}

type nativePluginBinding struct {
	controller  *manager.Controller
	client      *dsh.Client
	owner       uint64
	formSubject manager.Subject
}

// Coordinator owns marketplace installation/update state machines, batch
// receipts, native plugin manager binding, plugin inventory controller, and
// install readiness probing under one local mutex (mu), completely free of
// cross-lock nesting with dshMu or shell locks.
type Coordinator struct {
	deps HostDeps

	mu sync.Mutex

	store            *marketstore.Store
	client           *dsh.Client
	open             bool
	managerMode      bool
	batchReceipts    map[string]string
	batch            []marketstore.InstallTarget
	active           *Attempt
	worker           *Attempt
	approval         *Attempt
	catalogTarget    *marketstore.InstallTarget
	installSpec      string
	installBusy      bool
	repairBusy       bool
	installRequestID string
	installMsg       string
	installLog       string
	activation       marketstore.Activation
	needsAllowBuilds bool
	allowBuilds      []string
	incompatible     []api.ChangeResultErrorIncompatibleItem

	installReady bgProbe[bool]

	profileDir     string
	sidebarPlugins bgProbe[[]manager.SidebarPluginEntry]

	nativePlugins           nativePluginBinding
	pluginInventory         plugininventory.Controller
	inventoryRefreshRunning bool
	inventoryRefreshPending bool
	inventorySource         *manager.HostInventory
	pluginEventsReady       bool
	inventoryDeferred       bool
	inventoryTraceReady     bool

	view    state.Published[ViewSnapshot]
	viewEnv restartEnv
}

// NewCoordinator constructs a marketplace Coordinator attached to optional HostDeps.
func NewCoordinator(deps HostDeps) *Coordinator {
	return &Coordinator{deps: deps}
}

// EnsureStore initializes and returns the underlying market store safely.
func (c *Coordinator) EnsureStore() *marketstore.Store {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.store == nil {
		c.store = marketstore.New()
	}
	st := c.store
	c.unlockPublish()
	return st
}

// SetStore replaces the coordinator's market store (primarily for tests).
func (c *Coordinator) SetStore(st *marketstore.Store) {
	c.mu.Lock()
	c.store = st
	c.unlockPublish()
}

// SetClient sets the coordinator's active Host transport without touching shell mutexes.
func (c *Coordinator) SetClient(client *dsh.Client) {
	c.mu.Lock()
	c.client = client
	c.unlockPublish()
}

// AttachHost binds a connected DSH client and managed profile directory to the
// market store, plugin inventory, and native plugin manager.
func (c *Coordinator) AttachHost(client *dsh.Client, managedProfileDir string, onRefresh func()) {
	c.attachHost(client, managedProfileDir, onRefresh, false)
}

// AttachHostForWorkspace keeps the default desktop's inventory read behind its
// core workspace initialization. Explicit manager/settings/A2 consumers still
// read immediately through the same HostInventory owner.
func (c *Coordinator) AttachHostForWorkspace(client *dsh.Client, managedProfileDir string, onRefresh func()) {
	c.attachHost(client, managedProfileDir, onRefresh, true)
}

func (c *Coordinator) attachHost(client *dsh.Client, managedProfileDir string, onRefresh func(), deferred bool) {
	store := c.EnsureStore()
	c.mu.Lock()
	c.client = client
	c.profileDir = managedProfileDir
	c.unlockPublish()
	c.sidebarPlugins.clear()
	store.SetUpdateProfileDir(managedProfileDir)
	store.Attach(client, onRefresh)
	if client == nil {
		c.BindNativePluginManager(nil)
	} else {
		c.bindNativePluginManagerCaller(client, client, deferred)
	}
	if client != nil && !deferred {
		c.ensureInventory()
	}
}

// WorkspaceReady releases only the current transport's startup deferral. It
// does not gate user-driven inventory reads if workspace initialization fails.
func (c *Coordinator) WorkspaceReady(client *dsh.Client) bool {
	c.mu.Lock()
	if client == nil || c.nativePlugins.client != client {
		c.unlockPublish()
		return false
	}
	if !c.inventoryDeferred {
		c.unlockPublish()
		return true
	}
	c.inventoryDeferred = false
	c.unlockPublish()
	c.ensureInventory()
	return true
}

// DetachHost invalidates in-flight attempts and disconnects Host transports from
// marketplace stores and controllers.
func (c *Coordinator) DetachHost() {
	c.Invalidate()
	c.mu.Lock()
	c.client = nil
	c.profileDir = ""
	store := c.store
	c.unlockPublish()
	c.sidebarPlugins.clear()
	if store != nil {
		store.SetUpdateProfileDir("")
		store.Detach()
	}
	c.BindNativePluginManager(nil)
}

// Store returns the current market store if initialized.
func (c *Coordinator) Store() *marketstore.Store {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.store
}

// Open reports whether the marketplace panel is currently open.
func (c *Coordinator) Open() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.open
}

// SetOpen toggles whether the marketplace panel is visible.
func (c *Coordinator) SetOpen(open bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.open = open
	c.unlockPublish()
}

// SetManagerMode selects native plugin manager vs discovery market view.
func (c *Coordinator) SetManagerMode(managerMode bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.managerMode = managerMode
	c.unlockPublish()
}

// ManagerMode reports whether native plugin manager mode is active inside market.
func (c *Coordinator) ManagerMode() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.managerMode
}

// SetInstallSpec stores the editable GitHub/npm plugin specification input.
func (c *Coordinator) SetInstallSpec(spec string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.installSpec = spec
	c.unlockPublish()
}

// CommitInstallSpec updates installSpec only if it still matches expected before-frame value.
func (c *Coordinator) CommitInstallSpec(before, next string) {
	if c == nil || next == before {
		return
	}
	c.mu.Lock()
	if c.installSpec == before {
		c.installSpec = next
	}
	c.unlockPublish()
}

// InstallSpecPointer exposes a shell-safe read-only or mutable spec helper via ViewSnapshot.
func (c *Coordinator) CaptureOwner(store *marketstore.Store) Owner {
	c.mu.Lock()
	defer c.mu.Unlock()
	var gen uint64
	if store != nil {
		gen = store.RuntimeToken()
	}
	return Owner{Client: c.client, Store: store, Generation: gen}
}

// restartEnv is the part of the shell's HostEnvSnapshot the published view
// depends on; ViewSnapshot republishes when it changes.
type restartEnv struct {
	hasSupervisor, taskRestartPending, hostStarting bool
}

func restartEnvOf(env HostEnvSnapshot) restartEnv {
	return restartEnv{hasSupervisor: env.HasSupervisor, taskRestartPending: env.TaskRestartPending, hostStarting: env.HostStarting}
}

// storeRuntime reads the store's published RuntimeToken without its lock.
func storeRuntime(st *marketstore.Store) uint64 {
	if st == nil {
		return 0
	}
	return st.Snapshot().Value.Runtime
}

// buildViewLocked assembles the view from the coordinator's fields, the
// last observed shell environment and the store's published runtime token.
// Slices are shared: allowBuilds and incompatible are replaced wholesale.
func (c *Coordinator) buildViewLocked() ViewSnapshot {
	env := c.viewEnv
	return ViewSnapshot{
		Store:            c.store,
		Open:             c.open,
		Manager:          c.managerMode,
		InstallSpec:      c.installSpec,
		InstallBusy:      c.installBusy,
		CanCancelInstall: c.installBusy && c.installRequestID != "",
		InstallMsg:       c.installMsg,
		InstallLog:       c.installLog,
		Activation:       c.activation,
		CanRestart:       env.hasSupervisor && c.worker == nil && !env.taskRestartPending && !env.hostStarting,
		NeedsAllowBuilds: c.needsAllowBuilds,
		AllowBuilds:      c.allowBuilds,
		Incompatible:     c.incompatible,
		Owner:            Owner{Client: c.client, Store: c.store, Generation: storeRuntime(c.store)},
		WorkerBusy:       c.worker != nil,
		RepairBusy:       c.repairBusy,
		env:              env,
	}
}

// unlockPublish publishes the view if it differs from the current version,
// then releases c.mu. Every critical section that may have changed a view
// field ends with it; sections that changed nothing mint no revision.
func (c *Coordinator) unlockPublish() {
	next := c.buildViewLocked()
	if v := c.view.Load(); v == nil || !viewEqual(v.Value, next) {
		c.view.Publish(next)
	}
	c.mu.Unlock()
}

// ViewSnapshot returns the published point-in-time view for UI rendering.
// The same *Version is shared until a coordinator change, a new shell
// environment or a store reattachment publishes the next revision; the
// fast path neither locks the coordinator nor copies.
func (c *Coordinator) ViewSnapshot() *state.Version[ViewSnapshot] {
	env := restartEnvOf(c.hostEnv())
	if v := c.view.Load(); v != nil && v.Value.env == env && v.Value.Owner.Generation == storeRuntime(v.Value.Store) {
		return v
	}
	c.mu.Lock()
	c.viewEnv = env
	c.unlockPublish()
	return c.view.Load()
}

// ViewChanged closes when the next view version is published.
func (c *Coordinator) ViewChanged() <-chan struct{} { return c.view.Changed() }

// IsBusy reports whether an install worker, generic install, or startup repair owns the market slot.
func (c *Coordinator) IsBusy() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.worker != nil || c.installBusy || c.repairBusy
}

// TryAcquireRepairSlot reserves the shared install slot for offline plugin repair
// so concurrent market installations cannot interleave with profile edits.
func (c *Coordinator) TryAcquireRepairSlot() (release func(), ok bool) {
	if c == nil {
		return func() {}, true
	}
	c.mu.Lock()
	if c.worker != nil || c.installBusy || c.repairBusy {
		c.unlockPublish()
		return nil, false
	}
	c.installBusy = true
	c.repairBusy = true
	c.unlockPublish()
	var once sync.Once
	return func() {
		once.Do(func() {
			c.mu.Lock()
			c.installBusy = false
			c.repairBusy = false
			c.unlockPublish()
		})
	}, true
}

// InstalledPluginLabels returns non-empty plugin labels currently in the store snapshot.
func (c *Coordinator) InstalledPluginLabels() []string {
	if bundles, ok := c.installedBundles(context.Background(), false); ok {
		out := make([]string, 0, len(bundles))
		for _, bundle := range bundles {
			if bundle.Name != "" {
				out = append(out, bundle.Name)
			}
		}
		return out
	}
	st := c.Store()
	if st == nil {
		return nil
	}
	var out []string
	for _, row := range st.Snapshot().Value.Rows {
		if strings.TrimSpace(row.Label) != "" {
			out = append(out, row.Label)
		}
	}
	return out
}

// ListPluginsForInstallControl serves desktop install-control plugin listing queries.
func (c *Coordinator) ListPluginsForInstallControl(ctx context.Context, refresh bool, query string) []map[string]any {
	if bundles, ok := c.installedBundles(ctx, refresh); ok {
		q := strings.ToLower(strings.TrimSpace(query))
		out := make([]map[string]any, 0, len(bundles))
		for _, bundle := range bundles {
			if q != "" && !strings.Contains(strings.ToLower(bundle.Name+" "+bundle.Description), q) {
				continue
			}
			out = append(out, map[string]any{"id": bundle.Name, "name": bundle.Name, "summary": bundle.Description})
		}
		return out
	}
	st := c.Store()
	if st == nil {
		return []map[string]any{}
	}
	if refresh {
		st.Refresh(ctx)
	}
	q := strings.ToLower(strings.TrimSpace(query))
	rows := st.Snapshot().Value.Rows
	plugins := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if q != "" && !strings.Contains(strings.ToLower(row.Label+" "+row.Detail), q) {
			continue
		}
		plugins = append(plugins, map[string]any{"id": row.Label, "name": row.Label, "summary": row.Detail})
	}
	return plugins
}

func (c *Coordinator) installedBundles(ctx context.Context, refresh bool) ([]manager.Bundle, bool) {
	c.mu.Lock()
	source := c.inventorySource
	c.unlockPublish()
	if source == nil {
		return nil, false
	}
	if refresh {
		source.InvalidateInventory()
	}
	var bundles []manager.Bundle
	if err := source.Call(ctx, "pluginManager/listBundles", nil, &bundles); err != nil {
		return nil, false
	}
	return bundles, true
}

func (c *Coordinator) hostEnv() HostEnvSnapshot {
	if c != nil && c.deps != nil {
		return c.deps.Env()
	}
	return HostEnvSnapshot{}
}

func (c *Coordinator) lifecycleContext() context.Context {
	if c != nil && c.deps != nil {
		if ctx := c.deps.LifecycleContext(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

func (c *Coordinator) updateUI() {
	if c != nil && c.deps != nil {
		c.deps.UpdateWindow(func() {})
	}
}

func (c *Coordinator) invalidateUI() {
	if c != nil && c.deps != nil {
		c.deps.InvalidateWindow()
	}
}

func marketMessage(zh, en string) string {
	if i18n.Speaks("zh") {
		return zh
	}
	return en
}

func marketInstallRequestID() string {
	var raw [12]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "install-request"
	}
	return "install-" + hex.EncodeToString(raw[:])
}

func (c *Coordinator) currentAttemptLocked(op *Attempt) bool {
	return op != nil && c.active == op && op.Client == c.client && op.Store == c.store && (op.Store == nil || op.Store.RuntimeToken() == op.Generation)
}

// viewEqual compares two views field by field. Store and Client are
// compared by identity — a deep comparison would walk live objects — and
// the data slices by value.
func viewEqual(a, b ViewSnapshot) bool {
	return a.Store == b.Store && a.Open == b.Open && a.Manager == b.Manager &&
		a.InstallSpec == b.InstallSpec && a.InstallBusy == b.InstallBusy &&
		a.CanCancelInstall == b.CanCancelInstall && a.InstallMsg == b.InstallMsg &&
		a.InstallLog == b.InstallLog && a.Activation == b.Activation &&
		a.CanRestart == b.CanRestart && a.NeedsAllowBuilds == b.NeedsAllowBuilds &&
		slices.Equal(a.AllowBuilds, b.AllowBuilds) && reflect.DeepEqual(a.Incompatible, b.Incompatible) &&
		a.Owner == b.Owner && a.WorkerBusy == b.WorkerBusy && a.RepairBusy == b.RepairBusy &&
		a.env == b.env
}
