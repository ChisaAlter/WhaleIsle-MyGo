package marketctl

import (
	"context"
	"encoding/json"
	"fmt"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/i18n"
)

// NativeController returns (and lazily initializes) the native plugin manager controller.
func (c *Coordinator) NativeController() *manager.Controller {
	c.mu.Lock()
	created := c.nativePlugins.controller == nil
	if c.nativePlugins.controller == nil {
		c.nativePlugins.controller = manager.New()
	}
	ctrl := c.nativePlugins.controller
	c.mu.Unlock()

	if created {
		deps := c.deps
		ctrl.SetRegistryPersistence(func() (*manager.RegistryChoice, error) {
			if deps == nil {
				return nil, nil
			}
			p, err := deps.GoPluginInstallRegistry()
			if err != nil || p == nil {
				return nil, err
			}
			return &manager.RegistryChoice{Kind: p.Kind, Registry: p.Registry, URL: p.URL}, nil
		}, func(choice *manager.RegistryChoice) error {
			if deps == nil {
				return fmt.Errorf("%s", i18n.T(i18n.KeyCmdPluginPreferencesUnavailable))
			}
			var reg *config.PluginInstallRegistry
			if choice != nil {
				reg = &config.PluginInstallRegistry{Kind: choice.Kind, Registry: choice.Registry, URL: choice.URL}
			}
			return deps.SaveGoPluginInstallRegistry(reg)
		})
	}
	return ctrl
}

// BindNativePluginManagerCaller binds an explicit RPC Caller under transport client.
func (c *Coordinator) BindNativePluginManagerCaller(client *dsh.Client, caller manager.Caller) {
	c.bindNativePluginManagerCaller(client, caller, false)
}

func (c *Coordinator) bindNativePluginManagerCaller(client *dsh.Client, caller manager.Caller, deferred bool) {
	ctrl := c.NativeController()
	c.mu.Lock()
	oldSource := c.inventorySource
	c.inventorySource = nil
	c.pluginEventsReady = false
	c.inventoryDeferred = deferred && caller != nil
	c.inventoryTraceReady = false
	c.mu.Unlock()
	if oldSource != nil {
		oldSource.Close()
	}
	notify := func() { c.updateUI() }
	var source *manager.HostInventory
	var inventoryHost uint64
	if caller != nil {
		source = manager.NewHostInventory(caller, func(method string) { c.acceptInventory(source, inventoryHost, method) })
	}
	c.mu.Lock()
	c.inventorySource = source
	store := c.store
	c.mu.Unlock()
	c.AttachInventory(client)
	inventoryHost = c.pluginInventory.Snapshot().Value.Host
	if caller == nil {
		ctrl.Detach()
		ctrl.SetNotify(notify)
	} else {
		ctrl.Attach(source, notify)
	}
	c.mu.Lock()
	c.nativePlugins.client = client
	c.nativePlugins.owner = ctrl.Snapshot().Value.Owner
	c.mu.Unlock()
	if store != nil {
		if source == nil {
			store.SetInventoryCaller(nil)
		} else {
			store.SetInventoryCaller(source)
		}
	}
}

// BindNativePluginManager attaches or detaches client on the native plugin manager.
func (c *Coordinator) BindNativePluginManager(client *dsh.Client) {
	if client == nil {
		c.BindNativePluginManagerCaller(nil, nil)
		return
	}
	c.BindNativePluginManagerCaller(client, client)
}

// NativeEventHandler returns an event sink bound to client's captured transport owner.
func (c *Coordinator) NativeEventHandler(client *dsh.Client) func(dsh.EventsEmit) {
	c.mu.Lock()
	binding := c.nativePlugins
	c.mu.Unlock()
	if client == nil || binding.client != client || binding.controller == nil {
		return func(dsh.EventsEmit) {}
	}
	ctrl, owner := binding.controller, binding.owner
	return func(emit dsh.EventsEmit) {
		if ctrl.Snapshot().Value.Owner != owner {
			return
		}
		switch emit.Event {
		case "plugin-manager/changed":
			c.refreshChangedInventory()
			ctrl.BundlesChanged(owner)
		case "plugin-market/changed":
			ctrl.SourcesChanged(owner)
		case "plugin-manager/install-log":
			if len(emit.Args) != 1 {
				return
			}
			var chunk manager.InstallLog
			if json.Unmarshal(emit.Args[0], &chunk) == nil && chunk.RequestID != "" {
				ctrl.AppendLog(owner, chunk)
			}
		case "plugin-manager/install-state":
			if len(emit.Args) != 1 {
				return
			}
			var progress manager.InstallProgress
			if json.Unmarshal(emit.Args[0], &progress) == nil && progress.RequestID != "" {
				ctrl.Progress(owner, progress)
			}
		}
	}
}

// HostEventsReady distinguishes initial event subscription from a reconnect
// gap. Initial readiness is not a plugin mutation and must not force either the
// full market catalog or a deferred desktop inventory to load.
func (c *Coordinator) HostEventsReady(client *dsh.Client) {
	c.mu.Lock()
	if client == nil || c.nativePlugins.client != client {
		c.mu.Unlock()
		return
	}
	reconnect := c.pluginEventsReady
	c.pluginEventsReady = true
	deferred := c.inventoryDeferred
	ctrl, owner := c.nativePlugins.controller, c.nativePlugins.owner
	c.mu.Unlock()
	if reconnect {
		c.refreshChangedInventory()
	} else if !deferred {
		c.ensureInventory()
	}
	if ctrl != nil {
		ctrl.SourcesChanged(owner)
	}
}

func (c *Coordinator) refreshChangedInventory() {
	c.mu.Lock()
	source := c.inventorySource
	ctrl := c.nativePlugins.controller
	store := c.store
	deferred := c.inventoryDeferred
	c.mu.Unlock()
	if source != nil {
		source.InvalidateInventory()
	}
	if !deferred {
		c.ensureInventory()
	}
	if ctrl != nil && ctrl.Snapshot().Value.Status != "idle" {
		go ctrl.Load(context.Background())
	}
	if store != nil && store.InventoryRequested() {
		store.RefreshShared(c.lifecycleContext())
	}
}

// PrepareInstalledPlugin is called by explicit A2 open. The pure state query
// remains free of work; initial full catalog loading belongs to this action.
func (c *Coordinator) PrepareInstalledPlugin() {
	ctrl := c.NativeController()
	if ctrl.Snapshot().Value.Status == "error" {
		ctrl.Refresh(c.lifecycleContext())
	} else {
		ctrl.Ensure()
	}
}

// InstalledPluginState returns a completed Host inventory view without
// starting work. Callers can compare Owner/Revision to their open-panel lease.
func (c *Coordinator) InstalledPluginState(name string) manager.InstalledPluginState {
	if c == nil {
		return manager.InstalledPluginState{}
	}
	c.mu.Lock()
	ctrl := c.nativePlugins.controller
	c.mu.Unlock()
	if ctrl == nil {
		return manager.InstalledPluginState{}
	}
	return ctrl.InstalledPlugin(name)
}

// ObserveFormSubject tracks whether the native plugin manager form changed subject.
func (c *Coordinator) ObserveFormSubject(view manager.Subject) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	changed := c.nativePlugins.formSubject != view
	c.nativePlugins.formSubject = view
	return changed
}
