package frame

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/ui/planpreview"
)

func TestPlanPaneIndependentSplit(t *testing.T) {
	for _, vertical := range []bool{false, true} {
		kind := "plan"
		w := &WorkspacePanes{}
		w.syncRoute(kind)
		w.Root.Files = &WorkspaceFileState{Draft: "private file edit"}
		tabs := &w.Root.Files.Tabs
		tabs.Bind("host\x00parent", "/workspace")
		address := planpreview.SubagentPlanAddress("parent", "child", "continuable", "call")
		if err := tabs.OpenPlanResource("parent", "host\x00parent", planpreview.Document{Address: address}); err != "" {
			t.Fatal(err)
		}
		tabs.Plans.Observe(planpreview.Document{Address: address, Title: "Original", Markdown: "body"})
		m := Model{RightPaneOpen: true, RightPaneTab: &kind, Sizes: &LayoutSizes{Workspace: w}}
		original, before := w.Root, string(w.Export())
		SplitWorkspacePane(m, vertical)
		if vertical {
			if w.Root != original || string(w.Export()) != before || w.Root.Files.Draft != "private file edit" || w.Root.Files.Tabs.Plans.Documents[address].Title != "Original" {
				t.Fatal("rejected vertical split changed retained plan or draft")
			}
			continue
		}
		a, b := w.Root.First, w.Root.Second
		if a == nil || b == nil || w.Root.Axis != "horizontal" || b.Kind != "plan" || a.Files == b.Files {
			t.Fatal("plan did not split independently")
		}
		before = string(w.Export())
		SplitWorkspacePane(m, false)
		if w.Root.First != a || w.Root.Second != b || string(w.Export()) != before {
			t.Fatal("third dock split changed retained plan owners")
		}
		if b.Files.Draft != "" || b.Files.Tabs.Plans.ActiveAddress("parent") != address {
			t.Fatal("copied file draft or lost plan address")
		}
		b.Files.Tabs.Plans.Close("parent", address)
		if a.Files.Tabs.Plans.ActiveAddress("parent") != address {
			t.Fatal("closing second plan changed first")
		}
		b.Files.Tabs.Plans.Navigate("parent", address)
		b.Files.Tabs.Plans.Documents[address] = planpreview.Document{Address: address, Title: "Changed"}
		if a.Files.Tabs.Plans.Documents[address].Title != "Original" {
			t.Fatal("shared plan cache")
		}
		restored, err := RestoreWorkspacePanes(w.Export())
		if err != nil {
			t.Fatal(err)
		}
		if restored.Root.First.Files.Tabs.Plans.ActiveAddress("parent") != address {
			t.Fatal("lost durable target")
		}
		if len(restored.Root.First.Files.Tabs.Plans.Documents) != 0 {
			t.Fatal("serialized private plan contents")
		}
		w.closeNow(b.ID)
		if w.Root.Files.Tabs.Plans.ActiveAddress("parent") != address {
			t.Fatal("close destroyed surviving plan")
		}
		w.Root.Files.Tabs.Bind("replacement\x00parent", "/workspace")
		if len(w.Root.Files.Tabs.Plans.Tabs) != 0 {
			t.Fatal("replacement Host inherited plan")
		}
	}
}

func TestPlanPaneRecoveryDropsTemporaryReview(t *testing.T) {
	n := WorkspacePane{Kind: "plan", Files: &WorkspaceFileState{}}
	n.Files.Tabs.Bind("host\x00parent", "/workspace")
	addr := planpreview.ReviewAddress("parent", "approval")
	n.Files.Tabs.OpenPlanResource("parent", "host\x00parent", planpreview.Document{Address: addr, Markdown: "temporary"})
	raw, _ := json.Marshal(n)
	var restored WorkspacePane
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	restored.Files.Tabs.Bind("host\x00parent", "/workspace")
	if len(restored.Files.Tabs.Plans.Tabs) != 0 {
		t.Fatal("recovered temporary approval")
	}
}
