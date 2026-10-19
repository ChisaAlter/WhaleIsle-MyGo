package settings

import (
	"encoding/json"
	"testing"
)

func TestDeveloperToolsUsesAcceptedHostValues(t *testing.T) {
	// The schema's new-installation default is true, but the actual client
	// preference is scope.value?.enabled ?? false until a Host value is accepted.
	if DeveloperToolsEnabled(ViewState{}) {
		t.Fatal("missing accepted preference must remain off")
	}
	v := ViewState{Namespaces: []Namespace{{NS: "ui-settings", Value: json.RawMessage(`{"enabled":false}`)}}, Drafts: map[string]string{"enabled": "true"}}
	if DeveloperToolsEnabled(v) {
		t.Fatal("draft overrides accepted off")
	}
	v.Namespaces[0].Value = json.RawMessage(`{"enabled":true}`)
	v.Drafts["enabled"] = "false"
	if !DeveloperToolsEnabled(v) {
		t.Fatal("draft overrides accepted on")
	}
}
