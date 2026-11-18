package marketctl

import (
	"fmt"
	"time"

	manager "whaleisle/internal/pluginmanager"
)

const sidebarPluginCacheTTL = 10 * time.Second

// SidebarPlugins returns pure-Go discovered custom left-sidebar plugin entries
// for the currently attached Host, refreshing disk/manifest inspection off the
// UI thread whenever inventory, bundle status, profileDir, or locale changes.
func (c *Coordinator) SidebarPlugins(locale string) []manager.SidebarPluginEntry {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	client := c.client
	profileDir := c.profileDir
	store := c.store
	c.mu.Unlock()
	if client == nil {
		return nil
	}
	var token, refreshID uint64
	if store != nil {
		token = store.RuntimeToken()
		refreshID = store.CompletedRefreshID()
	}
	invSnap := c.pluginInventory.Snapshot()
	key := fmt.Sprintf("%s\x00%s\x00%d\x00%d\x00%d\x00%d", profileDir, locale, token, refreshID, invSnap.Host, invSnap.Revision)
	entries := c.sidebarPlugins.get(key, sidebarPluginCacheTTL, func() []manager.SidebarPluginEntry {
		var rows []manager.SidebarPluginCandidate
		if store != nil {
			for _, row := range store.Snapshot().Rows {
				enabled := row.Enabled == nil || *row.Enabled
				rows = append(rows, manager.SidebarPluginCandidate{
					Package:          row.Label,
					Title:            row.Title,
					Enabled:          enabled,
					ExplicitDisabled: row.Enabled != nil && !*row.Enabled,
					Official:         row.Official,
					Removable:        row.Removable,
					Capability:       row.CapabilityKind,
				})
			}
		}
		for _, invRow := range c.pluginInventory.Snapshot().Global {
			enabled := invRow.Enabled == "enabled"
			rows = append(rows, manager.SidebarPluginCandidate{
				Package:          invRow.Module,
				Title:            invRow.Title,
				Enabled:          enabled && invRow.Phase != "failed",
				ExplicitDisabled: invRow.Enabled == "disabled" || invRow.Phase == "failed",
			})
		}
		return manager.DiscoverSidebarPlugins(profileDir, locale, rows)
	}, c.invalidateUI)
	return append([]manager.SidebarPluginEntry(nil), entries...)
}
