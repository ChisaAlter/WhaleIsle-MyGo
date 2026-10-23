package frame

import (
	"fmt"
	"testing"
	"whaleisle/internal/store/session"
)

func inspectorLayoutRows(closed bool) []session.InspectorRecord {
	state := "open"
	if closed {
		state = "closed"
	}
	return []session.InspectorRecord{{Key: "assistant"}, {Key: "block", Parent: "assistant", Disclosure: state}, {Key: "delta1", Parent: "block"}, {Key: "delta2", Parent: "block"}}
}
func inspectorTestLayout(rows []session.InspectorRecord, previous *inspectorTableLayout, offset, height float32, anchor *inspectorRowAnchor) *inspectorTableLayout {
	_, branches := inspectorDepths(rows)
	return newInspectorTableLayout(inspectorDisclosedRows(rows, nil), branches, previous, offset, height, anchor)
}
func TestInspectorLayoutClosurePreservesParentAndConsumesTrailingSpace(t *testing.T) {
	before := inspectorTestLayout(inspectorLayoutRows(false), nil, 0, 0, nil)
	after := inspectorTestLayout(inspectorLayoutRows(true), before, 90, 300, nil)
	if !after.preservedCollapse || after.scrollOffset == nil || *after.scrollOffset != 0 || after.bottomPadding != 210 || len(after.rows) != 2 {
		t.Fatalf("after=%+v", after)
	}
	rows := append(inspectorLayoutRows(true), session.InspectorRecord{Key: "next"})
	next := inspectorTestLayout(rows, after, 0, 0, nil)
	if next.scrollOffset != nil || next.preservedCollapse || next.bottomPadding != 180 {
		t.Fatalf("new rows did not consume only trailing space: %+v", next)
	}
	// A later visible unrelated row retains its own visual position.
	before = inspectorTestLayout(append(inspectorLayoutRows(false), session.InspectorRecord{Key: "tail1"}, session.InspectorRecord{Key: "tail2"}), nil, 0, 0, nil)
	after = inspectorTestLayout(append(inspectorLayoutRows(true), session.InspectorRecord{Key: "tail1"}, session.InspectorRecord{Key: "tail2"}), before, 90, 300, nil)
	if after.scrollOffset == nil || *after.scrollOffset != 30 {
		t.Fatalf("later anchor moved: %+v", after)
	}
}
func TestInspectorLayoutManualFoldKeepsMeasuredHeaderAndCompactRows(t *testing.T) {
	var rows []session.InspectorRecord
	for i := 0; i < 20; i++ {
		rows = append(rows, session.InspectorRecord{Key: fmt.Sprint(i)})
	}
	rows = append(rows, session.InspectorRecord{Key: "turn", Disclosure: "closed"})
	for _, top := range []float32{30, 120} {
		layout := inspectorTestLayout(rows, nil, 2000, 300, &inspectorRowAnchor{key: "turn", top: top})
		if layout.scrollOffset == nil || *layout.scrollOffset != 630-top || layout.bottomPadding != 270-top {
			t.Fatalf("top%v: %+v", top, layout)
		}
		next := inspectorTestLayout(append(rows, session.InspectorRecord{Key: "next"}), layout, 0, 0, nil)
		if next.bottomPadding != 240-top || next.scrollOffset != nil {
			t.Fatalf("append top%v: %+v", top, next)
		}
	}
	compact := inspectorTestLayout(rows, nil, 0, 0, nil)
	if compact.bottomPadding != 0 {
		t.Fatal("closed history reserved space without a preceding fold")
	}
}
