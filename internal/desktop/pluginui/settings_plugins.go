package pluginui

import "github.com/egoist/mygo/ui"

const settingsPluginsTabSlot = "settings.plugins.tab"

// SettingsPluginTabs returns registered pages, including never-selected tabs.
// The navigation source owns their original PluginsSettingsSection and keeps
// each previously selected page mounted while its own hidden panel is inactive.
func (o *Owner) SettingsPluginTabs() []NativeView {
	if o == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed || o.incompatible || !o.state.Connected {
		return nil
	}
	var tabs []NativeView
	for _, view := range o.state.Views {
		if view.Slot == settingsPluginsTabSlot {
			tabs = append(tabs, view)
		}
	}
	return tabs
}

// RenderSettingsPluginTab selects only the active native tab through the
// original Settings navigation and exact tab ID. Inactive visited wrappers must
// still call this method, but never create a native guest or steal selection.
// The source section retains its DOM/drafts; EndNativeFrame removes unpainted
// rich seats without hiding another active page on this shared Owner.
func (o *Owner) RenderSettingsPluginTab(c *ui.Context, handle string, active bool) bool {
	if !active || o == nil {
		return false
	}
	found := false
	for _, tab := range o.SettingsPluginTabs() {
		if tab.Handle == handle {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	subject := Subject{Kind: "view", Name: handle}
	o.Select(subject)
	snapshot := o.Snapshot()
	if snapshot.Subject == nil || !snapshot.Subject.equal(subject) {
		return false
	}
	for _, document := range snapshot.Documents {
		if document.Handle == handle && document.Slot == settingsPluginsTabSlot {
			// An observed empty page is still mounted, not a loading state.
			return o.renderNativeSnapshot(c, snapshot, settingsPluginsTabSlot, "", -1, -1, handle) || len(document.Nodes) == 0
		}
	}
	return false
}
