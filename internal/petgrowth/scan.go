package petgrowth

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed scan.cjs
var scanScript string

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
	root string
	dirs map[string]scanDir
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
		return Usage{}, errors.New("pet usage scan unavailable or incomplete")
	}
	var result struct {
		Dirs     []scanResultDir `json:"dirs"`
		Complete bool            `json:"complete"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return Usage{}, errors.New("invalid pet usage scan result")
	}
	return mergeScanUsage(root, result.Dirs, result.Complete)
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
	samples := map[string]int64{}
	cacheInclusive := map[string]int64{}
	for _, d := range dirs {
		entry := scanDir{sig: d.S, samples: d.D, cacheInclusive: d.C}
		if d.D == nil {
			cached, ok := scanCache.dirs[d.R]
			if !ok || cached.sig != d.S {
				scanCache.dirs = map[string]scanDir{}
				return Usage{}, errors.New("pet usage scan cache out of step")
			}
			entry = cached
		}
		next[d.R] = entry
		for id, n := range entry.samples {
			samples[id] = n
			delete(cacheInclusive, id)
			if old := entry.cacheInclusive[id]; old > n && old <= MaxCount {
				cacheInclusive[id] = old
			}
		}
	}
	scanCache.dirs = next
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
