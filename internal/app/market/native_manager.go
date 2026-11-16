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
	ctrl := c.NativeController()
	notify := func() { c.updateUI() }
	if caller == nil {
		ctrl.Detach()
		ctrl.SetNotify(notify)
	} else {
		ctrl.Attach(caller, notify)
	}
	c.mu.Lock()
	c.nativePlugins.client = client
	c.nativePlugins.owner = ctrl.Snapshot().Owner
	c.mu.Unlock()
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
		if ctrl.Snapshot().Owner != owner {
			return
		}
		switch emit.Event {
		case "plugin-manager/changed":
			if ctrl.Snapshot().Status != "idle" {
				ctrl.Refresh(context.Background())
			}
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

// ObserveFormSubject tracks whether the native plugin manager form changed subject.
func (c *Coordinator) ObserveFormSubject(view manager.Subject) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	changed := c.nativePlugins.formSubject != view
	c.nativePlugins.formSubject = view
	return changed
}
