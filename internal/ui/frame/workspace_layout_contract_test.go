package frame

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/ui/files"
)

func TestLegacyWorkspaceMigrationRetainsEveryDraftAndFloatingIdentity(t *testing.T) {
	leaf := func(id, draft string) *WorkspacePane {
		return &WorkspacePane{ID: id, Kind: "files", Files: &WorkspaceFileState{Selected: id + ".txt", Preview: "baseline", Draft: draft, Editing: true, Mode: files.EditViewSplit}}
	}
	a, b, c, floating := leaf("a", "draft a"), leaf("b", "draft b"), leaf("c", "draft c"), leaf("floating", "draft float")
	floating.Float = &WorkspaceFloat{X: 120, Y: 60, Width: 440, Height: 300, Order: 4}
	old := WorkspacePanes{Active: c.ID, Root: &WorkspacePane{ID: "root", Axis: "vertical", Ratio: .05,
		First:  &WorkspacePane{ID: "left", Axis: "horizontal", Ratio: .95, First: a, Second: b},
		Second: &WorkspacePane{ID: "right", Axis: "vertical", Ratio: .5, First: c, Second: floating}}}
	raw, err := json.Marshal(old)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreWorkspacePanes(raw)
	if err != nil {
		t.Fatal(err)
	}
	docked := 0
	restored.Root.eachLeaf(func(p *WorkspacePane) {
		if p.dockedLeaf() != nil {
			docked++
		}
	})
	if docked != 2 || restored.Root.Axis != "horizontal" || restored.Active != c.ID {
		t.Fatalf("legacy topology not migrated/selection lost: docked=%d root=%+v active=%q", docked, restored.Root, restored.Active)
	}
	for _, original := range []*WorkspacePane{a, b, c, floating} {
		got := restored.Root.find(original.ID)
		if got == nil || got.Files == nil || got.Files.Draft != original.Files.Draft || got.Files.Preview != "baseline" || got.Files.Mode != files.EditViewSplit {
			t.Fatalf("migration lost resource draft/mode %q: %+v", original.ID, got)
		}
		if original.Float != nil && (got.Float == nil || *got.Float != *original.Float) {
			t.Fatalf("migration changed floating presentation: %+v", got.Float)
		}
	}
	second, err := RestoreWorkspacePanes(restored.Export())
	if err != nil || second.Root.find(c.ID) == nil || second.Active != c.ID {
		t.Fatalf("migrated layout does not roundtrip: err=%v", err)
	}
}

func TestWorkspaceRestoreRejectsMalformedIdentityAndSplitBeforeContent(t *testing.T) {
	for _, raw := range []string{
		`{"root":{"id":"r","axis":"horizontal","ratio":0.5,"first":{"id":"same","kind":"files"},"second":{"id":"same","kind":"files"}}}`,
		`{"root":{"id":"r","axis":"horizontal","ratio":0.5,"first":{"id":"a","kind":"files"}}}`,
		`{"root":{"id":"r","axis":"vertical","ratio":1,"first":{"id":"a","kind":"files"},"second":{"id":"b","kind":"files"}}}`,
	} {
		if got, err := RestoreWorkspacePanes(json.RawMessage(raw)); got != nil || err == nil {
			t.Fatalf("invalid tree admitted: %s; got=%+v err=%v", raw, got, err)
		}
	}
}

func TestWorkspaceSplitAdmissionUsesMeasuredChromeAndTwoSeats(t *testing.T) {
	w := &WorkspacePanes{Root: &WorkspacePane{ID: "files", Kind: "files"}}
	n := w.Root
	// 220/2 minus2 border < fixed10 +100 chip, although a raw half-width
	// heuristic would allow it. At224 the exact measured boundary is met.
	n.paneWidth, n.stripWidth, n.tabRunWidth = 220, 218, 208
	if w.canSplit(n) {
		t.Fatal("split ignored non-shrinking chrome")
	}
	n.paneWidth, n.stripWidth = 224, 222
	n.tabRunWidth = 212
	if !w.canSplit(n) {
		t.Fatal("measured chip-fit boundary rejected")
	}
	w.Root = &WorkspacePane{ID: "split", Axis: "horizontal", Ratio: .5, First: n, Second: &WorkspacePane{ID: "other", Kind: "diff"}}
	if w.canSplit(n) {
		t.Fatal("third docked seat was admitted")
	}
}
