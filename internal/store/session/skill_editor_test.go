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

// One fake Host fixture only: no user files, skills, commands, or installations.
func TestSkillEditorFieldsCoreFixture(t *testing.T) {
	var mu sync.Mutex
	path, when := "/fake/project/.dsh/skills/alpha/SKILL.md", "old condition"
	doc := api.SkillInventoryDetail{Name: "alpha", Source: "project-dsh", Path: &path, Writable: true, Description: "old description", Content: "# fake body\n", WhenToUse: &when, Groups: []string{"old"}, ModelInvocable: true, UserInvocable: true}
	var writes []map[string]any
	unknown := false
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
		mu.Lock()
		defer mu.Unlock()
		switch {
		case strings.HasSuffix(r.URL.Path, "/skillInventory/list"):
			rpcOK(w, body.RPCID, map[string]any{"skills": []api.SkillInventorySnapshotSkillsItem{{Name: doc.Name, Source: doc.Source, Path: doc.Path, Writable: doc.Writable, Description: doc.Description, WhenToUse: doc.WhenToUse, Groups: doc.Groups, ModelInvocable: doc.ModelInvocable, UserInvocable: doc.UserInvocable}}})
		case strings.HasSuffix(r.URL.Path, "/skillInventory/get"):
			rpcOK(w, body.RPCID, doc)
		case strings.HasSuffix(r.URL.Path, "/skillInventory/update"):
			writes = append(writes, req)
			if unknown {
				http.Error(w, "fake lost result", 500)
				return
			}
			doc.Description = req["description"].(string)
			doc.Content = req["content"].(string)
			doc.ModelInvocable = req["modelInvocable"].(bool)
			doc.UserInvocable = req["userInvocable"].(bool)
			doc.WhenToUse = nil
			if v, ok := req["whenToUse"].(string); ok {
				doc.WhenToUse = &v
			}
			doc.Groups = nil
			for _, g := range req["groups"].([]any) {
				doc.Groups = append(doc.Groups, g.(string))
			}
			rpcOK(w, body.RPCID, nil)
		case strings.HasSuffix(r.URL.Path, "/skills/list"):
			rpcOK(w, body.RPCID, map[string]any{"skills": []any{}})
		default:
			t.Errorf("unexpected RPC %s", r.URL.Path)
			rpcOK(w, body.RPCID, nil)
		}
	}))
	defer srv.Close()
	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.selectedID = "fake-session"
	st.cwd = "/fake/project"
	st.mu.Unlock()
	wait := func(done func() bool) {
		t.Helper()
		until := time.Now().Add(3 * time.Second)
		for !done() {
			if time.Now().After(until) {
				t.Fatal("fixture timeout")
			}
			time.Sleep(time.Millisecond)
		}
	}
	refresh := func() {
		st.RefreshSkillCatalog(context.Background())
		wait(func() bool {
			st.mu.Lock()
			defer st.mu.Unlock()
			return len(st.skillCatalog.Skills) == 1 && !st.skillCatalogLoading
		})
	}
	load := func() SkillEditTarget {
		t.Helper()
		refresh()
		target, err := st.CaptureCatalogSkillEdit("alpha")
		if err != nil {
			t.Fatal(err)
		}
		target, err = st.LoadCatalogSkillEdit(context.Background(), target)
		if err != nil {
			t.Fatal(err)
		}
		return target
	}
	count := func() int { mu.Lock(); defer mu.Unlock(); return len(writes) }
	target := load()
	invalid := target.Fields()
	invalid.Content = " "
	if err := st.SaveCatalogSkillEditor(context.Background(), target, invalid); err == nil || count() != 0 || !st.SkillEditorCurrent(target) {
		t.Fatal("validation must not write or consume valid draft")
	}
	fields := target.Fields()
	fields.Description = "new description"
	fields.WhenToUse = "when A\nand B"
	fields.Content = strings.Repeat("fake inert data\n", 220)
	fields.Groups = nil
	fields.ModelInvocable = false
	fields.UserInvocable = false
	if count() != 0 {
		t.Fatal("reading or drafting wrote")
	}
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err != nil {
		t.Fatal(err)
	}
	if count() != 1 {
		t.Fatal("save must write once")
	}
	mu.Lock()
	req := writes[0]
	mu.Unlock()
	if req["modelInvocable"] != false || req["userInvocable"] != false || req["whenToUse"] != "when A\nand B" || req["content"] != fields.Content || req["sessionId"] != "fake-session" || req["cwd"] != "/fake/project" || req["name"] != "alpha" {
		t.Fatalf("lost fields or target: %#v", req)
	}
	if groups, ok := req["groups"].([]any); !ok || len(groups) != 0 {
		t.Fatalf("clear groups must be []: %#v", req)
	}
	if req["root"] != nil || req["path"] != nil || req["source"] != nil {
		t.Fatal("update must use pinned Host name/scope contract, never retarget root")
	}
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil || count() != 1 {
		t.Fatal("consumed revision wrote again")
	}
	wait(func() bool {
		c := st.Snapshot().SkillCatalog
		return len(c.Skills) == 1 && !c.Skills[0].UserInvocable && c.Skills[0].WhenToUse != nil
	})
	target = load()
	fields = target.Fields()
	fields.WhenToUse = "  "
	fields.ModelInvocable = true
	fields.UserInvocable = true
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	req = writes[len(writes)-1]
	mu.Unlock()
	if _, ok := req["whenToUse"]; ok {
		t.Fatal("clearing whenToUse must omit it per Host renderer")
	}
	target = load()
	fields = target.Fields()
	mu.Lock()
	doc.Content = "external edit"
	mu.Unlock()
	before := count()
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil || count() != before {
		t.Fatal("changed source revision wrote")
	}
	target = load()
	fields = target.Fields()
	mu.Lock()
	changedPath := "/fake/other/SKILL.md"
	doc.Path = &changedPath
	mu.Unlock()
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil || count() != before {
		t.Fatal("changed source path wrote")
	}
	mu.Lock()
	doc.Path = &path
	mu.Unlock()
	target = load()
	fields = target.Fields()
	st.SetWorkspaceCwd("/fake/other")
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil || count() != before {
		t.Fatal("changed workspace wrote")
	}
	st.SetWorkspaceCwd("/fake/project")
	target = load()
	fields = target.Fields()
	st.mu.Lock()
	st.selectedID = "other-session"
	st.mu.Unlock()
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil || count() != before {
		t.Fatal("changed session wrote")
	}
	st.mu.Lock()
	st.selectedID = "fake-session"
	st.mu.Unlock()
	target = load()
	fields = target.Fields()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil || count() != before {
		t.Fatal("changed Host wrote")
	}
	target = load()
	fields = target.Fields()
	refresh()
	if _, err := st.LoadCatalogSkillEdit(context.Background(), target); err == nil {
		t.Fatal("stale detail read accepted")
	}
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil || count() != before {
		t.Fatal("refreshed catalog revision wrote")
	}
	target = load()
	fields = target.Fields()
	mu.Lock()
	unknown = true
	mu.Unlock()
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil {
		t.Fatal("unknown result reported success")
	}
	if count() != before+1 {
		t.Fatal("unknown write retried")
	}
	if err := st.SaveCatalogSkillEditor(context.Background(), target, fields); err == nil || count() != before+1 {
		t.Fatal("unknown write allows blind retry")
	}
	if fields.Content == "" {
		t.Fatal("failed save discarded draft")
	}
	mu.Lock()
	doc.Writable = false
	mu.Unlock()
	refresh()
	readonly, err := st.CaptureCatalogSkillEdit("alpha")
	if err != nil {
		t.Fatal(err)
	}
	readonly, err = st.LoadCatalogSkillEdit(context.Background(), readonly)
	if err != nil || readonly.Fields().Content == "" {
		t.Fatal("read-only inspection regressed", err)
	}
	if st.SkillEditorCurrent(readonly) || readonly.Writable() {
		t.Fatal("read-only skill became editable")
	}
	if err := st.SaveCatalogSkillEditor(context.Background(), readonly, readonly.Fields()); err == nil || count() != before+1 {
		t.Fatal("read-only save wrote")
	}
	if len(flattenSkillInventory(api.SkillInventorySnapshot{Skills: []api.SkillInventorySnapshotSkillsItem{{Name: "hidden", UserInvocable: false}, {Name: "visible", UserInvocable: true}}})) != 1 {
		t.Fatal("user invocation flag ignored by composer fallback")
	}
}
