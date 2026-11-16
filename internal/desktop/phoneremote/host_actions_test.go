package phoneremote

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type syncHostChild struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	lines  chan []byte
	output *io.PipeReader
	done   chan struct{}
}

func newSyncHostChild() *syncHostChild {
	r, _ := io.Pipe()
	return &syncHostChild{
		lines:  make(chan []byte, 32),
		output: r,
		done:   make(chan struct{}),
	}
}

func (s *syncHostChild) Input() io.Writer  { return s }
func (s *syncHostChild) Output() io.Reader { return s.output }
func (s *syncHostChild) Wait() error       { <-s.done; return nil }
func (s *syncHostChild) Kill()             {}

func (s *syncHostChild) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Write(p)
	for {
		line, err := s.buf.ReadBytes('\n')
		if err != nil {
			s.buf.Write(line)
			break
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			s.lines <- append([]byte(nil), trimmed...)
		}
	}
	return len(p), nil
}

func (s *syncHostChild) readResult(t *testing.T) map[string]any {
	t.Helper()
	select {
	case raw := <-s.lines:
		var result map[string]any
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
		return result
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async host action result")
		return nil
	}
}

func TestHostSaveConfigPropagatesValidationFailure(t *testing.T) {
	child := newSyncHostChild()
	r := NewRuntime()
	r.child = child
	var calls atomic.Int32
	r.SetHostActions(HostActions{SaveConfig: func(patch map[string]any) (map[string]any, error) {
		calls.Add(1)
		if patch["theme"] != "../dark" {
			t.Fatalf("unexpected patch forwarded to SaveConfig: %v", patch)
		}
		return nil, errors.New("theme must be a valid theme id")
	}})
	r.consume(child, []byte(`{"kind":"host-save-config","id":"1","patch":{"theme":42}}`))
	badType := child.readResult(t)
	if calls.Load() != 0 || badType["ok"] != false || badType["error"] != "theme must be a string" || badType["id"] != "1" {
		t.Fatalf("wrong-type field reached callback or returned unexpected payload: %v, calls=%d", badType, calls.Load())
	}

	r.consume(child, []byte(`{"kind":"host-save-config","id":"2","patch":{"theme":"../dark"}}`))
	result := child.readResult(t)
	if calls.Load() != 1 || result["ok"] != false || result["error"] != "theme must be a valid theme id" || result["config"] != nil || result["id"] != "2" {
		t.Fatalf("validation error became a success: %v, calls=%d", result, calls.Load())
	}
}

func TestHostSaveConfigWhitelistAndTypes(t *testing.T) {
	child := newSyncHostChild()
	r := NewRuntime()
	r.child = child
	var calls atomic.Int32
	var received map[string]any
	r.SetHostActions(HostActions{SaveConfig: func(patch map[string]any) (map[string]any, error) {
		calls.Add(1)
		received = patch
		return map[string]any{"theme": patch["theme"]}, nil
	}})

	rejectedPayloads := []string{
		`{"kind":"host-save-config","id":"1","patch":{"remoteToken":"secret"}}`,
		`{"kind":"host-save-config","id":"2","patch":{"remoteRelayToken":"secret2"}}`,
		`{"kind":"host-save-config","id":"3","patch":{"unknownKey":"abc"}}`,
		`{"kind":"host-save-config","id":"4","patch":{"remoteEnabled":"true"}}`,
		`{"kind":"host-save-config","id":"5","patch":{"remotePort":80}}`,
		`{"kind":"host-save-config","id":"6","patch":{"remotePort":3499.5}}`,
		`{"kind":"host-save-config","id":"7","patch":{"locale":"zh\u0000"}}`,
	}
	for _, raw := range rejectedPayloads {
		r.consume(child, []byte(raw))
		res := child.readResult(t)
		if res["ok"] != false {
			t.Fatalf("expected rejection for %s, got %v", raw, res)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("rejected patches must never invoke SaveConfig: calls=%d", calls.Load())
	}

	valid := `{"kind":"host-save-config","id":"8","patch":{"remoteEnabled":true,"remoteMode":"relay","remotePort":3499,"remoteBindAddress":"127.0.0.1","remoteLanTls":false,"remoteAppBaseUrl":"https://app.example.test","locale":"zh","theme":"deepseek"}}`
	r.consume(child, []byte(valid))
	okRes := child.readResult(t)
	if calls.Load() != 1 || okRes["ok"] != true {
		t.Fatalf("expected whitelisted patch to succeed: %v calls=%d", okRes, calls.Load())
	}
	if received["remotePort"] != 3499 || len(received) != 8 {
		t.Fatalf("unexpected sanitized patch: %#v", received)
	}
}

func TestHostActionsAsyncNonBlockingAndPanicRecovery(t *testing.T) {
	child := newSyncHostChild()
	r := NewRuntime()
	r.child = child
	started := make(chan struct{})
	release := make(chan struct{})
	r.SetHostActions(HostActions{
		SaveConfig: func(patch map[string]any) (map[string]any, error) {
			if patch["theme"] == "panic" {
				panic("boom")
			}
			close(started)
			<-release
			return map[string]any{"theme": "deepseek"}, nil
		},
		OpenSettings: func(section string) bool {
			if section == "panic" {
				panic("open-boom")
			}
			return true
		},
	})

	r.consume(child, []byte(`{"kind":"host-save-config","id":"1","patch":{"theme":"deepseek"}}`))
	<-started
	// Bridge stdout consume continues processing snapshots while SaveConfig blocks.
	r.consume(child, []byte(`{"kind":"snapshot","listening":false,"relayConnected":false}`))
	if r.Snapshot().Status == "ready" {
		t.Fatal("unexpected status")
	}
	close(release)
	res := child.readResult(t)
	if res["ok"] != true || res["id"] != "1" {
		t.Fatalf("blocked saveConfig result mismatch: %v", res)
	}

	r.consume(child, []byte(`{"kind":"host-save-config","id":"2","patch":{"theme":"panic"}}`))
	panicSave := child.readResult(t)
	if panicSave["ok"] != false || panicSave["error"] != "host action failed" {
		t.Fatalf("saveConfig panic not recovered cleanly: %v", panicSave)
	}

	r.consume(child, []byte(`{"kind":"host-open-settings","id":"3","section":"panic"}`))
	panicOpen := child.readResult(t)
	if panicOpen["opened"] != false || panicOpen["id"] != "3" {
		t.Fatalf("openSettings panic not recovered cleanly: %v", panicOpen)
	}
}

func TestHostActionsRejectRevokedChildAndInvalidIdentity(t *testing.T) {
	active, stale := newSyncHostChild(), newSyncHostChild()
	r := NewRuntime()
	r.child = active
	var calls atomic.Int32
	r.SetHostActions(HostActions{
		SaveConfig:   func(map[string]any) (map[string]any, error) { calls.Add(1); return map[string]any{"theme": "deepseek"}, nil },
		OpenSettings: func(string) bool { calls.Add(1); return true },
	})
	for _, kind := range []string{"host-save-config", "host-open-settings"} {
		r.consume(stale, []byte(`{"kind":"`+kind+`","id":"1","patch":{"theme":"deepseek"},"section":"appearance"}`))
		for _, id := range []string{"", "bad", "1\n2", "123456789012345678901"} {
			r.handleHostAction(active, kind, id, "appearance", nil)
		}
	}
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 0 || len(stale.lines) != 0 || len(active.lines) != 0 {
		t.Fatalf("revoked/malformed action reached desktop: calls=%d", calls.Load())
	}
	r.consume(active, []byte(`{"kind":"host-save-config","id":"2","patch":{"theme":"deepseek"}}`))
	result := active.readResult(t)
	if calls.Load() != 1 || result["ok"] != true || result["error"] != nil {
		t.Fatalf("valid active action failed: %v", result)
	}
}
