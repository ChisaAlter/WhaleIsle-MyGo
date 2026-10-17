package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
)

func TestSearchAndInstallSkillHub(t *testing.T) {
	var searched, installed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/skillInventory/searchHub"):
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				searched, _ = req["query"].(string)
			}
			rpcOK(w, body["rpcId"], []map[string]any{{
				"description": "d",
				"namespace":   "n",
				"path":        "skills/x",
				"repo":        "org/repo",
				"skillName":   "cool",
				"stars":       3.0,
			}})
		case strings.HasSuffix(r.URL.Path, "/skillInventory/installHub"):
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				installed, _ = req["skillName"].(string)
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"status": "ok", "repo": "org/repo", "path": "skills/x",
				"skillName": "cool", "commit": "abc", "directory": "/tmp",
			})
		case strings.HasSuffix(r.URL.Path, "/skills/list"):
			rpcOK(w, body["rpcId"], map[string]any{"skills": []any{}})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	done := make(chan struct{})
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		if len(st.Snapshot().SkillHubHits) > 0 {
			select {
			case <-done:
			default:
				close(done)
			}
		}
	})
	st.Select("s1")
	st.SearchSkillHub(context.Background(), "cool", "user")
	wait(t, done)
	if searched != "cool" {
		t.Fatalf("query=%q", searched)
	}
	hits := st.Snapshot().SkillHubHits
	if len(hits) != 1 || hits[0].SkillName != "cool" {
		t.Fatalf("%#v", hits)
	}
	instDone := make(chan struct{})
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, func() {
		if strings.Contains(st.Snapshot().LastAction, "skill installed") {
			select {
			case <-instDone:
			default:
				close(instDone)
			}
		}
	})
	st.Select("s1")
	st.InstallSkillHub(context.Background(), hits[0], "user")
	wait(t, instDone)
	if installed != "cool" {
		t.Fatalf("installed=%q", installed)
	}
}
