package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/dirpick"
	"whaleisle/internal/ui/i18n"
)

func TestDirectoryPickerRemoteTabPreservesLocalLanding(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	cfg, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sessions := session.New()
	root := browserRoot()
	s := &mainState{dirPickOpen: true, dirPickMode: "workspace", cfgStore: cfg, remoteMachines: remotestore.New()}
	s.dirPickBrowser.Parent = &root
	s.dirPickBrowser.ShowHidden = true
	tt := ui.NewTester(func(c *ui.Context) { s.directoryPickerView(c, nil, sessions, "system", false, nil) }, 900, 700)
	if err := tt.Click("Remote"); err != nil {
		t.Fatal(err)
	}
	if !s.dirPickBrowser.Remote || !s.dirPickBrowser.ShowHidden || s.dirPickBrowser.Parent != &root {
		t.Fatal("remote page discarded local directory state")
	}
	saveSettingsSectionEvidence(t, tt, "workspace-remote-tab")
	if err := tt.Click("Local"); err != nil {
		t.Fatal(err)
	}
	if s.dirPickBrowser.Remote || s.dirPickBrowser.Parent != &root || !s.dirPickBrowser.ShowHidden {
		t.Fatal("local page did not restore its original landing")
	}
}

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
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickBrowser.Creating && !s.dirPickLoading
	})
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
		closeButton, ok := tt.Find("Cancel")
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

func directoryBrowserFixture(t *testing.T, handle func(string, map[string]any) (any, error)) (*mainState, *session.Store) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			RPCID   string
			Payload struct{ Args map[string]any }
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			t.Error(err)
			return
		}
		method := strings.TrimPrefix(r.URL.Path, "/api/")
		if method != "directoryPicker/list" && method != "directoryPicker/createDirectory" {
			t.Errorf("unexpected endpoint %s", method)
		}
		value, err := handle(method, q.Payload.Args)
		result := map[string]any{"ok": true, "value": value}
		if err != nil {
			result = map[string]any{"ok": false, "error": map[string]any{"code": "directory-picker/unreadable", "message": err.Error()}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": q.RPCID, "result": result})
	}))
	t.Cleanup(srv.Close)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	sessions := session.New()
	sessions.Attach(client, nil)
	return &mainState{dshClient: client}, sessions
}

func browserRoot() api.DirectoryListing {
	return api.DirectoryListing{Path: "/", Home: "/", Crumbs: []api.DirectoryListingCrumbsItem{{Name: "/", Path: "/"}}, Entries: []api.DirectoryListingCrumbsItem{{Name: "Project", Path: "/project"}, {Name: "Other", Path: "/other"}}}
}

func browserProject() api.DirectoryListing {
	return api.DirectoryListing{Path: "/project", Home: "/", Crumbs: []api.DirectoryListingCrumbsItem{{Name: "/", Path: "/"}, {Name: "Project", Path: "/project"}}, Entries: []api.DirectoryListingCrumbsItem{{Name: "Deep", Path: "/project/deep"}}}
}

func TestDirectoryBrowserMillerSelectAdvanceAndAnchoredNavigation(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s, sessions := directoryBrowserFixture(t, func(method string, args map[string]any) (any, error) {
		path, _ := args["path"].(string)
		if path == "/project" {
			once.Do(func() { close(entered); <-release })
			return browserProject(), nil
		}
		if path == "/project/deep" {
			return api.DirectoryListing{Path: path, Home: "/", Crumbs: []api.DirectoryListingCrumbsItem{{Path: "/"}, {Path: "/project"}, {Path: path}}}, nil
		}
		return browserRoot(), nil
	})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	s.openDirectoryPicker(sessions, nil, "workspace")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Parent != nil
	})
	s.selectDirectoryPicker(sessions, nil, api.DirectoryListingCrumbsItem{Name: "Project", Path: "/project"}, false)
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-entered:
			return true
		default:
			return false
		}
	})
	s.mu.Lock()
	immediate := s.dirPickLoading && s.dirPickBrowser.Selected != nil && s.dirPickBrowser.Selected.Path == "/project" && s.dirPickBrowser.Child == nil && s.dirPickBrowser.Parent.Path == "/"
	s.mu.Unlock()
	if !immediate {
		t.Fatal("left pick did not immediately split and acknowledge selection")
	}
	close(release)
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.dirPickBrowser.Child != nil && s.dirPickBrowser.Child.Path == "/project"
	})
	s.selectDirectoryPicker(sessions, nil, api.DirectoryListingCrumbsItem{Name: "Deep", Path: "/project/deep"}, true)
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Child != nil && s.dirPickBrowser.Child.Path == "/project/deep"
	})
	s.mu.Lock()
	advanced := s.dirPickBrowser.Parent.Path == "/project" && s.dirPickBrowser.Selected.Path == "/project/deep"
	s.mu.Unlock()
	if !advanced {
		t.Fatal("right pick did not move child into parent column")
	}
	s.requestDirectoryPicker(sessions, nil, "/project", "")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Parent.Path == "/" && s.dirPickBrowser.Child != nil && s.dirPickBrowser.Child.Path == "/project"
	})
}

func TestDirectoryBrowserParentWaitAndLateUpgrade(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s, sessions := directoryBrowserFixture(t, func(method string, args map[string]any) (any, error) {
		if args["path"] == "/" {
			once.Do(func() { close(entered); <-release })
			return browserRoot(), nil
		}
		return browserProject(), nil
	})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	s.openDirectoryPicker(sessions, nil, "workspace")
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-entered:
			return true
		default:
			return false
		}
	})
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Parent != nil && s.dirPickBrowser.Parent.Path == "/project" && s.dirPickBrowser.Selected == nil
	})
	close(release)
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.dirPickBrowser.Parent.Path == "/" && s.dirPickBrowser.Selected != nil && s.dirPickBrowser.Child != nil
	})
}

func TestDirectoryBrowserTypedPreviewWaitsBothLegsAndCancelKeepsLanding(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	s, sessions := directoryBrowserFixture(t, func(method string, args map[string]any) (any, error) {
		path, _ := args["path"].(string)
		if path == "/project/" {
			return browserProject(), nil
		}
		if path == "/" {
			once.Do(func() { close(entered); <-release })
		}
		return browserRoot(), nil
	})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	s.openDirectoryPicker(sessions, nil, "workspace")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Parent != nil
	})
	s.beginDirectoryPath(sessions, nil)
	s.editDirectoryPath(sessions, nil, "/project/De")
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-entered:
			return true
		default:
			return false
		}
	})
	// A speculative preview has no200ms single-pane fallback; holding the
	// parent beyond that bound must preserve the previous readable pane.
	timer := time.NewTimer(230 * time.Millisecond)
	<-timer.C
	s.mu.Lock()
	held := s.dirPickLoading && s.dirPickBrowser.Parent.Path == "/" && s.dirPickBrowser.Child == nil
	s.mu.Unlock()
	if !held {
		t.Fatal("draft preview flashed target alone before parent leg")
	}
	close(release)
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Child != nil
	})
	s.mu.Lock()
	prefix, ok := s.dirPickBrowser.Prefix()
	retained := s.dirPickBrowser.PathDraft != nil && *s.dirPickBrowser.PathDraft == "/project/De"
	s.mu.Unlock()
	if !retained || !ok || prefix != "De" {
		t.Fatal("preview lost draft or canonicalized prefix binding")
	}
	s.cancelDirectoryPath(sessions, nil)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirPickBrowser.PathDraft != nil || s.dirPickBrowser.Child == nil || s.dirPickBrowser.Child.Path != "/project" {
		t.Fatal("path cancellation undid already-landed panes")
	}
}

func TestDirectoryBrowserCreateFailureRetryRawNameAndLanding(t *testing.T) {
	const name = " child "
	var creates atomic.Int32
	s, sessions := directoryBrowserFixture(t, func(method string, args map[string]any) (any, error) {
		if method == "directoryPicker/createDirectory" {
			if args["path"] != "/" || args["name"] != name {
				t.Errorf("create changed raw Host arguments %+v", args)
			}
			if creates.Add(1) == 1 {
				return nil, fmt.Errorf("the folder could not be created")
			}
			return "/ child ", nil
		}
		if args["path"] == "/ child " {
			return api.DirectoryListing{Path: "/ child ", Home: "/"}, nil
		}
		return browserRoot(), nil
	})
	s.openDirectoryPicker(sessions, nil, "workspace")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Parent != nil
	})
	s.createDirectoryPicker(sessions, nil, "/", name)
	awaitDirectoryPicker(t, func() bool { s.mu.Lock(); defer s.mu.Unlock(); return !s.dirPickBrowser.Creating })
	s.mu.Lock()
	failed := s.dirPickBrowser.FolderDraft != nil && *s.dirPickBrowser.FolderDraft == name && s.dirPickBrowser.CreateError == "the folder could not be created" && s.dirPickErr == ""
	s.mu.Unlock()
	if !failed {
		t.Fatal("create failure lost raw draft, Host message or nested ownership")
	}
	s.createDirectoryPicker(sessions, nil, "/", name)
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Child != nil && s.dirPickBrowser.Child.Path == "/ child "
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if creates.Load() != 2 || s.dirPickBrowser.Parent.Path != "/" || s.dirPickBrowser.Selected.Name != name || s.dirPickBrowser.FolderDraft != nil || s.dirPickBrowser.Creating {
		t.Fatal("create retry did not relist parent and select created child")
	}
}

func TestDirectoryBrowserEnteredPathPreservesWhitespaceAndErrorsStaySilentForPreview(t *testing.T) {
	var entered atomic.Bool
	s, sessions := directoryBrowserFixture(t, func(method string, args map[string]any) (any, error) {
		path, _ := args["path"].(string)
		if path == "/trailing " {
			entered.Store(true)
			return api.DirectoryListing{Path: path, Home: "/"}, nil
		}
		if path == "/denied/" {
			return nil, fmt.Errorf("preview unavailable")
		}
		if path != "" {
			t.Errorf("unexpected rewritten path %q", path)
		}
		return browserRoot(), nil
	})
	s.openDirectoryPicker(sessions, nil, "workspace")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Parent != nil
	})
	s.beginDirectoryPath(sessions, nil)
	s.editDirectoryPath(sessions, nil, "/denied/name")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.dirPickBrowser.ScanWindow >= 2 && !s.dirPickLoading
	})
	s.mu.Lock()
	silent := s.dirPickErr == "" && s.dirPickBrowser.Parent.Path == "/" && s.dirPickBrowser.PathDraft != nil
	s.mu.Unlock()
	if !silent {
		t.Fatal("failed speculative path scan replaced panes or surfaced an error")
	}
	s.requestDirectoryPicker(sessions, nil, "/trailing ", "")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickListing.Path == "/trailing "
	})
	if !entered.Load() {
		t.Fatal("submitted path was trimmed before Host")
	}
}

func TestDirectoryBrowserClosedCreationCannotRepopulateReopenedFlow(t *testing.T) {
	entered, release, responded := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	var lists atomic.Int32
	s, sessions := directoryBrowserFixture(t, func(method string, args map[string]any) (any, error) {
		if method == "directoryPicker/createDirectory" {
			close(entered)
			<-release
			defer close(responded)
			return "/late-created", nil
		}
		lists.Add(1)
		if args["path"] != nil {
			t.Errorf("old create issued follow-up after close: %+v", args)
		}
		return browserRoot(), nil
	})
	defer once.Do(func() { close(release) })
	s.openDirectoryPicker(sessions, nil, "workspace")
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Parent != nil
	})
	s.mu.Lock()
	s.dirPickBrowser.ShowHidden = true
	s.mu.Unlock()
	s.createDirectoryPicker(sessions, nil, "/", "late")
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-entered:
			return true
		default:
			return false
		}
	})
	s.closeDirectoryPicker(nil)
	s.openDirectoryPicker(sessions, nil, "mention")
	once.Do(func() { close(release) })
	awaitDirectoryPicker(t, func() bool {
		select {
		case <-responded:
			return true
		default:
			return false
		}
	})
	awaitDirectoryPicker(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return !s.dirPickLoading && s.dirPickBrowser.Parent != nil
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirPickOpen || s.dirPickMode != "mention" || s.dirPickBrowser.ShowHidden || s.dirPickBrowser.Creating || s.dirPickBrowser.FolderDraft != nil || s.dirPickBrowser.Selected != nil || lists.Load() != 2 {
		t.Fatal("old creation contaminated freshly reopened browser")
	}
}

func TestDirectoryBrowserRenderMergePreservesSameRequestCommit(t *testing.T) {
	root := browserRoot()
	draft := "original draft"
	s := &mainState{dirPickOpen: true}
	s.dirPickBrowser.Parent = &root
	s.dirPickBrowser.FolderDraft = &draft
	s.dirPickBrowser.FocusEdit = true
	sessions := session.New()
	op := s.beginDirectoryScan(sessions, nil)
	if op == nil {
		t.Fatal("could not capture picker scope")
	}
	defer op.finish()
	s.mu.Lock()
	before := s.dirPickBrowser.Clone()
	s.mu.Unlock()
	edited := before.Clone()
	edited.ShowHidden = true
	*edited.FolderDraft = "typed in stale frame"
	edited.FocusEdit = false
	child := browserProject()
	if !op.commit(func(b *dirpick.BrowserState) {
		b.Child = &child
		b.Creating = true
		b.CreateError = "new Host failure"
		b.FocusEdit = true
		s.dirPickLoading = false
	}) {
		t.Fatal("same-request completion rejected")
	}
	s.mergeDirectoryPickerUI(op.request, before, edited)
	s.mu.Lock()
	retained := s.dirPickBrowser.Child == &child && s.dirPickBrowser.Creating && s.dirPickBrowser.CreateError == "new Host failure" && s.dirPickBrowser.FocusEdit && *s.dirPickBrowser.FolderDraft == "original draft" && s.dirPickBrowser.ShowHidden
	s.mu.Unlock()
	if !retained {
		t.Fatal("render merge lost same-request Host state or UI toggle")
	}
	// Once editable, canceling the nested draft is a UI action, not a stale
	// replacement of the freshly committed panes or error state.
	op.commit(func(b *dirpick.BrowserState) { b.Creating = false })
	s.mu.Lock()
	before = s.dirPickBrowser.Clone()
	s.mu.Unlock()
	edited = before.Clone()
	edited.FolderDraft = nil
	edited.CreateError = ""
	edited.PathHadFocus = true
	s.mergeDirectoryPickerUI(op.request, before, edited)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirPickBrowser.FolderDraft != nil || s.dirPickBrowser.CreateError != "" || s.dirPickBrowser.Child != &child || !s.dirPickBrowser.PathHadFocus {
		t.Fatal("UI nested cancel did not merge independently of Host panes")
	}
}

func TestDirectoryBrowserEditorEscapeMaskAndNestedEscapeOwnership(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	root := browserRoot()
	draft := "/"
	s := &mainState{dirPickOpen: true}
	s.dirPickBrowser.Parent = &root
	s.dirPickBrowser.PathDraft = &draft
	s.dirPickBrowser.FocusPath = true
	sessions := session.New()
	tt := ui.NewTester(func(c *ui.Context) { s.directoryPickerView(c, nil, sessions, "system", false, nil) }, 900, 700)
	if !s.dirPickOpen || s.dirPickBrowser.PathDraft == nil {
		t.Fatalf("editor lost during initial modal focus: open=%v browser=%+v", s.dirPickOpen, s.dirPickBrowser)
	}
	tt.Key(0, ui.KeyEscape)
	s.mu.Lock()
	editorClosed := s.dirPickOpen && s.dirPickBrowser.PathDraft == nil
	s.mu.Unlock()
	if !editorClosed {
		t.Fatalf("Escape closed browser instead of its path editor: open=%v browser=%+v", s.dirPickOpen, s.dirPickBrowser)
	}
	if err := tt.Click("Edit path"); err != nil {
		t.Fatal(err)
	}
	tt.ClickAt(4, 4)
	if s.dirPickOpen {
		t.Fatal("mask click canceled only the path instead of closing browser")
	}
	s.mu.Lock()
	s.dirPickOpen = true
	s.dirPickBrowser.FolderDraft = &draft
	s.mu.Unlock()
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirPickOpen || s.dirPickBrowser.FolderDraft != nil {
		t.Fatal("nested Escape escaped both modal layers")
	}
}

func TestDirectoryPickerNativeModeAndWorkspacePersistMergeExternalConfigEdit(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Update(func(c *config.Config) { c.SidebarWidth = 280 }); err != nil {
		t.Fatal(err)
	}
	other, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Update(func(c *config.Config) { c.SidebarWidth = 312 }); err != nil {
		t.Fatal(err)
	}
	root := browserRoot()
	sessions := session.New()
	s := &mainState{dirPickOpen: true, dirPickMode: "workspace", cfgStore: cfg}
	s.dirPickBrowser.Parent = &root
	s.setDirectoryPickerNativeMode(sessions, nil, true)
	s.mu.Lock()
	useNative, errText := s.dirPickBrowser.UseNative, s.dirPickErr
	s.mu.Unlock()
	if errText != "" || !useNative || !cfg.Get().DirectoryPickerUseNative || cfg.Get().SidebarWidth != 312 {
		t.Fatalf("native mode failed or clobbered external config: useNative=%v err=%q cfg=%+v", useNative, errText, cfg.Get())
	}
	if err := other.Merge(func(c *config.Config) { c.TerminalHeight = 300 }); err != nil {
		t.Fatal(err)
	}
	if err := s.persistRegisteredWorkspace(`C:\AI\WhaleIsle-MyGo`); err != nil {
		t.Fatalf("persistRegisteredWorkspace after external config edit: %v", err)
	}
	reloaded, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Get()
	if !got.DirectoryPickerUseNative || got.Workspace != `C:\AI\WhaleIsle-MyGo` || got.SidebarWidth != 312 || got.TerminalHeight != 300 {
		t.Fatalf("merged config lost fields: %+v", got)
	}
}
