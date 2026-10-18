package git

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Progress is per-operation, bounded and goroutine-safe. It never stores argv,
// request payloads or full patches; the UI receives phases and sanitized output.
type Progress struct {
	mu      sync.Mutex
	state   ProgressSnapshot
	changed func()
}
type ProgressSnapshot struct {
	Root, Phase  string
	Lines        []string
	Done, Failed bool
}
type progressKey struct{}

func NewProgress(root, phase string, changed func()) *Progress {
	return &Progress{state: ProgressSnapshot{Root: root, Phase: phase}, changed: changed}
}
func WithProgress(ctx context.Context, p *Progress) context.Context {
	return context.WithValue(ctx, progressKey{}, p)
}
func progressFrom(ctx context.Context) *Progress {
	if ctx == nil {
		return nil
	}
	p, _ := ctx.Value(progressKey{}).(*Progress)
	return p
}
func (p *Progress) Snapshot() ProgressSnapshot {
	if p == nil {
		return ProgressSnapshot{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.state
	s.Lines = append([]string(nil), s.Lines...)
	return s
}
func (p *Progress) notify() {
	if p.changed != nil {
		p.changed()
	}
}
func (p *Progress) Phase(phase string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.state.Done {
		p.mu.Unlock()
		return
	}
	p.state.Phase = phase
	p.mu.Unlock()
	p.notify()
}
func (p *Progress) Finish(phase string, failed bool) {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.state.Done {
		p.mu.Unlock()
		return
	}
	p.state.Phase = phase
	p.state.Done, p.state.Failed = true, failed
	p.mu.Unlock()
	p.notify()
}

var progressANSI = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func sanitizeProgressLine(line string) string {
	line = strings.TrimSpace(progressANSI.ReplaceAllString(line, ""))
	lower := strings.ToLower(line)
	if line == "" || strings.HasPrefix(lower, "hint:") || strings.Contains(lower, "lf will be replaced by crlf") || strings.Contains(lower, "crlf will be replaced by lf") || strings.Contains(lower, "warning: in the working copy of ") {
		return ""
	}
	r := []rune(line)
	if len(r) > 500 {
		line = string(r[:499]) + "…"
	}
	return line
}
func (p *Progress) Line(line string) {
	if p == nil {
		return
	}
	line = sanitizeProgressLine(line)
	if line == "" {
		return
	}
	p.mu.Lock()
	if p.state.Done {
		p.mu.Unlock()
		return
	}
	p.state.Lines = append(p.state.Lines, line)
	if len(p.state.Lines) > 64 {
		p.state.Lines = append([]string(nil), p.state.Lines[len(p.state.Lines)-64:]...)
	}
	p.mu.Unlock()
	p.notify()
}
func emitPhase(ctx context.Context, phase string) {
	if p := progressFrom(ctx); p != nil {
		p.Phase(phase)
	}
}

// Git TRACE2 is the original desktop's source of hook start/finish events.
// Child-exit records without child_class are correlated with their start.
type hookTrace struct {
	pending   map[string]string
	remainder string
	offset    int64
}

func (t *hookTrace) line(line string, p *Progress) {
	var r struct {
		Event, ChildClass, HookName, SID string
		ChildID                          json.RawMessage
		Code                             *int
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal([]byte(line), &raw) != nil {
		return
	}
	json.Unmarshal(raw["event"], &r.Event)
	json.Unmarshal(raw["child_class"], &r.ChildClass)
	json.Unmarshal(raw["hook_name"], &r.HookName)
	json.Unmarshal(raw["sid"], &r.SID)
	r.ChildID = raw["child_id"]
	for _, key := range []string{"exit_code", "exitCode", "code"} {
		if json.Unmarshal(raw[key], &r.Code) == nil && r.Code != nil {
			break
		}
	}
	key := r.SID + "/" + string(r.ChildID)
	if len(r.ChildID) == 0 {
		key = r.SID + "/" + r.HookName
	}
	if t.pending == nil {
		t.pending = map[string]string{}
	}
	if r.Event == "child_start" && r.ChildClass == "hook" && r.HookName != "" {
		t.pending[key] = r.HookName
		p.Line("Running " + r.HookName + "…")
		return
	}
	if r.Event == "child_exit" {
		name := t.pending[key]
		if name == "" && r.ChildClass == "hook" {
			name = r.HookName
		}
		if name == "" {
			return
		}
		delete(t.pending, key)
		text := "Finished " + name
		if r.Code != nil && *r.Code != 0 {
			text += " (failed)"
		}
		p.Line(text)
	}
}
func (t *hookTrace) poll(path string, p *Progress, final bool) {
	f, e := os.Open(path)
	if e != nil {
		return
	}
	defer f.Close()
	if _, e = f.Seek(t.offset, io.SeekStart); e != nil {
		return
	}
	data, e := io.ReadAll(io.LimitReader(f, 256*1024))
	if e != nil {
		return
	}
	t.offset += int64(len(data))
	joined := t.remainder + string(data)
	lines := strings.Split(joined, "\n")
	t.remainder = lines[len(lines)-1]
	for _, line := range lines[:len(lines)-1] {
		t.line(line, p)
	}
	if len(t.remainder) > 64*1024 {
		t.remainder = ""
	}
	if final && t.remainder != "" {
		t.line(t.remainder, p)
		t.remainder = ""
	}
}

type progressOutput struct {
	mu       sync.Mutex
	tail     []byte
	line     []byte
	progress *Progress
}

func (w *progressOutput) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tail = append(w.tail, data...)
	if len(w.tail) > 1024*1024 {
		w.tail = append([]byte("[earlier output truncated]\n"), w.tail[len(w.tail)-1024*1024:]...)
	}
	for _, b := range data {
		if b == '\n' || b == '\r' {
			w.progress.Line(string(w.line))
			w.line = nil
		} else if len(w.line) < 8192 {
			w.line = append(w.line, b)
		}
	}
	return len(data), nil
}
func (w *progressOutput) finish() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.progress.Line(string(w.line))
	w.line = nil
	return append([]byte(nil), w.tail...)
}
func observedCombinedOutput(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	p := progressFrom(ctx)
	if p == nil {
		return cmd.CombinedOutput()
	}
	output := &progressOutput{progress: p}
	cmd.Stdout, cmd.Stderr = output, output
	trace, e := os.CreateTemp("", "mygo-git-trace2-*.json")
	if e != nil {
		err := cmd.Run()
		return output.finish(), err
	}
	path := trace.Name()
	trace.Close()
	defer os.Remove(path)
	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}
	cmd.Env = nil
	for _, item := range env {
		if !strings.HasPrefix(item, "GIT_TRACE2_EVENT=") {
			cmd.Env = append(cmd.Env, item)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_TRACE2_EVENT="+path)
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		monitor := &hookTrace{}
		for {
			select {
			case <-ticker.C:
				monitor.poll(path, p, false)
			case <-stop:
				monitor.poll(path, p, true)
				return
			}
		}
	}()
	err := cmd.Run()
	close(stop)
	<-done
	return output.finish(), err
}
func shouldObserveGit(args []string) bool {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-c" {
			i++
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a == "commit" || a == "push" || a == "pull" || a == "fetch"
	}
	return false
}
