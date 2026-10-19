package remote

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type retainedTreeTransport func(*http.Request) (*http.Response, error)

func (f retainedTreeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func treeResponse(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
}
func retainedTreeFiles(t *testing.T, transport retainedTreeTransport) *Files {
	t.Helper()
	s := NewFiles()
	s.Bind(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://remote-tree-fixture.invalid"}, HTTP: &http.Client{Transport: transport}}, "session-tree", nil)
	v := s.states[s.key]
	v.Cwd, v.RootPath, v.Path, v.RemoteIdentity = "/work", "/work", "/work", "fixture-host\x00fixture-user\x0022\x00/work"
	v.Bound, v.Loaded = true, true
	v.Tree = map[string]FileLevel{"/work": {Loaded: true, Items: []DirectoryEntry{{Name: "retained.txt", Path: "/work/retained.txt", Type: "file"}}}}
	v.Expanded = map[string]bool{"/work": true}
	return s
}
func waitRetainedTree(t *testing.T, s *Files) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for s.Snapshot().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.Snapshot().Busy {
		t.Fatal("mock tree request did not settle")
	}
}

func TestLoadedRemoteDirectoryReopensWithRefreshAndRetainedSoftError(t *testing.T) {
	for _, outcome := range []string{"success", "soft-error"} {
		t.Run(outcome, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			var statusCalls, listCalls atomic.Int32
			s := retainedTreeFiles(t, func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" || r.URL.Query().Get("sessionId") != "session-tree" {
					return nil, fmt.Errorf("unexpected remote request owner: %s", r.URL)
				}
				switch r.URL.Path {
				case "/api/dsh-remote/status":
					statusCalls.Add(1)
					return treeResponse(r, 200, `{"sessionBound":true,"sessionMode":"remote","workspace":"/work","host":"fixture-host","username":"fixture-user","port":22}`), nil
				case "/api/dsh-remote/ls":
					listCalls.Add(1)
					if r.URL.Query().Get("path") != "/work" {
						return nil, fmt.Errorf("wrong tree path %s", r.URL)
					}
					close(entered)
					select {
					case <-release:
					case <-r.Context().Done():
						return nil, r.Context().Err()
					}
					if outcome == "soft-error" {
						return treeResponse(r, 503, `{"error":"synthetic level refresh failure"}`), nil
					}
					return treeResponse(r, 200, `{"bound":true,"path":"/work","items":[{"name":"fresh.txt","path":"/work/fresh.txt","type":"file"}]}`), nil
				default:
					return nil, fmt.Errorf("unexpected route %s", r.URL.Path)
				}
			})
			s.ToggleDirectory("/work")
			if s.Snapshot().Expanded["/work"] || statusCalls.Load() != 0 || listCalls.Load() != 0 {
				t.Fatal("collapse made a remote request")
			}
			s.ToggleDirectory("/work")
			select {
			case <-entered:
			case <-time.After(2 * time.Second):
				t.Fatal("loaded directory did not refresh when reopened")
			}
			pending := s.Snapshot()
			level := pending.Tree["/work"]
			if !pending.Busy || !level.Loading || !level.Loaded || len(level.Items) != 1 || level.Items[0].Name != "retained.txt" {
				t.Fatal("refresh discarded retained entries", pending)
			}
			s.ToggleDirectory("/work") // The existing busy owner blocks duplicate state/request changes.
			if !s.Snapshot().Expanded["/work"] || listCalls.Load() != 1 {
				t.Fatal("busy toggle changed ownership or queued another read")
			}
			unblock()
			waitRetainedTree(t, s)
			got := s.Snapshot().Tree["/work"]
			if statusCalls.Load() != 1 || listCalls.Load() != 1 || got.Loading || !got.Loaded {
				t.Fatal("tree refresh did not settle exactly once", got)
			}
			if outcome == "success" {
				if got.Error != "" || len(got.Items) != 1 || got.Items[0].Name != "fresh.txt" {
					t.Fatal("loaded level did not adopt its refresh", got)
				}
			} else if got.Error == "" || len(got.Items) != 1 || got.Items[0].Name != "retained.txt" {
				t.Fatal("soft error discarded cached level", got)
			}
		})
	}
}

func TestLoadedTreeRefreshCannotPublishAfterCapturedOwnerRetires(t *testing.T) {
	entered, release, retired := make(chan struct{}), make(chan struct{}), make(chan struct{}, 1)
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	var current atomic.Bool
	current.Store(true)
	s := retainedTreeFiles(t, func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/dsh-remote/status" {
			return treeResponse(r, 200, `{"sessionBound":true,"sessionMode":"remote","workspace":"/work","host":"fixture-host","username":"fixture-user","port":22}`), nil
		}
		if r.URL.Path != "/api/dsh-remote/ls" {
			return nil, fmt.Errorf("unexpected route %s", r.URL.Path)
		}
		close(entered)
		<-release
		return treeResponse(r, 200, `{"bound":true,"path":"/work","items":[{"name":"late.txt","path":"/work/late.txt","type":"file"}]}`), nil
	})
	s.current = func() bool {
		if current.Load() {
			return true
		}
		select {
		case retired <- struct{}{}:
		default:
		}
		return false
	}
	s.ToggleDirectory("/work")
	s.ToggleDirectory("/work")
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("refresh did not reach mock transport")
	}
	current.Store(false)
	unblock()
	select {
	case <-retired:
	case <-time.After(2 * time.Second):
		t.Fatal("completion did not check its captured session owner")
	}
	got := s.Snapshot().Tree["/work"]
	if len(got.Items) != 1 || got.Items[0].Name != "retained.txt" {
		t.Fatal("retired owner published into retained tree", got)
	}
	s.Suspend()
	if s.Snapshot().Busy || s.Snapshot().Tree["/work"].Loading {
		t.Fatal("retirement left a busy tree occurrence")
	}
}
