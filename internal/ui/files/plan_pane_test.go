package files

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/ui/planpreview"
)

func TestCopyPlanPaneExplicitReadAndRecovery(t *testing.T) {
	var source Tabs
	source.Bind("host\x00parent", "/workspace")
	addr := planpreview.SubagentPlanAddress("parent", "child", "continuable", "call")
	if err := source.OpenPlanResource("parent", "host\x00parent", planpreview.Document{Address: addr}); err != "" {
		t.Fatal(err)
	}
	copy := source.CopyPlanPane()
	if copy.planRead != nil || copy.planReadAddress != addr {
		t.Fatal("split copied reader or lost explicit navigation")
	}
	source.CancelPlanRead()
	if copy.planReadAddress != addr {
		t.Fatal("closing source revoked independent copy")
	}
	raw, _ := json.Marshal(copy)
	var restored Tabs
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	restored.Bind("host\x00parent", "/workspace")
	if restored.planReadAddress != "" || restored.planRead != nil {
		t.Fatal("recovery reauthorized child reader")
	}
	if restored.Plans.ActiveAddress("parent") != addr {
		t.Fatal("recovery lost address")
	}
}
