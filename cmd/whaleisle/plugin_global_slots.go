package main

import "github.com/egoist/mygo/ui"

func (s *mainState) renderPluginGlobalSlot(c *ui.Context, slot string, fallback func()) {
	owner := s.pluginNavigationOwner()
	// Trailing is a list: the native builtin controls remain alongside external
	// entries. Single occupants replace only their corresponding native region.
	if slot == "shell.titlebar.trailing" {
		if fallback != nil {
			fallback()
		}
		if owner != nil {
			owner.RenderGlobalNativeSlot(c, slot)
		}
		return
	}
	if owner != nil && owner.RenderGlobalNativeSlot(c, slot) {
		return
	}
	if fallback != nil {
		fallback()
	}
}
