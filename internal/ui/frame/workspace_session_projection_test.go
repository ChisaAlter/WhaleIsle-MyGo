package frame

import (
	"testing"
	"whaleisle/internal/ui/planpreview"
)

func TestWorkspaceSessionPlanPauseFixture(t *testing.T) {
	w := &WorkspacePanes{}
	w.syncRoute("plan")
	w.Root.Files = &WorkspaceFileState{}
	tabs := &w.Root.Files.Tabs
	tabs.Bind("host\x00one", "/project")
	address := planpreview.SubagentPlanAddress("one", "child", "continuable", "call")
	doc := planpreview.Document{Address: address}
	if err := tabs.OpenPlanResource("one", "host\x00one", doc); err != "" {
		t.Fatal(err)
	}
	w.SuspendSession()
	if tabs.Plans.ActiveAddress("one") != address {
		t.Fatal("session pause removed its plan address")
	}
	if err := tabs.OpenPlanResource("one", "host\x00one", doc); err == "" {
		t.Fatal("paused session accepted a stale binding")
	}
	tabs.Bind("host\x00one", "/project")
	if tabs.Plans.ActiveAddress("one") != address {
		t.Fatal("same Host/session did not restore the plan")
	}
	tabs.Bind("other-host\x00one", "/project")
	if tabs.Plans.ActiveAddress("one") != "" {
		t.Fatal("plan capability crossed Host boundary")
	}
}
