//go:build windows && browser_workspace_acceptance

package browserpreview

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/htmldocument"
	desktop "whaleisle/internal/desktop/webview"
)

var nativeBrowserAcceptanceError error

func TestMain(m *testing.M) {
	if output := os.Getenv("WHALEISLE_BROWSER_ACCEPTANCE_OUTPUT"); output != "" {
		nativeBrowserAcceptanceError = runNativeBrowserAcceptance(output)
	}
	os.Exit(m.Run())
}

func TestNativeBrowserWorkspaceAcceptance(t *testing.T) {
	if os.Getenv("WHALEISLE_BROWSER_ACCEPTANCE_OUTPUT") == "" {
		t.Fatal("explicit isolated Browser acceptance invocation required")
	}
	if nativeBrowserAcceptanceError != nil {
		t.Fatal(nativeBrowserAcceptanceError)
	}
}

func TestFunctionalBrowserDocumentLeaseAndBorrowedClose(t *testing.T) {
	record := map[string]any{"root": t.TempDir(), "scope": "real loopback document backend and controller ownership; no native floating window"}
	if err := checkBrowserDocumentLease(record["root"].(string), record); err != nil {
		t.Fatal(err)
	}
	if output := os.Getenv("WHALEISLE_BROWSER_BACKEND_OUTPUT"); output != "" {
		if !filepath.IsAbs(output) {
			t.Fatal("absolute backend evidence output required")
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		var combined map[string]any
		if err := json.Unmarshal(data, &combined); err != nil {
			t.Fatal(err)
		}
		combined["independentDocumentBackend"] = record
		data, err = json.MarshalIndent(combined, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(output, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("real owned loopback HTML GET, borrowed close preserves source, controller close revokes capability: PASS")
}

func runNativeBrowserAcceptance(output string) error {
	if !filepath.IsAbs(output) {
		return fmt.Errorf("absolute acceptance output required")
	}
	root, err := os.MkdirTemp(filepath.Dir(output), "browser-workspace-fresh-")
	if err != nil {
		return err
	}
	privateBrowserData, err := os.MkdirTemp(os.TempDir(), "wbqa-")
	if err != nil {
		return err
	}
	if err := os.Setenv("LOCALAPPDATA", privateBrowserData); err != nil {
		return err
	}
	mygo.App.SetName("WhaleIsle Browser acceptance " + filepath.Base(root))
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(root, "native"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(root, "cache"))
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(root, "logs"))
	visible := os.Getenv("WHALEISLE_BROWSER_ACCEPTANCE_VISIBLE") == "1"
	record := map[string]any{
		"schemaVersion": 1, "root": root, "environment": "Windows MyGo native App.Run and WebView2",
		"network": "owned ephemeral IPv4 loopback only", "modelRequests": 0, "hostStarted": false,
		"visibleWindows": visible, "activation": "ShowInactive only when explicitly requested", "visualAcceptance": "not performed", "childProcessExit": "not independently observed",
		"readinessDiagnosticIsLoadFailure": false,
		"matrix":                           []string{"WORK-07", "WORK-08", "DESK-03"},
	}
	record["privateBrowserDataBase"] = privateBrowserData
	var requestMu sync.Mutex
	requests := map[string]int{}
	write := func(err error) error {
		requestMu.Lock()
		hits := map[string]int{}
		for path, count := range requests {
			hits[path] = count
		}
		requestMu.Unlock()
		record["ownedHTTPRequests"] = hits
		if previous, readErr := os.ReadFile(output); readErr == nil {
			var old map[string]any
			if json.Unmarshal(previous, &old) == nil && old["pass"] == false {
				record["priorFailure"] = map[string]any{"root": old["root"], "error": old["error"], "backendIsLoadingAtTimeout": old["backendIsLoadingAtTimeout"], "navigationEvents": old["navigationEvents"], "pendingOwnedScriptBeacons": old["pendingOwnedScriptBeacons"]}
				if visible && old["visibleWindows"] == false {
					record["hiddenNavigationFailure"] = old
				}
				record["independentDocumentBackend"] = old["independentDocumentBackend"]
			}
		}
		record["pass"] = err == nil
		if err != nil {
			record["error"] = err.Error()
		}
		data, marshalErr := json.MarshalIndent(record, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		if writeErr := os.WriteFile(output, data, 0o600); writeErr != nil {
			return writeErr
		}
		return err
	}
	beacons := make(chan string, 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMu.Lock()
		requests[r.URL.Path]++
		requestMu.Unlock()
		if r.URL.Path == "/beacon" {
			select {
			case beacons <- r.URL.Query().Get("value"):
			default:
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/alpha" {
			fmt.Fprint(w, `<!doctype html><title>Alpha native Browser</title><p>Owned loopback Alpha</p><script>localStorage.setItem('whale-qa','workspace-A');fetch('/beacon?value=alpha:'+localStorage.getItem('whale-qa'))</script>`)
		} else {
			fmt.Fprint(w, `<!doctype html><title>Beta native Browser</title><p>Owned loopback Beta</p><script>fetch('/beacon?value=beta:'+(localStorage.getItem('whale-qa')||'empty'))</script>`)
		}
	}))
	defer server.Close()
	completed := make(chan error, 1)
	mygo.App.OnReady(func() {
		parent := mygo.NewWindow(mygo.WindowOptions{Title: "Owned Browser acceptance", Width: 1000, Height: 720, Hidden: true, Content: ui.View(func(c *ui.Context) { ui.Text(c, "Owned Browser backend acceptance") })})
		browser := NewController(parent)
		owner := WorkspaceOwner{Authority: "owned-qa:" + root, Cwd: "workspace-A"}
		browser.SetWorkspaceOwner(func() WorkspaceOwner { return owner })
		// A zero-area seat exercises real browser navigation without showing the
		// URL child or relying on OS/browser UI automation.
		browser.viewport = &mygo.Rectangle{}
		if visible {
			parent.ShowInactive()
			browser.viewport = &mygo.Rectangle{Width: 640, Height: 400}
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
			defer cancel()
			var pages []*mygo.Page
			var events []string
			call := func(action func() error) error {
				done := make(chan error, 1)
				parent.Update(func() { done <- action() })
				select {
				case err := <-done:
					return err
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			wait := func(host *desktop.Host, path, title string) error {
				deadline := time.Now().Add(12 * time.Second)
				for time.Now().Before(deadline) {
					state := host.State()
					if state.Error != "" {
						return fmt.Errorf("Browser load: %s", state.Error)
					}
					if !state.Loading && state.URL == server.URL+path && state.Title == title {
						return nil
					}
					time.Sleep(50 * time.Millisecond)
				}
				_ = call(func() error { record["navigationEvents"] = append([]string(nil), events...); return nil })
				requestMu.Lock()
				hits := map[string]int{}
				for path, count := range requests {
					hits[path] = count
				}
				requestMu.Unlock()
				record["ownedHTTPRequestsAtTimeout"] = hits
				if len(pages) > 0 {
					record["backendIsLoadingAtTimeout"] = pages[0].IsLoading()
					diagnostic, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					value, err := pages[0].EvalContext(diagnostic, "document.readyState")
					cancel()
					record["documentReadyStateAtTimeout"] = value
					if err != nil {
						record["readinessDiagnosticError"] = err.Error()
					}
				}
				record["pendingOwnedScriptBeacons"] = len(beacons)
				return fmt.Errorf("Browser load/title timed out: %+v", host.State())
			}
			beacon := func(want string) error {
				timer := time.NewTimer(8 * time.Second)
				defer timer.Stop()
				for {
					select {
					case got := <-beacons:
						if got == want {
							return nil
						}
					case <-timer.C:
						return fmt.Errorf("owned page script/storage beacon missing: %s", want)
					}
				}
			}
			var first, second *desktop.Host
			check := func() error {
				if err := call(func() error {
					browser.NewTab(server.URL + "/alpha")
					if browser.current() == nil {
						return fmt.Errorf("owned controller closed before initial Host returned")
					}
					first = browser.current().Host
					if first == nil {
						return fmt.Errorf("native Browser host creation: %s", browser.current().Error)
					}
					first.SetViewport(&mygo.Rectangle{Width: 640, Height: 400})
					record["nativeViewportDIP"] = []int{640, 400}
					for _, window := range mygo.Windows() {
						if page := window.Page(); page != nil {
							pages = append(pages, page)
							page.OnWillNavigate(func(*mygo.NavigateEvent) { events = append(events, "will-navigate") })
							page.OnDidNavigate(func(string) { events = append(events, "did-navigate") })
							page.OnDidFinishLoad(func() { events = append(events, "did-finish-load") })
							page.OnDidFailLoad(func(*mygo.LoadError) { events = append(events, "did-fail-load") })
						}
					}
					return nil
				}); err != nil {
					return err
				}
				if err := wait(first, "/alpha", "Alpha native Browser"); err != nil {
					return err
				}
				if err := beacon("alpha:workspace-A"); err != nil {
					return err
				}
				record["nativeURLTitleScript"] = "pass"
				if err := call(func() error {
					for _, event := range events {
						if event == "did-finish-load" {
							record["nativeFinishCallback"] = "pass"
							return nil
						}
					}
					return fmt.Errorf("native finish callback did not return to later listeners")
				}); err != nil {
					return err
				}
				if err := call(func() error {
					browser.Navigate(server.URL + "/beta")
					record["secondControllerDispatch"] = map[string]any{"sameHost": browser.current().Host == first, "tabCount": len(browser.Tabs), "tabError": browser.current().Error, "draft": browser.current().Draft, "hostTarget": first.URL()}
					first.SetViewport(&mygo.Rectangle{Width: 640, Height: 400})
					return nil
				}); err != nil {
					return err
				}
				if err := wait(first, "/beta", "Beta native Browser"); err != nil {
					return err
				}
				if err := beacon("beta:workspace-A"); err != nil {
					return err
				}
				first.Back()
				if err := wait(first, "/alpha", "Alpha native Browser"); err != nil {
					return err
				}
				first.Forward()
				if err := wait(first, "/beta", "Beta native Browser"); err != nil {
					return err
				}
				record["nativeHistory"] = "pass"
				if err := call(func() error {
					owner.Cwd = "workspace-B"
					browser.Navigate(server.URL + "/beta")
					second = browser.current().Host
					if second == nil || second == first || len(browser.Tabs) != 2 || browser.Tabs[0].workspace.key == browser.Tabs[1].workspace.key {
						return fmt.Errorf("workspace switch did not create a separate native profile")
					}
					second.SetViewport(&mygo.Rectangle{Width: 640, Height: 400})
					return nil
				}); err != nil {
					return err
				}
				if err := wait(second, "/beta", "Beta native Browser"); err != nil {
					return err
				}
				if err := beacon("beta:empty"); err != nil {
					return err
				}
				if err := call(func() error {
					browser.CloseTab(1)
					owner.Cwd = "workspace-A"
					browser.Select(0)
					browser.Navigate(server.URL + "/beta")
					first.SetViewport(&mygo.Rectangle{Width: 640, Height: 400})
					return nil
				}); err != nil {
					return err
				}
				if second.Alive() || !first.Alive() {
					return fmt.Errorf("closing one Browser tab destroyed the wrong Host")
				}
				if err := beacon("beta:workspace-A"); err != nil {
					return err
				}
				record["nativeWorkspaceStorageIsolation"] = "pass"
				if err := call(func() error {
					browser.Hide()
					if !first.Alive() {
						return fmt.Errorf("hiding Browser destroyed its resident page")
					}
					parent.SetSize(920, 680)
					width, height := parent.Size()
					record["ownedNativeWindowSizeDIP"] = []int{width, height}
					browser.Close()
					if first.Alive() || len(browser.Tabs) != 0 {
						return fmt.Errorf("Browser Close left a native Host or tab resident")
					}
					return nil
				}); err != nil {
					return err
				}
				record["nativeHideTabCloseControllerClose"] = "pass"
				return checkBrowserDocumentLease(root, record)
			}
			err := check()
			cleanupErr := call(func() error {
				record["navigationEvents"] = append([]string(nil), events...)
				browser.Close()
				if (first != nil && first.Alive()) || (second != nil && second.Alive()) || len(browser.Tabs) != 0 {
					return fmt.Errorf("owned Browser Host remained alive after Close")
				}
				parent.Destroy()
				if !parent.IsDestroyed() {
					return fmt.Errorf("owned native parent was not destroyed")
				}
				record["ownedNativeCloseAPI"] = "pass"
				return nil
			})
			if cleanupErr != nil {
				record["cleanupError"] = cleanupErr.Error()
				if err == nil {
					err = cleanupErr
				}
			}
			completed <- err
			mygo.App.Quit()
		}()
	})
	if err := mygo.App.Run(); err != nil {
		return write(err)
	}
	select {
	case err := <-completed:
		return write(err)
	case <-time.After(2 * time.Second):
		return write(fmt.Errorf("owned Browser desktop exited before completion"))
	}
}

func checkBrowserDocumentLease(root string, record map[string]any) error {
	workspace := filepath.Join(root, "document-workspace")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workspace, "owned.html"), []byte("<!doctype html><p>Owned document capability</p>"), 0o600); err != nil {
		return err
	}
	doc, err := htmldocument.OpenWorkspace(workspace, "owned.html", func() bool { return true })
	if err != nil {
		return err
	}
	defer doc.Close()
	url := doc.URL()
	transport := &http.Transport{Proxy: nil}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 2 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("owned HTML lease HTTP status %d", response.StatusCode)
	}
	tab := &Tab{ID: 1, HTML: doc}
	opts, err := floatingDocumentOptions(tab)
	if err != nil || opts.URL != url || opts.SessionCookie != "" || !opts.Untrusted {
		return fmt.Errorf("floating capability did not borrow its private document: %v", err)
	}
	borrower := &Controller{Tabs: []*Tab{tab}}
	borrower.floatingDocument = &floatingDocumentWindow{tab: tab, html: doc, done: make(chan struct{})}
	borrower.stopFloatingDocument()
	if !doc.Valid() {
		return fmt.Errorf("closing borrowed floating state revoked source lease")
	}
	borrower.Close()
	if doc.Valid() {
		return fmt.Errorf("closing source controller retained document lease")
	}
	response, err = client.Get(url)
	if err == nil {
		response.Body.Close()
		if response.StatusCode == http.StatusOK {
			return fmt.Errorf("closed document still served HTTP content")
		}
	}
	record["documentLeaseHTTPAndBorrowedClose"] = "pass: real loopback HTTP; floating state only, no floating window"
	record["floatingWindowAndDockDrag"] = "not performed"
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		return fmt.Errorf("owned document escaped IPv4 loopback")
	}
	return nil
}
