package session

import "testing"

func TestPresetDuplicateCaptionIsNotAnIdentity(t *testing.T) {
	if _, ok := permissionOptionByLabel("Same", []PermissionOption{{Label: "Same", Value: "safe"}, {Label: "Same", Value: "unsafe"}}); ok {
		t.Error("permission caption silently selected first ID")
	}
	if _, ok := agentPresetOptionByLabel("Same", []AgentPresetOption{{Label: "Same", ID: "a"}, {Label: "Same", ID: "b"}}); ok {
		t.Error("agent caption silently selected first ID")
	}
}

func TestPresetOpaqueIdentitySurvivesDuplicateCaption(t *testing.T) {
	permissions := []PermissionOption{{Label: "Same", Value: "safe"}, {Label: "Same", Value: "unsafe"}}
	if got, ok := permissionOptionByValue("unsafe", permissions); !ok || got.Value != "unsafe" {
		t.Fatalf("permission identity: %+v %v", got, ok)
	}
	agents := []AgentPresetOption{{Label: "Same", ID: "a"}, {Label: "Same", ID: "b"}}
	if got, ok := agentPresetOptionByID("b", agents); !ok || got.ID != "b" {
		t.Fatalf("agent identity: %+v %v", got, ok)
	}
	if _, ok := permissionOptionByValue("Same", permissions); ok {
		t.Fatal("caption accepted as value")
	}
	if _, ok := agentPresetOptionByID("Same", agents); ok {
		t.Fatal("caption accepted as ID")
	}
}
