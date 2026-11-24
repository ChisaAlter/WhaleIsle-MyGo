package pluginmanager

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/state"
)

type Controller struct {
	mu                                                                sync.Mutex
	client                                                            Caller
	life                                                              context.Context
	owners                                                            state.LeaseOwner
	ownerL                                                            state.Lease
	pub                                                               state.Published[State]
	inspectCancel                                                     context.CancelFunc
	notify                                                            func()
	state                                                             State
	read, listingRead, indexRead, installRead, registryRead, revision uint64
	inventoryRevision                                                 uint64
	inventoryPlugins                                                  []Plugin
	inventorySourceVersion                                            uint64
	pending                                                           *action
	selectAdded                                                       bool
	failedDiscover                                                    *discoverRead
	discoverTextTimer                                                 *time.Timer
	textRead                                                          uint64
	choiceRead                                                        uint64
	registryMemory                                                    *RegistryChoice
	registrySave                                                      func(*RegistryChoice) error
	nativeInstall                                                     *action
	nativeCancel                                                      context.CancelFunc
}
type action struct {
	kind, target, spec, runtime string
	enabled                     bool
	urls                        []string
	options                     InstallOptions
	owner, revision             uint64
	capabilityKind              CapabilityKind
	mcpSpecs                    []json.RawMessage
	localSkills                 []LocalSkillManifest
	progress                    *nativeProgress
}

func initial(owner uint64, connected bool) State {
	status := "disconnected"
	if connected {
		status = "idle"
	}
	return State{Owner: owner, Connected: connected, Status: status, Tab: "installed", View: Subject{Kind: "list"}, Discover: DiscoverState{Status: "idle", Query: Query{Sort: MarketSort{Kind: "index"}, Limit: 50}}}
}
func New() *Controller { return &Controller{state: initial(0, false)} }
func (c *Controller) Attach(call Caller, notify func()) {
	c.mu.Lock()
	c.inspectCancel = nil
	if c.discoverTextTimer != nil {
		c.discoverTextTimer.Stop()
		c.discoverTextTimer = nil
	}
	c.textRead++
	// The owner lease replaces the owner counter and its lifecycle context:
	// renewing it cancels the retired attachment's work, and State.Owner is
	// the lease generation the UI hands back with every action.
	c.ownerL = c.owners.Renew(context.Background())
	c.life = c.ownerL.Context()
	c.client = call
	c.notify = notify
	preferenceError := c.state.PreferenceError
	c.state = initial(c.ownerL.Generation(), call != nil)
	c.state.PreferenceError = preferenceError
	c.read++
	c.listingRead++
	c.revision++
	c.inventoryRevision = 0
	c.inventoryPlugins = nil
	c.inventorySourceVersion = 0
	c.pending = nil
	c.nativeInstall = nil
	c.nativeCancel = nil
	c.selectAdded = false
	c.failedDiscover = nil
	c.unlockPublish()
	c.fire()
}
func (c *Controller) Detach()                 { c.mu.Lock(); notify := c.notify; c.mu.Unlock(); c.Attach(nil, notify) }
func (c *Controller) SetNotify(notify func()) { c.mu.Lock(); c.notify = notify; c.mu.Unlock() }
func (c *Controller) fire() {
	c.mu.Lock()
	fn := c.notify
	c.mu.Unlock()
	if fn != nil {
		fn()
	}
}
func (c *Controller) patch(owner uint64, fn func(*State)) bool {
	c.mu.Lock()
	if c.state.Owner != owner {
		c.unlockPublish()
		return false
	}
	fn(&c.state)
	c.unlockPublish()
	c.fire()
	return true
}
func (c *Controller) capture(ctx context.Context) (Client, uint64, context.Context, func(), error) {
	c.mu.Lock()
	call, owner, life := c.client, c.state.Owner, c.life
	c.unlockPublish()
	if call == nil {
		return Client{}, owner, nil, func() {}, errors.New("Plugin manager is disconnected")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	linked, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(life, cancel)
	return Client{call}, owner, linked, func() { stop(); cancel() }, nil
}
func (c *Controller) Ensure() {
	c.mu.Lock()
	needed := c.state.Status == "idle"
	if source, ok := c.client.(interface{ InventoryVersion() uint64 }); ok && c.inventorySourceVersion != source.InventoryVersion() && c.state.Status == "ready" {
		needed = true
	}
	needed = needed && !c.state.Loading
	if needed {
		c.state.Status = "loading"
		c.state.Loading = true
	}
	c.unlockPublish()
	if needed {
		go c.Load(context.Background())
	}
}
func (c *Controller) Refresh(ctx context.Context) {
	c.mu.Lock()
	caller := c.client
	c.unlockPublish()
	if source, ok := caller.(interface{ InvalidateInventory() }); ok {
		source.InvalidateInventory()
	}
	go c.Load(ctx)
}

// Load reads the authoritative management capability before the manager RPCs.
// A failed same-owner refresh keeps cards; a new owner always starts empty.
func (c *Controller) Load(ctx context.Context) error {
	client, owner, ctx, done, err := c.capture(ctx)
	if err != nil {
		// Ensure already moved the status to loading. A capture failure (Host
		// still booting, no owner yet) must not leave the page on an endless
		// skeleton; surface the error and its retry instead.
		c.mu.Lock()
		c.state.Loading = false
		c.state.Status = "error"
		c.state.Error = err.Error()
		c.unlockPublish()
		c.fire()
		return err
	}
	defer done()
	c.mu.Lock()
	if c.state.Owner != owner {
		c.unlockPublish()
		return context.Canceled
	}
	c.read++
	read := c.read
	c.state.Loading = true
	c.state.Error = ""
	c.unlockPublish()
	c.fire()
	var sourceVersion uint64
	if source, ok := client.Caller.(interface{ InventoryVersion() uint64 }); ok {
		sourceVersion = source.InventoryVersion()
	}
	inventory, err := client.Inventory(ctx)
	var packages []Package
	var plugins []Plugin
	available := inventory.ManagementAvailable != nil && *inventory.ManagementAvailable
	if err == nil && available {
		bundles, e := client.Bundles(ctx)
		err = e
		if err == nil {
			plugins, err = client.Plugins(ctx)
			if err == nil {
				packages = JoinPackages(bundles, plugins)
			}
		}
	}
	c.mu.Lock()
	if c.state.Owner != owner || c.read != read {
		c.unlockPublish()
		return err
	}
	if source, ok := client.Caller.(interface{ InventoryVersion() uint64 }); ok && err == nil && sourceVersion != source.InventoryVersion() {
		c.state.Loading = false
		c.unlockPublish()
		return c.Load(ctx)
	}
	c.state.Loading = false
	if err != nil {
		c.state.Status = "error"
		c.state.Error = err.Error()
	} else if !available {
		c.state.Status = "unavailable"
		c.state.Packages = nil
		c.inventoryPlugins = nil
	} else {
		c.state.Status = "ready"
		c.state.Packages = packages
		c.inventoryPlugins = plugins
		c.inventorySourceVersion = sourceVersion
		c.revision++
		c.inventoryRevision++
		if c.state.View.Kind == "package" || c.state.View.Kind == "row" {
			if !hasPackage(packages, c.state.View.Name) {
				c.state.View = Subject{Kind: "list"}
			}
		}
	}
	c.unlockPublish()
	c.fire()
	return err
}
func JoinPackages(bundles []Bundle, plugins []Plugin) []Package {
	packages := make([]Package, 0, len(bundles))
	for _, bundle := range bundles {
		p := Package{Bundle: bundle}
		for _, row := range bundle.Rows {
			r := Row{BundleRow: row}
			if row.EntryID != "" {
				for _, live := range plugins {
					if live.EntryID == row.EntryID {
						r.Enabled, r.Phase, r.ReadOnlyReason = live.Enabled, live.Phase, live.ReadOnlyReason
						break
					}
				}
			}
			p.Parts = append(p.Parts, r)
		}
		caps := classifyInstalledBundle(bundle)
		p.CapabilityKind = caps.Kind
		p.NativeCompatible = caps.NativeCompatible
		p.CompatibilityLabel = caps.CompatibilityLabel
		packages = append(packages, p)
	}
	sort.SliceStable(packages, func(i, j int) bool {
		return strings.ToLower(ShortName(packages[i].Name)) < strings.ToLower(ShortName(packages[j].Name))
	})
	return packages
}

func classifyInstalledBundle(bundle Bundle) PackageCapabilities {
	if strings.HasPrefix(bundle.Source, "file:") {
		if dir := fileSpecToDir(bundle.Source); dir != "" {
			if caps, err := InspectLocalPackage(dir); err == nil && caps.Kind != CapabilityUnknown {
				return caps
			}
		}
	}
	hasClient, hasHost := false, false
	for _, row := range bundle.Rows {
		lower := strings.ToLower(row.ModuleName)
		if strings.Contains(lower, "client-ui-") || strings.HasSuffix(lower, "/client") || strings.Contains(lower, "-client-") {
			hasClient = true
		} else {
			hasHost = true
		}
	}
	var caps PackageCapabilities
	switch {
	case hasHost && hasClient:
		caps.HasHostContribution, caps.HasWebClient = true, true
		finalizeCapabilities(&caps)
		return caps
	case hasClient:
		caps.HasWebClient = true
		finalizeCapabilities(&caps)
		return caps
	case hasHost:
		caps.HasHostContribution = true
		finalizeCapabilities(&caps)
		return caps
	default:
		return InferCapabilityFromCatalogMetadata("", bundle.Name, bundle.Name, bundle.Description)
	}
}

func fileSpecToDir(spec string) string {
	if strings.HasPrefix(spec, "file://") {
		if u, err := url.Parse(spec); err == nil && u.Path != "" {
			p := filepath.FromSlash(u.Path)
			if len(p) >= 3 && p[0] == '\\' && p[2] == ':' {
				p = p[1:]
			}
			return p
		}
	}
	if strings.HasPrefix(spec, "file:") {
		raw := strings.TrimSpace(strings.TrimPrefix(spec, "file:"))
		if raw != "" {
			return filepath.FromSlash(raw)
		}
	}
	if spec != "" {
		if info, err := os.Stat(spec); err == nil && info.IsDir() {
			return spec
		}
	}
	return ""
}
func hasPackage(packages []Package, name string) bool {
	for _, p := range packages {
		if p.Name == name {
			return true
		}
	}
	return false
}

// Actions captures a rendered owner. Every action checks it inside the mutex;
// old controls and dialogs can never silently target a replacement Host.
type Actions struct {
	c     *Controller
	owner uint64
}

func (c *Controller) Actions(owner uint64) Actions { return Actions{c, owner} }
func (a Actions) edit(fn func(*State)) bool        { return a.c.patch(a.owner, fn) }
func (a Actions) SelectTab(tab string) {
	if tab != "installed" && tab != "discover" {
		return
	}
	if a.edit(func(s *State) {
		s.Tab = tab
		if tab != "discover" {
			a.c.cancelDiscoverText()
			s.Discover.TextDraft = s.Discover.Query.Text
			s.Discover.SelectedEntry = ""
			s.Discover.Opened = nil
			s.Discover.Choice = nil
		}
	}) && tab == "discover" {
		a.c.EnsureDiscover(a.owner)
	}
}
func (a Actions) OpenPackage(name string) {
	a.edit(func(s *State) {
		if hasPackage(s.Packages, name) {
			a.c.cancelDiscoverText()
			s.Discover.TextDraft = s.Discover.Query.Text
			s.Discover.SelectedEntry = ""
			s.Discover.Opened = nil
			s.Discover.Choice = nil
			if s.View.Name != name {
				s.RowFilter = ""
			}
			from := s.View.Kind == "more" || s.View.FromMore
			s.Tab = "installed"
			s.View = Subject{Kind: "package", Name: name, FromMore: from}
		}
	})
}
func (c *Controller) OpenPackage(name string) {
	s := c.Snapshot().Value
	c.Actions(s.Owner).OpenPackage(name)
}
func (a Actions) OpenRow(name, id string) {
	a.edit(func(s *State) {
		for _, p := range s.Packages {
			if p.Name == name {
				for _, r := range p.Parts {
					if r.RowID == id {
						s.View = Subject{Kind: "row", Name: name, RowID: id, FromMore: s.View.FromMore}
						return
					}
				}
			}
		}
	})
}
func (a Actions) OpenItem(id string) {
	a.edit(func(s *State) { s.View = Subject{Kind: "item", ItemID: id} })
}
func (a Actions) ShowMore()               { a.edit(func(s *State) { s.View = Subject{Kind: "more"} }) }
func (a Actions) FilterRows(value string) { a.edit(func(s *State) { s.RowFilter = value }) }
func (a Actions) Back() {
	a.edit(func(s *State) {
		switch s.View.Kind {
		case "row":
			s.View = Subject{Kind: "package", Name: s.View.Name, FromMore: s.View.FromMore}
		case "package":
			if s.View.FromMore {
				s.View = Subject{Kind: "more"}
			} else {
				s.View = Subject{Kind: "list"}
			}
		default:
			s.View = Subject{Kind: "list"}
		}
	})
}
func (c *Controller) Leave() {
	s := c.Snapshot().Value
	c.Actions(s.Owner).edit(func(s *State) {
		c.cancelDiscoverText()
		s.View = Subject{Kind: "list"}
		s.Discover.SelectedEntry = ""
		s.Discover.Opened = nil
		s.Discover.Choice = nil
		s.Discover.TextDraft = s.Discover.Query.Text
	})
}
func (a Actions) DismissNotice() { a.edit(func(s *State) { s.Notice = "" }) }

func (a Actions) Enable(name string, enabled bool) {
	a.c.prepare(a.owner, action{kind: "bundle", target: name, enabled: enabled}, false)
}
func (a Actions) EnableRow(id string, enabled bool) {
	a.c.prepare(a.owner, action{kind: "row", target: id, enabled: enabled}, false)
}
func (a Actions) Remove(name string) {
	a.c.prepare(a.owner, action{kind: "remove", target: name}, true)
}
func (c *Controller) prepare(owner uint64, op action, confirm bool) {
	c.mu.Lock()
	if c.state.Owner != owner || !c.state.Connected || c.state.ActionBusy(op.target) || !c.state.ManagementReady() {
		c.unlockPublish()
		return
	}
	op.owner, op.revision = owner, c.revision
	valid := false
	for _, p := range c.state.Packages {
		if p.Name == op.target && op.kind != "row" {
			valid = p.ReadOnlyReason == ""
			if op.kind == "bundle" {
				valid = !p.SwitchLocked()
			}
			if op.kind == "remove" {
				valid = valid && p.Removable
			}
			if op.kind == "bundle" && p.Availability == "missing" && op.enabled && p.InstallTarget != nil {
				c.unlockPublish()
				c.openCatalogInstall(owner, p, op.enabled)
				return
			}
		}
		if op.kind == "row" {
			if !p.Enabled || c.state.ActionBusy(p.Name) {
				continue
			}
			for _, r := range p.Parts {
				if r.EntryID == op.target && r.EntryID != "" {
					valid = r.ReadOnlyReason == ""
				}
			}
		}
	}
	if !valid {
		c.unlockPublish()
		return
	}
	if confirm {
		c.pending = &op
		c.state.Confirmation = &Confirmation{Kind: op.kind, Target: op.target, Description: "Remove plugin " + op.target + "?"}
		c.unlockPublish()
		c.fire()
		return
	}
	c.unlockPublish()
	c.run(op)
}
func (a Actions) CancelConfirmation() {
	a.c.mu.Lock()
	if a.c.state.Owner == a.owner {
		if op := a.c.pending; op != nil && op.kind == "install" && a.c.state.Install.Phase == "idle" && a.c.state.Install.RequestID == op.options.RequestID {
			a.c.state.Install.RequestID = ""
		}
		a.c.pending = nil
		a.c.state.Confirmation = nil
	}
	a.c.unlockPublish()
	a.c.fire()
}
func (a Actions) Confirm() {
	c := a.c
	c.mu.Lock()
	op := c.pending
	if c.state.Owner != a.owner || op == nil {
		c.unlockPublish()
		return
	}
	c.pending = nil
	c.state.Confirmation = nil
	if op.owner != a.owner || op.revision != c.revision {
		c.state.Notice = "The plugin inventory changed. Refresh and review the action again."
		c.unlockPublish()
		c.fire()
		return
	}
	c.unlockPublish()
	c.run(*op)
}
func (c *Controller) run(op action) {
	c.mu.Lock()
	if c.state.Owner != op.owner || c.client == nil || op.kind == "install" && (c.state.Busy || c.state.Install.RequestID != op.options.RequestID) || op.kind != "install" && c.state.ActionBusy(op.target) {
		c.unlockPublish()
		return
	}
	client, ctx := Client{c.client}, c.life
	refreshContext := c.life
	if nativeAction(op) {
		if op.progress == nil {
			op.progress = newNativeProgress(op)
		}
		op.progress.setRunning(true)
		ctx, c.nativeCancel = context.WithCancel(c.life)
		copy := op
		c.nativeInstall = &copy
	}
	if op.kind == "install" {
		c.state.Busy = true
	} else {
		if c.state.BusyKeys == nil {
			c.state.BusyKeys = map[string]bool{}
		}
		c.state.BusyKeys[op.target] = true
	}
	c.state.Notice = ""
	if op.kind == "install" {
		c.state.Install.Phase = "starting"
		c.state.Install.Result = nil
		c.state.Install.Recovering = false
		c.state.Install.Acknowledged = false
		c.state.Install.CancellationRequested = false
		c.state.Install.WaitingForStart = false
		c.state.Install.Runs = nil
		c.state.Install.Attempts = nil
		c.state.Install.Log = ""
		c.state.Install.Error = ""
	}
	c.unlockPublish()
	c.fire()
	go func() {
		var result Change
		var err error
		switch op.kind {
		case "bundle":
			result, err = client.SetBundleEnabled(ctx, op.target, op.enabled)
		case "row":
			result, err = client.SetRowEnabled(ctx, op.target, op.enabled)
		case "remove":
			result, err = client.Remove(ctx, op.target)
		case "install":
			if nativeAction(op) {
				result, err = client.applyNative(ctx, op)
				op.progress.setRunning(false)
			} else {
				result, err = client.Install(ctx, op.spec, op.options)
			}
		case "exemption":
			result, err = client.Exempt(ctx, op.target, op.runtime)
		}
		if !c.patch(op.owner, func(s *State) {
			if op.kind == "install" && (s.Install.RequestID != op.options.RequestID || !installPending(s.Install.Phase)) {
				return
			}
			if op.kind == "install" {
				s.Busy = false
				if nativeAction(op) {
					s.Install.NativeCanFinish = true
				}
				if nativeAction(op) && c.nativeCancel != nil {
					c.nativeCancel()
					c.nativeCancel = nil
				}
			} else {
				delete(s.BusyKeys, op.target)
			}
			if err != nil {
				s.Notice = err.Error()
				if op.kind == "install" {
					if s.Install.Phase != "applying" {
						s.Install.Phase = "unconfirmed"
					}
					s.Install.Error = "Installation result is unconfirmed: " + err.Error()
				}
				return
			}
			s.Notice = changeMessage(result)
			if op.kind == "install" {
				settleInstallState(s, result)
			}
		}) {
			return
		}
		if err == nil || op.kind != "install" {
			_ = c.Load(refreshContext)
			c.BundlesChanged(op.owner)
		} else if op.kind == "install" {
			c.reconcileInstall(op.owner, op.options.RequestID)
		}
	}()
}
func changeMessage(result Change) string {
	message := result.Application
	if result.Application == "restart-required" {
		message = "Restart required: " + result.RestartReason
	}
	if result.Error != nil {
		message += " · " + result.Error.Code
		if result.Error.Diagnostic != "" {
			message += " · " + result.Error.Diagnostic
		}
	}
	if len(result.Warnings) > 0 {
		message += " · " + strings.Join(result.Warnings, "; ")
	}
	return message
}
