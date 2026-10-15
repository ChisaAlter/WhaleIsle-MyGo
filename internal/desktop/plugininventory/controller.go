// Package plugininventory projects the Host's read-only Loader/preset inventory.
// Bundle installation and configuration writes remain in their existing owners.
package plugininventory

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"whaleisle/internal/dsh/api"
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
	Connected, Ready, Busy, GlobalOpen bool
	Query, SelectedID, Expanded, Err   string
	Global                             []Row
	Presets                            []Preset
}
type Controller struct {
	mu          sync.Mutex
	call        Caller
	hostContext context.Context
	cancel      context.CancelFunc
	state       Snapshot
}

// Attach invalidates both in-flight reads and UI callbacks from the prior Host.
func (c *Controller) Attach(call Caller) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		c.cancel()
	}
	c.hostContext, c.cancel = context.WithCancel(context.Background())
	c.call = call
	c.state = Snapshot{Host: c.state.Host + 1, Connected: call != nil}
}
func cloneRows(rows []Row) []Row {
	out := append([]Row(nil), rows...)
	for i := range out {
		out[i].Title = append(json.RawMessage(nil), out[i].Title...)
		out[i].Description = append(json.RawMessage(nil), out[i].Description...)
	}
	return out
}
func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.state
	s.Global = cloneRows(s.Global)
	s.Presets = append([]Preset(nil), s.Presets...)
	for i := range s.Presets {
		s.Presets[i].Rows = cloneRows(s.Presets[i].Rows)
	}
	return s
}
func (c *Controller) SetQuery(host uint64, query string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Host == host {
		c.state.Query = query
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
}
func (c *Controller) ToggleGlobal(host uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Host == host {
		c.state.GlobalOpen = !c.state.GlobalOpen
	}
}

// Refresh only reads pluginInventory/list. A same-Host failure preserves the
// previous snapshot with an explicit error, while Attach always clears it.
func (c *Controller) Refresh(parent context.Context) {
	c.mu.Lock()
	if c.call == nil || c.state.Busy {
		c.mu.Unlock()
		return
	}
	call, host := c.call, c.state.Host
	ctx, cancel := context.WithCancel(c.hostContext)
	stop := context.AfterFunc(parent, cancel)
	c.state.Busy, c.state.Err = true, ""
	c.mu.Unlock()
	defer func() { stop(); cancel() }()
	var wire api.PluginInventorySnapshot
	err := call.Call(ctx, api.PluginInventoryListMethod, nil, &wire)
	if err == nil {
		err = ctx.Err()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state.Host != host {
		return
	}
	c.state.Busy = false
	if err != nil {
		c.state.Err = err.Error()
		return
	}
	c.state.Global, c.state.Presets = normalize(wire)
	c.state.Ready = true
	// Refresh keeps filter, chosen preset and expanded row; selection falls back
	// only when that preset is no longer present in the authoritative inventory.
	c.state.SelectedID = selectedPreset(c.state.Presets, c.state.SelectedID)
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
