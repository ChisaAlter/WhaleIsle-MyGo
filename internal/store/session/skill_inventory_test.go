package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
				if req["root"] != "user-dsh" || req["cwd"] != "/authorized/project" || req["sessionId"] != "s1" {
					t.Errorf("invalid skill target: %#v", req)
				}
				searched, _ = req["query"].(string)
			}
			rpcOK(w, body["rpcId"], []map[string]any{{
				"description":        "d",
				"namespace":          "n",
				"path":               "skills/x",
				"repo":               "org/repo",
				"skillName":          "cool",
				"stars":              3.0,
				"installationStatus": "available",
			}})
		case strings.HasSuffix(r.URL.Path, "/skillInventory/installHub"):
			payload, _ := body["payload"].(map[string]any)
			args, _ := payload["args"].(map[string]any)
			req, _ := args["request"].(map[string]any)
			if req != nil {
				if req["root"] != "user-dsh" || req["cwd"] != "/authorized/project" || req["sessionId"] != "s1" {
					t.Errorf("invalid skill target: %#v", req)
				}
				installed, _ = req["skillName"].(string)
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"status": "installed", "repo": "org/repo", "path": "skills/x",
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
	st.SetWorkspaceCwd("/authorized/project")
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
	if len(hits) != 1 || hits[0].SkillName != "cool" || hits[0].Status != "available" {
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
	if got := st.Snapshot().SkillHubHits; len(got) != 1 || got[0].Status != "installed" || !got[0].Installed {
		t.Fatalf("installation status not folded: %#v", got)
	}
	if installed != "cool" {
		t.Fatalf("installed=%q", installed)
	}
}

// The HTTP fixture checks the actual serialized create request without starting
// a Host, creating any skill directory, or installing remote content.
func TestCreateSkillInventoryWireTargets(t *testing.T) {
	for _, root := range []string{"user", "project"} {
		t.Run(root, func(t *testing.T) {
			requests := make(chan map[string]any, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if strings.HasSuffix(r.URL.Path, "/skillInventory/create") {
					payload, _ := body["payload"].(map[string]any)
					args, _ := payload["args"].(map[string]any)
					request, _ := args["request"].(map[string]any)
					requests <- request
				}
				rpcOK(w, body["rpcId"], map[string]any{})
			}))
			defer srv.Close()
			st := New()
			st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
			st.SetWorkspaceCwd("/authorized/project")
			st.CreateSkillInventory(context.Background(), "new-skill", "description", "body", root)
			select {
			case request := <-requests:
				if request["root"] != root+"-dsh" || request["cwd"] != "/authorized/project" || request["name"] != "new-skill" {
					t.Fatalf("create target=%#v", request)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("create request was not sent to the local fixture")
			}
		})
	}
}
