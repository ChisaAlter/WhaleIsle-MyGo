package settings

import "encoding/json"

// TitlebarPreferences controls entries only; it never closes a panel or disables its shortcut.
type TitlebarPreferences struct {
	HideTerminalToggle bool
	HideSurfacesToggle bool
}

func TitlebarPreferencesFrom(v ViewState) TitlebarPreferences {
	accepted := struct {
		TerminalToggle bool `json:"terminalToggle"`
		SurfacesToggle bool `json:"surfacesToggle"`
	}{true, true}
	for _, ns := range v.Namespaces {
		if ns.NS == "ui-titlebar" {
			_ = json.Unmarshal(ns.Value, &accepted)
			break
		}
	}
	return TitlebarPreferences{HideTerminalToggle: !accepted.TerminalToggle, HideSurfacesToggle: !accepted.SurfacesToggle}
}
