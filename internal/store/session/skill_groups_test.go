package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// A single mock fixture covers catalog fallback, writable batching, permission
// preservation, partial failure rollback, and explicit empty group persistence.
func TestSkillGroupsCoreFixture(t *testing.T) {
	var mu sync.Mutex
	var writes []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				Args struct {
					Request map[string]any `json:"request"`
				} `json:"args"`
			} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		req := body.Payload.Args.Request
		switch {
		case strings.HasSuffix(r.URL.Path, "/skillInventory/list"):
			if req["sessionId"] != nil {
				http.Error(w, "session unavailable", 500)
				return
			}
			rpcOK(w, body.RPCID, map[string]any{"skills": []api.SkillInventorySnapshotSkillsItem{
				{Name: "alpha", Groups: []string{"one", "two"}, Writable: true, ModelInvocable: true, UserInvocable: false},
				{Name: "beta", Groups: []string{"one"}, Writable: true, ModelInvocable: true, UserInvocable: true},
				{Name: "readonly", Groups: []string{"one"}, Writable: false, ModelInvocable: true},
			}})
		case strings.HasSuffix(r.URL.Path, "/skillInventory/setInvocation"), strings.HasSuffix(r.URL.Path, "/skillInventory/update"):
			mu.Lock()
			writes = append(writes, req)
			mu.Unlock()
			if req["name"] == "beta" {
				http.Error(w, "write denied", 500)
				return
			}
			rpcOK(w, body.RPCID, map[string]any{})
		default:
			rpcOK(w, body.RPCID, map[string]any{})
		}
	}))
	defer srv.Close()
	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.Select("broken")
	await := func(done func() bool) {
		t.Helper()
		until := time.Now().Add(3 * time.Second)
		for !done() {
			if time.Now().After(until) {
				t.Fatal("timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	st.RefreshSkillCatalog(context.Background())
	await(func() bool { return len(st.Snapshot().SkillCatalog.Skills) == 3 })
	if st.Snapshot().SkillCatalog.Notice == "" {
		t.Fatal("fallback notice missing")
	}
	st.SetSkillGroupInvocation(context.Background(), "one", false)
	await(func() bool { return !st.Snapshot().SkillCatalog.Busy && st.Snapshot().SkillCatalog.Action != "" })
	cat := st.Snapshot().SkillCatalog
	if cat.Skills[0].ModelInvocable || !cat.Skills[1].ModelInvocable || !cat.Skills[2].ModelInvocable || cat.Errors["beta"] == "" {
		t.Fatalf("rollback/readonly: %+v", cat)
	}
	mu.Lock()
	if len(writes) != 2 || writes[0]["userInvocable"] != false || writes[0]["sessionId"] != nil {
		t.Fatalf("requests: %+v", writes)
	}
	mu.Unlock()
	st.SaveCatalogSkill(context.Background(), api.SkillInventoryDetail{Name: "alpha", Writable: true, Groups: []string{}, Description: "edited", Content: "body"})
	await(func() bool {
		return !st.Snapshot().SkillCatalog.Busy && strings.Contains(st.Snapshot().SkillCatalog.Action, "saved")
	})
	mu.Lock()
	defer mu.Unlock()
	last := writes[len(writes)-1]
	groups, ok := last["groups"].([]any)
	if !ok || len(groups) != 0 {
		t.Fatalf("clear must send []: %#v", last)
	}
}
