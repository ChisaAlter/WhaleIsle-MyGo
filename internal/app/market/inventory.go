package marketctl

import (
	"context"
	"time"

	"whaleisle/internal/desktop/plugininventory"
	"whaleisle/internal/dsh"
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
		caller = notifyingInventoryCaller{Caller: client, notify: c.invalidateUI}
	}
	c.pluginInventory.Attach(caller)
}

// RefreshInventory refreshes plugin inventory asynchronously and invalidates the UI.
func (c *Coordinator) RefreshInventory() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		c.pluginInventory.Refresh(ctx)
		c.invalidateUI()
	}()
}

// ParkPluginViewForSettings applies Route A policy (Decision A2): leaving the
// explicit compatible panel or opening Settings closes the WebView host.
func (c *Coordinator) ParkPluginViewForSettings(_ bool) {
	if c != nil && c.deps != nil {
		c.deps.CloseWebViewHost()
	}
}
