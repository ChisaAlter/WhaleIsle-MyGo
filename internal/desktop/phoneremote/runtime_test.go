package phoneremote

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"sync"
	"testing"
	"time"
)

type fixtureChild struct {
	input  bytes.Buffer
	output *io.PipeReader
	writer *io.PipeWriter
	done   chan struct{}
	once   sync.Once
}

func (f *fixtureChild) Input() io.Writer  { return &f.input }
func (f *fixtureChild) Output() io.Reader { return f.output }
func (f *fixtureChild) Wait() error       { <-f.done; return nil }
func (f *fixtureChild) Kill()             { f.once.Do(func() { close(f.done); f.writer.Close() }) }

func TestRuntimeFixture(t *testing.T) {
	r, w := io.Pipe()
	child := &fixtureChild{output: r, writer: w, done: make(chan struct{})}
	m := NewRuntime()
	m.stopTimeout = time.Millisecond
	m.spawn = func(Resources) (Child, error) { return child, nil }
	m.resources = func(string, string) (Resources, error) { return Resources{}, nil }
	m.Start("fixture", "fixture", map[string]any{"remoteEnabled": true})
	defer m.Stop()
	expires := time.Now().Add(time.Minute).UnixMilli()
	offer := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"v":2,"serverId":"fixture","daemonPublicKeyB64":"fictional","authBootstrap":{"version":1,"pairingToken":"fictional-only","expiresAtMs":%d},"relay":{"endpoint":"relay.example.test:443"}}`, expires)))
	var qr bytes.Buffer
	if err := png.Encode(&qr, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	go fmt.Fprintf(w, `{"kind":"snapshot","listening":true,"relayConnected":true,"url":"https://app.example.test/#offer=%s","qrPNG":"%s"}`+"\n", offer, base64.StdEncoding.EncodeToString(qr.Bytes()))
	deadline := time.Now().Add(time.Second)
	for m.Snapshot().Status != "ready" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	s := m.Snapshot()
	if s.Status != "ready" || s.PairingURL == "" || s.PairingImage == nil {
		t.Fatal("valid offer not exposed")
	}
	m.mu.Lock()
	m.state.ExpiresAt = time.Now().Add(-time.Second)
	m.mu.Unlock()
	if s = m.Snapshot(); s.Status != "expired" || s.PairingURL != "" || s.PairingImage != nil {
		t.Fatal("expired offer leaked")
	}
	m.mu.Lock()
	m.state = RuntimeState{Status: "relay-offline", Listening: true, PairingURL: "fictional-sensitive"}
	m.mu.Unlock()
	if m.Snapshot().PairingURL != "" {
		t.Fatal("disconnected relay exposed offer")
	}
	m.consume(child, []byte(`{"kind":"error","code":"operation","error":"fictional-secret-token"}`))
	if bytes.Contains([]byte(m.Snapshot().Message), []byte("fictional-secret-token")) {
		t.Fatal("raw error leaked")
	}
	m.Stop()
	if s = m.Snapshot(); s.PairingURL != "" || s.Listening || s.PairingImage != nil {
		t.Fatal("stop retained offer")
	}
	if !bytes.Contains(child.input.Bytes(), []byte(`"action":"stop"`)) {
		t.Fatal("missing stop control")
	}
	m.Start("fixture", "fixture", map[string]any{"remoteEnabled": false})
	if _, err := ResolveResources(t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("missing runtime accepted")
	}
	if m.Snapshot().Status != "disabled" {
		t.Fatal("disabled state")
	}
}
