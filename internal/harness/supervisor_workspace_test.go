package harness

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

type workspaceHandshakeTransport func(*http.Request) (*http.Response, error)

func (f workspaceHandshakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The real rc.2 Host declines workspace/initializeDefault once *any* workspace
// has been registered. A transport handshake must not pre-register the launch
// cwd, even in an otherwise fresh profile; only the UI owns that initialization.
func TestSupervisorWorkspaceIsChildCwdWithoutPreRegistering(t *testing.T) {
	workspace := t.TempDir()
	marker := filepath.Join(t.TempDir(), "child-cwd.json")
	t.Setenv("FAKE_DSH_WORKSPACE_MARKER", marker)
	s := supervisorFor(t, "serve", Options{Workspace: workspace})
	var registrations atomic.Int64
	s.client = &http.Client{Transport: workspaceHandshakeTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/workspace/create" {
			registrations.Add(1)
			var request struct {
				RPCID string `json:"rpcId"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				return nil, err
			}
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"type":"server-response","rpcId":%q,"result":{"ok":true,"value":{}}}`, request.RPCID)))}, nil
		}
		return http.DefaultTransport.RoundTrip(r)
	})}
	s.Start()
	t.Cleanup(s.Stop)
	waitFor(t, s, Ready)
	if got := registrations.Load(); got != 0 {
		t.Fatalf("handshake pre-registered the workspace %d times", got)
	}
	var child struct {
		Cwd string `json:"cwd"`
	}
	raw, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &child); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatal(err)
	}
	got, err := filepath.EvalSymlinks(child.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(filepath.Clean(got), filepath.Clean(want)) {
		t.Fatalf("child cwd %q, want configured %q", child.Cwd, workspace)
	}
}

func TestSupervisorUnavailableWorkspaceDoesNotBlockStartup(t *testing.T) {
	parent, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"missing", "file"} {
		t.Run(kind, func(t *testing.T) {
			workspace := filepath.Join(t.TempDir(), "workspace")
			if kind == "file" {
				if err := os.WriteFile(workspace, []byte("not a directory"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			marker := filepath.Join(t.TempDir(), "child-cwd.json")
			t.Setenv("FAKE_DSH_WORKSPACE_MARKER", marker)
			s := supervisorFor(t, "serve", Options{Workspace: workspace})
			s.Start()
			t.Cleanup(s.Stop)
			waitFor(t, s, Ready)
			raw, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			var child struct {
				Cwd string `json:"cwd"`
			}
			if err := json.Unmarshal(raw, &child); err != nil {
				t.Fatal(err)
			}
			got, err := filepath.EvalSymlinks(child.Cwd)
			if err != nil {
				t.Fatal(err)
			}
			want, err := filepath.EvalSymlinks(parent)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.EqualFold(filepath.Clean(got), filepath.Clean(want)) {
				t.Fatalf("child cwd %q, want original parent %q", child.Cwd, parent)
			}
			if !strings.Contains(strings.Join(s.Logs(), "\n"), "workspace unavailable") {
				t.Fatal("unavailable workspace recovery was not logged")
			}
		})
	}
}
