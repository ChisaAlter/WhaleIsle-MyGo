package session

import (
	"encoding/json"
	"testing"
)

func TestChildImageLimitsScope(t *testing.T) {
	v := AgentNavigationView{SessionID: "child", Address: AgentSessionAddress{Kind: "subagent", ParentSessionID: "parent", ChildSessionID: "child", Mode: "continuable"}}
	seq := float64(3)
	seedChildImageLimits(&v, json.RawMessage(`{"imageLimits":{"mediaTypes":["image/png"],"maxImagesPerMessage":1,"maxImageBytes":100,"maxMessageImageBytes":100}}`), &seq)
	if v.ImageLimits == nil || v.ImageLimits.MaxImagesPerMessage != 1 {
		t.Fatal("missing child limits")
	}
	if err := validateChildImages([]string{"a", "b"}, v.ImageLimits); err == nil {
		t.Fatal("count limit ignored")
	}
	old := float64(2)
	seedChildImageLimits(&v, json.RawMessage(`{}`), &old)
	if v.ImageLimits == nil {
		t.Fatal("stale baseline cleared")
	}
	copy := cloneAgentNavigationView(v)
	copy.ImageLimits.MediaTypes[0] = "changed"
	if v.ImageLimits.MediaTypes[0] != "image/png" {
		t.Fatal("aliased")
	}
	seq = 4
	seedChildImageLimits(&v, json.RawMessage(`{}`), &seq)
	if v.ImageLimits != nil {
		t.Fatal("missing projection must clear")
	}
}
