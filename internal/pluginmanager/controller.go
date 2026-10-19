package pluginmanager

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
)

type Controller struct {
	mu                                                                sync.Mutex
	client                                                            Caller
	life                                                              context.Context
	cancel                                                            context.CancelFunc
	inspectCancel                                                     context.CancelFunc
	notify                                                            func()
	state                                                             State
	read, listingRead, indexRead, installRead, registryRead, revision uint64
	pending                                                           *action
}
type action struct {
	kind, target, spec, runtime string
	enabled                     bool
	urls                        []string
	options                     InstallOptions
	owner, revision             uint64
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
	if c.cancel != nil {
		c.cancel()
	}
	c.inspectCancel = nil
	c.life, c.cancel = context.WithCancel(context.Background())
	c.client = call
	c.notify = notify
	c.state = initial(c.state.Owner+1, call != nil)
	c.read++
	c.listingRead++
	c.revision++
	c.pending = nil
	c.mu.Unlock()
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
func (c *Controller) Snapshot() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, _ := json.Marshal(c.state)
	var copy State
	_ = json.Unmarshal(raw, &copy)
	return copy
}
func (c *Controller) patch(owner uint64, fn func(*State)) bool {
	c.mu.Lock()
	if c.state.Owner != owner {
		c.mu.Unlock()
		return false
	}
	fn(&c.state)
	c.mu.Unlock()
	c.fire()
	return true
}
func (c *Controller) capture(ctx context.Context) (Client, uint64, context.Context, func(), error) {
	c.mu.Lock()
	call, owner, life := c.client, c.state.Owner, c.life
	c.mu.Unlock()
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
	if needed {
		c.state.Status = "loading"
	}
	c.mu.Unlock()
	if needed {
		go c.Load(context.Background())
	}
}
func (c *Controller) Refresh(ctx context.Context) { go c.Load(ctx) }

// Load reads the authoritative management capability before the manager RPCs.
// A failed same-owner refresh keeps cards; a new owner always starts empty.
func (c *Controller) Load(ctx context.Context) error {
	client, owner, ctx, done, err := c.capture(ctx)
	if err != nil {
		return err
	}
	defer done()
	c.mu.Lock()
	if c.state.Owner != owner {
		c.mu.Unlock()
		return context.Canceled
	}
	c.read++
	read := c.read
	unknownRequest := ""
	if c.state.Install.Phase == "unknown" {
		unknownRequest = c.state.Install.RequestID
	}
	c.state.Loading = true
	c.state.Error = ""
	c.mu.Unlock()
	c.fire()
	inventory, err := client.Inventory(ctx)
	var packages []Package
	available := inventory.ManagementAvailable != nil && *inventory.ManagementAvailable
	if err == nil && available {
		bundles, e := client.Bundles(ctx)
		err = e
		if err == nil {
			plugins, e := client.Plugins(ctx)
			err = e
			if err == nil {
				packages = JoinPackages(bundles, plugins)
			}
		}
	}
	c.mu.Lock()
	if c.state.Owner != owner || c.read != read {
		c.mu.Unlock()
		return err
	}
	c.state.Loading = false
	if err != nil {
		c.state.Status = "error"
		c.state.Error = err.Error()
	} else if !available {
		c.state.Status = "unavailable"
		c.state.Packages = nil
	} else {
		c.state.Status = "ready"
		c.state.Packages = packages
		c.revision++
		if unknownRequest != "" && c.state.Install.Phase == "unknown" && c.state.Install.RequestID == unknownRequest {
			c.state.Install.Phase = "reviewed"
			c.state.Install.Inspection = nil
			c.state.Install.Error = "Inventory refreshed. Review the installed plugins before starting a new installation."
		}
		if c.state.View.Kind == "package" || c.state.View.Kind == "row" {
			if !hasPackage(packages, c.state.View.Name) {
				c.state.View = Subject{Kind: "list"}
			}
		}
	}
	c.mu.Unlock()
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
		packages = append(packages, p)
	}
	sort.SliceStable(packages, func(i, j int) bool {
		return strings.ToLower(ShortName(packages[i].Name)) < strings.ToLower(ShortName(packages[j].Name))
	})
	return packages
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
	if a.edit(func(s *State) { s.Tab = tab; s.View = Subject{Kind: "list"} }) && tab == "discover" {
		a.c.EnsureDiscover(a.owner)
	}
}
func (a Actions) OpenPackage(name string) {
	a.edit(func(s *State) {
		if hasPackage(s.Packages, name) {
			from := s.View.Kind == "more" || s.View.FromMore
			s.Tab = "installed"
			s.View = Subject{Kind: "package", Name: name, FromMore: from}
		}
	})
}
func (c *Controller) OpenPackage(name string) {
	s := c.Snapshot()
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
func (a Actions) ShowMore() { a.edit(func(s *State) { s.View = Subject{Kind: "more"} }) }
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
	s := c.Snapshot()
	c.Actions(s.Owner).edit(func(s *State) { s.View = Subject{Kind: "list"}; s.Discover.SelectedEntry = "" })
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
	if c.state.Owner != owner || !c.state.Connected || c.state.Busy || c.state.Status != "ready" {
		c.mu.Unlock()
		return
	}
	op.owner, op.revision = owner, c.revision
	valid := false
	for _, p := range c.state.Packages {
		if p.Name == op.target && op.kind != "row" {
			valid = p.ReadOnlyReason == ""
			if op.kind == "remove" {
				valid = valid && p.Removable
			}
			if op.kind == "bundle" && p.Availability == "missing" && op.enabled && p.InstallTarget != nil {
				target := *p.InstallTarget
				c.mu.Unlock()
				c.openInstall(owner, target.Spec, "", &op.enabled, true)
				return
			}
		}
		if op.kind == "row" {
			for _, r := range p.Parts {
				if r.EntryID == op.target && r.EntryID != "" {
					valid = r.ReadOnlyReason == ""
				}
			}
		}
	}
	if !valid {
		c.mu.Unlock()
		return
	}
	if confirm {
		c.pending = &op
		c.state.Confirmation = &Confirmation{Kind: op.kind, Target: op.target, Description: "Remove plugin " + op.target + "?"}
		c.mu.Unlock()
		c.fire()
		return
	}
	c.mu.Unlock()
	c.run(op)
}
func (a Actions) CancelConfirmation() {
	a.c.mu.Lock()
	if a.c.state.Owner == a.owner {
		a.c.pending = nil
		a.c.state.Confirmation = nil
	}
	a.c.mu.Unlock()
	a.c.fire()
}
func (a Actions) Confirm() {
	c := a.c
	c.mu.Lock()
	op := c.pending
	if c.state.Owner != a.owner || op == nil {
		c.mu.Unlock()
		return
	}
	c.pending = nil
	c.state.Confirmation = nil
	if op.owner != a.owner || op.revision != c.revision {
		c.state.Notice = "The plugin inventory changed. Refresh and review the action again."
		c.mu.Unlock()
		c.fire()
		return
	}
	c.mu.Unlock()
	c.run(*op)
}
func (c *Controller) run(op action) {
	c.mu.Lock()
	if c.state.Owner != op.owner || c.client == nil || c.state.Busy {
		c.mu.Unlock()
		return
	}
	client, ctx := Client{c.client}, c.life
	c.state.Busy = true
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
	c.mu.Unlock()
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
			result, err = client.Install(ctx, op.spec, op.options)
		case "exemption":
			result, err = client.Exempt(ctx, op.target, op.runtime)
		}
		if !c.patch(op.owner, func(s *State) {
			s.Busy = false
			if op.kind == "install" && (s.Install.RequestID != op.options.RequestID || !installPending(s.Install.Phase)) {
				return
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
				s.Install.Recovering = false
				s.Install.Result = &result
				s.Install.Phase = "done"
				if result.Application == "failed" || result.Application == "cancelled" {
					s.Install.Phase = result.Application
				}
				s.Install.Error = ""
			}
		}) {
			return
		}
		if err == nil {
			_ = c.Load(ctx)
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
