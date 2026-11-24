package frame

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

func TestInspectorRenderCacheReusesAndInvalidatesOnDisclosureChange(t *testing.T) {
	records := make([]session.InspectorRecord, 0, 40)
	for i := 0; i < 10; i++ {
		turnKey := fmt.Sprintf("turn:%d", i)
		childKey := fmt.Sprintf("evt:%d", i)
		records = append(records,
			session.InspectorRecord{
				Key:        turnKey,
				Type:       "turn/start",
				Seq:        float64(i*2 + 1),
				Disclosure: "open",
				Raw:        json.RawMessage(fmt.Sprintf(`{"type":"turn/start","seq":%d,"data":{"turn":%d}}`, i*2+1, i)),
			},
			session.InspectorRecord{
				Key:    childKey,
				Parent: turnKey,
				Type:   "assistant/message",
				Seq:    float64(i*2 + 2),
				Raw:    json.RawMessage(fmt.Sprintf(`{"type":"assistant/message","seq":%d,"data":{"turn":%d}}`, i*2+2, i)),
			},
		)
	}
	source := session.InspectorSnapshot{
		Binding:   "bind-1",
		SessionID: "sess-1",
		Records:   records,
	}
	p := &SessionInspectorPaneState{}

	chat1, obj1 := p.cachedSourceProjection(source)
	chat2, obj2 := p.cachedSourceProjection(source)
	if len(chat1) != len(chat2) || len(obj1) != len(obj2) {
		t.Fatal("source cache returned mismatched projection")
	}
	// Verify pointer identity of cached map on unchanged snapshot.
	obj1["__probe__"] = inspectorObject{Key: "__probe__"}
	if _, ok := obj2["__probe__"]; !ok {
		t.Fatal("expected cached objects map reuse on unchanged source")
	}
	delete(obj1, "__probe__")

	_, _, _, shown1 := p.cachedTableProjection(records)
	_, _, _, shown2 := p.cachedTableProjection(records)
	if len(shown1) != len(records) || &shown1[0] != &shown2[0] {
		t.Fatal("expected cached disclosed rows slice reuse on unchanged table")
	}

	// Collapsing turn:0 must invalidate table cache immediately.
	p.setInspectorCollapsed(records[0], true)
	_, _, _, shown3 := p.cachedTableProjection(records)
	if len(shown3) != len(records)-1 {
		t.Fatalf("collapsing turn:0 produced %d rows, want %d", len(shown3), len(records)-1)
	}

	// Resetting collapsed must restore all rows.
	p.resetCollapsed()
	_, _, _, shown4 := p.cachedTableProjection(records)
	if len(shown4) != len(records) {
		t.Fatalf("resetting collapsed produced %d rows, want %d", len(shown4), len(records))
	}
}

func BenchmarkFrameViewSteadyWithoutInspector(b *testing.B) {
	store := session.New()
	layout := DefaultLayout()
	draft := ""
	rightTab := "files"
	m := Model{
		Store:        store,
		Sizes:        &layout,
		Draft:        &draft,
		RightPaneTab: &rightTab,
		ThemeID:      "deepseek.light",
	}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, m)
	}, 1280, 800)
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

func BenchmarkInspectorPresentationSteady(b *testing.B) {
	records := make([]session.InspectorRecord, 0, 200)
	for i := 0; i < 100; i++ {
		turnKey := fmt.Sprintf("turn:%d", i)
		records = append(records,
			session.InspectorRecord{
				Key:        turnKey,
				Type:       "turn/start",
				Seq:        float64(i*2 + 1),
				Disclosure: "open",
				Raw:        json.RawMessage(fmt.Sprintf(`{"type":"turn/start","seq":%d,"data":{"turn":%d}}`, i*2+1, i)),
			},
			session.InspectorRecord{
				Key:    fmt.Sprintf("evt:%d", i),
				Parent: turnKey,
				Type:   "assistant/message",
				Seq:    float64(i*2 + 2),
				Raw:    json.RawMessage(fmt.Sprintf(`{"type":"assistant/message","seq":%d,"data":{"turn":%d,"text":"hello world"}}`, i*2+2, i)),
			},
		)
	}
	source := session.InspectorSnapshot{Binding: "b", SessionID: "s", Records: records}
	state := session.ViewState{SelectedID: "s"}
	p := &SessionInspectorPaneState{Mode: "event"}
	colors := theme.Resolve("deepseek.light").Active(false)
	tt := ui.NewTester(func(c *ui.Context) {
		inspectorPresentation(c, p, source, state, colors, func() {})
	}, 720, 600)
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}
