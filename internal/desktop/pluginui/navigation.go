package pluginui

// NativeView is a live navigable entry. Its handle identifies the actual slot,
// entry ID and registrant; withdrawing that entry revokes the navigation.
type NativeView struct {
	Handle     string `json:"handle"`
	Slot       string `json:"slot"`
	ID         string `json:"id"`
	Label      string `json:"label"`
	Registrant string `json:"registrant"`
	Order      int    `json:"order,omitempty"`
}

var navigationSlots = map[string]bool{"main": true, "sidebar.page": true, "settings.section": true, "settings.remote.tab": true, "settings.plugins.tab": true, "sidebar.right.pane.tab": true}

var globalSlots = map[string]bool{"root": true, "shell.overlay": true, "shell.titlebar.trailing": true, "shell.terminalDrawer": true, "shell.bottom": true}

func validNativeViews(views []NativeView) bool {
	if len(views) > 512 {
		return false
	}
	seen := map[string]bool{}
	for _, view := range views {
		if !navigationSlots[view.Slot] || view.Handle == "" || len(view.Handle) > 16384 || seen[view.Handle] || view.ID == "" || len(view.ID) > 8192 || len(view.Label) > 8192 || len(view.Registrant) > 8192 {
			return false
		}
		seen[view.Handle] = true
	}
	return true
}
