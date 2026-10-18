package projectflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectMaterialEditing(t *testing.T) {
	fail := false
	writes := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Method, RPCID string
			Payload       map[string]any
		}
		json.NewDecoder(r.Body).Decode(&q)
		if r.URL.Path != "/dsh-project/"+q.Method || q.Payload["projectId"] != "p" {
			t.Error("wrong route or project")
		}
		path := q.Payload["path"].(string)
		text := "existing"
		if q.Method == "store/write" {
			writes++
			text = q.Payload["text"].(string)
		} else if q.Method != "store/read" {
			t.Error("invented RPC")
		}
		if fail {
			json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": false, "error": map[string]any{"message": "archived"}}})
			return
		}
		v := map[string]any{"path": path, "text": text}
		if path == "notes.md" {
			v["text"] = "generated" + NotesMarker + text
			v["generatedText"] = "generated"
			v["userText"] = text
		}
		json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": v}})
	}))
	defer srv.Close()
	c := Client{Origin: srv.URL}
	ctx := context.Background()
	drafts := map[string]*MaterialDraft{}
	for _, path := range []string{"notes.md", "preferences.md"} {
		d := &MaterialDraft{ProjectID: "p", Path: path}
		drafts[path] = d
		m, e := c.ReadMaterial(ctx, "p", path)
		d.Apply(m, e)
		if e != nil || d.Draft != "existing" {
			t.Fatal(d, e)
		}
		d.Draft = "user edit"
		fail = true
		m, e = c.WriteMaterial(ctx, "p", path, d.Draft)
		d.Apply(m, e)
		if e == nil || d.Draft != "user edit" || !d.Dirty() {
			t.Fatal("failed save lost draft")
		}
		fail = false
		m, e = c.WriteMaterial(ctx, "p", path, d.Draft)
		d.Apply(m, e)
		if e != nil || d.Dirty() {
			t.Fatal(d, e)
		}
	}
	before := writes
	if _, e := c.WriteMaterial(ctx, "p", "notes.md", NotesMarker); e == nil || writes != before {
		t.Fatal("generated notes writable")
	}
	if drafts["notes.md"] == drafts["preferences.md"] {
		t.Fatal("draft identities collided")
	}
}
