package frame

import "fmt"

func (w *WorkspacePanes) syncPluginRightPanes(m Model) {
	present := map[string]bool{}
	for _, tab := range m.PluginRightPaneTabs {
		present[tab.Handle] = true
	}
	var closed []string
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.PluginRightPaneHandle == "" {
			return
		}
		if n.pluginCloseRequested && !present[n.PluginRightPaneHandle] {
			closed = append(closed, n.ID)
			return
		}
		n.pluginClose = nil
		if present[n.PluginRightPaneHandle] && m.OnClosePluginRightPaneTab != nil {
			handle := n.PluginRightPaneHandle
			n.pluginClose = func() { m.OnClosePluginRightPaneTab(handle) }
		}
	})
	for _, id := range closed {
		w.disposeWorkspaceResource(id)
		w.message = ""
		if m.RightPaneTab != nil {
			*m.RightPaneTab = w.route
		}
	}
}

// OpenPluginRightPaneTab addresses an already mounted source occurrence. It
// never creates a plugin owner from a kind name or duplicates its hook context.
func (w *WorkspacePanes) OpenPluginRightPaneTab(handle, label string) string {
	if handle == "" {
		return ""
	}
	var target *WorkspacePane
	w.Root.eachResource(func(n *WorkspacePane) {
		if n.PluginRightPaneHandle == handle {
			target = n
		}
	})
	if target == nil {
		target = w.leaf("plugin-native")
		target.PluginRightPaneHandle, target.PluginRightPaneLabel = handle, label
		if w.Root == nil {
			w.Root = target
		} else {
			w.placeDockResource(target, w.Root.find(w.Active), false)
		}
	}
	w.activateDockTab(target.ID)
	w.focusPending = target.ID
	if target.Float != nil {
		w.MinimizeWorkspaceFloat(target.ID, false)
	}
	return target.ID
}

// ReplaceGuideAfterOpen consumes only the initiating guide after the source
// reports its actual mounted handle. Navigating during the asynchronous open
// cannot replace a different resource or manufacture a plugin occurrence.
func (w *WorkspacePanes) ReplaceGuideAfterOpen(guideID, handle, label string) string {
	guide := w.Root.find(guideID)
	if guideID != "" && (guide == nil || guide.Kind != "guide") {
		return ""
	}
	if guide != nil && guide.Kind == "guide" {
		w.activateDockTab(guide.ID)
	}
	id := w.OpenPluginRightPaneTab(handle, label)
	if id != "" && guide != nil && guide.Kind == "guide" && guide.ID != id {
		w.disposeWorkspaceResource(guide.ID)
	}
	return id
}

// NewHostTerminal creates a fresh occurrence, not a route selection. Mounting
// consumes its existing terminalFresh marker once; recovery never recreates it.
func (w *WorkspacePanes) NewHostTerminal() string {
	added := w.leaf("host-terminal")
	if w.Root == nil {
		w.Root = added
	} else {
		w.placeDockResource(added, w.Root.find(w.Active), false)
	}
	w.activateDockTab(added.ID)
	w.focusPending = added.ID
	return added.ID
}

// PresentOpenedResource changes only the accepted resource's in-shell
// presentation. A document's pending open is consumed first, so Browser content
// floats as the same occurrence instead of spawning an unrelated native window.
func (w *WorkspacePanes) PresentOpenedResource(presentation string) error {
	if presentation == "" || presentation == "default" || presentation == "dock" {
		return nil
	}
	if presentation != "mini" {
		return fmt.Errorf("unsupported resource presentation %q", presentation)
	}
	apply := func() {
		n := w.Root.find(w.Active)
		if n == nil || n.Kind != "files" && n.Kind != "browser" {
			w.message = "The preview resource is no longer available"
			return
		}
		w.FloatWorkspaceTab(n.ID)
		w.focusPending = n.ID
	}
	if pending := w.pending; pending != nil {
		w.pending = func() { pending(); apply() }
	} else {
		apply()
	}
	return nil
}
