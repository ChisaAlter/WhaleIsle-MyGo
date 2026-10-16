package settings

import (
	"encoding/json"
	"testing"
)

func TestDeveloperToolsUsesAcceptedHostValues(t *testing.T) {
	if !DeveloperToolsEnabled(ViewState{}) {
		t.Fatal("missing preference must default on")
	}
	v := ViewState{Namespaces: []Namespace{{NS: "ui-settings", Value: json.RawMessage(`{"enabled":false}`)}}, Drafts: map[string]string{"enabled": "true"}}
	if DeveloperToolsEnabled(v) {
		t.Fatal("draft overrides accepted off")
	}
}
