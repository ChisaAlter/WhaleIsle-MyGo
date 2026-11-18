package pluginmanager

// InstalledPluginState describes one package in the last completed Host
// inventory. Known is false while loading, disconnected, or after a failed
// refresh. Absence from both bundle and Loader inventories is known disabled.
type InstalledPluginState struct {
	Owner, Revision uint64
	Known, Enabled  bool
}

// InstalledPlugin reads only in-memory inventory; it never initiates an RPC,
// scans manifests, or treats the static sidebar hints as runtime availability.
func (c *Controller) InstalledPlugin(name string) InstalledPluginState {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := InstalledPluginState{Owner: c.state.Owner, Revision: c.inventoryRevision}
	s.Known = c.state.Connected && !c.state.Loading && c.state.Status == "ready"
	if source, ok := c.client.(interface{ InventoryVersion() uint64 }); ok && c.inventorySourceVersion != source.InventoryVersion() {
		s.Known = false
	}
	if !s.Known {
		return s
	}
	for _, p := range c.state.Packages {
		if p.Name != name {
			continue
		}
		s.Enabled = p.Enabled && p.Availability != "missing" && p.Error == nil
		if s.Enabled && len(p.Parts) > 0 {
			allFailed := true
			for _, row := range p.Parts {
				if row.Phase != "failed" {
					allFailed = false
					break
				}
			}
			s.Enabled = !allFailed
		}
		return s
	}
	// Desktop overlays inserted through --patch need not appear in the bundle
	// catalog. Use their exact Loader identity from the same completed read;
	// an explicit bundle state above always owns enablement when present.
	for _, plugin := range c.inventoryPlugins {
		if plugin.ModuleName == name && plugin.Enabled && plugin.Phase != "failed" {
			s.Enabled = true
			break
		}
	}
	return s
}
