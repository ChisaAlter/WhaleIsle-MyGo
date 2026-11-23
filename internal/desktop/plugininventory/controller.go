// Package plugininventory projects the Host's read-only Loader/preset inventory.
// Bundle installation and configuration writes remain in their existing owners.
package plugininventory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"whaleisle/internal/dsh/api"
	"whaleisle/internal/state"
)

type Caller interface {
	Call(context.Context, string, any, any) error
}

type Row struct {
	Key, EntryID, Module, Enabled, Phase, Condition, MetadataError string
	Title, Description                                             json.RawMessage
}
type Preset struct {
	ID, Name, Broken string
	Default          bool
	Rows             []Row
}
type Snapshot struct {
	Host                               uint64
	Revision                           uint64
	Connected, Ready, Busy, GlobalOpen bool
	Query, SelectedID, Expanded        string
	Err                                *state.Fault
	Global                             []Row
	Presets                            []Preset
}

// Controller keeps the working Snapshot under mu and publishes an immutable
// version on every change, so readers share one value per revision instead
// of cloning the rows on each read. Row slices come from normalize and are
// replaced wholesale on Accept/Refresh, never edited in place. The Host
// generation is a state.Lease renewed on Attach; its context cancels the
// retired Host's in-flight read.
type Controller struct {
	mu               sync.Mutex
	call             Caller
	host             state.LeaseOwner
	hostL            state.Lease
	state            Snapshot
	pub              state.Published[Snapshot]
	acceptedRevision uint64
}

func (c *Controller) publishLocked() { c.pub.Publish(c.state) }

// Attach invalidates both in-flight reads and UI callbacks from the prior Host.
func (c *Controller) Attach(call Caller) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hostL = c.host.Renew(context.Background())
	c.call = call
	c.state = Snapshot{Host: c.hostL.Generation(), Connected: call != nil}
	c.acceptedRevision = 0
	c.publishLocked()
}

// Snapshot returns the current published version. The same *Version is
// shared by every reader until a change publishes the next revision; its
// Value and the rows it references are immutable.
func (c *Controller) Snapshot() *state.Version[Snapshot] {
	if v := c.pub.Load(); v != nil {
		return v
	}
	c.mu.Lock()
	if c.pub.Load() == nil {
		c.publishLocked()
	}
	c.mu.Unlock()
	return c.pub.Load()
}

// Changed closes when the next version is published.
func (c *Controller) Changed() <-chan struct{} { return c.pub.Changed() }

// Accept projects the completed shared Host read without starting another RPC.
// The captured Host owner rejects a publication from a retired attachment.
func (c *Controller) Accept(host, revision uint64, wire api.PluginInventorySnapshot) {
	global, presets := normalize(wire)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Host != host || !c.state.Connected || c.acceptedRevision >= revision {
		return
	}
	c.state.Global, c.state.Presets = global, presets
	c.acceptedRevision = revision
	c.state.Revision++
	c.state.Ready = true
	c.state.Busy = false
	c.state.Err = nil
	c.state.SelectedID = selectedPreset(c.state.Presets, c.state.SelectedID)
	c.publishLocked()
}
func (c *Controller) SetQuery(host uint64, query string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Host == host && c.state.Query != query {
		c.state.Query = query
		c.publishLocked()
	}
}
func (c *Controller) SelectPreset(host uint64, id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Host != host {
		return
	}
	for _, p := range c.state.Presets {
		if p.ID == id {
			c.state.SelectedID = id
			c.state.Expanded = ""
			c.publishLocked()
			return
		}
	}
}
func (c *Controller) ToggleExpanded(host uint64, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Host != host {
		return
	}
	if c.state.Expanded == key {
		c.state.Expanded = ""
	} else {
		c.state.Expanded = key
	}
	c.publishLocked()
}
func (c *Controller) ToggleGlobal(host uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Host == host {
		c.state.GlobalOpen = !c.state.GlobalOpen
		c.publishLocked()
	}
}

// Refresh only reads pluginInventory/list. A same-Host failure preserves the
// previous snapshot with an explicit error, while Attach always clears it.
// The RPC and the normalize pass run without the lock; a completion whose
// Host lease was renewed meanwhile is dropped.
func (c *Controller) Refresh(parent context.Context) {
	c.mu.Lock()
	if c.call == nil || c.state.Busy {
		c.mu.Unlock()
		return
	}
	call, lease := c.call, c.hostL
	ctx, cancel := context.WithCancel(lease.Context())
	stop := context.AfterFunc(parent, cancel)
	c.state.Busy, c.state.Err = true, nil
	c.publishLocked()
	c.mu.Unlock()
	defer func() { stop(); cancel() }()
	var wire api.PluginInventorySnapshot
	err := call.Call(ctx, api.PluginInventoryListMethod, nil, &wire)
	if err == nil {
		err = ctx.Err()
	}
	var global []Row
	var presets []Preset
	if err == nil {
		global, presets = normalize(wire)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !lease.Valid() {
		return
	}
	c.state.Busy = false
	if err != nil {
		c.state.Err = state.FaultOf(err)
		c.publishLocked()
		return
	}
	c.state.Global, c.state.Presets = global, presets
	c.state.Revision++
	c.state.Ready = true
	// Refresh keeps filter, chosen preset and expanded row; selection falls back
	// only when that preset is no longer present in the authoritative inventory.
	c.state.SelectedID = selectedPreset(c.state.Presets, c.state.SelectedID)
	c.publishLocked()
}

func withMeta(row Row, meta *api.ListBundlesResultItemMeta) Row {
	if meta != nil {
		row.Title = append(json.RawMessage(nil), meta.Title...)
		row.Description = append(json.RawMessage(nil), meta.Description...)
		if meta.Error != nil {
			row.MetadataError = *meta.Error
		}
	}
	return row
}
func normalize(wire api.PluginInventorySnapshot) ([]Row, []Preset) {
	global := make([]Row, 0, len(wire.Entries))
	for i, r := range wire.Entries {
		enabled := "disabled"
		if r.Enabled {
			enabled = "enabled"
		}
		global = append(global, withMeta(Row{Key: fmt.Sprintf("global:%d:%s", i, r.EntryId), EntryID: r.EntryId, Module: r.ModuleName, Enabled: enabled, Phase: r.FiberPhase}, r.Meta))
	}
	presets := make([]Preset, 0, len(wire.AgentPresets))
	for _, p := range wire.AgentPresets {
		preset := Preset{ID: p.Id, Default: p.IsDefault}
		if p.Name != nil {
			preset.Name = *p.Name
		}
		if p.Broken != nil {
			preset.Broken = *p.Broken
		}
		for i, r := range p.Rows {
			enabled := "unknown"
			var value any
			if json.Unmarshal(r.Enabled, &value) == nil {
				switch value {
				case true:
					enabled = "enabled"
				case false:
					enabled = "disabled"
				case "conditional":
					enabled = "conditional"
				}
			}
			row := Row{Key: fmt.Sprintf("preset:%s:%d:%s", p.Id, i, r.EntryId), EntryID: r.EntryId, Module: r.ModuleName, Enabled: enabled, Phase: r.FiberPhase}
			if r.Condition != nil {
				row.Condition = *r.Condition
			}
			preset.Rows = append(preset.Rows, withMeta(row, r.Meta))
		}
		presets = append(presets, preset)
	}
	return global, presets
}
