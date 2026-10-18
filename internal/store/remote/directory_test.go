package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"whaleisle/internal/dsh"
)

func TestDirectoryBrowseAndExplicitMirror(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/dsh-remote/ls":
			if r.Method != "GET" || r.URL.Query().Get("path") != "/work/a & b" {
				t.Errorf("bad listing request: %s", r.URL)
			}
			_, _ = w.Write([]byte(`{"path":"/work/a & b","platform":"posix","items":[{"name":"file","path":"/work/a & b/file","type":"file"},{"name":"src","path":"/work/a & b/src","type":"dir"}]}`))
		case "/api/dsh-remote/mirror":
			calls++
			var b map[string]string
			_ = json.NewDecoder(r.Body).Decode(&b)
			if r.Method != "POST" || b["path"] != "/work/a & b" || len(b) != 1 {
				t.Errorf("bad mirror payload: %v", b)
			}
			if calls == 1 {
				_, _ = w.Write([]byte(`{"ok":true,"localMirror":"/host/mirrors/project"}`))
			} else {
				_, _ = w.Write([]byte(`{"ok":true}`))
			}
		default:
			t.Errorf("unexpected route %s", r.URL)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	s.view.CurrentID = "one"
	s.Browse("/work/a & b")
	v := waitStore(t, s)
	if !v.Directory.Loaded || v.Directory.Path != "/work/a & b" || len(v.Directory.Items) != 2 || calls != 0 {
		t.Fatalf("browse: %+v", v)
	}
	s.ChooseDirectory()
	v = waitStore(t, s)
	if v.Directory.LocalMirror != "/host/mirrors/project" || v.Directory.SelectedPath != "/work/a & b" || v.Directory.Error != "" {
		t.Fatalf("choose: %+v", v)
	}
	s.ChooseDirectory()
	v = waitStore(t, s)
	if v.Directory.Error == "" || v.Directory.LocalMirror != "/host/mirrors/project" {
		t.Fatalf("invalid reply lost selection: %+v", v)
	}
}

func TestDirectoryFailurePreservesPositionAndDisablesChoose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":"permission denied"}`))
	}))
	defer server.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	s.view.CurrentID = "one"
	s.view.Directory = DirectorySnapshot{Path: "/work", Loaded: true, LocalMirror: "/mirror", SelectedPath: "/old"}
	s.Browse("/denied")
	v := waitStore(t, s)
	if v.Directory.Error == "" || v.Directory.Path != "/work" || v.Directory.LocalMirror != "/mirror" || v.Directory.CanChoose() {
		t.Fatalf("bad error state %+v", v)
	}
}

func TestRemoteParent(t *testing.T) {
	for in, want := range map[string]string{"/": "/", "/home": "/", "/home/project": "/home", `C:\`: "", `C:\Users`: `C:\`, `C:\Users\dev`: `C:\Users`} {
		if got := ParentDirectory(in); got != want {
			t.Errorf("parent %q = %q, want %q", in, got, want)
		}
	}
}

func TestMachineSwitchClearsDirectoryEvenWhenRefreshFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/dsh-remote/current" {
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"error":"registry unavailable"}`))
	}))
	defer server.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	s.view.CurrentID = "old"
	s.view.Directory = DirectorySnapshot{Path: "/old", Loaded: true, LocalMirror: "/old-mirror"}
	s.Select("new")
	v := waitStore(t, s)
	if v.Error == "" || v.Directory.Loaded || v.Directory.LocalMirror != "" {
		t.Fatalf("stale machine directory: %+v", v)
	}
}

func TestDirectoryReattachIgnoresLateListing(t *testing.T) {
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = w.Write([]byte(`{"path":"/stale","items":[]}`))
		close(done)
	}))
	defer server.Close()
	s := New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	s.view.CurrentID = "old"
	s.Browse("")
	<-started
	if !s.Snapshot().Busy {
		t.Fatal("missing loading state")
	}
	s.Attach(nil, nil)
	close(release)
	<-done
	v := s.Snapshot()
	if v.Busy || v.Directory.Loaded || v.Directory.Path != "" {
		t.Fatalf("reattach kept stale data: %+v", v)
	}
}
