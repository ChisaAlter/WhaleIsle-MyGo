package frame

import (
	"context"
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestArchiveUndoToastDispatch(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale("zh")
	var restores atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		v := map[string]any{}
		switch {
		case strings.HasSuffix(r.URL.Path, "/workspace/archiveSession"):
			v["archivedSessionIds"] = []string{"a"}
		case strings.HasSuffix(r.URL.Path, "/workspace/unarchiveSession"):
			restores.Add(1)
			v["archivedSessionIds"] = []string{}
		case strings.HasSuffix(r.URL.Path, "/session/list"):
			v["items"] = []any{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": b["rpcId"], "result": map[string]any{"ok": true, "value": v}})
	}))
	defer srv.Close()
	s := session.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.ArchiveSession(context.Background(), "a")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && s.Snapshot().LastAction != "session archived" {
		time.Sleep(time.Millisecond)
	}
	if s.Snapshot().LastAction != "session archived" {
		t.Fatal("archive did not finish")
	}
	tt := ui.NewTester(func(c *ui.Context) { archiveUndoToast(c, Model{Store: s}) }, 600, 400)
	if !tt.HasText("Session archived") {
		t.Fatal("archive toast missing")
	}
	if err := tt.Click("Undo archive"); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(time.Second)
	for restores.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if restores.Load() != 1 {
		t.Fatal("toast action did not call real restore")
	}
}
