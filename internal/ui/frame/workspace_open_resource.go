package frame

import "fmt"

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
