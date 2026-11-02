package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginui"
	manager "whaleisle/internal/pluginmanager"
	managerui "whaleisle/internal/ui/pluginmanager"
)

func TestDynamicContributionLedgerPreservesNativeEditorsAndExactRowKeys(t *testing.T) {
	rendered := false
	native := []managerui.Contribution{{Kind: "item", Key: "agent-loop", Title: "old", Render: func(*ui.Context, manager.Subject) { rendered = true }}}
	live := pluginui.Snapshot{Entries: []pluginui.Entry{{Slot: "plugins.item", ID: "agent-loop", Label: "Live label", Summary: "Live summary", Order: 9}, {Slot: "plugins.item", ID: "third-party", Label: "Third party"}, {Slot: "plugins.bundle.config", Key: "fixture"}, {Slot: "plugins.row.config", Key: "fixture#patch-row"}}}
	merged := mergePluginContributions(native, nil, live)
	if len(merged) != 4 || merged[0].Title != "Live label" || merged[0].Summary != "Live summary" || merged[0].Order != 9 {
		t.Fatalf("live ledger not applied %+v", merged)
	}
	merged[0].Render(nil, manager.Subject{})
	if !rendered {
		t.Fatal("mature native editor replaced")
	}
	if merged[3].Key != "fixture#patch-row" {
		t.Fatal("patch-row identity changed")
	}
	if got := contributionSubject(manager.Subject{Kind: "row", Name: "fixture", RowID: "patch-row"}, []manager.Package{{Name: "fixture", Parts: []manager.Row{{RowID: "patch-row", EntryID: "loader-row"}}}}); got.RowID != "patch-row" || got.EntryID != "loader-row" {
		t.Fatalf("DOM navigation lost separate patch and loader IDs %+v", got)
	}
	withdrawn := mergePluginContributions(native, nil, pluginui.Snapshot{})
	if len(withdrawn) != 1 {
		t.Fatal("withdrawn third party entries retained")
	}
}
