package petgrowth

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed scan.cjs
var scanScript string

var logGenerationPattern = regexp.MustCompile(`^session(?:\.v([1-9][0-9]*))?\.jsonl\.zstd$`)

// scanDir is one session directory's settlements and the log signature they
// were decoded from.
type scanDir struct {
	sig            string
	samples        map[string]int64
	cacheInclusive map[string]int64
}

// scanResultDir is one directory as the scanner reports it; D is nil when the
// directory's signature matched the one it was given.
type scanResultDir struct {
	R string           `json:"r"`
	S string           `json:"s"`
	D map[string]int64 `json:"d"`
	C map[string]int64 `json:"c"`
}

// scanCache keeps the last scan per directory. The minute refresh used to
// decompress every session log each time; with a large history it read and
// decoded the decode budget's 512 MiB and then failed, every minute. Unchanged
// directories are now reused, and a scan that runs out of budget keeps the
// directories it finished for the next one.
var scanCache struct {
	sync.Mutex
	root     string
	dirs     map[string]scanDir
	order    []string
	goRoot   string
	goSigs   map[string]string
	complete bool
}

// Scan reads only the authorized app DSH home's sessions directory. The caller
// supplies existing native config, never URL/page parameters. Node executes our
// embedded scanner, not a package from disk. No IPC/Host token is passed.
func Scan(ctx context.Context, node, home string) (map[string]int64, error) {
	usage, err := scanSessionUsage(ctx, node, home)
	return usage.Samples, err
}

func scanSessionUsage(ctx context.Context, node, home string) (Usage, error) {
	if node == "" || home == "" || !filepath.IsAbs(home) {
		return Usage{}, errors.New("pet scanner runtime unavailable")
	}
	root := filepath.Join(home, "sessions")
	fastUsage, goSigs, handled, fastErr := tryFastSessionScan(root)
	if handled {
		return fastUsage, fastErr
	}
	prior := map[string]string{}
	scanCache.Lock()
	if scanCache.root == root {
		for rel, d := range scanCache.dirs {
			prior[rel] = d.sig
		}
	}
	scanCache.Unlock()
	priorJSON, err := json.Marshal(prior)
	if err != nil {
		return Usage{}, err
	}
	script := strings.Replace(scanScript, "'use strict';", "'use strict';\nconst PET_SCAN_PRIOR = "+string(priorJSON)+";", 1)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "--max-old-space-size=512", "--input-type=commonjs", "-", root)
	hideScanner(cmd)
	cmd.Stdin = strings.NewReader(script)
	// Avoid NODE_OPTIONS/NODE_PATH loaders and unrelated credentials inherited
	// by the shell. The fixed builtin-only script needs just platform temp/root.
	cmd.Env = []string{}
	for _, key := range []string{"SystemRoot", "WINDIR", "TEMP", "TMP", "TMPDIR", "LANG"} {
		if value, ok := os.LookupEnv(key); ok {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	var out cappedOutput
	cmd.Stdout = &out
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		scanCache.Lock()
		scanCache.complete = false
		scanCache.goSigs = nil
		scanCache.Unlock()
		return Usage{}, errors.New("pet usage scan unavailable or incomplete")
	}
	var result struct {
		Dirs     []scanResultDir `json:"dirs"`
		Complete bool            `json:"complete"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		scanCache.Lock()
		scanCache.complete = false
		scanCache.goSigs = nil
		scanCache.Unlock()
		return Usage{}, errors.New("invalid pet usage scan result")
	}
	usage, err := mergeScanUsage(root, result.Dirs, result.Complete)
	if err != nil {
		scanCache.Lock()
		scanCache.complete = false
		scanCache.goSigs = nil
		scanCache.Unlock()
		return Usage{}, err
	}
	scanCache.Lock()
	if scanCache.root == root {
		scanCache.goRoot = root
		scanCache.goSigs = goSigs
		scanCache.complete = true
	}
	scanCache.Unlock()
	return usage, nil
}

func logGeneration(name string) int {
	m := logGenerationPattern.FindStringSubmatch(name)
	if m == nil {
		return -1
	}
	if m[1] == "" {
		return 0
	}
	v, err := strconv.Atoi(m[1])
	if err != nil {
		return -1
	}
	return v
}

func walkSessionLogSigs(root string) (map[string]string, error) {
	rootStat, err := os.Lstat(root)
	if err != nil || !rootStat.IsDir() || rootStat.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("pet usage scan unavailable or incomplete")
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, errors.New("pet usage scan unavailable or incomplete")
	}
	withinRoot := func(realPath string) (string, bool) {
		rel, err := filepath.Rel(realRoot, realPath)
		if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", false
		}
		if rel == "." {
			return "", true
		}
		return rel, true
	}

	out := map[string]string{}
	stack := []string{realRoot}
	dirs, files := 0, 0
	for len(stack) > 0 {
		dir := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		dirs++
		if dirs > 50000 {
			return nil, errors.New("scan directory limit")
		}
		ds, err := os.Lstat(dir)
		if err != nil || !ds.IsDir() || ds.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("invalid session directory")
		}
		realDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return nil, errors.New("invalid session directory")
		}
		dr, ok := withinRoot(realDir)
		if !ok {
			return nil, errors.New("invalid session directory")
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, errors.New("invalid session directory")
		}
		var (
			hasBest  bool
			bestGen  int
			bestName string
			bestSize int64
			bestMod  int64
		)
		for _, ent := range entries {
			if ent.Type()&os.ModeSymlink != 0 {
				continue
			}
			full := filepath.Join(dir, ent.Name())
			if ent.IsDir() {
				stack = append(stack, full)
				continue
			}
			gen := logGeneration(ent.Name())
			if gen < 0 {
				continue
			}
			files++
			if files > 20000 {
				return nil, errors.New("scan file limit")
			}
			st, err := os.Lstat(full)
			if err != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || st.Size() > 128*1024*1024 {
				return nil, errors.New("invalid session file")
			}
			realFile, err := filepath.EvalSymlinks(full)
			if err != nil {
				return nil, errors.New("invalid session file")
			}
			if _, ok := withinRoot(realFile); !ok {
				return nil, errors.New("invalid session file")
			}
			if !hasBest || gen > bestGen {
				hasBest = true
				bestGen = gen
				bestName = ent.Name()
				bestSize = st.Size()
				bestMod = st.ModTime().UnixNano()
			}
		}
		if hasBest {
			out[dr] = fmt.Sprintf("%s:%d:%d", bestName, bestSize, bestMod)
		}
	}
	return out, nil
}

func tryFastSessionScan(root string) (Usage, map[string]string, bool, error) {
	goSigs, err := walkSessionLogSigs(root)
	if err != nil {
		return Usage{}, nil, true, errors.New("pet usage scan unavailable or incomplete")
	}
	if len(goSigs) == 0 {
		scanCache.Lock()
		scanCache.root = root
		scanCache.goRoot = root
		scanCache.dirs = map[string]scanDir{}
		scanCache.order = nil
		scanCache.goSigs = map[string]string{}
		scanCache.complete = true
		scanCache.Unlock()
		return Usage{Samples: map[string]int64{}, Source: "sessions", CacheInclusive: map[string]int64{}}, nil, true, nil
	}
	scanCache.Lock()
	defer scanCache.Unlock()
	if scanCache.root != root || scanCache.goRoot != root || !scanCache.complete || len(scanCache.goSigs) != len(goSigs) {
		return Usage{}, goSigs, false, nil
	}
	for k, v := range goSigs {
		if scanCache.goSigs[k] != v {
			return Usage{}, goSigs, false, nil
		}
	}
	samples := map[string]int64{}
	cacheInclusive := map[string]int64{}
	for _, rel := range scanCache.order {
		entry := scanCache.dirs[rel]
		for id, n := range entry.samples {
			samples[id] = n
			delete(cacheInclusive, id)
			if old := entry.cacheInclusive[id]; old > n && old <= MaxCount {
				cacheInclusive[id] = old
			}
		}
	}
	if len(samples) > 200000 {
		return Usage{}, nil, true, errors.New("pet usage scan settlement limit")
	}
	if _, err := ValidateSamples(samples); err != nil {
		return Usage{}, nil, true, err
	}
	return Usage{Samples: samples, Source: "sessions", CacheInclusive: cacheInclusive}, goSigs, true, nil
}

// mergeScan folds directory results in scan order, as the scanner once did in
// one pass: a later directory's settlement replaces an earlier equal id. A
// directory reported unchanged (nil samples) is served from the cache.
func mergeScan(root string, dirs []scanResultDir, complete bool) (map[string]int64, error) {
	usage, err := mergeScanUsage(root, dirs, complete)
	return usage.Samples, err
}

func mergeScanUsage(root string, dirs []scanResultDir, complete bool) (Usage, error) {
	scanCache.Lock()
	defer scanCache.Unlock()
	if scanCache.root != root {
		scanCache.root, scanCache.dirs = root, map[string]scanDir{}
	}
	next := make(map[string]scanDir, len(dirs))
	order := make([]string, 0, len(dirs))
	samples := map[string]int64{}
	cacheInclusive := map[string]int64{}
	for _, d := range dirs {
		entry := scanDir{sig: d.S, samples: d.D, cacheInclusive: d.C}
		if d.D == nil {
			cached, ok := scanCache.dirs[d.R]
			if !ok || cached.sig != d.S {
				scanCache.dirs = map[string]scanDir{}
				scanCache.order = nil
				return Usage{}, errors.New("pet usage scan cache out of step")
			}
			entry = cached
		}
		next[d.R] = entry
		order = append(order, d.R)
		for id, n := range entry.samples {
			samples[id] = n
			delete(cacheInclusive, id)
			if old := entry.cacheInclusive[id]; old > n && old <= MaxCount {
				cacheInclusive[id] = old
			}
		}
	}
	scanCache.dirs = next
	scanCache.order = order
	if !complete {
		return Usage{}, errors.New("pet usage scan unavailable or incomplete")
	}
	if len(samples) > 200000 {
		return Usage{}, errors.New("pet usage scan settlement limit")
	}
	if _, err := ValidateSamples(samples); err != nil {
		return Usage{}, err
	}
	return Usage{Samples: samples, Source: "sessions", CacheInclusive: cacheInclusive}, nil
}

type cappedOutput struct{ bytes.Buffer }

func (b *cappedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 32*1024*1024 {
		return 0, errors.New("pet usage scan result limit")
	}
	return b.Buffer.Write(p)
}
