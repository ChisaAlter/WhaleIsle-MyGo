package pad

import (
	"encoding/json"
	"fmt"
	"image/png"
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

func TestNativePadBallDockAndNarrowControls(t *testing.T) {
	for _, width := range []int{1024, 480} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			ready := make(chan struct{}, 16)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"type\":\"panels\",\"panels\":[]}\n\n")
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			defer server.Close()
			s, err := store.Open(t.TempDir(), func() {
				select {
				case ready <- struct{}{}:
				default:
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Detach()
			client, _ := store.NewClient(server.URL, "")
			s.Attach(client, 1)
			timer := time.After(2 * time.Second)
			for !s.Snapshot().Loaded {
				select {
				case <-ready:
				case <-timer:
					t.Fatal("not loaded")
				}
			}
			d := &desktop.Desktop{Store: s}
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Desktop: d, ThemeID: "deepseek.dark", Width: float32(width), Height: 520})
			}, width, 520)
			tt.SetDark(true)
			r, ok := tt.Find(copy("展开面板", "Open panels"))
			if !ok || r.X < 0 || r.X+r.W > float32(width) {
				t.Fatalf("ball bounds %+v found=%v", r, ok)
			}
			tt.Click(copy("展开面板", "Open panels"))
			tt.Frame()
			if !s.Snapshot().Layout.Expanded {
				t.Fatal("ball did not open native dock")
			}
			saveFrame(t, tt, fmt.Sprintf("empty-%d", width))
			for _, label := range []string{copy("新建", "New"), copy("刷新", "Refresh"), copy("收起", "Fold"), copy("还没有面板", "No panels yet")} {
				r, ok = tt.Find(label)
				if !ok || r.X < 0 || r.X+r.W > float32(width) {
					t.Fatalf("%s: %+v %v", label, r, ok)
				}
			}
			r, _ = tt.Find(copy("收起", "Fold"))
			tt.Press(r.X+r.W/2, r.Y+r.H/2)
			tt.Move(r.X+r.W/2-20, r.Y+r.H/2+20)
			tt.Release(r.X+r.W/2-20, r.Y+r.H/2+20)
			tt.Frame()
			if !s.Snapshot().Layout.Expanded || s.Snapshot().Layout.Dock == nil {
				t.Fatal("dragging collapse control folded the dock or lost placement")
			}
			tt.Click(copy("收起", "Fold"))
			tt.Frame()
			if s.Snapshot().Layout.Expanded {
				t.Fatal("fold did not restore native ball")
			}
		})
	}
}
func saveFrame(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	out := os.Getenv("WHALEISLE_PAD_RENDER_DIR")
	if out == "" {
		return
	}
	if err := os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(out, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
func TestNativePadApprovalModelsAndDeleteConfirmation(t *testing.T) {
	changed := make(chan struct{}, 32)
	actions := make(chan store.Action, 8)
	p := store.Panel{ID: "panel-qa", Version: 1, CanUndo: true, NeedsApproval: true, Fingerprint: "exact-source", Definition: store.Definition{Title: "服务器状态", Height: json.RawMessage(`"auto"`), Sources: []store.Source{{ID: "cpu", Kind: "command", Command: "echo qa-local", Parse: "text", IntervalSec: 3}}}, Data: store.Data{Sources: map[string]store.Reading{}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/dsh-pad.stream":
			w.Header().Set("Content-Type", "text/event-stream")
			raw, _ := json.Marshal(map[string]any{"type": "panels", "panels": []store.Panel{p}})
			fmt.Fprintf(w, "data: %s\n\n", raw)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/api/dsh-pad.action":
			var a store.Action
			_ = json.NewDecoder(r.Body).Decode(&a)
			actions <- a
			fmt.Fprint(w, `{"ok":true}`)
		case "/api/dsh-pad.models":
			fmt.Fprint(w, `{"default":null,"providers":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	s, err := store.Open(t.TempDir(), func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Detach()
	client, _ := store.NewClient(server.URL, "")
	s.Attach(client, 3)
	timer := time.After(2 * time.Second)
	for !s.Snapshot().Loaded {
		select {
		case <-changed:
		case <-timer:
			t.Fatal("not loaded")
		}
	}
	s.UpdateLayout(func(l *store.Layout) { l.Expanded = true; l.Height = 700 })
	d := &desktop.Desktop{Store: s}
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Desktop: d, ThemeID: "deepseek.light", Width: 480, Height: 900}) }, 480, 900)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	expect := func(op string) store.Action {
		t.Helper()
		select {
		case a := <-actions:
			if a.Op != op {
				t.Fatalf("got action %s want %s", a.Op, op)
			}
			return a
		case <-time.After(time.Second):
			t.Fatalf("missing action %s", op)
			return store.Action{}
		}
	}
	saveFrame(t, tt, "approval-480")
	click(copy("允许运行", "Allow execution"))
	if a := expect("approve"); a.Fingerprint != "exact-source" {
		t.Fatal(a)
	}
	click(copy("更多操作", "More actions"))
	// The shipped menu includes the current model beside the action label.
	if err := tt.ChooseMenuItem(copy("面板模型", "Panel model") + " · " + copy("跟随全局默认", "Follow default")); err != nil {
		t.Fatal(err)
	}
	timer = time.After(2 * time.Second)
	for s.Snapshot().Catalog == nil {
		select {
		case <-changed:
		case <-timer:
			t.Fatal("models not loaded")
		}
	}
	tt.Frame()
	click("✓ " + copy("跟随全局默认", "Follow default"))
	if a := expect("model"); a.Model != nil {
		t.Fatal(a)
	}
	timer = time.After(2 * time.Second)
	for !s.Snapshot().ModelActions[p.ID].Succeeded {
		select {
		case <-changed:
		case <-timer:
			t.Fatal("model success receipt did not arrive")
		}
	}
	tt.Frame()
	if tt.HasText(copy("关闭", "Close")) {
		t.Fatal("model picker stayed open after its successful receipt")
	}
	click(copy("更多操作", "More actions"))
	if err := tt.ChooseMenuItem(copy("删除面板…", "Delete panel…")); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-actions:
		t.Fatalf("deletion occurred without confirmation: %+v", a)
	default:
	}
	click(copy("取消", "Cancel"))
	click(copy("更多操作", "More actions"))
	if err := tt.ChooseMenuItem(copy("删除面板…", "Delete panel…")); err != nil {
		t.Fatal(err)
	}
	click(copy("确认删除", "Delete"))
	expect("delete")
}
