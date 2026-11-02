package session

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// viewTrace counts why the Snapshot revision advanced. It is a developer
// diagnostic selected with WHALEISLE_VIEW_TRACE=1: a per-frame caller that is
// usually invisible in a CPU profile shows up here as hundreds of samples.
// Unset, noteViewInvalidation returns after one boolean test.
var viewTraceOn = os.Getenv("WHALEISLE_VIEW_TRACE") != ""

var viewTrace struct {
	mu     sync.Mutex
	counts map[string]int
	total  int
}

// noteViewInvalidation records the five callers above invalidateViewLocked.
func noteViewInvalidation() {
	if !viewTraceOn {
		return
	}
	var pcs [12]uintptr
	n := runtime.Callers(3, pcs[:])
	if n == 0 {
		return
	}
	frames := runtime.CallersFrames(pcs[:n])
	var parts []string
	for len(parts) < 5 {
		f, more := frames.Next()
		if f.Function != "" && !strings.HasPrefix(f.Function, "runtime.") {
			parts = append(parts, shortFuncName(f.Function))
		}
		if !more {
			break
		}
	}
	key := strings.Join(parts, " <- ")
	viewTrace.mu.Lock()
	if viewTrace.counts == nil {
		viewTrace.counts = map[string]int{}
	}
	viewTrace.counts[key]++
	viewTrace.total++
	viewTrace.mu.Unlock()
}

func shortFuncName(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// ViewTraceReport lists the invalidation callers, most frequent first.
func ViewTraceReport() string {
	if !viewTraceOn {
		return ""
	}
	viewTrace.mu.Lock()
	defer viewTrace.mu.Unlock()
	type row struct {
		key   string
		count int
	}
	rows := make([]row, 0, len(viewTrace.counts))
	for key, count := range viewTrace.counts {
		rows = append(rows, row{key, count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].count != rows[j].count {
			return rows[i].count > rows[j].count
		}
		return rows[i].key < rows[j].key
	})
	var b strings.Builder
	fmt.Fprintf(&b, "viewTrace total=%d distinct=%d\n", viewTrace.total, len(rows))
	for i, r := range rows {
		if i >= 24 {
			break
		}
		fmt.Fprintf(&b, "%8d  %s\n", r.count, r.key)
	}
	return b.String()
}
