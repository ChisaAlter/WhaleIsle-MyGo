package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/processutil"
	"whaleisle/internal/store/storelocale"
)

const (
	genericLogMaxBytes      = 256 * 1024
	genericStartGrace       = 1500 * time.Millisecond
	genericStartPoll        = 150 * time.Millisecond
	genericStateWait        = 4000 * time.Millisecond
	genericStatePoll        = 200 * time.Millisecond
	genericStopSoftGrace    = 1500 * time.Millisecond
	genericStopForceGrace   = 4000 * time.Millisecond
	genericStopPoll         = 200 * time.Millisecond
	genericRestartBaseDelay = 500 * time.Millisecond
	genericMaxRestarts      = 3
)

// GenericOptions uses the Host's already resolved Node and resource root.
// No PATH/toolchain discovery is performed by this package.
type GenericOptions struct {
	NodeBin     string
	SamplesRoot string
	OnProgress  func(id, action string, progress GenericProgress)
}

type GenericProgress struct {
	ID      string `json:"id"`
	Op      string `json:"op"`
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
}

type GenericResult struct {
	OK         bool   `json:"ok"`
	Error      string `json:"error"`
	Message    string `json:"message"`
	State      string `json:"state"`
	From       string `json:"from"`
	To         string `json:"to"`
	Components []Row  `json:"components"`
}

type genericProcessExpectation struct {
	ID        string
	Version   string
	EntryFile string
	NodeBin   string
}

type genericStoredRecord struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Previous    string `json:"previous"`
	State       string `json:"state"`
	PID         *int   `json:"pid"`
	URL         string `json:"url"`
	Source      string `json:"source"`
	InstalledAt string `json:"installedAt"`
	UpdatedAt   string `json:"updatedAt"`
	LastError   string `json:"lastError"`
}

// MarshalJSON keeps `previous: null` / `pid: null` parity with the JS registry schema.
func (r genericStoredRecord) MarshalJSON() ([]byte, error) {
	var prev any
	if r.Previous != "" {
		prev = r.Previous
	}
	var pid any
	if r.PID != nil && *r.PID > 0 {
		pid = *r.PID
	}
	return json.Marshal(struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		Previous    any    `json:"previous"`
		State       string `json:"state"`
		PID         any    `json:"pid"`
		URL         string `json:"url"`
		Source      string `json:"source"`
		InstalledAt string `json:"installedAt"`
		UpdatedAt   string `json:"updatedAt"`
		LastError   string `json:"lastError"`
	}{
		ID:          r.ID,
		Name:        r.Name,
		Version:     r.Version,
		Previous:    prev,
		State:       r.State,
		PID:         pid,
		URL:         r.URL,
		Source:      r.Source,
		InstalledAt: r.InstalledAt,
		UpdatedAt:   r.UpdatedAt,
		LastError:   r.LastError,
	})
}

type genericRegistryFile struct {
	Components map[string]genericStoredRecord `json:"components"`
}

type genericPayloadManifest struct {
	ID          string
	Name        string
	Version     string
	Description string
	Kind        string
	Entry       string
	EntryFile   string
	Dir         string
	Source      string
}

type genericCatalogItem struct {
	ID          string
	Name        string
	Description string
	Kind        string
	Source      string
	Versions    []genericPayloadManifest
	Latest      string
}

type genericSpawnSpec struct {
	EntryFile string
	Dir       string
	DataDir   string
	LogFile   string
	ID        string
	Version   string
	NodeBin   string
}

type genericChildHandle interface {
	PID() int
	Exited() <-chan error
	Kill() error
}

type execGenericChild struct {
	cmd    *exec.Cmd
	pid    int
	exited chan error
}

func (c *execGenericChild) PID() int             { return c.pid }
func (c *execGenericChild) Exited() <-chan error { return c.exited }
func (c *execGenericChild) Kill() error {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

type genericRuntimeDeps struct {
	startGrace      time.Duration
	startPoll       time.Duration
	stateWait       time.Duration
	statePoll       time.Duration
	stopSoftGrace   time.Duration
	stopForceGrace  time.Duration
	stopPoll        time.Duration
	restartBaseWait time.Duration
	maxRestarts     int
	pidAlive        func(int) bool
	inspectProcess  func(int) (string, string, bool)
	killPID         func(int, bool) bool
	spawn           func(genericSpawnSpec) (genericChildHandle, error)
	removeAll       func(string) error
	now             func() time.Time
}

func defaultGenericRuntimeDeps() genericRuntimeDeps {
	return genericRuntimeDeps{
		startGrace:      genericStartGrace,
		startPoll:       genericStartPoll,
		stateWait:       genericStateWait,
		statePoll:       genericStatePoll,
		stopSoftGrace:   genericStopSoftGrace,
		stopForceGrace:  genericStopForceGrace,
		stopPoll:        genericStopPoll,
		restartBaseWait: genericRestartBaseDelay,
		maxRestarts:     genericMaxRestarts,
		pidAlive:        genericPIDAlive,
		inspectProcess:  genericInspectProcess,
		killPID:         genericKillPID,
		spawn:           defaultSpawnGenericChild,
		removeAll:       os.RemoveAll,
		now:             time.Now,
	}
}

type genericProc struct {
	mu           sync.Mutex
	child        genericChildHandle
	pid          int
	gen          uint64
	expectedExit bool
	attempts     int
	timer        *time.Timer
	exited       bool
	exitCode     int
}

type genericService struct {
	mu       sync.Mutex
	userData string
	options  GenericOptions
	deps     genericRuntimeDeps
	busy     map[string]bool
	procs    map[string]*genericProc
	gen      uint64
	loaded   bool
	reg      genericRegistryFile
	closed   bool
}

var genericServices = struct {
	sync.Mutex
	byRoot map[string]*genericService
}{byRoot: map[string]*genericService{}}

// ConfigureGeneric does not start the worker or any component. Call with the
// same resolved resources as the Host before showing the component catalog.
func ConfigureGeneric(userData string, options GenericOptions) {
	root := filepath.Clean(userData)
	genericServices.Lock()
	defer genericServices.Unlock()
	if old := genericServices.byRoot[root]; old != nil {
		old.mu.Lock()
		old.options = options
		old.mu.Unlock()
		return
	}
	genericServices.byRoot[root] = newGenericService(root, options, defaultGenericRuntimeDeps())
}

func newGenericService(userData string, options GenericOptions, deps genericRuntimeDeps) *genericService {
	return &genericService{
		userData: filepath.Clean(userData),
		options:  options,
		deps:     deps,
		busy:     map[string]bool{},
		procs:    map[string]*genericProc{},
		reg:      genericRegistryFile{Components: map[string]genericStoredRecord{}},
	}
}

func genericServiceFor(userData string) *genericService {
	genericServices.Lock()
	defer genericServices.Unlock()
	return genericServices.byRoot[filepath.Clean(userData)]
}

func listGeneric(userData string) []Row {
	s := genericServiceFor(userData)
	if s == nil {
		return genericDiskRows(userData, "")
	}
	return s.list()
}

// RunGenericAction preserves the original service result, including a switched
// version whose restart failed. Callers must display Message/State, not only OK.
func RunGenericAction(userData, id, action string) (GenericResult, error) {
	return runGenericTarget(userData, id, action, id, "")
}

// InstallGenericVersion selects an existing catalog version; empty selects latest.
func InstallGenericVersion(userData, id, version string) (GenericResult, error) {
	return runGenericTarget(userData, id, "install", id, version)
}

func runGenericTarget(userData, id, action, targetID, version string) (GenericResult, error) {
	id = strings.TrimSpace(id)
	targetID = strings.TrimSpace(targetID)
	version = strings.TrimSpace(version)
	if !genericID.MatchString(id) || id == "whalebridge" || id == HostComponentID {
		return GenericResult{}, errors.New("invalid-component-id")
	}
	switch action {
	case "install", "start", "stop", "update", "rollback", "uninstall":
	default:
		return GenericResult{}, ErrUnknownAction
	}
	s := genericServiceFor(userData)
	if s == nil {
		return GenericResult{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheComponentRuntimeIsNotConfigured))
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return GenericResult{}, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheComponentServiceIsExiting))
	}

	var result GenericResult
	switch action {
	case "install":
		result = s.install(targetID, version)
	case "start":
		result = s.start(targetID)
	case "stop":
		result = s.stop(targetID)
	case "update":
		result = s.update(targetID)
	case "rollback":
		result = s.rollback(targetID)
	case "uninstall":
		result = s.uninstall(targetID)
	}
	if !result.OK {
		if result.Error == "not-installed" {
			return result, ErrNotInstalled
		}
		return result, fmt.Errorf("%s: %s", result.Error, result.Message)
	}
	if result.State == "error" {
		return result, errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreComponentsTheComponentSwitchedVersionsButFailedT), result.Message))
	}
	return result, nil
}

// ShutdownGeneric belongs only on committed quit, never window close/cancel.
// Terminates supervised and persisted running components directly in Go without
// spawning any Node helper processes.
func ShutdownGeneric(userData string) error {
	s := genericServiceFor(userData)
	if s == nil {
		return nil
	}
	return s.shutdown()
}

func (s *genericService) componentsRoot() string {
	return filepath.Join(s.userData, "components")
}

func (s *genericService) registryPath() string {
	return filepath.Join(s.componentsRoot(), "registry.json")
}

func (s *genericService) componentDir(id string) string {
	return filepath.Join(s.componentsRoot(), id)
}

func (s *genericService) versionsDir(id string) string {
	return filepath.Join(s.componentDir(id), "versions")
}

func (s *genericService) versionDir(id, version string) string {
	return filepath.Join(s.versionsDir(id), version)
}

func (s *genericService) dataDir(id string) string {
	return filepath.Join(s.componentDir(id), "data")
}

func (s *genericService) stateFile(id string) string {
	return filepath.Join(s.dataDir(id), "state.json")
}

func (s *genericService) logFile(id string) string {
	return filepath.Join(s.dataDir(id), "component.log")
}

func (s *genericService) opts() GenericOptions {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.options
}

func (s *genericService) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *genericService) emit(op, id, phase string, percent int, message string) {
	fn := s.opts().OnProgress
	if fn == nil {
		return
	}
	fn(id, op, GenericProgress{ID: id, Op: op, Phase: phase, Percent: percent, Message: message})
}

func (s *genericService) claimBusy(id string) (release func(), ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.busy[id] {
		return nil, false
	}
	s.busy[id] = true
	return func() {
		s.mu.Lock()
		delete(s.busy, id)
		s.mu.Unlock()
	}, true
}

func (s *genericService) ensureLoadedLocked() {
	if s.loaded {
		return
	}
	s.reg = loadGenericRegistry(s.registryPath())
	s.loaded = true
}

func (s *genericService) saveRegistryLocked() error {
	data, err := json.MarshalIndent(s.reg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFileAtomicRetry(s.registryPath(), data, 0o600)
}

func (s *genericService) recordFor(id string) (genericStoredRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	rec, ok := s.reg.Components[id]
	return rec, ok
}

func (s *genericService) saveRecord(rec genericStoredRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	s.reg.Components[rec.ID] = rec
	return s.saveRegistryLocked()
}

func (s *genericService) deleteRecord(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	delete(s.reg.Components, id)
	return s.saveRegistryLocked()
}

func (s *genericService) list() []Row {
	s.reconcile()
	catalog := s.scanCatalog()
	s.mu.Lock()
	s.ensureLoadedLocked()
	regCopy := make(map[string]genericStoredRecord, len(s.reg.Components))
	for id, rec := range s.reg.Components {
		regCopy[id] = rec
	}
	s.mu.Unlock()

	rows := make([]Row, 0, len(catalog)+len(regCopy))
	seen := make(map[string]bool, len(catalog))
	for _, cat := range catalog {
		seen[cat.ID] = true
		var recPtr *genericStoredRecord
		if rec, ok := regCopy[cat.ID]; ok {
			recPtr = &rec
		}
		rows = append(rows, buildGenericRow(&cat, recPtr))
	}
	var orphanIDs []string
	for id := range regCopy {
		if !seen[id] {
			orphanIDs = append(orphanIDs, id)
		}
	}
	sort.Strings(orphanIDs)
	for _, id := range orphanIDs {
		rec := regCopy[id]
		rows = append(rows, buildGenericRow(nil, &rec))
	}
	return rows
}

func (s *genericService) reconcile() {
	type checkTarget struct {
		id  string
		pid int
		exp genericProcessExpectation
	}
	s.mu.Lock()
	s.ensureLoadedLocked()
	nodeBin := s.options.NodeBin
	var checks []checkTarget
	for id, rec := range s.reg.Components {
		if rec.State != "running" || s.procs[id] != nil {
			continue
		}
		pid := 0
		if rec.PID != nil {
			pid = *rec.PID
		}
		checks = append(checks, checkTarget{id: id, pid: pid, exp: s.expectedProcess(id, rec, nodeBin)})
	}
	s.mu.Unlock()

	var deadIDs []string
	for _, c := range checks {
		if !s.isExpectedProcessAlive(c.pid, c.exp) {
			deadIDs = append(deadIDs, c.id)
		}
	}
	if len(deadIDs) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	dirty := false
	for _, id := range deadIDs {
		rec, ok := s.reg.Components[id]
		if !ok || rec.State != "running" || s.procs[id] != nil {
			continue
		}
		rec.State = "stopped"
		rec.PID = nil
		rec.URL = ""
		rec.LastError = storelocale.T(storelocale.KeyStoreComponentsTheProcessExitedLauncherRestartOrExter)
		s.reg.Components[id] = rec
		dirty = true
	}
	if dirty {
		_ = s.saveRegistryLocked()
	}
}

func (s *genericService) expectedProcess(id string, rec genericStoredRecord, nodeBin string) genericProcessExpectation {
	exp := genericProcessExpectation{ID: id, Version: rec.Version, NodeBin: nodeBin}
	if !genericID.MatchString(id) || !validComponentVersionSegment(rec.Version) {
		return exp
	}
	if m, ok := readSafeGenericManifest(s.versionDir(id, rec.Version)); ok {
		exp.EntryFile = m.EntryFile
	}
	return exp
}

func (s *genericService) isExpectedProcessAlive(pid int, exp genericProcessExpectation) bool {
	if pid <= 0 || !s.deps.pidAlive(pid) {
		return false
	}
	return genericMatchProcessIdentity(pid, exp, s.deps.inspectProcess)
}

func (s *genericService) install(id, version string) GenericResult {
	if !genericID.MatchString(id) {
		return GenericResult{OK: false, Error: "invalid-id"}
	}
	release, ok := s.claimBusy(id)
	if !ok {
		return GenericResult{OK: false, Error: "busy"}
	}
	defer release()

	if _, exists := s.recordFor(id); exists {
		return GenericResult{OK: false, Error: "already-installed"}
	}
	cat, found := s.catalogEntry(id)
	if !found {
		return GenericResult{OK: false, Error: "unknown-component"}
	}
	picked, found := resolveCatalogVersion(cat, version)
	if !found {
		return GenericResult{OK: false, Error: "unknown-version"}
	}
	if code, err := s.stagePayload(id, picked, "install"); err != nil {
		s.emit("install", id, "error", 100, err.Error())
		return GenericResult{OK: false, Error: code, Message: err.Error()}
	}
	if err := os.MkdirAll(s.dataDir(id), 0o755); err != nil {
		s.emit("install", id, "error", 100, err.Error())
		return GenericResult{OK: false, Error: "install-failed", Message: err.Error()}
	}
	now := s.deps.now().UTC().Format(time.RFC3339Nano)
	rec := genericStoredRecord{
		ID:          id,
		Name:        cat.Name,
		Version:     picked.Version,
		State:       "installed",
		Source:      picked.Source,
		InstalledAt: now,
		UpdatedAt:   now,
	}
	if err := s.saveRecord(rec); err != nil {
		s.emit("install", id, "error", 100, err.Error())
		return GenericResult{OK: false, Error: "install-failed", Message: err.Error()}
	}
	s.emit("install", id, "done", 100, "")
	return GenericResult{OK: true, State: "installed", Components: s.list()}
}

func (s *genericService) start(id string) GenericResult {
	if !genericID.MatchString(id) {
		return GenericResult{OK: false, Error: "invalid-id"}
	}
	release, ok := s.claimBusy(id)
	if !ok {
		return GenericResult{OK: false, Error: "busy"}
	}
	defer release()
	return s.startComponent(id, "start")
}

func (s *genericService) stop(id string) GenericResult {
	if !genericID.MatchString(id) {
		return GenericResult{OK: false, Error: "invalid-id"}
	}
	release, ok := s.claimBusy(id)
	if !ok {
		return GenericResult{OK: false, Error: "busy"}
	}
	defer release()
	res := s.stopComponent(id, "stop")
	phase := "done"
	msg := ""
	if !res.OK {
		phase = "error"
		msg = res.Error
	}
	s.emit("stop", id, phase, 100, msg)
	return res
}

func (s *genericService) update(id string) GenericResult {
	if !genericID.MatchString(id) {
		return GenericResult{OK: false, Error: "invalid-id"}
	}
	release, ok := s.claimBusy(id)
	if !ok {
		return GenericResult{OK: false, Error: "busy"}
	}
	defer release()

	rec, exists := s.recordFor(id)
	if !exists {
		return GenericResult{OK: false, Error: "not-installed"}
	}
	cat, found := s.catalogEntry(id)
	if !found {
		return GenericResult{OK: false, Error: "unknown-component"}
	}
	latest, found := resolveCatalogVersion(cat, "")
	if !found {
		return GenericResult{OK: false, Error: "unknown-component"}
	}
	from := rec.Version
	if compareVersions(latest.Version, from) <= 0 {
		return GenericResult{OK: false, Error: "no-newer-version", From: from, To: from}
	}

	wasRunning := s.isRecordRunningOrSupervised(id, rec)
	if wasRunning {
		s.emit("update", id, "stop", 10, "")
		stopped := s.stopComponent(id, "update")
		if !stopped.OK {
			return stopped
		}
	}
	if code, err := s.stagePayload(id, latest, "update"); err != nil {
		s.emit("update", id, "error", 100, err.Error())
		return GenericResult{OK: false, Error: code, From: from, To: from, Message: err.Error()}
	}

	oldPrevious := rec.Previous
	rec.Previous = from
	rec.Version = latest.Version
	rec.Name = cat.Name
	rec.State = "installed"
	rec.PID = nil
	rec.URL = ""
	rec.LastError = ""
	rec.UpdatedAt = s.deps.now().UTC().Format(time.RFC3339Nano)
	if err := s.saveRecord(rec); err != nil {
		return GenericResult{OK: false, Error: "update-failed", From: from, To: from, Message: err.Error()}
	}

	if oldPrevious != "" && oldPrevious != rec.Previous && oldPrevious != rec.Version && validComponentVersionSegment(oldPrevious) {
		oldDir := s.versionDir(id, oldPrevious)
		if isContainedResolvedPath(s.versionsDir(id), oldDir, false) {
			_ = s.deps.removeAll(oldDir)
		}
	}

	finalState := rec.State
	if wasRunning {
		restarted := s.startComponent(id, "update")
		if !restarted.OK {
			return GenericResult{OK: true, From: from, To: latest.Version, State: "error", Message: restarted.Message}
		}
		finalState = restarted.State
	}
	s.emit("update", id, "done", 100, "")
	return GenericResult{OK: true, From: from, To: latest.Version, State: finalState}
}

func (s *genericService) rollback(id string) GenericResult {
	if !genericID.MatchString(id) {
		return GenericResult{OK: false, Error: "invalid-id"}
	}
	release, ok := s.claimBusy(id)
	if !ok {
		return GenericResult{OK: false, Error: "busy"}
	}
	defer release()

	rec, exists := s.recordFor(id)
	if !exists {
		return GenericResult{OK: false, Error: "not-installed"}
	}
	if rec.Previous == "" {
		return GenericResult{OK: false, Error: "nothing-to-rollback", To: rec.Version}
	}
	target := rec.Previous
	if !validComponentVersionSegment(target) {
		return GenericResult{OK: false, Error: "rollback-payload-missing", To: rec.Version}
	}
	targetDir := s.versionDir(id, target)
	if !isContainedResolvedPath(s.versionsDir(id), targetDir, false) {
		return GenericResult{OK: false, Error: "rollback-payload-missing", To: rec.Version}
	}
	if info, err := lstatNotReparse(targetDir); err != nil || !info.IsDir() {
		return GenericResult{OK: false, Error: "rollback-payload-missing", To: rec.Version}
	}
	verified, ok := readSafeGenericManifest(targetDir)
	if !ok || verified.ID != id || verified.Version != target {
		return GenericResult{OK: false, Error: "rollback-payload-missing", To: rec.Version}
	}

	wasRunning := s.isRecordRunningOrSupervised(id, rec)
	if wasRunning {
		s.emit("rollback", id, "stop", 15, "")
		stopped := s.stopComponent(id, "rollback")
		if !stopped.OK {
			return stopped
		}
	}
	s.emit("rollback", id, "switch", 60, "")
	rec.Previous = rec.Version
	rec.Version = target
	rec.State = "installed"
	rec.PID = nil
	rec.URL = ""
	rec.LastError = ""
	rec.UpdatedAt = s.deps.now().UTC().Format(time.RFC3339Nano)
	if err := s.saveRecord(rec); err != nil {
		return GenericResult{OK: false, Error: "rollback-failed", To: target, Message: err.Error()}
	}

	finalState := rec.State
	if wasRunning {
		restarted := s.startComponent(id, "rollback")
		if !restarted.OK {
			return GenericResult{OK: true, To: target, State: "error", Message: restarted.Message}
		}
		finalState = restarted.State
	}
	s.emit("rollback", id, "done", 100, "")
	return GenericResult{OK: true, To: target, State: finalState}
}

func (s *genericService) uninstall(id string) GenericResult {
	if !genericID.MatchString(id) {
		return GenericResult{OK: false, Error: "invalid-id"}
	}
	release, ok := s.claimBusy(id)
	if !ok {
		return GenericResult{OK: false, Error: "busy"}
	}
	defer release()

	rec, exists := s.recordFor(id)
	if !exists {
		return GenericResult{OK: false, Error: "not-installed"}
	}
	if rec.State == "running" || s.hasSupervisedProc(id) {
		s.emit("uninstall", id, "stop", 20, "")
		stopped := s.stopComponent(id, "uninstall")
		if !stopped.OK {
			return stopped
		}
	}
	s.emit("uninstall", id, "remove", 60, "")
	versionsRoot := s.versionsDir(id)
	if !isContainedResolvedPath(s.componentsRoot(), versionsRoot, false) {
		rec.LastError = "组件目录路径无效，已拒绝删除。"
		_ = s.saveRecord(rec)
		return GenericResult{OK: false, Error: "remove-failed", Message: rec.LastError}
	}
	if err := s.deps.removeAll(versionsRoot); err != nil {
		rec.LastError = "组件程序文件删除失败，请重试卸载。"
		_ = s.saveRecord(rec)
		return GenericResult{OK: false, Error: "remove-failed", Message: rec.LastError}
	}
	_ = s.deleteRecord(id)
	s.emit("uninstall", id, "done", 100, "")
	return GenericResult{OK: true}
}

func (s *genericService) stagePayload(id string, manifest genericPayloadManifest, op string) (string, error) {
	if !genericID.MatchString(id) {
		return "invalid-id", errors.New("invalid component id")
	}
	if !validComponentVersionSegment(manifest.Version) {
		return "bad-payload", errors.New("invalid component version")
	}
	root := s.componentsRoot()
	versionsRoot := s.versionsDir(id)
	target := s.versionDir(id, manifest.Version)
	if !isContainedResolvedPath(root, versionsRoot, false) || !isContainedResolvedPath(versionsRoot, target, false) {
		return "bad-payload", errors.New("component target escapes component root")
	}

	s.emit(op, id, "copy", 30, "")
	if err := os.MkdirAll(versionsRoot, 0o755); err != nil {
		return "install-failed", err
	}
	for _, checkDir := range []string{root, s.componentDir(id), versionsRoot} {
		if _, err := lstatNotReparse(checkDir); err != nil {
			return "bad-payload", err
		}
	}

	staging, err := os.MkdirTemp(versionsRoot, ".staging-*")
	if err != nil {
		return "install-failed", err
	}
	committed := false
	defer func() {
		if !committed {
			_ = s.deps.removeAll(staging)
		}
	}()

	if err := copyGenericDir(manifest.Dir, staging); err != nil {
		return "bad-payload", err
	}
	s.emit(op, id, "verify", 55, "")
	staged, ok := readSafeGenericManifest(staging)
	if !ok || staged.ID != manifest.ID || staged.Version != manifest.Version || staged.EntryFile == "" {
		return "bad-payload", errors.New("staged payload failed manifest verification")
	}

	s.emit(op, id, "switch", 75, "")
	if err := s.deps.removeAll(target); err != nil {
		return "switch-failed", fmt.Errorf("cannot replace %s: %w", target, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "switch-failed", err
	}
	if err := os.Rename(staging, target); err != nil {
		return "switch-failed", err
	}
	committed = true
	return "", nil
}

func (s *genericService) activePayload(id string, rec genericStoredRecord) (dir string, manifest genericPayloadManifest, ok bool) {
	if !genericID.MatchString(id) || !validComponentVersionSegment(rec.Version) {
		return "", genericPayloadManifest{}, false
	}
	versionsRoot := s.versionsDir(id)
	targetDir := s.versionDir(id, rec.Version)
	if !isContainedResolvedPath(s.componentsRoot(), versionsRoot, false) || !isContainedResolvedPath(versionsRoot, targetDir, false) {
		return "", genericPayloadManifest{}, false
	}
	m, valid := readSafeGenericManifest(targetDir)
	if !valid || m.ID != id || m.Version != rec.Version {
		return "", genericPayloadManifest{}, false
	}
	return targetDir, m, true
}

func (s *genericService) isRecordRunningOrSupervised(id string, rec genericStoredRecord) bool {
	if s.hasSupervisedProc(id) {
		return true
	}
	if rec.State == "running" && rec.PID != nil {
		exp := s.expectedProcess(id, rec, s.opts().NodeBin)
		return s.isExpectedProcessAlive(*rec.PID, exp)
	}
	return false
}

func (s *genericService) hasSupervisedProc(id string) bool {
	s.mu.Lock()
	p := s.procs[id]
	s.mu.Unlock()
	if p == nil {
		return false
	}
	p.mu.Lock()
	exited := p.exited
	p.mu.Unlock()
	return !exited
}

func (s *genericService) startComponent(id, op string) GenericResult {
	rec, exists := s.recordFor(id)
	if !exists {
		return GenericResult{OK: false, Error: "not-installed"}
	}
	nodeBin := s.opts().NodeBin
	exp := s.expectedProcess(id, rec, nodeBin)
	if s.hasSupervisedProc(id) {
		pid := 0
		if rec.PID != nil {
			pid = *rec.PID
		}
		return GenericResult{OK: true, State: "running", Message: rec.LastError, To: rec.URL, Components: nil, From: strconv.Itoa(pid)}
	}
	if rec.State == "running" && rec.PID != nil && s.isExpectedProcessAlive(*rec.PID, exp) {
		return GenericResult{OK: true, State: "running"}
	}

	dir, payload, ok := s.activePayload(id, rec)
	if !ok {
		rec.State = "error"
		rec.LastError = "组件负载缺失（versions 目录被移除或损坏）。"
		_ = s.saveRecord(rec)
		return GenericResult{OK: false, Error: "payload-missing", State: "error", Message: rec.LastError}
	}

	s.emit(op, id, "spawn", 40, "")
	child, err := s.deps.spawn(genericSpawnSpec{
		EntryFile: payload.EntryFile,
		Dir:       dir,
		DataDir:   s.dataDir(id),
		LogFile:   s.logFile(id),
		ID:        id,
		Version:   rec.Version,
		NodeBin:   nodeBin,
	})
	if err != nil {
		rec.State = "error"
		rec.LastError = fmt.Sprintf("启动失败：%v", err)
		_ = s.saveRecord(rec)
		s.emit(op, id, "error", 100, rec.LastError)
		return GenericResult{OK: false, Error: "spawn-failed", State: "error", Message: rec.LastError}
	}

	proc := s.attachSupervision(id, child, 0)
	if earlyErr, exited := s.waitStartGrace(child); exited {
		s.detachProcIfMatch(id, proc)
		rec.State = "error"
		rec.PID = nil
		rec.URL = ""
		if earlyErr != nil {
			rec.LastError = fmt.Sprintf("组件进程启动即退出（%v）。", earlyErr)
		} else {
			rec.LastError = "组件进程启动即退出。"
		}
		_ = s.saveRecord(rec)
		s.emit(op, id, "error", 100, rec.LastError)
		return GenericResult{OK: false, Error: "early-exit", State: "error", Message: rec.LastError}
	}

	s.emit(op, id, "wait", 70, "")
	liveURL, alive := s.waitForStateReady(id, child)
	if !alive {
		s.detachProcIfMatch(id, proc)
		rec.State = "error"
		rec.PID = nil
		rec.URL = ""
		rec.LastError = "组件进程在启动就绪等待期间退出。"
		_ = s.saveRecord(rec)
		return GenericResult{OK: false, Error: "early-exit", State: "error", Message: rec.LastError}
	}

	if s.isClosed() {
		proc.mu.Lock()
		proc.expectedExit = true
		proc.mu.Unlock()
		_ = child.Kill()
		s.detachProcIfMatch(id, proc)
		return GenericResult{OK: false, Error: "service-exiting", Message: storelocale.T(storelocale.KeyStoreComponentsTheComponentServiceIsExiting)}
	}

	pid := child.PID()
	rec.State = "running"
	if pid > 0 {
		p := pid
		rec.PID = &p
	}
	rec.URL = liveURL
	rec.LastError = ""
	if err := s.saveRecord(rec); err != nil {
		return GenericResult{OK: false, Error: "spawn-failed", Message: err.Error()}
	}
	s.emit(op, id, "done", 100, "")
	return GenericResult{OK: true, State: "running"}
}

func (s *genericService) waitStartGrace(child genericChildHandle) (error, bool) {
	if s.deps.startGrace <= 0 {
		select {
		case err := <-child.Exited():
			return err, true
		default:
			return nil, false
		}
	}
	timer := time.NewTimer(s.deps.startGrace)
	defer timer.Stop()
	select {
	case err := <-child.Exited():
		return err, true
	case <-timer.C:
		return nil, false
	}
}

func (s *genericService) waitForStateReady(id string, child genericChildHandle) (url string, alive bool) {
	deadline := time.Now().Add(s.deps.stateWait)
	for {
		select {
		case <-child.Exited():
			return "", false
		default:
		}
		if raw, err := os.ReadFile(s.stateFile(id)); err == nil {
			var st struct {
				PID  int    `json:"pid"`
				Port int    `json:"port"`
				URL  string `json:"url"`
			}
			if json.Unmarshal(raw, &st) == nil && st.PID > 0 {
				resolvedURL := st.URL
				if resolvedURL == "" && st.Port > 0 {
					resolvedURL = fmt.Sprintf("http://127.0.0.1:%d/", st.Port)
				}
				return resolvedURL, true
			}
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		step := s.deps.statePoll
		if step <= 0 || step > remaining {
			step = remaining
		}
		timer := time.NewTimer(step)
		select {
		case <-child.Exited():
			timer.Stop()
			return "", false
		case <-timer.C:
		}
	}
	select {
	case <-child.Exited():
		return "", false
	default:
	}
	return "", s.deps.pidAlive(child.PID())
}

func (s *genericService) attachSupervision(id string, child genericChildHandle, attempts int) *genericProc {
	s.mu.Lock()
	s.gen++
	gen := s.gen
	proc := &genericProc{
		child:    child,
		pid:      child.PID(),
		gen:      gen,
		attempts: attempts,
	}
	s.procs[id] = proc
	s.mu.Unlock()

	go s.superviseChild(id, proc)
	return proc
}

func (s *genericService) detachProcIfMatch(id string, proc *genericProc) {
	proc.mu.Lock()
	proc.expectedExit = true
	if proc.timer != nil {
		proc.timer.Stop()
		proc.timer = nil
	}
	proc.mu.Unlock()

	s.mu.Lock()
	if s.procs[id] == proc {
		delete(s.procs, id)
	}
	s.mu.Unlock()
}

func (s *genericService) superviseChild(id string, proc *genericProc) {
	err := <-proc.child.Exited()
	proc.mu.Lock()
	proc.exited = true
	expected := proc.expectedExit
	attempts := proc.attempts
	proc.mu.Unlock()
	if expected {
		return
	}

	s.mu.Lock()
	if s.closed || s.procs[id] != proc {
		s.mu.Unlock()
		return
	}
	s.ensureLoadedLocked()
	rec, ok := s.reg.Components[id]
	if !ok || rec.State != "running" {
		delete(s.procs, id)
		s.mu.Unlock()
		return
	}
	if attempts >= s.deps.maxRestarts {
		delete(s.procs, id)
		rec.State = "error"
		rec.PID = nil
		rec.URL = ""
		rec.LastError = fmt.Sprintf("组件进程反复退出（%v），已达重试上限。", err)
		s.reg.Components[id] = rec
		_ = s.saveRegistryLocked()
		s.mu.Unlock()
		s.emit("supervise", id, "error", 100, rec.LastError)
		return
	}
	nextAttempts := attempts + 1
	delay := s.deps.restartBaseWait * (1 << uint(nextAttempts-1))
	proc.mu.Lock()
	proc.timer = time.AfterFunc(delay, func() {
		s.restartSupervised(id, proc, nextAttempts)
	})
	proc.mu.Unlock()
	s.mu.Unlock()

	s.emit("supervise", id, "retry", 0, fmt.Sprintf("进程退出（%v），%v 后重启", err, delay))
}

func (s *genericService) restartSupervised(id string, prev *genericProc, attempts int) {
	s.mu.Lock()
	if s.closed || s.procs[id] != prev {
		s.mu.Unlock()
		return
	}
	prev.mu.Lock()
	expected := prev.expectedExit
	prev.mu.Unlock()
	if expected {
		s.mu.Unlock()
		return
	}
	s.ensureLoadedLocked()
	rec, ok := s.reg.Components[id]
	if !ok || rec.State != "running" {
		delete(s.procs, id)
		s.mu.Unlock()
		return
	}
	nodeBin := s.options.NodeBin
	s.mu.Unlock()

	dir, payload, valid := s.activePayload(id, rec)
	if !valid {
		s.mu.Lock()
		delete(s.procs, id)
		if current, exists := s.reg.Components[id]; exists {
			current.State = "error"
			current.LastError = "组件负载缺失，无法重启。"
			s.reg.Components[id] = current
			_ = s.saveRegistryLocked()
		}
		s.mu.Unlock()
		return
	}
	child, err := s.deps.spawn(genericSpawnSpec{
		EntryFile: payload.EntryFile,
		Dir:       dir,
		DataDir:   s.dataDir(id),
		LogFile:   s.logFile(id),
		ID:        id,
		Version:   rec.Version,
		NodeBin:   nodeBin,
	})
	if err != nil {
		s.mu.Lock()
		delete(s.procs, id)
		if current, exists := s.reg.Components[id]; exists {
			current.State = "error"
			current.PID = nil
			current.URL = ""
			current.LastError = fmt.Sprintf("组件重启失败：%v", err)
			s.reg.Components[id] = current
			_ = s.saveRegistryLocked()
		}
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	if s.closed || s.procs[id] != prev {
		s.mu.Unlock()
		_ = child.Kill()
		return
	}
	s.gen++
	next := &genericProc{
		child:    child,
		pid:      child.PID(),
		gen:      s.gen,
		attempts: attempts,
	}
	s.procs[id] = next
	if current, exists := s.reg.Components[id]; exists && child.PID() > 0 {
		pid := child.PID()
		current.PID = &pid
		s.reg.Components[id] = current
		_ = s.saveRegistryLocked()
	}
	s.mu.Unlock()
	go s.superviseChild(id, next)
}

func (s *genericService) stopComponent(id, op string) GenericResult {
	rec, exists := s.recordFor(id)
	if !exists {
		return GenericResult{OK: false, Error: "not-installed"}
	}

	s.mu.Lock()
	proc := s.procs[id]
	pid := 0
	if proc != nil {
		pid = proc.pid
		proc.mu.Lock()
		proc.expectedExit = true
		if proc.timer != nil {
			proc.timer.Stop()
			proc.timer = nil
		}
		proc.mu.Unlock()
	} else if rec.PID != nil {
		pid = *rec.PID
	}
	s.mu.Unlock()

	if rec.State != "running" && proc == nil {
		return GenericResult{OK: true, State: rec.State}
	}

	s.emit(op, id, "stop", 40, "")
	if pid > 0 {
		exp := s.expectedProcess(id, rec, s.opts().NodeBin)
		identityOK := proc != nil || genericMatchProcessIdentity(pid, exp, s.deps.inspectProcess)
		if identityOK {
			s.deps.killPID(pid, false)
			dead := s.waitStopped(pid, proc, exp, s.deps.stopSoftGrace)
			if !dead {
				if !s.deps.killPID(pid, true) && proc != nil {
					_ = proc.child.Kill()
				}
				dead = s.waitStopped(pid, proc, exp, s.deps.stopForceGrace)
			}
			if !dead {
				rec.State = "running"
				p := pid
				rec.PID = &p
				rec.LastError = "组件进程退出尚未确认，请重试停止。"
				_ = s.saveRecord(rec)
				return GenericResult{OK: false, Error: "stop-unconfirmed", State: "running", Message: rec.LastError}
			}
		}
	}

	s.mu.Lock()
	if s.procs[id] == proc {
		delete(s.procs, id)
	}
	s.mu.Unlock()
	rec.State = "stopped"
	rec.PID = nil
	rec.URL = ""
	rec.LastError = ""
	_ = s.saveRecord(rec)
	return GenericResult{OK: true, State: "stopped"}
}

func (s *genericService) waitStopped(pid int, proc *genericProc, exp genericProcessExpectation, grace time.Duration) bool {
	if proc != nil {
		if grace <= 0 {
			select {
			case <-proc.child.Exited():
				return true
			default:
				return !s.deps.pidAlive(pid)
			}
		}
		timer := time.NewTimer(grace)
		defer timer.Stop()
		select {
		case <-proc.child.Exited():
			return true
		case <-timer.C:
			return !s.deps.pidAlive(pid)
		}
	}
	deadline := time.Now().Add(grace)
	for {
		if !s.isExpectedProcessAlive(pid, exp) {
			return true
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		step := s.deps.stopPoll
		if step <= 0 || step > remaining {
			step = remaining
		}
		time.Sleep(step)
	}
	return !s.isExpectedProcessAlive(pid, exp)
}

func (s *genericService) shutdown() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.ensureLoadedLocked()
	nodeBin := s.options.NodeBin
	procsSnapshot := make(map[string]*genericProc, len(s.procs))
	for id, proc := range s.procs {
		proc.mu.Lock()
		proc.expectedExit = true
		if proc.timer != nil {
			proc.timer.Stop()
			proc.timer = nil
		}
		proc.mu.Unlock()
		procsSnapshot[id] = proc
	}
	recordsSnapshot := make([]genericStoredRecord, 0, len(s.reg.Components))
	for _, rec := range s.reg.Components {
		recordsSnapshot = append(recordsSnapshot, rec)
	}
	s.mu.Unlock()

	for _, rec := range recordsSnapshot {
		proc := procsSnapshot[rec.ID]
		pid := 0
		if proc != nil {
			pid = proc.pid
		} else if rec.PID != nil {
			pid = *rec.PID
		}
		stillAlive := false
		if rec.State == "running" && pid > 0 {
			exp := s.expectedProcess(rec.ID, rec, nodeBin)
			if proc != nil || genericMatchProcessIdentity(pid, exp, s.deps.inspectProcess) {
				if !s.deps.killPID(pid, true) && proc != nil {
					_ = proc.child.Kill()
				}
				stillAlive = !s.waitStopped(pid, proc, exp, 200*time.Millisecond)
			}
		} else if proc != nil {
			_ = proc.child.Kill()
		}

		s.mu.Lock()
		if current, ok := s.reg.Components[rec.ID]; ok && current.State == "running" {
			if stillAlive {
				current.LastError = "退出时组件进程终止尚未确认。"
			} else {
				current.State = "stopped"
				current.PID = nil
				current.URL = ""
			}
			s.reg.Components[rec.ID] = current
		}
		s.mu.Unlock()
	}

	s.mu.Lock()
	clear(s.procs)
	err := s.saveRegistryLocked()
	s.mu.Unlock()
	return err
}

func (s *genericService) scanCatalog() []genericCatalogItem {
	samplesRoot := strings.TrimSpace(s.opts().SamplesRoot)
	if samplesRoot == "" {
		return nil
	}
	entries, err := os.ReadDir(samplesRoot)
	if err != nil {
		return nil
	}
	byID := map[string]*genericCatalogItem{}
	for _, idDir := range entries {
		id := idDir.Name()
		if !idDir.IsDir() || !genericID.MatchString(id) || id == "whalebridge" || id == HostComponentID {
			continue
		}
		idPath := filepath.Join(samplesRoot, id)
		if info, err := lstatNotReparse(idPath); err != nil || !info.IsDir() || !isContainedResolvedPath(samplesRoot, idPath, false) {
			continue
		}
		versions, err := os.ReadDir(idPath)
		if err != nil {
			continue
		}
		for _, vDir := range versions {
			ver := vDir.Name()
			if !vDir.IsDir() || !validComponentVersionSegment(ver) {
				continue
			}
			dir := filepath.Join(idPath, ver)
			if info, err := lstatNotReparse(dir); err != nil || !info.IsDir() || !isContainedResolvedPath(idPath, dir, false) {
				continue
			}
			m, ok := readSafeGenericManifest(dir)
			if !ok || m.ID != id || m.Version != ver {
				continue
			}
			item := byID[id]
			if item == nil {
				item = &genericCatalogItem{
					ID:          m.ID,
					Name:        m.Name,
					Description: m.Description,
					Kind:        m.Kind,
					Source:      m.Source,
				}
				byID[id] = item
			}
			item.Versions = append(item.Versions, m)
		}
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]genericCatalogItem, 0, len(ids))
	for _, id := range ids {
		item := byID[id]
		sort.Slice(item.Versions, func(i, j int) bool {
			return compareVersions(item.Versions[i].Version, item.Versions[j].Version) < 0
		})
		if len(item.Versions) == 0 {
			continue
		}
		latest := item.Versions[len(item.Versions)-1]
		item.Latest = latest.Version
		item.Name = latest.Name
		item.Kind = latest.Kind
		if item.Description == "" {
			item.Description = item.Versions[0].Description
		}
		if latest.Description != "" {
			item.Description = latest.Description
		}
		out = append(out, *item)
	}
	return out
}

func (s *genericService) catalogEntry(id string) (genericCatalogItem, bool) {
	for _, row := range s.scanCatalog() {
		if row.ID == id {
			return row, true
		}
	}
	return genericCatalogItem{}, false
}

func resolveCatalogVersion(item genericCatalogItem, version string) (genericPayloadManifest, bool) {
	if len(item.Versions) == 0 {
		return genericPayloadManifest{}, false
	}
	if version == "" {
		return item.Versions[len(item.Versions)-1], true
	}
	for _, v := range item.Versions {
		if v.Version == version {
			return v, true
		}
	}
	return genericPayloadManifest{}, false
}

func buildGenericRow(cat *genericCatalogItem, rec *genericStoredRecord) Row {
	id := ""
	if cat != nil {
		id = cat.ID
	} else if rec != nil {
		id = rec.ID
	}
	name := id
	if cat != nil && cat.Name != "" {
		name = cat.Name
	} else if rec != nil && rec.Name != "" {
		name = rec.Name
	}
	version := ""
	if cat != nil {
		version = cat.Latest
	} else if rec != nil {
		version = rec.Version
	}
	installedVersion := ""
	previousVersion := ""
	state := "available"
	source := "installed"
	if cat != nil {
		source = cat.Source
	} else if rec != nil && rec.Source != "" {
		source = rec.Source
	}
	kind := "service"
	if cat != nil && cat.Kind != "" {
		kind = cat.Kind
	}
	desc := ""
	if cat != nil {
		desc = cat.Description
	}
	pid := 0
	url := ""
	msg := ""
	if rec != nil {
		installedVersion = rec.Version
		previousVersion = rec.Previous
		state = rec.State
		msg = rec.LastError
		if rec.State == "running" {
			if rec.PID != nil {
				pid = *rec.PID
			}
			url = rec.URL
		}
	}
	updateAvail := rec != nil && cat != nil && cat.Latest != "" && compareVersions(cat.Latest, rec.Version) > 0
	return Row{
		ID:               id,
		Name:             name,
		Description:      desc,
		Version:          version,
		InstalledVersion: installedVersion,
		PreviousVersion:  previousVersion,
		Kind:             kind,
		Source:           source,
		State:            state,
		UpdateAvailable:  updateAvail,
		Message:          msg,
		PID:              pid,
		URL:              url,
		Orphan:           cat == nil,
	}
}

func loadGenericRegistry(file string) genericRegistryFile {
	out := genericRegistryFile{Components: map[string]genericStoredRecord{}}
	raw, err := os.ReadFile(file)
	if err != nil {
		return out
	}
	var parsed genericRegistryFile
	if json.Unmarshal(raw, &parsed) != nil {
		_ = os.Rename(file, file+".broken")
		return out
	}
	for id, rec := range parsed.Components {
		if !genericID.MatchString(id) || !validComponentVersionSegment(rec.Version) {
			continue
		}
		if !validComponentVersionSegment(rec.Previous) {
			rec.Previous = ""
		}
		switch rec.State {
		case "installed", "stopped", "running", "error":
		default:
			rec.State = "installed"
		}
		if rec.PID != nil && *rec.PID <= 0 {
			rec.PID = nil
		}
		rec.ID = id
		if len(rec.Name) > 120 {
			rec.Name = rec.Name[:120]
		}
		if len(rec.URL) > 200 {
			rec.URL = rec.URL[:200]
		}
		if len(rec.Source) > 40 {
			rec.Source = rec.Source[:40]
		}
		if len(rec.LastError) > 500 {
			rec.LastError = rec.LastError[:500]
		}
		out.Components[id] = rec
	}
	return out
}

func readSafeGenericManifest(dir string) (genericPayloadManifest, bool) {
	m, ok := readGenericManifest(dir)
	if !ok {
		return genericPayloadManifest{}, false
	}
	if !validComponentVersionSegment(m.Version) {
		return genericPayloadManifest{}, false
	}
	entryFile := genericEntry(dir, m.Entry)
	if entryFile == "" || !isContainedResolvedPath(dir, entryFile, false) {
		return genericPayloadManifest{}, false
	}
	info, err := lstatNotReparse(entryFile)
	if err != nil || !info.Mode().IsRegular() {
		return genericPayloadManifest{}, false
	}
	name := m.Name
	if len(name) > 120 {
		name = name[:120]
	}
	desc := m.Description
	if len(desc) > 400 {
		desc = desc[:400]
	}
	return genericPayloadManifest{
		ID:          m.ID,
		Name:        name,
		Version:     m.Version,
		Description: desc,
		Kind:        m.Kind,
		Entry:       m.Entry,
		EntryFile:   entryFile,
		Dir:         dir,
		Source:      "bundled",
	}, true
}

func resolveExistingAncestorRealpath(target string) string {
	current, err := filepath.Abs(target)
	if err != nil {
		current = filepath.Clean(target)
	}
	var trailing []string
	for {
		realPath, err := filepath.EvalSymlinks(current)
		if err == nil {
			if absReal, absErr := filepath.Abs(realPath); absErr == nil {
				realPath = absReal
			}
			for i := len(trailing) - 1; i >= 0; i-- {
				realPath = filepath.Join(realPath, trailing[i])
			}
			return filepath.Clean(realPath)
		}
		if !os.IsNotExist(err) {
			return filepath.Clean(target)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(target)
		}
		trailing = append(trailing, filepath.Base(current))
		current = parent
	}
}

func isContainedResolvedPath(baseDir, targetPath string, allowEqual bool) bool {
	realBase := resolveExistingAncestorRealpath(baseDir)
	realTarget := resolveExistingAncestorRealpath(targetPath)
	if runtime.GOOS == "windows" {
		realBase = strings.ToLower(strings.TrimPrefix(realBase, `\\?\`))
		realTarget = strings.ToLower(strings.TrimPrefix(realTarget, `\\?\`))
	}
	rel, err := filepath.Rel(realBase, realTarget)
	if err != nil {
		return false
	}
	if rel == "." || rel == "" {
		return allowEqual
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func copyGenericDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		from := filepath.Join(src, entry.Name())
		to := filepath.Join(dst, entry.Name())
		info, err := lstatNotReparse(from)
		if err != nil {
			if isReparseOrSymlink(info) || strings.Contains(err.Error(), "symlink") || strings.Contains(err.Error(), "junction") {
				continue
			}
			return err
		}
		if info.IsDir() {
			if err := copyGenericDir(from, to); err != nil {
				return err
			}
			continue
		}
		if info.Mode().IsRegular() {
			if err := copyRegularFile(from, to); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyRegularFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func matchCommandLineIdentity(exePath, cmdLine string, expected genericProcessExpectation) bool {
	haystack := strings.TrimSpace(cmdLine + " " + exePath)
	if runtime.GOOS == "windows" {
		haystack = strings.ToLower(haystack)
	}
	if expected.EntryFile != "" {
		entryAbs, err := filepath.Abs(expected.EntryFile)
		if err != nil {
			entryAbs = filepath.Clean(expected.EntryFile)
		}
		entryBase := filepath.Base(entryAbs)
		idNorm := expected.ID
		if runtime.GOOS == "windows" {
			entryAbs = strings.ToLower(entryAbs)
			entryBase = strings.ToLower(entryBase)
			idNorm = strings.ToLower(idNorm)
		}
		if strings.Contains(haystack, entryAbs) {
			return true
		}
		if idNorm != "" && strings.Contains(haystack, idNorm) && strings.Contains(haystack, entryBase) {
			return true
		}
		return false
	}
	if expected.ID != "" {
		idNorm := expected.ID
		if runtime.GOOS == "windows" {
			idNorm = strings.ToLower(idNorm)
		}
		return strings.Contains(haystack, idNorm)
	}
	return true
}

func defaultSpawnGenericChild(spec genericSpawnSpec) (genericChildHandle, error) {
	if strings.TrimSpace(spec.NodeBin) == "" {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreComponentsTheComponentNeedsTheHostNodeRuntimeCh))
	}
	if err := os.MkdirAll(spec.DataDir, 0o755); err != nil {
		return nil, err
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if st, err := os.Stat(spec.LogFile); err == nil && st.Size() > genericLogMaxBytes {
		flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	}
	logF, err := os.OpenFile(spec.LogFile, flags, 0o600)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(spec.NodeBin, spec.EntryFile)
	processutil.BackgroundTree(cmd)
	cmd.Dir = spec.Dir
	cmd.Env = sanitizeComponentEnv(os.Environ(),
		"ELECTRON_RUN_AS_NODE=1",
		"LAUNCHER_COMPONENT_ID="+spec.ID,
		"LAUNCHER_COMPONENT_VERSION="+spec.Version,
		"LAUNCHER_COMPONENT_DATA_DIR="+spec.DataDir,
	)
	cmd.Stdout = logF
	cmd.Stderr = logF
	if err := cmd.Start(); err != nil {
		_ = logF.Close()
		return nil, err
	}
	_ = logF.Close()

	exited := make(chan error, 1)
	go func() {
		exited <- cmd.Wait()
		close(exited)
	}()
	return &execGenericChild{cmd: cmd, pid: cmd.Process.Pid, exited: exited}, nil
}
