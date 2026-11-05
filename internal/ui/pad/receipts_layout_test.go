package pad

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	desktop "whaleisle/internal/pad"
	store "whaleisle/internal/store/pad"
)

func fixturePad(t *testing.T, panels []store.Panel, action func(http.ResponseWriter, *http.Request)) (*store.Store, <-chan struct{}, string) {
	t.Helper()
	changed := make(chan struct{}, 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/dsh-pad.stream":
			w.Header().Set("Content-Type", "text/event-stream")
			raw, _ := json.Marshal(map[string]any{"type": "panels", "panels": panels})
			fmt.Fprintf(w, "data: %s\n\n", raw)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/api/dsh-pad.models":
			fmt.Fprint(w, `{"default":{"provider":"fixture","model":"small"},"providers":[{"id":"fixture","name":"Fixture provider","models":[{"id":"small","name":"Small fixture"}]}]}`)
		case "/api/dsh-pad.action":
			if action == nil {
				t.Error("unexpected Host mutation")
				return
			}
			action(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	home := t.TempDir()
	s, err := store.Open(home, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Detach(); server.Close() })
	client, _ := store.NewClient(server.URL, "")
	s.Attach(client, 12)
	waitPad(t, s, changed, func(v store.Snapshot) bool { return v.Loaded })
	return s, changed, home
}

func waitPad(t *testing.T, s *store.Store, changed <-chan struct{}, predicate func(store.Snapshot) bool) {
	t.Helper()
	timer := time.After(2 * time.Second)
	for !predicate(s.Snapshot()) {
		select {
		case <-changed:
		case <-timer:
			t.Fatal("Pad receipt did not arrive", s.Snapshot())
		}
	}
}

func TestCollapsedModelPickerKeepsFailureAndWaitsForSuccess(t *testing.T) {
	replies, started := make(chan string), make(chan store.Action, 4)
	p := store.Panel{ID: "collapsed", Version: 1, Collapsed: true, Definition: store.Definition{Title: "Collapsed fixture"}}
	s, changed, _ := fixturePad(t, []store.Panel{p}, func(w http.ResponseWriter, r *http.Request) {
		var a store.Action
		_ = json.NewDecoder(r.Body).Decode(&a)
		started <- a
		select {
		case reply := <-replies:
			fmt.Fprint(w, reply)
		case <-r.Context().Done():
		}
	})
	s.UpdateLayout(func(l *store.Layout) { l.Expanded = true; l.Height = 700 })
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Desktop: &desktop.Desktop{Store: s}, ThemeID: "deepseek.light", Width: 400, Height: 800})
	}, 400, 800)
	if err := tt.Click(copy("更多操作", "More actions")); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem(copy("面板模型", "Panel model") + " · " + copy("跟随全局默认", "Follow default")); err != nil {
		t.Fatal(err)
	}
	waitPad(t, s, changed, func(v store.Snapshot) bool { return v.Catalog != nil })
	tt.Frame()
	label := "✓ " + copy("跟随全局默认", "Follow default") + " · Small fixture"
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if a := <-started; a.Op != "model" || a.Model != nil {
		t.Fatal(a)
	}
	if !tt.HasText(copy("关闭", "Close")) {
		t.Fatal("picker closed before Host result")
	}
	if err := tt.Click("Small fixture"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
		t.Fatal("busy picker sent a concurrent selection")
	default:
	}
	replies <- `{"ok":false,"error":"fixture model rejected"}`
	waitPad(t, s, changed, func(v store.Snapshot) bool { return v.ModelActions[p.ID].Error != "" })
	tt.Frame()
	if !tt.HasText("fixture model rejected") || !tt.HasText(copy("关闭", "Close")) {
		t.Fatal("collapsed card hid model rejection", tt.Texts())
	}
	saveFrame(t, tt, "model-rejection-collapsed-400")
	if err := tt.Click("Small fixture"); err != nil {
		t.Fatal(err)
	}
	if a := <-started; a.Model == nil || a.Model.Model != "small" {
		t.Fatal(a)
	}
	replies <- `{"ok":true}`
	waitPad(t, s, changed, func(v store.Snapshot) bool { return v.ModelActions[p.ID].Succeeded })
	tt.Frame()
	if tt.HasText(copy("关闭", "Close")) {
		t.Fatal("successful selection did not close picker")
	}
	if err := tt.Click(copy("放大查看", "Expand panel")); err != nil {
		t.Fatal(err)
	}
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	tt.Frame()
	close, ok := tt.Find(copy("关闭", "Close"))
	if !ok || close.X < 0 || close.X+close.W > 400 || close.Y < 0 || close.Y+close.H > 800 {
		t.Fatal("narrow focus close escaped viewport", close)
	}
	saveFrame(t, tt, "focus-400")
	tt.Key(0, ui.KeyEscape)
	if tt.HasText(copy("关闭", "Close")) {
		t.Fatal("Escape did not close focus dialog")
	}
}

func TestSplitterUsesMeasuredPixelsAndPersistsOnlyOnRelease(t *testing.T) {
	panels := []store.Panel{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	s, _, home := fixturePad(t, panels, nil)
	s.UpdateLayout(func(l *store.Layout) {
		l.Tree = &store.Tree{Kind: "split", Dir: "row", Children: []*store.Tree{{Kind: "leaf", ID: "a"}, {Kind: "leaf", ID: "b"}, {Kind: "leaf", ID: "c"}}, Weights: []float32{1, 1, 1}}
	})
	path := filepath.Join(home, "pad", "layout.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		n := s.Snapshot().Layout.Tree
		row := ui.Row(c).Width(600).Height(100)
		row.Children(func() {
			for i, weight := range n.Weights {
				if i > 0 {
					splitter(c, row, s, n, "root", i-1)
				}
				ui.Box(c).Grow(weight).MinWidth(0).Height(100)
			}
		})
	}, 650, 150)
	r, ok := tt.Find(copy("调整分栏比例", "Resize panel columns"))
	if !ok {
		t.Fatal("splitter missing")
	}
	x, y := r.X+r.W/2, r.Y+r.H/2
	tt.Press(x, y)
	tt.Move(x+40, y)
	weights := s.Snapshot().Layout.Tree.Weights
	want := float32(1 + 40.0/(600-12)*3)
	if math.Abs(float64(weights[0]-want)) > .001 || math.Abs(float64(weights[0]+weights[1]-2)) > .001 || weights[2] != 1 {
		t.Fatalf("drag ignored measured card area or touched third pane: got %v want left=%f", weights, want)
	}
	during, _ := os.ReadFile(path)
	if string(before) != string(during) {
		t.Fatal("pointer motion wrote layout to disk")
	}
	tt.Release(x+40, y)
	tt.Frame()
	after, _ := os.ReadFile(path)
	if string(before) == string(after) {
		t.Fatal("release did not persist split")
	}
}
