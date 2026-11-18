package components

// WhaleBridge's launcher component lifecycle. The Electron reference keeps a
// signed two-file release (WhaleBridge-component.json plus
// WhaleBridge-win32-x64.exe), verifies it before publishing a version
// directory, and only then changes the installed pointer. This Go service
// mirrors that contract while leaving process and HTTP seams injectable for
// the desktop host.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/update"
	"whaleisle/internal/processutil"
	"whaleisle/internal/store/storelocale"
)

const (
	whaleBridgeID       = "whalebridge"
	whaleBridgeManifest = "WhaleBridge-component.json"
	whaleBridgeAsset    = "WhaleBridge-win32-x64.exe"
	whaleBridgeTag      = "whalebridge-v"
	whaleBridgeGitHub   = "https://github.com/ChisaAlter/WhaleBridge"
	whaleBridgeCNB      = "https://cnb.cool/ayasealter/WhaleIsle/-"
)

var strictVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// PlatformSpec is the signed asset metadata in WhaleBridge-component.json.
type PlatformSpec struct {
	Asset  string `json:"asset"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// WhaleBridgeManifest is the release manifest downloaded before the binary.
type WhaleBridgeManifest struct {
	ID        string                  `json:"id"`
	Version   string                  `json:"version"`
	License   string                  `json:"license"`
	Platforms map[string]PlatformSpec `json:"platforms"`
}

// WhaleBridgeCatalog is the cached release selection. Local is used by the
// development package path; URL is the already route-pinned asset URL.
type WhaleBridgeCatalog struct {
	Manifest WhaleBridgeManifest `json:"manifest"`
	URL      string              `json:"url,omitempty"`
	Local    string              `json:"local,omitempty"`
	Route    string              `json:"route,omitempty"`
}

type WhaleBridgeInstalled struct {
	Version  string `json:"version"`
	Previous string `json:"previous"`
}

// Progress is emitted during install/update/rollback operations.
type Progress struct {
	ID      string
	Phase   string
	Percent int
	Message string
}

// ServiceOptions controls a WhaleBridgeService. Platform and Arch are test
// seams; empty values select the current process values. FetchJSON and the
// runtime hooks let callers exercise the release protocol without launching a
// native process.
type ServiceOptions struct {
	Root         string
	DshHome      string
	DevPackage   string
	Route        string
	HTTPClient   *http.Client
	Platform     string
	Arch         string
	ReadyTimeout time.Duration
	FetchJSON    func(context.Context, string, http.Header) ([]byte, error)
	StartRuntime func(context.Context, string, string) error
	StopRuntime  func(context.Context, string) error
	Disconnect   func(context.Context, string, string) error
}

// WhaleBridgeService owns the component's version pointer and data directory.
type WhaleBridgeService struct {
	root         string
	dshHome      string
	devPackage   string
	route        string
	client       *http.Client
	platform     string
	arch         string
	readyTimeout time.Duration
	fetchJSON    func(context.Context, string, http.Header) ([]byte, error)
	startRuntime func(context.Context, string, string) error
	stopRuntime  func(context.Context, string) error
	disconnect   func(context.Context, string, string) error
	mu           sync.Mutex
}

// NewWhaleBridgeService creates a lifecycle service rooted at userData's
// components/whalebridge directory.
func NewWhaleBridgeService(opts ServiceOptions) *WhaleBridgeService {
	root := strings.TrimSpace(opts.Root)
	if root == "" {
		root = filepath.Join(defaultComponentsRoot(opts.Root), whaleBridgeID)
	}
	dsh := strings.TrimSpace(opts.DshHome)
	if dsh == "" {
		dsh = filepath.Join(filepath.Dir(filepath.Dir(root)), "dsh-home")
	}
	platform := opts.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	arch := opts.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}
	client := bridgeHardenedHTTPClient(opts.HTTPClient, 30*time.Second)
	s := &WhaleBridgeService{
		root: root, dshHome: dsh, devPackage: strings.TrimSpace(opts.DevPackage),
		route: normalizeRoute(opts.Route), client: client, platform: platform,
		arch: arch, readyTimeout: opts.ReadyTimeout,
	}
	if s.readyTimeout <= 0 {
		s.readyTimeout = 30 * time.Second
	}
	s.fetchJSON = opts.FetchJSON
	if s.fetchJSON == nil {
		s.fetchJSON = s.defaultFetchJSON
	}
	s.startRuntime = opts.StartRuntime
	if s.startRuntime == nil {
		s.startRuntime = func(_ context.Context, root, dsh string) error {
			return startWhalebridge(root, dsh)
		}
	}
	s.stopRuntime = opts.StopRuntime
	if s.stopRuntime == nil {
		s.stopRuntime = func(_ context.Context, root string) error { return stopWhalebridge(root) }
	}
	s.disconnect = opts.Disconnect
	if s.disconnect == nil {
		s.disconnect = func(ctx context.Context, root, version string) error {
			return disconnectWhalebridge(ctx, root, version)
		}
	}
	return s
}

func defaultComponentsRoot(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if ud := strings.TrimSpace(os.Getenv("WHALEISLE_USER_DATA")); ud != "" {
		return filepath.Join(ud, "components")
	}
	if cfg, err := os.UserConfigDir(); err == nil {
		return filepath.Join(cfg, "Deepseek-Harness-Desktop", "components")
	}
	return filepath.Join(".", "components")
}

func normalizeRoute(route string) string {
	route = strings.ToLower(strings.TrimSpace(route))
	if route == "gitee" {
		route = "cnb"
	}
	if route != "cnb" && route != "github" {
		route = "github"
	}
	return route
}

// ValidateWhaleBridgeManifest applies the release manifest invariants from the
// reference launcher. It rejects substituted assets and unbounded payloads.
func ValidateWhaleBridgeManifest(m WhaleBridgeManifest) error {
	p, ok := m.Platforms["win32-x64"]
	if m.ID != whaleBridgeID || !strictVersion.MatchString(m.Version) || !ok ||
		p.Asset != whaleBridgeAsset || !digestPattern.MatchString(strings.ToLower(p.SHA256)) ||
		p.Size <= 0 || p.Size > 200*1024*1024 || !strings.Contains(m.License, "MIT License") {
		return errors.New("invalid WhaleBridge release manifest")
	}
	return nil
}

// Installed returns the persisted component pointer, or nil when absent.
func (s *WhaleBridgeService) Installed() (*WhaleBridgeInstalled, error) {
	b, err := os.ReadFile(filepath.Join(s.root, "installed.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rec WhaleBridgeInstalled
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, fmt.Errorf("invalid installed record: %w", err)
	}
	if strings.TrimSpace(rec.Version) == "" {
		return nil, nil
	}
	return &rec, nil
}

// Catalog returns the cached catalog if valid, otherwise nil.
func (s *WhaleBridgeService) Catalog() (*WhaleBridgeCatalog, error) {
	b, err := os.ReadFile(filepath.Join(s.root, "catalog.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cat WhaleBridgeCatalog
	if err := json.Unmarshal(b, &cat); err != nil {
		return nil, err
	}
	if err := ValidateWhaleBridgeManifest(cat.Manifest); err != nil {
		return nil, err
	}
	return &cat, nil
}

// RefreshCatalog resolves the selected GitHub/CNB route and caches its
// validated manifest and route-pinned asset URL.
func (s *WhaleBridgeService) RefreshCatalog(ctx context.Context) (Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cat, err := s.refreshCatalog(ctx)
	if err != nil {
		return Row{}, err
	}
	return s.row(cat), nil
}

func (s *WhaleBridgeService) refreshCatalog(ctx context.Context) (*WhaleBridgeCatalog, error) {
	var cat *WhaleBridgeCatalog
	var err error
	if s.devPackage != "" {
		manifestPath := filepath.Join(s.devPackage, whaleBridgeManifest)
		m, err := readManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		if err := ValidateWhaleBridgeManifest(m); err != nil {
			return nil, err
		}
		cat = &WhaleBridgeCatalog{Manifest: m, Local: s.devPackage, Route: "development"}
	} else if s.route == "cnb" {
		cat, err = s.refreshCNB(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		cat, err = s.refreshGitHub(ctx)
		if err != nil {
			return nil, err
		}
	}
	if err := writeJSONAtomic(filepath.Join(s.root, "catalog.json"), cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func (s *WhaleBridgeService) refreshGitHub(ctx context.Context) (*WhaleBridgeCatalog, error) {
	base := whaleBridgeGitHub
	m, err := s.fetchManifest(ctx, base+"/releases/latest/download/"+whaleBridgeManifest)
	if err != nil {
		return nil, err
	}
	if err := ValidateWhaleBridgeManifest(m); err != nil {
		return nil, err
	}
	return &WhaleBridgeCatalog{Manifest: m, URL: base + "/releases/download/" + whaleBridgeTag + m.Version + "/" + whaleBridgeAsset, Route: "github"}, nil
}

type whaleRelease struct {
	TagName    string       `json:"tag_name"`
	Draft      bool         `json:"draft"`
	Prerelease bool         `json:"prerelease"`
	Assets     []whaleAsset `json:"assets"`
}
type whaleAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Alt  string `json:"download_url"`
	Size int64  `json:"size"`
}

func (s *WhaleBridgeService) refreshCNB(ctx context.Context) (*WhaleBridgeCatalog, error) {
	body, err := s.fetchJSON(ctx, whaleBridgeCNB+"/releases?page_size=100", http.Header{"Accept": []string{"application/vnd.cnb.api+json"}})
	if err != nil {
		return nil, err
	}
	var releases []whaleRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("invalid CNB releases response: %w", err)
	}
	var picked *whaleRelease
	for i := range releases {
		r := &releases[i]
		if r.Draft || r.Prerelease || !strings.HasPrefix(r.TagName, whaleBridgeTag) {
			continue
		}
		if picked == nil || update.CompareVersions(strings.TrimPrefix(r.TagName, whaleBridgeTag), strings.TrimPrefix(picked.TagName, whaleBridgeTag)) > 0 {
			picked = r
		}
	}
	if picked == nil {
		return nil, errors.New("no WhaleBridge release on selected route")
	}
	var manifestAsset, exeAsset *whaleAsset
	for i := range picked.Assets {
		a := &picked.Assets[i]
		switch a.Name {
		case whaleBridgeManifest:
			manifestAsset = a
		case whaleBridgeAsset:
			exeAsset = a
		}
	}
	if manifestAsset == nil || exeAsset == nil {
		return nil, errors.New("WhaleBridge release is missing manifest or asset")
	}
	manifestURL := manifestAsset.URL
	if manifestURL == "" {
		manifestURL = manifestAsset.Alt
	}
	m, err := s.fetchManifest(ctx, manifestURL)
	if err != nil {
		return nil, err
	}
	if err := ValidateWhaleBridgeManifest(m); err != nil {
		return nil, err
	}
	p := m.Platforms["win32-x64"]
	if picked.TagName != whaleBridgeTag+m.Version || exeAsset.Size != p.Size {
		return nil, errors.New("release and WhaleBridge manifest disagree")
	}
	assetURL := exeAsset.URL
	if assetURL == "" {
		assetURL = exeAsset.Alt
	}
	if assetURL == "" {
		return nil, errors.New("WhaleBridge asset URL is empty")
	}
	return &WhaleBridgeCatalog{Manifest: m, URL: assetURL, Route: "cnb"}, nil
}

func (s *WhaleBridgeService) fetchManifest(ctx context.Context, url string) (WhaleBridgeManifest, error) {
	b, err := s.fetchJSON(ctx, url, http.Header{"Accept": []string{"application/json"}})
	if err != nil {
		return WhaleBridgeManifest{}, err
	}
	var m WhaleBridgeManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return WhaleBridgeManifest{}, fmt.Errorf("invalid WhaleBridge manifest JSON: %w", err)
	}
	return m, nil
}

func (s *WhaleBridgeService) defaultFetchJSON(ctx context.Context, url string, headers http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

func readManifest(path string) (WhaleBridgeManifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return WhaleBridgeManifest{}, err
	}
	var m WhaleBridgeManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return WhaleBridgeManifest{}, err
	}
	return m, nil
}

func writeJSONAtomic(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomicRetry(path, append(b, '\n'), 0o600)
}

type operationLockPayload struct {
	PID       int    `json:"pid"`
	Token     string `json:"token"`
	CreatedAt string `json:"createdAt"`
}

func readOperationLockPayload(path string) (operationLockPayload, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return operationLockPayload{}, err
	}
	var p operationLockPayload
	if json.Unmarshal(b, &p) == nil && p.PID > 0 {
		return p, nil
	}
	var pid int
	if _, scanErr := fmt.Sscan(strings.TrimSpace(string(b)), &pid); scanErr == nil && pid > 0 {
		return operationLockPayload{PID: pid}, nil
	}
	return operationLockPayload{}, errors.New("unrecognized operation lock contents")
}

func randomLockToken() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}

// acquireOperationLock prevents two launcher processes from switching the
// same component generation at once. A stale lock is reclaimed only when its
// owner is provably absent; otherwise the operation fails closed.
func (s *WhaleBridgeService) acquireOperationLock() (func(), error) {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return nil, err
	}
	if _, err := lstatNotReparse(s.root); err != nil {
		return nil, err
	}
	path := filepath.Join(s.root, "operation.lock")
	payload := operationLockPayload{
		PID:       os.Getpid(),
		Token:     randomLockToken(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	encoded = append(encoded, '\n')

	for attempt := 0; attempt < 3; attempt++ {
		if existingInfo, lerr := os.Lstat(path); lerr == nil && isReparseOrSymlink(existingInfo) {
			return nil, errors.New("WhaleBridge operation lock is an unsafe symlink or junction")
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, writeErr := f.Write(encoded)
			syncErr := f.Sync()
			closeErr := f.Close()
			if writeErr != nil || syncErr != nil || closeErr != nil {
				_ = os.Remove(path)
				if writeErr != nil {
					return nil, writeErr
				}
				if syncErr != nil {
					return nil, syncErr
				}
				return nil, closeErr
			}
			return func() {
				if cur, curErr := readOperationLockPayload(path); curErr == nil && cur.Token == payload.Token {
					_ = os.Remove(path)
				}
			}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		info, statErr := os.Lstat(path)
		if statErr != nil {
			continue
		}
		if isReparseOrSymlink(info) {
			return nil, errors.New("WhaleBridge operation lock is an unsafe symlink or junction")
		}
		cur, parseErr := readOperationLockPayload(path)
		if parseErr != nil {
			// Give a concurrent creator a moment to finish writing PID metadata.
			if time.Since(info.ModTime()) < 5*time.Second {
				time.Sleep(50 * time.Millisecond)
				cur, parseErr = readOperationLockPayload(path)
			}
		}
		stale := false
		if parseErr == nil {
			if cur.PID != os.Getpid() && !genericPIDAlive(cur.PID) {
				stale = true
			}
		} else if time.Since(info.ModTime()) > 5*time.Second {
			stale = true
		}
		if !stale {
			return nil, errors.New("WhaleBridge operation busy")
		}
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return nil, errors.New("WhaleBridge operation busy")
		}
	}
	return nil, errors.New("WhaleBridge operation busy")
}

func (s *WhaleBridgeService) row(cat *WhaleBridgeCatalog) Row {
	rec, _ := s.Installed()
	state := "available"
	installedVersion := ""
	previous := ""
	if rec != nil {
		state = "stopped"
		installedVersion, previous = rec.Version, rec.Previous
		if readStatePID(filepath.Join(s.root, "data", "state.json")) {
			state = "running"
		}
	}
	version := "1.0.0"
	route := "official"
	updateAvailable := false
	if cat != nil {
		version = cat.Manifest.Version
		route = cat.Route
		if route == "development" {
			route = "development"
		}
		updateAvailable = rec != nil && update.CompareVersions(version, rec.Version) > 0
	} else if installedVersion != "" {
		version = installedVersion
	}
	return Row{ID: whaleBridgeID, Name: "WhaleBridge", Description: "Native API gateway component", Version: version,
		InstalledVersion: installedVersion, PreviousVersion: previous, Kind: "service", Source: route,
		State: state, UpdateAvailable: updateAvailable}
}

// Row returns current component state and cached catalog metadata.
func (s *WhaleBridgeService) Row() Row {
	cat, _ := s.Catalog()
	return s.row(cat)
}

// Install downloads/verifies/releases a first WhaleBridge version, then starts
// it. A failed first start leaves the verified version installed for retry.
func (s *WhaleBridgeService) Install(ctx context.Context, onProgress func(Progress)) (Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	release, err := s.acquireOperationLock()
	if err != nil {
		return Row{}, err
	}
	defer release()
	if rec, _ := s.Installed(); rec != nil {
		return Row{}, errors.New("WhaleBridge already installed")
	}
	if s.platform != "windows" && s.platform != "win32" {
		return Row{}, errors.New("WhaleBridge installation requires Windows x64")
	}
	if s.arch != "amd64" && s.arch != "x64" {
		return Row{}, errors.New("WhaleBridge installation requires Windows x64")
	}
	cat, err := s.refreshCatalog(ctx)
	if err != nil {
		return Row{}, err
	}
	version, err := s.stage(ctx, cat, onProgress)
	if err != nil {
		return Row{}, err
	}
	if err := writeJSONAtomic(filepath.Join(s.root, "installed.json"), WhaleBridgeInstalled{Version: version}); err != nil {
		return Row{}, err
	}
	if err := s.startRuntime(ctx, s.root, s.dshHome); err != nil {
		msg := "WhaleBridge installed but failed to start: " + err.Error()
		if tail := readLogTail(filepath.Join(s.root, "data", "component.log")); tail != "" {
			msg += "\n" + tail
		}
		return Row{}, errors.New(msg)
	}
	return s.row(cat), nil
}

// Update stages and verifies the newer payload before stopping the current
// process; a failed restart restores the previous installed pointer.
func (s *WhaleBridgeService) Update(ctx context.Context, onProgress func(Progress)) (Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	release, err := s.acquireOperationLock()
	if err != nil {
		return Row{}, err
	}
	defer release()
	rec, err := s.Installed()
	if err != nil || rec == nil {
		if err != nil {
			return Row{}, err
		}
		return Row{}, ErrNotInstalled
	}
	cat, err := s.refreshCatalog(ctx)
	if err != nil {
		return Row{}, err
	}
	if update.CompareVersions(cat.Manifest.Version, rec.Version) <= 0 {
		return Row{}, errors.New("no newer WhaleBridge version")
	}
	if _, err := s.stage(ctx, cat, onProgress); err != nil {
		return Row{}, err
	}
	return s.switchVersion(ctx, rec.Version, cat.Manifest.Version, rec.Previous, onProgress)
}

// Rollback switches to the one retained previous version and retains the
// rolled-forward version for a redo.
func (s *WhaleBridgeService) Rollback(ctx context.Context, onProgress func(Progress)) (Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	release, err := s.acquireOperationLock()
	if err != nil {
		return Row{}, err
	}
	defer release()
	rec, err := s.Installed()
	if err != nil {
		return Row{}, err
	}
	if rec == nil || rec.Previous == "" {
		return Row{}, errors.New("no WhaleBridge version to roll back")
	}
	if !validComponentVersionSegment(rec.Previous) {
		return Row{}, errors.New("invalid WhaleBridge rollback version")
	}
	if err := s.verifyInstalledVersionPayload(rec.Previous); err != nil {
		return Row{}, fmt.Errorf("rollback verification failed: %w", err)
	}
	return s.switchVersion(ctx, rec.Version, rec.Previous, rec.Previous, onProgress)
}

func (s *WhaleBridgeService) verifyInstalledVersionPayload(version string) error {
	if !validComponentVersionSegment(version) {
		return errors.New("invalid WhaleBridge version")
	}
	versionsDir := filepath.Join(s.root, "versions")
	if info, err := lstatNotReparse(versionsDir); err != nil || !info.IsDir() {
		return errors.New("WhaleBridge versions directory missing or unsafe")
	}
	dir := filepath.Join(versionsDir, version)
	if info, err := lstatNotReparse(dir); err != nil || !info.IsDir() {
		return errors.New("WhaleBridge version directory missing or unsafe")
	}
	manifestPath := filepath.Join(dir, whaleBridgeManifest)
	mInfo, err := lstatNotReparse(manifestPath)
	if err != nil || !mInfo.Mode().IsRegular() {
		return errors.New("WhaleBridge version manifest missing or unsafe")
	}
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var m WhaleBridgeManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("invalid WhaleBridge manifest JSON: %w", err)
	}
	if m.ID != whaleBridgeID || m.Version != version {
		return errors.New("WhaleBridge manifest identity or version mismatch")
	}
	p, ok := m.Platforms["win32-x64"]
	if !ok || p.Size <= 0 || !digestPattern.MatchString(strings.ToLower(p.SHA256)) || (p.Asset != "" && p.Asset != whaleBridgeAsset) {
		return errors.New("WhaleBridge manifest platform metadata invalid")
	}
	binPath := filepath.Join(dir, whaleBridgeAsset)
	bInfo, err := lstatNotReparse(binPath)
	if err != nil || !bInfo.Mode().IsRegular() {
		return errors.New("WhaleBridge binary missing or unsafe")
	}
	return verifyFile(binPath, p.Size, p.SHA256)
}

func (s *WhaleBridgeService) switchVersion(ctx context.Context, oldVersion, newVersion, oldPrevious string, onProgress func(Progress)) (Row, error) {
	stateLive := readStatePID(filepath.Join(s.root, "data", "state.json"))
	if stateLive {
		emitProgress(onProgress, whaleBridgeID, "stop", 15, "")
		if err := s.stopRuntime(ctx, s.root); err != nil {
			return Row{}, err
		}
	}
	if err := writeJSONAtomic(filepath.Join(s.root, "installed.json"), WhaleBridgeInstalled{Version: newVersion, Previous: oldVersion}); err != nil {
		return Row{}, err
	}
	if err := s.startRuntime(ctx, s.root, s.dshHome); err != nil {
		_ = s.stopRuntime(ctx, s.root)
		restoreErr := writeJSONAtomic(filepath.Join(s.root, "installed.json"), WhaleBridgeInstalled{Version: oldVersion, Previous: oldPrevious})
		if restartErr := s.startRuntime(ctx, s.root, s.dshHome); restartErr != nil {
			return Row{}, fmt.Errorf("switch failed and original restart failed: %v (original: %w)", restartErr, err)
		}
		if restoreErr != nil {
			return Row{}, fmt.Errorf("%s: %v (original: %w)", fmt.Sprintf(storelocale.T(storelocale.KeyStoreWhalebridgeRollbackRecordFailed), oldVersion), restoreErr, err)
		}
		return Row{}, fmt.Errorf("switch failed; restored previous version: %w", err)
	}
	emitProgress(onProgress, whaleBridgeID, "done", 100, "")
	return s.Row(), nil
}

// Uninstall disconnects the running gateway before removing payloads. The data
// directory is retained unless removeData is explicitly requested.
func (s *WhaleBridgeService) Uninstall(ctx context.Context, removeData bool, onProgress func(Progress)) (Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	release, err := s.acquireOperationLock()
	if err != nil {
		return Row{}, err
	}
	defer release()
	rec, err := s.Installed()
	if err != nil {
		return Row{}, err
	}
	if rec == nil {
		return Row{}, ErrNotInstalled
	}
	if readStatePID(filepath.Join(s.root, "data", "state.json")) {
		if err := s.stopRuntime(ctx, s.root); err != nil {
			return Row{}, err
		}
	}
	staged, err := os.MkdirTemp(s.root, ".uninstall-")
	if err != nil {
		return Row{}, err
	}
	stagedVersions := filepath.Join(staged, "versions")
	versions := filepath.Join(s.root, "versions")
	hasVersions := false
	if _, statErr := os.Lstat(versions); statErr == nil {
		if err := os.Rename(versions, stagedVersions); err != nil {
			_ = os.RemoveAll(staged)
			return Row{}, err
		}
		hasVersions = true
	} else if !errors.Is(statErr, os.ErrNotExist) {
		_ = os.RemoveAll(staged)
		return Row{}, statErr
	}
	if err := os.Remove(filepath.Join(s.root, "installed.json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		var restoreVersionsErr error
		if hasVersions {
			restoreVersionsErr = os.Rename(stagedVersions, versions)
		}
		if restoreVersionsErr != nil {
			return Row{}, fmt.Errorf("%s: %v (original: %w)", fmt.Sprintf(storelocale.T(storelocale.KeyStoreWhalebridgeRollbackRetained), staged), restoreVersionsErr, err)
		}
		_ = os.RemoveAll(staged)
		return Row{}, err
	}
	disconnectRoot := staged
	if hasVersions {
		disconnectRoot = stagedVersions
	}
	if err := s.disconnect(ctx, disconnectRoot, rec.Version); err != nil {
		restoreRecordErr := writeJSONAtomic(filepath.Join(s.root, "installed.json"), *rec)
		var restoreVersionsErr error
		if hasVersions {
			restoreVersionsErr = os.Rename(stagedVersions, versions)
		}
		if restoreErr := errors.Join(restoreRecordErr, restoreVersionsErr); restoreErr != nil {
			// Keep the payload where it is. Deleting it here would destroy the
			// only copy when the rename back into place failed.
			return Row{}, fmt.Errorf("%s: %v (original: %w)", fmt.Sprintf(storelocale.T(storelocale.KeyStoreWhalebridgeRollbackRetained), staged), restoreErr, err)
		}
		_ = os.RemoveAll(staged)
		return Row{}, err
	}
	_ = os.RemoveAll(staged)
	if removeData {
		if err := os.RemoveAll(filepath.Join(s.root, "data")); err != nil {
			return Row{}, err
		}
	}
	emitProgress(onProgress, whaleBridgeID, "done", 100, "")
	return s.Row(), nil
}

func (s *WhaleBridgeService) stage(ctx context.Context, cat *WhaleBridgeCatalog, onProgress func(Progress)) (string, error) {
	if cat == nil {
		return "", errors.New("WhaleBridge catalog unavailable")
	}
	m := cat.Manifest
	if err := ValidateWhaleBridgeManifest(m); err != nil {
		return "", err
	}
	p := m.Platforms["win32-x64"]
	staging := filepath.Join(s.root, "versions", fmt.Sprintf(".staging-%d-%d", os.Getpid(), time.Now().UnixNano()))
	if err := os.RemoveAll(staging); err != nil {
		return "", err
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)
	emitProgress(onProgress, whaleBridgeID, "download", 10, "")
	file := filepath.Join(staging, whaleBridgeAsset)
	if cat.Local != "" {
		if err := copyFile(filepath.Join(cat.Local, whaleBridgeAsset), file); err != nil {
			return "", err
		}
	} else {
		if err := s.downloadAsset(ctx, cat.URL, file, p.Size, p.SHA256); err != nil {
			return "", err
		}
	}
	emitProgress(onProgress, whaleBridgeID, "verify", 60, "")
	if err := verifyFile(file, p.Size, p.SHA256); err != nil {
		return "", err
	}
	if err := writeJSONAtomic(filepath.Join(staging, whaleBridgeManifest), m); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(staging, "LICENSE"), []byte(m.License), 0o600); err != nil {
		return "", err
	}
	target := filepath.Join(s.root, "versions", m.Version)
	if err := os.RemoveAll(target); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(staging, target); err != nil {
		return "", err
	}
	return m.Version, nil
}

func (s *WhaleBridgeService) downloadAsset(ctx context.Context, url, path string, expectedSize int64, expectedSHA string) error {
	if strings.TrimSpace(url) == "" {
		return errors.New("empty WhaleBridge asset URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("WhaleBridge download HTTP %d", resp.StatusCode)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, 200<<20+1))
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n != expectedSize || hex.EncodeToString(h.Sum(nil)) != strings.ToLower(expectedSHA) {
		return fmt.Errorf("WhaleBridge asset size or SHA256 verification failed")
	}
	return nil
}

func verifyFile(path string, expectedSize int64, expectedSHA string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, 200<<20+1))
	if err != nil {
		return err
	}
	if n != expectedSize || hex.EncodeToString(h.Sum(nil)) != strings.ToLower(expectedSHA) {
		return errors.New("WhaleBridge asset size or SHA256 verification failed")
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
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

func emitProgress(fn func(Progress), id, phase string, percent int, message string) {
	if fn != nil {
		fn(Progress{ID: id, Phase: phase, Percent: percent, Message: message})
	}
}

func readLogTail(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(b))
	if len(text) > 1200 {
		text = text[len(text)-1200:]
	}
	return text
}

func disconnectComponentRoot(root string) string {
	componentRoot := root
	base := filepath.Base(root)
	if base == "versions" && strings.HasPrefix(filepath.Base(filepath.Dir(root)), ".uninstall-") {
		componentRoot = filepath.Dir(filepath.Dir(root))
	} else if strings.HasPrefix(base, ".uninstall-") {
		componentRoot = filepath.Dir(root)
	}
	return componentRoot
}

func disconnectWhalebridge(ctx context.Context, root, version string) error {
	if version == "" {
		return nil
	}
	// The native disconnect command is Windows-only. Keep a deterministic
	// no-op on other platforms so uninstall can be statically exercised.
	if runtime.GOOS != "windows" {
		return nil
	}
	exe := filepath.Join(root, "versions", version, whaleBridgeAsset)
	if _, err := os.Stat(exe); err != nil {
		// Uninstall passes the temporary versions root after it has been
		// atomically moved out of <component>/versions.
		exe = filepath.Join(root, version, whaleBridgeAsset)
		if _, secondErr := os.Stat(exe); secondErr != nil {
			return err
		}
	}
	componentRoot := disconnectComponentRoot(root)
	dshHome := filepath.Join(filepath.Dir(filepath.Dir(componentRoot)), "dsh-home")
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := processutil.CommandContext(ctx, exe, "whalebridge-disconnect")
	processutil.Background(cmd)
	cmd.Dir = filepath.Dir(exe)
	cmd.Env = sanitizeComponentEnv(os.Environ(),
		"LAUNCHER_COMPONENT_ID="+whaleBridgeID,
		"LAUNCHER_COMPONENT_DATA_DIR="+filepath.Join(componentRoot, "data"),
		"WHALEBRIDGE_DSH_HOME="+dshHome,
		"MAGPIE_ADDR=127.0.0.1:3427",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("whalebridge-disconnect: %w: %s", err, msg)
		}
		return fmt.Errorf("whalebridge-disconnect: %w", err)
	}
	return nil
}
