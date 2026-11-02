package harness

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"whaleisle/internal/startuptrace"
)

// Runtime warm-ahead.
//
// A cold desktop start is dominated by the Host's own module loading: the staged
// runtime holds ~26k files (17k of them JavaScript) behind ~3.1k junctions, and a
// node.exe start that has never touched them pays file-open/scan latency once per
// file, serially. On the packaged stage the Host boot stage measures ~9s when the
// tree is cached and ~100s on first touch (fresh install, or after the file cache
// was evicted).
//
// warmRuntimeTree reads the staged module files in a bounded worker pool while
// node.exe boots. It is read-only: it never changes the launch plan, never skips
// validation or plugin setup, and does nothing when the tree is absent. Warm, the
// pass takes a few hundred milliseconds and overlaps the Host boot; cold, it turns
// the Host's serial first-touch into a parallel one.
//
// WHALEISLE_STARTUP_WARM=0 disables it (for A/B measurement only).

const warmEnv = "WHALEISLE_STARTUP_WARM"

// warmMaxFile bounds one file; a larger blob is not part of module loading.
const warmMaxFile = 24 << 20

// warmMaxTotal bounds one pass so a pathological tree cannot read forever.
const warmMaxTotal = 1536 << 20

// warmExtensions are the file kinds node reads while resolving and loading the
// Host and its desktop plugin overlays.
var warmExtensions = map[string]bool{
	".js": true, ".mjs": true, ".cjs": true, ".json": true,
	".node": true, ".wasm": true, ".yml": true, ".yaml": true,
}

var errWarmSkip = errors.New("warm: skipped")

type warmupRun struct {
	cancel  chan struct{}
	done    chan struct{}
	files   atomic.Int64
	bytes   atomic.Int64
	skipped atomic.Int64
	started time.Time
	once    sync.Once
}

// warmRuntimeTree starts the read-ahead for one runtime vendor directory and
// returns nil when it is disabled, the tree is missing, or the tree answers from
// the file cache already (every start after the first one).
func warmRuntimeTree(vendorDir string) *warmupRun {
	if vendorDir == "" || warmupDisabled() {
		return nil
	}
	if st, err := os.Stat(vendorDir); err != nil || !st.IsDir() {
		return nil
	}
	cold, probe := warmCacheCold(vendorDir)
	startuptrace.Mark("runtime.warmup.gate", "cold="+strconv.FormatBool(cold)+
		" probeUs="+strconv.FormatInt(probe.Microseconds(), 10))
	if !cold {
		return nil
	}
	w := &warmupRun{cancel: make(chan struct{}), done: make(chan struct{}), started: time.Now()}
	startuptrace.Mark("runtime.warmup.started", vendorDir)
	go w.run(vendorDir)
	return w
}

// warmCacheCold samples staged modules spread across the entry layer and the
// dependency store. A cold tree needs milliseconds per file (uncached read plus
// on-access scan); a cached tree answers in microseconds. Sampling keeps every
// warm start free of the read-ahead's background I/O.
func warmCacheCold(vendorDir string) (bool, time.Duration) {
	paths := warmSamplePaths(vendorDir)
	if len(paths) < 4 {
		return false, 0
	}
	buf := make([]byte, 64<<10)
	start := time.Now()
	for _, path := range paths {
		_, _ = readThrough(path, buf)
	}
	per := time.Since(start) / time.Duration(len(paths))
	return per > 2*time.Millisecond, per
}

// warmSamplePaths returns a few module files from each layer the Host loads.
func warmSamplePaths(vendorDir string) []string {
	var paths []string
	add := func(dir string, max int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		taken := 0
		for _, entry := range entries {
			if taken >= max || len(paths) >= 12 {
				return
			}
			// Probe only regular files, just like the full warm-ahead plan.
			if entry.Type() != 0 || !warmExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
				continue
			}
			paths = append(paths, filepath.Join(dir, entry.Name()))
			taken++
		}
	}
	harness := filepath.Join(vendorDir, "deepseek-harness")
	add(filepath.Join(harness, "apps", "cli", "lib"), 3)
	add(filepath.Join(harness, "apps", "web", "dist"), 2)
	add(filepath.Join(harness, "packages", "boot"), 2)
	instances, err := os.ReadDir(filepath.Join(harness, "node_modules", ".dsh-instances"))
	if err == nil && len(instances) > 0 {
		step := len(instances) / 4
		if step < 1 {
			step = 1
		}
		for i := 0; i < len(instances) && len(paths) < 12; i += step {
			if !instances[i].IsDir() {
				continue
			}
			dir := filepath.Join(harness, "node_modules", ".dsh-instances", instances[i].Name())
			add(dir, 1)
			add(filepath.Join(dir, "lib"), 1)
		}
	}
	return paths
}

func warmupDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(warmEnv))) {
	case "0", "false", "off", "no":
		return true
	}
	return false
}

// Stop cancels the pass; it is safe to call more than once and from any goroutine.
func (w *warmupRun) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() { close(w.cancel) })
}

func (w *warmupRun) run(vendorDir string) {
	defer close(w.done)
	paths := collectWarmPaths(vendorDir, w.cancel)
	if len(paths) == 0 {
		return
	}
	sort.SliceStable(paths, func(i, j int) bool { return warmPriority(paths[i]) < warmPriority(paths[j]) })
	startuptrace.Mark("runtime.warmup.plan", "files="+strconv.Itoa(len(paths)))
	workers := runtime.NumCPU() / 4
	if workers > 4 {
		workers = 4
	}
	if workers < 2 {
		workers = 2
	}
	jobs := make(chan string)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			buf := make([]byte, 128<<10)
			for path := range jobs {
				select {
				case <-w.cancel:
					return
				default:
				}
				if w.bytes.Load() >= warmMaxTotal {
					w.skipped.Add(1)
					continue
				}
				n, err := readThrough(path, buf)
				if err != nil {
					w.skipped.Add(1)
					continue
				}
				w.files.Add(1)
				w.bytes.Add(n)
			}
		}()
	}
dispatch:
	for _, path := range paths {
		select {
		case <-w.cancel:
			break dispatch
		case jobs <- path:
		}
	}
	close(jobs)
	wg.Wait()
	startuptrace.Mark("runtime.warmup.done", "files="+strconv.FormatInt(w.files.Load(), 10)+
		" bytes="+strconv.FormatInt(w.bytes.Load(), 10)+
		" skipped="+strconv.FormatInt(w.skipped.Load(), 10)+
		" ms="+strconv.FormatInt(time.Since(w.started).Milliseconds(), 10))
}

// readThrough reads a file once through buf so the bytes reach the file cache
// and the platform's on-access scanner.
func readThrough(path string, buf []byte) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	if info.Size() > warmMaxFile {
		return 0, errWarmSkip
	}
	var total int64
	for {
		n, err := f.Read(buf)
		total += int64(n)
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

// collectWarmPaths walks the runtime without following junctions and returns the
// module-like files it finds.
func collectWarmPaths(root string, cancel chan struct{}) []string {
	var paths []string
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if depth > 24 {
			return
		}
		select {
		case <-cancel:
			return
		default:
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			name := entry.Name()
			path := filepath.Join(dir, name)
			// ReadDir reports link types without following them (and uses lstat
			// for unknown types on Unix). Windows junctions are ModeIrregular.
			// Admit only plain directories and regular files, before suffix checks.
			typ := entry.Type()
			if typ != 0 && typ != os.ModeDir {
				continue
			}
			if typ.IsDir() {
				if strings.EqualFold(name, ".git") || strings.EqualFold(name, ".cache") {
					continue
				}
				walk(path, depth+1)
				continue
			}
			if !warmExtensions[strings.ToLower(filepath.Ext(name))] {
				continue
			}
			paths = append(paths, path)
		}
	}
	walk(root, 0)
	return paths
}

// warmPriority orders the read-ahead by what node.exe imports first: the CLI and
// web entry points, the boot kernel, then the desktop overlays, then the rest of
// the dependency tree.
func warmPriority(path string) int {
	p := strings.ToLower(filepath.ToSlash(path))
	switch {
	case strings.Contains(p, "/apps/cli/"), strings.Contains(p, "/apps/web/"):
		return 0
	case strings.Contains(p, "/packages/boot/"), strings.Contains(p, "/packages/core/"):
		return 1
	case strings.Contains(p, "/vendor/dsh-"), strings.Contains(p, "/vendor/dshbot/"):
		return 2
	}
	return 3
}
