//go:build windows && projectvisualacceptance && !release && !nativeacceptance && !appearanceacceptance && !livedesktopacceptance && !sidebaracceptance && !sessionrecoveryacceptance && !settingsacceptance && !richhistoryacceptance && !composerflowacceptance && !nativetextacceptance

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"golang.org/x/sys/windows"
	projectctl "whaleisle/internal/app/project"
	"whaleisle/internal/desktop/previewprocess"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/projects"
)

var projectVisualAcceptanceError error

func TestMain(m *testing.M) {
	flag.Parse()
	budget := 100 * time.Second
	if os.Getenv("WHALEISLE_PROJECT_VISUAL_MODE") == "hidden-close" && os.Getenv("WHALEISLE_PROJECT_VISUAL_CYCLES") == "50" {
		budget = 6 * time.Minute
	}
	watchdog := time.AfterFunc(budget, func() {
		fmt.Fprintln(os.Stderr, "Project visual acceptance exceeded", budget)
		os.Exit(124)
	})
	projectVisualAcceptanceError = runProjectVisualAcceptance()
	watchdog.Stop()
	os.Exit(m.Run())
}

func TestProjectVisualNativeLifecycle(t *testing.T) {
	if projectVisualAcceptanceError != nil {
		t.Fatal(projectVisualAcceptanceError)
	}
}

// This desktop is never switched to. A window can have real WS_VISIBLE and
// receive MyGo Hide/Show events without appearing on the user's input desktop.
func projectVisualPrivateDesktop() (string, error) {
	user := windows.NewLazySystemDLL("user32.dll")
	objectName := func(handle uintptr) (string, error) {
		var value [256]uint16
		var needed uint32
		ok, _, err := user.NewProc("GetUserObjectInformationW").Call(handle, 2, uintptr(unsafe.Pointer(&value[0])), unsafe.Sizeof(value), uintptr(unsafe.Pointer(&needed)))
		if ok == 0 {
			return "", err
		}
		return windows.UTF16ToString(value[:]), nil
	}
	input, _, err := user.NewProc("OpenInputDesktop").Call(0, 0, 1)
	if input == 0 {
		return "", fmt.Errorf("OpenInputDesktop: %w", err)
	}
	inputName, nameErr := objectName(input)
	user.NewProc("CloseDesktop").Call(input)
	if nameErr != nil {
		return "", nameErr
	}
	name := fmt.Sprintf("WhaleIsleProjectVisual-%d-%d", os.Getpid(), time.Now().UnixNano())
	text, _ := windows.UTF16PtrFromString(name)
	desktop, _, err := user.NewProc("CreateDesktopW").Call(uintptr(unsafe.Pointer(text)), 0, 0, 0, 0x00c7, 0)
	if desktop == 0 {
		return "", fmt.Errorf("CreateDesktopW: %w", err)
	}
	ok, _, err := user.NewProc("SetThreadDesktop").Call(desktop)
	if ok == 0 {
		user.NewProc("CloseDesktop").Call(desktop)
		return "", fmt.Errorf("SetThreadDesktop: %w", err)
	}
	current, _, _ := user.NewProc("GetThreadDesktop").Call(uintptr(windows.GetCurrentThreadId()))
	currentName, err := objectName(current)
	if err != nil || currentName != name || strings.EqualFold(currentName, inputName) {
		return "", fmt.Errorf("private desktop verification failed: current=%q input=%q error=%v", currentName, inputName, err)
	}
	// Keep the desktop alive until this standalone test process exits. MyGo's
	// backend and OLE message windows must not outlive their desktop.
	return name, nil
}

type projectVisualProcess struct {
	PID    uint32 `json:"pid"`
	Name   string `json:"name"`
	Alive  bool   `json:"alive"`
	handle windows.Handle
}
type projectVisualProcessAudit struct {
	mu         sync.Mutex
	known      map[uint32]bool
	processes  map[uint32]*projectVisualProcess
	err        error
	stop, done chan struct{}
}

func projectVisualResources() (map[string]any, error) {
	var memory struct {
		Size, PageFaults                                                                                                      uint32
		PeakWorkingSet, WorkingSet, PeakPagedPool, PagedPool, PeakNonpagedPool, NonpagedPool, PageFile, PeakPageFile, Private uintptr
	}
	memory.Size = uint32(unsafe.Sizeof(memory))
	proc := windows.CurrentProcess()
	ok, _, err := windows.NewLazySystemDLL("psapi.dll").NewProc("GetProcessMemoryInfo").Call(uintptr(proc), uintptr(unsafe.Pointer(&memory)), uintptr(memory.Size))
	if ok == 0 {
		return nil, fmt.Errorf("GetProcessMemoryInfo: %w", err)
	}
	var handles uint32
	ok, _, err = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetProcessHandleCount").Call(uintptr(proc), uintptr(unsafe.Pointer(&handles)))
	if ok == 0 {
		return nil, fmt.Errorf("GetProcessHandleCount: %w", err)
	}
	return map[string]any{"privateBytes": memory.Private, "workingSetBytes": memory.WorkingSet, "handles": handles, "goroutines": runtime.NumGoroutine()}, nil
}

func newProjectVisualProcessAudit() *projectVisualProcessAudit {
	a := &projectVisualProcessAudit{known: map[uint32]bool{uint32(os.Getpid()): true}, processes: map[uint32]*projectVisualProcess{}, stop: make(chan struct{}), done: make(chan struct{})}
	a.sample()
	go func() {
		defer close(a.done)
		tick := time.NewTicker(25 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-a.stop:
				return
			case <-tick.C:
				a.sample()
			}
		}
	}()
	return a
}
func (a *projectVisualProcessAudit) sample() {
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
			name := strings.ToLower(windows.UTF16ToString(row.ExeFile[:]))
			if a.processes[row.ProcessID] != nil || name != "whaleisle-preview.exe" && name != "msedgewebview2.exe" {
				continue
			}
			handle, e := windows.OpenProcess(windows.SYNCHRONIZE, false, row.ProcessID)
			if e == nil {
				a.processes[row.ProcessID] = &projectVisualProcess{PID: row.ProcessID, Name: name, handle: handle}
			}
		}
	}
}
func (a *projectVisualProcessAudit) facts() ([]projectVisualProcess, int, error) {
	a.sample()
	a.mu.Lock()
	defer a.mu.Unlock()
	var facts []projectVisualProcess
	alive := 0
	for _, process := range a.processes {
		fact := *process
		result, err := windows.WaitForSingleObject(fact.handle, 0)
		if err != nil {
			return facts, alive, err
		}
		fact.Alive = result == uint32(windows.WAIT_TIMEOUT)
		if fact.Alive {
			alive++
		}
		facts = append(facts, fact)
	}
	return facts, alive, a.err
}
func (a *projectVisualProcessAudit) close() {
	close(a.stop)
	<-a.done
	for _, process := range a.processes {
		windows.CloseHandle(process.handle)
	}
}

func runProjectVisualAcceptance() (failure error) {
	root, output := os.Getenv("WHALEISLE_PROJECT_VISUAL_ROOT"), os.Getenv("WHALEISLE_PROJECT_VISUAL_OUTPUT")
	hiddenClose := os.Getenv("WHALEISLE_PROJECT_VISUAL_MODE") == "hidden-close"
	visible := os.Getenv("WHALEISLE_PROJECT_VISUAL_MODE") == "visible"
	cycles := 2
	if raw := os.Getenv("WHALEISLE_PROJECT_VISUAL_CYCLES"); raw != "" {
		if !hiddenClose || raw != "50" {
			return errors.New("only the fixed 50-cycle hidden-close workload is supported")
		}
		cycles, _ = strconv.Atoi(raw)
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(output) || os.Getenv("WHALEISLE_ISOLATED") != "1" {
		return errors.New("absolute owned root/output and WHALEISLE_ISOLATED=1 required")
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		return errors.New("Project visual root must be fresh and absent")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	helper := previewprocess.HelperPath(exe)
	if _, err = os.Stat(helper); err != nil {
		return fmt.Errorf("candidate companion helper: %w", err)
	}
	record := map[string]any{"schemaVersion": 1, "appPID": os.Getpid(), "helper": helper,
		"scope": "production Project Manager, ObserverView, mainProjectDeps renderer, publishedHTMLSurface and real AppContainer/WebView2",
		"host":  "loopback synthetic Project/readVisualReply responses", "modelRequests": 0,
		"visibility":         "native ShowInactive/Hide on a private desktop that is never switched to",
		"processObservation": "owned helper/WebView2 descendants sampled every 25 ms with retained process handles"}
	var stages []map[string]any
	defer func() {
		record["passed"], record["stages"] = failure == nil, stages
		if failure != nil {
			record["error"] = failure.Error()
		}
		body, err := json.MarshalIndent(record, "", "  ")
		if err == nil {
			err = os.WriteFile(output, append(body, '\n'), 0600)
		}
		failure = errors.Join(failure, err)
	}()
	if hiddenClose {
		record["visibility"] = "ordinary desktop; parent stays Hidden for its entire life; hidden child attachment and Close only"
		record["osHideShowExercised"] = false
	} else if visible {
		if os.Getenv("WHALEISLE_ISOLATED_VISIBLE") != "1" {
			return errors.New("visible acceptance requires WHALEISLE_ISOLATED_VISIBLE=1")
		}
		record["visibility"] = "authorized visible synthetic window on the ordinary input desktop"
		record["osHideShowExercised"] = true
	} else {
		desktop, err := projectVisualPrivateDesktop()
		if err != nil {
			return err
		}
		record["privateDesktop"] = desktop
		record["osHideShowExercised"] = true
	}
	record["plannedCycles"] = cycles
	if err = os.Mkdir(root, 0700); err != nil {
		return err
	}
	mygo.App.SetName("Project visual lifecycle acceptance")
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(root, "native"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(root, "cache"))
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(root, "logs"))
	i18n.SetLocale("en")

	const html = "<!doctype html><html><body><h1>Owned Project page</h1><button onclick=\"this.textContent='Clicked'\">Local action</button></body></html>"
	var card projectflow.RequestCard
	if err = json.Unmarshal([]byte(`{"id":"A","projectId":"project","revision":1,"title":"Owned request","status":"completed","delivery":{"id":"delivery-A","applicable":true,"outcome":"completed","pages":[{"sessionId":"lead","callId":"page-A","title":"Owned published page","height":180,"file":{"attachmentId":"owned-html","name":"owned.html"}}]}}`), &card); err != nil {
		return err
	}
	card.Delivery.Pages[0].File.Bytes = int64(len(html))
	project := projectflow.Project{ID: "project", CoordinatorSessionID: "lead", Lifecycle: "ready", Title: "Owned Project", CanonicalWorkingDirectory: root}
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			RPCID   string         `json:"rpcId"`
			Payload map[string]any `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		var value any
		switch r.URL.Path {
		case "/dsh-project/list":
			value = projectflow.Listing{Available: true, Projects: []projectflow.Project{project}}
		case "/dsh-project/detail":
			value = map[string]any{"project": project}
		case "/dsh-project/requests/list":
			page := projectflow.RequestPage{}
			if request.Payload["kind"] == "completed" {
				page.Items, page.Total = []projectflow.RequestCard{card}, 1
			}
			value = page
		case "/dsh-project/requests/attention":
			value = projectflow.RequestPage{}
		case "/dsh-project/requests/get":
			value = map[string]any{"card": card}
		case "/api/session/projections":
			value = map[string]any{"values": map[string]any{"agentTeam": map[string]any{"members": []any{}, "tasks": []any{}}}}
		case "/api/session/list":
			value = map[string]any{"items": []any{}}
		case "/api/session/readVisualReply":
			reads.Add(1)
			value = map[string]any{"attachment": map[string]any{"attachmentId": "owned-html", "name": "owned.html", "bytes": len(html)}, "html": html}
		default:
			http.Error(w, "unexpected Project acceptance endpoint", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer server.Close()
	s := &mainState{dshClient: &dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}, HTTP: server.Client()}}
	deps := mainProjectDeps{s: s}.hostDeps()
	deps.VisualReaderAvailable = func() (bool, string) { return true, "" } // Fixture supplies this Host capability only.
	mgr := projectctl.NewManager(deps)
	s.projectCtl = mgr
	defer mgr.Close()
	audit := newProjectVisualProcessAudit()
	defer audit.close()
	completed := make(chan error, 1)
	mygo.App.OnReady(func() {
		var win *mygo.Window
		var model projects.Model
		var presented projects.VisualPagePreview
		var presentedWidth int
		observer := &projects.ObserverState{ProjectID: "project", CoordinatorSessionID: "lead", Section: "tasks",
			RequestModes: map[string]projects.RequestMode{"A": {Mode: "card", Completed: true, Tab: "results"}}}
		win = mygo.NewWindow(mygo.WindowOptions{Title: "WhaleIsle Project lifecycle QA", Width: 900, Height: 800, Hidden: true, Content: ui.View(func(c *ui.Context) {
			presented = projects.VisualPagePreview{}
			if win == nil {
				return
			}
			defer mgr.BeginPresentationFrame()()
			base := mgr.BindModel(win, "", false)
			if !base.Loaded || !base.Listing.Available {
				ui.Text(c, "Loading Project")
				return
			}
			observer.Host = base.ObserverHost
			model = base.ObserverModel(observer)
			model.SelectedSession = "lead"
			visual := model.VisualPage
			model.VisualPage = func(source string, page projectflow.DeliveryPageReference, width int) projects.VisualPagePreview {
				result := visual(source, page, width)
				if source == "request:A" && page == card.Delivery.Pages[0] {
					presented, presentedWidth = result, width
				}
				return result
			}
			projects.ObserverView(c, model, observer)
		})})
		s.win = win
		var hides, shows int
		win.OnHide(func() { hides++; mgr.SetWindowHidden(true) })
		win.OnShow(func() { shows++; mgr.SetWindowHidden(false); win.Invalidate() })
		driver := ui.DriveWindow(win)
		go func() {
			err := func() error {
				if driver == nil {
					return errors.New("native WindowDriver unavailable")
				}
				call := func(fn func() error) error {
					var err error
					mygo.RunOnMain(func() {
						err = fn()
						driver.Frame()
						if hiddenClose && win.IsVisible() {
							err = errors.Join(err, errors.New("hidden acceptance parent became visible"))
						}
					})
					return err
				}
				wait := func(timeout time.Duration, fn func() (bool, error)) error {
					deadline := time.Now().Add(timeout)
					for {
						done, err := fn()
						if err != nil || done {
							return err
						}
						if time.Now().After(deadline) {
							return errors.New("Project native lifecycle condition timed out")
						}
						time.Sleep(40 * time.Millisecond)
					}
				}
				preview := func() projects.VisualPagePreview {
					// Observe the actual rendered snapshot. Calling VisualPage here
					// would mutate width/presentation ownership during the wait.
					return presented
				}
				checkInactive := func(stage string, expectedReads int32) error {
					facts, alive, err := audit.facts()
					if err != nil {
						return err
					}
					resources, resourceErr := projectVisualResources()
					if resourceErr != nil {
						return resourceErr
					}
					resources["retainedAuditHandles"] = len(facts)
					resources["handlesWithoutAudit"] = resources["handles"].(uint32) - uint32(len(facts))
					stages = append(stages, map[string]any{"stage": stage, "active": alive, "htmlReads": reads.Load(), "resources": resources})
					if alive != 0 || reads.Load() != expectedReads || expectedReads == 0 && len(facts) != 0 {
						return fmt.Errorf("%s: active=%d HTML reads=%d expected=%d", stage, alive, reads.Load(), expectedReads)
					}
					return call(func() error {
						p := preview()
						if p.Open == nil || p.Opened || p.Loading || p.Surface != nil || model.RequestCard == nil || model.RequestCard.ID != "A" {
							return fmt.Errorf("%s lost the card/open entry or reopened its renderer", stage)
						}
						return nil
					})
				}
				if err := call(func() error {
					if hiddenClose {
						return nil
					}
					win.ShowInactive()
					if !win.IsVisible() || shows != 1 {
						return errors.New("private window did not enter real visible state")
					}
					return nil
				}); err != nil {
					return err
				}
				if err := wait(8*time.Second, func() (bool, error) {
					ready := false
					err := call(func() error {
						if model.Error != "" {
							return errors.New(model.Error)
						}
						ready = preview().Open != nil
						return nil
					})
					return ready, err
				}); err != nil {
					return err
				}
				if err := checkInactive("default-awaiting-open", 0); err != nil {
					return err
				}
				for cycle := 0; cycle < cycles; cycle++ {
					if err := call(func() error {
						rect, _ := driver.Find("Open interactive page")
						record["lastOpenButtonBounds"] = rect
						return driver.Click("Open interactive page")
					}); err != nil {
						return err
					}
					if err := call(func() error {
						if !presented.Opened {
							record["clickFailureTexts"], record["clickFailureWidth"] = driver.Texts(), presentedWidth
							return errors.New("native Open interactive page click did not open its presented page")
						}
						return nil
					}); err != nil {
						return err
					}
					var surface *publishedHTMLSurface
					if err := wait(35*time.Second, func() (bool, error) {
						err := call(func() error {
							p := preview()
							if p.Error != "" || p.InteractiveError != "" {
								return fmt.Errorf("Project renderer: %s %s", p.Error, p.InteractiveError)
							}
							if p.Surface != nil {
								var ok bool
								surface, ok = p.Surface.(*publishedHTMLSurface)
								if !ok {
									return errors.New("Project did not use the production publishedHTMLSurface")
								}
							}
							return nil
						})
						return surface != nil, err
					}); err != nil {
						return err
					}
					facts, alive, err := audit.facts()
					if err != nil {
						return err
					}
					helperAlive, webAlive := false, false
					for _, p := range facts {
						helperAlive = helperAlive || p.Alive && p.Name == "whaleisle-preview.exe"
						webAlive = webAlive || p.Alive && p.Name == "msedgewebview2.exe"
					}
					stages = append(stages, map[string]any{"stage": fmt.Sprintf("explicit-open-%d", cycle+1), "active": alive, "htmlReads": reads.Load()})
					if !helperAlive || !webAlive {
						surface.mu.Lock()
						paintFailure := surface.failure
						surface.mu.Unlock()
						if paintFailure != nil {
							record["nativeSurfaceFailure"] = paintFailure.Error()
						}
						return errors.Join(errors.New("opened Project page has no observed live helper and WebView2 tree"), paintFailure)
					}
					if visible && cycle == 0 {
						// Leave the actual interactive render available for inspection;
						// the existing lifecycle checks resume without external input.
						fmt.Fprintln(os.Stdout, "VISIBLE_PROJECT_PAGE_READY")
						time.Sleep(15 * time.Second)
					}
					closedAt := time.Now()
					if cycle == 0 && !hiddenClose {
						err = call(func() error {
							win.Hide()
							if win.IsVisible() || hides != 1 {
								return errors.New("Hide did not invoke the real native visibility callback")
							}
							return nil
						})
					} else {
						err = call(func() error { return driver.Click("Close page preview") })
					}
					if err != nil {
						return err
					}
					if err = wait(8*time.Second, func() (bool, error) {
						select {
						case <-surface.Done():
						default:
							return false, nil
						}
						_, alive, e := audit.facts()
						return alive == 0, e
					}); err != nil {
						return err
					}
					if err = surface.Close(); err != nil {
						return fmt.Errorf("production renderer cleanup: %w", err)
					}
					stages = append(stages, map[string]any{"stage": fmt.Sprintf("close-completed-%d", cycle+1), "releaseMilliseconds": time.Since(closedAt).Milliseconds(), "goroutines": runtime.NumGoroutine()})
					if cycle == 0 && !hiddenClose {
						if err = call(func() error {
							win.ShowInactive()
							if !win.IsVisible() || shows != 2 {
								return errors.New("native restore did not fire")
							}
							return nil
						}); err != nil {
							return err
						}
					}
					if err = checkInactive(fmt.Sprintf("released-and-restored-%d", cycle+1), int32(cycle+1)); err != nil {
						return err
					}
					// Exercise subsequent real UI frames to catch an automatic restart.
					for frame := 0; frame < 6; frame++ {
						if err = call(func() error { return nil }); err != nil {
							return err
						}
						time.Sleep(50 * time.Millisecond)
					}
					if err = checkInactive(fmt.Sprintf("remained-closed-%d", cycle+1), int32(cycle+1)); err != nil {
						return err
					}
				}
				record["hideEvents"], record["showEvents"] = hides, shows
				record["completedCycles"] = cycles
				return nil
			}()
			mgr.Close()
			// Cancellation owns renderer teardown even when a scenario fails.
			// Do not report an exited job solely from a cancelled UI context.
			deadline := time.Now().Add(8 * time.Second)
			for {
				facts, alive, auditErr := audit.facts()
				if alive == 0 || auditErr != nil || time.Now().After(deadline) {
					record["finalProcesses"] = facts
					err = errors.Join(err, auditErr)
					if alive != 0 {
						err = errors.Join(err, errors.New("owned renderer processes remained after manager teardown"))
					}
					break
				}
				time.Sleep(40 * time.Millisecond)
			}
			mygo.RunOnMain(func() { win.Destroy(); mygo.App.Quit() })
			completed <- err
		}()
	})
	runErr := mygo.App.Run()
	select {
	case err := <-completed:
		return errors.Join(runErr, err)
	case <-time.After(time.Second):
		return errors.Join(runErr, errors.New("native acceptance ended without completion"))
	}
}
