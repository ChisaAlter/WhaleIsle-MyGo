package main

import "whaleisle/internal/config"

// migrateViewTabsSetting persists the legacy view-tabs toggle. It returns the
// write error so the caller retries instead of marking the migration done.
func migrateViewTabsSetting(cfgStore *config.Store, hideChatViewTabs bool) error {
	return cfgStore.Update(func(c *config.Config) {
		c.HideChatViewTabs = hideChatViewTabs
	})
}

// dropDisabledPlugin removes name from the persisted disabled-plugin list.
func dropDisabledPlugin(cfgStore *config.Store, name string) error {
	return cfgStore.Update(func(c *config.Config) {
		kept := c.DisabledPlugins[:0]
		for _, existing := range c.DisabledPlugins {
			if existing != name {
				kept = append(kept, existing)
			}
		}
		c.DisabledPlugins = kept
	})
}
