package settings

import (
	"encoding/json"
	"testing"
)

func TestTitlebarPreferencesUseAcceptedHostValues(t *testing.T) {
	if p := TitlebarPreferencesFrom(ViewState{}); p.HideTerminalToggle || p.HideSurfacesToggle {
		t.Fatal("missing settings must retain both entries")
	}
	v := ViewState{Namespaces: []Namespace{{NS: "ui-titlebar", Value: json.RawMessage(`{"terminalToggle":false,"surfacesToggle":true}`)}}, SelectedNS: "ui-titlebar", Drafts: map[string]string{"terminalToggle": "true", "surfacesToggle": "false"}}
	p := TitlebarPreferencesFrom(v)
	if !p.HideTerminalToggle || p.HideSurfacesToggle {
		t.Fatalf("unaccepted draft changed titlebar: %+v", p)
	}
	v.Namespaces = nil
	if p = TitlebarPreferencesFrom(v); p.HideTerminalToggle || p.HideSurfacesToggle {
		t.Fatal("old Host preference retained")
	}
}
