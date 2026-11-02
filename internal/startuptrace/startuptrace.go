// Package startuptrace records opt-in phase timings for the desktop boot chain:
// process start, config load, runtime/profile staging, Host spawn, handshake and
// Ready. It is inert unless WHALEISLE_STARTUP_TRACE names the JSON output file,
// so a normal run pays one getenv and nothing else — the same style as the
// ui_diagnostics hooks in cmd/whaleisle.
//
//	WHALEISLE_STARTUP_TRACE=/path/startup-trace.json
//
// The file is rewritten atomically after every mark, so a run that is killed
// before Ready still leaves the phases it reached. Marks never block the caller
// on disk I/O: a writer goroutine flushes dirty state every 100ms, and Finish
// forces one final synchronous write.
package startuptrace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EnvPath selects the trace output file.
const EnvPath = "WHALEISLE_STARTUP_TRACE"

// maxEvents bounds the in-memory record so a chatty Host log cannot grow without
// limit; the earliest phases are the ones a cold-start investigation needs.
const maxEvents = 1024

// maxDetail bounds one detail string.
const maxDetail = 400

// maxLogs bounds Host output lines kept as timeline evidence.
const maxLogs = 240

// Event is one mark, span or Host log line.
type Event struct {
	Stage  string  `json:"stage"`
	Kind   string  `json:"kind"`
	AtMs   float64 `json:"atMs"`
	DurMs  float64 `json:"durMs,omitempty"`
	Detail string  `json:"detail,omitempty"`
}

// Report is the JSON document written to EnvPath.
type Report struct {
	Schema    int               `json:"schema"`
	PID       int               `json:"pid"`
	StartedAt string            `json:"startedAt"`
	UpdatedAt string            `json:"updatedAt"`
	Finished  bool              `json:"finished"`
	Reason    string            `json:"reason,omitempty"`
	TotalMs   float64           `json:"totalMs"`
	Meta      map[string]string `json:"meta,omitempty"`
	Events    []Event           `json:"events"`
}

var (
	mu      sync.Mutex
	path    string
	started = time.Now()
	report  *Report
	dirty   bool
	written int64
	logs    int
)

func init() {
	raw := strings.TrimSpace(os.Getenv(EnvPath))
	if raw == "" {
		return
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		abs = raw
	}
	path = abs
	report = &Report{
		Schema:    1,
		PID:       os.Getpid(),
		StartedAt: started.Format(time.RFC3339Nano),
		Meta:      map[string]string{},
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	go writer()
}

// Enabled reports whether tracing is on.
func Enabled() bool { return path != "" }

func msSince() float64 { return float64(time.Since(started).Microseconds()) / 1000 }

func truncate(text string) string {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r", " "), "\n", " ")
	r := []rune(strings.TrimSpace(text))
	if len(r) > maxDetail {
		return string(r[:maxDetail-1]) + "…"
	}
	return string(r)
}

func appendEvent(event Event) {
	mu.Lock()
	if report == nil {
		mu.Unlock()
		return
	}
	event.Stage = strings.TrimSpace(event.Stage)
	event.Detail = truncate(event.Detail)
	if len(report.Events) < maxEvents {
		report.Events = append(report.Events, event)
		dirty = true
	}
	mu.Unlock()
}

// Mark records one instant.
func Mark(stage string, detail ...string) {
	if path == "" {
		return
	}
	appendEvent(Event{Stage: stage, Kind: "mark", AtMs: msSince(), Detail: first(detail)})
}

// Log records one Host output line, bounded; it is timeline evidence, not a stage.
func Log(line string) {
	if path == "" {
		return
	}
	mu.Lock()
	full := logs >= maxLogs
	logs++
	mu.Unlock()
	if full {
		return
	}
	appendEvent(Event{Stage: "host.log", Kind: "log", AtMs: msSince(), Detail: line})
}

// Begin opens a span and returns its end function.
func Begin(stage string, detail ...string) func() {
	if path == "" {
		return func() {}
	}
	began := msSince()
	firstDetail := first(detail)
	return func() {
		appendEvent(Event{Stage: stage, Kind: "span", AtMs: began, DurMs: msSince() - began, Detail: firstDetail})
	}
}

// Meta records one descriptive fact about this run (paths, environment).
func Meta(key, value string) {
	if path == "" {
		return
	}
	mu.Lock()
	if report != nil {
		report.Meta[key] = truncate(value)
		dirty = true
	}
	mu.Unlock()
}

// Finish records the terminal reason and writes the file synchronously.
func Finish(reason string) {
	if path == "" {
		return
	}
	Mark("trace.finish", reason)
	mu.Lock()
	report.Finished = true
	report.Reason = truncate(reason)
	dirty = true
	flushLocked()
	mu.Unlock()
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func writer() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		mu.Lock()
		if dirty {
			flushLocked()
		}
		mu.Unlock()
	}
}

// flushLocked writes through a temporary file so a reader never sees a partial
// report. Callers hold mu.
func flushLocked() {
	if report == nil || path == "" {
		return
	}
	report.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	report.TotalMs = msSince()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return
	}
	dirty = false
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		// A lock held by a reader is not fatal: retry on the next tick.
		dirty = true
		return
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		// Fall back to a direct write; a torn read is better than no evidence.
		if werr := os.WriteFile(path, append(data, '\n'), 0o644); werr != nil {
			dirty = true
			return
		}
	}
	written++
}

// Summary returns a compact stage table for logs; it is empty unless tracing is on.
func Summary() string {
	mu.Lock()
	defer mu.Unlock()
	if report == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("startup trace: ")
	b.WriteString(strconv.FormatFloat(report.TotalMs, 'f', 0, 64))
	b.WriteString("ms")
	if report.Finished {
		b.WriteString(" reason=" + report.Reason)
	}
	return b.String()
}
