package sessioncost

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/store/settings"
)

func TestPreferencesIgnoreUncommittedSwitches(t *testing.T) {
	v := settings.ViewState{Namespaces: []settings.Namespace{{NS: "ui-conversation", Value: json.RawMessage(`{"statsLine":false,"sessionCost":true,"officialPeakValley":false}`)}}, SelectedNS: "ui-conversation", Drafts: map[string]string{"statsLine": "true", "sessionCost": "false", "officialPeakValley": "true"}, Dirty: true, Err: "save failed"}
	stats, show, official := preferences(v)
	if stats || !show || official {
		t.Fatalf("draft changed accepted preferences: %v %v %v", stats, show, official)
	}
	v.Namespaces[0].Value = json.RawMessage(`{"statsLine":true,"sessionCost":false,"officialPeakValley":true}`)
	stats, show, official = preferences(v)
	if !stats || show || !official {
		t.Fatal("accepted save did not update preferences")
	}
	v.Namespaces = nil
	stats, show, official = preferences(v)
	if !stats || show || official {
		t.Fatal("old Host preferences leaked")
	}
}
