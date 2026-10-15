// Package startuprec owns bounded, credential-redacted startup evidence and
// the pure admission rules shared by the cold-start and launcher surfaces.
package startuprec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const FileName = "last-desktop-start.json"

var ErrConflict = errors.New("startup outcome changed on disk; reopen before retrying")

// Outcome retains the original launcher's file shape. A missing OK is unknown,
// never a successful start. LogTail contains only sanitized current-attempt data.
type Outcome struct {
	OK         *bool    `json:"ok"`
	At         string   `json:"at"`
	Error      string   `json:"error"`
	LogTail    []string `json:"logTail"`
	AppVersion string   `json:"appVersion,omitempty"`
}

func (o Outcome) Failed() bool { return o.OK != nil && !*o.OK }
func (o Outcome) clone() Outcome {
	out := o
	out.LogTail = append([]string(nil), o.LogTail...)
	if o.OK != nil {
		v := *o.OK
		out.OK = &v
	}
	return out
}

type Store struct {
	mu       sync.Mutex
	path     string
	baseline []byte
	exists   bool
	outcome  Outcome
}

func Open(dir string) (*Store, error) {
	s := &Store{path: filepath.Join(dir, FileName)}
	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 256*1024+1))
	if err != nil {
		return nil, err
	}
	if len(b) > 256*1024 {
		return nil, errors.New("startup outcome exceeds size limit")
	}
	if err = json.Unmarshal(b, &s.outcome); err != nil {
		return nil, fmt.Errorf("startup outcome is unreadable: %w", err)
	}
	// Existing evidence may predate redaction; never expose it verbatim.
	s.outcome.Error = RedactLine(s.outcome.Error)
	s.outcome.LogTail = SanitizeTail(s.outcome.LogTail)
	s.baseline = b
	s.exists = true
	return s, nil
}
func (s *Store) Last() Outcome { s.mu.Lock(); defer s.mu.Unlock(); return s.outcome.clone() }
func (s *Store) Record(ok bool, reason string, logs []string, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := Outcome{OK: &ok, At: time.Now().UTC().Format(time.RFC3339), Error: RedactLine(reason), LogTail: SanitizeTail(logs), AppVersion: version}
	if ok {
		next.Error = ""
		next.LogTail = nil
	}
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	existing, err := os.ReadFile(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if (err == nil) != s.exists || !bytes.Equal(existing, s.baseline) {
		return ErrConflict
	}
	if err = os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".desktop-start-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	// A second comparison avoids overwriting an edit made while serializing.
	again, e := os.ReadFile(s.path)
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if (e == nil) != s.exists || !bytes.Equal(again, s.baseline) {
		return ErrConflict
	}
	if err = os.Rename(f.Name(), s.path); err != nil {
		return err
	}
	s.baseline = append(b, '\n')
	s.exists = true
	s.outcome = next
	return nil
}

func ShouldAutoStart(workspace string, auto, importReady, lastFailed, skip bool) bool {
	return strings.TrimSpace(workspace) != "" && auto && importReady && !lastFailed && !skip
}
func ShouldCloseLauncher(ready, quitAfterStart, recovery, persistenceFailed bool) bool {
	return ready && quitAfterStart && !recovery && !persistenceFailed
}
func CanPromptUpdate(ask, pending, visible, launcher, starting, busy, quitting bool) bool {
	return ask && pending && visible && launcher && !starting && !busy && !quitting
}

var secretHeaders = regexp.MustCompile(`(?i)\b(?:authorization|proxy-authorization|cookie|set-cookie)\s*[:=].*`)
var secretValues = regexp.MustCompile(`(?i)(["']?(?:[a-z0-9_-]*(?:token|password|passwd|secret|api[_-]?key|credential)[a-z0-9_-]*)["']?\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;&]+)`)
var urlCredentials = regexp.MustCompile(`(?i)(https?://)[^\s/@]+(?::[^\s/@]*)?@`)
var urlQuery = regexp.MustCompile(`(https?://[^\s?#]+)[?#][^\s]*`)
var bearer = regexp.MustCompile(`(?i)\bbearer\s+[^\s,;]+`)
var keyShapes = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{8,}|gh[pousr]_[A-Za-z0-9_]+|eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)\b`)

// RedactLine deliberately drops whole auth/cookie headers, query/fragment data
// and familiar credential assignments before truncation. Never pass raw logs to
// persistence or diagnostics. The caller must not read unrelated historic logs.
func RedactLine(line string) string {
	line = strings.ReplaceAll(strings.ReplaceAll(line, "\r", " "), "\n", " ")
	line = secretHeaders.ReplaceAllString(line, "[credential header redacted]")
	line = secretValues.ReplaceAllString(line, "${1}[redacted]")
	line = urlCredentials.ReplaceAllString(line, "${1}[redacted]@")
	line = urlQuery.ReplaceAllString(line, "${1}?[redacted]")
	line = bearer.ReplaceAllString(line, "Bearer [redacted]")
	line = keyShapes.ReplaceAllString(line, "[redacted]")
	r := []rune(line)
	if len(r) > 240 {
		line = string(r[:239]) + "…"
	}
	return line
}
func SanitizeTail(lines []string) []string {
	if len(lines) > 80 {
		lines = lines[len(lines)-80:]
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, RedactLine(line))
	}
	return out
}
