package main

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
)

func (s *mainState) renderPluginPublicSlot(c *ui.Context, slot, ownerKey string, fallback func()) {
	if owner := s.pluginNavigationOwner(); owner != nil {
		owner.RenderPublicSlot(c, slot, ownerKey, fallback)
	} else if fallback != nil {
		fallback()
	}
}
func (s *mainState) pluginPublicMenuActions(slot, ownerKey string) []pluginui.PublicMenuAction {
	if owner := s.pluginNavigationOwner(); owner != nil {
		return owner.PublicMenuActions(slot, ownerKey)
	}
	return nil
}
func (s *mainState) invokePluginPublicMenuAction(action pluginui.PublicMenuAction) {
	if owner := s.pluginNavigationOwner(); owner != nil {
		if err := owner.InvokePublicMenuAction(action); err != nil {
			s.mu.Lock()
			win := s.win
			s.mu.Unlock()
			if win != nil {
				s.notifyUser(win, err.Error())
			}
		}
	}
}
func (s *mainState) pluginPublicContributions() pluginui.PublicContributions {
	return pluginui.PublicContributions{
		KeepNative: func(slot, ownerKey, id string) bool {
			if owner := s.pluginNavigationOwner(); owner != nil {
				return owner.PublicNativeWinner(slot, ownerKey, id)
			}
			return true
		},
		Slot: s.renderPluginPublicSlot,
		Before: func(c *ui.Context, slot, ownerKey, beforeID string) {
			if owner := s.pluginNavigationOwner(); owner != nil {
				owner.RenderPublicSlotBefore(c, slot, ownerKey, beforeID)
			}
		},
		Entries: func(slot, ownerKey string) []pluginui.PublicSlotEntry {
			if owner := s.pluginNavigationOwner(); owner != nil {
				return owner.PublicSlotEntries(slot, ownerKey)
			}
			return nil
		},
		RenderEntry: func(c *ui.Context, slot, ownerKey, handle string) bool {
			if owner := s.pluginNavigationOwner(); owner != nil {
				return owner.RenderPublicSlotEntry(c, slot, ownerKey, handle)
			}
			return false
		},
		MenuActions: s.pluginPublicMenuActions, Invoke: s.invokePluginPublicMenuAction,
		WorkspaceContext: func(query string, archived bool) {
			if owner := s.pluginNavigationOwner(); owner != nil {
				owner.SetPublicWorkspaceContext(query, archived)
			}
		},
		RowContext: func(sessionID string, menuOpen, hoverOpen bool) {
			if owner := s.pluginNavigationOwner(); owner != nil {
				owner.SetPublicRowContext(sessionID, menuOpen, hoverOpen)
			}
		},
	}
}

// Call from the actual native Settings route, using the installed section ID.
// Contributed section/tab routes already own Select(view) and must not be reset.
func (s *mainState) syncPluginPublicSettingsSection(sectionID string, open bool) {
	owner := s.pluginNavigationOwner()
	if owner == nil {
		return
	}
	current := owner.Snapshot().Subject
	if !open {
		if current != nil {
			sourceSettings := current.Kind == "navigation" && (current.Name == "models" || strings.HasPrefix(current.Name, "settings:"))
			if current.Kind == "view" {
				for _, view := range owner.Snapshot().Views {
					if view.Handle == current.Name && strings.HasPrefix(view.Slot, "settings.") {
						sourceSettings = true
						break
					}
				}
			}
			if sourceSettings {
				owner.Select(pluginui.Subject{Kind: "navigation"})
			}
		}
		return
	}
	if sectionID == "" || strings.HasPrefix(sectionID, "plugin/") {
		return
	}
	name := ""
	if sourceID, ok := pluginPublicSettingsSourceID(sectionID); ok {
		name = "settings:" + sourceID
		if sourceID == "models" {
			name = "models"
		}
	}
	owner.Select(pluginui.Subject{Kind: "navigation", Name: name})
}

// These are actual installed section IDs. Native-only namespace subpages do not
// manufacture source sections; keeping the source closed retires its children.
func pluginPublicSettingsSourceID(sectionID string) (string, bool) {
	switch sectionID {
	case "account", "general", "appearance", "interface", "conversation", "features", "models", "plugins", "skills", "market", "remote", "mcp", "agent-presets", "pet", "about", "usage-stats", "remote-workspace":
		return sectionID, true
	}
	return "", false
}
