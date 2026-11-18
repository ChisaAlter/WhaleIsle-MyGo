//go:build windows && nativeacceptance

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"golang.org/x/sys/windows"
	"whaleisle/internal/app/updater"
	"whaleisle/internal/config"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

func TestNativeA2OwnedLifecycle(t *testing.T) {
	if os.Getenv("WHALEISLE_ACCEPTANCE_A2_OUTPUT") == "" {
		t.Skip("explicit owned A2 acceptance required")
	}
	if nativeAcceptanceError != nil {
		t.Fatal(nativeAcceptanceError)
	}
}

func TestNativeA2OwnedPanel(t *testing.T) {
	if os.Getenv("WHALEISLE_ACCEPTANCE_A2_PANEL") != "1" || os.Getenv("WHALEISLE_ACCEPTANCE_A2_OUTPUT") == "" {
		t.Skip("explicit owned A2 functional panel acceptance required")
	}
	if nativeAcceptanceError != nil {
		t.Fatal(nativeAcceptanceError)
	}
}

// Open process handles pin identity through PID reuse. This observer never
// stops a process, reads another profile or treats a name-only match as owned.
type a2ProcessAudit struct {
	mu    sync.Mutex
	known map[uint32]bool
	web   map[uint32]windows.Handle
	err   error
	stop  chan struct{}
	done  chan struct{}
}

func startA2ProcessAudit() *a2ProcessAudit {
	a := &a2ProcessAudit{known: map[uint32]bool{uint32(os.Getpid()): true}, web: map[uint32]windows.Handle{}, stop: make(chan struct{}), done: make(chan struct{})}
	a.sample()
	go func() {
		defer close(a.done)
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-a.stop:
				return
			case <-ticker.C:
				a.sample()
			}
		}
	}()
	return a
}

func (a *a2ProcessAudit) sample() {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		a.mu.Lock()
		a.err = err
		a.mu.Unlock()
		return
	}
	defer windows.CloseHandle(snapshot)
	var rows []windows.ProcessEntry32
	var row windows.ProcessEntry32
	row.Size = uint32(unsafe.Sizeof(row))
	for err = windows.Process32First(snapshot, &row); err == nil; err = windows.Process32Next(snapshot, &row) {
		rows = append(rows, row)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for changed := true; changed; {
		changed = false
		for _, row := range rows {
			if !a.known[row.ParentProcessID] {
				continue
			}
			if !a.known[row.ProcessID] {
				a.known[row.ProcessID] = true
				changed = true
			}
			if !strings.Contains(strings.ToLower(windows.UTF16ToString(row.ExeFile[:])), "msedgewebview2") || a.web[row.ProcessID] != 0 {
				continue
			}
			h, openErr := windows.OpenProcess(windows.SYNCHRONIZE, false, row.ProcessID)
			if openErr == nil {
				a.web[row.ProcessID] = h
			}
		}
	}
}

func (a *a2ProcessAudit) facts() (observed, alive []uint32, err error) {
	a.sample()
	a.mu.Lock()
	defer a.mu.Unlock()
	for pid, h := range a.web {
		observed = append(observed, pid)
		if state, e := windows.WaitForSingleObject(h, 0); e == nil && state == uint32(windows.WAIT_TIMEOUT) {
			alive = append(alive, pid)
		}
	}
	return observed, alive, a.err
}

func (a *a2ProcessAudit) close() {
	close(a.stop)
	<-a.done
	for _, h := range a.web {
		_ = windows.CloseHandle(h)
	}
}

func waitA2(duration time.Duration, ready func() (bool, error)) error {
	deadline := time.Now().Add(duration)
	for {
		ok, err := ready()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return errors.New("A2 condition timed out")
		}
		time.Sleep(40 * time.Millisecond)
	}
}

func runNativeA2Desktop(output string) error {
	panelOnly := os.Getenv("WHALEISLE_ACCEPTANCE_A2_PANEL") == "1"
	if panelOnly {
		fixture := os.Getenv(harness.EnvExtraOverlay)
		if !filepath.IsAbs(fixture) || filepath.Base(fixture) != "qa-panel" {
			return errors.New("functional A2 acceptance requires the existing absolute qa-panel extra-overlay fixture")
		}
		if body, err := os.ReadFile(filepath.Join(fixture, "package.json")); err != nil || !strings.Contains(string(body), `"name":"qa-panel"`) {
			return errors.New("functional A2 fixture package manifest unavailable")
		}
	}
	data, vendor, workspace := os.Getenv(paths.EnvUserData), os.Getenv(paths.EnvVendor), os.Getenv("WHALEISLE_ACCEPTANCE_WORKSPACE")
	if !isolatedDesktopHidden() || !filepath.IsAbs(data) || !filepath.IsAbs(vendor) || !filepath.IsAbs(workspace) || !filepath.IsAbs(output) || os.Getenv("LOCALAPPDATA") != data {
		return errors.New("A2 requires hidden isolation, absolute owned data/workspace/vendor/output and LOCALAPPDATA=data")
	}
	entries, err := os.ReadDir(data)
	if err != nil {
		return fmt.Errorf("A2 cannot inspect caller-owned profile %q: %w", data, err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		return fmt.Errorf("A2 requires an empty caller-owned profile %q; existing children: %q", data, names)
	}
	if err := os.MkdirAll(workspace, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
		return err
	}
	cfg, err := config.Open(data)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	if err = cfg.Update(func(c *config.Config) {
		c.Workspace, c.NodeBin, c.Host, c.Port, c.GoPort = workspace, os.Getenv("WHALEISLE_NODE"), "127.0.0.1", port, port
		c.AutoStartDesktop, c.QuitAfterStart = true, true
		c.AutoDownloadUpdates, c.HarnessAutoRestart, c.RemoteEnabled, c.CloseToTray = false, false, false, false
	}); err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(data, "welcome-shown"), nil, 0600)
	if err = os.Setenv(updater.EnvDisableAutoUpdate, "1"); err != nil {
		return err
	}
	audit := startA2ProcessAudit()
	defer audit.close()
	record := map[string]any{"scope": "real hidden production main, real Host and WebView2; callbacks and WindowDriver; no physical input", "processObservation": "owned descendants sampled every 25 ms; retained process handles", "modelRequestsByDriver": 0, "allWindowsHidden": true, "appPID": os.Getpid()}
	record["acceptance"] = "Client Runtime and resource lifecycle; standalone plugin content not asserted"
	record["pluginPanelContentVerified"] = false
	if panelOnly {
		record["acceptance"] = "one real qa-panel overlay: dynamic sidebar/main registration, mounted panel content and owned teardown"
	}
	var steps []map[string]any
	completed := make(chan error, 1)
	go func() {
		var win *mygo.Window
		var driver *ui.WindowDriver
		var failures []error
		var host *desktopwebview.Host
		finish := func(result error) {
			state.closeWebViewHost()
			state.mu.Lock()
			sup := state.sup
			state.mu.Unlock()
			if sup != nil {
				sup.Stop()
			}
			record["steps"] = steps
			record["passed"] = result == nil
			if result != nil {
				record["error"] = result.Error()
			}
			if body, e := json.MarshalIndent(record, "", "  "); e == nil {
				if e = os.WriteFile(output, body, 0600); result == nil {
					result = e
				}
			}
			completed <- result
			stopNativeAcceptance()
		}
		call := func(action func() error) error {
			var e error
			mygo.RunOnMain(func() {
				e = action()
				if driver != nil {
					driver.Frame()
				}
				for _, window := range mygo.Windows() {
					if window.IsVisible() {
						e = errors.New("owned acceptance window became visible")
						record["allWindowsHidden"] = false
					}
				}
			})
			return e
		}
		if e := waitA2(100*time.Second, func() (bool, error) {
			state.mu.Lock()
			win, status, sessions, mux := state.win, state.bootStatus, state.sessions, state.dshMuxReady
			state.mu.Unlock()
			if status.State == harness.Failed {
				return false, fmt.Errorf("Host startup: %v", status.Err)
			}
			return win != nil && status.State == harness.Ready && sessions != nil && mux && sessions.Snapshot().WorkspaceReady, nil
		}); e != nil {
			finish(e)
			return
		}
		state.mu.Lock()
		win = state.win
		state.mu.Unlock()
		if e := call(func() error {
			state.revealMainConversation()
			driver = ui.DriveWindow(win)
			if driver == nil {
				return errors.New("hidden native WindowDriver unavailable")
			}
			return nil
		}); e != nil {
			finish(e)
			return
		}
		if e := waitA2(time.Second, func() (bool, error) {
			seen, alive, e := audit.facts()
			record["defaultObservedWebViewPIDs"], record["defaultActiveWebViewPIDs"] = seen, alive
			if len(seen) != 0 {
				return false, errors.New("default native startup created WebView2")
			}
			return false, e
		}); e != nil && e.Error() != "A2 condition timed out" {
			finish(e)
			return
		}
		if png, e := win.CapturePage(); e != nil {
			finish(e)
			return
		} else if e = os.WriteFile(output+".native.png", png, 0600); e != nil {
			finish(e)
			return
		}
		plugin := os.Getenv("WHALEISLE_ACCEPTANCE_A2_PLUGIN")
		if panelOnly {
			plugin = "qa-panel"
		} else if plugin == "" {
			plugin = desktopwebview.PluginUsagePanel
		}
		record["plugin"] = plugin
		state.dshMu.Lock()
		client := state.dshClient
		state.dshMu.Unlock()
		if client == nil {
			finish(errors.New("ready A2 desktop has no authenticated Host client"))
			return
		}
		inventoryContext, inventoryCancel := context.WithTimeout(context.Background(), 8*time.Second)
		var loaderInventory api.PluginInventorySnapshot
		inventoryErr := client.Call(inventoryContext, "pluginInventory/list", nil, &loaderInventory)
		inventoryCancel()
		if inventoryErr != nil {
			finish(inventoryErr)
			return
		}
		var selectedEntries []api.PluginInventorySnapshotEntriesItem
		for _, entry := range loaderInventory.Entries {
			if entry.ModuleName == plugin {
				selectedEntries = append(selectedEntries, entry)
			}
		}
		record["selectedHostLoaderEntries"] = selectedEntries
		state.nativePluginController().Refresh(context.Background())
		if e := waitA2(15*time.Second, func() (bool, error) {
			s := state.marketCoordinator().InstalledPluginState(plugin)
			if s.Known {
				record["selectedPluginState"] = s
				for _, entry := range state.nativePluginController().Snapshot().Packages {
					if entry.Name == plugin {
						record["selectedHostBundle"] = entry
					}
				}
			}
			if s.Known && !s.Enabled {
				return false, fmt.Errorf("required bundled plugin %s unavailable", plugin)
			}
			return s.Known, nil
		}); e != nil {
			finish(e)
			return
		}
		open := func(source mainRoute) (*mygo.Window, string, error) {
			if e := call(func() error { state.navigateMainRoute(source); return nil }); e != nil {
				return nil, "", e
			}
			if e := call(func() error {
				state.openInstalledPluginViewWithLabel(plugin, "Owned A2 lifecycle")
				state.mu.Lock()
				host = state.webviewHost
				current, message := state.mainRoute, state.webviewHostErr
				state.mu.Unlock()
				if host == nil || !host.Alive() || current != routeWebView {
					return fmt.Errorf("explicit A2 open failed: %s", message)
				}
				return nil
			}); e != nil {
				return nil, "", e
			}
			var child *mygo.Window
			var gateway string
			e := waitA2(25*time.Second, func() (bool, error) {
				if st := host.State(); st.Error != "" {
					return false, errors.New(st.Error)
				}
				if e := call(func() error {
					for _, window := range mygo.Windows() {
						if window == win || window.IsDestroyed() {
							continue
						}
						u, e := url.Parse(window.Page().URL())
						if e == nil && net.ParseIP(u.Hostname()).IsLoopback() && u.Query().Get(desktopwebview.DesktopPluginQuery) == plugin {
							child, gateway = window, u.Host
						}
					}
					return nil
				}); e != nil {
					return false, e
				}
				return child != nil && host.ClientModules().Connected && !host.ClientModules().Syncing, nil
			})
			if e != nil {
				return child, gateway, fmt.Errorf("A2 live Client Runtime: %w", e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			value, e := child.Page().EvalContext(ctx, "JSON.stringify({ready:document.readyState,title:document.title,text:document.body.innerText.slice(0,180)})")
			if e != nil {
				return child, gateway, e
			}
			record["liveDocument"] = value
			connection, e := net.DialTimeout("tcp", gateway, time.Second)
			if e != nil {
				return child, gateway, e
			}
			_ = connection.Close()
			_, alive, e := audit.facts()
			if e == nil && len(alive) == 0 {
				e = errors.New("live A2 had no owned WebView2 process")
			}
			return child, gateway, e
		}
		retired := func(child *mygo.Window, gateway string, started time.Time) map[string]any {
			step := map[string]any{}
			e := waitA2(2*time.Second, func() (bool, error) {
				connection, dialErr := net.DialTimeout("tcp", gateway, 100*time.Millisecond)
				if connection != nil {
					_ = connection.Close()
				}
				_, alive, auditErr := audit.facts()
				step["gatewayClosed"], step["windowDestroyed"], step["remainingWebViewPIDs"] = dialErr != nil, child.IsDestroyed(), alive
				return dialErr != nil && child.IsDestroyed() && len(alive) == 0, auditErr
			})
			step["releaseMilliseconds"] = time.Since(started).Milliseconds()
			step["passed"] = e == nil
			if e != nil {
				failures = append(failures, fmt.Errorf("A2 resources did not all exit within 2 seconds: %w", e))
			}
			return step
		}
		if panelOnly {
			child, gateway, e := open(routeMarket)
			if e != nil {
				finish(e)
				return
			}
			var mounted map[string]any
			e = waitA2(15*time.Second, func() (bool, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				value, e := child.Page().EvalContext(ctx, `JSON.stringify((()=>{const panel=document.querySelector('[data-qa="qa-panel-view"]'),icon=document.querySelector('[data-qa="qa-panel-icon"]'),picker=document.querySelector('[aria-label="Plugin views"]'),box=panel?.getBoundingClientRect();return{plugin:new URLSearchParams(location.search).get('desktopPlugin'),ready:document.readyState,panelText:panel?.textContent||'',panelCount:document.querySelectorAll('[data-qa="qa-panel-view"]').length,sidebarIconCount:document.querySelectorAll('[data-qa="qa-panel-icon"]').length,pickerButtons:[...(picker?.querySelectorAll('button')||[])].map(b=>b.textContent),panelWidth:box?.width||0,panelHeight:box?.height||0,panelDisplay:panel?getComputedStyle(panel).display:''}})())`)
				if e != nil {
					return false, e
				}
				encoded, ok := value.(string)
				if !ok {
					return false, fmt.Errorf("unexpected functional panel observation type %T", value)
				}
				if e = json.Unmarshal([]byte(encoded), &mounted); e != nil {
					return false, e
				}
				record["functionalPanel"] = mounted
				buttons, _ := mounted["pickerButtons"].([]any)
				pickerMatched := false
				for _, button := range buttons {
					pickerMatched = pickerMatched || button == "QA Panel"
				}
				return mounted["plugin"] == "qa-panel" && mounted["panelText"] == "QA plugin panel view" && mounted["panelCount"] == float64(1) && mounted["sidebarIconCount"].(float64) > 0 && mounted["panelWidth"].(float64) > 0 && mounted["panelHeight"].(float64) > 0 && mounted["panelDisplay"] != "none" && pickerMatched, nil
			})
			if e != nil {
				finish(fmt.Errorf("dynamic qa-panel did not mount: %w", e))
				return
			}
			// A fresh profile shows the Host's preview notice after its settings
			// read settles. Follow its ordinary Continue/later actions so the
			// captured plugin surface is usable, not covered by onboarding.
			var noticeActions []string
			var clearSince time.Time
			e = waitA2(12*time.Second, func() (bool, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				value, e := child.Page().EvalContext(ctx, `JSON.stringify((()=>{let clicked='';const titles={'预览版说明':['继续'],'Preview Notice':['Continue'],'添加一个 API Key 开始使用':['稍后配置'],'Add an API Key to get started':['Configure later'],'开始使用':['关闭'],'Get started':['Close']};for(const dialog of document.querySelectorAll('[role="dialog"]')){const title=dialog.querySelector('h2')?.textContent?.trim(),labels=titles[title];if(!labels)continue;const buttons=[...dialog.querySelectorAll('button')].filter(b=>!b.disabled&&labels.includes((b.getAttribute('aria-label')||b.textContent).trim()));if(buttons.length===1){buttons[0].click();clicked=title;break;}}return{clicked,inert:!!document.getElementById('root')?.inert,dialogs:[...document.querySelectorAll('[role="dialog"]')].map(d=>d.querySelector('h2')?.textContent||d.getAttribute('aria-label')||'dialog')}})())`)
				if e != nil {
					return false, e
				}
				var observation struct {
					Clicked string   `json:"clicked"`
					Inert   bool     `json:"inert"`
					Dialogs []string `json:"dialogs"`
				}
				encoded, ok := value.(string)
				if !ok {
					return false, fmt.Errorf("unexpected onboarding observation %T", value)
				}
				if e = json.Unmarshal([]byte(encoded), &observation); e != nil {
					return false, e
				}
				if observation.Clicked != "" {
					noticeActions = append(noticeActions, observation.Clicked)
				}
				record["onboardingActions"], record["onboardingRemaining"] = noticeActions, observation
				if len(noticeActions) == 0 || observation.Inert || len(observation.Dialogs) > 0 {
					clearSince = time.Time{}
					return false, nil
				}
				if clearSince.IsZero() {
					clearSince = time.Now()
				}
				return time.Since(clearSince) >= 750*time.Millisecond, nil
			})
			if e != nil {
				finish(fmt.Errorf("owned onboarding did not release the plugin surface: %w", e))
				return
			}
			record["pluginPanelContentVerified"] = true
			if png, e := child.CapturePage(); e != nil {
				finish(e)
				return
			} else if e = os.WriteFile(output+".panel.png", png, 0600); e != nil {
				finish(e)
				return
			}
			record["panelCapture"] = output + ".panel.png"
			started := time.Now()
			if e = call(func() error { return driver.Click(i18n.T(i18n.KeyWebViewBack)) }); e != nil {
				finish(e)
				return
			}
			step := retired(child, gateway, started)
			step["navigation"] = "functional-panel-teardown"
			steps = append(steps, step)
			finish(errors.Join(failures...))
			return
		}
		for _, target := range []struct {
			name  string
			route mainRoute
		}{{"market", routeMarket}, {"settings", routeSettings}, {"schedule", routeSchedule}} {
			child, gateway, e := open(target.route)
			if e != nil {
				finish(e)
				return
			}
			started := time.Now()
			if e = call(func() error { state.navigateMainRoute(target.route); return nil }); e != nil {
				finish(e)
				return
			}
			step := retired(child, gateway, started)
			step["navigation"] = target.name
			steps = append(steps, step)
		}
		child, gateway, e := open(routeConversation)
		if e != nil {
			finish(e)
			return
		}
		started := time.Now()
		if e = call(func() error { return driver.Click(i18n.T(i18n.KeyWebViewBack)) }); e != nil {
			finish(e)
			return
		}
		step := retired(child, gateway, started)
		step["navigation"] = "native-back-button"
		steps = append(steps, step)
		child, gateway, e = open(routeConversation)
		if e != nil {
			finish(e)
			return
		}
		state.mu.Lock()
		sup, generation := state.sup, state.bootStatus.Generation
		state.mu.Unlock()
		started = time.Now()
		sup.Restart()
		step = retired(child, gateway, started)
		step["navigation"] = "host-restart"
		steps = append(steps, step)
		if e = waitA2(80*time.Second, func() (bool, error) {
			st := sup.Status()
			if st.State == harness.Failed {
				return false, st.Err
			}
			return st.State == harness.Ready && st.Generation > generation, nil
		}); e != nil {
			finish(e)
			return
		}
		record["hostGenerationBefore"], record["hostGenerationAfter"] = generation, sup.Status().Generation
		finish(errors.Join(failures...))
	}()
	main()
	select {
	case result := <-completed:
		return result
	case <-time.After(2 * time.Second):
		return errors.New("A2 desktop exited before acceptance completed")
	}
}
