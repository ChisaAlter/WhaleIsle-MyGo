package media

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// diskCache keeps rasterised outputs (PNG bytes) under dir/<2 hex>/<hex>.png
// within a byte budget, least recently used first out: a hit refreshes the
// file's modification time, and pruning removes the oldest files.
type diskCache struct {
	dir    string
	budget int64

	mu      sync.Mutex
	scanned bool
	total   int64
}

func newDiskCache(dir string, budget int64) *diskCache {
	return &diskCache{dir: dir, budget: budget}
}

func (d *diskCache) path(name string) string {
	return filepath.Join(d.dir, name[:2], name+".png")
}

func (d *diskCache) get(name string) []byte {
	if d == nil || len(name) < 3 {
		return nil
	}
	p := d.path(name)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	now := time.Now()
	_ = os.Chtimes(p, now, now)
	return data
}

func (d *diskCache) put(name string, data []byte) {
	if d == nil || len(name) < 3 || int64(len(data)) > d.budget/4 {
		return
	}
	p := d.path(name)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".media-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), p) != nil {
		_ = os.Remove(tmp.Name())
		return
	}
	d.mu.Lock()
	d.total += int64(len(data))
	over := !d.scanned || d.total > d.budget
	d.mu.Unlock()
	if over {
		d.prune()
	}
}

// prune measures the cache and removes the least recently used files until
// it fits. Only files this cache names (.png, .media-* leftovers) are touched.
func (d *diskCache) prune() {
	type file struct {
		path string
		size int64
		mod  time.Time
	}
	var files []file
	var total int64
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return
	}
	for _, shard := range entries {
		if !shard.IsDir() || len(shard.Name()) != 2 {
			continue
		}
		dir := filepath.Join(d.dir, shard.Name())
		items, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, item := range items {
			name := item.Name()
			if item.IsDir() || item.Type()&os.ModeSymlink != 0 || !(strings.HasSuffix(name, ".png") || strings.HasPrefix(name, ".media-")) {
				continue
			}
			info, err := item.Info()
			if err != nil {
				continue
			}
			if strings.HasPrefix(name, ".media-") && time.Since(info.ModTime()) > time.Hour {
				_ = os.Remove(filepath.Join(dir, name))
				continue
			}
			files = append(files, file{filepath.Join(dir, name), info.Size(), info.ModTime()})
			total += info.Size()
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	for _, f := range files {
		if total <= d.budget*9/10 {
			break
		}
		if os.Remove(f.path) == nil {
			total -= f.size
		}
	}
	d.mu.Lock()
	d.scanned, d.total = true, total
	d.mu.Unlock()
}
