package frame

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
)

// PublicPluginSlot is an original-position replacement or an external-only list.
// Mixed native/plugin lists use PublicPluginBefore at each original builtin ID.
func PublicPluginSlot(c *ui.Context, contributions pluginui.PublicContributions, slot, ownerKey string, fallback func()) {
	if contributions.Slot != nil {
		contributions.Slot(c, slot, ownerKey, fallback)
	} else if fallback != nil {
		fallback()
	}
}

func PublicPluginBefore(c *ui.Context, contributions pluginui.PublicContributions, slot, ownerKey, beforeID string) bool {
	if contributions.Before != nil {
		contributions.Before(c, slot, ownerKey, beforeID)
	}
	return contributions.NativeWinner(slot, ownerKey, beforeID)
}

func PublicSlotEntries(contributions pluginui.PublicContributions, slot, ownerKey string) []pluginui.PublicSlotEntry {
	if contributions.Entries != nil {
		return contributions.Entries(slot, ownerKey)
	}
	return nil
}

func RenderPublicSlotEntry(c *ui.Context, contributions pluginui.PublicContributions, entry pluginui.PublicSlotEntry) bool {
	return !entry.Catalog && !entry.Native && contributions.RenderEntry != nil && contributions.RenderEntry(c, entry.Slot, entry.OwnerKey, entry.Handle)
}

// PublicPluginMenu stays live while the actual native owner is open. Source
// menu children can mount after the opening gesture and withdraw on its close.
func PublicPluginMenu(menu *ui.Menu, contributions pluginui.PublicContributions, slot, ownerKey string) {
	contributions.RenderMenu(menu, slot, ownerKey)
}

// PublicPluginMenuBefore interleaves source menu items at original builtin IDs.
func PublicPluginMenuBefore(menu *ui.Menu, contributions pluginui.PublicContributions, slot, ownerKey, beforeID string) bool {
	return contributions.RenderMenuBefore(menu, slot, ownerKey, beforeID)
}
