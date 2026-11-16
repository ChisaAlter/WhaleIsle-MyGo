package marketctl

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	manager "whaleisle/internal/pluginmanager"
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

// ViewSnapshot is a value copy describing all UI-relevant market coordinator state.
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

	nativePlugins   nativePluginBinding
	pluginInventory plugininventory.Controller
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
	c.mu.Unlock()
	return st
}

// SetStore replaces the coordinator's market store (primarily for tests).
func (c *Coordinator) SetStore(st *marketstore.Store) {
	c.mu.Lock()
	c.store = st
	c.mu.Unlock()
}

// SetClient sets the coordinator's active Host transport without touching shell mutexes.
func (c *Coordinator) SetClient(client *dsh.Client) {
	c.mu.Lock()
	c.client = client
	c.mu.Unlock()
}

// AttachHost binds a connected DSH client and managed profile directory to the
// market store, plugin inventory, and native plugin manager.
func (c *Coordinator) AttachHost(client *dsh.Client, managedProfileDir string, onRefresh func()) {
	store := c.EnsureStore()
	c.mu.Lock()
	c.client = client
	c.mu.Unlock()
	store.SetUpdateProfileDir(managedProfileDir)
	store.Attach(client, onRefresh)
	c.AttachInventory(client)
	c.BindNativePluginManager(client)
}

// DetachHost invalidates in-flight attempts and disconnects Host transports from
// marketplace stores and controllers.
func (c *Coordinator) DetachHost() {
	c.Invalidate()
	c.mu.Lock()
	c.client = nil
	store := c.store
	c.mu.Unlock()
	if store != nil {
		store.SetUpdateProfileDir("")
		store.Detach()
	}
	c.AttachInventory(nil)
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
	c.mu.Unlock()
}

// SetManagerMode selects native plugin manager vs discovery market view.
func (c *Coordinator) SetManagerMode(managerMode bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.managerMode = managerMode
	c.mu.Unlock()
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
	c.mu.Unlock()
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
	c.mu.Unlock()
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

// ViewSnapshot returns a consistent point-in-time snapshot for UI rendering.
func (c *Coordinator) ViewSnapshot() ViewSnapshot {
	env := c.hostEnv()
	c.mu.Lock()
	defer c.mu.Unlock()
	var gen uint64
	if c.store != nil {
		gen = c.store.RuntimeToken()
	}
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
		CanRestart:       env.HasSupervisor && c.worker == nil && !env.TaskRestartPending && !env.HostStarting,
		NeedsAllowBuilds: c.needsAllowBuilds,
		AllowBuilds:      append([]string(nil), c.allowBuilds...),
		Incompatible:     append([]api.ChangeResultErrorIncompatibleItem(nil), c.incompatible...),
		Owner:            Owner{Client: c.client, Store: c.store, Generation: gen},
		WorkerBusy:       c.worker != nil,
		RepairBusy:       c.repairBusy,
	}
}

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
		c.mu.Unlock()
		return nil, false
	}
	c.installBusy = true
	c.repairBusy = true
	c.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			c.mu.Lock()
			c.installBusy = false
			c.repairBusy = false
			c.mu.Unlock()
		})
	}, true
}

// InstalledPluginLabels returns non-empty plugin labels currently in the store snapshot.
func (c *Coordinator) InstalledPluginLabels() []string {
	st := c.Store()
	if st == nil {
		return nil
	}
	var out []string
	for _, row := range st.Snapshot().Rows {
		if strings.TrimSpace(row.Label) != "" {
			out = append(out, row.Label)
		}
	}
	return out
}

// ListPluginsForInstallControl serves desktop install-control plugin listing queries.
func (c *Coordinator) ListPluginsForInstallControl(ctx context.Context, refresh bool, query string) []map[string]any {
	st := c.Store()
	if st == nil {
		return []map[string]any{}
	}
	if refresh {
		st.Refresh(ctx)
	}
	q := strings.ToLower(strings.TrimSpace(query))
	rows := st.Snapshot().Rows
	plugins := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if q != "" && !strings.Contains(strings.ToLower(row.Label+" "+row.Detail), q) {
			continue
		}
		plugins = append(plugins, map[string]any{"id": row.Label, "name": row.Label, "summary": row.Detail})
	}
	return plugins
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
	if strings.HasPrefix(i18n.Locale(), "zh") {
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
