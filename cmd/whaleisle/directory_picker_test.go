package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func awaitDirectoryPicker(t *testing.T, predicate func() bool) {
	t.Helper()
	until := time.Now().Add(3 * time.Second)
	for time.Now().Before(until) {
		if predicate() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("directory picker did not settle")
}

func TestDirectoryPickerRetryUsesFailedPath(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, target := range []string{"", "/unavailable"} {
		t.Run("failed-path="+target, func(t *testing.T) {
			var targetCalls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					RPCID   string
					Payload struct{ Args map[string]any }
				}
				_ = json.NewDecoder(r.Body).Decode(&q)
				path, _ := q.Payload.Args["path"].(string)
				if path == target && targetCalls.Add(1) == 1 {
					http.Error(w, "temporary scan failure", http.StatusServiceUnavailable)
					return
				}
				if path == "" {
					path = "/home"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": api.DirectoryListing{Path: path}}})
			}))
			defer srv.Close()
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
			sessions := session.New()
			sessions.Attach(client, nil)
			s := &mainState{dshClient: client}
			settled := func() bool { s.mu.Lock(); defer s.mu.Unlock(); return !s.dirPickLoading }
			s.openDirectoryPicker(sessions, nil, "workspace")
			awaitDirectoryPicker(t, settled)
			if target != "" {
				// Keep the successful home listing visible while navigation fails.
				s.requestDirectoryPicker(sessions, nil, target, "")
				awaitDirectoryPicker(t, settled)
			}
			s.mu.Lock()
			failed := s.dirPickErr != "" && s.dirPickPath == target
			retained := s.dirPickListing.Path
			s.mu.Unlock()
			if !failed || target != "" && retained != "/home" || target == "" && retained != "" {
				t.Fatalf("failed scan lost retry target or old listing: target=%q listing=%q", target, retained)
			}
			tt := ui.NewTester(func(c *ui.Context) { s.directoryPickerView(c, nil, sessions, "system", false, nil) }, 900, 700)
			if err := tt.Click("Retry"); err != nil {
				t.Fatal(err)
			}
			awaitDirectoryPicker(t, settled)
			want := target
			if want == "" {
				want = "/home"
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if targetCalls.Load() != 2 || s.dirPickErr != "" || s.dirPickListing.Path != want {
				t.Fatalf("retry used retained listing instead of attempted path: calls=%d listing=%q error=%q", targetCalls.Load(), s.dirPickListing.Path, s.dirPickErr)
			}
		})
	}
}

func TestDirectoryPickerImmediateLoadingAndClose(t *testing.T) {
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct{ RPCID string }
		_ = json.NewDecoder(r.Body).Decode(&q)
		if r.URL.Path != "/api/directoryPicker/list" {
			t.Errorf("unexpected request %s", r.URL.Path)
		}
		close(entered)
		<-r.Context().Done()
		close(canceled)
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": api.DirectoryListing{Path: "/late"}}})
	}))
	defer srv.Close()
	defer once.Do(func() { close(release) })
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	sessions := session.New()
	sessions.Attach(client, nil)
	s := &mainState{dshClient: client}
	s.openDirectoryPicker(sessions, nil, "mention")
	s.mu.Lock()
	opened := s.dirPickOpen && s.dirPickLoading && s.dirPickMode == "mention"
	s.mu.Unlock()
	if !opened {
		t.Fatal("directory picker waits for its first listing before opening")
	}
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-entered:
			return true
		default:
			return false
		}
	})
	s.closeDirectoryPicker(nil)
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-canceled:
			return true
		default:
			return false
		}
	})
	once.Do(func() { close(release) })
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirPickOpen || s.dirPickLoading || s.dirPickListing.Path != "" || s.dirPickCancel != nil {
		t.Fatal("closed directory picker retained request or reopened")
	}
}

func TestDirectoryPickerSupersededNavigationKeepsLatestListing(t *testing.T) {
	entered, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			RPCID   string
			Payload struct{ Args map[string]any }
		}
		_ = json.NewDecoder(r.Body).Decode(&q)
		path, _ := q.Payload.Args["path"].(string)
		if path == "" {
			close(entered)
			<-r.Context().Done()
			close(canceled)
			<-release
			path = "/superseded"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": api.DirectoryListing{Path: path}}})
	}))
	defer srv.Close()
	defer once.Do(func() { close(release) })
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	sessions := session.New()
	sessions.Attach(client, nil)
	s := &mainState{dshClient: client}
	s.openDirectoryPicker(sessions, nil, "workspace")
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-entered:
			return true
		default:
			return false
		}
	})
	s.requestDirectoryPicker(sessions, nil, "/fresh", "")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickListing.Path == "/fresh"
	})
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-canceled:
			return true
		default:
			return false
		}
	})
	once.Do(func() { close(release) })
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirPickOpen || s.dirPickListing.Path != "/fresh" || s.dirPickErr != "" {
		t.Fatal("superseded directory scan overwrote its successor")
	}
}

func TestDirectoryPickerListingFailureAndCreateNavigation(t *testing.T) {
	created := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			RPCID   string
			Payload struct{ Args map[string]any }
		}
		_ = json.NewDecoder(r.Body).Decode(&q)
		value := any(api.DirectoryListing{Path: "/home", Entries: []api.DirectoryListingCrumbsItem{{Name: "project", Path: "/home/project"}}})
		if r.URL.Path == "/api/directoryPicker/createDirectory" {
			if q.Payload.Args["path"] != "/home" || q.Payload.Args["name"] != "child" {
				t.Errorf("wrong create target %+v", q.Payload.Args)
			}
			created <- struct{}{}
			value = "/home/child"
		} else if path, _ := q.Payload.Args["path"].(string); path == "/denied" {
			http.Error(w, "directory access denied", http.StatusForbidden)
			return
		} else if path != "" {
			value = api.DirectoryListing{Path: path}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer srv.Close()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	sessions := session.New()
	sessions.Attach(client, nil)
	s := &mainState{dshClient: client}
	settled := func() bool { s.mu.Lock(); defer s.mu.Unlock(); return !s.dirPickLoading }
	s.openDirectoryPicker(sessions, nil, "")
	awaitDirectoryPicker(t, settled)
	s.requestDirectoryPicker(sessions, nil, "/denied", "")
	awaitDirectoryPicker(t, settled)
	s.mu.Lock()
	if !s.dirPickOpen || !strings.Contains(s.dirPickErr, "directory access denied") || s.dirPickListing.Path != "/home" {
		t.Fatalf("failed navigation lost previous listing or error: %+v %s", s.dirPickListing, s.dirPickErr)
	}
	s.mu.Unlock()
	s.requestDirectoryPicker(sessions, nil, "/home", "child")
	awaitDirectoryPicker(t, settled)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(created) != 1 || s.dirPickListing.Path != "/home/child" || s.dirPickErr != "" {
		t.Fatalf("create did not select returned Host directory: %+v %s", s.dirPickListing, s.dirPickErr)
	}
}

func TestDirectoryPickerHostChangeIgnoresOldListing(t *testing.T) {
	entered, release, responded := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct{ RPCID string }
		_ = json.NewDecoder(r.Body).Decode(&q)
		close(entered)
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": map[string]any{"ok": true, "value": api.DirectoryListing{Path: "/old-host"}}})
		close(responded)
	}))
	defer srv.Close()
	defer once.Do(func() { close(release) })
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	sessions := session.New()
	sessions.Attach(client, nil)
	s := &mainState{dshClient: client}
	s.openDirectoryPicker(sessions, nil, "workspace")
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-entered:
			return true
		default:
			return false
		}
	})
	sessions.Attach(&dsh.Client{}, nil)
	once.Do(func() { close(release) })
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-responded:
			return true
		default:
			return false
		}
	})
	// Rendering detects the changed Host and cancels that open's authority.
	ui.NewTester(func(c *ui.Context) { s.directoryPickerView(c, nil, sessions, "system", false, nil) }, 900, 700)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirPickOpen || s.dirPickListing.Path != "" || s.dirPickLoading {
		t.Fatal("replacement Host inherited prior listing")
	}
}

func TestDirectoryPickerModalBoundsAndEscape(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, size := range [][2]int{{1000, 700}, {420, 300}} {
		s := &mainState{dirPickOpen: true, dirPickLoading: true}
		sessions := session.New()
		tt := ui.NewTester(func(c *ui.Context) { s.directoryPickerView(c, nil, sessions, "system", false, nil) }, size[0], size[1])
		closeButton, ok := tt.Find(i18n.T(i18n.KeyDirPickClose))
		if !ok || closeButton.X < 24 || closeButton.Y < 24 || closeButton.X+closeButton.W > float32(size[0]-24) || closeButton.Y+closeButton.H > float32(size[1]-24) {
			t.Fatalf("close escaped viewport budget at %v: %+v", size, closeButton)
		}
		if tt.HasText(i18n.T(i18n.KeyDirPickEmpty)) {
			t.Fatal("loading dialog presents a false empty listing")
		}
		tt.Key(0, ui.KeyEscape)
		if s.dirPickOpen || s.dirPickLoading {
			t.Fatal("Escape failed to dismiss loading directory picker")
		}
	}
}
