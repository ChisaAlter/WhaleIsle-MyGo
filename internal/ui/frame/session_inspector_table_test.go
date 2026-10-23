package frame

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestInspectorPreviewMatchesReferenceFieldsAndDepth(t *testing.T) {
	for _, test := range []struct{ raw, want string }{
		{`{"type":"reasoning-delta","index":0,"text":"thought"}`, "thought"},
		{`{"type":"tool-call-delta","index":3,"argumentsDelta":"{\"a\":1}"}`, `{"a":1}`},
		{`{"type":"other","seq":0,"time":123,"data":{"turn":0,"step":1,"content":["a","b"]}}`, "turn: 0 · step: 1 · content: [2]"},
		{`{"type":"future","seq":1,"a":{"inner":{"value":2}},"b":true}`, "a: {…} · b: true"},
		{`{"type":"future","a":1,"b":2,"c":3,"d":4,"e":5,"f":6}`, "a: 1 · b: 2 · c: 3 · d: 4 · e: 5"},
	} {
		if got := inspectorPreview([]byte(test.raw)); got != test.want {
			t.Fatalf("%s: %q != %q", test.raw, got, test.want)
		}
	}
	fields := inspectorFields([]byte(`{"z":0,"2":2,"1":1,"a":3,"z":4}`))
	var names []string
	for _, field := range fields {
		names = append(names, field.name)
	}
	if !reflect.DeepEqual(names, []string{"1", "2", "z", "a"}) || string(fields[2].raw) != "4" {
		t.Fatal(fields)
	}
}

func TestInspectorTableColumnsScrollingSelectionAndDetails(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	rows := []session.InspectorRecord{
		{Key: "event:0", Type: "turn/start", Raw: json.RawMessage(`{"type":"turn/start","seq":0,"time":0,"data":{"turn":0}}`)},
		{Key: "event:1", Parent: "event:0", Type: "user/message", Seq: 1, Raw: json.RawMessage(`{"type":"user/message","seq":1,"data":{"turn":0,"step":0,"text":"fixture preview"}}`)},
	}
	p := SessionInspectorPaneState{collapsed: map[string]bool{}}
	source := session.InspectorSnapshot{Records: rows}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, source, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 600)
	header, ok := tt.Find("Inspector table header")
	if !ok || header.H != 30 {
		t.Fatalf("header=%+v %v", header, ok)
	}
	row, ok := tt.Find("Inspector record event:1")
	if !ok || row.H != 30 {
		t.Fatalf("row=%+v %v", row, ok)
	}
	if !tt.HasText("Time (UTC)") || !tt.HasText("00:00:00.000") || !tt.HasText("fixture preview") || inspectorRecordIdentity(rows[0]) != "0" {
		t.Fatal("missing table metadata")
	}
	if tt.HasText("Load earlier") {
		t.Fatal("unavailable pagination still visible")
	}
	if err := tt.Click("user/message"); err != nil {
		t.Fatal(err)
	}
	if p.selected != "event:1" || !tt.HasText("Raw data") {
		t.Fatal("selection did not open raw details")
	}
	divider, ok := tt.Find("Inspector details divider")
	if !ok || divider.H != 5 || divider.Y != 360 {
		t.Fatalf("divider=%+v", divider)
	}
	if err := tt.Click("Close"); err != nil {
		t.Fatal(err)
	}
	if p.selected != "" {
		t.Fatal("details did not close")
	}
	row, ok = tt.Find("Inspector record event:1")
	if !ok {
		t.Fatal("record row missing after closing details")
	}
	// Clicking blank row space selects the same record as its type/data controls.
	tt.ClickAt(row.X+300, row.Y+15)
	if p.selected != "event:1" {
		t.Fatal("record row action did not select details")
	}
	tt.Key(0, ui.KeyEnter)
	if p.selected != "event:1" {
		t.Fatal("record row keyboard action lost selection")
	}
	if err := tt.Click("Close"); err != nil {
		t.Fatal(err)
	}
	tt.SetSize(360, 600)
	if p.tableX.MaxX < 299 {
		t.Fatalf("narrow table lost X range: %+v", p.tableX)
	}
	p.tableX.X = p.tableX.MaxX
	tt.Frame()
	if !tt.HasText("fixture preview") {
		t.Fatal("data did not remain reachable after X scroll")
	}
}

func TestInspectorPinnedControlsKeepOneInstanceAndScrollOwner(t *testing.T) {
	rows := []session.InspectorRecord{{Key: "turn", Type: "turn/start"}, {Key: "step", Parent: "turn", Type: "step/start"}}
	for i := 0; i < 80; i++ {
		rows = append(rows, session.InspectorRecord{Key: fmt.Sprintf("leaf:%d", i), Parent: "step", Type: "delta"})
	}
	p := SessionInspectorPaneState{collapsed: map[string]bool{}}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 300)
	checkUnique := func() {
		t.Helper()
		count := 0
		for _, text := range tt.Texts() {
			if text == "Inspector record turn" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("pinned record instances=%d", count)
		}
	}
	checkUnique()
	pinned, ok := tt.Find("turn/start")
	if !ok {
		t.Fatal("missing pinned turn")
	}
	tt.Scroll(pinned.X+2, pinned.Y+2, 0, 180)
	tt.Frame()
	if p.tableY.Y <= 0 {
		t.Fatal("wheel over pinned record did not reach vertical list")
	}
	checkUnique()
	if err := tt.Click("turn/start"); err != nil {
		t.Fatal(err)
	}
	before := p.tableY.Y
	tt.Key(0, ui.KeyPageDown)
	tt.Frame()
	if p.tableY.Y <= before {
		t.Fatalf("scroll key from pinned focus failed: before=%v after=%v", before, p.tableY.Y)
	}
	checkUnique()
}

func TestInspectorStickyAncestryAndFilterContext(t *testing.T) {
	rows := []session.InspectorRecord{{Key: "turn", Type: "turn/start"}, {Key: "step", Parent: "turn", Type: "step/start"}}
	for i := 0; i < 30; i++ {
		rows = append(rows, session.InspectorRecord{Key: fmt.Sprintf("leaf:%d", i), Parent: "step", Type: "delta"})
	}
	depths, branches := inspectorDepths(rows)
	if depths["leaf:0"] != 2 {
		t.Fatal(depths)
	}
	if got := inspectorStickyIndexes(rows, branches, 150); !reflect.DeepEqual(got, []int{0, 1}) {
		t.Fatal(got)
	}
	p := SessionInspectorPaneState{Filter: " *DELTA* ", collapsed: map[string]bool{"turn": true}}
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, &p, session.InspectorSnapshot{Records: rows}, session.ViewState{}, theme.Default().Light, nil)
	}, 780, 300)
	if !tt.HasText("30 / 32") || !tt.HasText("delta") {
		t.Fatal("filter count or contextual descendants missing")
	}
	p.changeInspectorMode("chat-node", "s")
	tt.Frame()
	if p.Filter != "" || p.selected != "" || p.navigation.SessionID != "s" || p.detailsPercent != 0 || p.tableX.X != 0 {
		t.Fatal("mode retained old presentation state")
	}
}
