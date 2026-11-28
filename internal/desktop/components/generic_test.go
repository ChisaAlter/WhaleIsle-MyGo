package components

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

// No component process, installation, or network access: only private fixture files.
func TestGenericCatalogAndOriginalRegistry(t *testing.T) {
	root := t.TempDir()
	samples := filepath.Join(root, "samples")
	payload := filepath.Join(samples, "notes", "2.0.0")
	if err := os.MkdirAll(payload, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"manifest.json": `{"id":"notes","name":"Notes","version":"2.0.0","entry":"notes.js","kind":"service"}`, "notes.js": ""} {
		if err := os.WriteFile(filepath.Join(payload, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "components"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "components", "registry.json"), []byte(`{"components":{"notes":{"version":"1.0.0","previous":"0.9.0","state":"stopped","source":"bundled"},"orphan":{"version":"1.0.0","state":"error","lastError":"missing payload"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	rows := genericDiskRows(root, samples)
	if len(rows) != 2 {
		t.Fatalf("rows = %#v", rows)
	}
	if rows[0].ID != "notes" || rows[0].PreviousVersion != "0.9.0" || !rows[0].UpdateAvailable || rows[0].Kind != "service" {
		t.Fatalf("catalog/registry mismatch: %#v", rows[0])
	}
	if rows[1].ID != "orphan" || !rows[1].Orphan || rows[1].Message != "missing payload" {
		t.Fatalf("orphan missing: %#v", rows[1])
	}
}

func TestGenericManifestEntryContainment(t *testing.T) {
	for _, entry := range []string{"../escape.js", `C:\escape.js`, "/escape.js", `..\escape.js`} {
		if genericEntry("/payload", entry) != "" {
			t.Fatalf("accepted escaping entry %q", entry)
		}
	}
	if genericEntry("/payload", "nested/entry.js") == "" {
		t.Fatal("rejected relative entry")
	}
}

type fakeGenericChild struct {
	pid       int
	exited    chan error
	once      sync.Once
	killFails func() bool
}

func newFakeGenericChild(pid int) *fakeGenericChild {
	return &fakeGenericChild{pid: pid, exited: make(chan error, 1)}
}

func (c *fakeGenericChild) PID() int             { return c.pid }
func (c *fakeGenericChild) Exited() <-chan error { return c.exited }
func (c *fakeGenericChild) Kill() error {
	if c.killFails != nil && c.killFails() {
		return errors.New("access denied")
	}
	c.exitWith(nil)
	return nil
}
func (c *fakeGenericChild) exitWith(err error) {
	c.once.Do(func() {
		c.exited <- err
		close(c.exited)
	})
}

func fastGenericDeps() genericRuntimeDeps {
	return genericRuntimeDeps{
		startGrace:      0,
		startPoll:       5 * time.Millisecond,
		stateWait:       10 * time.Millisecond,
		statePoll:       2 * time.Millisecond,
		stopSoftGrace:   10 * time.Millisecond,
		stopForceGrace:  10 * time.Millisecond,
		stopPoll:        2 * time.Millisecond,
		restartBaseWait: 5 * time.Millisecond,
		maxRestarts:     genericMaxRestarts,
		pidAlive:        func(int) bool { return false },
		inspectProcess:  func(int) (string, string, bool) { return "", "", false },
		killPID:         func(int, bool) bool { return false },
		removeAll:       os.RemoveAll,
		now:             time.Now,
	}
}

func writeSampleComponent(t *testing.T, samplesRoot, id, version string) string {
	t.Helper()
	dir := filepath.Join(samplesRoot, id, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"id":"`+id+`","name":"Notes","version":"`+version+`","entry":"entry.js"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(dir, "entry.js")
	if err := os.WriteFile(entry, []byte("// fixture payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	return entry
}

func TestPureGoGenericLifecycleCatalogUpdateRollbackUninstallAndDataRetention(t *testing.T) {
	root := t.TempDir()
	samplesRoot := filepath.Join(root, "samples")
	userData := filepath.Join(root, "user")
	writeSampleComponent(t, samplesRoot, "notes", "1.0.0")

	var mu sync.Mutex
	live := map[int]*fakeGenericChild{}
	nextPID := 100
	var spawns []genericSpawnSpec

	deps := fastGenericDeps()
	deps.pidAlive = func(pid int) bool {
		mu.Lock()
		defer mu.Unlock()
		_, ok := live[pid]
		return ok
	}
	deps.inspectProcess = func(pid int) (string, string, bool) {
		mu.Lock()
		defer mu.Unlock()
		if _, ok := live[pid]; !ok {
			return "", "", false
		}
		return "node.exe", "notes entry.js", true
	}
	deps.spawn = func(spec genericSpawnSpec) (genericChildHandle, error) {
		mu.Lock()
		defer mu.Unlock()
		nextPID++
		c := newFakeGenericChild(nextPID)
		live[nextPID] = c
		spawns = append(spawns, spec)
		return c, nil
	}
	deps.killPID = func(pid int, _ bool) bool {
		mu.Lock()
		c := live[pid]
		delete(live, pid)
		mu.Unlock()
		if c != nil {
			c.exitWith(nil)
			return true
		}
		return false
	}

	svc := newGenericService(userData, GenericOptions{SamplesRoot: samplesRoot}, deps)
	t.Cleanup(func() { _ = svc.shutdown() })

	rows := svc.list()
	if len(rows) != 1 || rows[0].State != "available" {
		t.Fatalf("initial catalog=%+v", rows)
	}
	// Install without NodeBin must succeed in pure Go with 0 process spawns.
	if res := svc.install("notes", ""); !res.OK {
		t.Fatalf("install failed: %+v", res)
	}
	if len(spawns) != 0 {
		t.Fatalf("install unexpectedly spawned processes: %d", len(spawns))
	}

	dataFile := filepath.Join(svc.dataDir("notes"), "notes.txt")
	if err := os.WriteFile(dataFile, []byte("keep my notes"), 0o600); err != nil {
		t.Fatal(err)
	}

	if res := svc.start("notes"); !res.OK || res.State != "running" {
		t.Fatalf("start failed: %+v", res)
	}
	if res := svc.start("notes"); !res.OK || len(spawns) != 1 {
		t.Fatalf("start not idempotent: res=%+v spawns=%d", res, len(spawns))
	}

	writeSampleComponent(t, samplesRoot, "notes", "2.0.0")
	if rows = svc.list(); !rows[0].UpdateAvailable {
		t.Fatalf("expected UpdateAvailable after 2.0.0 added: %+v", rows[0])
	}
	upd := svc.update("notes")
	if !upd.OK || upd.From != "1.0.0" || upd.To != "2.0.0" || upd.State != "running" {
		t.Fatalf("update failed: %+v", upd)
	}
	if rows = svc.list(); rows[0].PreviousVersion != "1.0.0" || rows[0].State != "running" {
		t.Fatalf("post-update row mismatch: %+v", rows[0])
	}

	rb := svc.rollback("notes")
	if !rb.OK || rb.To != "1.0.0" || rb.State != "running" {
		t.Fatalf("rollback failed: %+v", rb)
	}
	if rows = svc.list(); rows[0].PreviousVersion != "2.0.0" || rows[0].InstalledVersion != "1.0.0" {
		t.Fatalf("post-rollback row mismatch: %+v", rows[0])
	}
	gotData, err := os.ReadFile(dataFile)
	if err != nil || string(gotData) != "keep my notes" {
		t.Fatalf("data lost on update/rollback: err=%v data=%q", err, gotData)
	}

	if res := svc.stop("notes"); !res.OK {
		t.Fatalf("stop failed: %+v", res)
	}
	mu.Lock()
	remaining := len(live)
	mu.Unlock()
	if remaining != 0 {
		t.Fatalf("expected 0 live procs after stop, got %d", remaining)
	}

	if res := svc.uninstall("notes"); !res.OK {
		t.Fatalf("uninstall failed: %+v", res)
	}
	if _, err := os.Stat(svc.versionsDir("notes")); !os.IsNotExist(err) {
		t.Fatalf("versions dir survived uninstall: err=%v", err)
	}
	gotData, err = os.ReadFile(dataFile)
	if err != nil || string(gotData) != "keep my notes" {
		t.Fatalf("data dir deleted on uninstall: err=%v data=%q", err, gotData)
	}
	if rows = svc.list(); len(rows) != 1 || rows[0].State != "available" {
		t.Fatalf("post-uninstall state=%+v", rows)
	}
}

func TestPureGoGenericFailuresNeverPublishFalseSuccessAndShutdownWithoutNode(t *testing.T) {
	root := t.TempDir()
	samplesRoot := filepath.Join(root, "samples")
	userData := filepath.Join(root, "user")
	writeSampleComponent(t, samplesRoot, "notes", "1.0.0")

	var mu sync.Mutex
	var current *fakeGenericChild
	killFails := true
	removeFails := false
	dieDuringWait := false

	deps := fastGenericDeps()
	deps.stateWait = 25 * time.Millisecond
	deps.pidAlive = func(pid int) bool {
		mu.Lock()
		defer mu.Unlock()
		return current != nil && current.pid == pid
	}
	deps.inspectProcess = func(pid int) (string, string, bool) {
		mu.Lock()
		defer mu.Unlock()
		if current == nil || current.pid != pid {
			return "", "", false
		}
		return "node.exe", "notes entry.js", true
	}
	deps.spawn = func(genericSpawnSpec) (genericChildHandle, error) {
		mu.Lock()
		defer mu.Unlock()
		current = newFakeGenericChild(321)
		current.killFails = func() bool {
			mu.Lock()
			defer mu.Unlock()
			return killFails
		}
		if dieDuringWait {
			go func(c *fakeGenericChild) {
				time.Sleep(3 * time.Millisecond)
				mu.Lock()
				if current == c {
					current = nil
				}
				mu.Unlock()
				c.exitWith(errors.New("crash"))
			}(current)
		}
		return current, nil
	}
	deps.killPID = func(pid int, _ bool) bool {
		mu.Lock()
		defer mu.Unlock()
		if killFails {
			return false
		}
		if current != nil && current.pid == pid {
			c := current
			current = nil
			c.exitWith(nil)
			return true
		}
		return false
	}
	deps.removeAll = func(path string) error {
		if removeFails && filepath.Base(path) == "versions" {
			return errors.New("EPERM locked")
		}
		return os.RemoveAll(path)
	}

	svc := newGenericService(userData, GenericOptions{SamplesRoot: samplesRoot}, deps)
	if !svc.install("notes", "").OK {
		t.Fatal("install failed")
	}
	if !svc.start("notes").OK {
		t.Fatal("start failed")
	}

	// Failed stop must keep state running and block update & uninstall.
	writeSampleComponent(t, samplesRoot, "notes", "2.0.0")
	if res := svc.stop("notes"); res.OK || res.Error != "stop-unconfirmed" {
		t.Fatalf("expected stop-unconfirmed, got %+v", res)
	}
	if row := svc.list()[0]; row.PID != 321 || row.State != "running" || row.Message == "" {
		t.Fatalf("failed stop mutated running record: %+v", row)
	}
	if upd := svc.update("notes"); upd.OK {
		t.Fatalf("update proceeded despite failed stop: %+v", upd)
	}
	if un := svc.uninstall("notes"); un.OK {
		t.Fatalf("uninstall proceeded despite failed stop: %+v", un)
	}

	killFails = false
	if res := svc.stop("notes"); !res.OK {
		t.Fatalf("stop retry failed: %+v", res)
	}
	removeFails = true
	if un := svc.uninstall("notes"); un.OK || svc.list()[0].InstalledVersion != "1.0.0" {
		t.Fatalf("failed remove dropped registry: %+v", un)
	}
	removeFails = false

	// Death during stateWait must land in error, not running.
	dieDuringWait = true
	if res := svc.start("notes"); res.OK || res.State != "error" || svc.list()[0].State != "error" {
		t.Fatalf("early death during stateWait did not mark error: %+v", res)
	}
	dieDuringWait = false

	// ShutdownGeneric without NodeBin and with disk-recorded dead/foreign PID
	// must NEVER spawn Node and must reconcile cleanly.
	ConfigureGeneric(userData, GenericOptions{SamplesRoot: samplesRoot, NodeBin: ""})
	reg := genericRegistryFile{Components: map[string]genericStoredRecord{
		"notes": {ID: "notes", Version: "1.0.0", State: "running", PID: intPtr(54321)},
	}}
	if svcDisk := genericServiceFor(userData); svcDisk != nil {
		svcDisk.mu.Lock()
		svcDisk.reg = reg
		svcDisk.loaded = true
		_ = svcDisk.saveRegistryLocked()
		svcDisk.mu.Unlock()
	}
	if err := ShutdownGeneric(userData); err != nil {
		t.Fatalf("ShutdownGeneric with empty NodeBin failed: %v", err)
	}
}

func TestPureGoGenericVersionTraversalAndPIDReuseProtection(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "../evil", `..\evil`, "1.0.0/../../x", `1.0.0\x`, "1.0\x00.0", " 1.0.0"} {
		if validComponentVersionSegment(bad) {
			t.Fatalf("accepted unsafe version %q", bad)
		}
	}
	if !validComponentVersionSegment("1.2.3-beta.1+build42") {
		t.Fatal("rejected valid semver")
	}

	root := t.TempDir()
	compRoot := filepath.Join(root, "user", "components")
	if err := os.MkdirAll(compRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	regPath := filepath.Join(compRoot, "registry.json")
	if err := os.WriteFile(regPath, []byte(`{"components":{"demo":{"id":"demo","version":"../outside","state":"installed"},"good":{"id":"good","version":"1.0.0","previous":"../../escape","state":"running","pid":54321}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	killedForeign := false
	deps := fastGenericDeps()
	deps.pidAlive = func(pid int) bool { return pid == 54321 }
	deps.inspectProcess = func(pid int) (string, string, bool) {
		return "/usr/bin/unrelated", "/usr/bin/unrelated --daemon", true
	}
	deps.killPID = func(int, bool) bool { killedForeign = true; return true }

	svc := newGenericService(filepath.Join(root, "user"), GenericOptions{}, deps)
	rows := svc.list()
	if len(rows) != 1 || rows[0].ID != "good" || rows[0].PreviousVersion != "" || rows[0].State != "stopped" || rows[0].PID != 0 {
		t.Fatalf("registry sanitization or PID reuse check failed: %+v", rows)
	}
	if err := svc.shutdown(); err != nil {
		t.Fatal(err)
	}
	if killedForeign {
		t.Fatal("PID reuse check killed unrelated foreign process")
	}

	// Symlink containment rejects escaped target paths cleanly.
	base := filepath.Join(root, "base")
	inside := filepath.Join(base, "versions", "1.0.0")
	outside := filepath.Join(root, "outside")
	if isContainedResolvedPath(base, outside, false) {
		t.Fatal("accepted outside path")
	}
	if !isContainedResolvedPath(base, inside, false) {
		t.Fatal("rejected valid inside path")
	}

	// Corrupt JSON is preserved as .broken instead of crashing list.
	corruptRoot := t.TempDir()
	corruptComp := filepath.Join(corruptRoot, "components")
	_ = os.MkdirAll(corruptComp, 0o755)
	_ = os.WriteFile(filepath.Join(corruptComp, "registry.json"), []byte("{not json"), 0o600)
	loaded := loadGenericRegistry(filepath.Join(corruptComp, "registry.json"))
	if len(loaded.Components) != 0 {
		t.Fatalf("expected empty registry on corrupt json, got %+v", loaded)
	}
	if _, err := os.Stat(filepath.Join(corruptComp, "registry.json.broken")); err != nil {
		t.Fatalf("expected broken backup preserved: %v", err)
	}
}

func TestPureGoGenericCrashSupervisionCapsAndShutdownCancels(t *testing.T) {
	root := t.TempDir()
	samplesRoot := filepath.Join(root, "samples")
	userData := filepath.Join(root, "user")
	writeSampleComponent(t, samplesRoot, "notes", "1.0.0")

	var mu sync.Mutex
	var children []*fakeGenericChild
	var phases []string

	deps := fastGenericDeps()
	deps.stateWait = 5 * time.Millisecond
	deps.restartBaseWait = 5 * time.Millisecond
	deps.maxRestarts = 2
	deps.pidAlive = func(pid int) bool {
		mu.Lock()
		defer mu.Unlock()
		for _, c := range children {
			if c.pid == pid {
				select {
				case <-c.exited:
					return false
				default:
					return true
				}
			}
		}
		return false
	}
	firstStartReturned := make(chan struct{})
	deps.spawn = func(genericSpawnSpec) (genericChildHandle, error) {
		mu.Lock()
		defer mu.Unlock()
		c := newFakeGenericChild(500 + len(children))
		isFirst := len(children) == 0
		children = append(children, c)
		go func(ch *fakeGenericChild, first bool) {
			if first {
				<-firstStartReturned
			} else {
				time.Sleep(8 * time.Millisecond)
			}
			ch.exitWith(errors.New("crash"))
		}(c, isFirst)
		return c, nil
	}

	svc := newGenericService(userData, GenericOptions{
		SamplesRoot: samplesRoot,
		OnProgress: func(_, _ string, p GenericProgress) {
			mu.Lock()
			phases = append(phases, p.Op+":"+p.Phase)
			mu.Unlock()
		},
	}, deps)
	t.Cleanup(func() { _ = svc.shutdown() })

	if !svc.install("notes", "").OK {
		t.Fatal("install failed")
	}
	if !svc.start("notes").OK {
		t.Fatal("start failed")
	}
	close(firstStartReturned)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rows := svc.list()
		if len(rows) == 1 && rows[0].State == "error" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if row := svc.list()[0]; row.State != "error" || row.PID != 0 {
		t.Fatalf("expected crash-looping component to hit retry cap and enter error, got %+v", row)
	}
	mu.Lock()
	gotSpawns := len(children)
	gotSuperviseErr := slices.Contains(phases, "supervise:error")
	mu.Unlock()
	if gotSpawns != 3 || !gotSuperviseErr {
		t.Fatalf("expected 1 initial + 2 restarts and supervise:error, got spawns=%d phases=%v", gotSpawns, phases)
	}
}

func intPtr(v int) *int { return &v }
