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

	"whaleisle/internal/startuprec"
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

type traceRecorder struct {
	mu      sync.Mutex
	path    string
	started time.Time
	report  *Report
	dirty   bool
	written int64
	logs    int
	stopCh  chan struct{}
	doneCh  chan struct{}
}

var defaultRecorder *traceRecorder

func init() {
	raw := strings.TrimSpace(os.Getenv(EnvPath))
	if raw == "" {
		return
	}
	defaultRecorder = startRecorder(raw)
}

func startRecorder(raw string) *traceRecorder {
	abs, err := filepath.Abs(raw)
	if err != nil {
		abs = raw
	}
	now := time.Now()
	r := &traceRecorder{
		path:    abs,
		started: now,
		report: &Report{
			Schema:    1,
			PID:       os.Getpid(),
			StartedAt: now.Format(time.RFC3339Nano),
			Meta:      map[string]string{},
		},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
	_ = os.MkdirAll(filepath.Dir(r.path), 0o755)
	go r.writer(r.stopCh, r.doneCh)
	return r
}

// Enabled reports whether tracing is on.
func Enabled() bool { return defaultRecorder != nil }

func (r *traceRecorder) msSince() float64 {
	return float64(time.Since(r.started).Microseconds()) / 1000
}

func truncate(text string) string {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r", " "), "\n", " ")
	r := []rune(strings.TrimSpace(text))
	if len(r) > maxDetail {
		return string(r[:maxDetail-1]) + "…"
	}
	return string(r)
}

func (r *traceRecorder) appendEvent(event Event) {
	r.mu.Lock()
	if r.report == nil {
		r.mu.Unlock()
		return
	}
	event.Stage = strings.TrimSpace(event.Stage)
	event.Detail = truncate(event.Detail)
	if len(r.report.Events) < maxEvents {
		r.report.Events = append(r.report.Events, event)
		r.dirty = true
	}
	r.mu.Unlock()
}

// Mark records one instant.
func Mark(stage string, detail ...string) {
	if defaultRecorder == nil {
		return
	}
	defaultRecorder.mark(stage, detail...)
}

func (r *traceRecorder) mark(stage string, detail ...string) {
	r.appendEvent(Event{Stage: stage, Kind: "mark", AtMs: r.msSince(), Detail: first(detail)})
}

// Log records one Host output line, bounded; it is timeline evidence, not a stage.
func Log(line string) {
	if defaultRecorder == nil {
		return
	}
	defaultRecorder.log(line)
}

func (r *traceRecorder) log(line string) {
	r.mu.Lock()
	full := r.logs >= maxLogs
	r.logs++
	r.mu.Unlock()
	if full {
		return
	}
	r.appendEvent(Event{Stage: "host.log", Kind: "log", AtMs: r.msSince(), Detail: startuprec.RedactLine(line)})
}

// Begin opens a span and returns its end function.
func Begin(stage string, detail ...string) func() {
	if defaultRecorder == nil {
		return func() {}
	}
	return defaultRecorder.begin(stage, detail...)
}

func (r *traceRecorder) begin(stage string, detail ...string) func() {
	began := r.msSince()
	firstDetail := first(detail)
	return func() {
		r.appendEvent(Event{Stage: stage, Kind: "span", AtMs: began, DurMs: r.msSince() - began, Detail: firstDetail})
	}
}

// Meta records one descriptive fact about this run (paths, environment).
func Meta(key, value string) {
	if defaultRecorder == nil {
		return
	}
	defaultRecorder.meta(key, value)
}

func (r *traceRecorder) meta(key, value string) {
	r.mu.Lock()
	if r.report != nil {
		r.report.Meta[key] = truncate(value)
		r.dirty = true
	}
	r.mu.Unlock()
}

// Finish records the terminal reason, writes the file synchronously, and stops
// the background writer goroutine.
func Finish(reason string) {
	if defaultRecorder == nil {
		return
	}
	defaultRecorder.finish(reason)
}

func (r *traceRecorder) finish(reason string) {
	r.mark("trace.finish", reason)
	r.mu.Lock()
	if r.report != nil {
		r.report.Finished = true
		r.report.Reason = truncate(reason)
		r.dirty = true
		r.flushLocked()
	}
	stop := r.stopCh
	r.stopCh = nil
	done := r.doneCh
	r.mu.Unlock()
	if stop != nil {
		close(stop)
	}
	if done != nil {
		<-done
	}
	r.mu.Lock()
	if r.dirty {
		r.flushLocked()
	}
	r.mu.Unlock()
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (r *traceRecorder) writer(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			select {
			case <-stop:
				return
			default:
			}
			r.mu.Lock()
			if r.dirty {
				r.flushLocked()
			}
			r.mu.Unlock()
		}
	}
}

// flushLocked writes through a temporary file so a reader never sees a partial
// report. Callers hold r.mu.
func (r *traceRecorder) flushLocked() {
	if r.report == nil || r.path == "" {
		return
	}
	r.report.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	r.report.TotalMs = r.msSince()
	data, err := json.MarshalIndent(r.report, "", "  ")
	if err != nil {
		return
	}
	r.dirty = false
	payload := append(data, '\n')
	f, err := os.CreateTemp(filepath.Dir(r.path), "."+filepath.Base(r.path)+"-*.tmp")
	if err != nil {
		// A lock held by a reader is not fatal: retry on the next tick.
		r.dirty = true
		return
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(payload); err != nil {
		_ = f.Close()
		r.dirty = true
		return
	}
	_ = f.Chmod(0o644)
	if err = f.Close(); err != nil {
		r.dirty = true
		return
	}
	if err = os.Rename(tmp, r.path); err != nil {
		// Fall back to a direct write; a torn read is better than no evidence.
		if werr := os.WriteFile(r.path, payload, 0o644); werr != nil {
			r.dirty = true
			return
		}
	}
	r.written++
}

// Summary returns a compact stage table for logs; it is empty unless tracing is on.
func Summary() string {
	if defaultRecorder == nil {
		return ""
	}
	return defaultRecorder.summary()
}

func (r *traceRecorder) summary() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.report == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("startup trace: ")
	b.WriteString(strconv.FormatFloat(r.report.TotalMs, 'f', 0, 64))
	b.WriteString("ms")
	if r.report.Finished {
		b.WriteString(" reason=" + r.report.Reason)
	}
	return b.String()
}
