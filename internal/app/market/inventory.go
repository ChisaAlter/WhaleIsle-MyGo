package marketctl

import (
	"context"
	"time"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/startuptrace"
)

type notifyingInventoryCaller struct {
	plugininventory.Caller
	notify func()
}

func (n notifyingInventoryCaller) Call(ctx context.Context, method string, args, out any) error {
	n.notify()
	return n.Caller.Call(ctx, method, args, out)
}

// Inventory returns the underlying plugin inventory controller.
func (c *Coordinator) Inventory() *plugininventory.Controller {
	return &c.pluginInventory
}

// AttachInventory binds client (or nil to detach) to the plugin inventory controller.
func (c *Coordinator) AttachInventory(client *dsh.Client) {
	var caller plugininventory.Caller
	if client != nil {
		c.mu.Lock()
		source := c.inventorySource
		c.mu.Unlock()
		if source != nil {
			caller = source
		} else {
			caller = client
		}
		caller = notifyingInventoryCaller{Caller: caller, notify: c.invalidateUI}
	}
	c.pluginInventory.Attach(caller)
}

// RefreshInventory refreshes plugin inventory asynchronously and invalidates the UI.
func (c *Coordinator) RefreshInventory() {
	c.mu.Lock()
	source := c.inventorySource
	c.mu.Unlock()
	if source != nil {
		source.InvalidateInventory()
	}
	c.ensureInventory()
}

func (c *Coordinator) acceptInventory(source *manager.HostInventory, host uint64, method string) {
	c.mu.Lock()
	current := c.inventorySource == source
	ctrl := c.nativePlugins.controller
	c.mu.Unlock()
	if !current {
		return
	}
	var wire api.PluginInventorySnapshot
	if revision, ok := source.Cached(api.PluginInventoryListMethod, &wire); ok {
		c.pluginInventory.Accept(host, revision, wire)
		c.mu.Lock()
		first := c.inventorySource == source && !c.inventoryTraceReady
		if first {
			c.inventoryTraceReady = true
		}
		c.mu.Unlock()
		if first {
			startuptrace.Mark("plugins.inventory_ready")
			// workspace.ready may already have completed the trace writer.
			startuptrace.Finish("ready")
		}
	} else if method != api.PluginInventoryListMethod {
		// A manual market refresh invalidates the same shared generation;
		// refresh its already-visible runtime projection once as well.
		c.ensureInventory()
	}
	if ctrl != nil && ctrl.Snapshot().Status == "ready" {
		ctrl.Ensure()
	}
	c.invalidateUI()
}

func (c *Coordinator) ensureInventory() {
	c.mu.Lock()
	c.inventoryRefreshPending = true
	if c.inventoryRefreshRunning {
		c.mu.Unlock()
		return
	}
	c.inventoryRefreshRunning = true
	c.mu.Unlock()
	go func() {
		for {
			c.mu.Lock()
			if !c.inventoryRefreshPending {
				c.inventoryRefreshRunning = false
				c.mu.Unlock()
				return
			}
			c.inventoryRefreshPending = false
			c.mu.Unlock()
			ctx, cancel := context.WithTimeout(c.lifecycleContext(), 30*time.Second)
			c.pluginInventory.Refresh(ctx)
			cancel()
			c.invalidateUI()
		}
	}()
}

// ParkPluginViewForSettings applies Route A policy (Decision A2): leaving the
// explicit compatible panel or opening Settings closes the WebView host.
func (c *Coordinator) ParkPluginViewForSettings(_ bool) {
	if c != nil && c.deps != nil {
		c.deps.CloseWebViewHost()
	}
}
