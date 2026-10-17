package harness

import "testing"

func TestApplyPluginLogLineOverlays(t *testing.T) {
	var p PluginProgress
	if !ApplyPluginLogLine(&p, "» 3 desktop --patch overlay(s)") {
		t.Fatal("expected change")
	}
	if p.Phase != "overlays" || p.Ready != 3 || p.Total != 3 {
		t.Fatalf("%+v", p)
	}
}

func TestApplyPluginLogLineRuntime(t *testing.T) {
	var p PluginProgress
	if !ApplyPluginLogLine(&p, "dsh plugin-load: 2/5 (web-app)") {
		t.Fatal("expected change")
	}
	if p.Phase != "runtime" || p.Ready != 2 || p.Total != 5 || p.Detail != "web-app" {
		t.Fatalf("%+v", p)
	}
}
